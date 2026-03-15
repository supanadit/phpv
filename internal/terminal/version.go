package terminal

import (
	"context"
	"fmt"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/supanadit/phpv/domain"
)

type VersionService interface {
	GetVersions(ctx context.Context) ([]domain.Version, error)
}

type VersionHandler struct {
	service VersionService
}

func NewVersionHandler(ctx context.Context, svc VersionService) bool {
	handler := &VersionHandler{
		service: svc,
	}

	// Parse flags first
	pflag.Parse()
	viper.BindPFlags(pflag.CommandLine)

	// Get positional arguments (non-flag arguments)
	args := pflag.Args()

	// Check if the first argument is "list"
	if len(args) > 0 && args[0] == "list" {
		handler.ListVersions(ctx)
		return true
	}

	return false
}

func (h *VersionHandler) ListVersions(ctx context.Context) {
	versions, err := h.service.GetVersions(ctx)
	if err != nil {
		fmt.Println("Error fetching versions:", err)
		return
	}

	// Get positional arguments (non-flag arguments)
	args := pflag.Args()
	var filterMajor, filterMinor *int

	// args[0] is "list", so check for args[1] (e.g., "8" or "8.3")
	if len(args) > 1 {
		var major, minor int
		n, err := fmt.Sscanf(args[1], "%d.%d", &major, &minor)
		if err == nil && n == 2 {
			filterMajor = &major
			filterMinor = &minor
		} else {
			n, err := fmt.Sscanf(args[1], "%d", &major)
			if err == nil && n == 1 {
				filterMajor = &major
			}
		}
	}

	for _, v := range versions {
		if filterMajor != nil && v.Major != *filterMajor {
			continue
		}
		if filterMinor != nil && v.Minor != *filterMinor {
			continue
		}
		println(h.formatVersion(v))
	}
}

func (h *VersionHandler) formatVersion(v domain.Version) string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}
