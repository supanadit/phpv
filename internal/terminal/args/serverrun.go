package args

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/supanadit/phpv/domain"
	webidentity "github.com/supanadit/phpv/pkg/webidentity"
)

func (h *PHPHandler) serverrun() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run <dir>",
		Short: "Serve a directory through the webserver and PHP for the active PHP version",
		Long: `Serve a directory (replaces ` + "`php -S localhost:... -t public`" + `).

The PHP version follows the active version (` + "`phpv use`" + `, ` + "`.php-version`" + `,
default), exactly like ` + "`php`" + ` itself. The webserver is picked per PHP version:
` + "`--server`" + `, the server associated with that version (` + "`phpv server use`" + `),
` + "`PHPV_DEFAULT_SERVER`" + `, the global default (` + "`phpv server default`" + `), then the
first installed server. No association is required to cmd.

APIs: ` + "`--sapi fpm`" + ` (default, FastCGI) or ` + "`--sapi modphp`" + ` (Apache's
embedded PHP module; requires apache). Without ` + "`--sapi`" + `, the association for
this PHP version, ` + "`PHPV_DEFAULT_SAPI`" + `, then ` + "`phpv server default`" + ` apply.

Examples:
  phpv server run .                          # serve cwd on localhost:8080
  phpv server run . --host myapp.local       # vhost + /etc/hosts entry
  phpv server run . --server nginx --version 1.26.3   # this run only
  phpv server run . --server apache --sapi modphp      # embedded Apache PHP
  phpv server run . --php 7.4                # pin a PHP version explicitly
  phpv server run . -d                       # detach; manage with phpv server stop/list`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := filepath.Abs(args[0])
			if err != nil {
				return err
			}
			server, _ := cmd.Flags().GetString("server")
			serverVersion, _ := cmd.Flags().GetString("version")
			phpVersion, _ := cmd.Flags().GetString("php")
			sapiFlag, _ := cmd.Flags().GetString("sapi")
			host, _ := cmd.Flags().GetString("host")
			port, _ := cmd.Flags().GetInt("port")
			detach, _ := cmd.Flags().GetBool("detach")
			verbose, _ := cmd.Flags().GetBool("verbose")
			jobs, _ := cmd.Flags().GetInt("jobs")

			// Resolution order: --server → per-version association →
			// PHPV_DEFAULT_SERVER → global default → first installed.
			resolvedPHP, err := h.resolveServerPHPVersion(phpVersion)
			if err != nil {
				return err
			}
			server, serverSource, err := h.resolveServerForPHP(server, resolvedPHP)
			if err != nil {
				return err
			}
			sapi, err := h.resolveSAPIForPHP(sapiFlag, server, resolvedPHP)
			if err != nil {
				return err
			}

			m, err := h.serverManager()
			if err != nil {
				return err
			}

			// A cold run builds the php-fpm variant first; EnsureFPM reports
			// each stage through the spinner.

			inst, err := m.Start(h.ctx, domain.ServerRunOptions{
				Server:        server,
				ServerVersion: serverVersion,
				PHPVersion:    resolvedPHP,
				SAPI:          sapi,
				Dir:           dir,
				Host:          host,
				Port:          port,
				Detach:        detach,
				Verbose:       verbose,
				Jobs:          jobs,
			})
			if err != nil {
				return err
			}

			fmt.Printf("✓ %s %s serving %s with PHP %s (%s)\n", inst.Server, inst.ServerVersion, inst.Dir, inst.PHPVersion, sapiLabel(inst.SAPI))
			fmt.Printf("  url:     %s\n", webidentity.InstanceURL(inst.Host, inst.Port))
			if inst.Host != "" && inst.Host != "localhost" {
				fmt.Printf("  vhost:   %s (via /etc/hosts)\n", inst.Host)
			}
			if serverSource == serverFromFlag {
				if pref, ok := h.serverPreference(inst.PHPVersion); !ok || pref != inst.Server {
					fmt.Printf("  note:    --server applies to this run only (make it this version's default: phpv server use %s --php %s)\n", inst.Server, inst.PHPVersion)
				}
			}
			if serverSource == serverFromFallback && len(h.installedServers()) > 1 {
				fmt.Printf("  note:    using %s by default (override with %s=<name> or phpv server use <name>)\n", inst.Server, defaultServerEnv)
			}
			if detach {
				fmt.Printf("  detached: instance %s — stop it with: phpv server stop %s\n", inst.ID, inst.ID)
				return nil
			}
			fmt.Println("  press Ctrl-C to stop")
			// Block until interrupted; then tear down cleanly.
			<-h.ctx.Done()
			fmt.Println("\nStopping...")
			return m.Stop(inst)
		},
	}
	cmd.Flags().String("server", "", "Webserver to use (caddy, nginx, apache); default: per-version association, then PHPV_DEFAULT_SERVER, then global default")
	cmd.Flags().String("version", "", "Webserver version to use; default from config or newest installed")
	cmd.Flags().String("php", "", "PHP version to serve with (e.g. 5.6, 8.3); default: active/default")
	cmd.Flags().String("sapi", "", "PHP SAPI: fpm (default) or modphp (apache only)")
	cmd.Flags().String("host", "", "Hostname for auto vhost (adds /etc/hosts entry)")
	cmd.Flags().Int("port", 0, "TCP port (default: 8080, auto-increments when busy)")
	cmd.Flags().BoolP("detach", "d", false, "Run in the background")
	cmd.Flags().Bool("verbose", false, "Show full build output (when a PHP runtime must be built)")
	cmd.Flags().Int("jobs", 0, "Number of parallel build jobs (default: CPU count)")
	_ = cmd.RegisterFlagCompletionFunc("server", serverNameCompletion)
	_ = cmd.RegisterFlagCompletionFunc("sapi", sapiCompletion)
	_ = cmd.RegisterFlagCompletionFunc("php", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return h.versionNameCompletion(toComplete)
	})

	return cmd
}
