package domain

import "time"

type Download struct {
	ID          string
	URL         string
	Destination string
	FilePath    string
	Status      string
	Size        int64
	Checksum    string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

const (
	DownloadStatusPending     = "pending"
	DownloadStatusDownloading = "downloading"
	DownloadStatusCompleted   = "completed"
	DownloadStatusFailed      = "failed"
)
