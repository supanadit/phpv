package domain

type ForgeConfig struct {
	Name           string
	Version        string
	Prefix         string
	ConfigureFlags []string
	Jobs           int
}
