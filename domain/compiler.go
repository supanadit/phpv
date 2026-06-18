package domain

// CompilerFlagDef describes a compiler flag with its version requirements and purpose.
type CompilerFlagDef struct {
	Flag    string // e.g., "-fno-strict-function-pointer-casts"
	Needs   string // version constraint on the compiler, e.g., ">=gcc5.0 <gcc14" or "" (always)
	Purpose string // human-readable explanation, e.g., "old PHP code casts function pointers"
}
