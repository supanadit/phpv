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

var doctorPkgNames = map[string]map[string]string{
	"make":       {"brew": "make", "apt": "build-essential", "*": "make"},
	"gcc":        {"brew": "gcc", "apt": "build-essential", "*": "gcc"},
	"g++":        {"brew": "gcc", "apt": "build-essential", "*": "gcc-c++"},
	"pkg-config": {"brew": "pkg-config", "apt": "pkg-config", "*": "pkgconfig"},
	"bison":      {"*": "bison"},
	"flex":       {"*": "flex"},
	"re2c":       {"*": "re2c"},
	"autoconf":   {"*": "autoconf"},
	"automake":   {"*": "automake"},
	"libtool":    {"brew": "libtool", "*": "libtool"},
	"m4":         {"*": "m4"},
	"perl":       {"*": "perl"},
	"cmake":      {"brew": "cmake", "*": "cmake"},
	"xz":         {"brew": "xz", "*": "xz"},
	"libxml2":    {"brew": "libxml2", "apt": "libxml2-dev", "*": "libxml2-devel"},
	"openssl":    {"brew": "openssl", "apt": "libssl-dev", "*": "openssl-devel"},
	"curl":       {"brew": "curl", "apt": "libcurl4-openssl-dev", "*": "libcurl-devel"},
	"zlib":       {"brew": "zlib", "apt": "zlib1g-dev", "*": "zlib-devel"},
	"oniguruma":  {"brew": "oniguruma", "apt": "libonig-dev", "*": "oniguruma-devel"},
	"icu":        {"brew": "icu4c", "apt": "libicu-dev", "*": "libicu-devel"},
	"bzip2":      {"brew": "bzip2", "apt": "libbz2-dev", "*": "bzip2-devel"},
	"zig":        {"brew": "zig", "*": "zig"},
}

// Tools that phpv auto-downloads (pre-built binary) — never a blocker
var autodownloadTools = map[string]bool{
	"zig":   true,
	"cmake": true,
}

// Tools that phpv builds from source — not required on system
var autoBuildTools = map[string]bool{
	"m4":       true,
	"autoconf": true,
	"automake": true,
	"libtool":  true,
	"perl":     true,
	"bison":    true,
	"flex":     true,
	"re2c":     true,
}

// Libraries that phpv builds from source — system package optional
var autoBuildLibs = map[string]bool{
	"libxml2":   true,
	"openssl":   true,
	"curl":      true,
	"zlib":      true,
	"oniguruma": true,
	"icu":       true,
}

func doctorSuggestion(name string, osInfo utils.OSInfo) string {
	names, ok := doctorPkgNames[name]
	if !ok {
		return osInfo.InstallCmd + " " + name
	}
	pkg, ok := names[osInfo.PkgMgr]
	if !ok {
		pkg = names["*"]
	}
	return osInfo.InstallCmd + " " + pkg
}

func (h *TerminalHandler) DoctorV2(version string) (*DoctorResultV2, error) {
	osInfo := utils.DetectOSInfo()
	buildTools := h.doctorCheckBuildTools(osInfo, version)
	libChecks := h.doctorCheckSystemLibs(osInfo)

	var extChecks []DoctorExtCheck
	var phpInstall *DoctorPHPInstall
	if version != "" {
		extChecks = h.doctorAnalyzeExtensions(version, osInfo)
		phpInstall = h.doctorCheckPHPInstall(version)
	}

	// Count truly missing items (only "system" category)
	sysMissing := 0
	allItems := append(buildTools, libChecks...)
	var sysPkgs []string
	for _, item := range allItems {
		if !item.Available && item.Category == "system" {
			sysMissing++
			pkg := strings.TrimPrefix(item.Suggestion, osInfo.InstallCmd+" ")
			sysPkgs = append(sysPkgs, pkg)
		}
	}

	// Build consolidated quick-fix command (system packages only)
	quickFix := ""
	if len(sysPkgs) > 0 {
		quickFix = osInfo.InstallCmd + " " + strings.Join(sysPkgs, " ")
	}

	// Determine buildable PHP versions from available compilers
	hasGcc := toolAvailable(buildTools, "gcc")
	hasMake := toolAvailable(buildTools, "make")

	// zig and cmake are auto-downloaded — always available
	hasZig := toolAvailable(buildTools, "zig") || autodownloadTools["zig"]

	canBuildPHP8 := hasMake && hasGcc
	canBuildPHP7 := hasMake && hasZig

	// Compute verdict
	verdict := "ready"
	verdictMsg := "System is ready for PHP installation"
	if sysMissing > 0 {
		if !hasMake {
			verdict = "blocked"
			verdictMsg = "Missing core build tools (make). PHP cannot be built."
		} else if !hasGcc && !hasZig {
			verdict = "blocked"
			verdictMsg = "No C compiler found (gcc or zig). PHP cannot be built."
		} else {
			verdict = "minor"
			verdictMsg = fmt.Sprintf("%d system packages recommended. phpv handles the rest automatically.", sysMissing)
		}
	}

	summary := fmt.Sprintf("System: %s | System packages needed: %d", verdict, sysMissing)
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
		BuildTools:   buildTools,
		LibChecks:    libChecks,
		Extensions:   extChecks,
		PHPInstall:   phpInstall,
		Verdict:      verdict,
		VerdictMsg:   verdictMsg,
		CanBuildPHP8: canBuildPHP8,
		CanBuildPHP7: canBuildPHP7,
		QuickFix:     quickFix,
		Summary:      summary,
	}, nil
}

