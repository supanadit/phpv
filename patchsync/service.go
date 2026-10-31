// Package patchsync keeps the managed patch tree at PHPV_ROOT/patches/
// current with the publication source (this repository's main branch).
// Sync is delta-based: the local manifest (path → blob SHA) is diffed
// against the remote tree listing at the latest revision, and only
// added/changed files are downloaded. The user-local tree
// (PHPV_ROOT/patches-local/) is never touched by sync.
package patchsync

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/internal/repository/patches"
)

const (
	// ManifestFileName records the synced file set (path → blob SHA).
	ManifestFileName = ".manifest.json"
	// RevisionFileName records the remote revision of the last fully
	// applied sync.
	RevisionFileName = ".revision"
	// LastCheckFileName records the last successful revision check.
	LastCheckFileName = ".lastcheck"
	// DefaultLease is the install-path stale-lease window.
	DefaultLease = 6 * time.Hour
)

// RemoteFile is one file in the remote patch tree.
type RemoteFile struct {
	// Path is the tree-relative slash path, e.g.
	// "php/8.0.0-8.1.0/0001-intl-cxx17.patch".
	Path string
	// BlobSHA is the git blob SHA pinning the file's content.
	BlobSHA string
}

// Remote is the publication source of the patch tree.
type Remote interface {
	// ListTree returns every file under patches/ at the latest revision,
	// plus the revision identifier.
	ListTree(ctx context.Context) ([]RemoteFile, string, error)
	// FetchBlob downloads one file's content to dest.
	FetchBlob(ctx context.Context, path, dest string) error
}

// Scope limits a sync to [package] [version]; a nil scope syncs the
// whole tree.
type Scope struct {
	Package string
	Version string
}

// Result reports what a sync did.
type Result struct {
	// Revision is the revision recorded after the sync (unchanged for
	// scoped syncs, which must not claim the full tree is current).
	Revision string
	// Downloaded are the tree-relative paths written.
	Downloaded []string
	// Deleted are the tree-relative paths removed.
	Deleted []string
	// UpToDate reports that the local tree already matches the remote
	// revision and nothing was downloaded.
	UpToDate bool
	// SkippedCheck reports that the stale-lease window suppressed the
	// remote check entirely (install path only).
	SkippedCheck bool
}

// Status is the observable state of the managed tree.
type Status struct {
	Revision  string
	RemoteRev string
	LastCheck time.Time
	UpdateDue bool // remote revision differs from recorded revision
	Files     int
	InLease   bool
}

type Service struct {
	remote Remote
	root   string
	lease  time.Duration
}

// NewService creates a patch sync service over the managed tree at root,
// fetched from remote, with the given stale-lease window (0 = default).
func NewService(remote Remote, root string, lease time.Duration) *Service {
	if lease <= 0 {
		lease = leaseFromEnv(DefaultLease)
	}
	return &Service{remote: remote, root: root, lease: lease}
}

