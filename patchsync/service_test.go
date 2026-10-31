package patchsync

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// fakeRemote is an in-memory publication source.
type fakeRemote struct {
	tree  map[string]string // path → content (blob SHA derived from it)
	rev   string
	calls int
	fail  error
}

func (f *fakeRemote) ListTree(ctx context.Context) ([]RemoteFile, string, error) {
	f.calls++
	if f.fail != nil {
		return nil, "", f.fail
	}
	out := make([]RemoteFile, 0, len(f.tree))
	for p, c := range f.tree {
		out = append(out, RemoteFile{Path: p, BlobSHA: "sha-" + c})
	}
	return out, f.rev, nil
}

func (f *fakeRemote) FetchBlob(ctx context.Context, path, dest string) error {
	if f.fail != nil {
		return f.fail
	}
	return os.WriteFile(dest, []byte(f.tree[path]), 0o644)
}

func newService(t *testing.T, r Remote) (*Service, string) {
	t.Helper()
	root := filepath.Join(t.TempDir(), "patches")
	return NewService(r, root, time.Hour), root
}

func mustRead(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

func TestFirstSyncDownloadsEverything(t *testing.T) {
	fr := &fakeRemote{tree: map[string]string{
		"php/8.0.0-8.1.0/0001-intl-cxx17.patch":  "A",
		"php/7.0.0-7.1.0/0001-curl-cflags.patch": "B",
		"icu/58.0-59.0/0001-gcc15-compat.patch":  "C",
		"libzip/any/flags.yaml":                  "D",
	}, rev: "rev1"}
	svc, root := newService(t, fr)

	res, err := svc.Update(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Downloaded) != 4 {
		t.Fatalf("downloaded %d files, want 4", len(res.Downloaded))
	}
	if got := mustRead(t, filepath.Join(root, "php/8.0.0-8.1.0/0001-intl-cxx17.patch")); got != "A" {
		t.Errorf("content = %q", got)
	}
	if svc.recordedRevision() != "rev1" {
		t.Errorf("recorded revision = %q", svc.recordedRevision())
	}

	// Second sync at the same revision downloads nothing.
	res, err = svc.Update(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Downloaded) != 0 || !res.UpToDate {
		t.Fatalf("second sync = %+v, want up-to-date no-op", res)
	}
}

func TestDeltaDownloadsOnlyChangedFiles(t *testing.T) {
	fr := &fakeRemote{tree: map[string]string{
		"a/1.0-2.0/0001-a.patch": "A",
		"b/any/flags.yaml":       "B",
		"c/1.0-2.0/0001-c.patch": "C",
	}, rev: "rev1"}
	svc, root := newService(t, fr)
	if _, err := svc.Update(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	before := map[string]time.Time{}
	for _, p := range []string{"a/1.0-2.0/0001-a.patch", "b/any/flags.yaml", "c/1.0-2.0/0001-c.patch"} {
		fi, err := os.Stat(filepath.Join(root, filepath.FromSlash(p)))
		if err != nil {
			t.Fatal(err)
		}
		before[p] = fi.ModTime()
	}

	fr.tree["a/1.0-2.0/0001-a.patch"] = "A2"  // changed
	fr.tree["d/any/flags.yaml"] = "D"         // new
	delete(fr.tree, "c/1.0-2.0/0001-c.patch") // deleted upstream
	fr.rev = "rev2"

	res, err := svc.Update(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Downloaded) != 2 || len(res.Deleted) != 1 {
		t.Fatalf("delta = %+v, want 2 downloads and 1 deletion", res)
	}
	if _, err := os.Stat(filepath.Join(root, "c/1.0-2.0/0001-c.patch")); !os.IsNotExist(err) {
		t.Error("stale file was not removed from the managed tree")
	}
	if svc.recordedRevision() != "rev2" {
		t.Errorf("revision = %q, want rev2", svc.recordedRevision())
	}
	for p, mt := range before {
		if p == "a/1.0-2.0/0001-a.patch" || p == "c/1.0-2.0/0001-c.patch" {
			continue // downloaded, or legitimately deleted
		}
		fi, err := os.Stat(filepath.Join(root, filepath.FromSlash(p)))
		if err != nil {
			t.Errorf("untouched file %s disappeared: %v", p, err)
			continue
		}
		if !fi.ModTime().Equal(mt) {
			t.Errorf("untouched file %s was rewritten", p)
		}
	}
}

func TestInterruptedRunResumesCheaply(t *testing.T) {
	fr := &fakeRemote{tree: map[string]string{
		"a/1.0-2.0/0001-a.patch": "A",
		"b/1.0-2.0/0001-b.patch": "B",
	}, rev: "rev1"}
	svc, _ := newService(t, fr)
	if _, err := svc.Update(context.Background(), nil); err != nil {
		t.Fatal(err)
	}

	// Republish both files; the fetch fails partway.
	fr.tree["a/1.0-2.0/0001-a.patch"] = "A2"
	fr.tree["b/1.0-2.0/0001-b.patch"] = "B2"
	fr.rev = "rev2"
	fr.fail = errors.New("network gone")
	_, err := svc.Update(context.Background(), nil)
	if err == nil {
		t.Fatal("expected sync failure")
	}
	fr.fail = nil
	if svc.recordedRevision() != "rev1" {
		t.Fatalf("failed run advanced the revision to %q; manifest state must stay at last success", svc.recordedRevision())
	}

	// Retry downloads only what the manifest still misses (everything,
	// since the failed run completed nothing — but not more).
	res, err := svc.Update(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Downloaded) != 2 {
		t.Fatalf("retry downloaded %d files, want 2 (the manifest remainder)", len(res.Downloaded))
	}
	if svc.recordedRevision() != "rev2" {
		t.Errorf("revision = %q, want rev2", svc.recordedRevision())
	}
}

func TestScopedSyncTouchesOnlyMatchingFiles(t *testing.T) {
	fr := &fakeRemote{tree: map[string]string{
		"php/7.4.0-8.0.0/0001-a.patch": "A",
		"icu/58.0-60.0/0001-b.patch":   "B",
	}, rev: "rev1"}
	svc, root := newService(t, fr)
	if _, err := svc.Update(context.Background(), nil); err != nil {
		t.Fatal(err)
	}

	// Republish everything; scope to PHP 7.4.
	fr.tree["php/7.4.0-8.0.0/0001-a.patch"] = "A2"
	fr.tree["icu/58.0-60.0/0001-b.patch"] = "B2"
	fr.rev = "rev2"

	res, err := svc.Update(context.Background(), &Scope{Package: "php", Version: "7.4"})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Downloaded) != 1 || res.Downloaded[0] != "php/7.4.0-8.0.0/0001-a.patch" {
		t.Fatalf("scoped delta = %v, want only the php file", res.Downloaded)
	}
	if got := mustRead(t, filepath.Join(root, "icu/58.0-60.0/0001-b.patch")); got != "B" {
		t.Error("out-of-scope file was synchronized")
	}
	// A scoped sync must not claim the full revision.
	if svc.recordedRevision() != "rev1" {
		t.Errorf("scoped sync advanced revision to %q", svc.recordedRevision())
	}

	// Unscoped update now completes the remainder.
	res, err = svc.Update(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Downloaded) != 1 || res.Downloaded[0] != "icu/58.0-60.0/0001-b.patch" {
		t.Fatalf("follow-up delta = %v, want the icu file", res.Downloaded)
	}
}

func TestScopedSyncNoMatchesReportsEmpty(t *testing.T) {
	fr := &fakeRemote{tree: map[string]string{
		"php/8.0.0-8.1.0/0001-a.patch": "A",
	}, rev: "rev1"}
	svc, _ := newService(t, fr)
	if _, err := svc.Update(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	res, err := svc.Update(context.Background(), &Scope{Version: "8.9"})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Downloaded) != 0 || len(res.Deleted) != 0 {
		t.Fatalf("scope with no matches downloaded/deleted: %+v", res)
	}
}

func TestEnsureFreshLeaseAndExpiry(t *testing.T) {
	fr := &fakeRemote{tree: map[string]string{
		"php/8.0.0-8.1.0/0001-a.patch": "A",
	}, rev: "rev1"}
	svc, _ := newService(t, fr)
	if _, err := svc.Update(context.Background(), nil); err != nil {
		t.Fatal(err)
	}

	// Within the lease: no listing call.
	calls := fr.calls
	if _, err := svc.EnsureFresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	if fr.calls != calls {
		t.Fatal("lease window did not skip the remote check")
	}

	// Expired lease (machine slept past it): revision unchanged → just
	// a check, no downloads.
	if err := os.Remove(filepath.Join(svc.root, LastCheckFileName)); err != nil {
		t.Fatal(err)
	}
	res, err := svc.EnsureFresh(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !res.UpToDate || len(res.Downloaded) != 0 {
		t.Fatalf("EnsureFresh = %+v, want an up-to-date check", res)
	}
	if fr.calls != calls+1 {
		t.Fatal("expired lease did not trigger a revision check")
	}

	// Expired lease + new revision: delta sync happens.
	fr.tree["php/8.0.0-8.1.0/0001-a.patch"] = "A2"
	fr.rev = "rev2"
	if err := os.Remove(filepath.Join(svc.root, LastCheckFileName)); err != nil {
		t.Fatal(err)
	}
	res, err = svc.EnsureFresh(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Downloaded) != 1 {
		t.Fatalf("EnsureFresh after republish = %+v, want 1 download", res)
	}
}

func TestSyncNeverTouchesLocalTree(t *testing.T) {
	remote := &fakeRemote{tree: map[string]string{
		"php/8.0.0-8.1.0/0001-a.patch": "A",
	}, rev: "rev1"}
	svc, root := newService(t, remote)
	localRoot := filepath.Join(filepath.Dir(root), "patches-local")
	hotfix := filepath.Join(localRoot, "oniguruma", "6.9.7", "0001-hotfix.patch")
	if err := os.MkdirAll(filepath.Dir(hotfix), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(hotfix, []byte("user work"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Republished (renamed) range dir: old one removed, new one synced;
	// the local tree is untouched.
	remote.tree = map[string]string{"php/8.0.0-8.2.0/0001-a.patch": "A2"}
	remote.rev = "rev2"
	if _, err := svc.Update(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "php/8.0.0-8.1.0/0001-a.patch")); !os.IsNotExist(err) {
		t.Error("stale range directory was left alongside the renamed one")
	}
	if got := mustRead(t, hotfix); got != "user work" {
		t.Error("user-local tree was modified by sync")
	}
}

func TestUpdateOfflineFailsLoudly(t *testing.T) {
	fr := &fakeRemote{tree: map[string]string{}, rev: "rev1", fail: errors.New("dial: no route to host")}
	svc, _ := newService(t, fr)
	_, err := svc.Update(context.Background(), nil)
	if err == nil || !errors.Is(err, fr.fail) && err.Error() == "" {
		t.Fatalf("explicit update must fail non-zero with a clear message, got %v", err)
	}
}
