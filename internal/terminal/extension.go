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
	cmd.AddCommand(&cobra.Command{
		Use:   "list <version>",
		Short: "List installed extensions for a PHP version",
		Args:  cobra.ExactArgs(1),
		RunE:  h.extensionList,
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "available <version>",
		Short: "List extensions available for a PHP version",
		Args:  cobra.ExactArgs(1),
		RunE:  h.extensionAvailable,
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "add <version> <name>...",
		Short: "Install one or more extensions",
		Args:  cobra.MinimumNArgs(2),
		RunE:  h.extensionAdd,
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "remove <version> <name>...",
		Short: "Remove one or more extensions",
		Args:  cobra.MinimumNArgs(2),
		RunE:  h.extensionRemove,
	})
	return cmd
}

func (h *PHPHandler) extensionList(cmd *cobra.Command, args []string) error {
	version := args[0]
	prefix := h.siloSvc.PackagePrefix("php", version)

	phpBin := filepath.Join(prefix, "bin", "php")
	if _, err := os.Stat(phpBin); os.IsNotExist(err) {
		return fmt.Errorf("PHP %s is not installed. Run `phpv install %s` first", version, version)
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
	for _, e := range manifest.Extensions {
		installed[e.Name] = true
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

	return nil
}

func (h *PHPHandler) extensionAvailable(cmd *cobra.Command, args []string) error {
	version := args[0]
	exts := h.assemblerSvc.Graph().ListExtensionsForPHP(version)

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
	version := args[0]
	extNames := args[1:]

	prefix := h.siloSvc.PackagePrefix("php", version)
	phpBin := filepath.Join(prefix, "bin", "php")
	if _, err := os.Stat(phpBin); os.IsNotExist(err) {
		return fmt.Errorf("PHP %s is not installed. Run `phpv install %s` first", version, version)
	}

	sourceDir := h.siloSvc.SourcePath("php", version)
	srcPath := assembler.FindSourceDir(sourceDir, "php", version)
	if srcPath == "" {
		return fmt.Errorf("PHP source not found at %s (re-run `phpv install %s` to download it)", sourceDir, version)
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
		if err := h.assemblerSvc.InstallExtension(version, ext, srcPath, prefix); err != nil {
			return fmt.Errorf("install extension %s: %w", ext, err)
		}
		fmt.Printf("✓ %s installed\n", ext)
	}
	return nil
}

func (h *PHPHandler) extensionRemove(cmd *cobra.Command, args []string) error {
	version := args[0]
	extNames := args[1:]

	prefix := h.siloSvc.PackagePrefix("php", version)
	phpBin := filepath.Join(prefix, "bin", "php")
	if _, err := os.Stat(phpBin); os.IsNotExist(err) {
		return fmt.Errorf("PHP %s is not installed. Run `phpv install %s` first", version, version)
	}

	for _, ext := range extNames {
		fmt.Printf("Removing extension %s...\n", ext)
		if err := h.assemblerSvc.RemoveExtension(version, ext, prefix); err != nil {
			return fmt.Errorf("remove extension %s: %w", ext, err)
		}
		fmt.Printf("✓ %s removed\n", ext)
	}
	return nil
}
