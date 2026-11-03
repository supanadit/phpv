package terminal

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/supanadit/phpv/assembler"
	"github.com/supanadit/phpv/compiler"
	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/webserver"
)

// ServerManagerDeps adapts the PHPHandler's services to webserver.Manager.
// Built lazily so command construction stays cheap.
func (h *PHPHandler) serverManager() (*webserver.Manager, error) {
	root := resolvePHPVRootForServer()
	m := webserver.NewManager(webserver.Deps{
		Root: root,
		ResolveVersion: func(constraint string) (string, error) {
			if constraint == "" {
				if def, err := h.siloSvc.GetDefault(); err == nil && def != "" {
					return def, nil
				}
				return "", fmt.Errorf("no default PHP version (set one with `phpv default <version>` or pass --php)")
			}
			return h.resolveVersion(constraint)
		},
		EnsureFPM: func(ctx context.Context, phpVersion string, verbose bool, jobs int) (string, error) {
			fpm := assembler.NewFPMService(h.assemblerConcrete(), root)
			tc, err := compiler.Resolve(h.compilerName())
			if err != nil {
				return "", err
			}
			return fpm.EnsureFPM(ctx, phpVersion, verbose, jobs, tc, "")
		},
		CLIPrefix: func(phpVersion string) string {
			return h.siloConcrete().PHPOutputPath(phpVersion)
		},
		InstalledServerVersion: func(server, constraint string) (string, error) {
			return h.resolveInstalledServerVersion(server, constraint)
		},
		ServerPrefix: func(server, version string) string {
			return h.siloSvc.PackagePrefix(server, version)
		},
	})
	return m, nil
}

// resolveInstalledServerVersion resolves a server version constraint against
// installed server packages. Empty constraint picks: config default → newest
// installed. Unknown/missing servers produce actionable errors.
func (h *PHPHandler) resolveInstalledServerVersion(server, constraint string) (string, error) {
	pkgDir := filepath.Join(resolvePHPVRootForServer(), "packages", server)
	entries, err := os.ReadDir(pkgDir)
	if err != nil || len(entries) == 0 {
		return "", fmt.Errorf("%s is not installed (run: phpv server install %s)", server, server)
	}
	var installed []string
	for _, e := range entries {
		if e.IsDir() {
			installed = append(installed, e.Name())
		}
	}
	if constraint == "" {
		// The configured server_version belongs to the configured server;
		// never apply it to a server chosen by other means (flag override
		// or first-installed fallback).
		if cfgServer, _ := h.configSvc.Get("server"); cfgServer == server {
			constraint, _ = h.configSvc.Get("server_version")
		}
	}
	if constraint == "" {
		// Newest installed wins.
		best := installed[0]
		for _, v := range installed[1:] {
			if domain.CompareVersions(v, best) > 0 {
				best = v
			}
		}
		return best, nil
	}
	// Exact match first, then range match.
	for _, v := range installed {
		if v == constraint {
			return v, nil
		}
	}
	for _, v := range installed {
		if domain.MatchVersionRange(constraint, v) {
			return v, nil
		}
	}
	return "", fmt.Errorf("%s %s is not installed (have: %s)", server, constraint, strings.Join(installed, ", "))
}

// compilerName returns the configured compiler for builds.
func (h *PHPHandler) compilerName() string {
	name, _ := h.configSvc.Get("compiler")
	return name
}

// resolvePHPVRootForServer returns the phpv root used for server state.
func resolvePHPVRootForServer() string {
	return repositoryResolveRoot()
}

