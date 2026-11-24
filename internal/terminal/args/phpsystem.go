package args

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/supanadit/phpv/domain"

	"github.com/supanadit/phpv/pkg/compiler"
)

// writeLocalVersionFile writes the given version to .php-version in the
// current working directory.
func writeLocalVersionFile(version string) error {
	return os.WriteFile(".php-version", []byte(version+"\n"), 0644)
}

// resolveJobs resolves the effective parallelism value.
// Priority: CLI flag > config concurrency > runtime.NumCPU().
func resolveJobs(flagVal int, cfgSvc ConfigService) int {
	if flagVal > 0 {
		return flagVal
	}
	if cfgSvc != nil {
		val, err := cfgSvc.Get("concurrency")
		if err == nil && val != "" {
			if n, err := strconv.Atoi(val); err == nil && n > 0 {
				return n
			}
		}
	}
	return runtime.NumCPU()
}

// selectToolchain resolves the compiler a build should use: the explicit
// choice (--compiler) wins, then the `compiler` config key, then the platform
// default. It exports CC/CXX into this process so every child build process
// (configure, make, phpize, pecl) uses the same toolchain the build plan was
// generated for.
func (h *PHPHandler) selectToolchain(flagVal string) (compiler.Toolchain, error) {
	name := flagVal
	if name == "" && h.configSvc != nil {
		if v, err := h.configSvc.Get("compiler"); err == nil {
			name = v
		}
	}
	tc, err := compiler.Resolve(name)
	if err != nil {
		return compiler.Toolchain{}, err
	}
	tc.Setenv()
	return tc, nil
}

// checkSystemDeps checks for system packages needed by PHP deps and extensions.
// If autoDeps is true, installs missing packages without prompting.
// If dryRun is true, only prints what would be done.
// Returns a map of available system packages (name -> Package) for use in hybrid builds.
func (h *PHPHandler) checkSystemDeps(extensions []string, autoDeps, dryRun bool) (map[string]domain.SystemPackage, error) {
	// The dependency set is derived from the extension definitions served
	// through the graph repository, so the data lives in one place and cannot
	// drift from what the build plan actually consumes.
	phpDeps := h.systemSvc.DependencySet(h.extensionDefs(extensions))

	result, err := h.systemSvc.Check(phpDeps)
	if err != nil {
		return nil, fmt.Errorf("system check: %w", err)
	}

	systemAvail := usablePackages(result.Available)

	if result.Distro.PM == "unknown" {
		return systemAvail, nil
	}

	if len(result.Missing) == 0 {
		return systemAvail, nil
	}

	installed, err := h.promptAndInstall(result.Missing, autoDeps, dryRun, true)
	if err != nil {
		return nil, err
	}
	if !installed {
		return systemAvail, nil
	}

	// Re-check instead of assuming the install made every package usable: the
	// build only links a system package when it reports a version that satisfies
	// the dependency constraint, so a package left without one must not be
	// returned as available.
	refreshed, err := h.systemSvc.Check(phpDeps)
	if err != nil {
		return nil, fmt.Errorf("system re-check after install: %w", err)
	}
	return usablePackages(refreshed.Available), nil
}

// extensionDefs resolves the definitions for the named extensions, skipping any
// the graph repository does not know.
func (h *PHPHandler) extensionDefs(extensions []string) []domain.ExtensionDef {
	defs := make([]domain.ExtensionDef, 0, len(extensions))
	for _, ext := range extensions {
		if def, ok := h.assemblerSvc.GetExtensionDef(ext); ok {
			defs = append(defs, def)
		}
	}
	return defs
}

// usablePackages indexes system packages by dependency name, keeping only the
// ones the build can actually link against. A package without a resolved
// version cannot satisfy a dependency constraint, so it is treated as
// unavailable rather than assumed usable.
func usablePackages(pkgs []domain.SystemPackage) map[string]domain.SystemPackage {
	m := make(map[string]domain.SystemPackage, len(pkgs))
	for _, p := range pkgs {
		if p.Installed && p.Version != "" {
			m[p.Name] = p
		}
	}
	return m
}

