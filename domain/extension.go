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

type ExtensionConflictError struct {
	Extension   string
	Conflicting []string
}

func (e *ExtensionConflictError) Error() string {
	return "extension " + e.Extension + " conflicts with: " + joinStrings(e.Conflicting)
}

func joinStrings(strs []string) string {
	result := ""
	for i, s := range strs {
		if i > 0 {
			result += ", "
		}
		result += s
	}
	return result
}

type UnknownExtensionError struct {
	Extension string
}

func (e *UnknownExtensionError) Error() string {
	return "unknown extension: " + e.Extension
}
