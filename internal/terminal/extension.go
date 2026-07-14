package terminal

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"
	"github.com/supanadit/phpv/assembler"
	"github.com/supanadit/phpv/domain"
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
		Use:   "add <version> <name>...",
		Short: "Install one or more extensions",
		Args:  cobra.MinimumNArgs(2),
		RunE:  h.extensionAdd,
	}
	extCmd.Flags().Int("jobs", 0, "Number of parallel build jobs (default: CPU count)")
	cmd.AddCommand(extCmd)
	cmd.AddCommand(&cobra.Command{
		Use:   "remove <version> <name>...",
		Short: "Remove one or more extensions",
		Args:  cobra.MinimumNArgs(2),
		RunE:  h.extensionRemove,
	})

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

	allExts := h.assemblerSvc.Graph().ListExtensionsForPHP(version)
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

	fmt.Printf("Extensions for PHP %s:\n", version)
	var names []string
	for _, e := range allExts {
		names = append(names, e.Name)
	}
	sort.Strings(names)

	for _, name := range names {
		info := extMap[name]
		marker := "✗"
		if installed[name] {
			marker = "✓"
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

	return nil
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
	exts := h.assemblerSvc.Graph().ListExtensionsForPHP(version)

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
	version, err := h.resolveVersion(args[0])
	if err != nil {
		return err
	}
	extNames := args[1:]

	jobsFlag, _ := cmd.Flags().GetInt("jobs")
	jobs := resolveJobs(jobsFlag, h.configSvc)

	prefix := h.siloSvc.PackagePrefix("php", version)
	phpBin := filepath.Join(prefix, "bin", "php")
	if _, err := os.Stat(phpBin); os.IsNotExist(err) {
		return fmt.Errorf("PHP %s is not installed. Run `phpv install %s` first", version, version)
	}

	sourceDir := h.siloSvc.SourcePath("php", version)
	srcPath := assembler.FindSourceDir(sourceDir, "php", version)
	if srcPath == "" {
		fmt.Printf("PHP source not found, downloading PHP %s source...\n", version)
		if err := h.downloadPHPSource(version); err != nil {
			return fmt.Errorf("download PHP source: %w", err)
		}
		srcPath = assembler.FindSourceDir(sourceDir, "php", version)
		if srcPath == "" {
			return fmt.Errorf("PHP source not found at %s after download", sourceDir)
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

	for _, ext := range extNames {
		if installed[ext] {
			fmt.Printf("↷ %s already installed, skipping\n", ext)
			continue
		}
		fmt.Printf("Building extension %s...\n", ext)
		if err := h.assemblerSvc.InstallExtension(h.ctx, version, ext, srcPath, prefix, jobs); err != nil {
			return fmt.Errorf("install extension %s: %w", ext, err)
		}
		fmt.Printf("✓ %s installed\n", ext)
	}
	return nil
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
	version, err := h.resolveVersion(args[0])
	if err != nil {
		return err
	}
	extNames := args[1:]

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
	PHPVersion string          `json:"php_version"`
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
