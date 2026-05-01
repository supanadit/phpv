package terminal

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/internal/utils"
	"github.com/supanadit/phpv/shim"
)

func (h *TerminalHandler) Which() (string, error) {
	silo, err := h.Silo.GetSilo()
	if err != nil {
		return "", fmt.Errorf("failed to get silo: %w", err)
	}

	if shim.IsSystemMode(silo.Root) {
		phpvBin := filepath.Join(silo.Root, "bin")
		pathEnv := os.Getenv("PATH")
		var filteredParts []string
		for _, part := range strings.Split(pathEnv, ":") {
			if part != phpvBin && !strings.HasPrefix(part, silo.Root+"/") {
				filteredParts = append(filteredParts, part)
			}
		}
		filteredPath := strings.Join(filteredParts, ":")

		cmd := exec.Command("which", "php")
		cmd.Env = append(os.Environ(), "PATH="+filteredPath)
		out, err := cmd.Output()
		if err == nil {
			return strings.TrimSpace(string(out)), nil
		}
	}

	activeVer, err := h.resolveActivePHP()
	if err != nil {
		return "", err
	}

	if activeVer == "" {
		return "", nil
	}

	phpPath := filepath.Join(utils.PHPOutputPath(silo, activeVer), "bin", "php")
	if _, err := os.Stat(phpPath); os.IsNotExist(err) {
		return "", nil
	}
	return phpPath, nil
}

func (h *TerminalHandler) Uninstall(constraint string) (*UninstallResult, error) {
	exactVersion, err := h.resolveInstalledVersion(constraint)
	if err != nil {
		return nil, fmt.Errorf("version not installed: %w", err)
	}

	silo, err := h.Silo.GetSilo()
	if err != nil {
		return nil, fmt.Errorf("failed to get silo: %w", err)
	}

	outputPath := utils.PHPOutputPath(silo, exactVersion)
	phpBinary := filepath.Join(outputPath, "bin", "php")
	if _, err := os.Stat(phpBinary); os.IsNotExist(err) {
		h.Silo.RemovePHPInstallation(exactVersion)
		return nil, fmt.Errorf("PHP %s is not properly installed (binary not found: %s), cleaning up stale data", exactVersion, phpBinary)
	}

	state, err := h.Silo.GetState(exactVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to get state: %w", err)
	}
	if state == domain.StateNone {
		return nil, fmt.Errorf("version %s is not installed", exactVersion)
	}

	defaultVer, _ := h.Silo.GetDefault()
	wasDefault := defaultVer == exactVersion

	removedTools, err := h.Silo.RemovePHPInstallation(exactVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to uninstall: %w", err)
	}

	shimPath := utils.BinPath(silo)
	for _, name := range []string{"php", "php-cgi", "phpize", "php-config"} {
		shimPath := filepath.Join(shimPath, name)
		_ = os.RemoveAll(shimPath)
	}

	return &UninstallResult{
		Version:      exactVersion,
		RemovedTools: removedTools,
		WasDefault:   wasDefault,
	}, nil
}

func (h *TerminalHandler) CleanBuildTools(dryRun bool) (*CleanBuildToolsResult, error) {
	removed, willRemove, err := h.Silo.RemoveUnusedBuildTools(dryRun)
	if err != nil {
		return nil, err
	}

	return &CleanBuildToolsResult{
		Removed:    removed,
		WillRemove: willRemove,
		DryRun:     dryRun,
	}, nil
}

func (h *TerminalHandler) DoctorV2(version string) (*DoctorResultV2, error) {
	buildTools := h.doctorCheckBuildTools()
	libChecks := h.doctorCheckSystemLibs()

	var extChecks []DoctorExtCheck
	if version != "" {
		extChecks = h.doctorAnalyzeExtensions(version)
	}

	missingTools := 0
	for _, t := range buildTools {
		if !t.Available {
			missingTools++
		}
	}
	missingLibs := 0
	for _, l := range libChecks {
		if !l.Available {
			missingLibs++
		}
	}

	summary := fmt.Sprintf("Build tools: %d missing | Libraries: %d missing", missingTools, missingLibs)
	if len(extChecks) > 0 {
		missingExt := 0
		for _, e := range extChecks {
			if e.Status == "missing" {
				missingExt++
			}
		}
		summary += fmt.Sprintf(" | Extensions: %d missing", missingExt)
	}

	return &DoctorResultV2{
		BuildTools: buildTools,
		LibChecks:  libChecks,
		Extensions: extChecks,
		Summary:    summary,
	}, nil
}

