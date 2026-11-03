package terminal

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/internal/source"
)

func (h *PHPHandler) extensionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "extension",
		Short: "Manage PHP extensions",
		Long:  "List, add, or remove extensions for an installed PHP version.",
	}

	listCmd := &cobra.Command{
		Use:   "list [version]",
		Short: "List installed extensions for a PHP version",
		Args:  cobra.MaximumNArgs(1),
		RunE:  h.extensionList,
	}
	listCmd.Flags().Bool("json", false, "Output in JSON format")
	cmd.AddCommand(listCmd)

	availCmd := &cobra.Command{
		Use:   "available [version]",
		Short: "List extensions available for a PHP version",
		Args:  cobra.MaximumNArgs(1),
		RunE:  h.extensionAvailable,
	}
	availCmd.Flags().Bool("json", false, "Output in JSON format")
	cmd.AddCommand(availCmd)

	extCmd := &cobra.Command{
		Use:   "add <name>...",
		Short: "Install one or more extensions",
		Long: `Install one or more extensions for a PHP version.

Bundled extensions (gd, pdo_pgsql, ...) are built from the PHP source tree.
Non-bundled extensions (xdebug, redis, apcu, ...) are resolved from
pecl.php.net against the target PHP version's compatibility range, built,
verified to load, and enabled.

Without --version, the extension is installed into the active PHP version.
Examples:
  phpv extension add pdo_pgsql                # active version
  phpv extension add gd intl --version 8.4    # specific version
  phpv extension add xdebug --ini xdebug.mode=debug
  phpv extension add xdebug --dry-run         # show the resolved release`,
		Args: cobra.MinimumNArgs(1),
		RunE: h.extensionAdd,
	}
	extCmd.Flags().Int("jobs", 0, "Number of parallel build jobs (default: CPU count)")
	extCmd.Flags().Bool("force", false, "Force rebuild even if already installed")
	extCmd.Flags().Bool("auto-deps", false, "Install missing system packages without prompting")
	extCmd.Flags().Bool("no-system", false, "Skip system package check, always build from source")
	extCmd.Flags().Bool("dry-run", false, "Show what would be done without doing it")
	extCmd.Flags().String("version", "", "PHP version to install into (defaults to the active version)")
	extCmd.Flags().StringArray("ini", nil, "INI setting to add after install (key=value, repeatable)")
	cmd.AddCommand(extCmd)

	removeCmd := &cobra.Command{
		Use:   "remove <name>...",
		Short: "Remove one or more extensions",
		Long: `Remove one or more extensions from a PHP version.

Without --version, the extension is removed from the active PHP version.
Examples:
  phpv extension remove pdo_pgsql                # active version
  phpv extension remove gd intl --version 8.4    # specific version`,
		Args: cobra.MinimumNArgs(1),
		RunE: h.extensionRemove,
	}
	removeCmd.Flags().String("version", "", "PHP version to remove from (defaults to the active version)")
	cmd.AddCommand(removeCmd)

	peclCmd := &cobra.Command{
		Use:   "pecl [version]",
		Short: "List installed PECL extensions for a PHP version",
		Args:  cobra.MaximumNArgs(1),
		RunE:  h.extensionPecl,
	}
	peclCmd.Flags().Bool("json", false, "Output in JSON format")
	cmd.AddCommand(peclCmd)

	return cmd
}

type extListEntry struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Installed   bool   `json:"installed"`
	Type        string `json:"type,omitempty"`
}

type extListResponse struct {
	PHPVersion string         `json:"php_version"`
	Extensions []extListEntry `json:"extensions"`
}

func (h *PHPHandler) extensionList(cmd *cobra.Command, args []string) error {
	jsonFlag, _ := cmd.Flags().GetBool("json")

	version, err := h.resolveVersion("")
	if len(args) > 0 {
		version, err = h.resolveVersion(args[0])
	}
	if err != nil {
		return err
	}

	manifest, err := h.siloSvc.GetExtensionManifest(version)
	if err != nil {
		return fmt.Errorf("get extension manifest: %w", err)
	}

	allExts := h.assemblerSvc.ListExtensionsForPHP(version)
	extMap := make(map[string]domain.ExtensionInfo)
	for _, e := range allExts {
		extMap[e.Name] = e
	}

	installed := make(map[string]bool)
	peclInstalled := make(map[string]bool)
	for _, e := range manifest.Extensions {
		installed[e.Name] = true
		if e.Type == domain.ExtensionTypePECL {
			peclInstalled[e.Name] = true
		}
	}

	compiledIn, err := h.getCompiledModules(version)
	if err == nil {
		for name := range compiledIn {
			installed[name] = true
		}
	}

	if jsonFlag {
		var entries []extListEntry
		for _, e := range allExts {
			entries = append(entries, extListEntry{
				Name:        e.Name,
				Description: e.Description,
				Installed:   installed[e.Name],
			})
		}
		sort.Slice(entries, func(i, j int) bool { return entries[i].Name < entries[j].Name })
		for name := range peclInstalled {
			if _, ok := extMap[name]; !ok {
				entries = append(entries, extListEntry{
					Name:      name,
					Installed: true,
					Type:      "pecl",
				})
			}
		}
		return printJSON(jsonResponse{SchemaVersion: 1, Data: extListResponse{
			PHPVersion: version,
			Extensions: entries,
		}})
	}

	fmt.Printf("Extensions for PHP %s (compiled in + manifest):\n", version)
	var names []string
	for _, e := range allExts {
		names = append(names, e.Name)
	}
	sort.Strings(names)

	var installedCount int
	for _, name := range names {
		info := extMap[name]
		marker := "✗"
		if installed[name] {
			marker = "✓"
			installedCount++
		}
		fmt.Printf("  %s %-20s %s\n", marker, name, info.Description)
	}

	if len(peclInstalled) > 0 {
		fmt.Println("\nPECL extensions:")
		var peclNames []string
		for name := range peclInstalled {
			peclNames = append(peclNames, name)
		}
		sort.Strings(peclNames)
		for _, name := range peclNames {
			fmt.Printf("  ✓ %s\n", name)
		}
	}

	fmt.Printf("\n%d installed, %d available\n", installedCount, len(allExts))

	return nil
}

