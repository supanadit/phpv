package domain

// Registry represents a single downloadable package entry.
//
// OS indicates which operating system the entry targets. Standard values
// follow Go's runtime.GOOS convention (e.g., "linux", "darwin", "windows").
// When the entry is OS-agnostic (e.g., source code), OS is set to "all".
type Registry struct {
	Name          string
	Type          string
	URL           string
	Version       string
	OS            string
	ChecksumType  string
	ChecksumValue string
}