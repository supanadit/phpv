package doctor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/supanadit/phpv/domain"
)

type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityWarning  Severity = "warning"
	SeverityInfo     Severity = "info"
)

type Issue struct {
	Severity Severity `json:"severity"`
	Title    string   `json:"title"`
	Detail   string   `json:"detail"`
	Fix      string   `json:"fix,omitempty"`
}

func Check(root string) []Issue {
	var issues []Issue
	issues = append(issues, checkShimPresent(root)...)
	issues = append(issues, checkDefaultVersion(root)...)
	issues = append(issues, checkCacheWritable(root)...)
	issues = append(issues, checkStateFiles(root)...)
	issues = append(issues, checkExtensionManifests(root)...)
	issues = append(issues, checkPHPVEnv(root)...)
	issues = append(issues, checkShimInPath(root)...)
	issues = append(issues, checkSystemMode(root)...)
	issues = append(issues, checkDiskSpace(root)...)
	return issues
}

func checkShimPresent(root string) []Issue {
	shimPath := filepath.Join(root, "bin", "php")
	if _, err := os.Stat(shimPath); os.IsNotExist(err) {
		return []Issue{{
			Severity: SeverityWarning,
			Title:    "Shim not found",
			Detail:   fmt.Sprintf("Expected shim at %s", shimPath),
			Fix:      "Run `phpv init` to generate shims, or `phpv rehash` to regenerate",
		}}
	}
	return nil
}

func checkDefaultVersion(root string) []Issue {
	defaultPath := filepath.Join(root, "default")
	data, err := os.ReadFile(defaultPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []Issue{{
				Severity: SeverityInfo,
				Title:    "No default version set",
				Detail:   "No default PHP version has been configured",
				Fix:      "Run `phpv default <version>` to set one",
			}}
		}
		return []Issue{{
			Severity: SeverityWarning,
			Title:    "Cannot read default version",
			Detail:   fmt.Sprintf("Error reading %s: %v", defaultPath, err),
		}}
	}
	ver := strings.TrimSpace(string(data))
	if ver == "" {
		return []Issue{{
			Severity: SeverityInfo,
			Title:    "Default version is empty",
			Detail:   "The default version file exists but is empty",
			Fix:      "Run `phpv default <version>` to set one",
		}}
	}
	phpBin := filepath.Join(root, "packages", "php", ver, "bin", "php")
	if _, err := os.Stat(phpBin); os.IsNotExist(err) {
		return []Issue{{
			Severity: SeverityCritical,
			Title:    "Default version not installed",
			Detail:   fmt.Sprintf("Default version %s is not installed at %s", ver, phpBin),
			Fix:      fmt.Sprintf("Run `phpv install %s` or `phpv default <version>` to set a different default", ver),
		}}
	}
	return nil
}

func checkCacheWritable(root string) []Issue {
	cacheDir := filepath.Join(root, "caches")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return []Issue{{
			Severity: SeverityCritical,
			Title:    "Cache directory not writable",
			Detail:   fmt.Sprintf("Cannot create cache directory at %s: %v", cacheDir, err),
			Fix:      "Check permissions on " + root,
		}}
	}
	testFile := filepath.Join(cacheDir, ".phpv_write_test")
	if err := os.WriteFile(testFile, []byte{}, 0o644); err != nil {
		return []Issue{{
			Severity: SeverityCritical,
			Title:    "Cache directory not writable",
			Detail:   fmt.Sprintf("Cannot write to %s: %v", cacheDir, err),
			Fix:      "Check permissions on " + cacheDir,
		}}
	}
	os.Remove(testFile)
	return nil
}

