package domain

import "time"

type FileInfo struct {
	URL         string
	Size        int64
	ContentType string
	Exists      bool
}

type Download struct {
	ID             string
	URL            string
	Destination    string
	FilePath       string
	Status         string
	Size           int64
	DownloadedSize int64
	Checksum       string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

const (
	DownloadStatusPending      = "pending"
	DownloadStatusDownloading  = "downloading"
	DownloadStatusCompleted    = "completed"
	DownloadStatusFailed       = "failed"
	DownloadStatusNotFound     = "not_found"
	DownloadStatusUnauthorized = "unauthorized"
	DownloadStatusPartial      = "partial"
)
