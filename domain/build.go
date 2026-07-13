package domain

// FlagRule defines how a configure flag should be generated for a PHP version range.
type FlagRule struct {
	MinVersion string
	MaxVersion string
	Extension  string
	Flag       string
}

// CompilerRule defines C/C++ compiler standard flags for a PHP version range.
type CompilerRule struct {
	MinVersion string
	MaxVersion string
	CStd       string
	CXXStd     string
	CFlags     []string
}
