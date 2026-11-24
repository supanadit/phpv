package domain

import "time"

// RemoteFile is one file in a remote patch tree: path and content SHA.
type RemoteFile struct{ Path, BlobSHA string }

// PatchSyncScope limits a patch sync to one package/version; a zero scope syncs the whole tree.
type PatchSyncScope struct{ Package, Version string }

// PatchSyncResult reports what a patch sync did.
type PatchSyncResult struct {
	Revision               string
	Downloaded             []string
	Deleted                []string
	UpToDate, SkippedCheck bool
}

// PatchSyncStatus is the observable state of the managed patch tree.
type PatchSyncStatus struct {
	Revision, RemoteRev string
	LastCheck           time.Time
	UpdateDue           bool
	Files               int
	InLease             bool
}

// Patch-tree sync metadata filenames and the default freshness lease.
const (
	ManifestFileName  = ".manifest.json"
	RevisionFileName  = ".revision"
	LastCheckFileName = ".lastcheck"
	DefaultLease      = 6 * time.Hour
)
