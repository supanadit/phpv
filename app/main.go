package main

import (
	"context"

	"github.com/supanadit/phpv/internal/repository/memory"
	"github.com/supanadit/phpv/internal/terminal"
	"github.com/supanadit/phpv/version"
)

func main() {
	ctx := context.Background()

	versionRepo := memory.NewVersionRepository()
	svc := version.NewService(versionRepo)

	if !terminal.NewVersionHandler(ctx, svc) {
		terminal.NewNothingHandler()
	}
}
