package domain

// Release is the latest published release of the application.
type Release struct {
	TagName string
	Assets  []ReleaseAsset
}

// ReleaseAsset is one downloadable artifact of a release.
type ReleaseAsset struct {
	Name        string
	DownloadURL string
	Size        int64
}
