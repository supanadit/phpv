package terminal

import (
	"context"
	"fmt"

	"github.com/spf13/pflag"
	"github.com/supanadit/phpv/domain"
)

type BuildService interface {
	Build(ctx context.Context, version domain.Version) error
	FindMatchingVersion(ctx context.Context, versions []domain.Version, major int, minor *int, patch *int) (domain.Version, error)
	GetVersionsDir() string
	CheckCompiler() error
}

type BuildHandler struct {
	versionService VersionService
	buildService   BuildService
}

func NewBuildHandler(ctx context.Context, versionSvc VersionService, buildSvc BuildService) bool {
	handler := &BuildHandler{
		versionService: versionSvc,
		buildService:   buildSvc,
	}

	// Parse flags first
	pflag.Parse()

	// Get positional arguments (non-flag arguments)
	args := pflag.Args()

	// Check if the first argument is "build" or "install"
	if len(args) > 0 && (args[0] == "build" || args[0] == "install") {
		if len(args) < 2 {
			fmt.Println("Error: Please specify a version to build")
			fmt.Println("Examples:")
			fmt.Println("  phpv build 8")
			fmt.Println("  phpv build 8.3")
			fmt.Println("  phpv build 8.4.14")
			return true
		}
		handler.BuildVersion(ctx, args[1])
		return true
	}

	return false
}

func (h *BuildHandler) BuildVersion(ctx context.Context, versionInput string) {
	// Parse version input
	var major, minor, patch int
	var minorPtr, patchPtr *int

	n, err := fmt.Sscanf(versionInput, "%d.%d.%d", &major, &minor, &patch)
	if err == nil && n == 3 {
		// Specific version like 8.4.14
		minorPtr = &minor
		patchPtr = &patch
	} else {
		n, err = fmt.Sscanf(versionInput, "%d.%d", &major, &minor)
		if err == nil && n == 2 {
			// Major.Minor like 8.3
			minorPtr = &minor
		} else {
			n, err = fmt.Sscanf(versionInput, "%d", &major)
			if err != nil || n != 1 {
				fmt.Printf("Error: Invalid version format '%s'\n", versionInput)
				fmt.Println("Valid formats: 8, 8.3, or 8.4.14")
				return
			}
			// Major only like 8
		}
	}

	// Get all available versions
	versions, err := h.versionService.GetVersions(ctx)
	if err != nil {
		fmt.Println("Error fetching versions:", err)
		return
	}

	// Find matching version
	version, err := h.buildService.FindMatchingVersion(ctx, versions, major, minorPtr, patchPtr)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	// Check compiler before building
	if err := h.buildService.CheckCompiler(); err != nil {
		fmt.Println("Error:", err)
		fmt.Println()
		fmt.Println("If you're using a custom toolchain (PHPV_TOOLCHAIN_CC), make sure it's installed.")
		fmt.Println("Otherwise, LLVM will be automatically downloaded for building PHP.")
		return
	}

	// Build the version
	if err := h.buildService.Build(ctx, version); err != nil {
		fmt.Println("Error:", err)
		return
	}
}
