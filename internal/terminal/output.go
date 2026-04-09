package terminal

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/supanadit/phpv/domain"
)

const (
	minBoxWidth = 64
)

func PrintBox(width int, lines []string) {
	border := "+" + strings.Repeat("=", width) + "+"
	middle := "+" + strings.Repeat("=", width) + "+"
	bottom := "+" + strings.Repeat("=", width) + "+"

	fmt.Println(border)
	for i, line := range lines {
		if i == 1 {
			fmt.Println(middle)
		}
		padding := max(width-len(line), 0)
		fmt.Println("|" + line + strings.Repeat(" ", padding) + "|")
	}
	fmt.Println(bottom)
}

type InstallSummary struct {
	Version string
	Forge   domain.Forge
}

func (s *InstallSummary) GetBinaryPath() string {
	return filepath.Join(s.Forge.Prefix, "bin", "php")
}

func (s *InstallSummary) Print() {
	binaryPath := s.GetBinaryPath()

	labelVersion := "Version:"
	labelBinary := "Binary:"

	contentWidth := minBoxWidth - 2

	headerWidth := len(labelVersion) + 1 + len(s.Version)
	binaryWidth := len(labelBinary) + 1 + len(binaryPath)

	boxWidth := minBoxWidth
	if binaryWidth > contentWidth {
		boxWidth = binaryWidth + 2
		contentWidth = boxWidth - 2
	}
	if headerWidth > contentWidth {
		boxWidth = headerWidth + 2
		contentWidth = boxWidth - 2
	}

	displayBinaryPath := binaryPath
	availableBinaryContent := contentWidth - len(labelBinary) - 1
	if len(binaryPath) > availableBinaryContent {
		displayBinaryPath = "..." + binaryPath[len(binaryPath)-availableBinaryContent+3:]
	}

	header := "                    PHP Installation Summary"
	versionContent := fmt.Sprintf("%s %s", labelVersion, s.Version)
	binaryContent := fmt.Sprintf("%s %s", labelBinary, displayBinaryPath)

	fmt.Println()
	PrintBox(boxWidth, []string{
		"",
		header,
		versionContent,
		binaryContent,
	})
}

type VersionsPrinter struct {
	Versions   []VersionInfo
	DefaultVer string
}

func (p *VersionsPrinter) Print() {
	if len(p.Versions) == 0 {
		fmt.Println("No PHP versions installed")
		return
	}

	fmt.Println("Installed PHP versions:")
	for _, v := range p.Versions {
		if v.IsSystem {
			fmt.Printf("    system (%s -> %s)\n", v.SystemPath, v.Version)
		} else if v.IsDefault {
			fmt.Printf("  * %s (default)\n", v.Version)
		} else {
			fmt.Printf("    %s\n", v.Version)
		}
	}
}

func PrintInstallSummary(version string, forge domain.Forge) {
	summary := &InstallSummary{
		Version: version,
		Forge:   forge,
	}
	summary.Print()
}
