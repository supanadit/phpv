package terminal

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/supanadit/phpv/assembler"
	"github.com/supanadit/phpv/internal/repository"
	"github.com/supanadit/phpv/silo"
)

// fpmConcrete exposes the concrete assembler for FPM variant lookups.
func (h *PHPHandler) fpmConcrete() *assembler.FPMService {
	return assembler.NewFPMService(h.assemblerConcrete(), resolvePHPVRootForServer())
}

// siloPrefixer is the minimal prefix lookup the server manager needs; the
// concrete silo service implements it.
type siloPrefixer interface {
	PackagePrefix(name, version string) string
	PHPOutputPath(phpVersion string) string
}

// assemblerConcrete exposes the concrete assembler service for server paths
// that need methods beyond the AssemblerService interface (AssembleServer,
// FPMService construction).
func (h *PHPHandler) assemblerConcrete() *assembler.Service {
	return h.assemblerSvc.(*assembler.Service)
}

// siloConcrete exposes the concrete silo service for prefix lookups.
func (h *PHPHandler) siloConcrete() siloPrefixer {
	return h.siloSvc.(*silo.Service)
}

// repositoryResolveRoot proxies the repository root helper for server code.
func repositoryResolveRoot() string {
	return repository.ResolveRoot()
}

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
