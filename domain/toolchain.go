package domain

// ToolchainConfig describes an optional legacy toolchain/sysroot configuration
// that phpv can use when building dependencies and PHP itself.
type ToolchainConfig struct {
	CC       string
	CXX      string
	Sysroot  string
	Path     []string
	CFlags   []string
	CPPFlags []string
	LDFlags  []string
}

// IsEmpty returns true when no overrides are defined.
func (t *ToolchainConfig) IsEmpty() bool {
	if t == nil {
		return true
	}
	return t.CC == "" && t.CXX == "" && t.Sysroot == "" &&
		len(t.Path) == 0 && len(t.CFlags) == 0 && len(t.CPPFlags) == 0 && len(t.LDFlags) == 0
}
