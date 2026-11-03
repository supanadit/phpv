package domain

import (
	"strconv"
	"time"
)

// Supported webserver names for the `phpv server` command family.
const (
	ServerCaddy  = "caddy"
	ServerNginx  = "nginx"
	ServerApache = "apache"
)

// ServerNames lists every supported webserver in default preference order.
func ServerNames() []string {
	return []string{ServerCaddy, ServerNginx, ServerApache}
}

// IsValidServerName reports whether name is a supported webserver.
func IsValidServerName(name string) bool {
	for _, n := range ServerNames() {
		if n == name {
			return true
		}
	}
	return false
}

// ServerInstance describes one running (or recently stopped) server run.
// Instances are isolated per PHP version: each run pins one PHP version and
// gets its own PHP runtime (FPM master + socket, or an embedded mod_php
// module) and log directory, so e.g. a PHP 5.6 app and a PHP 8.3 app can be
// served simultaneously.
type ServerInstance struct {
	// ID uniquely identifies the instance, e.g. "caddy-8080-php5.6.40".
	ID string
	// Server is the webserver name (caddy, nginx, apache).
	Server string
	// ServerVersion is the installed webserver version used for this run.
	ServerVersion string
	// PHPVersion is the exact PHP version whose runtime serves this instance.
	PHPVersion string
	// SAPI is how PHP runs: "fpm" (default) or "modphp".
	SAPI string
	// Dir is the absolute document root being served.
	Dir string
	// Host is the optional vhost hostname (--host). Empty means localhost only.
	Host string
	// Port is the TCP port the webserver listens on.
	Port int
	// Socket is the absolute path of the instance's FastCGI unix socket.
	Socket string
	// StateDir holds the instance's generated configs, logs and socket.
	StateDir string
	// ServerPID is the webserver process pid (0 when not known).
	ServerPID int
	// FPMPID is the php-fpm master process pid (0 when not known).
	FPMPID int
	// Detached reports whether the instance runs in the background.
	Detached bool
	// StartedAt is when the instance was started.
	StartedAt time.Time
}

// URL returns the primary URL of the instance.
func (i ServerInstance) URL() string {
	host := i.Host
	if host == "" {
		host = "localhost"
	}
	return "http://" + host + ":" + strconv.Itoa(i.Port) + "/"
}
