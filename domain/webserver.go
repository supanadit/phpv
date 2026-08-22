package domain

// ConnectorMode defines how a webserver (e.g. Apache) talks to PHP.
// It maps to extra configure flags added to the PHP build.
type ConnectorMode string

const (
	// ConnectorNone means PHP is built without any webserver connector.
	// This is the default: a plain CLI build (--disable-all --enable-cli).
	ConnectorNone ConnectorMode = ""
	// ConnectorFPM builds PHP with --enable-fpm so a php-fpm pool can serve
	// requests via mod_proxy_fcgi. Recommended default; works with event/worker
	// MPMs and supports per-vhost PHP versions.
	ConnectorFPM ConnectorMode = "fpm"
	// ConnectorCGI builds PHP with --enable-cgi so mod_fcgid can spawn php-cgi.
	ConnectorCGI ConnectorMode = "cgi"
	// ConnectorModPHP builds PHP with --with-apxs2 so it compiles libphp.so,
	// loaded directly by Apache. Requires the prefork MPM (not thread-safe) and
	// supports a single PHP version for all vhosts.
	ConnectorModPHP ConnectorMode = "mod_php"
)

// IsValid reports whether the connector mode is a known value.
func (m ConnectorMode) IsValid() bool {
	switch m {
	case ConnectorNone, ConnectorFPM, ConnectorCGI, ConnectorModPHP:
		return true
	}
	return false
}

// Vhost describes a single virtual host managed by a webserver.
type Vhost struct {
	ServerName   string   `json:"server_name"`
	DocumentRoot string   `json:"document_root"`
	Port         int      `json:"port,omitempty"`
	Aliases      []string `json:"aliases,omitempty"`
	SSLEnabled   bool     `json:"ssl_enabled,omitempty"`
	SSLCertFile  string   `json:"ssl_cert_file,omitempty"`
	SSLKeyFile   string   `json:"ssl_key_file,omitempty"`
	// PHPVersion optionally pins a per-vhost PHP version (FPM/CGI modes only).
	PHPVersion string `json:"php_version,omitempty"`
	// FPMPort is the FPM backend port for this vhost when PHPVersion is set.
	FPMPort int  `json:"fpm_port,omitempty"`
	Enabled bool `json:"enabled"`
}

// WebserverConfig is the persisted configuration for a webserver instance.
type WebserverConfig struct {
	ServerType string  `json:"server_type"` // e.g. "apache"
	Version    string  `json:"version"`     // webserver (httpd) version
	Prefix     string  `json:"prefix"`      // install prefix
	PHPVersion string  `json:"php_version"` // configured PHP version
	Connector  string  `json:"connector"`   // one of ConnectorMode values
	MPM        string  `json:"mpm"`         // "event", "worker", "prefork"
	ListenPort int     `json:"listen_port"` // default 80
	User       string  `json:"user"`        // run-as user
	Group      string  `json:"group"`       // run-as group
	FPMBasePort int    `json:"fpm_base_port"` // default 9000
	Vhosts     []Vhost `json:"vhosts"`
}
