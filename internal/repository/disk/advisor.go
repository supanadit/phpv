package disk

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"github.com/supanadit/phpv/advisor"
	"github.com/supanadit/phpv/assembler"
	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/internal/utils"
	"github.com/supanadit/phpv/pattern"
)

type defaultExecutor struct{}

func (e *defaultExecutor) Which(cmd string) (string, error) {
	out, err := exec.Command("which", cmd).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func (e *defaultExecutor) PkgConfig(pkg string) (string, string, error) {
	out, err := exec.Command("pkg-config", "--libs", "--cflags", pkg).Output()
	if err != nil {
		return "", "", err
	}
	parts := strings.SplitN(strings.TrimSpace(string(out)), " ", 2)
	cflags := ""
	ldflags := ""
	if len(parts) >= 1 {
		ldflags = parts[0]
	}
	if len(parts) >= 2 {
		cflags = parts[1]
	}
	return cflags, ldflags, nil
}

func (e *defaultExecutor) PkgConfigModVersion(pkg string) (string, error) {
	out, err := exec.Command("pkg-config", "--modversion", pkg).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func (e *defaultExecutor) PkgConfigExists(pkg string) bool {
	env := os.Environ()
	pkgConfigPath := os.Getenv("PKG_CONFIG_PATH")
	standardPaths := utils.GetSystemPkgConfigPaths()
	for _, p := range standardPaths {
		if pkgConfigPath == "" {
			pkgConfigPath = p
		} else {
			pkgConfigPath = p + ":" + pkgConfigPath
		}
	}
	for i, v := range env {
		if strings.HasPrefix(v, "PKG_CONFIG_PATH=") {
			env[i] = "PKG_CONFIG_PATH=" + pkgConfigPath
			break
		}
	}
	if !strings.Contains(strings.Join(env, ""), "PKG_CONFIG_PATH") {
		env = append(env, "PKG_CONFIG_PATH="+pkgConfigPath)
	}
	cmd := exec.Command("pkg-config", "--exists", pkg)
	cmd.Env = env
	return cmd.Run() == nil
}

var buildToolVersionParsers = map[string]func(string) string{
	"m4":       parseM4Version,
	"autoconf": parseAutoconfVersion,
	"automake": parseAutomakeVersion,
	"bison":    parseBisonVersion,
	"flex":     parseFlexVersion,
	"libtool":  parseLibtoolVersion,
	"perl":     parsePerlVersion,
	"re2c":     parseRe2cVersion,
	"zig":      parseZigVersion,
}

func parseM4Version(output string) string {
	re := regexp.MustCompile(`\(GNU M4\) (\d+\.\d+(?:\.\d+)?)`)
	matches := re.FindStringSubmatch(output)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

func parseAutoconfVersion(output string) string {
	re := regexp.MustCompile(`\(GNU Autoconf\) (\d+\.\d+(?:\.\d+)?)`)
	matches := re.FindStringSubmatch(output)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

func parseAutomakeVersion(output string) string {
	re := regexp.MustCompile(`\(GNU Automake\) (\d+\.\d+(?:\.\d+)?)`)
	matches := re.FindStringSubmatch(output)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

func parseBisonVersion(output string) string {
	re := regexp.MustCompile(`\(GNU Bison\) (\d+\.\d+(?:\.\d+)?)`)
	matches := re.FindStringSubmatch(output)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

func parseFlexVersion(output string) string {
	re := regexp.MustCompile(`flex (\d+\.\d+(?:\.\d+)?)`)
	matches := re.FindStringSubmatch(output)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

func parseLibtoolVersion(output string) string {
	re := regexp.MustCompile(`\(GNU libtool\) (\d+\.\d+(?:\.\d+)?)`)
	matches := re.FindStringSubmatch(output)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

func parsePerlVersion(output string) string {
	re := regexp.MustCompile(`This is perl 5, version (\d+)`)
	matches := re.FindStringSubmatch(output)
	if len(matches) >= 2 {
		minor := matches[1]
		return "5." + minor
	}
	re2 := regexp.MustCompile(`v?(\d+\.\d+\.\d+)`)
	matches2 := re2.FindStringSubmatch(output)
	if len(matches2) >= 2 {
		return matches2[1]
	}
	return ""
}

func parseRe2cVersion(output string) string {
	parts := strings.Fields(output)
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}

func parseZigVersion(output string) string {
	parts := strings.Fields(output)
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}

func (e *defaultExecutor) GetVersion(name string) string {
	cmd := exec.Command(name, "--version")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	parser := buildToolVersionParsers[name]
	if parser == nil {
		return ""
	}
	return parser(strings.TrimSpace(string(out)))
}

type AdvisorRepository struct {
	fs              afero.Fs
	root            string
	exec            *defaultExecutor
	patternRegistry *pattern.PatternRegistry
	assembler       assembler.AssemblerRepository
}

var (
	libraryPackages = map[string]string{
		"libxml2":   "libxml-2.0",
		"openssl":   "openssl",
		"curl":      "libcurl",
		"zlib":      "zlib",
		"oniguruma": "oniguruma",
	}

	installSuggestions = map[string]string{
		"libxml2":   "sudo apt install libxml2-dev  # or: sudo dnf install libxml2-devel",
		"openssl":   "sudo apt install libssl-dev  # or: sudo dnf install openssl-devel",
		"curl":      "sudo apt install libcurl4-openssl-dev  # or: sudo dnf install libcurl-devel",
		"zlib":      "sudo apt install zlib1g-dev  # or: sudo dnf install zlib-devel",
		"oniguruma": "sudo apt install libonig-dev  # or: sudo dnf install oniguruma-devel",
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
	registry := pattern.NewPatternRegistry()
	registry.RegisterPatterns(pattern.DefaultURLPatterns)
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
		Message:         message,
		URL:             url,
		SourceType:      sourceType,
		Suggestion:      suggestion,
	}, nil
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
	if r.exec.PkgConfigExists(pkgConfigName) {
		version, _ := r.exec.PkgConfigModVersion(pkgConfigName)
		return true, "pkg-config:" + pkgConfigName, version
	}
	if r.checkHeaderExists(name) {
		return true, "headers:" + name, ""
	}
	return false, "", ""
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

func (e *defaultExecutor) PathExists(path string) bool {
	_, err := exec.Command("test", "-f", path).CombinedOutput()
	return err == nil
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

func mustBuildFromSource(name, phpVersion string) bool {
	if phpVersion == "" {
		return false
	}
	v := utils.ParseVersion(phpVersion)

	switch name {
	case "openssl":
		if v.Major < 8 {
			return true
		}
		if v.Major == 8 && v.Minor == 0 {
			return true
		}
		if v.Major == 8 && v.Minor == 1 && v.Patch < 33 {
			return true
		}
	case "libxml2":
		if v.Major < 8 {
			return true
		}
	case "curl":
		if v.Major < 8 {
			return true
		}
	}
	return false
}

func (r *AdvisorRepository) shouldBuildFromSource(name, phpVersion string) bool {
	if phpVersion == "" {
		return false
	}

	if _, isBuildTool := buildTools[name]; isBuildTool {
		return r.shouldBuildToolFromSource(name, phpVersion)
	}

	if r.assembler == nil {
		return mustBuildFromSource(name, phpVersion)
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

func extractConstraint(version string) string {
	idx := strings.Index(version, "|")
	if idx == -1 {
		return ""
	}
	return strings.TrimSpace(version[idx+1:])
}

func determineActionAndURL(state domain.PackageState, systemAvailable, shouldBuild bool, registry *pattern.PatternRegistry, name, version, phpVersion string) (string, string, string) {
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
