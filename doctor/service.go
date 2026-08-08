package doctor

import (
	"fmt"
	"os"
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

// Repository is the diagnostic data-access boundary for the doctor service.
// It exposes only the probes needed to diagnose a phpv installation, hiding
// the underlying filesystem and environment details behind domain-level
// questions.
type Repository interface {
	// LookPath reports whether an executable is available on PATH.
	LookPath(name string) (string, error)

	// ShimExists reports whether the php shim exists under root/bin.
	ShimExists(root string) bool

	// GetDefaultVersion returns the configured default PHP version and
	// whether a default has been set.
	GetDefaultVersion(root string) (version string, exists bool)

	// IsVersionInstalled reports whether a PHP version is installed.
	IsVersionInstalled(root, version string) bool

	// IsCacheWritable verifies that the cache directory can be written to.
	IsCacheWritable(root string) error

	// ListPHPVersions returns the PHP versions installed under the root.
	ListPHPVersions(root string) ([]string, error)

	// GetInstallState returns the install state for a PHP version and
	// whether a state file exists.
	GetInstallState(root, version string) (state domain.InstallState, exists bool)

	// ExtensionManifestReadable reports whether the extension manifest for a
	// PHP version can be read. A missing manifest is not an error.
	ExtensionManifestReadable(root, version string) error

	// GetPHPVRoot returns the PHPV_ROOT environment variable value.
	GetPHPVRoot() string

	// IsDirInPath reports whether dir is present in the PATH.
	IsDirInPath(dir string) bool

	// IsSystemMode reports whether system mode is active.
	IsSystemMode(root string) bool

	// FreeDiskBytes returns the free bytes on the filesystem containing path.
	FreeDiskBytes(path string) (uint64, error)
}

type Service struct {
	repo   Repository
	sysSvc *system.Service
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
	if !s.repo.ShimExists(root) {
		return []Issue{{
			Severity: SeverityWarning,
			Title:    "Shim not found",
			Detail:   fmt.Sprintf("Expected shim at %s/bin/php", root),
			Fix:      "Run `phpv init` to generate shims, or `phpv rehash` to regenerate",
		}}
	}
	return nil
}

func (s *Service) checkDefaultVersion(root string) []Issue {
	ver, exists := s.repo.GetDefaultVersion(root)
	if !exists {
		return []Issue{{
			Severity: SeverityInfo,
			Title:    "No default version set",
			Detail:   "No default PHP version has been configured",
			Fix:      "Run `phpv default <version>` to set one",
		}}
	}
	if ver == "" {
		return []Issue{{
			Severity: SeverityInfo,
			Title:    "Default version is empty",
			Detail:   "The default version file exists but is empty",
			Fix:      "Run `phpv default <version>` to set one",
		}}
	}
	if !s.repo.IsVersionInstalled(root, ver) {
		return []Issue{{
			Severity: SeverityCritical,
			Title:    "Default version not installed",
			Detail:   fmt.Sprintf("Default version %s is not installed", ver),
			Fix:      fmt.Sprintf("Run `phpv install %s` or `phpv default <version>` to set a different default", ver),
		}}
	}
	return nil
}

func (s *Service) checkCacheWritable(root string) []Issue {
	if err := s.repo.IsCacheWritable(root); err != nil {
		return []Issue{{
			Severity: SeverityCritical,
			Title:    "Cache directory not writable",
			Detail:   fmt.Sprintf("Cannot write to %s/caches: %v", root, err),
			Fix:      "Check permissions on " + root,
		}}
	}
	return nil
}

func (s *Service) checkStateFiles(root string) []Issue {
	versions, err := s.repo.ListPHPVersions(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return []Issue{{
			Severity: SeverityWarning,
			Title:    "Cannot read PHP packages directory",
			Detail:   fmt.Sprintf("Error reading %s/packages/php: %v", root, err),
		}}
	}
	var issues []Issue
	for _, ver := range versions {
		state, exists := s.repo.GetInstallState(root, ver)
		if !exists {
			issues = append(issues, Issue{
				Severity: SeverityWarning,
				Title:    fmt.Sprintf("Missing state file for PHP %s", ver),
				Detail:   fmt.Sprintf("No .state file found for PHP %s", ver),
				Fix:      fmt.Sprintf("Run `phpv install %s` to reinstall, or remove the directory manually", ver),
			})
			continue
		}
		switch state {
		case domain.StateFailed:
			issues = append(issues, Issue{
				Severity: SeverityWarning,
				Title:    fmt.Sprintf("PHP %s installation failed", ver),
				Detail:   fmt.Sprintf("State file for PHP %s contains 'failed'", ver),
				Fix:      fmt.Sprintf("Run `phpv install %s --force` to retry (deps preserved), or `--clean` to start fresh", ver),
			})
		case domain.StateInterrupted:
			issues = append(issues, Issue{
				Severity: SeverityWarning,
				Title:    fmt.Sprintf("PHP %s installation was interrupted", ver),
				Detail:   fmt.Sprintf("State file for PHP %s contains 'interrupted'", ver),
				Fix:      fmt.Sprintf("Run `phpv install %s --force` to retry (deps preserved), or `--clean` to start fresh", ver),
			})
		case domain.StateInProgress:
			issues = append(issues, Issue{
				Severity: SeverityWarning,
				Title:    fmt.Sprintf("PHP %s installation is in progress (likely crashed)", ver),
				Detail:   fmt.Sprintf("State file for PHP %s contains 'in_progress'", ver),
				Fix:      fmt.Sprintf("Run `phpv install %s --force` to retry (deps preserved), or `--clean` to start fresh", ver),
			})
		}
	}
	return issues
}

func (s *Service) checkExtensionManifests(root string) []Issue {
	versions, err := s.repo.ListPHPVersions(root)
	if err != nil {
		return nil
	}
	var issues []Issue
	for _, ver := range versions {
		if err := s.repo.ExtensionManifestReadable(root, ver); err != nil {
			issues = append(issues, Issue{
				Severity: SeverityWarning,
				Title:    fmt.Sprintf("Cannot read extension manifest for PHP %s", ver),
				Detail:   fmt.Sprintf("Error reading extension manifest for PHP %s: %v", ver, err),
				Fix:      fmt.Sprintf("Delete the extension manifest for PHP %s and reinstall extensions", ver),
			})
		}
	}
	return issues
}

func (s *Service) checkPHPVEnv(root string) []Issue {
	envRoot := s.repo.GetPHPVRoot()
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
	binDir := root + string(os.PathSeparator) + "bin"
	if s.repo.IsDirInPath(binDir) {
		return nil
	}
	return []Issue{{
		Severity: SeverityInfo,
		Title:    "Shim directory not in PATH",
		Detail:   fmt.Sprintf("%s is not in your PATH", binDir),
		Fix:      "Run `phpv init` and add the output to your shell profile",
	}}
}

func (s *Service) checkSystemMode(root string) []Issue {
	if s.repo.IsSystemMode(root) {
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
	freeBytes, err := s.repo.FreeDiskBytes(root)
	if err != nil {
		return nil
	}
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
