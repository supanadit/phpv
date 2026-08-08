package disk

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/supanadit/phpv/doctor"
	"github.com/supanadit/phpv/domain"
)

// DoctorRepository is a disk-backed implementation of doctor.Repository.
// It exposes only the probes the doctor service needs to diagnose a phpv
// installation, hiding the underlying filesystem and environment details.
type DoctorRepository struct{}

// NewDoctorRepository creates a doctor.Repository backed by the OS.
func NewDoctorRepository() doctor.Repository {
	return &DoctorRepository{}
}

// LookPath reports whether an executable is available on PATH.
func (r *DoctorRepository) LookPath(name string) (string, error) {
	return exec.LookPath(name)
}

// ShimExists reports whether the php shim exists under root/bin.
func (r *DoctorRepository) ShimExists(root string) bool {
	_, err := os.Stat(filepath.Join(root, "bin", "php"))
	return err == nil
}

// GetDefaultVersion returns the configured default PHP version and whether a
// default has been set.
func (r *DoctorRepository) GetDefaultVersion(root string) (version string, exists bool) {
	data, err := os.ReadFile(filepath.Join(root, "default"))
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(string(data)), true
}

// IsVersionInstalled reports whether a PHP version is installed.
func (r *DoctorRepository) IsVersionInstalled(root, version string) bool {
	_, err := os.Stat(filepath.Join(root, "packages", "php", version, "bin", "php"))
	return err == nil
}

// IsCacheWritable verifies that the cache directory can be written to by
// creating and removing a probe file.
func (r *DoctorRepository) IsCacheWritable(root string) error {
	cacheDir := filepath.Join(root, "caches")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return err
	}
	testFile := filepath.Join(cacheDir, ".phpv_write_test")
	if err := os.WriteFile(testFile, []byte{}, 0o644); err != nil {
		return err
	}
	return os.Remove(testFile)
}

// ListPHPVersions returns the PHP versions installed under root/packages/php.
func (r *DoctorRepository) ListPHPVersions(root string) ([]string, error) {
	phpDir := filepath.Join(root, "packages", "php")
	entries, err := os.ReadDir(phpDir)
	if err != nil {
		return nil, err
	}
	var versions []string
	for _, e := range entries {
		if e.IsDir() {
			versions = append(versions, e.Name())
		}
	}
	return versions, nil
}

// GetInstallState returns the install state for a PHP version and whether a
// state file exists.
func (r *DoctorRepository) GetInstallState(root, version string) (domain.InstallState, bool) {
	data, err := os.ReadFile(filepath.Join(root, "packages", "php", version, ".state"))
	if err != nil {
		return domain.StateNone, false
	}
	return domain.InstallState(strings.TrimSpace(string(data))), true
}

// ExtensionManifestReadable reports whether the extension manifest for a PHP
// version can be read. A missing manifest is not an error.
func (r *DoctorRepository) ExtensionManifestReadable(root, version string) error {
	manifestPath := filepath.Join(root, "packages", "php", version, "extensions.json")
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		return nil
	}
	_, err := os.ReadFile(manifestPath)
	return err
}

// GetPHPVRoot returns the PHPV_ROOT environment variable value.
func (r *DoctorRepository) GetPHPVRoot() string {
	return os.Getenv("PHPV_ROOT")
}

// IsDirInPath reports whether dir is present in the PATH.
func (r *DoctorRepository) IsDirInPath(dir string) bool {
	for _, p := range filepath.SplitList(os.Getenv("PATH")) {
		if p == dir {
			return true
		}
	}
	return false
}

// IsSystemMode reports whether system mode is active.
func (r *DoctorRepository) IsSystemMode(root string) bool {
	_, err := os.Stat(filepath.Join(root, ".phpv_system"))
	return err == nil
}

// FreeDiskBytes returns the free bytes on the filesystem containing path.
func (r *DoctorRepository) FreeDiskBytes(path string) (uint64, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, err
	}
	return stat.Bavail * uint64(stat.Bsize), nil
}
