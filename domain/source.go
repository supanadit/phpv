package domain

const (
	SourceTypeBinary = "binary"
	SourceTypeSource = "source"

	OSLinux   = "linux"
	OSDarwin  = "darwin"
	OSWindows = "windows"

	ArchX86_64  = "x86_64"
	ArchArm64   = "arm64"
	ArchAarch64 = "aarch64"
)

type Source struct {
	Name        string
	Version     string
	URL         string
	Type        string // "source" or "binary"
	OS          string // target OS, "" = all
	Arch        string // target arch, "" = all
	PackageType string // "tar.gz", "tar.xz", "zip", "deb", "rpm", "dmg", "exe"
}

type Version struct {
	Major  int
	Minor  int
	Patch  int
	Suffix string
	Raw    string
}

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
