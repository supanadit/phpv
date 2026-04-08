package disk

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/internal/utils"
)

var (
	ErrNotFound     = errors.New("item not found")
	ErrExists       = errors.New("item already exists")
	ErrInvalidInput = errors.New("invalid input")
)

type SiloRepository struct {
	fs              afero.Fs
	silo            *domain.Silo
	buildToolsMutex sync.Mutex
}

func NewSiloRepository() (*SiloRepository, error) {
	root := viper.GetString("PHPV_ROOT")
	if root == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get user home directory: %w", err)
		}
		root = filepath.Join(homeDir, ".phpv")
	}

	return &SiloRepository{
		fs:   afero.NewOsFs(),
		silo: &domain.Silo{Root: root},
	}, nil
}

func (r *SiloRepository) GetSilo() (*domain.Silo, error) {
	return r.silo, nil
}

func (r *SiloRepository) EnsurePaths() error {
	paths := []string{
		utils.CachePath(r.silo),
		utils.SourcePath(r.silo),
		utils.VersionPath(r.silo),
		utils.BinPath(r.silo),
	}

	for _, path := range paths {
		if err := r.fs.MkdirAll(path, 0o755); err != nil {
			return fmt.Errorf("failed to create path %s: %w", path, err)
		}
	}

	return nil
}

func (r *SiloRepository) validateInput(pkg, ver string) error {
	if pkg == "" {
		return fmt.Errorf("package name cannot be empty: %w", ErrInvalidInput)
	}
	if ver == "" {
		return fmt.Errorf("version cannot be empty: %w", ErrInvalidInput)
	}
	return nil
}

func (r *SiloRepository) getSourceFilePath(pkg, ver string) string {
	return filepath.Join(utils.GetSourcePath(r.silo, pkg, ver), "source.tar.gz")
}

func (r *SiloRepository) getVersionFilePath(pkg, ver string) string {
	return filepath.Join(utils.GetVersionPath(r.silo, pkg, ver), "version.tar.gz")
}

func (r *SiloRepository) ArchiveExists(pkg, ver string) bool {
	if err := r.validateInput(pkg, ver); err != nil {
		return false
	}
	path := utils.GetArchivePath(r.silo, pkg, ver)
	exists, _ := afero.Exists(r.fs, path)
	return exists
}

func (r *SiloRepository) GetArchivePath(pkg, ver string) string {
	return utils.GetArchivePath(r.silo, pkg, ver)
}

func (r *SiloRepository) StoreArchive(pkg, ver string, data io.Reader) error {
	if err := r.validateInput(pkg, ver); err != nil {
		return err
	}

	path := utils.GetArchivePath(r.silo, pkg, ver)
	dir := filepath.Dir(path)

	if err := r.fs.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	file, err := r.fs.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", path, err)
	}
	defer file.Close()

	if _, err := io.Copy(file, data); err != nil {
		return fmt.Errorf("failed to write archive: %w", err)
	}

	return nil
}

func (r *SiloRepository) RetrieveArchive(pkg, ver string) (io.ReadCloser, error) {
	if err := r.validateInput(pkg, ver); err != nil {
		return nil, err
	}

	path := utils.GetArchivePath(r.silo, pkg, ver)
	if exists, _ := afero.Exists(r.fs, path); !exists {
		return nil, fmt.Errorf("archive not found: %w", ErrNotFound)
	}

	return r.fs.Open(path)
}

func (r *SiloRepository) RemoveArchive(pkg, ver string) error {
	if err := r.validateInput(pkg, ver); err != nil {
		return err
	}

	path := utils.GetArchivePath(r.silo, pkg, ver)
	if exists, _ := afero.Exists(r.fs, path); !exists {
		return nil
	}

	return r.fs.Remove(path)
}

func (r *SiloRepository) ListArchives() []string {
	return r.listItems(utils.CachePath(r.silo))
}

