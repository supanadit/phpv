package terminal

import (
	"context"
	"fmt"

	"github.com/spf13/pflag"
	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/storage"
)

type UninstallService interface {
	Remove(version domain.Version) error
	RemoveDependencies(version domain.Version) error
	List(ctx context.Context) ([]domain.Version, error)
	GetDefault(ctx context.Context) (*domain.Version, error)
}

type UninstallHandler struct {
	uninstallService UninstallService
}

func NewUninstallHandler(ctx context.Context, uninstallSvc UninstallService) bool {
	handler := &UninstallHandler{
		uninstallService: uninstallSvc,
	}

	pflag.Parse()
	args := pflag.Args()

	if len(args) > 0 && args[0] == "uninstall" {
		if len(args) < 2 {
			fmt.Println("Error: Please specify a version to uninstall")
			fmt.Println("Examples:")
			fmt.Println("  phpv uninstall 8.3")
			fmt.Println("  phpv uninstall 8.3.0")
			fmt.Println("  phpv uninstall 4.0.0")
			return true
		}

		versionInput := args[1]
		handler.UninstallVersion(ctx, versionInput)
		return true
	}

	return false
}

func (h *UninstallHandler) UninstallVersion(ctx context.Context, versionInput string) {
	version, err := h.findInstalledVersion(ctx, versionInput)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	defaultVersion, err := h.uninstallService.GetDefault(ctx)
	if err != nil {
		fmt.Println("Error checking default version:", err)
		return
	}

	if defaultVersion != nil && version.String() == defaultVersion.String() {
		fmt.Printf("Error: Cannot uninstall PHP %s - it is set as default.\n", version.String())
		fmt.Println("Run 'phpv default none' first to deselect it, then try again.")
		return
	}

	fmt.Printf("Uninstalling PHP %s...\n", version.String())
	fmt.Println()

	if err := h.uninstallService.Remove(version); err != nil {
		fmt.Println("Error removing PHP version:", err)
		return
	}
	fmt.Printf("✓ Removed PHP %s from versions directory\n", version.String())

	if err := h.uninstallService.RemoveDependencies(version); err != nil {
		fmt.Println("Warning: Error removing dependencies:", err)
	} else {
		fmt.Printf("✓ Removed dependencies for PHP %s\n", version.String())
	}

	fmt.Println()
	fmt.Printf("✓ Successfully uninstalled PHP %s\n", version.String())
}

func (h *UninstallHandler) findInstalledVersion(ctx context.Context, versionInput string) (domain.Version, error) {
	versions, err := h.uninstallService.List(ctx)
	if err != nil {
		return domain.Version{}, err
	}

	for _, v := range versions {
		if matchesVersion(v, versionInput) {
			return v, nil
		}
	}

	return domain.Version{}, fmt.Errorf("PHP version %s is not installed", versionInput)
}

func matchesVersion(v domain.Version, spec string) bool {
	specParts := splitVersion(spec)
	vParts := []int{v.Major, v.Minor, v.Patch}

	if len(specParts) > len(vParts) {
		return false
	}

	for i, part := range specParts {
		if vParts[i] != part {
			return false
		}
	}

	return true
}

func splitVersion(s string) []int {
	var result []int
	var current int
	started := false

	for _, c := range s {
		if c >= '0' && c <= '9' {
			current = current*10 + int(c-'0')
			started = true
		} else if started {
			result = append(result, current)
			current = 0
			started = false
		}
	}

	if started {
		result = append(result, current)
	}

	return result
}

type UninstallServiceAdapter struct {
	installedStorage *storage.InstalledStorage
	defaultStorage   *storage.DefaultVersionStorage
}

func NewUninstallServiceAdapter() *UninstallServiceAdapter {
	return &UninstallServiceAdapter{
		installedStorage: storage.NewInstalledStorage(),
		defaultStorage:   storage.NewDefaultVersionStorage(),
	}
}

func (s *UninstallServiceAdapter) Remove(version domain.Version) error {
	return s.installedStorage.Remove(version)
}

func (s *UninstallServiceAdapter) RemoveDependencies(version domain.Version) error {
	return s.installedStorage.RemoveDependencies(version)
}

func (s *UninstallServiceAdapter) List(ctx context.Context) ([]domain.Version, error) {
	return s.installedStorage.List(ctx)
}

func (s *UninstallServiceAdapter) GetDefault(ctx context.Context) (*domain.Version, error) {
	return s.defaultStorage.Get()
}