// getCompiledModules runs php -m for the given version and returns the set
// of compiled-in module names (lowercased). Returns an error if the PHP
// binary is not found or cannot be executed.
func (h *PHPHandler) getCompiledModules(version string) (map[string]bool, error) {
	prefix := h.siloSvc.PackagePrefix("php", version)
	phpBin := filepath.Join(prefix, "bin", "php")
	if _, err := os.Stat(phpBin); os.IsNotExist(err) {
		return nil, fmt.Errorf("PHP binary not found at %s", phpBin)
	}

	cmd := exec.Command(phpBin, "-m")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("php -m failed: %w", err)
	}

	modules := make(map[string]bool)
	inZend := false
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if line == "[Zend Modules]" {
			inZend = true
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			continue
		}
		moduleName := strings.ToLower(line)
		modules[moduleName] = true
		if inZend {
			// Zend modules like "Zend OPcache" → also register as "opcache"
			shortName := strings.TrimPrefix(moduleName, "zend ")
			if shortName != moduleName {
				modules[shortName] = true
			}
		}
	}

	return modules, nil
}

type extAvailEntry struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type extAvailResponse struct {
	PHPVersion string          `json:"php_version"`
	Extensions []extAvailEntry `json:"extensions"`
}

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
			if err := h.assemblerSvc.InstallExtension(h.ctx, version, ext, src, prefix, jobs); err != nil {
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
		res, err := h.peclSvc.InstallCompatible(h.ctx, ext, version, jobs, iniSettings)
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

func (h *PHPHandler) extensionRemove(cmd *cobra.Command, args []string) error {
	versionFlag, _ := cmd.Flags().GetString("version")
	version, err := h.resolveVersion(versionFlag)
	if err != nil {
		return err
	}
	extNames := args

	prefix := h.siloSvc.PackagePrefix("php", version)
	phpBin := filepath.Join(prefix, "bin", "php")
	if _, err := os.Stat(phpBin); os.IsNotExist(err) {
		return fmt.Errorf("PHP %s is not installed. Run `phpv install %s` first", version, version)
	}

	manifest, err := h.siloSvc.GetExtensionManifest(version)
	if err != nil {
		return fmt.Errorf("get extension manifest: %w", err)
	}
	extMap := make(map[string]string)
	for _, e := range manifest.Extensions {
		extMap[e.Name] = e.Type
	}

	for _, ext := range extNames {
		if extMap[ext] == domain.ExtensionTypePECL {
			fmt.Printf("Removing PECL extension %s...\n", ext)
			if err := h.peclSvc.Uninstall(ext, version); err != nil {
				return fmt.Errorf("remove PECL extension %s: %w", ext, err)
			}
		} else {
			fmt.Printf("Removing extension %s...\n", ext)
			if err := h.assemblerSvc.RemoveExtension(version, ext, prefix); err != nil {
				return fmt.Errorf("remove extension %s: %w", ext, err)
			}
		}
		fmt.Printf("✓ %s removed\n", ext)
	}
	return nil
}

type extPeclEntry struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type extPeclResponse struct {
	PHPVersion string         `json:"php_version"`
	Extensions []extPeclEntry `json:"extensions"`
}

func (h *PHPHandler) extensionPecl(cmd *cobra.Command, args []string) error {
	jsonFlag, _ := cmd.Flags().GetBool("json")

	version, err := h.resolveVersion("")
	if len(args) > 0 {
		version, err = h.resolveVersion(args[0])
	}
	if err != nil {
		return err
	}
	exts, err := h.peclSvc.List(version)
	if err != nil {
		return fmt.Errorf("list PECL extensions: %w", err)
	}

	if jsonFlag {
		var entries []extPeclEntry
		for _, e := range exts {
			entries = append(entries, extPeclEntry{Name: e.Name, Version: e.Version})
		}
		return printJSON(jsonResponse{SchemaVersion: 1, Data: extPeclResponse{
			PHPVersion: version,
			Extensions: entries,
		}})
	}

	if len(exts) == 0 {
		fmt.Printf("No PECL extensions installed for PHP %s\n", version)
		return nil
	}
	fmt.Printf("PECL extensions for PHP %s:\n", version)
	for _, e := range exts {
		v := e.Version
		if v == "" {
			v = "?"
		}
		fmt.Printf("  ✓ %s (%s)\n", e.Name, v)
	}
	return nil
}
