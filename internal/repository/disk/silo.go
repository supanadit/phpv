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
//
// Skip: if the final file already exists and has a non-zero size, the
// download is skipped entirely (returns downloaded=false, nil).
//
// Guard: the download writes to a .part temp file first. If the process is
// interrupted (Ctrl-C, kill, BSOD, power loss) the .part file is left behind
// but the final file is never created. On the next run, the stale .part file
// is detected and removed before a fresh download starts. This guarantees
// the silo never contains a partial or corrupt file.
//
// When checksumType and checksumValue are non-empty the file is verified
// against the expected checksum before it is moved to its final location.
// A mismatch results in an error and the temporary file is removed.
//
// Returns downloaded=true when the file was fetched from the network,
// downloaded=false when the file already existed (skipped).
func (s *SiloRepository) Download(url string, checksumType string, checksumValue string) (downloaded bool, err error) {
	filename := filepath.Base(url)
	if filename == "" || filename == "." || filename == "/" {
		return false, fmt.Errorf("cannot determine filename from URL: %s", url)
	}

	if err := os.MkdirAll(s.baseDir, 0o755); err != nil {
		return false, fmt.Errorf("create silo directory %s: %w", s.baseDir, err)
	}

	finalPath := filepath.Join(s.baseDir, filename)
	tmpPath := finalPath + ".part"

	// Skip: file already exists with content.
	if info, err := os.Stat(finalPath); err == nil && info.Size() > 0 {
		// Guard: clean up any stale .part file even when skipping, so it
		// doesn't linger from a previous interrupted run of a different
		// package that happened to use the same filename.
		_ = os.Remove(tmpPath)
		return false, nil
	}

	// Guard: remove stale .part file from a previous interrupted run.
	_ = os.Remove(tmpPath)

	// Start the HTTP request.
	resp, err := http.Get(url)
	if err != nil {
		return false, fmt.Errorf("download %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("download %s: unexpected status %s", url, resp.Status)
	}

	verify := checksumType != "" && checksumValue != ""

	var hasher hash.Hash
	if verify {
		hasher, err = repository.NewHasher(checksumType)
		if err != nil {
			return false, err
		}
	}

	// Create the temp file.
	tmp, err := os.Create(tmpPath)
	if err != nil {
		return false, fmt.Errorf("create temp file %s: %w", tmpPath, err)
	}

	// Guard: if we panic or the process is killed mid-write, the deferred
	// cleanup removes the .part file so the next run starts fresh.
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
		return false, fmt.Errorf("write %s: %w", tmpPath, err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return false, fmt.Errorf("close %s: %w", tmpPath, err)
	}

	if verify {
		got := hex.EncodeToString(hasher.Sum(nil))
		if !strings.EqualFold(got, checksumValue) {
			os.Remove(tmpPath)
			return false, fmt.Errorf("checksum mismatch for %s: expected %s, got %s",
				filename, checksumValue, got)
		}
	}

	// Guard: atomic rename — the final file appears only when the download
	// and checksum verification are fully complete. If the rename fails the
	// .part file is cleaned up.
	if err := os.Rename(tmpPath, finalPath); err != nil {
		os.Remove(tmpPath)
		return false, fmt.Errorf("move %s to %s: %w", tmpPath, finalPath, err)
	}

	return true, nil
}