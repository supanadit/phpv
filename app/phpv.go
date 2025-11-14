package main

import (
	"context"

	"github.com/spf13/pflag"
	"github.com/supanadit/phpv/internal/repository/memory"
	"github.com/supanadit/phpv/internal/terminal"
	"github.com/supanadit/phpv/version"
)

func main() {
	ctx := context.Background()

	// Register and check help flag
	pflag.BoolP("help", "h", false, "Show help")
	pflag.Parse()
	help, _ := pflag.CommandLine.GetBool("help")
	h, _ := pflag.CommandLine.GetBool("h")
	if help || h {
		terminal.NewNothingHandler()
		return
	}

	versionRepo := memory.NewVersionRepository()
	svc := version.NewService(versionRepo)

	if !terminal.NewVersionHandler(ctx, svc) {
		terminal.NewNothingHandler()
	}
}
