package terminal

import (
	"context"
	"fmt"

	"github.com/spf13/pflag"
	"github.com/supanadit/phpv/domain"
)

type DownloadService interface {
	Download(ctx context.Context, version domain.Version) error
	FindMatchingVersion(ctx context.Context, versions []domain.Version, major int, minor *int, patch *int) (domain.Version, error)
	GetSourcesDir() string
}

type DownloadHandler struct {
	versionService  VersionService
	downloadService DownloadService
}

func NewDownloadHandler(ctx context.Context, versionSvc VersionService, downloadSvc DownloadService) bool {
	handler := &DownloadHandler{
		versionService:  versionSvc,
		downloadService: downloadSvc,
	}

	// Parse flags first
	pflag.Parse()

	// Get positional arguments (non-flag arguments)
	args := pflag.Args()

	// Check if the first argument is "download"
	if len(args) > 0 && args[0] == "download" {
		if len(args) < 2 {
			fmt.Println("Error: Please specify a version to download")
			fmt.Println("Examples:")
			fmt.Println("  phpv download 8")
			fmt.Println("  phpv download 8.3")
			fmt.Println("  phpv download 8.4.14")
			return true
		}
		handler.DownloadVersion(ctx, args[1])
		return true
	}

	return false
}

func (h *DownloadHandler) DownloadVersion(ctx context.Context, versionInput string) {
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
	version, err := h.downloadService.FindMatchingVersion(ctx, versions, major, minorPtr, patchPtr)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	// Download the version
	if err := h.downloadService.Download(ctx, version); err != nil {
		fmt.Println("Error:", err)
		return
	}
}
