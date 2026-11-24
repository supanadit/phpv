package repository

import (
	"archive/tar"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var DownloadClient = &http.Client{Timeout: 5 * time.Minute}

func DownloadArchive(dir string, url string, checksumType string, checksumValue string) (downloaded bool, err error) {
	filename := filepath.Base(url)
	if filename == "" || filename == "." || filename == "/" {
		return false, fmt.Errorf("cannot determine filename from URL: %s", url)
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return false, fmt.Errorf("create silo directory %s: %w", dir, err)
	}

	finalPath := filepath.Join(dir, filename)
	tmpPath := finalPath + ".part"

	// Skip: file already exists with content. When a checksum is available the
	// existing file is re-verified so a corrupt or truncated file left behind by
	// an interrupted run is not silently trusted — it is removed and re-downloaded.
	if info, err := os.Stat(finalPath); err == nil && info.Size() > 0 {
		if checksumType != "" && checksumValue != "" {
			ok, err := FileMatchesChecksum(finalPath, checksumType, checksumValue)
			if err != nil {
				return false, fmt.Errorf("verify existing %s: %w", filename, err)
			}
			if !ok {
				_ = os.Remove(finalPath)
			} else {
				// Guard: clean up any stale .part file even when skipping, so it
				// doesn't linger from a previous interrupted run of a different
				// package that happened to use the same filename.
				_ = os.Remove(tmpPath)
				return false, nil
			}
		} else {
			// Guard: clean up any stale .part file even when skipping, so it
			// doesn't linger from a previous interrupted run of a different
			// package that happened to use the same filename.
			_ = os.Remove(tmpPath)
			return false, nil
		}
	}

	// Guard: remove stale .part file from a previous interrupted run.
	_ = os.Remove(tmpPath)

	// Start the HTTP request.
	resp, err := DownloadClient.Get(url)
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
		hasher, err = NewHasher(checksumType)
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

	written, err := io.Copy(tmp, body)
	if err != nil {
		cleanup()
		return false, fmt.Errorf("write %s: %w", tmpPath, err)
	}

	// Guard: a server that closes the connection early (truncated download)
	// can make io.Copy return cleanly. When the server advertised a
	// Content-Length, require the bytes written to match it so a partial
	// file is never committed to the silo.
	if resp.ContentLength >= 0 && written != resp.ContentLength {
		cleanup()
		return false, fmt.Errorf("download %s: truncated (expected %d bytes, got %d)",
			filename, resp.ContentLength, written)
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

// fileMatchesChecksum reports whether the file at path hashes to the expected
// checksum. It is used to re-verify an already-cached file before trusting it.

func FileMatchesChecksum(path, checksumType, checksumValue string) (bool, error) {
	hasher, err := NewHasher(checksumType)
	if err != nil {
		return false, err
	}
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()
	if _, err := io.Copy(hasher, f); err != nil {
		return false, err
	}
	got := hex.EncodeToString(hasher.Sum(nil))
	return strings.EqualFold(got, checksumValue), nil
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

func ExtractTarEntries(tr *tar.Reader, tmpDir string) error {
	for {
		header, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("tar read: %w", err)
		}

		target := filepath.Join(tmpDir, header.Name)

		// Guard against path traversal.
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(tmpDir)+string(os.PathSeparator)) {
			continue
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("mkdir %s: %w", target, err)
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return fmt.Errorf("mkdir for %s: %w", target, err)
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("create %s: %w", target, err)
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return fmt.Errorf("write %s: %w", target, err)
			}
			out.Close()
			_ = os.Chtimes(target, header.ModTime, header.ModTime)
		case tar.TypeSymlink:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return fmt.Errorf("mkdir for symlink %s: %w", target, err)
			}
			os.Symlink(header.Linkname, target)
		}
	}
}

// GetSilo returns the storage root.

func ExtractXz(archivePath string, tmpDir string, destDir string) (bool, error) {
	tempTar, err := os.CreateTemp("", "phpv-xz-*.tar")
	if err != nil {
		os.RemoveAll(tmpDir)
		return false, fmt.Errorf("create temp tar: %w", err)
	}
	tempPath := tempTar.Name()
	defer func() {
		tempTar.Close()
		os.Remove(tempPath)
	}()

	cmd := exec.Command("xz", "-dc", archivePath)
	cmd.Stdout = tempTar
	if err := cmd.Run(); err != nil {
		os.RemoveAll(tmpDir)
		return false, fmt.Errorf("xz decompression: %w", err)
	}
	if _, err := tempTar.Seek(0, io.SeekStart); err != nil {
		os.RemoveAll(tmpDir)
		return false, fmt.Errorf("seek temp tar: %w", err)
	}

	if err := ExtractTarEntries(tar.NewReader(tempTar), tmpDir); err != nil {
		os.RemoveAll(tmpDir)
		return false, fmt.Errorf("extract %s: %w", archivePath, err)
	}

	// Guard: atomic rename.
	if err := os.Rename(tmpDir, destDir); err != nil {
		os.RemoveAll(tmpDir)
		return false, fmt.Errorf("move %s to %s: %w", tmpDir, destDir, err)
	}

	return true, nil
}
