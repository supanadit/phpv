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

// ZigVersion represents a Zig compiler version
type ZigVersion struct {
	Version     string
	DownloadURL string
}

// GetZigVersion returns the appropriate Zig version
func GetZigVersion() ZigVersion {
	return ZigVersion{
		Version:     "0.14.0",
		DownloadURL: "https://ziglang.org/download/0.14.0/zig-linux-x86_64-0.14.0.tar.xz",
	}
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

// UseZig returns true if the user wants to use Zig instead of LLVM/GCC
func UseZig() bool {
	// Check environment variable: PHPV_USE_ZIG=1
	val := viper.GetString("PHPV_USE_ZIG")
	if val == "" {
		val = os.Getenv("PHPV_USE_ZIG")
	}
	return strings.ToLower(val) == "1" || strings.ToLower(val) == "true"
}

// ShouldUseZigToolchain returns true if we should use Zig for building
// This is true when PHPV_USE_ZIG=1
func ShouldUseZigToolchain(phpVersion Version) bool {
	// If user explicitly wants Zig, use it
	if UseZig() {
		return true
	}
	return false
}

// GetCompilerType returns which compiler to use: "zig", "llvm", or "system"
func GetCompilerType(phpVersion Version) string {
	if UseZig() {
		return "zig"
	}
	if ShouldUseLLVMToolchain(phpVersion) {
		return "llvm"
	}
	return "system"
}

// IsEmpty returns true when no overrides are defined.
func (t *ToolchainConfig) IsEmpty() bool {
	if t == nil {
		return true
	}
	return t.CC == "" && t.CXX == "" && t.Sysroot == "" &&
		len(t.Path) == 0 && len(t.CFlags) == 0 && len(t.CPPFlags) == 0 && len(t.LDFlags) == 0
}
