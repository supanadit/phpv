package assembler

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/supanadit/phpv/patchsync"
)

type fakePatchRemote struct {
	tree map[string]string
	rev  string
	fail error
}

func (f *fakePatchRemote) ListTree(ctx context.Context) ([]patchsync.RemoteFile, string, error) {
	if f.fail != nil {
		return nil, "", f.fail
	}
	out := make([]patchsync.RemoteFile, 0, len(f.tree))
	for p, c := range f.tree {
		out = append(out, patchsync.RemoteFile{Path: p, BlobSHA: "sha-" + c})
	}
	return out, f.rev, nil
}

func (f *fakePatchRemote) FetchBlob(ctx context.Context, path, dest string) error {
	if f.fail != nil {
		return f.fail
	}
	return nil
}

func TestSyncPatchesWarnAndContinueOnFailure(t *testing.T) {
	svc := &Service{patches: patchsync.NewService(
		&fakePatchRemote{fail: errors.New("dial: connection refused")},
		t.TempDir(), time.Hour)}

	var events []string
	svc.syncPatches(context.Background(), func(stage, msg string) {
		events = append(events, stage+": "+msg)
	})
	if len(events) != 1 {
		t.Fatalf("events = %v, want exactly the warning", events)
	}
	if events[0][:4] != "warn" {
		t.Errorf("event = %q, want a warn event", events[0])
	}
}

func TestSyncPatchesSyncsDeltaBeforeBuild(t *testing.T) {
	svc := &Service{patches: patchsync.NewService(
		&fakePatchRemote{tree: map[string]string{
			"php/8.0.0-8.1.0/0001-intl-cxx17.patch": "A",
		}, rev: "rev1"},
		t.TempDir(), time.Hour)}

	var events []string
	svc.syncPatches(context.Background(), func(stage, msg string) {
		events = append(events, stage+": "+msg)
	})
	if len(events) != 1 || events[0][:10] != "patch-sync" {
		t.Fatalf("events = %v, want a patch-sync event", events)
	}
}

func TestSyncPatchesSkipsWithinLease(t *testing.T) {
	root := t.TempDir()
	svc := &Service{patches: patchsync.NewService(
		&fakePatchRemote{tree: map[string]string{"a/1.0-2.0/0001-a.patch": "A"}, rev: "rev1"},
		root, time.Hour)}
	// Prime the lease with a successful full sync.
	if _, err := svc.patches.Update(context.Background(), nil); err != nil {
		t.Fatal(err)
	}

	var events []string
	svc.syncPatches(context.Background(), func(stage, msg string) {
		events = append(events, stage+": "+msg)
	})
	if len(events) != 1 || events[0][:10] != "patch-sync" {
		t.Fatalf("events = %v, want the lease-skip report", events)
	}
}

func TestSyncPatchesNilServiceIsNoop(t *testing.T) {
	svc := &Service{}
	var events []string
	svc.syncPatches(context.Background(), func(stage, msg string) {
		events = append(events, stage+": "+msg)
	})
	if len(events) != 0 {
		t.Fatalf("events = %v, want none", events)
	}
}
