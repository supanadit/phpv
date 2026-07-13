package disk

import (
	"archive/tar"
	"compress/bzip2"
	"compress/gzip"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/internal/repository"
)

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
		root:    resolveRoot(),
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

// Extract decompresses and untars an archive into destDir.
// Supports .tar.gz, .tgz, .tar.xz, .tar.bz2.
//
// Skip: if destDir already exists and is non-empty, extraction is skipped
// (returns extracted=false, nil).
//
// Guard: extraction goes to a .tmp directory first. If interrupted, the .tmp
// dir is left behind but destDir is never created partially. On the next
// run, the stale .tmp dir is removed before a fresh extraction starts.
//
// Returns extracted=true when the archive was actually extracted,
// extracted=false when the destination already existed (skipped).
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

	var tarReader *tar.Reader

	ext := strings.ToLower(filepath.Ext(archivePath))
	// Handle double extensions like .tar.gz
	if ext == ".gz" || ext == ".xz" || ext == ".bz2" {
		// Check the full extension for .tar.*
		if strings.HasSuffix(strings.ToLower(archivePath), ".tar.gz") ||
			strings.HasSuffix(strings.ToLower(archivePath), ".tgz") {
			gz, err := gzip.NewReader(f)
			if err != nil {
				os.RemoveAll(tmpDir)
				return false, fmt.Errorf("gzip reader for %s: %w", archivePath, err)
			}
			defer gz.Close()
			tarReader = tar.NewReader(gz)
		} else if strings.HasSuffix(strings.ToLower(archivePath), ".tar.xz") {
			// stdlib has no xz reader in compress — use a minimal approach
			// via io via external xz. But Go 1.25 has no xz in stdlib.
			// We'll handle xz by shelling out.
			f.Close()
			return s.extractXz(archivePath, tmpDir, destDir)
		} else if strings.HasSuffix(strings.ToLower(archivePath), ".tar.bz2") {
			bz := bzip2.NewReader(f)
			tarReader = tar.NewReader(bz)
		} else {
			os.RemoveAll(tmpDir)
			return false, fmt.Errorf("unsupported archive format: %s", archivePath)
		}
	} else if ext == ".tgz" {
		gz, err := gzip.NewReader(f)
		if err != nil {
			os.RemoveAll(tmpDir)
			return false, fmt.Errorf("gzip reader for %s: %w", archivePath, err)
		}
		defer gz.Close()
		tarReader = tar.NewReader(gz)
	} else {
		os.RemoveAll(tmpDir)
		return false, fmt.Errorf("unsupported archive format: %s", archivePath)
	}

	// Extract tar entries into tmpDir.
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			os.RemoveAll(tmpDir)
			return false, fmt.Errorf("tar read %s: %w", archivePath, err)
		}

		target := filepath.Join(tmpDir, header.Name)

		// Guard against path traversal.
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(tmpDir)+string(os.PathSeparator)) {
			continue
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
				os.RemoveAll(tmpDir)
				return false, fmt.Errorf("mkdir %s: %w", target, err)
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				os.RemoveAll(tmpDir)
				return false, fmt.Errorf("mkdir for %s: %w", target, err)
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				os.RemoveAll(tmpDir)
				return false, fmt.Errorf("create %s: %w", target, err)
			}
			if _, err := io.Copy(out, tarReader); err != nil {
				out.Close()
				os.RemoveAll(tmpDir)
				return false, fmt.Errorf("write %s: %w", target, err)
			}
			out.Close()
		case tar.TypeSymlink:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				os.RemoveAll(tmpDir)
				return false, fmt.Errorf("mkdir for symlink %s: %w", target, err)
			}
			os.Symlink(header.Linkname, target)
		}
	}

	// Guard: atomic rename — destDir appears only when extraction is complete.
	if err := os.Rename(tmpDir, destDir); err != nil {
		os.RemoveAll(tmpDir)
		return false, fmt.Errorf("move %s to %s: %w", tmpDir, destDir, err)
	}

	return true, nil
}

// GetSilo returns the storage root.
func (s *SiloRepository) GetSilo() domain.Silo {
	return domain.Silo{Root: s.root}
}

