package domain

type ConfigureFlag struct {
	Name  string
	Flags []string
}

type FlagResolverRepository interface {
	GetConfigureFlags(name string) []string
	GetPHPConfigureFlags(phpVersion string, extensions []string) []string
}
