package utils

var BuildTools = map[string]bool{
	"m4":       true,
	"autoconf": true,
	"automake": true,
	"libtool":  true,
	"perl":     true,
	"bison":    true,
	"flex":     true,
	"re2c":     true,
	"zig":      true,
}

func IsBuildTool(name string) bool {
	return BuildTools[name]
}