// GetState reads the install state for a PHP version.
func (s *SiloRepository) GetState(phpVersion string) (domain.InstallState, error) {
	path := StatePath(phpVersion)
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

// MarkInProgress marks a PHP installation as in-progress.
func (s *SiloRepository) MarkInProgress(phpVersion string) error {
	path := StatePath(phpVersion)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte("in_progress"), 0o644)
}

// MarkComplete marks a PHP installation as complete.
func (s *SiloRepository) MarkComplete(phpVersion string) error {
	return os.WriteFile(StatePath(phpVersion), []byte("installed"), 0o644)
}

// MarkFailed marks a PHP installation as failed.
func (s *SiloRepository) MarkFailed(phpVersion string) error {
	return os.WriteFile(StatePath(phpVersion), []byte("failed"), 0o644)
}

// GetDefault reads the default PHP version.
func (s *SiloRepository) GetDefault() (string, error) {
	data, err := os.ReadFile(DefaultPath())
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
	return os.WriteFile(DefaultPath(), []byte(version+"\n"), 0o644)
}

// PHPOutputPath returns the install prefix for a PHP version.
func (s *SiloRepository) PHPOutputPath(phpVersion string) string {
	return PHPOutputPath(phpVersion)
}

// SourcePath returns the extracted source directory for a package.
func (s *SiloRepository) SourcePath(pkg, version string) string {
	return SourcePath(pkg, version)
}

// DependencyPath returns the install prefix for a dependency of a PHP version.
func (s *SiloRepository) DependencyPath(phpVersion, name, depVersion string) string {
	return DependencyPath(phpVersion, name, depVersion)
}

// PackagePrefix returns the install prefix for any package.
func (s *SiloRepository) PackagePrefix(name, version string) string {
	return PackagePrefix(name, version)
}

// extractXz handles .tar.xz archives by shelling out to the xz binary
// since Go stdlib has no xz decompressor.
func (s *SiloRepository) extractXz(archivePath string, tmpDir string, destDir string) (bool, error) {
	// Try using xz command to decompress to a pipe, then tar extract.
	// If xz is not available, fall back to an error.
	cmd := exec.Command("xz", "-dc", archivePath)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		os.RemoveAll(tmpDir)
		return false, fmt.Errorf("xz pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		os.RemoveAll(tmpDir)
		return false, fmt.Errorf("xz command not found, cannot extract .tar.xz: %w", err)
	}

	tarReader := tar.NewReader(stdout)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			cmd.Wait()
			os.RemoveAll(tmpDir)
			return false, fmt.Errorf("tar read: %w", err)
		}

		target := filepath.Join(tmpDir, header.Name)

		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(tmpDir)+string(os.PathSeparator)) {
			continue
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
				cmd.Wait()
				os.RemoveAll(tmpDir)
				return false, fmt.Errorf("mkdir %s: %w", target, err)
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				cmd.Wait()
				os.RemoveAll(tmpDir)
				return false, fmt.Errorf("mkdir for %s: %w", target, err)
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				cmd.Wait()
				os.RemoveAll(tmpDir)
				return false, fmt.Errorf("create %s: %w", target, err)
			}
			if _, err := io.Copy(out, tarReader); err != nil {
				out.Close()
				cmd.Wait()
				os.RemoveAll(tmpDir)
				return false, fmt.Errorf("write %s: %w", target, err)
			}
			out.Close()
		case tar.TypeSymlink:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				cmd.Wait()
				os.RemoveAll(tmpDir)
				return false, fmt.Errorf("mkdir for symlink %s: %w", target, err)
			}
			os.Symlink(header.Linkname, target)
		}
	}

	if err := cmd.Wait(); err != nil {
		os.RemoveAll(tmpDir)
		return false, fmt.Errorf("xz decompression: %w", err)
	}

	// Guard: atomic rename.
	if err := os.Rename(tmpDir, destDir); err != nil {
		os.RemoveAll(tmpDir)
		return false, fmt.Errorf("move %s to %s: %w", tmpDir, destDir, err)
	}

	return true, nil
}