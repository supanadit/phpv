package args

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"
)

func (h *PHPHandler) versions(cmd *cobra.Command, args []string) error {
	jsonFlag, _ := cmd.Flags().GetBool("json")

	silo := h.siloSvc.GetSilo()
	phpDir := filepath.Join(silo.Root, "packages", "php")

	entries, err := os.ReadDir(phpDir)
	if err != nil {
		if os.IsNotExist(err) {
			if jsonFlag {
				return printJSON(jsonResponse{SchemaVersion: 1, Data: []versionEntry{}})
			}
			fmt.Println("No PHP versions installed.")
			return nil
		}
		return fmt.Errorf("read php versions dir: %w", err)
	}

	defaultVer, _ := h.siloSvc.GetDefault()

	var installed []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		phpBin := filepath.Join(phpDir, e.Name(), "bin", "php")
		if _, err := os.Stat(phpBin); err == nil {
			installed = append(installed, e.Name())
		}
	}

	sort.Sort(sort.Reverse(sort.StringSlice(installed)))

	if jsonFlag {
		var vers []versionEntry
		for _, v := range installed {
			vers = append(vers, versionEntry{Version: v, Default: v == defaultVer})
		}
		return printJSON(jsonResponse{SchemaVersion: 1, Data: vers})
	}

	if len(installed) == 0 {
		fmt.Println("No PHP versions installed.")
		return nil
	}

	fmt.Println("Installed PHP versions:")
	for _, v := range installed {
		marker := " "
		if v == defaultVer {
			marker = "*"
		}
		fmt.Printf("  %s %s\n", marker, v)
	}
	if defaultVer != "" {
		fmt.Printf("\n(* = default)\n")
	}
	return nil
}

// listCmd lists available PHP versions from the registry.
func (h *PHPHandler) listCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available PHP versions",
		Args:  cobra.NoArgs,
		RunE:  h.listAvailable,
	}
	cmd.Flags().Bool("json", false, "Output in JSON format")
	return cmd
}

type listEntry struct {
	Version string `json:"version"`
}

func (h *PHPHandler) listAvailable(cmd *cobra.Command, args []string) error {
	jsonFlag, _ := cmd.Flags().GetBool("json")

	entries, err := h.registrySvc.List("php")
	if err != nil {
		return fmt.Errorf("list php versions: %w", err)
	}

	seen := make(map[string]bool)
	var versions []string
	for _, e := range entries {
		if !seen[e.Version] {
			seen[e.Version] = true
			versions = append(versions, e.Version)
		}
	}

	sort.Sort(sort.Reverse(sort.StringSlice(versions)))

	if jsonFlag {
		var vers []listEntry
		for _, v := range versions {
			vers = append(vers, listEntry{Version: v})
		}
		return printJSON(jsonResponse{SchemaVersion: 1, Data: vers})
	}

	fmt.Println("Available PHP versions:")
	for _, v := range versions {
		fmt.Printf("  %s\n", v)
	}
	return nil
}

// whichCmd shows the path to the current PHP binary.
func (h *PHPHandler) whichCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "which",
		Short: "Show path to current PHP binary",
		Args:  cobra.NoArgs,
		RunE:  h.which,
	}
}

func (h *PHPHandler) which(cmd *cobra.Command, args []string) error {
	path, err := h.resolveActivePHP()
	if err != nil {
		return err
	}
	fmt.Println(path)
	return nil
}

// defaultCmd sets the global default PHP version.
func (h *PHPHandler) defaultCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "default <version>",
		Short: "Set default PHP version",
		Long: `Set the default PHP version.

--global (default) writes to ~/.phpv/default.
--local writes to .php-version in the current directory.`,
		Args: cobra.ExactArgs(1),
		RunE: h.setDefault,
	}
	cmd.Flags().Bool("global", true, "Set as global default (writes to ~/.phpv/default)")
	cmd.Flags().Bool("local", false, "Set as local project version (writes .php-version in CWD)")
	cmd.MarkFlagsMutuallyExclusive("global", "local")
	return cmd
}

func (h *PHPHandler) setDefault(cmd *cobra.Command, args []string) error {
	version, err := h.resolveVersion(args[0])
	if err != nil {
		return err
	}

	local, _ := cmd.Flags().GetBool("local")
	if local {
		if err := writeLocalVersionFile(version); err != nil {
			return fmt.Errorf("write local version: %w", err)
		}
		fmt.Printf("✓ Local PHP version set to %s (.php-version)\n", version)
		return nil
	}

	if err := h.siloSvc.SetDefault(version); err != nil {
		return fmt.Errorf("set default: %w", err)
	}
	fmt.Printf("✓ Default PHP version set to %s\n", version)
	return nil
}

// useCmd switches the active PHP version for the current session.
func (h *PHPHandler) useCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "use <version>",
		Short: "Switch to a PHP version",
		Long: `Switch the active PHP version.

'phpv use 8' switches per-shell only (exports PHPV_CURRENT) — it never
touches the global default or writes project files. This requires shell
integration via 'phpv init'. Use 'system' to fall back to the system PHP.

--global writes to ~/.phpv/default and regenerates shims.
--local writes to .php-version in the current directory only.`,
		Args: cobra.ExactArgs(1),
		RunE: h.use,
	}
	cmd.Flags().Bool("global", false, "Set as global default (writes to ~/.phpv/default, regenerates shims)")
	cmd.Flags().Bool("local", false, "Set as local project version (writes .php-version in CWD)")
	cmd.Flags().Bool("print", false, "Print an export statement for the current shell (used by shell integration)")
	cmd.Flags().MarkHidden("print")
	cmd.MarkFlagsMutuallyExclusive("global", "local")
	return cmd
}
