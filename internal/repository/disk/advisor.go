package disk

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
	"github.com/supanadit/phpv/advisor"
	"github.com/supanadit/phpv/assembler"
	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/extension"
	"github.com/supanadit/phpv/internal/config"
	"github.com/supanadit/phpv/internal/repository/memory"
	"github.com/supanadit/phpv/internal/utils"
	"github.com/supanadit/phpv/pattern"
)

type AdvisorRepository struct {
	fs              afero.Fs
	root            string
	exec            *defaultExecutor
	patternRegistry *pattern.Service
	assembler       assembler.AssemblerRepository
	extensionRepo   extension.Repository
}

var (
	libraryPackages = map[string]string{
		"libxml2":   "libxml-2.0",
		"openssl":   "openssl",
		"curl":      "libcurl",
		"zlib":      "zlib",
		"oniguruma": "oniguruma",
		"icu":       "icu-uc",
	}

	// Maps backing library packages back to the extension that depends on them.
	// Used to look up version constraints in shouldBuildFromSource fallback.
	packageExtensionMap = map[string]string{
		"libxml2":   "libxml",
		"openssl":   "openssl",
		"curl":      "curl",
		"zlib":      "zlib",
		"oniguruma": "mbstring",
		"icu":       "intl",
	}

	// packageNameByPM maps generic tool names to distro-specific package names.
	// This is the package-name mapping that was previously in PlatformService.
	packageNameByPM = map[string]map[string]string{
		"apt": {
			"libxml2":   "libxml2-dev",
			"openssl":   "libssl-dev",
			"curl":      "libcurl4-openssl-dev",
			"zlib":      "zlib1g-dev",
			"oniguruma": "libonig-dev",
			"icu":       "libicu-dev",
			"m4":        "m4",
			"autoconf":  "autoconf",
			"automake":  "automake",
			"libtool":   "libtool",
			"perl":      "perl",
			"bison":     "bison",
			"flex":      "flex",
			"re2c":      "re2c",
			"zig":       "zig",
		},
		"dnf": {
			"libxml2":   "libxml2-devel",
			"openssl":   "openssl-devel",
			"curl":      "libcurl-devel",
			"zlib":      "zlib-devel",
			"oniguruma": "oniguruma-devel",
			"icu":       "libicu-devel",
			"m4":        "m4",
			"autoconf":  "autoconf",
			"automake":  "automake",
			"libtool":   "libtool",
			"perl":      "perl",
			"bison":     "bison",
			"flex":      "flex",
			"re2c":      "re2c",
			"zig":       "zig",
		},
		"pacman": {
			"libxml2":   "libxml2",
			"openssl":   "openssl",
			"curl":      "curl",
			"zlib":      "zlib",
			"oniguruma": "oniguruma",
			"icu":       "icu",
			"m4":        "m4",
			"autoconf":  "autoconf",
			"automake":  "automake",
			"libtool":   "libtool",
			"perl":      "perl",
			"bison":     "bison",
			"flex":      "flex",
			"re2c":      "re2c",
			"zig":       "zig",
		},
		"zypper": {
			"libxml2":   "libxml2-devel",
			"openssl":   "libssl-devel",
			"curl":      "libcurl-devel",
			"zlib":      "zlib-devel",
			"oniguruma": "libonig-devel",
			"icu":       "libicu-devel",
			"m4":        "m4",
			"autoconf":  "autoconf",
			"automake":  "automake",
			"libtool":   "libtool",
			"perl":      "perl",
			"bison":     "bison",
			"flex":      "flex",
			"re2c":      "re2c",
			"zig":       "zig",
		},
		"apk": {
			"libxml2":   "libxml2-dev",
			"openssl":   "openssl-dev",
			"curl":      "curl-dev",
			"zlib":      "zlib-dev",
			"oniguruma": "onig-dev",
			"icu":       "icu-dev",
			"m4":        "m4",
			"autoconf":  "autoconf",
			"automake":  "automake",
			"libtool":   "libtool",
			"perl":      "perl",
			"bison":     "bison",
			"flex":      "flex",
			"re2c":      "re2c",
			"zig":       "zig",
		},
		"xbps": {
			"libxml2":   "libxml2-devel",
			"openssl":   "openssl-devel",
			"curl":      "libcurl-devel",
			"zlib":      "zlib-devel",
			"oniguruma": "onig-devel",
			"icu":       "libicu-devel",
			"m4":        "m4",
			"autoconf":  "autoconf",
			"automake":  "automake",
			"libtool":   "libtool",
			"perl":      "perl",
			"bison":     "bison",
			"flex":      "flex",
			"re2c":      "re2c",
			"zig":       "zig",
		},
		"brew": {
			"libxml2":   "libxml2",
			"openssl":   "openssl",
			"curl":      "curl",
			"zlib":      "zlib",
			"oniguruma": "oniguruma",
			"icu":       "icu4c",
			"m4":        "m4",
			"autoconf":  "autoconf",
			"automake":  "automake",
			"libtool":   "libtool",
			"perl":      "perl",
			"bison":     "bison",
			"flex":      "flex",
			"re2c":      "re2c",
			"zig":       "zig",
		},
	}

	multiPkgConfigPackages = map[string][]string{
		"icu": {"icu-uc", "icu-io", "icu-i18n"},
	}

	// ICU version compatibility matrix:
	// - ICU 60-74: C++11 compatible (works with PHP 5.x-8.1)
	// - ICU 75+: Requires C++14+ (needs PHP with C++17 support)
	//
	// For PHP <8.2, we must NOT use system ICU 75+ because:
	// 1. ICU 75+ headers use C++14 features (std::enable_if_t, std::u16string_view)
	// 2. PHP 8.0-8.1 default to C++11 (unless patched)
	// 3. The config.m4 patch helps but we should prefer matching ICU versions
	icuMaxCompatMajor = 74 // max ICU major version compatible with PHP <8.2
)