func toolAvailable(items []DoctorCheckItem, name string) bool {
	for _, item := range items {
		if item.Name == name {
			return item.Available
		}
	}
	return false
}

func usesZig(version string) bool {
	if version == "" {
		return false
	}
	v := utils.ParseVersion(version)
	if v.Major < 5 {
		return true
	}
	return false
}

func (h *TerminalHandler) doctorCheckBuildTools(osInfo utils.OSInfo, version string) []DoctorCheckItem {
	useZig := usesZig(version)

	var tools []struct {
		name    string
		version []string
	}

	if useZig {
		tools = []struct {
			name    string
			version []string
		}{
			{"make", []string{"--version"}},
			{"zig", []string{"version"}},
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
	} else {
		tools = []struct {
			name    string
			version []string
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
	}

	var items []DoctorCheckItem
	for _, tool := range tools {
		item := DoctorCheckItem{Name: tool.name}
		path, err := exec.LookPath(tool.name)
		if err != nil {
			item.Available = false
			switch {
			case autodownloadTools[tool.name]:
				item.Category = "autodownload"
				item.Suggestion = "phpv will auto-download"
			case autoBuildTools[tool.name]:
				item.Category = "buildable"
				item.Suggestion = "phpv will build from source"
			default:
				item.Category = "system"
				item.Suggestion = doctorSuggestion(tool.name, osInfo)
			}
			items = append(items, item)
			continue
		}
		item.Available = true
		item.Category = "available"
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

func (h *TerminalHandler) doctorCheckSystemLibs(osInfo utils.OSInfo) []DoctorCheckItem {
	libs := []struct {
		name        string
		pkgConfig   string
		headerPaths []string
	}{
		{"libxml2", "libxml-2.0", []string{"/usr/include/libxml2/libxml/parser.h"}},
		{"openssl", "openssl", []string{"/usr/include/openssl/ssl.h"}},
		{"curl", "libcurl", []string{"/usr/include/curl/curl.h"}},
		{"zlib", "zlib", []string{"/usr/include/zlib.h"}},
		{"oniguruma", "oniguruma", []string{"/usr/include/oniguruma/onigmo.h", "/usr/include/oniguruma/oniguruma.h", "/usr/include/oniguruma.h"}},
		{"icu", "icu-uc", []string{"/usr/include/unicode/umachine.h"}},
	}

	var items []DoctorCheckItem
	for _, lib := range libs {
		item := DoctorCheckItem{Name: lib.name}

		// Try pkg-config with system paths (same logic as advisor's PkgConfigExists)
		sysPkgPaths := utils.GetSystemPkgConfigPaths()
		pkgEnv := append(os.Environ(), "PKG_CONFIG_PATH="+strings.Join(sysPkgPaths, ":"))

		cmd := exec.Command("pkg-config", "--exists", lib.pkgConfig)
		cmd.Env = pkgEnv
		if cmd.Run() == nil {
			verCmd := exec.Command("pkg-config", "--modversion", lib.pkgConfig)
			verCmd.Env = pkgEnv
			if verOut, err := verCmd.Output(); err == nil {
				item.Version = strings.TrimSpace(string(verOut))
			}
			item.Available = true
			item.Category = "available"
			items = append(items, item)
			continue
		}

		// Fallback to header check
		for _, hPath := range lib.headerPaths {
			if _, err := os.Stat(hPath); err == nil {
				item.Available = true
				item.Version = "(headers only)"
				item.Category = "available"
				break
			}
		}

		if !item.Available {
			item.Category = "buildable"
			item.Suggestion = "phpv will build from source"
		}
		items = append(items, item)
	}
	return items
}

func (h *TerminalHandler) doctorAnalyzeExtensions(version string, osInfo utils.OSInfo) []DoctorExtCheck {
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
		case "libpq":
			pkgConfigName = "libpq"
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
		check.Suggestion = doctorSuggestion(ext.Package, osInfo)
		check.Status = "missing"
		checks = append(checks, check)
	}
	return checks
}

func (h *TerminalHandler) doctorCheckPHPInstall(version string) *DoctorPHPInstall {
	silo, err := h.Silo.GetSilo()
	if err != nil {
		return &DoctorPHPInstall{Version: version, Installed: false}
	}

	phpBinary := filepath.Join(utils.PHPOutputPath(silo, version), "bin", "php")
	if _, err := os.Stat(phpBinary); os.IsNotExist(err) {
		return &DoctorPHPInstall{Version: version, Installed: false}
	}

	result := &DoctorPHPInstall{
		Version:    version,
		Installed:  true,
		BinaryPath: phpBinary,
	}

	// Get configure flags
	cmd := exec.Command(phpBinary, "-i")
	out, err := cmd.Output()
	if err == nil {
		for _, line := range strings.Split(string(out), "\n") {
			if strings.Contains(line, "Configure Command") {
				if idx := strings.Index(line, "=>"); idx != -1 {
					cfg := strings.TrimSpace(line[idx+2:])
					if len(cfg) > 200 {
						cfg = cfg[:200] + "..."
					}
					result.ConfigFlags = cfg
				}
				break
			}
		}
	}

	// Get loaded extensions from php -m
	cmd = exec.Command(phpBinary, "-m")
	out, err = cmd.Output()
	if err == nil {
		started := false
		for _, line := range strings.Split(string(out), "\n") {
			line = strings.TrimSpace(line)
			if line == "[PHP Modules]" {
				started = true
				continue
			}
			if line == "" || line == "[Zend Modules]" {
				if started {
					break
				}
				continue
			}
			if started {
				result.EnabledExts = append(result.EnabledExts, line)
			}
		}
	}

	return result
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