func (r *SiloRepository) SourceExists(pkg, ver string) bool {
	if err := r.validateInput(pkg, ver); err != nil {
		return false
	}
	path := r.getSourceFilePath(pkg, ver)
	exists, _ := afero.Exists(r.fs, path)
	return exists
}

func (r *SiloRepository) GetSourcePath(pkg, ver string) string {
	return utils.GetSourcePath(r.silo, pkg, ver)
}

func (r *SiloRepository) StoreSource(pkg, ver string, data io.Reader) error {
	if err := r.validateInput(pkg, ver); err != nil {
		return err
	}

	path := utils.GetSourcePath(r.silo, pkg, ver)

	if err := r.fs.MkdirAll(path, 0o755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", path, err)
	}

	destPath := r.getSourceFilePath(pkg, ver)
	file, err := r.fs.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", destPath, err)
	}
	defer file.Close()

	if _, err := io.Copy(file, data); err != nil {
		return fmt.Errorf("failed to write source: %w", err)
	}

	return nil
}

func (r *SiloRepository) RetrieveSource(pkg, ver string) (io.ReadCloser, error) {
	if err := r.validateInput(pkg, ver); err != nil {
		return nil, err
	}

	path := r.getSourceFilePath(pkg, ver)
	if exists, _ := afero.Exists(r.fs, path); !exists {
		return nil, fmt.Errorf("source not found: %w", ErrNotFound)
	}

	return r.fs.Open(path)
}

func (r *SiloRepository) RemoveSource(pkg, ver string) error {
	if err := r.validateInput(pkg, ver); err != nil {
		return err
	}

	path := utils.GetSourcePath(r.silo, pkg, ver)
	if exists, _ := afero.Exists(r.fs, path); !exists {
		return nil
	}

	return r.fs.RemoveAll(path)
}

func (r *SiloRepository) ListSources() []string {
	return r.listItems(utils.SourcePath(r.silo))
}

func (r *SiloRepository) VersionExists(pkg, ver string) bool {
	if err := r.validateInput(pkg, ver); err != nil {
		return false
	}
	path := r.getVersionFilePath(pkg, ver)
	exists, _ := afero.Exists(r.fs, path)
	return exists
}

func (r *SiloRepository) GetVersionPath(pkg, ver string) string {
	return utils.GetVersionPath(r.silo, pkg, ver)
}

func (r *SiloRepository) StoreVersion(pkg, ver string, data io.Reader) error {
	if err := r.validateInput(pkg, ver); err != nil {
		return err
	}

	path := utils.GetVersionPath(r.silo, pkg, ver)

	if err := r.fs.MkdirAll(path, 0o755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", path, err)
	}

	destPath := r.getVersionFilePath(pkg, ver)
	file, err := r.fs.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", destPath, err)
	}
	defer file.Close()

	if _, err := io.Copy(file, data); err != nil {
		return fmt.Errorf("failed to write version: %w", err)
	}

	return nil
}

func (r *SiloRepository) RetrieveVersion(pkg, ver string) (io.ReadCloser, error) {
	if err := r.validateInput(pkg, ver); err != nil {
		return nil, err
	}

	path := r.getVersionFilePath(pkg, ver)
	if exists, _ := afero.Exists(r.fs, path); !exists {
		return nil, fmt.Errorf("version not found: %w", ErrNotFound)
	}

	return r.fs.Open(path)
}

func (r *SiloRepository) RemoveVersion(pkg, ver string) error {
	if err := r.validateInput(pkg, ver); err != nil {
		return err
	}

	path := utils.GetVersionPath(r.silo, pkg, ver)
	if exists, _ := afero.Exists(r.fs, path); !exists {
		return nil
	}

	return r.fs.RemoveAll(path)
}

func (r *SiloRepository) ListVersions() []string {
	return r.listItems(utils.VersionPath(r.silo))
}

