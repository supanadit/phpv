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

// UseLLVM returns true if the user wants to use LLVM instead of system GCC/Zig
func UseLLVM() bool {
	val := viper.GetString("PHPV_USE_LLVM")
	if val == "" {
		val = os.Getenv("PHPV_USE_LLVM")
	}
	return strings.ToLower(val) == "1" || strings.ToLower(val) == "true"
}

// UseZig returns true if the user wants to use Zig instead of LLVM/GCC
func UseZig() bool {
	val := viper.GetString("PHPV_USE_ZIG")
	if val == "" {
		val = os.Getenv("PHPV_USE_ZIG")
	}
	return strings.ToLower(val) == "1" || strings.ToLower(val) == "true"
}

// UseGCC returns true if the user wants to use system GCC instead of Zig/LLVM
func UseGCC() bool {
	val := viper.GetString("PHPV_USE_GCC")
	if val == "" {
		val = os.Getenv("PHPV_USE_GCC")
	}
	return strings.ToLower(val) == "1" || strings.ToLower(val) == "true"
}

// ShouldUseZigToolchain returns true if we should use Zig for building
// For PHP < 8.1 (4.x, 5.x, 7.x, 8.0): DEFAULT - use Zig to avoid LLVM libtinfo issues
// For PHP >= 8.1: only if PHPV_USE_ZIG=1
func ShouldUseZigToolchain(phpVersion Version) bool {
	// If user explicitly wants LLVM, don't use Zig
	if UseLLVM() {
		return false
	}
	// If user explicitly wants GCC, don't use Zig
	if UseGCC() {
		return false
	}
	// If user explicitly wants Zig, use it
	if UseZig() {
		return true
	}
	// DEFAULT: Use Zig for PHP < 8.1 (4.x, 5.x, 7.x, 8.0)
	if phpVersion.Major < 8 || (phpVersion.Major == 8 && phpVersion.Minor < 1) {
		return true
	}
	return false
}

// ShouldUseLLVMToolchain returns true if we should use LLVM for building
// Only when PHPV_USE_LLVM=1 is explicitly set
func ShouldUseLLVMToolchain(phpVersion Version) bool {
	return UseLLVM()
}

// GetCompilerType returns which compiler to use: "zig", "llvm", or "system"
func GetCompilerType(phpVersion Version) string {
	// Priority: LLVM (explicit) > Zig (explicit or default for <8.1) > system GCC
	if UseLLVM() {
		return "llvm"
	}
	if ShouldUseZigToolchain(phpVersion) {
		return "zig"
	}
	return "system"
}

// GetSystemLibPaths returns common system library paths for dynamic linking
func GetSystemLibPaths() []string {
	paths := []string{
		"/usr/lib",
		"/usr/lib64",
		"/usr/lib/x86_64-linux-gnu",
		"/usr/local/lib",
		"/lib/x86_64-linux-gnu",
		"/lib64",
	}

	var existingPaths []string
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			existingPaths = append(existingPaths, p)
		}
	}
	return existingPaths
}

// IsEmpty returns true when no overrides are defined.
func (t *ToolchainConfig) IsEmpty() bool {
	if t == nil {
		return true
	}
	return t.CC == "" && t.CXX == "" && t.Sysroot == "" &&
		len(t.Path) == 0 && len(t.CFlags) == 0 && len(t.CPPFlags) == 0 && len(t.LDFlags) == 0
}
