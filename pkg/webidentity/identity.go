// Package webidentity provides pure webserver-name and instance-identity helpers.
package webidentity

import (
	"strconv"
	"strings"
)

var names = []string{"caddy", "nginx", "apache"}

// ServerNames lists every supported webserver in default preference order.
func ServerNames() []string {
	return append([]string(nil), names...)
}

// IsValidServerName reports whether name is a supported webserver.
func IsValidServerName(name string) bool {
	for _, n := range names {
		if n == name {
			return true
		}
	}
	return false
}

// InstanceURL builds the primary URL of a server instance from its host and port.
func InstanceURL(host string, port int) string {
	if host == "" {
		host = "localhost"
	}
	return "http://" + host + ":" + strconv.Itoa(port) + "/"
}

// NormalizeSAPI maps an SAPI spelling to its canonical "fpm" or "modphp" form.
func NormalizeSAPI(value string) string {
	v := strings.ToLower(strings.TrimSpace(value))
	switch v {
	case "", "fpm":
		return "fpm"
	case "modphp", "mod_php", "apache":
		return "modphp"
	default:
		return v
	}
}

// ServerPackageName maps the user-facing server name to its registry package name.
func ServerPackageName(server string) string {
	if server == "apache" {
		return "httpd"
	}
	return server
}
