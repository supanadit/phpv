// Package patches implements the file-based patcher repository: patches
// live as unified diff files in a versioned directory tree
// (<root>/<package>/<version-range>/<ordinal>-<slug>.patch) with optional
// flags.yaml manifests, merged from two on-disk sources (synced and
// user-local) with per-directory precedence. No patch content is embedded
// in the binary; everything is read from the injected roots.
package patches

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/patcher"
)

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
// expression evaluated by domain.MatchVersionRange. Directory names use
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
type root struct {
	path  string
	label string
}

// Repository is the file-based patcher.PatcherRepository.
type Repository struct {
	roots []root
}

// NewRepository creates a patcher.PatcherRepository reading patches from
// the given on-disk trees in precedence order: localRoot (user-local)
// shadows syncedRoot per directory. Missing roots are simply empty.
func NewRepository(syncedRoot, localRoot string, opts ...Option) patcher.PatcherRepository {
	r := &Repository{}
	for _, o := range opts {
		o(r)
	}
	// User-local patches take precedence: they are consulted first.
	if localRoot != "" {
		r.roots = append(r.roots, root{path: localRoot, label: SourceLocal})
	}
	if syncedRoot != "" {
		r.roots = append(r.roots, root{path: syncedRoot, label: SourceSynced})
	}
	return r
}

// Option customizes the repository.
type Option func(*Repository)

// Entry is one selected patch-tree directory: its patch files (in
// application order), optional flags manifest, and the source tree it
// came from.
type Entry struct {
	// DirPath is the directory path relative to its source root,
	// e.g. "php/>=8.0.0,<8.1.0".
	DirPath string
	// Package is the directory's package name.
	Package string
	// RangeDir is the directory's version-range name (or "any").
	RangeDir string
	// Source is the tree label this directory was claimed by
	// (SourceLocal or SourceSynced).
	Source string
	// PatchFiles are absolute paths to the .patch files in
	// ascending ordinal order.
	PatchFiles []string
	// Manifest is the parsed flags.yaml, or nil when absent.
	Manifest *Manifest
}

// EntriesFor returns all patch-tree directories matching (name, version)
// in ordinal order, labeled by their owning source tree. It is the
// observability view behind PatchesFor (used by `phpv patch list`).
func (r *Repository) EntriesFor(name string, version string) []Entry {
	owned := make(map[string]Entry)
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
	entries := make([]Entry, 0, len(owned))
	for _, e := range owned {
		if !entryMatches(e, version) {
			continue
		}
		entries = append(entries, e)
	}
	sortEntries(entries)
	return entries
}

// PatchesFor implements patcher.PatcherRepository: it resolves the
// selected entries for (name, version) and converts them into Patch
// structs whose Apply runs `patch -p1` against the located source root.
func (r *Repository) PatchesFor(name string, version string) []patcher.Patch {
	entries := r.EntriesFor(name, version)
	if len(entries) == 0 {
		return nil
	}
	patches := make([]patcher.Patch, 0, len(entries))
	for _, e := range entries {
		p := patcher.Patch{
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
				return applyPatches(name, version, dirName, files, sourceDir)
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
func entryMatches(e Entry, version string) bool {
	if e.RangeDir == anyDirName {
		return true
	}
	return domain.MatchVersionRange(DirRangeExpr(e.RangeDir), version)
}

// sortEntries orders selected directories by their first patch file's
// ordinal filename, then by range name for stability. Flags-only entries
// sort by range name.
func sortEntries(entries []Entry) {
	sort.SliceStable(entries, func(i, j int) bool {
		return firstOrdinal(entries[i]) < firstOrdinal(entries[j])
	})
}

func firstOrdinal(e Entry) string {
	if len(e.PatchFiles) > 0 {
		return filepath.Base(e.PatchFiles[0])
	}
	return e.RangeDir
}

// readPackageDir reads every range directory under <pkgDir>, skipping
// per-directory entries that fail to read (the other sources or a later
// run still work).
func readPackageDir(pkgDir string, label string) ([]Entry, error) {
	dirs, err := os.ReadDir(pkgDir)
	if err != nil {
		return nil, err
	}
	entries := make([]Entry, 0, len(dirs))
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
		manifest, err := LoadManifest(filepath.Join(dirPath, ManifestFileName))
		if err != nil {
			continue
		}
		rel, err := filepath.Rel(filepath.Dir(pkgDir), dirPath)
		if err != nil {
			continue
		}
		entries = append(entries, Entry{
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
