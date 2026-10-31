package patches

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// applyPatches applies a range directory's patch files, in ordinal
// order, against the resolved source root under sourceDir. The first
// failure aborts loudly (the caller surfaces the error and the build
// stops) — silent skips of mismatching context are gone.
func applyPatches(pkg, version, rangeDir string, files []string, sourceDir string) error {
	if err := preflightPatchTool(); err != nil {
		return err
	}
	for _, f := range files {
		if err := applyPatchFile(f, sourceDir); err != nil {
			return fmt.Errorf("patch %s failed for %s@%s (range %s): %w",
				filepath.Base(f), pkg, version, rangeDir, err)
		}
	}
	return nil
}

// preflightPatchTool verifies the system `patch` binary is available,
// with an actionable per-distro install hint when it is not.
func preflightPatchTool() error {
	if _, err := exec.LookPath("patch"); err == nil {
		return nil
	}
	return fmt.Errorf("the 'patch' tool is required to apply source patches but was not found in PATH; install it with e.g. 'sudo dnf install patch', 'sudo apt install patch', or 'sudo pacman -S patch'")
}

// applyPatchFile applies one unified diff with `patch -p1 --forward -d
// <root> -i <file>`. Already-applied patches (patch reports them and
// skips under --forward) are tolerated; any other failure is loud.
func applyPatchFile(patchFile, sourceDir string) error {
	root, err := resolveSourceRoot(patchFile, sourceDir)
	if err != nil {
		return err
	}
	cmd := exec.Command("patch", "-p1", "--forward", "-d", root, "-i", patchFile)
	out, err := cmd.CombinedOutput()
	if err == nil {
		return nil
	}
	if isAlreadyApplied(string(out)) {
		return nil
	}
	return fmt.Errorf("applying patch failed (source root %s):\n%s", root, strings.TrimRight(string(out), "\n"))
}

// isAlreadyApplied recognizes patch's "previously applied" / skipped
// output, the one tolerated outcome per spec.
func isAlreadyApplied(out string) bool {
	l := strings.ToLower(out)
	return strings.Contains(l, "previously applied") ||
		strings.Contains(l, "reversed (or previously applied)")
}

// resolveSourceRoot locates the directory the patch's first target path
// lives in, probing the extracted tree shallowly (the tree is typically
// nested, e.g. <sourceDir>/php-7.4.33/). It replaces the old findFile
// content walk: only the first target path of the diff is probed, at
// directory depth <= 2, and the first candidate root containing that
// path wins.
func resolveSourceRoot(patchFile, sourceDir string) (string, error) {
	target, err := firstDiffTarget(patchFile)
	if err != nil {
		return "", err
	}
	// Candidate roots: sourceDir itself, then subdirectories at depth 1
	// and 2 (e.g. <sourceDir>/php-8.0.30/, <sourceDir>/a/b/).
	candidates := []string{sourceDir}
	for _, sub := range subdirs(sourceDir) {
		candidates = append(candidates, sub)
		candidates = append(candidates, subdirs(sub)...)
	}
	for _, c := range candidates {
		if _, err := os.Stat(filepath.Join(c, target)); err == nil {
			return c, nil
		}
	}
	// Nothing probed: fall back to the top of the extracted tree so the
	// patch tool produces the context-mismatch failure itself, which the
	// caller reports with the root included.
	return sourceDir, nil
}

// subdirs lists the immediate subdirectories of dir.
func subdirs(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			out = append(out, filepath.Join(dir, e.Name()))
		}
	}
	return out
}

// firstDiffTarget extracts the first real target path from the diff's
// +++ lines (skipping /dev/null entries used for created files) and
// strips the a/ prefix added by git/diff -u.
func firstDiffTarget(patchFile string) (string, error) {
	data, err := os.ReadFile(patchFile)
	if err != nil {
		return "", fmt.Errorf("read patch: %w", err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.HasPrefix(line, "+++ ") {
			continue
		}
		path := strings.TrimSpace(strings.TrimPrefix(line, "+++ "))
		path = strings.TrimPrefix(path, "b/")
		path = strings.TrimPrefix(path, "a/")
		if path == "/dev/null" {
			continue
		}
		// Drop a trailing tab-timestamp that plain `diff -u` headers carry.
		if i := strings.IndexAny(path, "\t"); i >= 0 {
			path = path[:i]
		}
		return path, nil
	}
	return "", fmt.Errorf("%s: no target path found in diff", filepath.Base(patchFile))
}
