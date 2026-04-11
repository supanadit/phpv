package domain

type ConfigureFlag struct {
	Name  string
	Flags []string
}

type FlagResolverRepository interface {
	GetConfigureFlags(name string, version string) []string
	GetPHPConfigureFlags(phpVersion string, extensions []string) []string
	ValidateExtensions(extensions []string, phpVersion string) ([]string, error)
	CheckExtensionConflicts(extensions []string) ([]string, [][]string)
	GetExtensionDependency(ext string) (string, bool)
	GetExtensionDependencyWithVersion(ext, phpVersion string) (string, string, bool)
}
