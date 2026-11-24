package disk

import (
	"archive/tar"
	"compress/bzip2"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/internal/repository"
)

// downloadClient has a finite timeout so a stalled download fails instead of
// hanging the whole install. Generous enough for large archives (PHP source,
// ICU) but prevents an unreachable URL from blocking forever.

// SiloRepository is a disk-backed implementation of silo.SiloRepository.
// It downloads files via HTTP and stores them in baseDir. When a checksum
// is provided the downloaded file is verified before being committed to its
// final location so that corrupt or mismatched files never reach the silo.
type SiloRepository struct {
	baseDir string
	root    string
}

// NewSiloRepository creates a SiloRepository that stores downloaded files
// under $PHPV_ROOT/caches. When PHPV_ROOT is not set, $HOME/.phpv/caches is
// used as the base directory.
func NewSiloRepository() *SiloRepository {
	return &SiloRepository{
		baseDir: repository.ResolveCacheDir(),
		root:    repository.ResolveRoot(),
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
// Download delegates to the shared archive downloader.
func (s *SiloRepository) Download(url, checksumType, checksumValue string) (bool, error) {
	return repository.DownloadArchive(s.baseDir, url, checksumType, checksumValue)
}

func (s *SiloRepository) Extract(archivePath string, destDir string) (extracted bool, err error) {
	// Skip: destination already exists with content.
	if entries, err := os.ReadDir(destDir); err == nil && len(entries) > 0 {
		return false, nil
	}

	tmpDir := destDir + ".tmp"

	// Guard: remove stale .tmp dir from a previous interrupted run.
	_ = os.RemoveAll(tmpDir)

	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		return false, fmt.Errorf("create temp dir %s: %w", tmpDir, err)
	}

	f, err := os.Open(archivePath)
	if err != nil {
		os.RemoveAll(tmpDir)
		return false, fmt.Errorf("open archive %s: %w", archivePath, err)
	}
	defer f.Close()

	var tarSource io.Reader
	var gz *gzip.Reader

	ext := strings.ToLower(filepath.Ext(archivePath))
	// Handle double extensions like .tar.gz
	switch ext {
	case ".gz", ".xz", ".bz2", ".tgz":
		// Check the full extension for .tar.*
		if strings.HasSuffix(strings.ToLower(archivePath), ".tar.gz") ||
			strings.HasSuffix(strings.ToLower(archivePath), ".tgz") {
			var err error
			gz, err = gzip.NewReader(f)
			if err != nil {
				os.RemoveAll(tmpDir)
				return false, fmt.Errorf("gzip reader for %s: %w", archivePath, err)
			}
			tarSource = gz
		} else if strings.HasSuffix(strings.ToLower(archivePath), ".tar.xz") {
			// stdlib has no xz reader in compress — decompress via the xz
			// binary to a temp file, then extract the tar stream normally.
			f.Close()
			return repository.ExtractXz(archivePath, tmpDir, destDir)
		} else if strings.HasSuffix(strings.ToLower(archivePath), ".tar.bz2") {
			tarSource = bzip2.NewReader(f)
		} else {
			os.RemoveAll(tmpDir)
			return false, fmt.Errorf("unsupported archive format: %s", archivePath)
		}
	default:
		os.RemoveAll(tmpDir)
		return false, fmt.Errorf("unsupported archive format: %s", archivePath)
	}

	// Extract tar entries into tmpDir.
	if err := repository.ExtractTarEntries(tar.NewReader(tarSource), tmpDir); err != nil {
		os.RemoveAll(tmpDir)
		return false, fmt.Errorf("extract %s: %w", archivePath, err)
	}

	// Guard: for gzip archives the CRC32 and size footer are only validated
	// when the reader reaches the end of the stream. A truncated download can
	// otherwise end exactly at a tar end-of-archive boundary, extract
	// "cleanly", and be renamed into place with a partial last file. Force the
	// gzip reader to consume the trailer so a corrupt archive never produces a
	// destDir.
	if gz != nil {
		if _, err := io.Copy(io.Discard, gz); err != nil {
			os.RemoveAll(tmpDir)
			return false, fmt.Errorf("gzip checksum for %s: %w", archivePath, err)
		}
		if err := gz.Close(); err != nil {
			os.RemoveAll(tmpDir)
			return false, fmt.Errorf("gzip close for %s: %w", archivePath, err)
		}
	}

	// Guard: atomic rename — destDir appears only when extraction is complete.
	if err := os.Rename(tmpDir, destDir); err != nil {
		os.RemoveAll(tmpDir)
		return false, fmt.Errorf("move %s to %s: %w", tmpDir, destDir, err)
	}

	return true, nil
}

// extractTarEntries writes every entry in tr into tmpDir. It restores the
// archive mtimes so make does not spuriously regenerate shipped generated
// sources (a .c older than its .re would trigger re2c/bison regeneration).
func (s *SiloRepository) GetSilo() domain.Silo {
	return domain.Silo{Root: s.root}
}

// GetState reads the install state for a package.
func (s *SiloRepository) GetState(name, version string) (domain.InstallState, error) {
	path := repository.PackageStatePath(name, version)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return domain.StateNone, nil
		}
		return domain.StateNone, err
	}
	state := domain.InstallState(strings.TrimSpace(string(data)))
	return state, nil
}