func (r *SiloRepository) GetDefault() (string, error) {
	defaultPath := filepath.Join(r.silo.Root, "default")
	data, err := afero.ReadFile(r.fs, defaultPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("failed to read default file: %w", err)
	}
	return strings.TrimSpace(string(data)), nil
}

func (r *SiloRepository) SetDefault(version string) error {
	defaultPath := filepath.Join(r.silo.Root, "default")
	if err := afero.WriteFile(r.fs, defaultPath, []byte(version), 0644); err != nil {
		return fmt.Errorf("failed to write default file: %w", err)
	}
	return nil
}

func (r *SiloRepository) FullClean(pkg, ver string) error {
	if err := r.validateInput(pkg, ver); err != nil {
		return err
	}

	if err := r.RemoveArchive(pkg, ver); err != nil {
		return err
	}
	if err := r.RemoveSource(pkg, ver); err != nil {
		return err
	}
	if err := r.RemoveVersion(pkg, ver); err != nil {
		return err
	}

	return nil
}

func (r *SiloRepository) CleanAll() error {
	paths := []string{
		utils.CachePath(r.silo),
		utils.SourcePath(r.silo),
		utils.VersionPath(r.silo),
	}

	for _, path := range paths {
		if exists, _ := afero.Exists(r.fs, path); exists {
			if err := r.fs.RemoveAll(path); err != nil {
				return fmt.Errorf("failed to clean %s: %w", path, err)
			}
		}
	}

	return nil
}

func (r *SiloRepository) listItems(basePath string) []string {
	var items []string

	entries, err := afero.ReadDir(r.fs, basePath)
	if err != nil {
		return items
	}

	for _, entry := range entries {
		if entry.IsDir() {
			items = append(items, entry.Name())
		}
	}

	return items
}

func (r *SiloRepository) getStateFilePath(phpVersion string) string {
	return filepath.Join(utils.PHPVersionPath(r.silo, phpVersion), ".state")
}

