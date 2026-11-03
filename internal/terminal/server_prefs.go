package terminal

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/webserver"
)

// serverPreferenceFile records which webserver belongs to which PHP version.
// `phpv server install` writes it for the active PHP version and
// `phpv server run` reads it back, so a per-shell `phpv use` also switches
// the webserver instead of leaking the global (default-PHP) server choice.
type serverPreferenceFile struct {
	Servers map[string]string `json:"servers,omitempty"`
	SAPIs   map[string]string `json:"sapis,omitempty"`
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
	if prefs.SAPIs == nil {
		prefs.SAPIs = map[string]string{}
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

func (p serverPreferenceFile) getSAPI(phpVersion string) (string, bool) {
	if phpVersion == "" {
		return "", false
	}
	sapi, ok := p.SAPIs[phpVersion]
	return sapi, ok && sapi != ""
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

func (p *serverPreferenceFile) setSAPI(phpVersion, sapi string) {
	if phpVersion == "" {
		return
	}
	if sapi == "" || sapi == "fpm" {
		delete(p.SAPIs, phpVersion)
		return
	}
	if p.SAPIs == nil {
		p.SAPIs = map[string]string{}
	}
	p.SAPIs[phpVersion] = sapi
}

// serverPreference returns the webserver recorded for a PHP version.
func (h *PHPHandler) serverPreference(phpVersion string) (string, bool) {
	return loadServerPreferences(resolvePHPVRootForServer()).get(phpVersion)
}

// recordServerPreference remembers the webserver chosen for a PHP version.
func (h *PHPHandler) recordServerPreference(phpVersion, server string) error {
	return h.recordServerSelection(phpVersion, server, "")
}

// recordServerSelection remembers both the webserver and SAPI for a version.
// An empty/fpm SAPI clears any previous mod_php choice.
func (h *PHPHandler) recordServerSelection(phpVersion, server, sapi string) error {
	if phpVersion == "" || server == "" {
		return nil
	}
	root := resolvePHPVRootForServer()
	prefs := loadServerPreferences(root)
	prefs.set(phpVersion, server)
	prefs.setSAPI(phpVersion, sapi)
	return saveServerPreferences(root, prefs)
}

// serverSAPI returns the SAPI recorded for a PHP version ("fpm" default).
func (h *PHPHandler) serverSAPI(phpVersion string) string {
	prefs := loadServerPreferences(resolvePHPVRootForServer())
	if sapi, ok := prefs.getSAPI(phpVersion); ok {
		return sapi
	}
	return "fpm"
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
	serverFromFallback      serverSelectionSource = "fallback"
)

// serverSourceLabel humanizes why a server was selected.
func serverSourceLabel(source serverSelectionSource) string {
	switch source {
	case serverFromFlag:
		return "--server"
	case serverFromPHPPreference:
		return "per-PHP association"
	case serverFromEnv:
		return defaultServerEnv
	case serverFromConfig:
		return "global default"
	case serverFromFallback:
		return "fallback"
	default:
		return string(source)
	}
}

// defaultServerEnv names the fallback webserver for PHP versions that have no
// explicit association. It is never written implicitly by `server install`.
const defaultServerEnv = "PHPV_DEFAULT_SERVER"

// defaultSAPIEnv names the fallback PHP SAPI for versions without an
// association (fpm or modphp).
const defaultSAPIEnv = "PHPV_DEFAULT_SAPI"

// resolveSAPIForPHP picks the PHP execution mode for a run. A --sapi flag
// wins; otherwise an associated PHP version uses its recorded SAPI (fpm when
// none was recorded), and unassociated versions fall back to
// PHPV_DEFAULT_SAPI, then the global server_sapi config.
func (h *PHPHandler) resolveSAPIForPHP(flagSAPI, server, phpVersion string) (string, error) {
	explicit := strings.TrimSpace(flagSAPI) != ""
	sapi := webserver.NormalizeSAPI(flagSAPI)

	if !explicit {
		prefs := loadServerPreferences(resolvePHPVRootForServer())
		if _, hasServer := prefs.get(phpVersion); hasServer {
			if recorded, ok := prefs.getSAPI(phpVersion); ok {
				sapi = webserver.NormalizeSAPI(recorded)
			} else {
				sapi = "fpm"
			}
		} else if env := strings.TrimSpace(os.Getenv(defaultSAPIEnv)); env != "" {
			sapi = webserver.NormalizeSAPI(env)
		} else if cfg, _ := h.configSvc.Get("server_sapi"); cfg != "" {
			sapi = webserver.NormalizeSAPI(cfg)
		} else {
			sapi = "fpm"
		}
	}

	if sapi != "fpm" && sapi != "modphp" {
		return "", fmt.Errorf("unknown SAPI %q (available: fpm, modphp)", sapi)
	}
	if sapi == "modphp" && server != domain.ServerApache {
		if explicit {
			return "", fmt.Errorf("SAPI modphp is only supported with apache (got %s)", server)
		}
		// A global default may not apply to the selected server; fall back
		// to FPM rather than blocking the run.
		sapi = "fpm"
	}
	return sapi, nil
}

// resolveServerForPHP picks the webserver for a run. Resolution is aware of
// the active PHP version: a per-version association wins, then an explicit
// PHPV_DEFAULT_SERVER, then the global default set with `phpv server default`,
// then a deterministic fallback in ServerNames order. `server use` is never
// required.
func (h *PHPHandler) resolveServerForPHP(flagServer, phpVersion string) (string, serverSelectionSource, error) {
	if flagServer != "" {
		if !domain.IsValidServerName(flagServer) {
			return "", "", invalidServerName(flagServer)
		}
		return flagServer, serverFromFlag, nil
	}

	installed := h.installedServers()
	if len(installed) == 0 {
		return "", "", fmt.Errorf("no webserver installed — run: phpv server install <caddy|nginx|apache>")
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

	// `phpv server default <name>` writes this global default. It applies to
	// every PHP version that has no per-version association.
	if cfg, _ := h.configSvc.Get("server"); cfg != "" && containsString(installed, cfg) {
		return cfg, serverFromConfig, nil
	}

	// Deterministic last resort: first installed in ServerNames preference
	// order (caddy, nginx, apache). Never "newest wins", so installing another
	// server cannot change what an unassociated version uses.
	return installed[0], serverFromFallback, nil
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

// invalidServerName explains that the server commands take a webserver name,
// not a PHP version (a common mix-up with `phpv server install 7`).
func invalidServerName(name string) error {
	return fmt.Errorf(
		"unknown server %q — expected a webserver name (%s); PHP versions are switched with `phpv use <version>`",
		name, strings.Join(domain.ServerNames(), ", "),
	)
}
