package args

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/supanadit/phpv/pkg/progress"

	webidentity "github.com/supanadit/phpv/pkg/webidentity"

	vsn "github.com/supanadit/phpv/pkg/version"

	"github.com/spf13/cobra"
	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/pkg/compiler"
)

// ServerManagerDeps adapts the PHPHandler's services to webserver.Manager.
// Built lazily so command construction stays cheap.
func (h *PHPHandler) serverManager() (ServerManager, error) {
	root := resolvePHPVRootForServer()
	m := h.serverMgrFn(ServerManagerDeps{
		Root:           root,
		ResolveVersion: h.resolveServerPHPVersion,
		EnsureFPM: func(ctx context.Context, phpVersion string, verbose bool, jobs int) (string, error) {
			fpm := h.fpmSvc
			tc, err := compiler.Resolve(h.compilerName())
			if err != nil {
				return "", err
			}
			// The cold FPM build takes minutes; show the same spinner used by
			// package installs unless raw build output was requested.
			needsBuild := fpm.NeedsRebuild(phpVersion, tc)
			var progress progress.Func
			stopProgress := func() {}
			if needsBuild && !verbose {
				progress, stopProgress = newProgressReporter()
			}
			bin, err := fpm.EnsureFPM(ctx, domain.FPMSpec{PHPVersion: phpVersion, Verbose: verbose, Jobs: jobs, TC: domainToolchain(tc), StagedFrom: "", Progress: progress})
			stopProgress()
			if needsBuild && !verbose {
				fmt.Println()
			}
			return bin, err
		},
		EnsureApacheModule: func(ctx context.Context, phpVersion, httpdPrefix string, opts domain.ServerRunOptions) (string, error) {
			verbose, jobs := opts.Verbose, opts.Jobs
			tc, err := compiler.Resolve(h.compilerName())
			if err != nil {
				return "", err
			}
			moduleSvc := h.apacheSvc
			prefork := filepath.Join(httpdPrefix, "modules", "mod_mpm_prefork.so")
			needsHTTPDRebuild := !fileExists(prefork) || apacheThreaded(httpdPrefix)
			needsModule := moduleSvc.NeedsRebuild(phpVersion, httpdPrefix, tc)

			if needsHTTPDRebuild {
				fmt.Printf("  rebuilding apache %s with a prefork MPM for mod_php…\n", filepath.Base(httpdPrefix))
			}

			var progress progress.Func
			stopProgress := func() {}
			if !verbose && (needsHTTPDRebuild || needsModule) {
				progress, stopProgress = newProgressReporter()
			}

			if needsHTTPDRebuild {
				version := filepath.Base(httpdPrefix)
				if err := h.rebuildServer(ctx, rebuildSpec{"apache", version, verbose, jobs, tc, progress}); err != nil {
					stopProgress()
					return "", fmt.Errorf("rebuild apache with prefork MPM: %w", err)
				}
				if !fileExists(prefork) || apacheThreaded(httpdPrefix) {
					stopProgress()
					return "", fmt.Errorf("apache at %s still has no prefork default MPM; reinstall with: phpv server install apache --force", httpdPrefix)
				}
			}
			module, err := moduleSvc.EnsureApacheModule(ctx, domain.ApacheModuleSpec{PHPVersion: phpVersion, HTTPDPrefix: httpdPrefix, Verbose: verbose, Jobs: jobs, TC: domainToolchain(tc), Progress: progress})
			stopProgress()
			if !verbose && (needsHTTPDRebuild || needsModule) {
				fmt.Println()
			}
			return module, err
		},
		CLIPrefix: func(phpVersion string) string {
			return h.siloSvc.PHPOutputPath(phpVersion)
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
type rebuildSpec struct {
	name, version string
	verbose       bool
	jobs          int
	tc            compiler.Toolchain
	progress      progress.Func
}

func (h *PHPHandler) rebuildServer(ctx context.Context, spec rebuildSpec) error {
	name, version, verbose, jobs, tc, progress := spec.name, spec.version, spec.verbose, spec.jobs, spec.tc, spec.progress
	prefix := h.siloSvc.PackagePrefix(name, version)
	bak := prefix + ".bak"
	_ = os.RemoveAll(bak)
	if _, err := os.Stat(prefix); err == nil {
		if err := os.Rename(prefix, bak); err != nil {
			return err
		}
	}
	_, err := h.serverAsbl.AssembleServer(ctx, domain.ServerAssembleSpec{Name: name, Version: version, Verbose: verbose, Jobs: jobs, TC: domainToolchain(tc), Progress: progress})
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
			if vsn.CompareVersions(v, best) > 0 {
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
		if vsn.MatchVersionRange(constraint, v) {
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
	return resolveRoot()
}

func (h *PHPHandler) serverCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "server",
		Short: "Install and run local webservers (caddy, nginx, apache) serving PHP via FPM or mod_php",
	}

	install := h.serverinstall()
	serverUse := h.serveruse()
	serverDefault := h.serverdefault()
	run := h.serverrun()
	stop := h.serverstop()
	list := h.serverlist()
	cmd.AddCommand(install, serverUse, serverDefault, run, stop, list)
	return cmd
}

func serverNameCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return webidentity.ServerNames(), cobra.ShellCompDirectiveNoFileComp
}

func sapiCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return []string{"fpm", "modphp"}, cobra.ShellCompDirectiveNoFileComp
}

// resolveCommandSAPI validates the --sapi flag for use/default commands;
// an omitted flag means the default fpm SAPI.
func resolveCommandSAPI(cmd *cobra.Command, server string) (string, error) {
	flag, _ := cmd.Flags().GetString("sapi")
	sapi := webidentity.NormalizeSAPI(flag)
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