func NewAdvisorRepository(asm assembler.AssemblerRepository, extRepo extension.Repository) advisor.AdvisorRepository {
	fs := afero.NewOsFs()
	root := config.Get().RootDir()
	registry := pattern.NewService()
	registry.RegisterPatterns(memory.DefaultPatterns)
	return &AdvisorRepository{
		fs:              fs,
		root:            root,
		exec:            &defaultExecutor{},
		patternRegistry: registry,
		assembler:       asm,
		extensionRepo:   extRepo,
	}
}

func (r *AdvisorRepository) Check(name string, version string, phpVersion string) (domain.AdvisorCheck, error) {
	state := determineState(r.fs, r.root, name, version, phpVersion)
	systemAvailable, systemPath, systemVersion := r.checkSystemPackage(name)

	constraint := r.getDependencyConstraint(name, phpVersion)
	shouldBuildFromSource := r.shouldBuildFromSource(name, phpVersion)
	action, url, sourceType := determineActionAndURL(state, systemAvailable, shouldBuildFromSource, r.patternRegistry, name, version, phpVersion)
	message := buildMessage(name, version, state, action)

	suggestion := ""
	if !systemAvailable {
		suggestion = getInstallSuggestion(name)
	}

	return domain.AdvisorCheck{
		Name:            name,
		Version:         version,
		PHPVersion:      phpVersion,
		State:           state,
		Action:          action,
		SystemAvailable: systemAvailable,
		SystemPath:      systemPath,
		SystemVersion:   systemVersion,
		Constraint:      constraint,
		Message:         message,
		URL:             url,
		SourceType:      sourceType,
		Suggestion:      suggestion,
	}, nil
}

func (r *AdvisorRepository) getDependencyConstraint(name, phpVersion string) string {
	if _, isBuildTool := utils.BuildTools[name]; isBuildTool {
		return r.getBuildToolConstraint(name, phpVersion)
	}

	if r.assembler == nil {
		return ""
	}

	deps, err := r.assembler.GetDependencies("php", phpVersion)
	if err != nil {
		return ""
	}

	for _, dep := range deps {
		if dep.Name != name {
			continue
		}
		return extractConstraint(dep.Version)
	}
	return ""
}

func (r *AdvisorRepository) checkSystemPackage(name string) (bool, string, string) {
	if pkgConfigName, isLib := libraryPackages[name]; isLib {
		return r.checkSystemLibrary(name, pkgConfigName)
	}
	available, path, version := r.checkSystemExecutable(name)
	if path != "" {
		return available, path, version
	}
	return false, "", ""
}

