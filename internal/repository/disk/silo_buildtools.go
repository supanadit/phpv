package disk

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
	"github.com/supanadit/phpv/internal/utils"
)

func (r *SiloRepository) getBuildToolsRefsFilePath() string {
	return filepath.Join(r.silo.Root, ".build-tools-refs.json")
}

func (r *SiloRepository) loadBuildToolsRefs() (map[string][]string, error) {
	refsFile := r.getBuildToolsRefsFilePath()
	data, err := afero.ReadFile(r.fs, refsFile)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string][]string), nil
		}
		return nil, fmt.Errorf("failed to read build-tools refs: %w", err)
	}

	var refs map[string][]string
	if err := json.Unmarshal(data, &refs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal build-tools refs: %w", err)
	}

	return refs, nil
}

func (r *SiloRepository) saveBuildToolsRefs(refs map[string][]string) error {
	refsFile := r.getBuildToolsRefsFilePath()
	data, err := json.MarshalIndent(refs, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal build-tools refs: %w", err)
	}

	if err := afero.WriteFile(r.fs, refsFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write build-tools refs: %w", err)
	}

	return nil
}

func (r *SiloRepository) IncrementBuildToolRef(name, version, phpVersion string) error {
	r.buildToolsMutex.Lock()
	defer r.buildToolsMutex.Unlock()

	refs, err := r.loadBuildToolsRefs()
	if err != nil {
		return err
	}

	key := name + "@" + version
	refs[key] = append(refs[key], phpVersion)

	return r.saveBuildToolsRefs(refs)
}

func (r *SiloRepository) DecrementBuildToolRef(name, version, phpVersion string) error {
	r.buildToolsMutex.Lock()
	defer r.buildToolsMutex.Unlock()

	refs, err := r.loadBuildToolsRefs()
	if err != nil {
		return err
	}

	key := name + "@" + version
	versions, exists := refs[key]
	if !exists {
		return nil
	}

	var newVersions []string
	for _, v := range versions {
		if v != phpVersion {
			newVersions = append(newVersions, v)
		}
	}

	if len(newVersions) == 0 {
		delete(refs, key)
	} else {
		refs[key] = newVersions
	}

	return r.saveBuildToolsRefs(refs)
}

func (r *SiloRepository) GetBuildToolRefs() (map[string][]string, error) {
	return r.loadBuildToolsRefs()
}

func (r *SiloRepository) RemoveBuildToolRef(name, version string) error {
	r.buildToolsMutex.Lock()
	defer r.buildToolsMutex.Unlock()

	refs, err := r.loadBuildToolsRefs()
	if err != nil {
		return err
	}

	key := name + "@" + version
	delete(refs, key)

	return r.saveBuildToolsRefs(refs)
}

func (r *SiloRepository) RemovePHPInstallation(phpVersion string) ([]string, error) {
	deps, err := r.GetDependencyInfo(phpVersion)
	if err != nil {
		return nil, fmt.Errorf("[bundler] failed to read dependency info: %w", err)
	}

	var removedTools []string
	var builtFromSource []string
	for _, dep := range deps {
		if dep.BuiltFromSource {
			builtFromSource = append(builtFromSource, dep.Name+"@"+dep.Version)
		}
	}

	refs, err := r.loadBuildToolsRefs()
	if err != nil {
		return nil, fmt.Errorf("failed to load build-tools refs: %w", err)
	}

	for key, phpVersions := range refs {
		var newVersions []string
		for _, v := range phpVersions {
			if v != phpVersion {
				newVersions = append(newVersions, v)
			}
		}
		if len(newVersions) == 0 {
			parts := strings.Split(key, "@")
			if len(parts) == 2 {
				name, version := parts[0], parts[1]
				toolPath := filepath.Join(r.silo.Root, "build-tools", name, version)
				if exists, _ := afero.Exists(r.fs, toolPath); exists {
					if err := r.fs.RemoveAll(toolPath); err != nil {
						return nil, fmt.Errorf("failed to remove build-tool %s: %w", key, err)
					}
					removedTools = append(removedTools, key)
				}
			}
			delete(refs, key)
		} else {
			refs[key] = newVersions
		}
	}

	if err := r.saveBuildToolsRefs(refs); err != nil {
		return nil, fmt.Errorf("failed to save build-tools refs: %w", err)
	}

	versionPath := utils.PHPVersionPath(r.silo, phpVersion)
	if exists, _ := afero.Exists(r.fs, versionPath); exists {
		if err := r.fs.RemoveAll(versionPath); err != nil {
			return nil, fmt.Errorf("failed to remove version directory: %w", err)
		}
	}

	depInfo := r.getDepsInfoFilePath(phpVersion)
	if exists, _ := afero.Exists(r.fs, depInfo); exists {
		if err := r.fs.Remove(depInfo); err != nil {
			return nil, fmt.Errorf("failed to remove dependency info: %w", err)
		}
	}

	defaultPath := filepath.Join(r.silo.Root, "default")
	if data, err := afero.ReadFile(r.fs, defaultPath); err == nil {
		if strings.TrimSpace(string(data)) == phpVersion {
			if err := afero.WriteFile(r.fs, defaultPath, []byte(""), 0644); err != nil {
				return nil, fmt.Errorf("failed to clear default: %w", err)
			}
		}
	}

	_ = builtFromSource

	return removedTools, nil
}
