package domain

// ConfigureFlagRule describes version-gated configure flags for a package.
type ConfigureFlagRule struct {
	MinVer string
	MaxVer string
	Flags  []string
	Needs  string
}
