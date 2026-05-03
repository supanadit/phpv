package compiler

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/supanadit/phpv/internal/config"
	"github.com/supanadit/phpv/internal/utils"
)

// CompilerRepository defines the interface for compiler operations
type CompilerRepository interface {
	GetRequiredCompilerForPHP(phpVersion string, forceCompiler CompilerType) CompilerType
	GetEffectiveCompilerForPHP(phpVersion string) CompilerType
	IsCompilerAvailable(compilerType CompilerType) bool
	UsesZigForPHP(phpVersion string) bool
	GetZigTarget() string
	GetZigTargetForGlibc(glibcVersion string) string
}

// CompilerService provides unified compiler detection and management
type CompilerService struct {
	siloRoot string
}

// NewCompilerService creates a new instance of CompilerService
func NewCompilerService(siloRoot string) *CompilerService {
	return &CompilerService{
		siloRoot: siloRoot,
	}
}

// CompilerType represents the type of compiler
type CompilerType string

const (
	CompilerTypeGCC CompilerType = "gcc"
	CompilerTypeZig CompilerType = "zig"
)

// CompilerInfo contains information about a compiler
type CompilerInfo struct {
	Type      CompilerType
	Path      string
	Name      string
	Version   string
	Available bool
}

// GetRequiredCompilerForPHP determines which compiler is required for the given PHP version
// This is the preferred compiler based on PHP version requirements (not availability)
func (c *CompilerService) GetRequiredCompilerForPHP(phpVersion string, forceCompiler CompilerType) CompilerType {
	if phpVersion == "" {
		return CompilerTypeGCC
	}

	v := utils.ParseVersion(phpVersion)

	// For forced compiler selection
	if forceCompiler == CompilerTypeZig {
		return CompilerTypeZig
	} else if forceCompiler == CompilerTypeGCC {
		return CompilerTypeGCC
	}

	// PHP versions 5.x through 7.x prefer gcc
	if v.Major >= 5 && v.Major < 8 {
		return CompilerTypeGCC
	}

	// PHP versions < 5 or >= 8 prefer zig
	return CompilerTypeZig
}

// GetEffectiveCompilerForPHP returns the compiler that will actually be used for building
// This considers both version requirements and actual availability
func (c *CompilerService) GetEffectiveCompilerForPHP(phpVersion string) CompilerType {
	if phpVersion == "" {
		return CompilerTypeGCC
	}

	v := utils.ParseVersion(phpVersion)

	// PHP 5+: always use gcc if available, else zig
	if v.Major >= 5 {
		if c.IsCompilerAvailable(CompilerTypeGCC) {
			return CompilerTypeGCC
		}
		return CompilerTypeZig
	}

	// PHP < 5: only zig (legacy)
	if c.IsCompilerAvailable(CompilerTypeZig) {
		return CompilerTypeZig
	}
	return "" // No viable compiler
}

// IsCompilerAvailable checks if a compiler is available on the system
func (c *CompilerService) IsCompilerAvailable(compilerType CompilerType) bool {
	var err error
	switch compilerType {
	case CompilerTypeGCC:
		_, err = exec.LookPath("gcc")
	case CompilerTypeZig:
		// Check for environment variable first
		if zigPath := os.Getenv("PHPV_ZIG_PATH"); zigPath != "" {
			if _, statErr := os.Stat(zigPath); statErr == nil {
				return true
			}
		}
		// Check for zig in phpv's managed tools
		zigBinary := utils.GetZigCompilerPath(c.siloRoot, "8.4.0")
		if _, statErr := os.Stat(zigBinary); statErr == nil {
			return true
		}
		// Fallback to system zig
		_, err = exec.LookPath("zig")
	default:
		return false
	}
	return err == nil
}

// UsesZigForPHP returns whether zig will be used for the given PHP version
func (c *CompilerService) UsesZigForPHP(phpVersion string) bool {
	return c.GetEffectiveCompilerForPHP(phpVersion) == CompilerTypeZig
}

// GetCompilerInfo returns information about the specified compiler
func (c *CompilerService) GetCompilerInfo(compilerType CompilerType) CompilerInfo {
	var path string
	var name string
	var err error

	switch compilerType {
	case CompilerTypeGCC:
		name = "gcc"
		path, err = exec.LookPath("gcc")
	case CompilerTypeZig:
		name = "zig"
		// Check for environment variable first
		if zigPath := os.Getenv("PHPV_ZIG_PATH"); zigPath != "" {
			if _, err := os.Stat(zigPath); err == nil {
				path = zigPath
				break
			}
		}
		// Check for zig in phpv's managed tools (use config)
		zigBinary := utils.GetZigCompilerPath(config.Get().RootDir(), "8.4.0")
		if _, err := os.Stat(zigBinary); err == nil {
			path = zigBinary
			break
		}
		// Fallback to system zig
		path, err = exec.LookPath("zig")
	default:
		return CompilerInfo{
			Type:      compilerType,
			Available: false,
		}
	}

	if err != nil {
		return CompilerInfo{
			Type:      compilerType,
			Name:      name,
			Available: false,
		}
	}

	version := c.getCompilerVersion(name, path)

	return CompilerInfo{
		Type:      compilerType,
		Path:      path,
		Name:      name,
		Version:   version,
		Available: true,
	}
}

// GetDefaultCompilerForPHP returns the best available compiler for the given PHP version
func (c *CompilerService) GetDefaultCompilerForPHP(phpVersion string) CompilerType {
	effective := c.GetEffectiveCompilerForPHP(phpVersion)
	if effective != "" {
		return effective
	}

	// Fallback: check what's available
	if c.IsCompilerAvailable(CompilerTypeGCC) {
		return CompilerTypeGCC
	}
	if c.IsCompilerAvailable(CompilerTypeZig) {
		return CompilerTypeZig
	}
	return ""
}

// GetZigTarget returns the zig target for the current platform
func (c *CompilerService) GetZigTarget() string {
	return utils.GetZigTarget()
}

// GetZigTargetForGlibc returns the zig target with a specific glibc version
func (c *CompilerService) GetZigTargetForGlibc(glibcVersion string) string {
	return utils.GetZigTargetForGlibc(glibcVersion)
}

func (c *CompilerService) getCompilerVersion(name, path string) string {
	var args []string
	switch name {
	case "gcc":
		args = []string{"--version"}
	case "zig":
		// For zig cc/c++ wrappers, we need to extract the zig binary
		if filepath.Base(path) != "zig" {
			parts := filepath.SplitList(path)
			if len(parts) > 0 {
				path = parts[0]
			}
		}
		args = []string{"version"}
	default:
		return ""
	}

	cmd := exec.Command(path, args...)
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	return string(output)
}
