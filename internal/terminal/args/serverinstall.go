package args

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/pkg/compiler"
	"github.com/supanadit/phpv/pkg/progress"
	webidentity "github.com/supanadit/phpv/pkg/webidentity"
)

func (h *PHPHandler) serverinstall() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "install <caddy|nginx|apache> [version]",
		Short:             "Install a webserver binary (does not change the PHP-version association)",
		Args:              cobra.RangeArgs(1, 2),
		ValidArgsFunction: serverNameCompletion,
		RunE: func(cmd *cobra.Command, args []string) error {
			server := strings.ToLower(args[0])
			if !webidentity.IsValidServerName(server) {
				return invalidServerName(server)
			}
			version := ""
			if len(args) == 2 {
				version = args[1]
			}
			verbose, _ := cmd.Flags().GetBool("verbose")
			jobs, _ := cmd.Flags().GetInt("jobs")
			force, _ := cmd.Flags().GetBool("force")
			tc, err := compiler.Resolve(h.compilerName())
			if err != nil {
				return err
			}

			banner := server
			exact := ""
			if resolved, err := h.assemblerSvc.ResolveVersion(webidentity.ServerPackageName(server), version); err == nil {
				exact = resolved
				banner = fmt.Sprintf("%s %s", server, exact)
			}
			fmt.Printf("Installing %s...\n\n", banner)

			var progress progress.Func
			stopProgress := func() {}
			if !verbose {
				progress, stopProgress = newProgressReporter()
			}
			if force && exact != "" {
				err = h.rebuildServer(h.ctx, rebuildSpec{server, exact, verbose, jobs, tc, progress})
			} else {
				_, err = h.serverAsbl.AssembleServer(h.ctx, domain.ServerAssembleSpec{Name: server, Version: version, Verbose: verbose, Jobs: jobs, TC: domainToolchain(tc), Progress: progress})
			}
			stopProgress()
			if err != nil {
				fmt.Println()
				return err
			}
			if !verbose {
				fmt.Println()
			}
			return nil
		},
	}
	cmd.Flags().Bool("verbose", false, "Show full build output")
	cmd.Flags().Int("jobs", 0, "Number of parallel build jobs (default: CPU count)")
	cmd.Flags().Bool("force", false, "Rebuild even when already installed (also upgrades httpd MPM support)")

	return cmd
}
