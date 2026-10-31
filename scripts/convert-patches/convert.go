package main

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/supanadit/phpv/patcher"
)

// fetchExtract downloads url and extracts the .tar.gz into destDir,
// returning the single extracted top-level directory.
func fetchExtract(ctx context.Context, url, destDir string) (string, error) {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", err
	}
	tgz := filepath.Join(destDir, "src.tar.gz")
	if err := download(ctx, url, tgz); err != nil {
		return "", err
	}
	if err := untar(tgz, destDir); err != nil {
		return "", err
	}
	entries, err := os.ReadDir(destDir)
	if err != nil {
		return "", err
	}
	var top string
	for _, e := range entries {
		if e.IsDir() && e.Name() != "." {
			if top != "" {
				return "", fmt.Errorf("multiple top-level dirs in %s", url)
			}
			top = filepath.Join(destDir, e.Name())
		}
	}
	if top == "" {
		return "", fmt.Errorf("no top-level dir extracted from %s", url)
	}
	return top, nil
}

func download(ctx context.Context, url, dest string) error {
	c := &http.Client{Timeout: 10 * time.Minute}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

func untar(tgz, dest string) error {
	f, err := os.Open(tgz)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		clean, ok := safeJoin(dest, h.Name)
		if !ok {
			return fmt.Errorf("unsafe tar entry %q", h.Name)
		}
		switch h.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(clean, os.FileMode(h.Mode)&0o777); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(clean), 0o755); err != nil {
				return err
			}
			out, err := os.OpenFile(clean, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(h.Mode)&0o777)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			out.Close()
		}
	}
}

func safeJoin(base, name string) (string, bool) {
	p := filepath.Join(base, filepath.FromSlash(name))
	rel, err := filepath.Rel(base, p)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", false
	}
	return p, true
}

// copyTree recursively copies src to dst preserving modes.
func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode().Perm())
	})
}

