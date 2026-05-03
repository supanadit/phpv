package domain

type BuildStrategy string

const (
	StrategyCMake         BuildStrategy = "cmake"
	StrategyConfigureMake BuildStrategy = "configure_make"
	StrategyMakeOnly      BuildStrategy = "make_only"
	StrategyAutogen       BuildStrategy = "autogen"
)

// ForgeConfig holds configuration for building a package with autotools/cmake.
type ForgeConfig struct {
	Name            string   // Package name (e.g., "php", "openssl", "icu")
	Version         string   // Package version (e.g., "8.0.30", "3.2.0")
	PHPVersion      string   // PHP version being built (for PHP package only)
	Prefix          string   // Installation prefix (--prefix value)
	ConfigureFlags  []string // Package-specific configure flags
	Env             map[string]string // Additional environment variables
	Jobs            int      // Number of parallel make jobs
	Strategy        BuildStrategy // Build strategy (configure_make, cmake, etc.)
	CPPFLAGS        []string // Preprocessor flags (-I, -D)
	LDFLAGS         []string // Linker flags (-L, -l)
	LD_LIBRARY_PATH []string // Runtime library paths
	CC              string   // C compiler path
	CFLAGS          []string // C compiler flags
	CStd            string   // C standard flag (e.g., "-std=gnu11"), set from flagresolver
	CXX             string   // C++ compiler path
	CXXFLAGS        []string // C++ compiler flags
	CXXStd          string   // C++ standard flag (e.g., "-std=gnu++17"), set from flagresolver
	PkgConfigPaths  []string // pkg-config search paths
	Libs            []string // Additional libraries
	Verbose         bool     // Enable verbose build output
}
