package terminal

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/supanadit/phpv/domain"
)

// serverPreferenceFile records which webserver belongs to which PHP version.
// `phpv server install` writes it for the active PHP version and
// `phpv server run` reads it back, so a per-shell `phpv use` also switches
// the webserver instead of leaking the global (default-PHP) server choice.
type serverPreferenceFile struct {
	Servers map[string]string `json:"servers,omitempty"`
}

func serverPreferencesPath(root string) string {
	return filepath.Join(root, "servers", "preferences.json")
}

func loadServerPreferences(root string) serverPreferenceFile {
	var prefs serverPreferenceFile
	data, err := os.ReadFile(serverPreferencesPath(root))
	if err != nil {
		return serverPreferenceFile{}
	}
	if err := json.Unmarshal(data, &prefs); err != nil {
		return serverPreferenceFile{}
	}
	if prefs.Servers == nil {
		prefs.Servers = map[string]string{}
	}
	return prefs
}

func saveServerPreferences(root string, prefs serverPreferenceFile) error {
	path := serverPreferencesPath(root)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(prefs, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func (p serverPreferenceFile) get(phpVersion string) (string, bool) {
	if phpVersion == "" {
		return "", false
	}
	server, ok := p.Servers[phpVersion]
	return server, ok && server != ""
}

func (p *serverPreferenceFile) set(phpVersion, server string) {
	if phpVersion == "" || server == "" {
		return
	}
	if p.Servers == nil {
		p.Servers = map[string]string{}
	}
	p.Servers[phpVersion] = server
}

// serverPreference returns the webserver recorded for a PHP version.
func (h *PHPHandler) serverPreference(phpVersion string) (string, bool) {
	return loadServerPreferences(resolvePHPVRootForServer()).get(phpVersion)
}

// recordServerPreference remembers the webserver chosen for a PHP version.
func (h *PHPHandler) recordServerPreference(phpVersion, server string) error {
	if phpVersion == "" || server == "" {
		return nil
	}
	root := resolvePHPVRootForServer()
	prefs := loadServerPreferences(root)
	prefs.set(phpVersion, server)
	return saveServerPreferences(root, prefs)
}

// installedServers lists webserver names with at least one installed version.
func (h *PHPHandler) installedServers() []string {
	root := resolvePHPVRootForServer()
	var installed []string
	for _, name := range domain.ServerNames() {
		if entries, err := os.ReadDir(filepath.Join(root, "packages", name)); err == nil && len(entries) > 0 {
			installed = append(installed, name)
		}
	}
	return installed
}

// serverSelectionSource explains why a webserver was picked.
type serverSelectionSource string

const (
	serverFromFlag          serverSelectionSource = "flag"
	serverFromPHPPreference serverSelectionSource = "php-preference"
	serverFromEnv           serverSelectionSource = "env"
	serverFromConfig        serverSelectionSource = "config"
)

// defaultServerEnv names the fallback webserver for PHP versions that have no
// explicit association. It is never written implicitly by `server install`.
const defaultServerEnv = "PHPV_DEFAULT_SERVER"

// resolveServerForPHP picks the webserver for a run. Each PHP version keeps
// its own association; switching versions never reuses a server chosen for
// another version. Resolution order:
//
//	--server flag → preference recorded for this PHP version →
//	PHPV_DEFAULT_SERVER → global config (default PHP only) →
//	association-required error.
func (h *PHPHandler) resolveServerForPHP(flagServer, phpVersion string) (string, serverSelectionSource, error) {
	if flagServer != "" {
		if !domain.IsValidServerName(flagServer) {
			return "", "", fmt.Errorf("unknown server %q (available: %s)", flagServer, strings.Join(domain.ServerNames(), ", "))
		}
		return flagServer, serverFromFlag, nil
	}

	installed := h.installedServers()
	if len(installed) == 0 {
		return "", "", fmt.Errorf("no webserver installed — run: phpv server install caddy")
	}

	if pref, ok := h.serverPreference(phpVersion); ok && containsString(installed, pref) {
		return pref, serverFromPHPPreference, nil
	}

	if def := strings.TrimSpace(os.Getenv(defaultServerEnv)); def != "" {
		if !domain.IsValidServerName(def) {
			return "", "", fmt.Errorf("%s=%q is not a valid server (available: %s)", defaultServerEnv, def, strings.Join(domain.ServerNames(), ", "))
		}
		if !containsString(installed, def) {
			return "", "", fmt.Errorf("%s=%s is set but not installed — run: phpv server install %s", defaultServerEnv, def, def)
		}
		return def, serverFromEnv, nil
	}

	defaultVersion, _ := h.siloSvc.GetDefault()
	isDefaultPHP := defaultVersion == "" || domain.CompareVersions(defaultVersion, phpVersion) == 0
	if cfg, _ := h.configSvc.Get("server"); isDefaultPHP && cfg != "" && containsString(installed, cfg) {
		return cfg, serverFromConfig, nil
	}

	return "", "", fmt.Errorf(
		"no webserver associated with PHP %s (installed: %s)\n"+
			"  associate one: phpv server use <name> --php %s\n"+
			"  or set a default: %s=<name>\n"+
			"  or override for this run: --server <name>",
		phpVersion, strings.Join(installed, ", "), phpVersion, defaultServerEnv,
	)
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
