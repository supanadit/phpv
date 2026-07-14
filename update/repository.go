package update

import (
	"context"
	"os"
)

type Release struct {
	TagName string
	Assets  []Asset
}

type Asset struct {
	Name        string
	DownloadURL string
	Size        int64
}

type Repository interface {
	FetchLatestRelease() (Release, error)
	DownloadFile(ctx context.Context, url, destPath string) error
	FetchChecksums(url string) (map[string]string, error)
	VerifyChecksum(filePath, expectedHash string) error
	ExecutablePath() (string, error)
	Stat(path string) (os.FileInfo, error)
	MkdirAll(path string, perm os.FileMode) error
	WriteFile(path string, data []byte, perm os.FileMode) error
	Remove(path string) error
	Chmod(path string, mode os.FileMode) error
	Rename(oldPath, newPath string) error
	Getenv(key string) string
}
