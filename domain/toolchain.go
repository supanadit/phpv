package domain

import (
	"os"
	"strings"

	"github.com/spf13/viper"
)

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

// UseLLVM returns true if the user wants to use LLVM instead of system GCC
func UseLLVM() bool {
	// Check environment variable: PHPV_USE_LLVM=1
	val := viper.GetString("PHPV_USE_LLVM")
	if val == "" {
		val = os.Getenv("PHPV_USE_LLVM")
	}
	return strings.ToLower(val) == "1" || strings.ToLower(val) == "true"
}

// ShouldUseLLVMToolchain returns true if we should use LLVM for building
// This is true for PHP versions < 8.3 OR when PHPV_USE_LLVM=1
func ShouldUseLLVMToolchain(phpVersion Version) bool {
	// If user explicitly wants LLVM, use it
	if UseLLVM() {
		return true
	}
	// For PHP 8.3+, use system GCC by default
	if phpVersion.Major == 8 && phpVersion.Minor >= 3 {
		return false
	}
	// For older PHP versions, use LLVM
	return true
}

// IsEmpty returns true when no overrides are defined.
func (t *ToolchainConfig) IsEmpty() bool {
	if t == nil {
		return true
	}
	return t.CC == "" && t.CXX == "" && t.Sysroot == "" &&
		len(t.Path) == 0 && len(t.CFlags) == 0 && len(t.CPPFlags) == 0 && len(t.LDFlags) == 0
}
