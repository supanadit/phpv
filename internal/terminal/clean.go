package terminal

import (
	"context"
	"fmt"

	"github.com/spf13/pflag"
	"github.com/supanadit/phpv/dependency"
	"github.com/supanadit/phpv/domain"
)

type CleanService interface {
	Clean(phpVersion domain.Version, depName string) error
	FindMatchingVersion(ctx context.Context, versions []domain.Version, major int, minor *int, patch *int) (domain.Version, error)
}

type CleanHandler struct {
	versionService VersionService
	cleanService   CleanService
}

func NewCleanHandler(ctx context.Context, versionSvc VersionService, cleanSvc CleanService) bool {
	handler := &CleanHandler{
		versionService: versionSvc,
		cleanService:   cleanSvc,
	}

	pflag.Parse()
	args := pflag.Args()

	if len(args) > 0 && args[0] == "clean" {
		if len(args) < 2 {
			fmt.Println("Error: Please specify a version to clean")
			fmt.Println("Examples:")
			fmt.Println("  phpv clean 8.3")
			fmt.Println("  phpv clean 8.3 flex")
			fmt.Println("  phpv clean 4.0.0")
			fmt.Println("  phpv clean 4.0.0 flex")
			return true
		}

		versionInput := args[1]
		var depName string
		if len(args) > 2 {
			depName = args[2]
		}

		handler.CleanVersion(ctx, versionInput, depName)
		return true
	}

	return false
}

func (h *CleanHandler) CleanVersion(ctx context.Context, versionInput string, depName string) {
	var major, minor, patch int
	var minorPtr, patchPtr *int

	n, err := fmt.Sscanf(versionInput, "%d.%d.%d", &major, &minor, &patch)
	if err == nil && n == 3 {
		minorPtr = &minor
		patchPtr = &patch
	} else {
		n, err = fmt.Sscanf(versionInput, "%d.%d", &major, &minor)
		if err == nil && n == 2 {
			minorPtr = &minor
		} else {
			n, err = fmt.Sscanf(versionInput, "%d", &major)
			if err != nil || n != 1 {
				fmt.Printf("Error: Invalid version format '%s'\n", versionInput)
				fmt.Println("Valid formats: 8, 8.3, or 8.4.14")
				return
			}
		}
	}

	versions, err := h.versionService.GetVersions(ctx)
	if err != nil {
		fmt.Println("Error fetching versions:", err)
		return
	}

	version, err := h.cleanService.FindMatchingVersion(ctx, versions, major, minorPtr, patchPtr)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println()
	if depName != "" {
		fmt.Printf("Cleaning dependency '%s' for PHP %s...\n", depName, version.String())
	} else {
		fmt.Printf("Cleaning all dependencies for PHP %s...\n", version.String())
	}
	fmt.Println()

	if err := h.cleanService.Clean(version, depName); err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println()
	if depName != "" {
		fmt.Printf("✓ Run 'phpv install %s' to rebuild the dependency\n", version.String())
	} else {
		fmt.Printf("✓ Run 'phpv install %s' to rebuild all dependencies\n", version.String())
	}
}

type CleanServiceAdapter struct {
	depSvc *dependency.Service
}

func NewCleanServiceAdapter(depSvc *dependency.Service) *CleanServiceAdapter {
	return &CleanServiceAdapter{depSvc: depSvc}
}

func (s *CleanServiceAdapter) Clean(phpVersion domain.Version, depName string) error {
	return s.depSvc.Clean(phpVersion, depName)
}

func (s *CleanServiceAdapter) FindMatchingVersion(ctx context.Context, versions []domain.Version, major int, minor *int, patch *int) (domain.Version, error) {
	var matchedVersion domain.Version
	found := false

	for _, v := range versions {
		if v.Major != major {
			continue
		}

		if minor != nil && v.Minor != *minor {
			continue
		}

		if patch != nil && v.Patch != *patch {
			continue
		}

		matchedVersion = v
		found = true
		break
	}

	if !found {
		return domain.Version{}, fmt.Errorf("version %d.%d.%d not found", major, func() int {
			if minor != nil {
				return *minor
			}
			return 0
		}(), func() int {
			if patch != nil {
				return *patch
			}
			return 0
		}())
	}

	return matchedVersion, nil
}
