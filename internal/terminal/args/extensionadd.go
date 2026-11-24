package args

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/pkg/source"
)

func (h *PHPHandler) extensionAvailable(cmd *cobra.Command, args []string) error {
	jsonFlag, _ := cmd.Flags().GetBool("json")

	version, err := h.resolveVersion("")
	if len(args) > 0 {
		version, err = h.resolveVersion(args[0])
	}
	if err != nil {
		return err
	}
	exts := h.assemblerSvc.ListExtensionsForPHP(version)

	if jsonFlag {
		var entries []extAvailEntry
		for _, e := range exts {
			entries = append(entries, extAvailEntry{Name: e.Name, Description: e.Description})
		}
		sort.Slice(entries, func(i, j int) bool { return entries[i].Name < entries[j].Name })
		return printJSON(jsonResponse{SchemaVersion: 1, Data: extAvailResponse{
			PHPVersion: version,
			Extensions: entries,
		}})
	}

	fmt.Printf("Available extensions for PHP %s:\n", version)
	sort.Slice(exts, func(i, j int) bool {
		return exts[i].Name < exts[j].Name
	})
	for _, e := range exts {
		fmt.Printf("  %-20s %s\n", e.Name, e.Description)
	}
	return nil
}

func (h *PHPHandler) extensionAdd(cmd *cobra.Command, args []string) error {
	versionFlag, _ := cmd.Flags().GetString("version")
	version, err := h.resolveVersion(versionFlag)
	if err != nil {
		return err
	}
	extNames := args

	jobsFlag, _ := cmd.Flags().GetInt("jobs")
	jobs := resolveJobs(jobsFlag, h.configSvc)

	prefix := h.siloSvc.PackagePrefix("php", version)
	phpBin := filepath.Join(prefix, "bin", "php")
	if _, err := os.Stat(phpBin); os.IsNotExist(err) {
		return fmt.Errorf("PHP %s is not installed. Run `phpv install %s` first", version, version)
	}

	autoDeps, _ := cmd.Flags().GetBool("auto-deps")
	noSystem, _ := cmd.Flags().GetBool("no-system")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	iniSettings, err := parseINITFlags(cmd)
	if err != nil {
		return err
	}

	tc, err := h.selectToolchain("")
	if err != nil {
		return err
	}

	if !noSystem {
		if _, err := h.checkSystemDeps(extNames, autoDeps, dryRun); err != nil {
			return err
		}
		if err := h.checkBuildTools(tc, autoDeps, dryRun); err != nil {
			return err
		}
	}

	manifest, err := h.siloSvc.GetExtensionManifest(version)
	if err != nil {
		return fmt.Errorf("get extension manifest: %w", err)
	}
	installed := make(map[string]bool)
	for _, e := range manifest.Extensions {
		installed[e.Name] = true
	}

	// Self-healing: cross-check manifest against php -m. If manifest says
	// installed but php -m doesn't show it, the entry is stale (e.g., wrong
	// php.ini path or partial build). Remove stale entries from manifest.
	compiledIn, err := h.getCompiledModules(version)
	if err == nil {
		for name := range installed {
			if !compiledIn[name] {
				// Stale entry: remove from manifest
				delete(installed, name)
				var clean []domain.ExtensionState
				for _, e := range manifest.Extensions {
					if e.Name != name {
						clean = append(clean, e)
					}
				}
				manifest.Extensions = clean
			}
		}
		if err := h.siloSvc.SaveExtensionManifest(version, manifest); err != nil {
			return fmt.Errorf("clean stale manifest: %w", err)
		}
	}

	// Bundled extensions need the PHP source tree; resolve it lazily so a
	// pure-PECL request never downloads PHP source.
	srcPath := ""
	resolveSource := func() (string, error) {
		if srcPath != "" {
			return srcPath, nil
		}
		sourceDir := h.siloSvc.SourcePath("php", version)
		srcPath = source.Find(sourceDir, "php")
		if srcPath == "" {
			fmt.Printf("PHP source not found, downloading PHP %s source...\n", version)
			if err := h.downloadPHPSource(version); err != nil {
				return "", fmt.Errorf("download PHP source: %w", err)
			}
			srcPath = source.Find(sourceDir, "php")
			if srcPath == "" {
				return "", fmt.Errorf("PHP source not found at %s after download", sourceDir)
			}
		}
		return srcPath, nil
	}

	dryRunPrinted := false
	for _, ext := range extNames {
		if installed[ext] {
			fmt.Printf("↷ %s already installed, skipping\n", ext)
			continue
		}

		// Route: graph-known extensions are bundled (build from the PHP
		// source tree); anything else comes from pecl.php.net with version
		// compatibility.
		_, knownBundled := h.assemblerSvc.GetExtensionDef(ext)
		if knownBundled {
			if dryRun {
				fmt.Printf("Would build bundled extension %s from PHP %s source\n", ext, version)
				dryRunPrinted = true
				continue
			}
			src, err := resolveSource()
			if err != nil {
				return err
			}
			extDir := filepath.Join(src, "ext", ext)
			if _, err := os.Stat(extDir); os.IsNotExist(err) {
				return fmt.Errorf("extension %q is a bundled extension but was not found in the PHP source tree at %s", ext, extDir)
			}
			fmt.Printf("Building extension %s...\n", ext)
			if err := h.assemblerSvc.InstallExtension(h.ctx, domain.ExtensionInstallRequest{PHPVersion: version, ExtName: ext, PHPSourceDir: src, PHPPrefix: prefix, Jobs: jobs}); err != nil {
				return fmt.Errorf("install extension %s: %w", ext, err)
			}
			fmt.Printf("✓ %s installed\n", ext)
			continue
		}

		if dryRun {
			rel, err := h.peclSvc.ResolveCompatibleRelease(ext, version)
			if err != nil {
				return err
			}
			fmt.Printf("Would install PECL extension %s %s for PHP %s\n", ext, rel, version)
			dryRunPrinted = true
			continue
		}
		fmt.Printf("Installing PECL extension %s (resolving compatible release for PHP %s)...\n", ext, version)
		res, err := h.peclSvc.InstallCompatible(h.ctx, domain.PECLInstallRequest{Name: ext, PHPVersion: version, Jobs: jobs, ExtraINI: iniSettings})
		if err != nil {
			return err
		}
		fmt.Printf("✓ %s %s installed (PECL)\n", res.Name, res.Version)
		h.printExtensionHint(ext)
	}
	if dryRun {
		fmt.Println("Dry run complete. Run without --dry-run to install.")
	}
	_ = dryRunPrinted
	return nil
}

