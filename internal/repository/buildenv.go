package repository

import (
	"path/filepath"
)

func ForgeEnv(prefix string) map[string]string {
	env := map[string]string{
		"PKG_CONFIG_PATH": filepath.Join(prefix, "lib", "pkgconfig"),
	}
	env["CPPFLAGS"] = "-I" + filepath.Join(prefix, "include")
	env["LDFLAGS"] = "-L" + filepath.Join(prefix, "lib") + " -Wl,-rpath," + filepath.Join(prefix, "lib")
	return env
}

func DetectStrategy(name string) string {
	switch name {
	case "zlib", "openssl", "ossl", "libtool", "libxml2", "php", "m4", "automake", "autoconf":
		return "configure"
	case "cmake", "libzip":
		return "cmake"
	case "bison", "flex", "perl":
		return "makeonly"
	default:
		return "configure"
	}
}
