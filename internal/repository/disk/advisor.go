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
)

type defaultExecutor struct{}

func (e *defaultExecutor) Which(cmd string) (string, error) {
	out, err := exec.Command("which", cmd).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

type AdvisorRepository struct {
	fs   afero.Fs
	root string
	exec *defaultExecutor
}

func NewAdvisorRepository() advisor.AdvisorRepository {
	fs := afero.NewOsFs()
	root := viper.GetString("PHPV_ROOT")
	return &AdvisorRepository{
		fs:   fs,
		root: root,
		exec: &defaultExecutor{},
	}
}

func (r *AdvisorRepository) Check(name string, version string) (domain.AdvisorCheck, error) {
	state := determineState(r.fs, r.root, name, version)
	action := determineAction(state)
	systemAvailable, systemPath := r.checkSystemPackage(name)
	message := buildMessage(name, version, state, action)

	return domain.AdvisorCheck{
		Name:            name,
		Version:         version,
		State:           state,
		Action:          action,
		SystemAvailable: systemAvailable,
		SystemPath:      systemPath,
		Message:         message,
	}, nil
}

func (r *AdvisorRepository) checkSystemPackage(name string) (bool, string) {
	path, err := r.exec.Which(name)
	if err != nil {
		return false, ""
	}
	return true, path
}

func determineState(fs afero.Fs, root, name, version string) domain.PackageState {
	cachePath := filepath.Join(root, "cache", fmt.Sprintf("%s-%s.tar.gz", name, version))
	sourcePath := filepath.Join(root, "sources", name, version)
	versionPath := filepath.Join(root, "versions", name, version)

	cacheExists, _ := afero.Exists(fs, cachePath)
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

func determineAction(state domain.PackageState) string {
	switch state {
	case domain.StateSourceMissing:
		return "download"
	case domain.StateSourceDownloaded:
		return "extract"
	case domain.StateSourceExtracted:
		return "build"
	case domain.StateSourceMissingBuilt:
		return "rebuild"
	case domain.StateBuilt:
		return "skip"
	default:
		return "unknown"
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
