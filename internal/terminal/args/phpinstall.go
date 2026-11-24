package args

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/pkg/source"
)

func (h *PHPHandler) install(cmd *cobra.Command, args []string) error {
	version := args[0]

	fromBundle, _ := cmd.Flags().GetString("from")
	if fromBundle != "" {
		fmt.Printf("Installing PHP %s from bundle %s...\n", version, fromBundle)
		if err := h.bundleSvc.Import(fromBundle, version); err != nil {
			return fmt.Errorf("install from bundle failed: %w", err)
		}
		fmt.Printf("✓ PHP %s installed from bundle\n", version)
		return nil
	}

	state, _ := h.siloSvc.GetState("php", version)
	if state == domain.StateFailed || state == domain.StateInterrupted || state == domain.StateInProgress {
		force, _ := cmd.Flags().GetBool("force")
		clean, _ := cmd.Flags().GetBool("clean")
		if !force && !clean {
			stateMsg := string(state)
			switch state {
			case domain.StateInterrupted:
				stateMsg = "interrupted (you pressed Ctrl+C)"
			case domain.StateInProgress:
				stateMsg = "in_progress (likely crashed)"
			}
			fmt.Printf("Previous install of PHP %s was '%s'.\n", version, stateMsg)
			fmt.Printf("Run `phpv install %s --force` to retry (deps preserved).\n", version)
			fmt.Printf("Or use `--clean` to delete the PHP prefix/source first.\n")
			return nil
		}
	}

	static, _ := cmd.Flags().GetBool("static")
	extStr, _ := cmd.Flags().GetString("ext")
	minimal, _ := cmd.Flags().GetBool("minimal")
	autoDeps, _ := cmd.Flags().GetBool("auto-deps")
	noSystem, _ := cmd.Flags().GetBool("no-system")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")
	clean, _ := cmd.Flags().GetBool("clean")
	fresh, _ := cmd.Flags().GetBool("fresh")
	verbose, _ := cmd.Flags().GetBool("verbose")
	jobsFlag, _ := cmd.Flags().GetInt("jobs")
	compilerFlag, _ := cmd.Flags().GetString("compiler")

	jobs := resolveJobs(jobsFlag, h.configSvc)

	if extStr != "" && minimal {
		return fmt.Errorf("cannot use --ext with --minimal")
	}

	tc, err := h.selectToolchain(compilerFlag)
	if err != nil {
		return err
	}

	resolvedVersion, resolveErr := h.assemblerSvc.ResolveVersion("php", version)
	if resolveErr != nil {
		return fmt.Errorf("resolve version: %w", resolveErr)
	}

	var extensions []string
	if extStr != "" {
		for _, e := range strings.Split(extStr, ",") {
			e = strings.TrimSpace(e)
			if e != "" {
				extensions = append(extensions, e)
			}
		}
	} else if !minimal {
		defaults, skipped := h.assemblerSvc.DefaultExtensions(resolvedVersion)
		if len(defaults) == 0 && len(skipped) > 0 {
			fmt.Printf("Note: No extensions from the default set are compatible with PHP %s.\n", resolvedVersion)
			fmt.Println("Use --ext <list> to build specific extensions, or --minimal for a bare PHP.")
		} else {
			fmt.Printf("Default extension set (%d extensions", len(defaults))
			if len(skipped) > 0 {
				fmt.Printf(", %d skipped", len(skipped))
			}
			fmt.Println("):")
			fmt.Printf("  %s\n", strings.Join(defaults, ", "))
			if len(skipped) > 0 {
				fmt.Println()
				for _, s := range skipped {
					fmt.Printf("  skipped: %s\n", s)
				}
			}
		}
		extensions = defaults
	}
	extensions, added := h.assemblerSvc.ExpandImplied(extensions)
	if len(added) > 0 {
		fmt.Printf("Including implied extensions: %s\n", strings.Join(added, ", "))
	}

	var systemPkgs map[string]domain.SystemPackage
	if !noSystem {
		systemPkgs, err = h.checkSystemDeps(extensions, autoDeps, dryRun)
		if err != nil {
			return err
		}
		if err := h.checkBuildTools(tc, autoDeps, dryRun); err != nil {
			return err
		}
	}

	if dryRun {
		fmt.Println("Dry run complete. Run without --dry-run to install.")
		return nil
	}

	if clean {
		prefix := h.siloSvc.PackagePrefix("php", version)
		fmt.Printf("Cleaning existing install at %s...\n", prefix)
		os.RemoveAll(prefix)
		sourceDir := h.siloSvc.SourcePath("php", version)
		os.RemoveAll(sourceDir)
		force = true
	} else if fresh {
		prefix := h.siloSvc.PackagePrefix("php", version)
		fmt.Printf("Refreshing install at %s (keeping cached source)...\n", prefix)
		os.RemoveAll(prefix)
		force = true
	}

	bannerVersion := version
	if exact, err := h.assemblerSvc.ResolveVersion("php", version); err == nil {
		bannerVersion = exact
	}
	fmt.Printf("Installing PHP %s...\n\n", bannerVersion)

	if verbose {
		result, err := h.assemblerSvc.Assemble(h.ctx, domain.AssembleSpec{Name: "php", Version: version, Static: static, Extensions: extensions, Verbose: true, SystemPkgs: systemPkgs, Jobs: jobs, Force: force, TC: domain.Toolchain{Name: tc.Name, CC: tc.CC, CXX: tc.CXX, Hint: tc.Hint}})
		if err != nil {
			return fmt.Errorf("install failed: %w", err)
		}
		fmt.Println()
		if result.AlreadyInstalled {
			fmt.Printf("✓ PHP %s is already installed\n", result.Version)
			return nil
		}
		sharedOnly := h.assemblerSvc.SharedOnlyExtensions(resolvedVersion, extensions)
		if len(sharedOnly) > 0 {
			sourceDir := h.siloSvc.SourcePath("php", resolvedVersion)
			srcPath := source.Find(sourceDir, "php")
			if srcPath == "" {
				fmt.Printf("Warning: PHP source not found at %s, skipping shared extension builds\n", sourceDir)
			} else {
				for _, ext := range sharedOnly {
					fmt.Printf("Building %s as shared extension...\n", ext)
					if err := h.assemblerSvc.InstallExtension(h.ctx, domain.ExtensionInstallRequest{PHPVersion: resolvedVersion, ExtName: ext, PHPSourceDir: srcPath, PHPPrefix: result.Prefix, Jobs: jobs}); err != nil {
						fmt.Printf("Warning: %s shared build failed: %v\n", ext, err)
						fmt.Printf("  Run `phpv extension %s add %s` to retry.\n", resolvedVersion, ext)
					} else {
						fmt.Printf("✓ %s installed (shared)\n", ext)
					}
				}
			}
		}
		if err := h.shimSvc.RegenerateAll(); err != nil {
			return fmt.Errorf("regenerate shims: %w", err)
		}
		return nil
	}

	progress, stopProgress := newProgressReporter()
	result, err := h.assemblerSvc.Assemble(h.ctx, domain.AssembleSpec{Name: "php", Version: version, Static: static, Extensions: extensions, Verbose: false, Progress: progress, SystemPkgs: systemPkgs, Jobs: jobs, Force: force, TC: domain.Toolchain{Name: tc.Name, CC: tc.CC, CXX: tc.CXX, Hint: tc.Hint}})
	stopProgress()

	if err != nil {
		fmt.Println()
		return fmt.Errorf("install failed: %w", err)
	}
	fmt.Println()
	if result.AlreadyInstalled {
		fmt.Printf("✓ PHP %s is already installed\n", result.Version)
		return nil
	}

	sharedOnly := h.assemblerSvc.SharedOnlyExtensions(resolvedVersion, extensions)
	if len(sharedOnly) > 0 {
		sourceDir := h.siloSvc.SourcePath("php", resolvedVersion)
		srcPath := source.Find(sourceDir, "php")
		if srcPath == "" {
			fmt.Printf("Warning: PHP source not found at %s, skipping shared extension builds\n", sourceDir)
		} else {
			for _, ext := range sharedOnly {
				fmt.Printf("Building %s as shared extension...\n", ext)
				if err := h.assemblerSvc.InstallExtension(h.ctx, domain.ExtensionInstallRequest{PHPVersion: resolvedVersion, ExtName: ext, PHPSourceDir: srcPath, PHPPrefix: result.Prefix, Jobs: jobs}); err != nil {
					fmt.Printf("Warning: %s shared build failed: %v\n", ext, err)
					fmt.Printf("  Run `phpv extension %s add %s` to retry.\n", resolvedVersion, ext)
				} else {
					fmt.Printf("✓ %s installed (shared)\n", ext)
				}
			}
		}
	}

	if err := h.shimSvc.RegenerateAll(); err != nil {
		return fmt.Errorf("regenerate shims: %w", err)
	}
	return nil
}

// versionsCmd lists installed PHP versions.
func (h *PHPHandler) versionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "versions",
		Short: "List installed PHP versions",
		Args:  cobra.NoArgs,
		RunE:  h.versions,
	}
	cmd.Flags().Bool("json", false, "Output in JSON format")
	return cmd
}

type versionEntry struct {
	Version string `json:"version"`
	Default bool   `json:"default"`
}