func checkStateFiles(root string) []Issue {
	phpDir := filepath.Join(root, "packages", "php")
	entries, err := os.ReadDir(phpDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return []Issue{{
			Severity: SeverityWarning,
			Title:    "Cannot read PHP packages directory",
			Detail:   fmt.Sprintf("Error reading %s: %v", phpDir, err),
		}}
	}
	var issues []Issue
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		statePath := filepath.Join(phpDir, e.Name(), ".state")
		data, err := os.ReadFile(statePath)
		if err != nil {
			if os.IsNotExist(err) {
				issues = append(issues, Issue{
					Severity: SeverityWarning,
					Title:    fmt.Sprintf("Missing state file for PHP %s", e.Name()),
					Detail:   fmt.Sprintf("No .state file found at %s", statePath),
					Fix:      fmt.Sprintf("Run `phpv install %s` to reinstall, or remove the directory manually", e.Name()),
				})
			}
			continue
		}
		state := domain.InstallState(strings.TrimSpace(string(data)))
		if state == domain.StateFailed {
			issues = append(issues, Issue{
				Severity: SeverityWarning,
				Title:    fmt.Sprintf("PHP %s installation failed", e.Name()),
				Detail:   fmt.Sprintf("State file at %s contains 'failed'", statePath),
				Fix:      fmt.Sprintf("Run `phpv install %s --clean` to retry", e.Name()),
			})
		}
	}
	return issues
}

func checkExtensionManifests(root string) []Issue {
	phpDir := filepath.Join(root, "packages", "php")
	entries, err := os.ReadDir(phpDir)
	if err != nil {
		return nil
	}
	var issues []Issue
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		manifestPath := filepath.Join(phpDir, e.Name(), "extensions.json")
		if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
			continue
		}
		data, err := os.ReadFile(manifestPath)
		if err != nil {
			issues = append(issues, Issue{
				Severity: SeverityWarning,
				Title:    fmt.Sprintf("Cannot read extension manifest for PHP %s", e.Name()),
				Detail:   fmt.Sprintf("Error reading %s: %v", manifestPath, err),
				Fix:      fmt.Sprintf("Delete %s and reinstall extensions", manifestPath),
			})
		}
		_ = data
	}
	return issues
}

func checkPHPVEnv(root string) []Issue {
	envRoot := os.Getenv("PHPV_ROOT")
	if envRoot != "" && envRoot != root {
		return []Issue{{
			Severity: SeverityWarning,
			Title:    "PHPV_ROOT mismatch",
			Detail:   fmt.Sprintf("PHPV_ROOT is set to %q but resolved root is %q", envRoot, root),
			Fix:      "Unset PHPV_ROOT or set it to " + root,
		}}
	}
	return nil
}

func checkShimInPath(root string) []Issue {
	binDir := filepath.Join(root, "bin")
	path := os.Getenv("PATH")
	for _, dir := range filepath.SplitList(path) {
		if dir == binDir {
			return nil
		}
	}
	return []Issue{{
		Severity: SeverityInfo,
		Title:    "Shim directory not in PATH",
		Detail:   fmt.Sprintf("%s is not in your PATH", binDir),
		Fix:      "Run `phpv init` and add the output to your shell profile",
	}}
}

func checkSystemMode(root string) []Issue {
	systemMarker := filepath.Join(root, ".phpv_system")
	if _, err := os.Stat(systemMarker); err == nil {
		return []Issue{{
			Severity: SeverityInfo,
			Title:    "System mode is active",
			Detail:   "phpv is currently using the system PHP instead of a managed version",
			Fix:      "Run `phpv use <version>` to switch to a managed version",
		}}
	}
	return nil
}

func checkDiskSpace(root string) []Issue {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(root, &stat); err != nil {
		return nil
	}
	freeBytes := stat.Bavail * uint64(stat.Bsize)
	if freeBytes < 500*1024*1024 {
		return []Issue{{
			Severity: SeverityWarning,
			Title:    "Low disk space",
			Detail:   fmt.Sprintf("Only %d MB free on %s", freeBytes/(1024*1024), root),
			Fix:      "Free up disk space or move PHPV_ROOT to a different partition",
		}}
	}
	return nil
}
