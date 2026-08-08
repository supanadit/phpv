package doctor

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/supanadit/phpv/domain"
)

// fakeRepository is a doctor.Repository test double. Filesystem probes
// delegate to the real temp directory; only env, LookPath, and free
// disk-space are controllable via fields.
type fakeRepository struct {
	env            map[string]string
	lookPathResult map[string]string
	diskFree       uint64
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{
		env:            make(map[string]string),
		lookPathResult: make(map[string]string),
		diskFree:       1 << 30,
	}
}

func (m *fakeRepository) LookPath(name string) (string, error) {
	if p, ok := m.lookPathResult[name]; ok {
		return p, nil
	}
	return "", os.ErrNotExist
}

func (m *fakeRepository) ShimExists(root string) bool {
	_, err := os.Stat(filepath.Join(root, "bin", "php"))
	return err == nil
}

func (m *fakeRepository) GetDefaultVersion(root string) (string, bool) {
	data, err := os.ReadFile(filepath.Join(root, "default"))
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(string(data)), true
}

func (m *fakeRepository) IsVersionInstalled(root, version string) bool {
	_, err := os.Stat(filepath.Join(root, "packages", "php", version, "bin", "php"))
	return err == nil
}

func (m *fakeRepository) IsCacheWritable(root string) error {
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

func (m *fakeRepository) ListPHPVersions(root string) ([]string, error) {
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

func (m *fakeRepository) GetInstallState(root, version string) (domain.InstallState, bool) {
	data, err := os.ReadFile(filepath.Join(root, "packages", "php", version, ".state"))
	if err != nil {
		return domain.StateNone, false
	}
	return domain.InstallState(strings.TrimSpace(string(data))), true
}

func (m *fakeRepository) ExtensionManifestReadable(root, version string) error {
	path := filepath.Join(root, "packages", "php", version, "extensions.json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}
	_, err := os.ReadFile(path)
	return err
}

func (m *fakeRepository) GetPHPVRoot() string {
	return m.env["PHPV_ROOT"]
}

func (m *fakeRepository) IsDirInPath(dir string) bool {
	for _, p := range filepath.SplitList(os.Getenv("PATH")) {
		if p == dir {
			return true
		}
	}
	return false
}

func (m *fakeRepository) IsSystemMode(root string) bool {
	_, err := os.Stat(filepath.Join(root, ".phpv_system"))
	return err == nil
}

func (m *fakeRepository) FreeDiskBytes(path string) (uint64, error) {
	return m.diskFree, nil
}