func (r *SiloRepository) MarkInProgress(phpVersion string) error {
	versionPath := utils.PHPVersionPath(r.silo, phpVersion)
	if err := r.fs.MkdirAll(versionPath, 0o755); err != nil {
		return fmt.Errorf("failed to create version directory: %w", err)
	}

	statePath := r.getStateFilePath(phpVersion)
	if err := afero.WriteFile(r.fs, statePath, []byte("in_progress"), 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

func (r *SiloRepository) MarkComplete(phpVersion string) error {
	statePath := r.getStateFilePath(phpVersion)
	if exists, _ := afero.Exists(r.fs, statePath); !exists {
		return nil
	}

	if err := afero.WriteFile(r.fs, statePath, []byte("installed"), 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

func (r *SiloRepository) MarkFailed(phpVersion string) error {
	statePath := r.getStateFilePath(phpVersion)
	if exists, _ := afero.Exists(r.fs, statePath); !exists {
		return nil
	}

	if err := afero.WriteFile(r.fs, statePath, []byte("failed"), 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

func (r *SiloRepository) GetState(phpVersion string) (domain.InstallState, error) {
	statePath := r.getStateFilePath(phpVersion)
	data, err := afero.ReadFile(r.fs, statePath)
	if err != nil {
		if os.IsNotExist(err) {
			return domain.StateNone, nil
		}
		return domain.StateNone, fmt.Errorf("failed to read state file: %w", err)
	}

	state := strings.TrimSpace(string(data))
	switch state {
	case "in_progress":
		return domain.StateInProgress, nil
	case "installed":
		return domain.StateInstalled, nil
	case "failed":
		return domain.StateFailed, nil
	default:
		return domain.StateNone, nil
	}
}

func (r *SiloRepository) Rollback(phpVersion string) error {
	versionPath := utils.PHPVersionPath(r.silo, phpVersion)

	if exists, _ := afero.Exists(r.fs, versionPath); exists {
		if err := r.fs.RemoveAll(versionPath); err != nil {
			return fmt.Errorf("failed to remove version directory: %w", err)
		}
	}

	depInfo := r.getDepsInfoFilePath(phpVersion)
	if exists, _ := afero.Exists(r.fs, depInfo); exists {
		if err := r.fs.Remove(depInfo); err != nil {
			return fmt.Errorf("failed to remove dependency info: %w", err)
		}
	}

	return nil
}

func (r *SiloRepository) getDepsInfoFilePath(phpVersion string) string {
	return filepath.Join(utils.PHPVersionPath(r.silo, phpVersion), ".deps.json")
}

func (r *SiloRepository) SaveDependencyInfo(phpVersion string, deps []domain.DependencyInfo) error {
	versionPath := utils.PHPVersionPath(r.silo, phpVersion)
	if err := r.fs.MkdirAll(versionPath, 0o755); err != nil {
		return fmt.Errorf("failed to create version directory: %w", err)
	}

	depInfoPath := r.getDepsInfoFilePath(phpVersion)
	data, err := json.MarshalIndent(deps, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal dependency info: %w", err)
	}

	if err := afero.WriteFile(r.fs, depInfoPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write dependency info: %w", err)
	}

	return nil
}

func (r *SiloRepository) GetDependencyInfo(phpVersion string) ([]domain.DependencyInfo, error) {
	depInfoPath := r.getDepsInfoFilePath(phpVersion)
	data, err := afero.ReadFile(r.fs, depInfoPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []domain.DependencyInfo{}, nil
		}
		return nil, fmt.Errorf("failed to read dependency info: %w", err)
	}

	var deps []domain.DependencyInfo
	if err := json.Unmarshal(data, &deps); err != nil {
		return nil, fmt.Errorf("failed to unmarshal dependency info: %w", err)
	}

	return deps, nil
}

func (r *SiloRepository) RemoveDependencyInfo(phpVersion string) error {
	depInfoPath := r.getDepsInfoFilePath(phpVersion)
	if exists, _ := afero.Exists(r.fs, depInfoPath); !exists {
		return nil
	}

	if err := r.fs.Remove(depInfoPath); err != nil {
		return fmt.Errorf("failed to remove dependency info: %w", err)
	}

	return nil
}

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

func (r *SiloRepository) GetInstalledBuildTools() ([]string, error) {
	buildToolsPath := filepath.Join(r.silo.Root, "build-tools")
	entries, err := afero.ReadDir(r.fs, buildToolsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read build-tools directory: %w", err)
	}

	var tools []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pkgPath := filepath.Join(buildToolsPath, entry.Name())
		versionEntries, err := afero.ReadDir(r.fs, pkgPath)
		if err != nil {
			continue
		}
		for _, vEntry := range versionEntries {
			if !vEntry.IsDir() {
				continue
			}
			tools = append(tools, entry.Name()+"@"+vEntry.Name())
		}
	}

	return tools, nil
}

func (r *SiloRepository) RemoveUnusedBuildTools(dryRun bool) ([]string, []string, error) {
	refs, err := r.loadBuildToolsRefs()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load build-tools refs: %w", err)
	}

	trackedTools := make(map[string]bool)
	for key := range refs {
		trackedTools[key] = true
	}

	installedTools, err := r.GetInstalledBuildTools()
	if err != nil {
		return nil, nil, err
	}

	var removed []string
	var wouldRemove []string

	for _, tool := range installedTools {
		if !trackedTools[tool] {
			if dryRun {
				wouldRemove = append(wouldRemove, tool)
			} else {
				parts := strings.Split(tool, "@")
				if len(parts) == 2 {
					name, version := parts[0], parts[1]
					toolPath := filepath.Join(r.silo.Root, "build-tools", name, version)
					if err := r.fs.RemoveAll(toolPath); err != nil {
						return nil, nil, fmt.Errorf("failed to remove %s: %w", tool, err)
					}
					removed = append(removed, tool)
				}
			}
		}
	}

	return removed, wouldRemove, nil
}
