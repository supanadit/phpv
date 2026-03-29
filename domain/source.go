package domain

type Source struct {
	Name    string
	Version string
	URL     string
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
	Constraint    func(v *Version) bool
	Template      string
	ExtensionFunc func(v *Version) string
}
