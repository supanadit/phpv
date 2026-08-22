package apache

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/supanadit/phpv/domain"
)

// vhostPath returns the config file path for a vhost by server name.
func vhostPath(serverName string) string {
	return filepath.Join(vhostsDir(), serverName+".conf")
}

// VhostAdd generates a vhost config from the current Apache configuration.
func (s *ApacheService) VhostAdd(v domain.Vhost) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	if cfg == nil {
		return fmt.Errorf("apache is not configured. Run `phpv apache install` then `phpv apache configure` first")
	}
	if v.ServerName == "" {
		return fmt.Errorf("vhost requires a ServerName")
	}
	if v.DocumentRoot == "" {
		return fmt.Errorf("vhost requires a document root")
	}
	if v.Port == 0 {
		v.Port = cfg.ListenPort
	}

	// Per-vhost PHP (FPM only) gets its own backend port.
	if v.PHPVersion != "" {
		if domain.ConnectorMode(cfg.Connector) != domain.ConnectorFPM {
			return fmt.Errorf("--php-version is only supported in FPM connector mode (current: %s)", cfg.Connector)
		}
		if v.FPMPort == 0 {
			v.FPMPort = cfg.FPMBasePort + len(cfg.Vhosts) + 1
		}
	}

	content := s.renderVhost(v)
	if err := ensureDirs(); err != nil {
		return err
	}
	if err := os.WriteFile(vhostPath(v.ServerName), []byte(content), 0o644); err != nil {
		return err
	}

	// Mark enabled by default.
	v.Enabled = true
	replaceVhost(cfg, v)
	if err := saveConfig(cfg); err != nil {
		return err
	}
	if err := s.restartIfRunning(); err != nil {
		return err
	}
	fmt.Printf("✓ Added vhost %s -> %s (port %d)\n", v.ServerName, v.DocumentRoot, v.Port)
	return nil
}

// renderVhost builds the Apache <VirtualHost> block.
func (s *ApacheService) renderVhost(v domain.Vhost) string {
	addr := "*"
	var b strings.Builder
	port := v.Port
	if port == 0 {
		port = 80
	}

	// Handler directive for the vhost. FPM mode with a per-vhost PHP version
	// points at a dedicated FPM port; otherwise it inherits the global
	// SetHandler from php-phpv.conf.
	if v.PHPVersion != "" && v.FPMPort > 0 {
		b.WriteString(fmt.Sprintf("<VirtualHost %s:%d>\n", addr, port))
		fmt.Fprintf(&b, "    ServerName %s\n", v.ServerName)
		for _, a := range v.Aliases {
			fmt.Fprintf(&b, "    ServerAlias %s\n", a)
		}
		fmt.Fprintf(&b, "    DocumentRoot %s\n", v.DocumentRoot)
		b.WriteString("    <FilesMatch \\.php$>\n")
		fmt.Fprintf(&b, "        SetHandler \"proxy:fcgi://127.0.0.1:%d\"\n", v.FPMPort)
		b.WriteString("    </FilesMatch>\n")
		writeVhostDirectory(&b, v.DocumentRoot)
		writeVhostLogs(&b, v.ServerName)
		if v.SSLEnabled {
			writeVhostSSL(&b, v.ServerName)
		}
		b.WriteString("</VirtualHost>\n")
		return b.String()
	}

	fmt.Fprintf(&b, "<VirtualHost %s:%d>\n", addr, port)
	fmt.Fprintf(&b, "    ServerName %s\n", v.ServerName)
	for _, a := range v.Aliases {
		fmt.Fprintf(&b, "    ServerAlias %s\n", a)
	}
	fmt.Fprintf(&b, "    DocumentRoot %s\n", v.DocumentRoot)
	writeVhostDirectory(&b, v.DocumentRoot)
	writeVhostLogs(&b, v.ServerName)
	if v.SSLEnabled {
		writeVhostSSL(&b, v.ServerName)
	}
	b.WriteString("</VirtualHost>\n")
	return b.String()
}

