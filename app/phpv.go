package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/supanadit/phpv/build"
	"github.com/supanadit/phpv/dependency"
	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/download"
	"github.com/supanadit/phpv/internal/repository/memory"
	"github.com/supanadit/phpv/internal/terminal"
	"github.com/supanadit/phpv/internal/ui"
	"github.com/supanadit/phpv/prune"
	"github.com/supanadit/phpv/shell"
	"github.com/supanadit/phpv/version"
)

func main() {
	ctx := context.Background()

	// Configure viper to read environment variables
	viper.AutomaticEnv()
	// Set default PHPV_ROOT to $HOME/.phpv, respecting OS
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic("You must have a home directory set for phpv to work")
	}
	viper.SetDefault("PHPV_ROOT", filepath.Join(homeDir, ".phpv"))
	viper.SetDefault("PHP_SOURCE", "github")

	// Register and check help flag
	pflag.BoolP("help", "h", false, "Show help")
	pflag.BoolP("quiet", "q", false, "Suppress progress output (for CI)")
	pflag.BoolP("version", "v", false, "Show phpv version")
	pflag.BoolP("verbose", "V", false, "Enable verbose output")
	pflag.Parse()
	viper.BindPFlags(pflag.CommandLine)

	help, _ := pflag.CommandLine.GetBool("help")
	h, _ := pflag.CommandLine.GetBool("h")
	if help || h {
		terminal.NewNothingHandler()
		return
	}

	versionFlag, _ := pflag.CommandLine.GetBool("version")
	if versionFlag {
		if domain.AppGitCommit != "unknown" {
			fmt.Printf("phpv %s (%s)\n", domain.AppGitBranch, domain.AppGitCommit)
		} else {
			fmt.Printf("phpv %s\n", domain.AppVersion)
		}
		return
	}

	ui.GetUI()

	versionRepo := memory.NewVersionRepository()
	versionSvc := version.NewService(versionRepo)
	downloadSvc := download.NewService()
	buildSvc := build.NewService()
	pruneSvc := prune.NewService()
	shellSvc := shell.NewService()
	depSvc := dependency.NewService(viper.GetString("PHPV_ROOT"))

	// Shell command handlers (sh-use, sh-shell) must come first
	// as they're called by shell wrappers
	if !terminal.NewShellCommandHandler(ctx, shellSvc) {
		// Shell integration handlers (init, use, default, versions, which)
		if !terminal.NewShellHandler(ctx, shellSvc) {
			if !terminal.NewDownloadHandler(ctx, versionSvc, downloadSvc) {
				if !terminal.NewBuildHandler(ctx, versionSvc, buildSvc) {
					if !terminal.NewCleanHandler(ctx, versionSvc, terminal.NewCleanServiceAdapter(depSvc)) {
						if !terminal.NewCleanSourceHandler(ctx, pruneSvc) {
							if !terminal.NewUninstallHandler(ctx, terminal.NewUninstallServiceAdapter()) {
								if !terminal.NewPruneHandler(ctx, pruneSvc) {
									if !terminal.NewVersionHandler(ctx, versionSvc) {
										terminal.NewNothingHandler()
									}
								}
							}
						}
					}
				}
			}
		}
	}
}
