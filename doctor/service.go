package doctor

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/system"
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

type Service struct {
	repo     Repository
	sysSvc   *system.Service
}

func NewService(repo Repository, sysSvc *system.Service) *Service {
	return &Service{repo: repo, sysSvc: sysSvc}
}

func (s *Service) Check(root string) []Issue {
	var issues []Issue
	issues = append(issues, s.checkDistroInfo()...)
	issues = append(issues, s.checkBuildTools()...)
	issues = append(issues, s.checkSystemPackages()...)
	issues = append(issues, s.checkShimPresent(root)...)
	issues = append(issues, s.checkDefaultVersion(root)...)
	issues = append(issues, s.checkCacheWritable(root)...)
	issues = append(issues, s.checkStateFiles(root)...)
	issues = append(issues, s.checkExtensionManifests(root)...)
	issues = append(issues, s.checkPHPVEnv(root)...)
	issues = append(issues, s.checkShimInPath(root)...)
	issues = append(issues, s.checkSystemMode(root)...)
	issues = append(issues, s.checkDiskSpace(root)...)
	return issues
}

func (s *Service) checkDistroInfo() []Issue {
	d := s.sysSvc.DistroInfo()
	return []Issue{{
		Severity: SeverityInfo,
		Title:    fmt.Sprintf("Detected OS: %s (%s)", d.Name, d.Version),
		Detail:   fmt.Sprintf("Package manager: %s", d.PM),
	}}
}

func (s *Service) checkBuildTools() []Issue {
	criticalTools := []string{"gcc", "g++", "make"}
	optionalTools := []string{"cmake", "autoconf", "automake", "m4", "perl", "bison", "re2c", "flex", "pkg-config", "xz"}

	var issues []Issue

	// Check critical tools first (must be on PATH)
	var missingCritical []string
	for _, tool := range criticalTools {
		if _, err := s.repo.LookPath(tool); err != nil {
			missingCritical = append(missingCritical, tool)
		}
	}
	if len(missingCritical) > 0 {
		issues = append(issues, Issue{
			Severity: SeverityCritical,
			Title:    fmt.Sprintf("Missing build tools: %s", strings.Join(missingCritical, ", ")),
			Detail:   fmt.Sprintf("These tools are required to compile PHP from source: %s", strings.Join(missingCritical, ", ")),
			Fix:      s.installCommandFor(missingCritical),
		})
	}

	// Check optional build tools
	var missingOptional []string
	for _, tool := range optionalTools {
		if _, err := s.repo.LookPath(tool); err != nil {
			missingOptional = append(missingOptional, tool)
		}
	}
	if len(missingOptional) > 0 {
		issues = append(issues, Issue{
			Severity: SeverityWarning,
			Title:    fmt.Sprintf("Optional build tools missing: %s", strings.Join(missingOptional, ", ")),
			Detail:   fmt.Sprintf("Some packages may require: %s", strings.Join(missingOptional, ", ")),
			Fix:      s.installCommandFor(missingOptional),
		})
	}

	return issues
}

func (s *Service) checkSystemPackages() []Issue {
	phpDeps := []string{"openssl", "libxml2", "zlib", "oniguruma", "curl", "sqlite3", "readline", "icu", "pcre2", "argon2", "sodium"}
	result, err := s.sysSvc.Check(phpDeps)
	if err != nil {
		return []Issue{{
			Severity: SeverityWarning,
			Title:    "Could not check system packages",
			Detail:   fmt.Sprintf("Error: %v", err),
		}}
	}
	if len(result.Missing) == 0 {
		return nil
	}
	var names []string
	for _, p := range result.Missing {
		names = append(names, p.Name)
	}
	return []Issue{{
		Severity: SeverityWarning,
		Title:    fmt.Sprintf("Missing system libraries (%d of %d)", len(result.Missing), len(phpDeps)),
		Detail:   fmt.Sprintf("Required dev libraries not installed: %s", strings.Join(names, ", ")),
		Fix:      s.sysSvc.InstallCommand(result.Missing),
	}}
}

func (s *Service) installCommandFor(tools []string) string {
	pkgs := make([]system.Package, 0, len(tools))
	for _, name := range tools {
		pkgs = append(pkgs, system.Package{Name: name, SystemName: name})
	}
	cmd := s.sysSvc.InstallCommand(pkgs)
	if cmd == "" {
		return fmt.Sprintf("Install %s using your system package manager", strings.Join(tools, ", "))
	}
	return cmd
}

func (s *Service) checkShimPresent(root string) []Issue {
	shimPath := filepath.Join(root, "bin", "php")
	if _, err := s.repo.Stat(shimPath); s.repo.IsNotExist(err) {
		return []Issue{{
			Severity: SeverityWarning,
			Title:    "Shim not found",
			Detail:   fmt.Sprintf("Expected shim at %s", shimPath),
			Fix:      "Run `phpv init` to generate shims, or `phpv rehash` to regenerate",
		}}
	}
	return nil
}

