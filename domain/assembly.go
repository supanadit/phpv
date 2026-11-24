package domain

// Toolchain is the resolved compiler used to build a package.
type Toolchain struct {
	Name, CC, CXX, Hint string
}

// DownloadResult holds the outcome of downloading and extracting one package.
type DownloadResult struct {
	Name       string
	Version    string
	Downloaded bool
	Extracted  bool
	Err        error
}

// AssemblerResult holds the outcome of a full assembly run.
type AssemblerResult struct {
	DownloadResults  []DownloadResult
	Version          string
	Prefix           string
	Env              map[string]string
	AlreadyInstalled bool
}

// AssembleSpec groups the inputs to a full assembly run.
type AssembleSpec struct {
	Name, Version string
	Static        bool
	Extensions    []string
	Verbose       bool
	Progress      func(stage, message string)
	SystemPkgs    map[string]SystemPackage
	Jobs          int
	Force         bool
	TC            Toolchain
}

// ApacheModuleSpec groups the inputs to building a mod_php module.
type ApacheModuleSpec struct {
	PHPVersion, HTTPDPrefix string
	Verbose                 bool
	Jobs                    int
	TC                      Toolchain
	Progress                func(stage, message string)
}

// FPMSpec groups the inputs to ensuring a php-fpm runtime.
type FPMSpec struct {
	PHPVersion, StagedFrom string
	Verbose                bool
	Jobs                   int
	TC                     Toolchain
	Progress               func(stage, message string)
}

// ServerAssembleSpec groups the inputs to assembling a webserver package.
type ServerAssembleSpec struct {
	Name, Version string
	Verbose       bool
	Jobs          int
	TC            Toolchain
	Progress      func(stage, message string)
}
