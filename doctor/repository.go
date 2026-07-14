package doctor

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

type Repository interface {
	Stat(path string) (os.FileInfo, error)
	ReadFile(path string) ([]byte, error)
	ReadDir(path string) ([]os.DirEntry, error)
	MkdirAll(path string, perm os.FileMode) error
	WriteFile(path string, data []byte, perm os.FileMode) error
	Remove(path string) error
	IsNotExist(err error) bool
	Getenv(key string) string
	PathList() []string
	Statfs(path string) (bavail, bsize uint64, err error)
	LookPath(name string) (string, error)
}

type osRepository struct{}

func newOSRepository() Repository {
	return &osRepository{}
}

func (r *osRepository) Stat(path string) (os.FileInfo, error) {
	return os.Stat(path)
}

func (r *osRepository) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (r *osRepository) ReadDir(path string) ([]os.DirEntry, error) {
	return os.ReadDir(path)
}

func (r *osRepository) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (r *osRepository) WriteFile(path string, data []byte, perm os.FileMode) error {
	return os.WriteFile(path, data, perm)
}

func (r *osRepository) Remove(path string) error {
	return os.Remove(path)
}

func (r *osRepository) IsNotExist(err error) bool {
	return os.IsNotExist(err)
}

func (r *osRepository) Getenv(key string) string {
	return os.Getenv(key)
}

func (r *osRepository) PathList() []string {
	return filepath.SplitList(os.Getenv("PATH"))
}

func (r *osRepository) LookPath(name string) (string, error) {
	return exec.LookPath(name)
}

func (r *osRepository) Statfs(path string) (bavail, bsize uint64, err error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, 0, err
	}
	return stat.Bavail, uint64(stat.Bsize), nil
}
