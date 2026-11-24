package domain

// BuildPlan consolidates everything needed to build a package.
type BuildPlan struct {
	Deps             []Dependency
	ConfigureFlags   []string
	CFlags           []string
	CompilerFlags    []string
	CXXCompilerFlags []string
	Warnings         []string
}