// promptAndInstall asks the user whether to install the given missing system
// packages, then installs them when confirmed. autoDeps skips the prompt and
// installs directly; dryRun only prints what would be done. It returns true
// when the packages were installed and false when they were skipped or
// declined.
//
// buildDeps marks a dependency prompt, which explains that a system package is
// only used when its version satisfies the build requirement. Build-tool
// prompts pass false: no version gate applies to them.
func (h *PHPHandler) promptAndInstall(missing []domain.SystemPackage, autoDeps, dryRun, buildDeps bool) (bool, error) {
	fmt.Println("\nMissing system packages:")
	for _, p := range missing {
		label := ""
		if buildDeps && !p.Required {
			label = " [optional]"
		}
		fmt.Printf("  ✗ %s (%s)%s\n", p.Name, p.SystemName, label)
	}
	if buildDeps {
		fmt.Println("\n  Used only when the installed version satisfies the build requirement;")
		fmt.Println("  incompatible versions are built from source instead.")
	}

	installCmd := h.systemSvc.InstallCommand(missing)
	fmt.Printf("\nInstall via %s? ", installCmd)
	os.Stdout.Sync()

	if autoDeps {
		fmt.Println("[Y] (--auto-deps)")
	} else if dryRun {
		fmt.Println("[skipped, --dry-run]")
		return false, nil
	} else {
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			fmt.Println("[non-interactive, skipping]")
			return false, nil
		}
		reader := bufio.NewReader(os.Stdin)
		answer, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("[read error: %v, skipping]\n", err)
			return false, nil
		}
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "y" && answer != "yes" && answer != "" {
			fmt.Println("Building from source instead.")
			return false, nil
		}
	}

	fmt.Println("Installing system packages...")
	if err := h.systemSvc.Install(missing); err != nil {
		return false, fmt.Errorf("install system packages: %w", err)
	}
	fmt.Println("✓ System packages installed")
	return true, nil
}

// checkBuildTools checks for the compiler and build toolchain needed to build
// PHP from source and, when any are missing, prompts to install them. Returns
// an error when a critical tool is missing and the user declines to install it,
// since the build cannot proceed without it.
func (h *PHPHandler) checkBuildTools(tc compiler.Toolchain, autoDeps, dryRun bool) error {
	toolNames := []string{"make", "cmake", "autoconf", "automake", "m4", "perl", "bison", "re2c", "flex", "pkg-config", "xz"}
	critical := map[string]bool{"make": true}
	if tc.Name == "gcc" {
		// gcc/g++ ship in a distro toolchain package, so let the package manager
		// prompt keep handling them the way it always has.
		toolNames = append([]string{"gcc", "g++"}, toolNames...)
		critical["gcc"] = true
		critical["g++"] = true
	} else if err := tc.Check(); err != nil {
		// clang/zig have no package-name mapping here; they must already be
		// installed and on PATH.
		return err
	}
	result, err := h.systemSvc.CheckBuildTools(toolNames)
	if err != nil {
		return fmt.Errorf("build tools check: %w", err)
	}
	if result.Distro.PM == "unknown" || len(result.Missing) == 0 {
		return nil
	}

	var criticalMissing []string
	for _, p := range result.Missing {
		if critical[p.Name] {
			criticalMissing = append(criticalMissing, p.Name)
		}
	}

	installed, err := h.promptAndInstall(result.Missing, autoDeps, dryRun, false)
	if err != nil {
		return err
	}
	if installed {
		return nil
	}
	if len(criticalMissing) > 0 {
		return fmt.Errorf("cannot build PHP without required build tools (%s). Install them manually or run `phpv doctor` for guidance", strings.Join(criticalMissing, ", "))
	}
	return nil
}
