package repository

import "strings"

// DirRangeAny is the range directory that matches every version.
const DirRangeAny = "any"

// Directory names encode version ranges in a Windows-safe form; these helpers
// translate them into the constraint syntax used by the version library.

func DirRangeExpr(dir string) string {
	if dir == DirRangeAny || dir == "" {
		return ""
	}
	if strings.HasSuffix(dir, "+") {
		return ">=" + strings.TrimSuffix(dir, "+")
	}
	if a, b, ok := strings.Cut(dir, "-"); ok && a != "" && b != "" {
		// Both sides must look like plain versions to avoid misreading
		// other directory names; versions carry no dashes in phpv.
		if isPlainVersion(a) && isPlainVersion(b) {
			return ">=" + a + ",<" + b
		}
	}
	return "=" + dir
}

// isPlainVersion reports whether s is a bare x[.y[.z]] version.
func isPlainVersion(s string) bool {
	if s == "" {
		return false
	}
	for _, seg := range strings.Split(s, ".") {
		if seg == "" {
			return false
		}
		for _, r := range seg {
			if r < '0' || r > '9' {
				return false
			}
		}
	}
	return true
}

// root is one patch source tree. Roots are consulted in precedence order;
// the first root to claim a directory path owns it wholly.
