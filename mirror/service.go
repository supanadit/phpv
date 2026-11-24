// Package mirror keeps the managed patch tree at PHPV_ROOT/patches/ current
// with its publication source; PHPV_ROOT/patches-local/ is never touched.
package mirror

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/supanadit/phpv/domain"
	vsn "github.com/supanadit/phpv/pkg/version"
)

// Sync metadata files (manifest, revision, last check) and the lease window.
const (
	ManifestFileName  = domain.ManifestFileName
	RevisionFileName  = domain.RevisionFileName
	LastCheckFileName = domain.LastCheckFileName
	DefaultLease      = domain.DefaultLease
)

// RemoteFile is one file in the remote patch tree: path and content SHA.
type RemoteFile = domain.RemoteFile

// Remote is the publication source of the patch tree.
//
//go:generate mockery --name Remote
type Remote interface {
	// ListTree returns every file under patches/ at the latest revision, plus the revision.
	ListTree(ctx context.Context) ([]RemoteFile, string, error)
	// FetchBlob downloads one file's content to dest.
	FetchBlob(ctx context.Context, path, dest string) error
}

// Scope limits a sync to [package] [version]; a nil scope syncs the whole tree.
type Scope = domain.PatchSyncScope

// Result reports what a sync did. Revision is unchanged for scoped syncs; UpToDate
// means the local tree already matched, SkippedCheck that the lease skipped the check.
type Result = domain.PatchSyncResult

// Status is the observable state of the managed tree.
type Status = domain.PatchSyncStatus

type Service struct {
	remote Remote
	root   string
	lease  time.Duration
}

// NewService creates a sync service over the managed tree at root; 0 lease = default.
func NewService(remote Remote, root string, lease time.Duration) *Service {
	if lease <= 0 {
		lease = DefaultLease
		if v := os.Getenv("PHPV_PATCH_LEASE"); v != "" {
			if d, err := time.ParseDuration(v); err == nil {
				lease = d
			}
		}
	}
	return &Service{remote: remote, root: root, lease: lease}
}

// Update syncs the managed tree; a scope restricts touches and keeps the revision.
func (s *Service) Update(ctx context.Context, scope *Scope) (Result, error) {
	files, revision, err := s.remote.ListTree(ctx)
	if err != nil {
		return Result{}, fmt.Errorf("list remote patch tree: %w", err)
	}
	manifest := s.loadManifest()
	var download, toDelete []RemoteFile
	remoteSet := make(map[string]string, len(files))
	for _, f := range files {
		remoteSet[f.Path] = f.BlobSHA
		if scope != nil && !scopeMatches(f.Path, *scope) {
			continue
		}
		if sha, ok := manifest[f.Path]; !ok || sha != f.BlobSHA {
			download = append(download, f)
		}
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
		// Full sync: claim the revision atomically with the manifest. A scoped
		// sync must not — it has not synced everything the revision covers.
		if err := s.saveManifest(manifest); err != nil {
			return Result{}, err
		}
		if err := writeAtomic(filepath.Join(s.root, RevisionFileName), []byte(revision)); err != nil {
			return Result{}, err
		}
		if err := s.touchLastCheck(); err != nil {
			return Result{}, err
		}
	} else if err := s.saveManifest(manifest); err != nil {
		return Result{}, err
	}
	res.UpToDate = len(res.Downloaded) == 0 && len(res.Deleted) == 0 && scope == nil && revision == s.recordedRevision()
	return res, nil
}

// EnsureFresh is the install-path hook: within the lease window it does nothing;
// otherwise it checks the remote revision and syncs only on change.
func (s *Service) EnsureFresh(ctx context.Context) (Result, error) {
	if lc, ok := s.lastCheck(); ok && time.Since(lc) < s.lease {
		return Result{SkippedCheck: true}, nil
	}
	_, revision, err := s.remote.ListTree(ctx)
	if err != nil {
		return Result{}, err
	}
	if revision == s.recordedRevision() {
		if err := s.touchLastCheck(); err != nil {
			return Result{Revision: revision}, err
		}
		return Result{Revision: revision, UpToDate: true}, nil
	}
	return s.Update(ctx, nil)
}

// Status reports the managed tree state, checking the remote for updates.
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

// scopeMatches reports whether "<package>/<range-dir>/<file>" falls under the scope.
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
	return vsn.MatchVersionRange(dirRangeExpr(rangeDir), sc.Version)
}

// dirRangeExpr maps a range-directory name to a version-range expression.
func dirRangeExpr(dir string) string {
	if dir == "any" || dir == "" {
		return ""
	}
	if strings.HasSuffix(dir, "+") {
		return ">=" + strings.TrimSuffix(dir, "+")
	}
	if a, b, ok := strings.Cut(dir, "-"); ok && a != "" && b != "" {
		if isPlainVersion(a) && isPlainVersion(b) {
			return ">=" + a + ",<" + b
		}
	}
	return "=" + dir
}

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

// stageBlob downloads into a sibling temp file and renames, so no partial file remains.
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

func (s *Service) saveManifest(manifest map[string]string) error {
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	return writeAtomic(filepath.Join(s.root, ManifestFileName), data)
}
func (s *Service) loadManifest() map[string]string {
	m := make(map[string]string)
	_ = json.Unmarshal([]byte(s.readMeta(ManifestFileName)), &m)
	return m
}

// readMeta returns the trimmed content of a sync metadata file, or "".
func (s *Service) readMeta(name string) string {
	data, err := os.ReadFile(filepath.Join(s.root, name))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func (s *Service) recordedRevision() string { return s.readMeta(RevisionFileName) }

func (s *Service) lastCheck() (time.Time, bool) {
	t, err := time.Parse(time.RFC3339, s.readMeta(LastCheckFileName))
	return t, err == nil
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