func (h *PHPHandler) serverCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "server",
		Short: "Install and run local webservers (caddy, nginx, apache) serving PHP via FastCGI",
	}

	install := &cobra.Command{
		Use:               "install <caddy|nginx|apache> [version]",
		Short:             "Install a webserver into the phpv silo (multi-version)",
		Args:              cobra.RangeArgs(1, 2),
		ValidArgsFunction: serverNameCompletion,
		RunE: func(cmd *cobra.Command, args []string) error {
			server := strings.ToLower(args[0])
			if !domain.IsValidServerName(server) {
				return fmt.Errorf("unknown server %q (available: %s)", server, strings.Join(domain.ServerNames(), ", "))
			}
			version := ""
			if len(args) == 2 {
				version = args[1]
			}
			verbose, _ := cmd.Flags().GetBool("verbose")
			jobs, _ := cmd.Flags().GetInt("jobs")
			tc, err := compiler.Resolve(h.compilerName())
			if err != nil {
				return err
			}
			_, err = h.assemblerConcrete().AssembleServer(h.ctx, server, version, verbose, jobs, tc)
			return err
		},
	}
	install.Flags().Bool("verbose", false, "Show full build output")
	install.Flags().Int("jobs", 0, "Number of parallel build jobs (default: CPU count)")

	run := &cobra.Command{
		Use:   "run <dir>",
		Short: "Serve a directory through the configured webserver + PHP-FPM",
		Long: `Serve a directory (replaces ` + "`php -S localhost:... -t public`" + `).

Examples:
  phpv server run .                          # serve cwd on localhost:8080
  phpv server run . --host myapp.local       # vhost + /etc/hosts entry
  phpv server run . --server nginx --version 1.26.3
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
			host, _ := cmd.Flags().GetString("host")
			port, _ := cmd.Flags().GetInt("port")
			detach, _ := cmd.Flags().GetBool("detach")
			verbose, _ := cmd.Flags().GetBool("verbose")
			jobs, _ := cmd.Flags().GetInt("jobs")

			// Resolution order: flags → config → first installed.
			if server == "" {
				server, _ = h.configSvc.Get("server")
			}
			if server == "" {
				server, err = h.firstInstalledServer()
				if err != nil {
					return err
				}
				fmt.Printf("No server configured; using %s (set a default: phpv config set server %s)\n", server, server)
			}
			if !domain.IsValidServerName(server) {
				return fmt.Errorf("unknown server %q (available: %s)", server, strings.Join(domain.ServerNames(), ", "))
			}

			m, err := h.serverManager()
			if err != nil {
				return err
			}

			// First `run` for a PHP version builds its php-fpm variant — a
			// one-time, multi-minute compile. Say so up front so the wait is
			// not mistaken for a hang. (Best-effort resolution: an
			// unresolvable version surfaces properly inside Start.)
			if resolvedPHP, err := h.resolveVersion(phpVersion); err == nil && resolvedPHP != "" {
				if h.fpmConcrete().FPMBinary(resolvedPHP) == "" {
					fmt.Printf("First run for PHP %s: building the php-fpm variant (one-time, a few minutes)…\n", resolvedPHP)
				}
			}

			inst, err := m.Start(h.ctx, webserver.RunOptions{
				Server:        server,
				ServerVersion: serverVersion,
				PHPVersion:    phpVersion,
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

			fmt.Printf("✓ %s %s serving %s with PHP %s (FPM)\n", inst.Server, inst.ServerVersion, inst.Dir, inst.PHPVersion)
			fmt.Printf("  url:     %s\n", inst.URL())
			if inst.Host != "" && inst.Host != "localhost" {
				fmt.Printf("  vhost:   %s (via /etc/hosts)\n", inst.Host)
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
	run.Flags().String("server", "", "Webserver to use (caddy, nginx, apache); default from config or first installed")
	run.Flags().String("version", "", "Webserver version to use; default from config or newest installed")
	run.Flags().String("php", "", "PHP version to serve with (e.g. 5.6, 8.3); default: active/default")
	run.Flags().String("host", "", "Hostname for auto vhost (adds /etc/hosts entry)")
	run.Flags().Int("port", 0, "TCP port (default: 8080, auto-increments when busy)")
	run.Flags().BoolP("detach", "d", false, "Run in the background")
	run.Flags().Bool("verbose", false, "Show full build output (when php-fpm must be built)")
	run.Flags().Int("jobs", 0, "Number of parallel build jobs (default: CPU count)")
	_ = run.RegisterFlagCompletionFunc("server", serverNameCompletion)
	_ = run.RegisterFlagCompletionFunc("php", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return h.versionNameCompletion(toComplete)
	})

	stop := &cobra.Command{
		Use:   "stop [instance-id]",
		Short: "Stop a running server instance (or all when omitted)",
		Args:  cobra.MaximumNArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			m, err := h.serverManager()
			if err != nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			running, _ := m.Running()
			var ids []string
			for _, inst := range running {
				ids = append(ids, inst.ID)
			}
			return ids, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			m, err := h.serverManager()
			if err != nil {
				return err
			}
			if len(args) == 1 {
				inst, err := m.StopByID(args[0])
				if err != nil {
					return err
				}
				fmt.Printf("✓ stopped %s (%s:%d)\n", inst.ID, inst.Server, inst.Port)
				return nil
			}
			running, err := m.Running()
			if err != nil {
				return err
			}
			if len(running) == 0 {
				fmt.Println("no running instances")
				return nil
			}
			for _, inst := range running {
				if err := m.Stop(inst); err != nil {
					return err
				}
				fmt.Printf("✓ stopped %s (%s:%d)\n", inst.ID, inst.Server, inst.Port)
			}
			return nil
		},
	}

	list := &cobra.Command{
		Use:   "list",
		Short: "List installed webservers and running instances",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			root := resolvePHPVRootForServer()
			fmt.Println("Installed webservers:")
			any := false
			for _, name := range domain.ServerNames() {
				pkgDir := filepath.Join(root, "packages", name)
				entries, err := os.ReadDir(pkgDir)
				if err != nil {
					continue
				}
				var versions []string
				for _, e := range entries {
					if e.IsDir() {
						versions = append(versions, e.Name())
					}
				}
				if len(versions) == 0 {
					continue
				}
				any = true
				fmt.Printf("  %-8s %s\n", name, strings.Join(versions, ", "))
			}
			if !any {
				fmt.Println("  (none — try: phpv server install caddy)")
			}

			m, err := h.serverManager()
			if err != nil {
				return err
			}
			if _, err := m.Prune(); err != nil {
				return err
			}
			instances, err := m.List()
			if err != nil {
				return err
			}
			fmt.Println("\nInstances:")
			if len(instances) == 0 {
				fmt.Println("  (none)")
				return nil
			}
			for _, inst := range instances {
				state := "stopped"
				if m.Store().Running(inst) {
					state = "running"
				}
				fmt.Printf("  %-28s %-8s %s  php-%s  %s  %s\n", inst.ID, state, inst.Server, inst.PHPVersion, inst.URL(), inst.Dir)
			}
			return nil
		},
	}

	cmd.AddCommand(install, run, stop, list)
	return cmd
}

// firstInstalledServer returns the first server with an installed package,
// in preference order.
func (h *PHPHandler) firstInstalledServer() (string, error) {
	root := resolvePHPVRootForServer()
	for _, name := range domain.ServerNames() {
		if entries, err := os.ReadDir(filepath.Join(root, "packages", name)); err == nil && len(entries) > 0 {
			return name, nil
		}
	}
	return "", fmt.Errorf("no webserver installed — run: phpv server install caddy")
}

func serverNameCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return domain.ServerNames(), cobra.ShellCompDirectiveNoFileComp
}
