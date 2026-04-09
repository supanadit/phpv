package domain

type PharConfig struct {
	Name        string
	Version     string
	URL         string
	Destination string
	Checksum    string
}

type PharResult struct {
	Name    string
	Version string
	Path    string
	Updated bool
}
