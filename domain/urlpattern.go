package domain

type URLPattern struct {
	Name          string
	Type          string // "source" or "binary"
	OS            string // target OS, "" = all
	Arch          string // target arch, "" = all
	Constraint    func(v *Version) bool
	Template      string
	Fallbacks     []string
	Checksum      string // SHA256 checksum for verification (optional)
	ExtensionFunc func(v *Version) string
}
