package args

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/supanadit/phpv/domain"
)

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
