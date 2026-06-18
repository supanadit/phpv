package utils

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// GetSystemPkgConfigPaths returns standard system pkg-config search paths.
func GetSystemPkgConfigPaths() []string {
	var paths []string

	basePaths := []string{
		"/usr/lib64/pkgconfig",
		"/usr/lib/pkgconfig",
		"/usr/share/pkgconfig",
		"/usr/local/lib/pkgconfig",
		"/usr/local/share/pkgconfig",
		"/opt/homebrew/lib/pkgconfig",
	}

	archSuffix := GetArch() + "-linux-gnu"

	linuxGnuPaths := []string{
		filepath.Join("/usr/lib", archSuffix, "pkgconfig"),
		filepath.Join("/usr/lib64", archSuffix, "pkgconfig"),
	}

	if runtime.GOOS == "linux" {
		for _, p := range linuxGnuPaths {
			if _, err := os.Stat(p); err == nil {
				paths = append(paths, p)
			}
		}
		paths = append(basePaths, paths...)
	} else {
		paths = basePaths
	}

	return paths
}

// GetZigCompilerPath returns the path to the zig compiler binary.
func GetZigCompilerPath(siloRoot, phpVersion string) string {
	return filepath.Join(siloRoot, "build-tools", "zig", "0.13.0", "zig")
}

// GetOS returns the runtime OS.
func GetOS() string {
	return runtime.GOOS
}

// GetArch returns the normalized CPU architecture.
func GetArch() string {
	a := runtime.GOARCH
	switch a {
	case "amd64":
		return "x86_64"
	case "arm64":
		return "aarch64"
	}
	return a
}

// OSInfo holds detected operating system information.
type OSInfo struct {
	GOOS       string
	Distro     string
	PkgMgr     string
	InstallCmd string
}

// DetectOSInfo probes the OS and returns info about package manager and distro.
func DetectOSInfo() OSInfo {
	info := OSInfo{GOOS: runtime.GOOS}

	switch runtime.GOOS {
	case "darwin":
		info.Distro = "macos"
		info.PkgMgr = "brew"
		info.InstallCmd = "brew install"
		return info

	case "linux":
		data, err := os.ReadFile("/etc/os-release")
		if err != nil {
			info.Distro = "linux"
			info.PkgMgr = "dnf"
			info.InstallCmd = "sudo dnf install"
			return info
		}

		info.Distro = extractOSReleaseID(string(data))
		switch info.Distro {
		case "ubuntu", "debian", "linuxmint", "pop", "elementary", "kali", "raspbian":
			info.PkgMgr = "apt"
			info.InstallCmd = "sudo apt install"
		case "fedora", "rhel", "centos", "rocky", "almalinux":
			info.PkgMgr = "dnf"
			info.InstallCmd = "sudo dnf install"
		case "arch", "manjaro", "endeavour", "arcolinux":
			info.PkgMgr = "pacman"
			info.InstallCmd = "sudo pacman -S"
		case "opensuse", "suse", "opensuse-tumbleweed":
			info.PkgMgr = "zypper"
			info.InstallCmd = "sudo zypper install"
		case "alpine":
			info.PkgMgr = "apk"
			info.InstallCmd = "apk add"
		case "void":
			info.PkgMgr = "xbps"
			info.InstallCmd = "sudo xbps-install"
		default:
			info.PkgMgr = "dnf"
			info.InstallCmd = "sudo dnf install"
		}
		return info

	default:
		info.Distro = runtime.GOOS
		info.PkgMgr = "pkg"
		info.InstallCmd = "pkg install"
		return info
	}
}

func extractOSReleaseID(data string) string {
	for _, line := range strings.Split(data, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "ID=") {
			val := strings.TrimPrefix(line, "ID=")
			val = strings.Trim(val, `'"`)
			return strings.ToLower(val)
		}
	}
	return "linux"
}

// GetZigTarget returns the zig target for the current platform.
func GetZigTarget() string {
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

// GetZigTargetForGlibc returns the zig target with a specific glibc version.
func GetZigTargetForGlibc(glibcVersion string) string {
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

// GetOpenSSLConfigureTarget returns the OpenSSL configure target string.
func GetOpenSSLConfigureTarget() string {
	goarch := runtime.GOARCH
	switch goarch {
	case "amd64":
		goarch = "x86_64"
	case "arm64":
		goarch = "aarch64"
	}
	switch runtime.GOOS {
	case "linux":
		return "linux-" + goarch
	case "darwin":
		if goarch == "x86_64" {
			return "darwin64-x86_64-cc"
		} else if goarch == "aarch64" {
			return "darwin64-arm64-cc"
		}
		return "darwin-" + goarch + "-cc"
	default:
		return ""
	}
}

// GetConfigureHostTriple returns the configure --host triple.
func GetConfigureHostTriple() string {
	goarch := runtime.GOARCH
	switch goarch {
	case "amd64":
		goarch = "x86_64"
	case "arm64":
		goarch = "aarch64"
	}
	switch runtime.GOOS {
	case "linux":
		return goarch + "-pc-linux-gnu"
	case "darwin":
		return goarch + "-apple-darwin"
	default:
		return goarch + "-pc-linux-gnu"
	}
}
