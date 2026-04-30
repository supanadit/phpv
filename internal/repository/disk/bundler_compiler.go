package disk

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

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

func getZigTargetForGlibc(glibcVersion string) string {
	goarch := runtime.GOARCH
	switch goarch {
	case "amd64":
		goarch = "x86_64"
	case "arm64":
		goarch = "aarch64"
	}

	goos := runtime.GOOS
	if goos == "darwin" {
		return goarch + "-" + goos
	}

	return goarch + "-linux-gnu." + glibcVersion
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
	if v.Major < 8 {
		return true
	}
	if v.Major == 8 && v.Minor == 0 {
		return true
	}
	return false
}

func (s *bundlerRepository) getCompilerForVersion(phpVersion string, forceCompiler string) (cc string, cflags []string, cxx string, ldFlags []string, err error) {
	if !s.needsAlternativeCC(phpVersion, forceCompiler) {
		systemCxx := findSystemCXX()
		return "", []string{}, systemCxx, nil, nil
	}

	v := utils.ParseVersion(phpVersion)
	target := getZigTarget()

	if v.Major < 8 {
		target = getZigTargetForGlibc("2.39")
	}

	if zigPath := os.Getenv("PHPV_ZIG_PATH"); zigPath != "" {
		if _, err := os.Stat(zigPath); err == nil {
			cxx = zigPath + " c++ -target " + target
			return zigPath + " cc -target " + target, []string{"-std=gnu11", "-fPIC", "-Wno-error", "-fno-sanitize=undefined", "-Wno-cast-align", "-Wno-unused-but-set-variable", "-Wno-deprecated-non-prototype", "-Wno-array-parameter", "-Wno-implicit-function-declaration"}, cxx, nil, nil
		}
	}

	zigBinary := utils.GetZigCompilerPath(s.silo.Root, phpVersion)

	if _, err := os.Stat(zigBinary); os.IsNotExist(err) {
		zigVersion := "0.14.0"
		if v.Major < 7 {
			zigVersion = "0.13.0"
		}
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
