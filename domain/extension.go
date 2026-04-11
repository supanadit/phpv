package domain

type ExtensionType string

const (
	ExtensionTypeBundled ExtensionType = "bundled"
	ExtensionTypePECL    ExtensionType = "pecl"
)

type Extension struct {
	Name        string
	Type        ExtensionType
	Version     string
	ArchivePath string
	Config      map[string]string
}