func (s *Service) checkDefaultVersion(root string) []Issue {
	defaultPath := filepath.Join(root, "default")
	data, err := s.repo.ReadFile(defaultPath)
	if err != nil {
		if s.repo.IsNotExist(err) {
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
	if _, err := s.repo.Stat(phpBin); s.repo.IsNotExist(err) {
		return []Issue{{
			Severity: SeverityCritical,
			Title:    "Default version not installed",
			Detail:   fmt.Sprintf("Default version %s is not installed at %s", ver, phpBin),
			Fix:      fmt.Sprintf("Run `phpv install %s` or `phpv default <version>` to set a different default", ver),
		}}
	}
	return nil
}

func (s *Service) checkCacheWritable(root string) []Issue {
	cacheDir := filepath.Join(root, "caches")
	if err := s.repo.MkdirAll(cacheDir, 0o755); err != nil {
		return []Issue{{
			Severity: SeverityCritical,
			Title:    "Cache directory not writable",
			Detail:   fmt.Sprintf("Cannot create cache directory at %s: %v", cacheDir, err),
			Fix:      "Check permissions on " + root,
		}}
	}
	testFile := filepath.Join(cacheDir, ".phpv_write_test")
	if err := s.repo.WriteFile(testFile, []byte{}, 0o644); err != nil {
		return []Issue{{
			Severity: SeverityCritical,
			Title:    "Cache directory not writable",
			Detail:   fmt.Sprintf("Cannot write to %s: %v", cacheDir, err),
			Fix:      "Check permissions on " + cacheDir,
		}}
	}
	s.repo.Remove(testFile)
	return nil
}

func (s *Service) checkStateFiles(root string) []Issue {
	phpDir := filepath.Join(root, "packages", "php")
	entries, err := s.repo.ReadDir(phpDir)
	if err != nil {
		if s.repo.IsNotExist(err) {
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
		data, err := s.repo.ReadFile(statePath)
		if err != nil {
			if s.repo.IsNotExist(err) {
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
				Fix:      fmt.Sprintf("Run `phpv install %s --force` to retry (deps preserved), or `--clean` to start fresh", e.Name()),
			})
		} else if state == domain.StateInterrupted {
			issues = append(issues, Issue{
				Severity: SeverityWarning,
				Title:    fmt.Sprintf("PHP %s installation was interrupted", e.Name()),
				Detail:   fmt.Sprintf("State file at %s contains 'interrupted'", statePath),
				Fix:      fmt.Sprintf("Run `phpv install %s --force` to retry (deps preserved), or `--clean` to start fresh", e.Name()),
			})
		} else if state == domain.StateInProgress {
			issues = append(issues, Issue{
				Severity: SeverityWarning,
				Title:    fmt.Sprintf("PHP %s installation is in progress (likely crashed)", e.Name()),
				Detail:   fmt.Sprintf("State file at %s contains 'in_progress'", statePath),
				Fix:      fmt.Sprintf("Run `phpv install %s --force` to retry (deps preserved), or `--clean` to start fresh", e.Name()),
			})
		}
	}
	return issues
}

func (s *Service) checkExtensionManifests(root string) []Issue {
	phpDir := filepath.Join(root, "packages", "php")
	entries, err := s.repo.ReadDir(phpDir)
	if err != nil {
		return nil
	}
	var issues []Issue
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		manifestPath := filepath.Join(phpDir, e.Name(), "extensions.json")
		if _, err := s.repo.Stat(manifestPath); s.repo.IsNotExist(err) {
			continue
		}
		data, err := s.repo.ReadFile(manifestPath)
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

func (s *Service) checkPHPVEnv(root string) []Issue {
	envRoot := s.repo.Getenv("PHPV_ROOT")
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

func (s *Service) checkShimInPath(root string) []Issue {
	binDir := filepath.Join(root, "bin")
	for _, dir := range s.repo.PathList() {
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

func (s *Service) checkSystemMode(root string) []Issue {
	systemMarker := filepath.Join(root, ".phpv_system")
	if _, err := s.repo.Stat(systemMarker); err == nil {
		return []Issue{{
			Severity: SeverityInfo,
			Title:    "System mode is active",
			Detail:   "phpv is currently using the system PHP instead of a managed version",
			Fix:      "Run `phpv use <version>` to switch to a managed version",
		}}
	}
	return nil
}

func (s *Service) checkDiskSpace(root string) []Issue {
	bavail, bsize, err := s.repo.Statfs(root)
	if err != nil {
		return nil
	}
	freeBytes := bavail * bsize
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