// parseINITFlags reads the repeatable --ini key=value flag and validates
// each entry: non-empty key, no whitespace, exactly one '='.
func parseINITFlags(cmd *cobra.Command) ([]string, error) {
	raw, _ := cmd.Flags().GetStringArray("ini")
	if len(raw) == 0 {
		return nil, nil
	}
	out := make([]string, 0, len(raw))
	for _, s := range raw {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		eq := strings.Index(s, "=")
		if eq <= 0 {
			return nil, fmt.Errorf("invalid --ini setting %q: expected key=value (e.g. xdebug.mode=debug)", s)
		}
		key := s[:eq]
		if strings.ContainsAny(key, " \t") {
			return nil, fmt.Errorf("invalid --ini key %q: keys must not contain whitespace", key)
		}
		out = append(out, s)
	}
	return out, nil
}

// printExtensionHint prints a short, informational post-install hint for
// extensions that commonly need one more step. No files are written.
func (h *PHPHandler) printExtensionHint(ext string) {
	switch ext {
	case "xdebug":
		fmt.Println("ℹ xdebug is disabled by default until xdebug.mode is set, e.g.:")
		fmt.Println("    phpv extension add xdebug --ini xdebug.mode=debug")
	}
}

func (h *PHPHandler) downloadPHPSource(version string) error {
	regEntry, err := h.registrySvc.Get("php", version)
	if err != nil {
		return fmt.Errorf("registry resolve php@%s: %w", version, err)
	}
	if _, err := h.siloSvc.DownloadURL(regEntry.URL, regEntry.ChecksumType, regEntry.ChecksumValue); err != nil {
		return fmt.Errorf("download php source: %w", err)
	}
	archivePath := filepath.Join(cacheDir(), filepath.Base(regEntry.URL))
	sourceDir := h.siloSvc.SourcePath("php", version)
	if _, err := h.siloSvc.Extract(archivePath, sourceDir); err != nil {
		return fmt.Errorf("extract php source: %w", err)
	}
	return nil
}

func cacheDir() string {
	return filepath.Join(resolvePHPVRoot(), "caches")
}

func resolvePHPVRoot(parts ...string) string {
	root := os.Getenv("PHPV_ROOT")
	if root == "" {
		home, _ := os.UserHomeDir()
		root = filepath.Join(home, ".phpv")
	}
	return filepath.Join(append([]string{root}, parts...)...)
}
