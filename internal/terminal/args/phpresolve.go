package args

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	vsn "github.com/supanadit/phpv/pkg/version"

	"github.com/spf13/cobra"
)

func (h *PHPHandler) use(cmd *cobra.Command, args []string) error {
	version := args[0]
	printOnly, _ := cmd.Flags().GetBool("print")
	global, _ := cmd.Flags().GetBool("global")
	local, _ := cmd.Flags().GetBool("local")

	// --print emits an export statement for the calling shell; the shell
	// function from `phpv init` evals it. It must never touch the global
	// default or regenerate shims.
	if printOnly {
		if version == "system" {
			fmt.Println("unset PHPV_CURRENT")
			return nil
		}
		exactVersion, err := h.resolveVersion(version)
		if err != nil {
			return err
		}
		fmt.Printf("export PHPV_CURRENT=%q\n", exactVersion)
		return nil
	}

	// A bare `phpv use` cannot change the parent shell's environment (the
	// phpv() function from `phpv init` does that via --print), so without
	// an explicit --global/--local it never silently writes the global
	// default or a project file. Tell the user how to switch.
	if !global && !local {
		return fmt.Errorf("per-shell version switching needs shell integration — run `phpv init %s` and restart your shell, or use `phpv use %s --global` (or `--local`)", detectShell(), version)
	}

	if version == "system" {
		systemPHP, err := exec.LookPath("php")
		if err != nil {
			return fmt.Errorf("system PHP not found in PATH")
		}
		silo := h.siloSvc.GetSilo()
		phpvBin := filepath.Join(silo.Root, "bin", "php")
		if systemPHP == phpvBin {
			return fmt.Errorf("system PHP is managed by phpv; use a specific version instead")
		}
		if err := h.shimSvc.SetSystemMode(true); err != nil {
			return fmt.Errorf("set system mode: %w", err)
		}
		if err := h.shimSvc.RegenerateAll(); err != nil {
			return fmt.Errorf("regenerate shims: %w", err)
		}
		fmt.Printf("→ Using system PHP at %s\n", systemPHP)
		fmt.Println("Run `phpv init` in your shell to enable version switching.")
		return nil
	}

	exactVersion, err := h.resolveVersion(version)
	if err != nil {
		return err
	}

	if local {
		if err := writeLocalVersionFile(exactVersion); err != nil {
			return fmt.Errorf("write local version: %w", err)
		}
		fmt.Printf("✓ Switched to PHP %s (local, .php-version)\n", exactVersion)
		return nil
	}

	if err := h.shimSvc.SetSystemMode(false); err != nil {
		return fmt.Errorf("clear system mode: %w", err)
	}
	if err := h.siloSvc.SetDefault(exactVersion); err != nil {
		return fmt.Errorf("set default: %w", err)
	}
	if err := h.shimSvc.RegenerateAll(); err != nil {
		return fmt.Errorf("regenerate shims: %w", err)
	}

	silo := h.siloSvc.GetSilo()
	phpBin := filepath.Join(silo.Root, "packages", "php", exactVersion, "bin", "php")
	fmt.Printf("✓ Switched to PHP %s (%s)\n", exactVersion, phpBin)
	fmt.Println("Run `phpv init` in your shell to enable version switching.")
	return nil
}

// resolveActivePHP resolves the active PHP binary path.
// Priority: PHPV_CURRENT env > .phpvrc > default > system PHP.
func (h *PHPHandler) resolveActivePHP() (string, error) {
	// 1. Check PHPV_CURRENT env var.
	if envVer := os.Getenv("PHPV_CURRENT"); envVer != "" {
		silo := h.siloSvc.GetSilo()
		phpBin := filepath.Join(silo.Root, "packages", "php", envVer, "bin", "php")
		if _, err := os.Stat(phpBin); err == nil {
			return phpBin, nil
		}
	}

	// 2. Check .php-version or .phpvrc in current or parent directories.
	if ver := findProjectVersionFile(); ver != "" {
		silo := h.siloSvc.GetSilo()
		phpBin := filepath.Join(silo.Root, "packages", "php", ver, "bin", "php")
		if _, err := os.Stat(phpBin); err == nil {
			return phpBin, nil
		}
	}

	// 3. Check default.
	defaultVer, err := h.siloSvc.GetDefault()
	if err == nil && defaultVer != "" {
		silo := h.siloSvc.GetSilo()
		phpBin := filepath.Join(silo.Root, "packages", "php", defaultVer, "bin", "php")
		if _, err := os.Stat(phpBin); err == nil {
			return phpBin, nil
		}
	}

	// 4. Fall back to system PHP.
	systemPHP, err := exec.LookPath("php")
	if err == nil {
		return systemPHP, nil
	}

	return "", fmt.Errorf("no PHP version found (install one with `phpv install <version>`)")
}

