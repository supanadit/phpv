package terminal

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/internal/utils"
)

func (h *TerminalHandler) PECLInstall(archivePath string) (*PECLInstallResult, error) {
	activeVer, err := h.resolveActivePHP()
	if err != nil {
		return nil, err
	}

	ext, err := h.BundlerRepo.PECLInstall(archivePath, activeVer)
	if err != nil {
		return nil, err
	}

	silo, err := h.Silo.GetSilo()
	if err != nil {
		return nil, fmt.Errorf("failed to get silo: %w", err)
	}
	extensionsDir := filepath.Join(utils.PHPOutputPath(silo, activeVer), "lib", "extensions", ext.Name)

	return &PECLInstallResult{
		Name:       ext.Name,
		Version:    ext.Version,
		InstallDir: extensionsDir,
	}, nil
}

func (h *TerminalHandler) PECLList() ([]string, error) {
	activeVer, err := h.resolveActivePHP()
	if err != nil {
		return nil, err
	}

	return h.BundlerRepo.PECLList(activeVer)
}

func (h *TerminalHandler) PECLUninstall(name string) error {
	activeVer, err := h.resolveActivePHP()
	if err != nil {
		return err
	}

	return h.BundlerRepo.PECLUninstall(name, activeVer)
}

func (h *TerminalHandler) ListExtensions(phpVersion string) (*ExtensionsResult, error) {
	var extensions []domain.ExtensionInfo
	if phpVersion == "" {
		extensions = h.ExtensionRepo.ListExtensions()
	} else {
		extensions = h.ExtensionRepo.ListExtensionsForPHP(phpVersion)
	}

	sort.Slice(extensions, func(i, j int) bool {
		return extensions[i].Name < extensions[j].Name
	})

	return &ExtensionsResult{
		Extensions: extensions,
		PHPVersion: phpVersion,
	}, nil
}

type ExtensionsPrinter struct {
	Extensions []domain.ExtensionInfo
	PHPVersion string
}

func (p *ExtensionsPrinter) Print() {
	if len(p.Extensions) == 0 {
		fmt.Println("No extensions found")
		return
	}

	if p.PHPVersion != "" {
		fmt.Printf("Available PHP extensions for PHP %s:\n\n", p.PHPVersion)
	} else {
		fmt.Println("Available PHP extensions:")
	}

	fmt.Println("  Name          Flag                Min PHP  Package")
	fmt.Println("  ------------  ------------------  -------  ------------")

	for _, ext := range p.Extensions {
		minPHP := "-"
		if ext.MinPHP != "" {
			minPHP = ext.MinPHP
		}

		pkg := "-"
		if ext.Package != "" {
			pkg = ext.Package
		}

		conflictNote := ""
		if ext.HasConflict {
			conflictNote = fmt.Sprintf(" (conflicts: %s)", strings.Join(ext.Conflicts, ", "))
		}

		fmt.Printf("  %-12s  %-19s  %-7s  %s%s\n", ext.Name, ext.Flag, minPHP, pkg, conflictNote)
	}

	fmt.Println()
	fmt.Println("Use: phpv install <version> --ext ext1,ext2,ext3")
}

type ExtensionValidateResult struct {
	Valid     []string
	Invalid   []string
	Conflicts [][]string
}

func (h *TerminalHandler) ValidateExtensions(extensions []string, phpVersion string) (*ExtensionValidateResult, error) {
	unknown, err := h.ExtensionRepo.ValidateExtensions(extensions, phpVersion)
	if err != nil {
		return nil, err
	}

	var invalid []string
	var conflictPairs [][]string

	if len(unknown) > 0 {
		invalid = unknown
	}

	if len(extensions) > 0 {
		_, conflictPairs = h.ExtensionRepo.CheckExtensionConflicts(extensions)
	}

	return &ExtensionValidateResult{
		Valid:     extensions,
		Invalid:   invalid,
		Conflicts: conflictPairs,
	}, nil
}

func (r *ExtensionValidateResult) HasErrors() bool {
	return len(r.Invalid) > 0 || len(r.Conflicts) > 0
}

func (r *ExtensionValidateResult) ErrorMessage() string {
	var msgs []string
	if len(r.Invalid) > 0 {
		msgs = append(msgs, fmt.Sprintf("Unknown extensions: %s", strings.Join(r.Invalid, ", ")))
	}
	if len(r.Conflicts) > 0 {
		var conflictStrs []string
		for _, pair := range r.Conflicts {
			conflictStrs = append(conflictStrs, fmt.Sprintf("%s conflicts with %s", pair[0], pair[1]))
		}
		msgs = append(msgs, strings.Join(conflictStrs, "; "))
	}
	return strings.Join(msgs, ". ")
}
