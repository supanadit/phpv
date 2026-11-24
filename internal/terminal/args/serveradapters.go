package args

import (
	"strings"

	"github.com/spf13/cobra"
)

// fpmConcrete exposes the concrete assembler for FPM variant lookups.
// siloPrefixer is the minimal prefix lookup the server manager needs; the
// concrete silo service implements it.
type siloPrefixer interface {
	PackagePrefix(name, version string) string
	PHPOutputPath(phpVersion string) string
}

// assemblerConcrete exposes the concrete assembler service for server paths
// that need methods beyond the AssemblerService interface (AssembleServer,
// FPMService construction).
// siloConcrete exposes the concrete silo service for prefix lookups.
// versionNameCompletion completes installed php versions for --php.
func (h *PHPHandler) versionNameCompletion(toComplete string) ([]string, cobra.ShellCompDirective) {
	installed, err := h.installedPHPVersions()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	var out []string
	for _, v := range installed {
		if strings.HasPrefix(v, toComplete) {
			out = append(out, v)
		}
	}
	return out, cobra.ShellCompDirectiveNoFileComp
}