func (h *TerminalHandler) doctorCheckBuildTools() []DoctorCheckItem {
	tools := []struct {
		name    string
		version []string // version flag variations
	}{
		{"make", []string{"--version"}},
		{"gcc", []string{"--version"}},
		{"g++", []string{"--version"}},
		{"pkg-config", []string{"--version"}},
		{"bison", []string{"--version"}},
		{"flex", []string{"--version"}},
		{"re2c", []string{"--version"}},
		{"autoconf", []string{"--version"}},
		{"automake", []string{"--version"}},
		{"libtool", []string{"--version"}},
		{"m4", []string{"--version"}},
		{"perl", []string{"--version"}},
		{"cmake", []string{"--version"}},
		{"xz", []string{"--version"}},
	}

	suggestions := map[string]string{
		"make":       "sudo dnf install make  # or: sudo apt install build-essential",
		"gcc":        "sudo dnf install gcc  # or: sudo apt install build-essential",
		"g++":        "sudo dnf install gcc-c++  # or: sudo apt install build-essential",
		"pkg-config": "sudo dnf install pkgconfig  # or: sudo apt install pkg-config",
		"bison":      "sudo dnf install bison  # or: sudo apt install bison",
		"flex":       "sudo dnf install flex  # or: sudo apt install flex",
		"re2c":       "sudo dnf install re2c  # or: sudo apt install re2c",
		"autoconf":   "sudo dnf install autoconf  # or: sudo apt install autoconf",
		"automake":   "sudo dnf install automake  # or: sudo apt install automake",
		"libtool":    "sudo dnf install libtool  # or: sudo apt install libtool",
		"m4":         "sudo dnf install m4  # or: sudo apt install m4",
		"perl":       "sudo dnf install perl  # or: sudo apt install perl",
		"cmake":      "sudo dnf install cmake  # or: sudo apt install cmake",
		"xz":         "sudo dnf install xz  # or: sudo apt install xz",
	}

	var items []DoctorCheckItem
	for _, tool := range tools {
		item := DoctorCheckItem{Name: tool.name}
		path, err := exec.LookPath(tool.name)
		if err != nil {
			item.Available = false
			item.Suggestion = suggestions[tool.name]
			items = append(items, item)
			continue
		}
		item.Available = true
		ver := getToolVersion(path, tool.version)
		if ver != "" {
			item.Version = ver
		}
		items = append(items, item)
	}
	return items
}

func getToolVersion(path string, flags []string) string {
	for _, flag := range flags {
		cmd := exec.Command(path, flag)
		out, err := cmd.Output()
		if err != nil {
			continue
		}
		line := string(out)
		// Take first non-empty line, strip common prefixes
		for _, l := range strings.Split(line, "\n") {
			l = strings.TrimSpace(l)
			if l == "" {
				continue
			}
			// Return first few words as version ID
			parts := strings.Fields(l)
			if len(parts) > 2 {
				return parts[0] + " " + parts[1] + " " + parts[2]
			}
			if len(parts) > 0 {
				return parts[0]
			}
		}
	}
	return ""
}

func (h *TerminalHandler) doctorCheckSystemLibs() []DoctorCheckItem {
	libs := []struct {
		name        string
		pkgConfig   string
		headerPaths []string
	}{
		{"libxml2", "libxml-2.0", []string{"/usr/include/libxml2/libxml/parser.h"}},
		{"openssl", "openssl", []string{"/usr/include/openssl/ssl.h"}},
		{"curl", "libcurl", []string{"/usr/include/curl/curl.h"}},
		{"zlib", "zlib", []string{"/usr/include/zlib.h"}},
		{"oniguruma", "oniguruma", []string{"/usr/include/oniguruma/onigmo.h", "/usr/include/oniguruma/oniguruma.h"}},
		{"icu", "icu-uc", []string{"/usr/include/unicode/umachine.h"}},
	}

	suggestions := map[string]string{
		"libxml2":   "sudo dnf install libxml2-devel  # or: sudo apt install libxml2-dev",
		"openssl":   "sudo dnf install openssl-devel  # or: sudo apt install libssl-dev",
		"curl":      "sudo dnf install libcurl-devel  # or: sudo apt install libcurl4-openssl-dev",
		"zlib":      "sudo dnf install zlib-devel  # or: sudo apt install zlib1g-dev",
		"oniguruma": "sudo dnf install oniguruma-devel  # or: sudo apt install libonig-dev",
		"icu":       "sudo dnf install libicu-devel  # or: sudo apt install libicu-dev",
	}

	var items []DoctorCheckItem
	for _, lib := range libs {
		item := DoctorCheckItem{Name: lib.name}

		// Try pkg-config first
		cmd := exec.Command("pkg-config", "--exists", lib.pkgConfig)
		if cmd.Run() == nil {
			verCmd := exec.Command("pkg-config", "--modversion", lib.pkgConfig)
			if verOut, err := verCmd.Output(); err == nil {
				item.Version = strings.TrimSpace(string(verOut))
			}
			item.Available = true
			items = append(items, item)
			continue
		}

		// Fallback to header check
		for _, hPath := range lib.headerPaths {
			if _, err := os.Stat(hPath); err == nil {
				item.Available = true
				item.Version = "(headers only)"
				break
			}
		}

		if !item.Available {
			item.Suggestion = suggestions[lib.name]
		}
		items = append(items, item)
	}
	return items
}

