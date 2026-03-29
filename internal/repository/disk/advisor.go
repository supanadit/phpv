package disk

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"github.com/supanadit/phpv/advisor"
	"github.com/supanadit/phpv/domain"
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

func (e *defaultExecutor) PkgConfigExists(pkg string) bool {
	err := exec.Command("pkg-config", "--exists", pkg).Run()
	return err == nil
}

type AdvisorRepository struct {
	fs              afero.Fs
	root            string
	exec            *defaultExecutor
	patternRegistry *pattern.PatternRegistry
}

var (
	libraryPackages = map[string]string{
		"libxml2":   "libxml-2.0",
		"openssl":   "openssl",
		"curl":      "libcurl",
		"zlib":      "zlib",
		"oniguruma": "oniguruma",
	}
)

func NewAdvisorRepository() advisor.AdvisorRepository {
	fs := afero.NewOsFs()
	root := viper.GetString("PHPV_ROOT")
	registry := pattern.NewPatternRegistry()
	registry.RegisterPatterns(pattern.DefaultURLPatterns)
	return &AdvisorRepository{
		fs:              fs,
		root:            root,
		exec:            &defaultExecutor{},
		patternRegistry: registry,
	}
}

func (r *AdvisorRepository) Check(name string, version string) (domain.AdvisorCheck, error) {
	state := determineState(r.fs, r.root, name, version)
	systemAvailable, systemPath := r.checkSystemPackage(name)
	action, url, sourceType := determineActionAndURL(state, systemAvailable, r.patternRegistry, name, version)
	message := buildMessage(name, version, state, action)

	return domain.AdvisorCheck{
		Name:            name,
		Version:         version,
		State:           state,
		Action:          action,
		SystemAvailable: systemAvailable,
		SystemPath:      systemPath,
		Message:         message,
		URL:             url,
		SourceType:      sourceType,
	}, nil
}

func (r *AdvisorRepository) checkSystemPackage(name string) (bool, string) {
	if pkgConfigName, isLib := libraryPackages[name]; isLib {
		return r.checkSystemLibrary(name, pkgConfigName)
	}
	return r.checkSystemExecutable(name)
}

func (r *AdvisorRepository) checkSystemExecutable(name string) (bool, string) {
	path, err := r.exec.Which(name)
	if err != nil {
		return false, ""
	}
	return true, path
}

func (r *AdvisorRepository) checkSystemLibrary(name, pkgConfigName string) (bool, string) {
	if r.exec.PkgConfigExists(pkgConfigName) {
		return true, "pkg-config:" + pkgConfigName
	}
	if r.checkHeaderExists(name) {
		return true, "headers:" + name
	}
	return false, ""
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

func determineState(fs afero.Fs, root, name, version string) domain.PackageState {
	cacheDir := filepath.Join(root, "cache", name, version)
	cacheExists := false
	if entries, err := afero.ReadDir(fs, cacheDir); err == nil && len(entries) > 0 {
		cacheExists = true
	}
	sourcePath := filepath.Join(root, "sources", name, version)
	versionPath := filepath.Join(root, "versions", name, version)

	sourceExists, _ := afero.Exists(fs, sourcePath)
	versionExists, _ := afero.Exists(fs, versionPath)

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

func determineActionAndURL(state domain.PackageState, systemAvailable bool, registry *pattern.PatternRegistry, name, version string) (string, string, string) {
	switch state {
	case domain.StateSourceMissing:
		if systemAvailable {
			return "skip", "", domain.SourceTypeBinary
		}
		url, err := registry.BuildURLByType(name, version, domain.SourceTypeBinary)
		if err == nil {
			return "download", url, domain.SourceTypeBinary
		}
		url, err = registry.BuildURLByType(name, version, domain.SourceTypeSource)
		if err == nil {
			return "download", url, domain.SourceTypeSource
		}
		return "unknown", "", ""
	case domain.StateSourceDownloaded:
		return "extract", "", ""
	case domain.StateSourceExtracted:
		return "build", "", ""
	case domain.StateSourceMissingBuilt:
		return "rebuild", "", ""
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
