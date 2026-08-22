package apache

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/supanadit/phpv/domain"
)

// httpdConf returns the path to the httpd.conf for an httpd prefix.
func httpdConf(prefix string) string {
	return filepath.Join(prefix, "conf", "httpd.conf")
}

// phpConnectorConf returns the path to the PHP connector include snippet.
func phpConnectorConf() string {
	return filepath.Join(confDir(), "php-phpv.conf")
}

// vhostIncludeConf returns the include snippet that pulls in all vhosts.
func vhostIncludeConf() string {
	return filepath.Join(confDir(), "vhosts.conf")
}

// writeBaseConfig writes the phpv-managed connector + vhost includes and points
// Apache's httpd.conf at them. It also rewrites the default Listen/User/Group
// so Apache runs fully in user space on the configured port.
func (s *ApacheService) writeBaseConfig(cfg *domain.WebserverConfig, prefix string) error {
	if err := ensureDirs(); err != nil {
		return err
	}

	// 1. Write the connector snippet (mod_proxy_fcgi for FPM, etc).
	if err := s.writeConnectorSnippet(cfg); err != nil {
		return err
	}

	// 2. Write the vhost include that globs every .conf under vhosts/.
	vhostInclude := fmt.Sprintf("Include %s/*.conf\n", vhostsDir())
	if err := os.WriteFile(vhostIncludeConf(), []byte(vhostInclude), 0o644); err != nil {
		return err
	}

	conf := httpdConf(prefix)
	if _, err := os.Stat(conf); err != nil {
		return fmt.Errorf("httpd.conf not found at %s (was httpd built with --sysconfdir?)", conf)
	}

	// 3. Rewrite the base httpd.conf directives.
	rewrites := map[string]string{
		"Listen":     fmt.Sprintf("Listen %d", cfg.ListenPort),
		"User":       fmt.Sprintf("User %s", cfg.User),
		"Group":      fmt.Sprintf("Group %s", cfg.Group),
	}
	if err := rewriteDirectives(conf, rewrites); err != nil {
		return err
	}

	// 4. Append phpv-managed Includes (idempotent).
	includes := []string{
		"Include " + phpConnectorConf(),
		"Include " + vhostIncludeConf(),
	}
	if err := appendIncludes(conf, includes); err != nil {
		return err
	}

	// 5. DirectoryIndex with index.php first.
	if err := ensureDirective(conf, "DirectoryIndex", "DirectoryIndex index.php index.html"); err != nil {
		return err
	}

	return nil
}

// writeConnectorSnippet generates the PHP connector config block.
func (s *ApacheService) writeConnectorSnippet(cfg *domain.WebserverConfig) error {
	connector := domain.ConnectorMode(cfg.Connector)
	var block string
	switch connector {
	case domain.ConnectorFPM:
		port := cfg.FPMBasePort
		if port == 0 {
			port = 9000
		}
		block = fmt.Sprintf(`# phpv: PHP-FPM connector (PHP %s)
LoadModule proxy_module modules/mod_proxy.so
LoadModule proxy_fcgi_module modules/mod_proxy_fcgi.so
<FilesMatch \.php$>
    SetHandler "proxy:fcgi://127.0.0.1:%d"
</FilesMatch>
`, cfg.PHPVersion, port)
	case domain.ConnectorCGI:
		block = fmt.Sprintf(`# phpv: PHP-CGI connector (PHP %s)
LoadModule fcgid_module modules/mod_fcgid.so
<FilesMatch \.php$>
  SetHandler fcgid-script
  Options +ExecCGI
</FilesMatch>
`, cfg.PHPVersion)
	case domain.ConnectorModPHP:
		block = fmt.Sprintf(`# phpv: mod_php connector (PHP %s)
LoadModule php_module modules/libphp.so
AddType application/x-httpd-php .php
`, cfg.PHPVersion)
	default:
		block = "# phpv: no PHP connector configured\n"
	}
	return os.WriteFile(phpConnectorConf(), []byte(block), 0o644)
}

// rewriteDirectives replaces the value of known single-line directives in an
// Apache config file (e.g. "Listen 80" -> "Listen 8080").
func rewriteDirectives(path string, rewrites map[string]string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		for key, newVal := range rewrites {
			if strings.HasPrefix(trimmed, key+" ") || trimmed == key {
				lines[i] = newVal
				break
			}
		}
	}
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644)
}

// appendIncludes appends Include directives to an Apache config if not already
// present.
func appendIncludes(path string, includes []string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	content := string(data)
	for _, inc := range includes {
		if !strings.Contains(content, inc) {
			content += "\n" + inc
		}
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

// ensureDirective ensures a directive line exists, adding it if absent.
func ensureDirective(path, key, full string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	content := string(data)
	for _, line := range strings.Split(content, "\n") {
		if strings.TrimSpace(line) == full {
			return nil
		}
	}
	// Remove any existing key line then append the new one.
	var kept []string
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, key+" ") || trimmed == key {
			continue
		}
		kept = append(kept, line)
	}
	kept = append(kept, full)
	return os.WriteFile(path, []byte(strings.Join(kept, "\n")), 0o644)
}
