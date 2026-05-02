package utils

import "runtime"

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
