package disk

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"github.com/supanadit/phpv/advisor"
	"github.com/supanadit/phpv/assembler"
	"github.com/supanadit/phpv/domain"
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

	multiPkgConfigPackages = map[string][]string{
		"icu": {"icu-uc", "icu-io", "icu-i18n"},
	}

	installSuggestions = map[string]string{
		"libxml2":   "sudo apt install libxml2-dev  # or: sudo dnf install libxml2-devel",
		"openssl":   "sudo apt install libssl-dev  # or: sudo dnf install openssl-devel",
		"curl":      "sudo apt install libcurl4-openssl-dev  # or: sudo dnf install libcurl-devel",
		"zlib":      "sudo apt install zlib1g-dev  # or: sudo dnf install zlib-devel",
		"oniguruma": "sudo apt install libonig-dev  # or: sudo dnf install oniguruma-devel",
		"icu":       "sudo apt install libicu-dev  # or: sudo dnf install libicu-devel",
		"m4":        "sudo apt install m4",
		"autoconf":  "sudo apt install autoconf",
		"automake":  "sudo apt install automake",
		"libtool":   "sudo apt install libtool",
		"perl":      "sudo apt install perl",
		"bison":     "sudo apt install bison",
		"flex":      "sudo apt install flex",
		"re2c":      "sudo apt install re2c",
		"zig":       "sudo apt install zig",
	}
)

func NewAdvisorRepository(asm assembler.AssemblerRepository) advisor.AdvisorRepository {
	fs := afero.NewOsFs()
	root := viper.GetString("PHPV_ROOT")
	registry := pattern.NewService()
	registry.RegisterPatterns(memory.DefaultPatterns)
	return &AdvisorRepository{
		fs:              fs,
		root:            root,
		exec:            &defaultExecutor{},
		patternRegistry: registry,
		assembler:       asm,
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
		if sug, ok := installSuggestions[name]; ok {
			suggestion = sug
		}
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
	if _, isBuildTool := buildTools[name]; isBuildTool {
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
		"oniguruma": {"/usr/include/oniguruma/onigmo.h", "/usr/include/oniguruma/oniguruma.h"},
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
	} else if _, isBuildTool := buildTools[name]; isBuildTool && phpVersion != "" {
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

	if cacheExists && sourceExists && !versionExists {
		return domain.StateSourceExtracted
	}

	if !cacheExists && !sourceExists && !versionExists {
		return domain.StateSourceMissing
	}

	return domain.StateUnknown
}

func checkBuilt(name, versionPath, version string, fs afero.Fs) bool {
	if name == "php" {
		phpBinary := filepath.Join(versionPath, "bin", "php")
		exists, _ := afero.Exists(fs, phpBinary)
		return exists
	}
	binPath := filepath.Join(versionPath, "bin")
	exists, _ := afero.Exists(fs, binPath)
	return exists
}

func (r *AdvisorRepository) shouldBuildFromSource(name, phpVersion string) bool {
	if phpVersion == "" {
		return false
	}

	if _, isBuildTool := buildTools[name]; isBuildTool {
		return r.shouldBuildToolFromSource(name, phpVersion)
	}

	if r.assembler == nil {
		return false
	}

	deps, err := r.assembler.GetDependencies("php", phpVersion)
	if err != nil {
		return false
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

		_, _, systemVersion := r.checkSystemLibrary(name, pkgConfigName)
		if systemVersion == "" {
			return true
		}

		if !utils.MatchVersionRange(constraint, systemVersion) {
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
		return "extract", "", domain.SourceTypeSource
	case domain.StateSourceExtracted:
		return "build", "", domain.SourceTypeSource
	case domain.StateSourceMissingBuilt:
		return "rebuild", "", domain.SourceTypeSource
	case domain.StateBuilt:
		return "skip", "", ""
	default:
		return "unknown", "", ""
	}
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
