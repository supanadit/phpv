package terminal

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
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
		Root:           root,
		ResolveVersion: h.resolveServerPHPVersion,
		EnsureFPM: func(ctx context.Context, phpVersion string, verbose bool, jobs int) (string, error) {
			fpm := h.fpmConcrete()
			tc, err := compiler.Resolve(h.compilerName())
			if err != nil {
				return "", err
			}
			// The cold FPM build takes minutes; show the same spinner used by
			// package installs unless raw build output was requested.
			needsBuild := fpm.NeedsRebuild(phpVersion, tc)
			var progress assembler.ProgressFunc
			stopProgress := func() {}
			if needsBuild && !verbose {
				progress, stopProgress = newProgressReporter()
			}
			bin, err := fpm.EnsureFPM(ctx, phpVersion, verbose, jobs, tc, "", progress)
			stopProgress()
			if needsBuild && !verbose {
				fmt.Println()
			}
			return bin, err
		},
		EnsureApacheModule: func(ctx context.Context, phpVersion, httpdPrefix string, verbose bool, jobs int) (string, error) {
			tc, err := compiler.Resolve(h.compilerName())
			if err != nil {
				return "", err
			}
			moduleSvc := assembler.NewApacheService(h.assemblerConcrete(), root)
			prefork := filepath.Join(httpdPrefix, "modules", "mod_mpm_prefork.so")
			needsHTTPDRebuild := !fileExists(prefork) || apacheThreaded(httpdPrefix)
			needsModule := moduleSvc.NeedsRebuild(phpVersion, httpdPrefix, tc)

			if needsHTTPDRebuild {
				fmt.Printf("  rebuilding apache %s with a prefork MPM for mod_php…\n", filepath.Base(httpdPrefix))
			}

			var progress assembler.ProgressFunc
			stopProgress := func() {}
			if !verbose && (needsHTTPDRebuild || needsModule) {
				progress, stopProgress = newProgressReporter()
			}

			if needsHTTPDRebuild {
				version := filepath.Base(httpdPrefix)
				if err := h.rebuildServer(ctx, "apache", version, verbose, jobs, tc, progress); err != nil {
					stopProgress()
					return "", fmt.Errorf("rebuild apache with prefork MPM: %w", err)
				}
				if !fileExists(prefork) || apacheThreaded(httpdPrefix) {
					stopProgress()
					return "", fmt.Errorf("apache at %s still has no prefork default MPM; reinstall with: phpv server install apache --force", httpdPrefix)
				}
			}
			module, err := moduleSvc.EnsureApacheModule(ctx, phpVersion, httpdPrefix, verbose, jobs, tc, progress)
			stopProgress()
			if !verbose && (needsHTTPDRebuild || needsModule) {
				fmt.Println()
			}
			return module, err
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

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// apacheThreaded reports whether the installed httpd advertises a threaded
// default MPM. mod_php must be built against a non-threaded (prefork) httpd,
// otherwise PHP configure enables ZTS and cannot load the CLI's shared
// extensions.
func apacheThreaded(prefix string) bool {
	httpd := filepath.Join(prefix, "bin", "httpd")
	out, err := exec.Command(httpd, "-V").CombinedOutput()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), "threaded:     yes") || strings.Contains(string(out), "threaded: yes")
}

// rebuildServer replaces an installed server package without losing the old
// build when the new one fails. Used to add shared prefork MPMs for mod_php.
func (h *PHPHandler) rebuildServer(ctx context.Context, name, version string, verbose bool, jobs int, tc compiler.Toolchain, progress assembler.ProgressFunc) error {
	prefix := h.siloSvc.PackagePrefix(name, version)
	bak := prefix + ".bak"
	_ = os.RemoveAll(bak)
	if _, err := os.Stat(prefix); err == nil {
		if err := os.Rename(prefix, bak); err != nil {
			return err
		}
	}
	_, err := h.assemblerConcrete().AssembleServer(ctx, name, version, verbose, jobs, tc, progress)
	if err != nil {
		_ = os.RemoveAll(prefix)
		_ = os.Rename(bak, prefix)
		return err
	}
	_ = os.RemoveAll(bak)
	return nil
}

