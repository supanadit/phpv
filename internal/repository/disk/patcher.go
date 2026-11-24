package disk

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	vsn "github.com/supanadit/phpv/pkg/version"

	"github.com/supanadit/phpv/domain"
	"gopkg.in/yaml.v3"
)

// PatchRepository implements the file-based patch repository: patches
// live as unified diff files in a versioned directory tree
// (<root>/<package>/<version-range>/<ordinal>-<slug>.patch) with optional
// flags.yaml manifests, merged from two on-disk sources (synced and
// user-local) with per-directory precedence. No patch content is embedded
// in the binary; everything is read from the injected roots.

// Source labels for the two patch trees, used for observability output
// (phpv patch list).
const (
	SourceLocal  = "user-local"
	SourceSynced = "synced"
)

// anyDirName is the special range-directory name matching every version
// of a package (used by config-only entries).
const anyDirName = "any"

// DirRangeExpr translates a range-directory name into the version-range
// expression evaluated by vsn.MatchVersionRange. Directory names use
// a Windows-safe encoding of the range syntax:
//
//	any          → ""   (matches every version)
//	8.0.0-8.1.0  → ">=8.0.0,<8.1.0"   (lower inclusive, upper exclusive)
//	6.0.0+       → ">=6.0.0"          (open-ended)
//	6.9.7        → "=6.9.7"           (exact version)
func DirRangeExpr(dir string) string {
	if dir == anyDirName || dir == "" {
		return ""
	}
	if strings.HasSuffix(dir, "+") {
		return ">=" + strings.TrimSuffix(dir, "+")
	}
	if a, b, ok := strings.Cut(dir, "-"); ok && a != "" && b != "" {
		// Both sides must look like plain versions to avoid misreading
		// other directory names; versions carry no dashes in phpv.
		if isPlainVersion(a) && isPlainVersion(b) {
			return ">=" + a + ",<" + b
		}
	}
	return "=" + dir
}

// isPlainVersion reports whether s is a bare x[.y[.z]] version.
func isPlainVersion(s string) bool {
	if s == "" {
		return false
	}
	for _, seg := range strings.Split(s, ".") {
		if seg == "" {
			return false
		}
		for _, r := range seg {
			if r < '0' || r > '9' {
				return false
			}
		}
	}
	return true
}

// root is one patch source tree. Roots are consulted in precedence order;
// the first root to claim a directory path owns it wholly.
type patchRoot struct {
	path  string
	label string
}

// Repository is the file-based patch repository.
type PatchRepository struct {
	roots []patchRoot
}

// NewRepository creates a patch repository reading patches from
// the given on-disk trees in precedence order: localRoot (user-local)
// shadows syncedRoot per directory. Missing roots are simply empty.
func NewPatchRepository(syncedRoot, localRoot string, opts ...PatchOption) *PatchRepository {
	r := &PatchRepository{}
	for _, o := range opts {
		o(r)
	}
	// User-local patches take precedence: they are consulted first.
	if localRoot != "" {
		r.roots = append(r.roots, patchRoot{path: localRoot, label: SourceLocal})
	}
	if syncedRoot != "" {
		r.roots = append(r.roots, patchRoot{path: syncedRoot, label: SourceSynced})
	}
	return r
}

// Option customizes the repository.
type PatchOption func(*PatchRepository)

// PatchEntry is one selected patch-tree directory: its patch files (in
// application order), optional flags manifest, and the source tree it
// came from.
type PatchEntry = domain.PatchEntry

// EntriesFor returns all patch-tree directories matching (name, version)
// in ordinal order, labeled by their owning source tree. It is the
// observability view behind PatchesFor (used by `phpv patch list`).
func (r *PatchRepository) EntriesFor(name string, version string) []PatchEntry {
	owned := make(map[string]PatchEntry)
	for _, src := range r.roots {
		pkgDir := filepath.Join(src.path, name)
		pkgEntries, err := readPackageDir(pkgDir, src.label)
		if err != nil {
			// A missing or unreadable package directory is simply "no
			// patches from this source"; per-directory read errors are
			// skipped inside readPackageDir.
			continue
		}
		for _, e := range pkgEntries {
			if _, claimed := owned[e.DirPath]; !claimed {
				owned[e.DirPath] = e
			}
		}
	}
	entries := make([]PatchEntry, 0, len(owned))
	for _, e := range owned {
		if !entryMatches(e, version) {
			continue
		}
		entries = append(entries, e)
	}
	sortEntries(entries)
	return entries
}

// PatchesFor resolves the
// selected entries for (name, version) and converts them into Patch
// structs whose Apply runs `patch -p1` against the located source root.
func (r *PatchRepository) PatchesFor(name string, version string) []domain.SourcePatch {
	entries := r.EntriesFor(name, version)
	if len(entries) == 0 {
		return nil
	}
	patches := make([]domain.SourcePatch, 0, len(entries))
	for _, e := range entries {
		p := domain.SourcePatch{
			Package: name,
			// "any" directories match every version; the service-side
			// re-filter passes an empty range.
			VersionRange: rangeExpr(e.RangeDir),
		}
		if e.Manifest != nil {
			p.ExtraCFlags = e.Manifest.ExtraCFlags
			p.ConfigureFlags = e.Manifest.ConfigureFlags
		}
		if len(e.PatchFiles) > 0 {
			files := e.PatchFiles
			dirName := filepath.Base(e.DirPath)
			p.Name = patchDisplayName(files)
			p.Apply = func(sourceDir string) error {
				return applyPatches(patchSet{pkg: name, version: version, rangeDir: dirName, files: files, sourceDir: sourceDir})
			}
		} else {
			p.Name = e.DirPath
		}
		patches = append(patches, p)
	}
	return patches
}