// resolveActiveVersion returns the active PHP version string.
// Priority: PHPV_CURRENT env > .php-version > .phpvrc > default.
func (h *PHPHandler) resolveActiveVersion() (string, error) {
	if envVer := os.Getenv("PHPV_CURRENT"); envVer != "" {
		return envVer, nil
	}
	if ver := findProjectVersionFile(); ver != "" {
		return ver, nil
	}
	defaultVer, err := h.siloSvc.GetDefault()
	if err == nil && defaultVer != "" {
		return defaultVer, nil
	}
	// No explicit active version: fall back to the latest installed PHP so
	// commands work right after a fresh install without requiring `phpv use`
	// or `phpv default` first.
	if latest, err := h.latestInstalledVersion(); err == nil {
		return latest, nil
	}
	return "", fmt.Errorf("no active PHP version and no PHP installed (run `phpv install <version>` first)")
}

// latestInstalledVersion returns the highest version installed under the
// packages/php directory, or an error if none are installed.
func (h *PHPHandler) latestInstalledVersion() (string, error) {
	installed, err := h.installedPHPVersions()
	if err != nil {
		return "", err
	}
	if len(installed) == 0 {
		return "", fmt.Errorf("no PHP versions installed")
	}
	best := installed[0]
	for _, v := range installed[1:] {
		if vsn.CompareVersions(v, best) > 0 {
			best = v
		}
	}
	return best, nil
}

// installedPHPVersions returns the names of installed PHP versions (directories
// under packages/php that contain a php binary).
func (h *PHPHandler) installedPHPVersions() ([]string, error) {
	silo := h.siloSvc.GetSilo()
	phpDir := filepath.Join(silo.Root, "packages", "php")

	entries, err := os.ReadDir(phpDir)
	if err != nil {
		return nil, fmt.Errorf("no PHP versions installed")
	}

	var installed []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		phpBin := filepath.Join(phpDir, e.Name(), "bin", "php")
		if _, err := os.Stat(phpBin); err == nil {
			installed = append(installed, e.Name())
		}
	}
	return installed, nil
}

// resolveVersion resolves a version constraint to an exact installed version.
// Empty string falls back to the active version.
func (h *PHPHandler) resolveVersion(constraint string) (string, error) {
	if constraint == "" {
		return h.resolveActiveVersion()
	}
	return h.resolveInstalledVersion(constraint)
}

// resolveInstalledVersion resolves a version constraint to an exact installed version.
func (h *PHPHandler) resolveInstalledVersion(constraint string) (string, error) {
	installed, err := h.installedPHPVersions()
	if err != nil {
		return "", err
	}

	// Exact match first.
	for _, v := range installed {
		if v == constraint {
			return v, nil
		}
	}

	// Major.minor or major-only match (e.g., "8.4" → latest 8.4.x, "8" → latest 8.x.x).
	if strings.Count(constraint, ".") == 1 || strings.Count(constraint, ".") == 0 {
		prefix := constraint + "."
		if best := vsn.LatestMatching(installed, prefix); best != "" {
			return best, nil
		}
	}

	return "", fmt.Errorf("PHP %s is not installed. Run `phpv install %s` first", constraint, constraint)
}

// findProjectVersionFile walks up from the current directory looking for
// a .php-version or .phpvrc file. .php-version takes priority.
func findProjectVersionFile() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		if data, err := os.ReadFile(filepath.Join(dir, ".php-version")); err == nil {
			return strings.TrimSpace(string(data))
		}
		if data, err := os.ReadFile(filepath.Join(dir, ".phpvrc")); err == nil {
			return strings.TrimSpace(string(data))
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}