func leaseFromEnv(def time.Duration) time.Duration {
	if v := os.Getenv("PHPV_PATCH_LEASE"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}

// Update syncs the managed tree. With a scope, only files under range
// directories matching the scope are downloaded or deleted, and the
// recorded revision is left untouched (files outside the scope remain
// governed by their previous syncs). Explicit updates ignore the
// stale-lease.
func (s *Service) Update(ctx context.Context, scope *Scope) (Result, error) {
	files, revision, err := s.remote.ListTree(ctx)
	if err != nil {
		return Result{}, fmt.Errorf("list remote patch tree: %w", err)
	}
	manifest := s.loadManifest()

	var (
		download []RemoteFile
		toDelete []RemoteFile
	)
	for _, f := range files {
		if scope != nil && !scopeMatches(f.Path, *scope) {
			continue
		}
		if sha, ok := manifest[f.Path]; !ok || sha != f.BlobSHA {
			download = append(download, f)
		}
	}
	// Remote deletions: manifest entries absent from the remote listing.
	remoteSet := make(map[string]string, len(files))
	for _, f := range files {
		remoteSet[f.Path] = f.BlobSHA
	}
	for path := range manifest {
		if _, ok := remoteSet[path]; !ok {
			if scope == nil || scopeMatches(path, *scope) {
				toDelete = append(toDelete, RemoteFile{Path: path})
			}
		}
	}

	res := Result{Revision: revision}
	for _, f := range download {
		if err := s.stageBlob(ctx, f); err != nil {
			return Result{}, err
		}
		res.Downloaded = append(res.Downloaded, f.Path)
		manifest[f.Path] = f.BlobSHA
	}
	for _, f := range toDelete {
		if err := os.Remove(filepath.Join(s.root, filepath.FromSlash(f.Path))); err != nil && !os.IsNotExist(err) {
			return Result{}, fmt.Errorf("remove %s: %w", f.Path, err)
		}
		res.Deleted = append(res.Deleted, f.Path)
		delete(manifest, f.Path)
	}

	if scope == nil {
		// Full sync: claim the remote revision atomically with the
		// manifest. A scoped sync must not (it has not synced
		// everything the revision covers).
		if err := s.commit(manifest, revision); err != nil {
			return Result{}, err
		}
	} else if err := s.saveManifest(manifest); err != nil {
		return Result{}, err
	}
	res.UpToDate = len(res.Downloaded) == 0 && len(res.Deleted) == 0 && scope == nil && revision == s.recordedRevision()
	return res, nil
}

// EnsureFresh is the install-path hook: within the stale-lease window it
// does nothing; otherwise it checks the remote revision and syncs only
// on change. Failures are returned; the assembler decides whether to
// warn-and-continue.
func (s *Service) EnsureFresh(ctx context.Context) (Result, error) {
	if lc, ok := s.lastCheck(); ok && time.Since(lc) < s.lease {
		return Result{SkippedCheck: true}, nil
	}
	_, revision, err := s.remote.ListTree(ctx)
	if err != nil {
		return Result{}, err
	}
	if revision == s.recordedRevision() {
		// Still current: just refresh the lease timestamp.
		if err := s.touchLastCheck(); err != nil {
			return Result{Revision: revision}, err
		}
		return Result{Revision: revision, UpToDate: true}, nil
	}
	return s.Update(ctx, nil)
}

// Status reports the managed tree state, checking the remote for
// available updates.
func (s *Service) Status(ctx context.Context) (Status, error) {
	st := Status{}
	manifest := s.loadManifest()
	st.Files = len(manifest)
	st.Revision = s.recordedRevision()
	if lc, ok := s.lastCheck(); ok {
		st.LastCheck = lc
		st.InLease = time.Since(lc) < s.lease
	}
	_, remoteRev, err := s.remote.ListTree(ctx)
	if err != nil {
		return st, fmt.Errorf("list remote patch tree: %w", err)
	}
	st.RemoteRev = remoteRev
	st.UpdateDue = remoteRev != st.Revision
	return st, nil
}

// scopeMatches reports whether a tree path falls under the scope. Paths
// are "<package>/<range-dir>/<file>"; a bare-version scope targets the
// php package.
func scopeMatches(path string, sc Scope) bool {
	parts := strings.Split(path, "/")
	if len(parts) < 3 {
		return false
	}
	pkg, rangeDir := parts[0], parts[1]
	if sc.Package != "" && pkg != sc.Package {
		return false
	}
	if sc.Version == "" {
		return true
	}
	return domain.MatchVersionRange(patches.DirRangeExpr(rangeDir), sc.Version)
}

// stageBlob downloads one file into a sibling temp file and renames it
// into place, so an interrupted run never leaves a partial file.
func (s *Service) stageBlob(ctx context.Context, f RemoteFile) error {
	dest := filepath.Join(s.root, filepath.FromSlash(f.Path))
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return fmt.Errorf("prepare %s: %w", f.Path, err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(dest), ".stage-*")
	if err != nil {
		return fmt.Errorf("stage %s: %w", f.Path, err)
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := s.remote.FetchBlob(ctx, f.Path, tmpPath); err != nil {
		return fmt.Errorf("fetch %s: %w", f.Path, err)
	}
	if err := os.Rename(tmpPath, dest); err != nil {
		return fmt.Errorf("move %s into place: %w", f.Path, err)
	}
	return nil
}

// commit writes manifest, revision, and lastcheck only after the delta
// completes, each via stage-then-rename.
func (s *Service) commit(manifest map[string]string, revision string) error {
	if err := s.saveManifest(manifest); err != nil {
		return err
	}
	if err := writeAtomic(filepath.Join(s.root, RevisionFileName), []byte(revision)); err != nil {
		return err
	}
	return s.touchLastCheck()
}

func (s *Service) saveManifest(manifest map[string]string) error {
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	return writeAtomic(filepath.Join(s.root, ManifestFileName), data)
}

func (s *Service) loadManifest() map[string]string {
	m := make(map[string]string)
	data, err := os.ReadFile(filepath.Join(s.root, ManifestFileName))
	if err != nil {
		return m
	}
	_ = json.Unmarshal(data, &m)
	return m
}

func (s *Service) recordedRevision() string {
	data, err := os.ReadFile(filepath.Join(s.root, RevisionFileName))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func (s *Service) lastCheck() (time.Time, bool) {
	data, err := os.ReadFile(filepath.Join(s.root, LastCheckFileName))
	if err != nil {
		return time.Time{}, false
	}
	t, err := time.Parse(time.RFC3339, strings.TrimSpace(string(data)))
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

func (s *Service) touchLastCheck() error {
	return writeAtomic(filepath.Join(s.root, LastCheckFileName),
		[]byte(time.Now().Format(time.RFC3339)))
}

// writeAtomic writes data to path via a sibling temp file and rename.
func writeAtomic(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".meta-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}
