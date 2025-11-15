package domain

type SourceType string

const (
	SourceTypeGitHub   SourceType = "github"
	SourceTypeOfficial SourceType = "official"
)

type DownloadSource struct {
	Type SourceType
	URL  string
}
