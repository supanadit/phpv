package disk

import (
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/supanadit/phpv/internal/repository"
)

// SiloRepository is a disk-backed implementation of silo.SiloRepository.
// It downloads files via HTTP and stores them in baseDir. When a checksum
// is provided the downloaded file is verified before being committed to its
// final location so that corrupt or mismatched files never reach the silo.
type SiloRepository struct {
	baseDir string
}

// NewSiloRepository creates a SiloRepository that stores downloaded files
// under $PHPV_ROOT/caches. When PHPV_ROOT is not set, $HOME/.phpv/caches is
// used as the base directory.
func NewSiloRepository() *SiloRepository {
	return &SiloRepository{
		baseDir: repository.ResolveCacheDir(),
	}
}

// Download fetches the file at url and stores it on disk under baseDir.
// The local filename is derived from the final path segment of the URL.
// When checksumType and checksumValue are non-empty the file is verified
// against the expected checksum before it is moved to its final location.
// A mismatch results in an error and the temporary file is removed.
func (s *SiloRepository) Download(url string, checksumType string, checksumValue string) (err error) {
	filename := filepath.Base(url)
	if filename == "" || filename == "." || filename == "/" {
		return fmt.Errorf("cannot determine filename from URL: %s", url)
	}

	if err := os.MkdirAll(s.baseDir, 0o755); err != nil {
		return fmt.Errorf("create silo directory %s: %w", s.baseDir, err)
	}

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("download %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: unexpected status %s", url, resp.Status)
	}

	verify := checksumType != "" && checksumValue != ""

	// When verifying we stream the body through a hasher so the checksum can
	// be computed in a single pass while writing to disk.
	var hasher hash.Hash
	if verify {
		hasher, err = repository.NewHasher(checksumType)
		if err != nil {
			return err
		}
	}

	// Download to a temporary file first so that a failed or mismatched
	// download never leaves a partial file in the final location.
	finalPath := filepath.Join(s.baseDir, filename)
	tmpPath := finalPath + ".part"

	tmp, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("create temp file %s: %w", tmpPath, err)
	}

	cleanup := func() {
		tmp.Close()
		os.Remove(tmpPath)
	}

	var body io.Reader = resp.Body
	if verify {
		body = io.TeeReader(resp.Body, hasher)
	}

	if _, err := io.Copy(tmp, body); err != nil {
		cleanup()
		return fmt.Errorf("write %s: %w", tmpPath, err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("close %s: %w", tmpPath, err)
	}

	if verify {
		got := hex.EncodeToString(hasher.Sum(nil))
		if !strings.EqualFold(got, checksumValue) {
			os.Remove(tmpPath)
			return fmt.Errorf("checksum mismatch for %s: expected %s, got %s",
				filename, checksumValue, got)
		}
	}

	if err := os.Rename(tmpPath, finalPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("move %s to %s: %w", tmpPath, finalPath, err)
	}

	return nil
}
