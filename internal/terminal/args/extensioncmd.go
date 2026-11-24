package args

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
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
