package domain

type BuildStrategy string

const (
	StrategyCMake         BuildStrategy = "cmake"
	StrategyConfigureMake BuildStrategy = "configure_make"
	StrategyMakeOnly      BuildStrategy = "make_only"
	StrategyAutogen       BuildStrategy = "autogen"
)

type ForgeConfig struct {
	Name            string
	Version         string
	PHPVersion      string
	Prefix          string
	ConfigureFlags  []string
	Env             map[string]string
	Jobs            int
	Strategy        BuildStrategy
	CPPFLAGS        []string
	LDFLAGS         []string
	LD_LIBRARY_PATH []string
	CC              string
	CFLAGS          []string
	CXX             string
	CXXFLAGS        []string
	PkgConfigPaths  []string
	Libs            []string
	Verbose         bool
}