func (r *AdvisorRepository) checkSystemExecutable(name string) (bool, string, string) {
	path, err := r.exec.Which(name)
	if err != nil {
		return false, "", ""
	}
	version := r.exec.GetVersion(name)
	return true, path, version
}

func (r *AdvisorRepository) checkSystemLibrary(name, pkgConfigName string) (bool, string, string) {
	if pkgConfigNames, isMulti := multiPkgConfigPackages[name]; isMulti {
		return r.checkMultiplePkgConfig(name, pkgConfigNames)
	}
	if r.exec.PkgConfigExists(pkgConfigName) {
		version, _ := r.exec.PkgConfigModVersion(pkgConfigName)
		return true, "pkg-config:" + pkgConfigName, version
	}
	if r.checkHeaderExists(name) {
		return true, "headers:" + name, ""
	}
	return false, "", ""
}

func (r *AdvisorRepository) checkMultiplePkgConfig(name string, pkgConfigNames []string) (bool, string, string) {
	for _, pkgConfigName := range pkgConfigNames {
		if !r.exec.PkgConfigExists(pkgConfigName) {
			return false, "", ""
		}
	}
	primaryName := pkgConfigNames[0]
	version, _ := r.exec.PkgConfigModVersion(primaryName)
	return true, "pkg-config:" + strings.Join(pkgConfigNames, ","), version
}

func (r *AdvisorRepository) checkHeaderExists(name string) bool {
	headerPaths := map[string][]string{
		"libxml2":   {"/usr/include/libxml2/libxml/parser.h", "/usr/include/libxml2/libxml/xmlversion.h"},
		"openssl":   {"/usr/include/openssl/ssl.h"},
		"curl":      {"/usr/include/curl/curl.h"},
		"zlib":      {"/usr/include/zlib.h"},
		"oniguruma": {"/usr/include/oniguruma/onigmo.h", "/usr/include/oniguruma/oniguruma.h", "/usr/include/oniguruma.h"},
	}

	paths, ok := headerPaths[name]
	if !ok {
		return false
	}

	for _, path := range paths {
		if r.exec.PathExists(path) {
			return true
		}
	}
	return false
}

func determineState(fs afero.Fs, root, name, version, phpVersion string) domain.PackageState {
	cacheDir := filepath.Join(root, "cache", name, version)
	cacheExists := false
	if entries, err := afero.ReadDir(fs, cacheDir); err == nil && len(entries) > 0 {
		cacheExists = true
	}

	sourcePath := filepath.Join(root, "sources", name, version)
	sourceExists, _ := afero.Exists(fs, sourcePath)

	var versionPath string
	if name == "php" {
		versionPath = filepath.Join(root, "versions", version, "output")
	} else if _, isBuildTool := utils.BuildTools[name]; isBuildTool && phpVersion != "" {
		versionPath = filepath.Join(root, "build-tools", name, version)
	} else if phpVersion != "" {
		versionPath = filepath.Join(root, "versions", phpVersion, "dependency", name, version)
	} else {
		versionPath = filepath.Join(root, "versions", name, version)
	}

	versionExists, _ := afero.Exists(fs, versionPath)

	builtCheck := checkBuilt(name, versionPath, version, fs)

	if versionExists && builtCheck {
		return domain.StateBuilt
	}

	if versionExists && !cacheExists && !sourceExists {
		return domain.StateSourceMissingBuilt
	}

	if cacheExists && sourceExists && versionExists {
		return domain.StateBuilt
	}

	if cacheExists && !sourceExists && !versionExists {
		return domain.StateSourceDownloaded
	}

	if cacheExists && !sourceExists && versionExists {
		if builtCheck {
			return domain.StateBuilt
		}
		return domain.StateSourceDownloaded
	}

	if cacheExists && sourceExists && !versionExists {
		return domain.StateSourceExtracted
	}

	if !cacheExists && sourceExists && !versionExists {
		return domain.StateSourceExtracted
	}

	if !cacheExists && !sourceExists && !versionExists {
		return domain.StateSourceMissing
	}

	if !cacheExists && !sourceExists && versionExists {
		if builtCheck {
			return domain.StateBuilt
		}
		return domain.StateSourceMissingBuilt
	}

	return domain.StateUnknown
}