// resolveServerPHPVersion resolves the PHP version a server run is pinned to.
// It is the same active-version chain every other command uses
// (PHPV_CURRENT → project file → default → latest installed), so a per-shell
// `phpv use` actually changes the version FPM serves.
func (h *PHPHandler) resolveServerPHPVersion(constraint string) (string, error) {
	return h.resolveVersion(constraint)
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
		Short: "Install and run local webservers (caddy, nginx, apache) serving PHP via FPM or mod_php",
	}

	install := &cobra.Command{
		Use:               "install <caddy|nginx|apache> [version]",
		Short:             "Install a webserver binary (does not change the PHP-version association)",
		Args:              cobra.RangeArgs(1, 2),
		ValidArgsFunction: serverNameCompletion,
		RunE: func(cmd *cobra.Command, args []string) error {
			server := strings.ToLower(args[0])
			if !domain.IsValidServerName(server) {
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
			if resolved, err := h.assemblerSvc.ResolveVersion(assembler.ServerPackageName(server), version); err == nil {
				exact = resolved
				banner = fmt.Sprintf("%s %s", server, exact)
			}
			fmt.Printf("Installing %s...\n\n", banner)

			var progress assembler.ProgressFunc
			stopProgress := func() {}
			if !verbose {
				progress, stopProgress = newProgressReporter()
			}
			if force && exact != "" {
				err = h.rebuildServer(h.ctx, server, exact, verbose, jobs, tc, progress)
			} else {
				_, err = h.assemblerConcrete().AssembleServer(h.ctx, server, version, verbose, jobs, tc, progress)
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
	install.Flags().Bool("verbose", false, "Show full build output")
	install.Flags().Int("jobs", 0, "Number of parallel build jobs (default: CPU count)")
	install.Flags().Bool("force", false, "Rebuild even when already installed (also upgrades httpd MPM support)")

	serverUse := &cobra.Command{
		Use:               "use <caddy|nginx|apache>",
		Short:             "Associate an installed webserver with a PHP version",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: serverNameCompletion,
		RunE: func(cmd *cobra.Command, args []string) error {
			server := strings.ToLower(args[0])
			if !domain.IsValidServerName(server) {
				return invalidServerName(server)
			}
			if !containsString(h.installedServers(), server) {
				return fmt.Errorf("%s is not installed — run: phpv server install %s", server, server)
			}
			phpFlag, _ := cmd.Flags().GetString("php")
			phpVersion, err := h.resolveVersion(phpFlag)
			if err != nil {
				return err
			}
			sapi, err := resolveCommandSAPI(cmd, server)
			if err != nil {
				return err
			}
			if err := h.recordServerSelection(phpVersion, server, sapi); err != nil {
				return fmt.Errorf("associate %s with PHP %s: %w", server, phpVersion, err)
			}
			fmt.Printf("✓ %s (%s) is now associated with PHP %s\n", server, sapiLabel(sapi), phpVersion)
			return nil
		},
	}
	serverUse.Flags().String("php", "", "PHP version to associate with (default: active version)")
	serverUse.Flags().String("sapi", "", "PHP SAPI: fpm (default) or modphp (apache only)")
	_ = serverUse.RegisterFlagCompletionFunc("php", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return h.versionNameCompletion(toComplete)
	})
	_ = serverUse.RegisterFlagCompletionFunc("sapi", sapiCompletion)

	serverDefault := &cobra.Command{
		Use:               "default <caddy|nginx|apache>",
		Short:             "Set the global default webserver for PHP versions without an association",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: serverNameCompletion,
		RunE: func(cmd *cobra.Command, args []string) error {
			server := strings.ToLower(args[0])
			if !domain.IsValidServerName(server) {
				return invalidServerName(server)
			}
			if !containsString(h.installedServers(), server) {
				return fmt.Errorf("%s is not installed — run: phpv server install %s", server, server)
			}
			sapi, err := resolveCommandSAPI(cmd, server)
			if err != nil {
				return err
			}
			if err := h.configSvc.Set("server", server); err != nil {
				return fmt.Errorf("set global default webserver: %w", err)
			}
			if err := h.configSvc.Set("server_sapi", sapi); err != nil {
				return fmt.Errorf("set global default SAPI: %w", err)
			}
			fmt.Printf("✓ global default webserver: %s (%s)\n", server, sapiLabel(sapi))
			return nil
		},
	}
	serverDefault.Flags().String("sapi", "", "PHP SAPI: fpm (default) or modphp (apache only)")
	_ = serverDefault.RegisterFlagCompletionFunc("sapi", sapiCompletion)

	run := &cobra.Command{
		Use:   "run <dir>",
		Short: "Serve a directory through the webserver and PHP for the active PHP version",
		Long: `Serve a directory (replaces ` + "`php -S localhost:... -t public`" + `).

The PHP version follows the active version (` + "`phpv use`" + `, ` + "`.php-version`" + `,
default), exactly like ` + "`php`" + ` itself. The webserver is picked per PHP version:
` + "`--server`" + `, the server associated with that version (` + "`phpv server use`" + `),
` + "`PHPV_DEFAULT_SERVER`" + `, the global default (` + "`phpv server default`" + `), then the
first installed server. No association is required to run.

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

			inst, err := m.Start(h.ctx, webserver.RunOptions{
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
			fmt.Printf("  url:     %s\n", inst.URL())
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
	run.Flags().String("server", "", "Webserver to use (caddy, nginx, apache); default: per-version association, then PHPV_DEFAULT_SERVER, then global default")
	run.Flags().String("version", "", "Webserver version to use; default from config or newest installed")
	run.Flags().String("php", "", "PHP version to serve with (e.g. 5.6, 8.3); default: active/default")
	run.Flags().String("sapi", "", "PHP SAPI: fpm (default) or modphp (apache only)")
	run.Flags().String("host", "", "Hostname for auto vhost (adds /etc/hosts entry)")
	run.Flags().Int("port", 0, "TCP port (default: 8080, auto-increments when busy)")
	run.Flags().BoolP("detach", "d", false, "Run in the background")
	run.Flags().Bool("verbose", false, "Show full build output (when a PHP runtime must be built)")
	run.Flags().Int("jobs", 0, "Number of parallel build jobs (default: CPU count)")
	_ = run.RegisterFlagCompletionFunc("server", serverNameCompletion)
	_ = run.RegisterFlagCompletionFunc("sapi", sapiCompletion)
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
		Short: "List installed webservers, defaults, associations and running instances",
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

			activePHP, _ := h.resolveVersion("")
			fmt.Println("\nGlobal default:")
			switch def := strings.TrimSpace(os.Getenv(defaultServerEnv)); {
			case def != "":
				envSAPI := webserver.NormalizeSAPI(os.Getenv(defaultSAPIEnv))
				if strings.TrimSpace(os.Getenv(defaultSAPIEnv)) == "" {
					cfgSAPI, _ := h.configSvc.Get("server_sapi")
					envSAPI = webserver.NormalizeSAPI(cfgSAPI)
				}
				if envSAPI == "" || (envSAPI == "modphp" && def != domain.ServerApache) {
					envSAPI = "fpm"
				}
				fmt.Printf("  %s (%s, from %s)\n", def, sapiLabel(envSAPI), defaultServerEnv)
			default:
				if cfg, _ := h.configSvc.Get("server"); cfg != "" {
					sapi, _ := h.configSvc.Get("server_sapi")
					if env := strings.TrimSpace(os.Getenv(defaultSAPIEnv)); env != "" {
						sapi = env
					}
					sapi = webserver.NormalizeSAPI(sapi)
					if sapi == "" || (sapi == "modphp" && cfg != domain.ServerApache) {
						sapi = "fpm"
					}
					fmt.Printf("  %s (%s, from phpv server default)\n", cfg, sapiLabel(sapi))
				} else {
					fmt.Println("  (none — falls back to the first installed server)")
				}
			}

			fmt.Println("\nPer-PHP associations:")
			prefs := loadServerPreferences(root)
			if len(prefs.Servers) == 0 {
				fmt.Println("  (none — run: phpv server use <name>)")
			} else {
				versions := make([]string, 0, len(prefs.Servers))
				for version := range prefs.Servers {
					versions = append(versions, version)
				}
				sort.Strings(versions)
				for _, version := range versions {
					marker := ""
					if version == activePHP {
						marker = "  (active)"
					}
					sapi := "fpm"
					if recorded, ok := prefs.getSAPI(version); ok {
						sapi = webserver.NormalizeSAPI(recorded)
					}
					fmt.Printf("  PHP %-10s -> %s (%s)%s\n", version, prefs.Servers[version], sapiLabel(sapi), marker)
				}
			}
			if activePHP != "" {
				if server, source, err := h.resolveServerForPHP("", activePHP); err == nil {
					sapi, sapiErr := h.resolveSAPIForPHP("", server, activePHP)
					if sapiErr != nil {
						sapi = "fpm"
					}
					fmt.Printf("\nActive PHP %s -> %s (%s) (%s)\n", activePHP, server, sapiLabel(sapi), serverSourceLabel(source))
				}
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
				fmt.Printf("  %-28s %-8s %s (%s)  php-%s  %s  %s\n", inst.ID, state, inst.Server, sapiLabel(inst.SAPI), inst.PHPVersion, inst.URL(), inst.Dir)
			}
			return nil
		},
	}

	cmd.AddCommand(install, serverUse, serverDefault, run, stop, list)
	return cmd
}

func serverNameCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return domain.ServerNames(), cobra.ShellCompDirectiveNoFileComp
}

func sapiCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return []string{"fpm", "modphp"}, cobra.ShellCompDirectiveNoFileComp
}

// resolveCommandSAPI validates the --sapi flag for use/default commands;
// an omitted flag means the default fpm SAPI.
func resolveCommandSAPI(cmd *cobra.Command, server string) (string, error) {
	flag, _ := cmd.Flags().GetString("sapi")
	sapi := webserver.NormalizeSAPI(flag)
	if strings.TrimSpace(flag) == "" {
		sapi = "fpm"
	}
	if sapi != "fpm" && sapi != "modphp" {
		return "", fmt.Errorf("unknown SAPI %q (available: fpm, modphp)", flag)
	}
	if sapi == "modphp" && server != domain.ServerApache {
		return "", fmt.Errorf("SAPI modphp is only supported with apache (got %s)", server)
	}
	return sapi, nil
}

// sapiLabel renders the SAPI for user-facing output.
func sapiLabel(sapi string) string {
	if sapi == "modphp" {
		return "mod_php"
	}
	return "FPM"
}
