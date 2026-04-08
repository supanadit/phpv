package disk

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/spf13/afero"
)

type DirectoryVersion struct {
	Path    string `json:"path"`
	Version string `json:"version"`
}

type DirectoryRepository struct {
	fs        afero.Fs
	siloPath  string
	dataPath  string
	directory map[string]string
}

func NewDirectoryRepository(siloPath string) (*DirectoryRepository, error) {
	repo := &DirectoryRepository{
		fs:       afero.NewOsFs(),
		siloPath: siloPath,
		dataPath: filepath.Join(siloPath, ".directory_versions"),
	}

	if err := repo.load(); err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
	}

	return repo, nil
}

func (r *DirectoryRepository) load() error {
	r.directory = make(map[string]string)

	data, err := afero.ReadFile(r.fs, r.dataPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var versions []DirectoryVersion
	if err := json.Unmarshal(data, &versions); err != nil {
		return err
	}

	for _, v := range versions {
		r.directory[v.Path] = v.Version
	}

	return nil
}

func (r *DirectoryRepository) save() error {
	var versions []DirectoryVersion
	for path, version := range r.directory {
		versions = append(versions, DirectoryVersion{Path: path, Version: version})
	}

	data, err := json.MarshalIndent(versions, "", "  ")
	if err != nil {
		return err
	}

	return afero.WriteFile(r.fs, r.dataPath, data, 0644)
}

func (r *DirectoryRepository) SetVersion(path, version string) error {
	r.directory[path] = version
	return r.save()
}

func (r *DirectoryRepository) GetVersion(path string) (string, bool) {
	version, ok := r.directory[path]
	return version, ok
}

func (r *DirectoryRepository) RemoveVersion(path string) error {
	delete(r.directory, path)
	return r.save()
}

func (r *DirectoryRepository) List() map[string]string {
	result := make(map[string]string)
	for k, v := range r.directory {
		result[k] = v
	}
	return result
}

func (r *DirectoryRepository) Cleanup() error {
	var toRemove []string
	for path := range r.directory {
		if exists, _ := afero.Exists(r.fs, path); !exists {
			toRemove = append(toRemove, path)
		}
	}

	for _, path := range toRemove {
		delete(r.directory, path)
	}

	if len(toRemove) > 0 {
		return r.save()
	}

	return nil
}