func checkBuilt(name, versionPath, version string, fs afero.Fs) bool {
	if name == "php" {
		phpBinary := filepath.Join(versionPath, "bin", "php")
		exists, _ := afero.Exists(fs, phpBinary)
		return exists
	}
	// Some deps (zlib, libxml2) only install lib/ and include/ — no bin/
	binPath := filepath.Join(versionPath, "bin")
	if exists, _ := afero.Exists(fs, binPath); exists {
		return true
	}
	libPath := filepath.Join(versionPath, "lib")
	exists, _ := afero.Exists(fs, libPath)
	return exists
}

func (r *AdvisorRepository) shouldBuildFromSource(name, phpVersion string) bool {
	if phpVersion == "" {
		return false
	}

	if _, isBuildTool := utils.BuildTools[name]; isBuildTool {
		return r.shouldBuildToolFromSource(name, phpVersion)
	}

	if r.assembler == nil {
		return false
	}

	// CRITICAL: For ICU, check version compatibility with PHP
	if name == "icu" {
		return r.shouldBuildICUFromSource(phpVersion)
	}

	deps, err := r.assembler.GetDependencies("php", phpVersion)
	if err != nil {
		return false
	}

	// CRITICAL: Check if OpenSSL requires building from source first
	// If so, we MUST also build curl and libxml2 from source to avoid OpenSSL version conflicts
	opensslNeedsSource := false
	for _, dep := range deps {
		if dep.Name == "openssl" {
			constraint := extractConstraint(dep.Version)
			if constraint == "" {
				continue
			}
			available, _, systemVersion := r.checkSystemLibrary("openssl", "openssl")
			if !available || (systemVersion != "" && !utils.MatchVersionRange(constraint, systemVersion)) {
				opensslNeedsSource = true
			}
			break
		}
	}

	// Force curl and libxml2 to build from source if OpenSSL is being built from source
	// This prevents system libraries compiled against OpenSSL 3.x from conflicting with our 1.1.1w
	if opensslNeedsSource && (name == "curl" || name == "libxml2") {
		return true
	}

	for _, dep := range deps {
		if dep.Name != name {
			continue
		}

		constraint := extractConstraint(dep.Version)
		if constraint == "" {
			return false
		}

		pkgConfigName, isLib := libraryPackages[name]
		if !isLib {
			return false
		}

		available, _, systemVersion := r.checkSystemLibrary(name, pkgConfigName)
		if !available {
			return true
		}

		if systemVersion != "" && !utils.MatchVersionRange(constraint, systemVersion) {
			return true
		}
	}

	// Fallback for extension-level deps not listed in PHP's assembler dependencies
	// (PHP >=8.2.0 has empty assembler deps; extension deps come from --ext flags).
	if pkgConfigName, isLib := libraryPackages[name]; isLib {
		available, _, systemVersion := r.checkSystemLibrary(name, pkgConfigName)
		if !available {
			return true
		}
		// Check extension-level version constraint
		if extName, ok := packageExtensionMap[name]; ok && r.extensionRepo != nil {
			if extDef, ok2 := r.extensionRepo.GetExtensionDef(extName); ok2 {
				for _, v := range extDef.Versions {
					if utils.MatchVersionRange(v.VersionRange, phpVersion) {
						if idx := strings.Index(v.Version, "|"); idx != -1 {
							constraint := v.Version[idx+1:]
							if systemVersion != "" && !utils.MatchVersionRange(constraint, systemVersion) {
								return true
							}
						}
						break
					}
				}
			}
		}
	}

	return false
}

