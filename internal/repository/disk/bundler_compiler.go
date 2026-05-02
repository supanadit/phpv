package disk

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/supanadit/phpv/internal/utils"
)

func getZigTarget() string {
	return utils.GetZigTarget()
}

func getZigTargetForGlibc(glibcVersion string) string {
	return utils.GetZigTargetForGlibc(glibcVersion)
}

func findSystemCXX() string {
	if path, err := exec.LookPath("g++"); err == nil {
		return path
	}
	if path, err := exec.LookPath("c++"); err == nil {
		return path
	}
	return ""
}

func findSystemLibStdCxx() []string {
	var flags []string

	libstdcxxA, err := exec.Command("g++", "-print-file-name=libstdc++.a").Output()
	if err == nil {
		libPath := strings.TrimSpace(string(libstdcxxA))
		if _, err := os.Stat(libPath); err == nil {
			flags = append(flags, libPath)
		}
	}

	libgccEhA, err := exec.Command("g++", "-print-file-name=libgcc_eh.a").Output()
	if err == nil {
		libPath := strings.TrimSpace(string(libgccEhA))
		if _, err := os.Stat(libPath); err == nil {
			flags = append(flags, libPath)
		}
	}

	if len(flags) > 0 {
		return flags
	}

	libstdcxxSo, err := exec.Command("g++", "-print-file-name=libstdc++.so").Output()
	if err == nil {
		libPath := strings.TrimSpace(string(libstdcxxSo))
		if idx := strings.LastIndex(libPath, "/"); idx != -1 {
			dir := libPath[:idx]
			if _, err := os.Stat(dir); err == nil {
				flags = append(flags, "-L"+dir, "-lstdc++")
			}
		}
	}

	return flags
}

func (s *bundlerRepository) needsAlternativeCC(phpVersion string, forceCompiler string) bool {
	if forceCompiler == "zig" {
		return true
	}
	if forceCompiler != "" && forceCompiler != "gcc" {
		return false
	}
	v := utils.ParseVersion(phpVersion)
	if v.Major < 5 {
		return true
	}
	return false
}

func (s *bundlerRepository) getCompilerForVersion(phpVersion string, forceCompiler string) (cc string, cflags []string, cxx string, ldFlags []string, err error) {
	v := utils.ParseVersion(phpVersion)

	if v.Major >= 5 && (forceCompiler == "" || forceCompiler == "gcc") {
		gccPath, err := exec.LookPath("gcc")
		if err != nil {
			return "", []string{}, "", nil, fmt.Errorf("[bundler] gcc not found in PATH: %w", err)
		}
		gxxPath, err := exec.LookPath("g++")
		if err != nil {
			return "", []string{}, "", nil, fmt.Errorf("[bundler] g++ not found in PATH: %w", err)
		}
		cflags = []string{"-Wno-error", "-fPIC"}
		return gccPath, cflags, gxxPath, nil, nil
	}

	target := getZigTargetForGlibc("2.39")

	if zigPath := os.Getenv("PHPV_ZIG_PATH"); zigPath != "" {
		if _, err := os.Stat(zigPath); err == nil {
			cxx = zigPath + " c++ -target " + target
			return zigPath + " cc -target " + target, []string{"-std=gnu11", "-fPIC", "-Wno-error", "-fno-sanitize=undefined", "-Wno-cast-align", "-Wno-unused-but-set-variable", "-Wno-deprecated-non-prototype", "-Wno-array-parameter", "-Wno-implicit-function-declaration"}, cxx, nil, nil
		}
	}

	zigBinary := utils.GetZigCompilerPath(s.silo.Root, phpVersion)
	zigVersion := "0.13.0"

	if _, err := os.Stat(zigBinary); os.IsNotExist(err) {
		if err := s.installBuildTool("zig", zigVersion, phpVersion); err != nil {
			return "", nil, "", nil, fmt.Errorf("[bundler] failed to install zig: %w", err)
		}
		zigBinary = utils.GetZigCompilerPath(s.silo.Root, phpVersion)
	} else {
		if err := s.siloRepo.IncrementBuildToolRef("zig", filepath.Base(filepath.Dir(zigBinary)), phpVersion); err != nil {
			s.logWarn("Warning: failed to increment zig ref: %v", err)
		}
	}

	cxx = zigBinary + " c++ -target " + target
	cflags = []string{"-std=gnu11", "-fPIC", "-Wno-error", "-fno-sanitize=undefined", "-Wno-cast-align", "-Wno-unused-but-set-variable", "-Wno-deprecated-non-prototype", "-Wno-array-parameter", "-Wno-implicit-function-declaration"}
	return zigBinary + " cc -target " + target, cflags, cxx, nil, nil
}
