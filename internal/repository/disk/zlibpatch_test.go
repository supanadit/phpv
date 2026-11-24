package disk

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// repoPatchesDir is the publication tree shipped in this repository.
func repoPatchesDir(t *testing.T) string {
	t.Helper()
	for _, c := range []string{"../../patches", "../../../patches"} {
		if _, err := os.Stat(filepath.Join(c, "zlib")); err == nil {
			abs, err := filepath.Abs(c)
			if err != nil {
				t.Fatal(err)
			}
			return abs
		}
	}
	t.Fatal("patches/ tree not found relative to test")
	return ""
}

// fetchZlib extracts a pristine zlib tarball into a temp dir and returns the
// source root. Only 1.2.13 (the version phpv pins) is hosted on GitHub; the
// rest live in zlib.net's fossil archive.
func fetchZlib(t *testing.T, version string) string {
	t.Helper()
	url := "https://github.com/madler/zlib/releases/download/v" + version + "/zlib-" + version + ".tar.gz"
	if version != "1.2.13" {
		url = "https://zlib.net/fossils/zlib-" + version + ".tar.gz"
	}
	resp, err := http.Get(url)
	if err != nil {
		if version == "1.2.13" {
			t.Fatalf("cannot verify pinned zlib 1.2.13 without network: %v", err)
		}
		t.Skipf("network unavailable, skipping real-source validation: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		if version == "1.2.13" {
			t.Fatalf("fetch %s returned %s (pinned version must be verifiable)", url, resp.Status)
		}
		t.Skipf("fetch %s returned %s", url, resp.Status)
	}
	gz, err := gzip.NewReader(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		p := filepath.Join(root, filepath.FromSlash(hdr.Name))
		switch hdr.Typeflag {
		case tar.TypeDir:
			os.MkdirAll(p, 0o755)
		case tar.TypeReg:
			os.MkdirAll(filepath.Dir(p), 0o755)
			f, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY, os.FileMode(hdr.Mode)&0o777|0o400)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := io.Copy(f, tr); err != nil {
				t.Fatal(err)
			}
			f.Close()
		}
	}
	return filepath.Join(root, "zlib-"+version)
}

func readZutil(t *testing.T, src string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(src, "zutil.h"))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// Regression: zlib 1.2.x guarded its classic-MacOS fdopen shim with
// `defined(MACOS) || defined(TARGET_OS_MAC)`. Every modern Apple SDK defines
// TARGET_OS_MAC=1, so on macOS the shim redefined fdopen() to NULL and the
// real <stdio.h> declaration failed to compile under Xcode 16 (CI: PHP 7.4 on
// macOS). The shipped patch must apply through the real repository to the
// versions phpv pins (1.2.13) and to the rest of its declared range, and must
// NOT be selected outside it.
func TestZlibFdopenGuardPatch_AppliesToRealSource(t *testing.T) {
	repo := NewPatchRepository(repoPatchesDir(t), "")

	// Range selection: in-range gets the patch, out-of-range does not.
	for _, v := range []string{"1.2.9", "1.2.11", "1.2.13"} {
		if got := repo.PatchesFor("zlib", v); len(got) != 1 {
			t.Fatalf("PatchesFor(zlib, %s) = %d, want 1", v, len(got))
		}
	}
	for _, v := range []string{"1.2.8", "1.3.0", "1.3.1"} {
		if got := repo.PatchesFor("zlib", v); len(got) != 0 {
			t.Fatalf("PatchesFor(zlib, %s) = %d, want 0 (patch does not apply to this version)", v, len(got))
		}
	}

	// Apply through the real `patch -p1` path against pristine upstream source.
	// 1.2.13 is the pinned version and must always be verified; the lower bound
	// of the range is best-effort (its archive host is less reliable).
	for _, v := range []string{"1.2.13", "1.2.9"} {
		if v != "1.2.13" {
			t.Logf("additionally validating range lower bound %s", v)
		}
		src := fetchZlib(t, v)
		before := readZutil(t, src)
		if !strings.Contains(before, "#if defined(MACOS) || defined(TARGET_OS_MAC)") {
			t.Fatalf("zlib %s: expected buggy guard not present; patch is stale", v)
		}
		patches := repo.PatchesFor("zlib", v)
		for _, p := range patches {
			if err := p.Apply(src); err != nil {
				t.Fatalf("zlib %s: applying %s: %v", v, p.Name, err)
			}
		}
		after := readZutil(t, src)
		if strings.Contains(after, "defined(TARGET_OS_MAC)") {
			t.Fatalf("zlib %s: TARGET_OS_MAC guard still present after patch", v)
		}
		if !strings.Contains(after, "#if defined(MACOS)\n") {
			t.Fatalf("zlib %s: MACOS-only guard missing after patch", v)
		}
	}
}
