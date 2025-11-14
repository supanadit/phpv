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

func NewVersionHandler(ctx context.Context, svc VersionService) {
	handler := &VersionHandler{
		service: svc,
	}

	// Define the flag before looking it up
	pflag.Bool("list-versions", false, "List all available PHP versions")
	pflag.Parse()
	viper.BindPFlags(pflag.CommandLine)

	if viper.GetBool("list-versions") {
		handler.ListVersions(ctx)
	}
}

func (h *VersionHandler) ListVersions(ctx context.Context) {
	versions, err := h.service.GetVersions(ctx)
	if err != nil {
		fmt.Println("Error fetching versions:", err)
	}

	for _, v := range versions {
		println(h.formatVersion(v))
	}
}

func (h *VersionHandler) formatVersion(v domain.Version) string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}
