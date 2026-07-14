package disk

import (
	"os"
	"path/filepath"
	"syscall"
)

type DoctorRepository struct{}

func NewDoctorRepository() *DoctorRepository {
	return &DoctorRepository{}
}

func (r *DoctorRepository) Stat(path string) (os.FileInfo, error) {
	return os.Stat(path)
}

func (r *DoctorRepository) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (r *DoctorRepository) ReadDir(path string) ([]os.DirEntry, error) {
	return os.ReadDir(path)
}

func (r *DoctorRepository) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (r *DoctorRepository) WriteFile(path string, data []byte, perm os.FileMode) error {
	return os.WriteFile(path, data, perm)
}

func (r *DoctorRepository) Remove(path string) error {
	return os.Remove(path)
}

func (r *DoctorRepository) IsNotExist(err error) bool {
	return os.IsNotExist(err)
}

func (r *DoctorRepository) Getenv(key string) string {
	return os.Getenv(key)
}

func (r *DoctorRepository) PathList() []string {
	return filepath.SplitList(os.Getenv("PATH"))
}

func (r *DoctorRepository) Statfs(path string) (bavail, bsize uint64, err error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, 0, err
	}
	return stat.Bavail, uint64(stat.Bsize), nil
}
