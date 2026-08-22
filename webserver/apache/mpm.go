package apache

import (
	"fmt"
	"os"
	"strings"

	"github.com/supanadit/phpv/domain"
)

// defaultMPM picks a sensible MPM for the connector: mod_php requires prefork
// (not thread-safe); FPM/CGI can use the efficient event MPM. Falls back to a
// user-provided MPM when valid.
func defaultMPM(connector domain.ConnectorMode, requested string) string {
	if requested != "" {
		switch requested {
		case "event", "worker", "prefork":
			return requested
		}
	}
	if connector == domain.ConnectorModPHP {
		return "prefork"
	}
	return "event"
}

// applyMPM rewrites the LoadModule lines in httpd.conf to enable the selected
// MPM. Apache builds with --enable-mpms-shared=all install all three as shared
// modules; only one may be loaded at a time.
func (s *ApacheService) applyMPM(cfg *domain.WebserverConfig, prefix string) error {
	conf := httpdConf(prefix)
	data, err := os.ReadFile(conf)
	if err != nil {
		return err
	}
	lines := strings.Split(string(data), "\n")
	var out []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Skip any existing LoadModule mpm_* lines; we re-add the selected one.
		if strings.Contains(trimmed, "mpm_") && strings.HasPrefix(trimmed, "LoadModule") {
			continue
		}
		out = append(out, line)
	}
	mod := fmt.Sprintf("mpm_%s_module", cfg.MPM)
	out = append(out, fmt.Sprintf("LoadModule %s modules/mod_%s.so", mod, cfg.MPM))
	return os.WriteFile(conf, []byte(strings.Join(out, "\n")), 0o644)
}