// shouldBuildICUFromSource determines if ICU should be built from source
// based on PHP version compatibility. ICU 75+ requires C++14 which older PHP versions
// may not handle well without additional patching.
func (r *AdvisorRepository) shouldBuildICUFromSource(phpVersion string) bool {
	v := utils.ParseVersion(phpVersion)

	// For PHP <8.2, only use ICU 74 or earlier from source
	// This ensures C++11 compatibility with PHP 8.0-8.1
	if v.Major < 8 || (v.Major == 8 && v.Minor < 2) {
		_, _, systemVersion := r.checkSystemLibrary("icu", "icu-uc")
		if systemVersion != "" {
			systemV := utils.ParseVersion(systemVersion)
			// If system ICU is 75+, build from source (use ICU 74)
			if systemV.Major > icuMaxCompatMajor {
				return true
			}
			// If system ICU is too old for the constraint, build from source
			deps, err := r.assembler.GetDependencies("php", phpVersion)
			if err == nil {
				for _, dep := range deps {
					if dep.Name == "icu" {
						constraint := extractConstraint(dep.Version)
						if constraint != "" && !utils.MatchVersionRange(constraint, systemVersion) {
							return true
						}
					}
				}
			}
		}
		return true // For PHP <8.2, always build ICU from source to ensure compatibility
	}

	// For PHP >=8.2, check if system ICU meets the constraint
	deps, err := r.assembler.GetDependencies("php", phpVersion)
	if err != nil {
		return false
	}

	for _, dep := range deps {
		if dep.Name != "icu" {
			continue
		}
		constraint := extractConstraint(dep.Version)
		if constraint == "" {
			return false
		}

		available, _, systemVersion := r.checkSystemLibrary("icu", "icu-uc")
		if !available {
			return true
		}
		if systemVersion != "" && !utils.MatchVersionRange(constraint, systemVersion) {
			return true
		}
	}

	return false
}

func (r *AdvisorRepository) shouldBuildToolFromSource(name, phpVersion string) bool {
	if r.assembler == nil {
		return false
	}

	constraint := r.getBuildToolConstraint(name, phpVersion)

	if constraint == "" {
		_, path, _ := r.checkSystemExecutable(name)
		return path == ""
	}

	_, _, systemVersion := r.checkSystemExecutable(name)
	if systemVersion == "" {
		return true
	}

	return !utils.MatchVersionRange(constraint, systemVersion)
}

func (r *AdvisorRepository) getBuildToolConstraint(name, phpVersion string) string {
	if phpVersion == "" {
		return ""
	}

	if r.assembler == nil {
		return ""
	}

	constraint := r.getDirectDependencyConstraint("php", phpVersion, name)
	if constraint != "" {
		return constraint
	}

	deps, err := r.assembler.GetDependencies("php", phpVersion)
	if err != nil {
		return ""
	}

	for _, dep := range deps {
		depVersion := dep.Version
		if idx := strings.Index(dep.Version, "|"); idx != -1 {
			depVersion = dep.Version[:idx]
		}
		constraint := r.getDirectDependencyConstraint(dep.Name, depVersion, name)
		if constraint != "" {
			return constraint
		}
	}

	return ""
}

func (r *AdvisorRepository) getDirectDependencyConstraint(pkgName, pkgVersion, toolName string) string {
	deps, err := r.assembler.GetDependencies(pkgName, pkgVersion)
	if err != nil {
		return ""
	}

	for _, dep := range deps {
		if dep.Name != toolName {
			continue
		}
		return extractConstraint(dep.Version)
	}
	return ""
}

func extractConstraint(version string) string {
	idx := strings.Index(version, "|")
	if idx == -1 {
		return ""
	}
	return strings.TrimSpace(version[idx+1:])
}

func determineActionAndURL(state domain.PackageState, systemAvailable, shouldBuild bool, registry *pattern.Service, name, version, phpVersion string) (string, string, string) {
	switch state {
	case domain.StateSourceMissing:
		if systemAvailable && !shouldBuild {
			return "skip", "", domain.SourceTypeBinary
		}

		url, err := registry.BuildURLByType(name, version, domain.SourceTypeBinary)
		if err == nil {
			return "download", url, domain.SourceTypeBinary
		}

		if name == "php" {
			url, err = registry.BuildURLByType(name, version, domain.SourceTypeSource)
			if err == nil {
				return "download", url, domain.SourceTypeSource
			}
			return "unknown", "", domain.SourceTypeSource
		}

		url, err = registry.BuildURLByType(name, version, domain.SourceTypeSource)
		if err == nil {
			return "download", url, domain.SourceTypeSource
		}
		return "unknown", "", domain.SourceTypeSource
	case domain.StateSourceDownloaded:
		if systemAvailable && !shouldBuild {
			return "skip", "", domain.SourceTypeBinary
		}
		return "extract", "", domain.SourceTypeSource
	case domain.StateSourceExtracted:
		if systemAvailable && !shouldBuild {
			return "skip", "", domain.SourceTypeBinary
		}
		return "build", "", domain.SourceTypeSource
	case domain.StateSourceMissingBuilt:
		if systemAvailable && !shouldBuild {
			return "skip", "", domain.SourceTypeBinary
		}
		return "rebuild", "", domain.SourceTypeSource
	case domain.StateBuilt:
		return "skip", "", ""
	default:
		return "unknown", "", ""
	}
}