func (h *TerminalHandler) doctorAnalyzeExtensions(version string) []DoctorExtCheck {
	var checks []DoctorExtCheck

	exts := h.ExtensionRepo.ListExtensionsForPHP(version)
	for _, ext := range exts {
		check := DoctorExtCheck{
			Extension: ext.Name,
			Flag:      ext.Flag,
			Package:   ext.Package,
		}

			if ext.Package == "" {
			check.Status = "builtin"
			checks = append(checks, check)
			continue
		}

		// Check if system provides the backing library via pkg-config
		pkgConfigName := ext.Package
		switch ext.Package {
		case "libxml2":
			pkgConfigName = "libxml-2.0"
		case "curl":
			pkgConfigName = "libcurl"
		case "oniguruma":
			pkgConfigName = "oniguruma"
		case "icu":
			pkgConfigName = "icu-uc"
		}

		cmd := exec.Command("pkg-config", "--exists", pkgConfigName)
		if cmd.Run() == nil {
			verCmd := exec.Command("pkg-config", "--modversion", pkgConfigName)
			if verOut, err := verCmd.Output(); err == nil {
				check.SystemVer = strings.TrimSpace(string(verOut))
			}
			check.Status = "system"

			// Look up version constraint for this extension + PHP version
			if _, verWithConstraint, ok := h.ExtensionRepo.GetExtensionDependencyWithVersion(ext.Name, version); ok {
				if idx := strings.Index(verWithConstraint, "|"); idx != -1 {
					constraint := verWithConstraint[idx+1:]
					check.ExpectedVer = constraint
					if check.SystemVer != "" && !utils.MatchVersionRange(constraint, check.SystemVer) {
						check.Status = "mismatch"
					}
				}
			}

			checks = append(checks, check)
			continue
		}

		// phpv can build these from source
		buildablePkgs := map[string]string{
			"libxml2":   "phpv builds libxml2 from source",
			"openssl":   "phpv builds openssl from source",
			"curl":      "phpv builds curl from source",
			"zlib":      "phpv builds zlib from source",
			"oniguruma": "phpv builds oniguruma from source",
			"icu":       "phpv builds icu from source",
		}
		if msg, ok := buildablePkgs[ext.Package]; ok {
			check.Status = "build"
			check.Suggestion = msg
			checks = append(checks, check)
			continue
		}

		// Not available on system, not buildable by phpv
		switch ext.Package {
		case "bzip2":
			check.Suggestion = "sudo dnf install bzip2-devel  # or: sudo apt install libbz2-dev"
		default:
			check.Suggestion = "sudo dnf install " + ext.Package + "-devel  # or: sudo apt install " + ext.Package + "-dev"
		}
		check.Status = "missing"
		checks = append(checks, check)
	}
	return checks
}

func (h *TerminalHandler) Doctor() (*DoctorResult, error) {
	var issues []DoctorIssue
	var warnings []DoctorWarning

	if runtime.GOOS == "linux" {
		requiredCommands := []string{"make", "gcc", "bison", "flex", "re2c", " autoconf", "automake", "libtool", "pkg-config"}
		for _, cmd := range requiredCommands {
			name := strings.TrimPrefix(cmd, " ")
			if name == "autoconf" || name == "automake" || name == "libtool" {
				name = cmd
			}
			if _, err := exec.LookPath(name); err != nil {
				issues = append(issues, DoctorIssue{
					Category: "system",
					Message:  fmt.Sprintf("required command not found: %s", name),
				})
			}
		}

		if _, err := exec.LookPath("xz"); err != nil {
			issues = append(issues, DoctorIssue{
				Category: "system",
				Message:  "xz utility not found (required for extracting .tar.xz archives)",
			})
		}
	}

	silo, err := h.Silo.GetSilo()
	if err != nil {
		issues = append(issues, DoctorIssue{
			Category: "phpv",
			Message:  fmt.Sprintf("failed to get silo: %v", err),
		})
	} else {
		if _, err := os.Stat(silo.Root); os.IsNotExist(err) {
			issues = append(issues, DoctorIssue{
				Category: "phpv",
				Message:  fmt.Sprintf("PHPV_ROOT does not exist: %s", silo.Root),
			})
		}

		defaultVer, _ := h.Silo.GetDefault()
		if defaultVer != "" {
			phpPath := filepath.Join(utils.PHPOutputPath(silo, defaultVer), "bin", "php")
			if _, err := os.Stat(phpPath); os.IsNotExist(err) {
				warnings = append(warnings, DoctorWarning{
					Category: "phpv",
					Message:  fmt.Sprintf("default PHP version set to %s but binary not found", defaultVer),
				})
			}
		}
	}

	if len(issues) == 0 && len(warnings) == 0 {
		warnings = append(warnings, DoctorWarning{
			Category: "system",
			Message:  "no issues detected",
		})
	}

	return &DoctorResult{
		Issues:   issues,
		Warnings: warnings,
	}, nil
}

func (h *TerminalHandler) GetInitCode(shell string) (string, error) {
	phpvRoot := GetPHPvRoot()
	return GetInitCodeForShell(shell, phpvRoot), nil
}

func (h *TerminalHandler) GetPHPvRoot() string {
	return GetPHPvRoot()
}