// patchDisplayName derives the reported name from the directory's patch
// files (first ordinal's slug; the whole set is applied together).
func patchDisplayName(files []string) string {
	base := filepath.Base(files[0])
	return strings.TrimSuffix(base, ".patch")
}

// rangeExpr converts a range directory name into the VersionRange value
// carried on the Patch struct. "any" matches everything.
func rangeExpr(rangeDir string) string {
	return DirRangeExpr(rangeDir)
}

// entryMatches reports whether the directory's range name matches the
// built version.
func entryMatches(e PatchEntry, version string) bool {
	if e.RangeDir == anyDirName {
		return true
	}
	return vsn.MatchVersionRange(DirRangeExpr(e.RangeDir), version)
}

// sortEntries orders selected directories by their first patch file's
// ordinal filename, then by range name for stability. Flags-only entries
// sort by range name.
func sortEntries(entries []PatchEntry) {
	sort.SliceStable(entries, func(i, j int) bool {
		return firstOrdinal(entries[i]) < firstOrdinal(entries[j])
	})
}

func firstOrdinal(e PatchEntry) string {
	if len(e.PatchFiles) > 0 {
		return filepath.Base(e.PatchFiles[0])
	}
	return e.RangeDir
}

// readPackageDir reads every range directory under <pkgDir>, skipping
// per-directory entries that fail to read (the other sources or a later
// run still work).
func readPackageDir(pkgDir string, label string) ([]PatchEntry, error) {
	dirs, err := os.ReadDir(pkgDir)
	if err != nil {
		return nil, err
	}
	entries := make([]PatchEntry, 0, len(dirs))
	for _, d := range dirs {
		if !d.IsDir() {
			continue
		}
		rangeDir := d.Name()
		dirPath := filepath.Join(pkgDir, rangeDir)
		files, err := patchFilesIn(dirPath)
		if err != nil {
			continue
		}
		manifest, err := LoadManifest(filepath.Join(dirPath, PatchFlagsFileName))
		if err != nil {
			continue
		}
		rel, err := filepath.Rel(filepath.Dir(pkgDir), dirPath)
		if err != nil {
			continue
		}
		entries = append(entries, PatchEntry{
			DirPath:    filepath.ToSlash(rel),
			Package:    filepath.Base(pkgDir),
			RangeDir:   rangeDir,
			Source:     label,
			PatchFiles: files,
			Manifest:   manifest,
		})
	}
	return entries, nil
}

// patchFilesIn lists the .patch files of a range directory in ascending
// ordinal order.
func patchFilesIn(dirPath string) ([]string, error) {
	dirs, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}
	files := make([]string, 0, len(dirs))
	for _, d := range dirs {
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".patch") {
			continue
		}
		files = append(files, filepath.Join(dirPath, d.Name()))
	}
	sort.Strings(files)
	return files, nil
}

// applyPatches applies a range directory's patch files, in ordinal
// order, against the resolved source root under sourceDir. The first
// failure aborts loudly (the caller surfaces the error and the build
// stops) — silent skips of mismatching context are gone.
type patchSet struct {
	pkg, version, rangeDir string
	files                  []string
	sourceDir              string
}

func applyPatches(set patchSet) error {
	pkg, version, rangeDir, files, sourceDir := set.pkg, set.version, set.rangeDir, set.files, set.sourceDir
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

// PatchFlagsFileName is the optional flags manifest inside a patch range
// directory. A directory containing only this file is still a valid
// selection: it contributes build flags without any patch application.
const PatchFlagsFileName = "flags.yaml"

// PatchManifest declares the build flags contributed by a patch range
// directory. Strings pass through unchanged so assembler-side
// placeholder resolution ({{prefix}}, {{source}}, {{dep:NAME}}) is
// preserved exactly as inline patch declarations resolve today.
type PatchManifest = domain.PatchManifest

// ParseManifest decodes a flags.yaml manifest from its raw bytes.
// Placeholder strings such as "{{dep:zlib}}" are preserved verbatim.
func ParseManifest(data []byte) (*PatchManifest, error) {
	var m PatchManifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse flags manifest: %w", err)
	}
	return &m, nil
}

// LoadManifest reads and parses the flags manifest at the given path.
// Returns nil (no error) when the file does not exist, so callers can
// probe for a manifest without treating absence as a failure.
func LoadManifest(path string) (*PatchManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read %s: %w", filepath.Base(path), err)
	}
	m, err := ParseManifest(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", filepath.Base(path), err)
	}
	return m, nil
}
