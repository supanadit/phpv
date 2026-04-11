package disk

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/supanadit/phpv/internal/utils"
)

func getZigTarget() string {
	goarch := runtime.GOARCH
	switch goarch {
	case "amd64":
		goarch = "x86_64"
	case "arm64":
		goarch = "aarch64"
	}

	goos := runtime.GOOS
	abi := "-gnu"
	if goos == "darwin" {
		abi = "-macos"
	}

	return goarch + "-" + goos + abi
}

func (s *bundlerRepository) needsAlternativeCC(phpVersion string, forceCompiler string) bool {
	if forceCompiler == "zig" {
		return true
	}
	if forceCompiler != "" && forceCompiler != "gcc" {
		return false
	}
	v := utils.ParseVersion(phpVersion)
	if v.Major < 8 {
		return true
	}
	if v.Major == 8 && v.Minor == 0 {
		return true
	}
	return false
}

func (s *bundlerRepository) getCompilerForVersion(phpVersion string, forceCompiler string) (cc string, cflags []string, cxx string, err error) {
	if !s.needsAlternativeCC(phpVersion, forceCompiler) {
		return "", []string{}, "", nil
	}

	if zigPath := os.Getenv("PHPV_ZIG_PATH"); zigPath != "" {
		if _, err := os.Stat(zigPath); err == nil {
			target := getZigTarget()
			return zigPath + " cc -target " + target, []string{"-fPIC", "-Wno-error", "-fno-sanitize=undefined", "-Wno-cast-align", "-Wno-unused-but-set-variable", "-Wno-deprecated-non-prototype", "-Wno-array-parameter", "-Wno-implicit-function-declaration"}, zigPath + " c++ -target " + target, nil
		}
	}

	zigBinary := utils.GetZigCompilerPath(s.silo.Root, phpVersion)

	if _, err := os.Stat(zigBinary); os.IsNotExist(err) {
		v := utils.ParseVersion(phpVersion)
		zigVersion := "0.14.0"
		if v.Major < 7 {
			zigVersion = "0.13.0"
		}
		if err := s.installBuildTool("zig", zigVersion, phpVersion); err != nil {
			return "", nil, "", fmt.Errorf("[bundler] failed to install zig: %w", err)
		}
		zigBinary = utils.GetZigCompilerPath(s.silo.Root, phpVersion)
	} else {
		if err := s.siloRepo.IncrementBuildToolRef("zig", filepath.Base(filepath.Dir(zigBinary)), phpVersion); err != nil {
			s.logWarn("Warning: failed to increment zig ref: %v", err)
		}
	}

	target := getZigTarget()
	return zigBinary + " cc -target " + target, []string{"-fPIC", "-Wno-error", "-fno-sanitize=undefined", "-Wno-cast-align", "-Wno-unused-but-set-variable", "-Wno-deprecated-non-prototype", "-Wno-array-parameter", "-Wno-implicit-function-declaration"}, zigBinary + " c++ -target " + target, nil
}