func (r *AdvisorRepository) IsCompilerAvailable(compilerType domain.CompilerType) bool {
	switch compilerType {
	case domain.CompilerTypeGCC:
		_, err := exec.LookPath("gcc")
		return err == nil
	case domain.CompilerTypeZig:
		// Check for environment variable first
		if zigPath := os.Getenv("PHPV_ZIG_PATH"); zigPath != "" {
			if _, statErr := os.Stat(zigPath); statErr == nil {
				return true
			}
		}
		// Check for zig in phpv's managed tools
		zigBinary := filepath.Join(r.root, "build-tools", "zig", "0.13.0", "zig")
		if _, statErr := os.Stat(zigBinary); statErr == nil {
			return true
		}
		// Fallback to system zig
		_, err := exec.LookPath("zig")
		return err == nil
	default:
		return false
	}
}

func (r *AdvisorRepository) GetCompilerReadiness(phpVersion string) (domain.CompilerInfo, error) {
	v := utils.ParseVersion(phpVersion)

	hasGcc := r.IsCompilerAvailable(domain.CompilerTypeGCC)
	hasZig := r.IsCompilerAvailable(domain.CompilerTypeZig)

	// make availability determines zig auto-download eligibility
	_, hasMake := exec.LookPath("make")

	result := domain.CompilerInfo{}

	if v.Major >= 5 {
		// PHP 5+: prefer gcc, fallback to zig (auto-download if available)
		if hasGcc {
			result.Type = domain.CompilerTypeGCC
			result.Name = string(domain.CompilerTypeGCC)
			result.Available = true
			result.Path, _ = exec.LookPath("gcc")
		} else {
			result.Type = domain.CompilerTypeZig
			result.Name = string(domain.CompilerTypeZig)
			if hasZig {
				result.Available = true
				result.Path, _ = exec.LookPath("zig")
			} else if hasMake == nil {
				result.Available = true
				result.AutoDownload = true
			}
		}
	} else {
		// PHP <5: only zig
		result.Type = domain.CompilerTypeZig
		result.Name = string(domain.CompilerTypeZig)
		if hasZig {
			result.Available = true
			result.Path, _ = exec.LookPath("zig")
		} else if hasMake == nil {
			result.Available = true
			result.AutoDownload = true
		}
	}

	return result, nil
}

// getInstallSuggestion returns a formatted install suggestion for the given tool
// based on the detected platform's package manager.
func getInstallSuggestion(tool string) string {
	osInfo := utils.DetectOSInfo()
	if osInfo.PkgMgr == "" {
		return ""
	}

	// Look up the package name for this tool on the detected package manager
	if pmNames, ok := packageNameByPM[osInfo.PkgMgr]; ok {
		if pkg, ok := pmNames[tool]; ok {
			return osInfo.InstallCmd + " " + pkg
		}
	}

	// Fallback to tool name itself
	return osInfo.InstallCmd + " " + tool
}

func buildMessage(name, version string, state domain.PackageState, action string) string {
	switch state {
	case domain.StateSourceMissing:
		return fmt.Sprintf("%s-%s needs to be downloaded", name, version)
	case domain.StateSourceDownloaded:
		return fmt.Sprintf("%s-%s is downloaded but not extracted", name, version)
	case domain.StateSourceExtracted:
		return fmt.Sprintf("%s-%s is extracted but not built", name, version)
	case domain.StateSourceMissingBuilt:
		return fmt.Sprintf("%s-%s is built but source archive is missing (possibly deleted)", name, version)
	case domain.StateBuilt:
		return fmt.Sprintf("%s-%s is already built", name, version)
	default:
		return fmt.Sprintf("%s-%s state is unknown", name, version)
	}
}