// generate produces one .patch file for an entry.
func generate(ctx context.Context, repo patcher.PatcherRepository, e entry, outRoot string) (string, error) {
	pkg := strings.SplitN(e.outFile, "/", 2)[0]
	work, err := os.MkdirTemp("", "convert-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(work)

	src, err := fetchExtract(ctx, e.url(e.genVersion), filepath.Join(work, "dl"))
	if err != nil {
		return "", fmt.Errorf("fetch %s: %w", e.genVersion, err)
	}
	top := filepath.Base(src)
	// Parents a/ and b/ each contain the pristine top dir; the Go patch
	// functions are applied against the b/ parent, mirroring how some of
	// them expect the source nested one level down.
	parentA, parentB := filepath.Join(work, "a"), filepath.Join(work, "b")
	for _, p := range []string{parentA, parentB} {
		if err := os.MkdirAll(p, 0o755); err != nil {
			return "", err
		}
	}
	if err := copyTree(src, filepath.Join(parentA, top)); err != nil {
		return "", err
	}
	if err := copyTree(src, filepath.Join(parentB, top)); err != nil {
		return "", err
	}

	// Run the historical Go patch function on the b/ copy via the public API.
	patch := selectPatch(repo, pkg, e.genVersion, e.patchName)
	if patch == nil || patch.Apply == nil {
		return "", fmt.Errorf("patch %q not found for %s@%s (or has no Apply)", e.patchName, pkg, e.genVersion)
	}
	if err := patch.Apply(parentB); err != nil {
		return "", fmt.Errorf("Go patch function failed on %s: %w", e.genVersion, err)
	}

	diff, diffErr := gitDiffNoIndex(work, "a", "b")
	if diffErr != "" {
		return "", fmt.Errorf("git diff: %s", diffErr)
	}
	if strings.TrimSpace(diff) == "" {
		return "", fmt.Errorf("empty diff: Go patch function made no changes on %s", e.genVersion)
	}

	body := normalizeDiff(diff, top)
	out := filepath.Join(outRoot, filepath.FromSlash(e.outFile))
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(out, []byte(header(e, body)+body), 0o644); err != nil {
		return "", err
	}
	return out, nil
}

func selectPatch(repo patcher.PatcherRepository, pkg, version, name string) *patcher.Patch {
	for _, p := range repo.PatchesFor(pkg, version) {
		if p.Name == name {
			p := p
			return &p
		}
	}
	return nil
}

// gitDiffNoIndex runs `git diff --no-index a b` in workdir. Differences
// produce exit code 1, which is expected: it returns (diff, ""). The
// second return value is a non-empty error message on real failure.
func gitDiffNoIndex(work, a, b string) (string, string) {
	cmd := exec.Command("git", "diff", "--no-index", "-U3", filepath.Base(a), filepath.Base(b))
	cmd.Dir = work
	out, _ := cmd.CombinedOutput()
	if !strings.Contains(string(out), "diff --git") {
		return "", string(out)
	}
	return string(out), ""
}

// normalizeDiff rewrites `git diff --no-index a b` header paths
// (--- a/a/<top>/x, +++ b/b/<top>/x) into -p1 form relative to the
// source root (--- a/x, +++ b/x).
func normalizeDiff(diff, top string) string {
	aPrefix, bPrefix := "a/a/"+top+"/", "b/b/"+top+"/"
	lines := strings.Split(diff, "\n")
	for i, l := range lines {
		switch {
		case strings.HasPrefix(l, "--- "+aPrefix):
			lines[i] = "--- a/" + strings.TrimPrefix(l, "--- "+aPrefix)
		case strings.HasPrefix(l, "+++ "+bPrefix):
			lines[i] = "+++ b/" + strings.TrimPrefix(l, "+++ "+bPrefix)
		}
	}
	return strings.Join(lines, "\n")
}

// header wraps the diff in a pyenv-style mail header carrying the
// root-cause description as the Subject/body.
func header(e entry, body string) string {
	slug := filepath.Base(e.outFile)
	slug = strings.TrimSuffix(slug, ".patch")
	var b strings.Builder
	b.WriteString("From 0000000000000000000000000000000000000000 Mon Sep 17 00:00:00 2001\n")
	b.WriteString("From: phpv patch conversion <patches@phpv.invalid>\n")
	fmt.Fprintf(&b, "Subject: [%s] %s\n\n", slug, firstSentence(e.description))
	b.WriteString(wrapWords(e.description, 72))
	b.WriteString("\n---\n")
	return b.String()
}

func firstSentence(s string) string {
	if i := strings.Index(s, ". "); i >= 0 {
		return s[:i+1]
	}
	return s
}

func wrapWords(s string, width int) string {
	var b strings.Builder
	lineLen := 0
	for _, w := range strings.Fields(s) {
		if lineLen > 0 && lineLen+1+len(w) > width {
			b.WriteString("\n")
			lineLen = 0
		}
		if lineLen > 0 {
			b.WriteString(" ")
			lineLen++
		}
		b.WriteString(w)
		lineLen += len(w)
	}
	b.WriteString("\n")
	return b.String()
}

// checkEquivalence verifies that applying the generated .patch with the
// patch tool yields byte-identical files to running the legacy Go patch
// function, on the entry's generation version.
func checkEquivalence(ctx context.Context, repo patcher.PatcherRepository, e entry) error {
	pkg := strings.SplitN(e.outFile, "/", 2)[0]
	work, err := os.MkdirTemp("", "equiv-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(work)

	src, err := fetchExtract(ctx, e.url(e.genVersion), filepath.Join(work, "dl"))
	if err != nil {
		return fmt.Errorf("fetch %s: %w", e.genVersion, err)
	}
	top := filepath.Base(src)

	// a/: legacy Go function output. b/: generated patch applied.
	parentA, parentB := filepath.Join(work, "a"), filepath.Join(work, "b")
	for _, p := range []string{parentA, parentB} {
		if err := os.MkdirAll(p, 0o755); err != nil {
			return err
		}
	}
	if err := copyTree(src, filepath.Join(parentA, top)); err != nil {
		return err
	}
	if err := copyTree(src, filepath.Join(parentB, top)); err != nil {
		return err
	}
	patch := selectPatch(repo, pkg, e.genVersion, e.patchName)
	if patch == nil || patch.Apply == nil {
		return fmt.Errorf("patch %q not found for %s@%s", e.patchName, pkg, e.genVersion)
	}
	if err := patch.Apply(parentA); err != nil {
		return fmt.Errorf("legacy function: %w", err)
	}
	patchFile, err := filepath.Abs(filepath.Join("patches", filepath.FromSlash(e.outFile)))
	if err != nil {
		return err
	}
	cmd := exec.Command("patch", "-p1", "--forward", "-d", filepath.Join(parentB, top), "-i", patchFile)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("patch apply:\n%s", strings.TrimRight(string(out), "\n"))
	}

	// Every file touched by the legacy function must be byte-identical.
	var mism int
	err = filepath.WalkDir(parentA, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(parentA, p)
		other := filepath.Join(parentB, rel)
		oa, erra := os.ReadFile(p)
		ob, errb := os.ReadFile(other)
		if erra != nil || errb != nil || string(oa) != string(ob) {
			fmt.Printf("      differs: %s\n", rel)
			mism++
		}
		return nil
	})
	if err != nil {
		return err
	}
	if mism > 0 {
		return fmt.Errorf("%d file(s) differ between legacy function and generated patch", mism)
	}
	return nil
}

// validate dry-runs the generated patch against pristine upstream source
// for the given version.
func validate(ctx context.Context, e entry, version, patchFile string) error {
	work, err := os.MkdirTemp("", "validate-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(work)
	src, err := fetchExtract(ctx, e.url(version), work)
	if err != nil {
		return fmt.Errorf("fetch %s: %w", version, err)
	}
	abs, err := filepath.Abs(patchFile)
	if err != nil {
		return err
	}
	cmd := exec.Command("patch", "--dry-run", "-p1", "-d", src, "-i", abs)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("dry-run on %s failed:\n%s", version, strings.TrimRight(string(out), "\n"))
	}
	return nil
}
