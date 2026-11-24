package disk

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/supanadit/phpv/internal/repository"
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
// DirRangeExpr translates a range-directory name into the version-range
// expression evaluated by vsn.MatchVersionRange. Directory names use
// a Windows-safe encoding of the range syntax:
//
//	any          → ""   (matches every version)
//	8.0.0-8.1.0  → ">=8.0.0,<8.1.0"   (lower inclusive, upper exclusive)
//	6.0.0+       → ">=6.0.0"          (open-ended)
//	6.9.7        → "=6.9.7"           (exact version)
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
				return repository.ApplyPatches(repository.PatchSet{Pkg: name, Version: version, RangeDir: dirName, Files: files, SourceDir: sourceDir})
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
	return repository.DirRangeExpr(rangeDir)
}

// entryMatches reports whether the directory's range name matches the
// built version.
func entryMatches(e PatchEntry, version string) bool {
	if e.RangeDir == repository.DirRangeAny {
		return true
	}
	return vsn.MatchVersionRange(repository.DirRangeExpr(e.RangeDir), version)
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