// MarkInProgress marks a package installation as in-progress.
func (s *SiloRepository) MarkInProgress(name, version string) error {
	path := repository.PackageStatePath(name, version)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte("in_progress"), 0o644)
}

// MarkComplete marks a package installation as complete.
func (s *SiloRepository) MarkComplete(name, version string) error {
	return os.WriteFile(repository.PackageStatePath(name, version), []byte("installed"), 0o644)
}

// MarkFailed marks a package installation as failed.
func (s *SiloRepository) MarkFailed(name, version string) error {
	return os.WriteFile(repository.PackageStatePath(name, version), []byte("failed"), 0o644)
}

func (s *SiloRepository) MarkInterrupted(name, version string) error {
	return os.WriteFile(repository.PackageStatePath(name, version), []byte("interrupted"), 0o644)
}

// GetDefault reads the default PHP version.
func (s *SiloRepository) GetDefault() (string, error) {
	data, err := os.ReadFile(repository.DefaultPath())
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// SetDefault writes the default PHP version.
func (s *SiloRepository) SetDefault(version string) error {
	return os.WriteFile(repository.DefaultPath(), []byte(version+"\n"), 0o644)
}

// IsSystemMode returns true if the .phpv_system marker exists.
func (s *SiloRepository) IsSystemMode() bool {
	_, err := os.Stat(repository.SystemMarkerPath())
	return err == nil
}

// SetSystemMode creates or removes the .phpv_system marker.
func (s *SiloRepository) SetSystemMode(enabled bool) error {
	path := repository.SystemMarkerPath()
	if enabled {
		return os.WriteFile(path, []byte{}, 0o644)
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}
	return os.Remove(path)
}

// PHPOutputPath returns the install prefix for a PHP version.
func (s *SiloRepository) PHPOutputPath(phpVersion string) string {
	return repository.PHPOutputPath(phpVersion)
}

// SourcePath returns the extracted source directory for a package.
func (s *SiloRepository) SourcePath(pkg, version string) string {
	return repository.SourcePath(pkg, version)
}

// PackagePrefix returns the install prefix for any package.
func (s *SiloRepository) PackagePrefix(name, version string) string {
	return repository.PackagePrefix(name, version)
}

// PECLArchivePath returns the download cache path for a PECL archive.
func (s *SiloRepository) PECLArchivePath(name, version string) string {
	return repository.PECLArchivePath(name, version)
}

// BuildLogPath returns the path to a build log file.
func (s *SiloRepository) BuildLogPath(pkg, version, logName string) string {
	return repository.BuildLogPath(pkg, version, logName)
}

// GetExtensionManifest reads the extension manifest for a PHP version.
func (s *SiloRepository) GetExtensionManifest(phpVersion string) (*domain.ExtensionManifest, error) {
	path := repository.ExtensionManifestPath(phpVersion)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &domain.ExtensionManifest{PHPVersion: phpVersion}, nil
		}
		return nil, err
	}
	var m domain.ExtensionManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// SaveExtensionManifest writes the extension manifest for a PHP version.
func (s *SiloRepository) SaveExtensionManifest(phpVersion string, m *domain.ExtensionManifest) error {
	path := repository.ExtensionManifestPath(phpVersion)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// extractXz handles .tar.xz archives by decompressing with the xz binary to a
// temp file, then extracting the tar stream with Go's tar reader. Decompressing
// to a file (instead of piping xz's stdout directly into the tar reader)
// avoids a pipe-buffer deadlock: xz would block on write() once the pipe fills
// if Go is not draining it, stalling the whole install.