func writeVhostDirectory(b *strings.Builder, root string) {
	fmt.Fprintf(b, "    <Directory %s>\n", root)
	b.WriteString("        AllowOverride All\n")
	b.WriteString("        Require all granted\n")
	b.WriteString("    </Directory>\n")
}

func writeVhostLogs(b *strings.Builder, name string) {
	fmt.Fprintf(b, "    ErrorLog %s\n", filepath.Join(logsDir(), name+"-error.log"))
	fmt.Fprintf(b, "    CustomLog %s combined\n", filepath.Join(logsDir(), name+"-access.log"))
}

func writeVhostSSL(b *strings.Builder, name string) {
	b.WriteString("    SSLEngine on\n")
	fmt.Fprintf(b, "    SSLCertificateFile %s\n", filepath.Join(sslDir(), name+".crt"))
	fmt.Fprintf(b, "    SSLCertificateKeyFile %s\n", filepath.Join(sslDir(), name+".key"))
}

// VhostRemove deletes a vhost config.
func (s *ApacheService) VhostRemove(serverName string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	path := vhostPath(serverName)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("vhost %q not found", serverName)
	}
	if err := os.Remove(path); err != nil {
		return err
	}
	if cfg != nil {
		cfg.Vhosts = removeVhost(cfg.Vhosts, serverName)
		if err := saveConfig(cfg); err != nil {
			return err
		}
	}
	if err := s.restartIfRunning(); err != nil {
		return err
	}
	fmt.Printf("✓ Removed vhost %s\n", serverName)
	return nil
}

// VhostList returns all managed vhosts.
func (s *ApacheService) VhostList() ([]domain.Vhost, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		return []domain.Vhost{}, nil
	}
	// Always reflect what is on disk.
	var out []domain.Vhost
	for _, v := range cfg.Vhosts {
		if _, err := os.Stat(vhostPath(v.ServerName)); err == nil {
			out = append(out, v)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ServerName < out[j].ServerName })
	return out, nil
}

// VhostEnable creates the enabled marker (vhosts are enabled by default since
// they are all included); it just sets Enabled=true.
func (s *ApacheService) VhostEnable(serverName string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	if _, err := os.Stat(vhostPath(serverName)); os.IsNotExist(err) {
		return fmt.Errorf("vhost %q not found", serverName)
	}
	if cfg != nil {
		for i := range cfg.Vhosts {
			if cfg.Vhosts[i].ServerName == serverName {
				cfg.Vhosts[i].Enabled = true
			}
		}
		if err := saveConfig(cfg); err != nil {
			return err
		}
	}
	if err := s.restartIfRunning(); err != nil {
		return err
	}
	fmt.Printf("✓ Enabled vhost %s\n", serverName)
	return nil
}

// VhostDisable renames the config so it is not included (disabled).
func (s *ApacheService) VhostDisable(serverName string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	src := vhostPath(serverName)
	if _, err := os.Stat(src); os.IsNotExist(err) {
		return fmt.Errorf("vhost %q not found", serverName)
	}
	dst := filepath.Join(vhostsDir(), serverName+".conf.disabled")
	if err := os.Rename(src, dst); err != nil {
		return err
	}
	if cfg != nil {
		for i := range cfg.Vhosts {
			if cfg.Vhosts[i].ServerName == serverName {
				cfg.Vhosts[i].Enabled = false
			}
		}
		if err := saveConfig(cfg); err != nil {
			return err
		}
	}
	if err := s.restartIfRunning(); err != nil {
		return err
	}
	fmt.Printf("✓ Disabled vhost %s\n", serverName)
	return nil
}

func removeVhost(vs []domain.Vhost, name string) []domain.Vhost {
	var out []domain.Vhost
	for _, v := range vs {
		if v.ServerName != name {
			out = append(out, v)
		}
	}
	return out
}

// replaceVhost upserts a vhost into the config's vhost list.
func replaceVhost(cfg *domain.WebserverConfig, v domain.Vhost) {
	for i := range cfg.Vhosts {
		if cfg.Vhosts[i].ServerName == v.ServerName {
			cfg.Vhosts[i] = v
			return
		}
	}
	cfg.Vhosts = append(cfg.Vhosts, v)
}
