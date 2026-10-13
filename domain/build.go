package domain

// FlagRule defines how a configure flag should be generated for a PHP version range.
type FlagRule struct {
	MinVersion string
	MaxVersion string
	Extension  string
	Flag       string
}

// CompilerFlagDef describes a compiler flag with its version requirements and purpose.
type CompilerFlagDef struct {
	Flag    string
	Needs   string
	Purpose string
}

// CompilerRule defines the compiler flags derived from a PHP version range:
// the language standard plus any flags the code of that era needs from every
// compiler (independent of which compiler is selected).
type CompilerRule struct {
	MinVersion string
	MaxVersion string
	CStd       string
	CXXStd     string
	CFlags     []string
}
