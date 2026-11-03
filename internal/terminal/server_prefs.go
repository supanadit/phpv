package terminal

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

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

// newestInstalledServer returns the webserver whose install finished most
// recently. It is the fallback for PHP versions without a recorded
// preference, e.g. servers installed before preferences existed.
func (h *PHPHandler) newestInstalledServer() (string, error) {
	root := resolvePHPVRootForServer()
	var (
		bestName string
		bestTime time.Time
	)
	for _, name := range domain.ServerNames() {
		pkgDir := filepath.Join(root, "packages", name)
		entries, err := os.ReadDir(pkgDir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			installedAt := serverInstallTime(filepath.Join(pkgDir, entry.Name()))
			if installedAt.IsZero() {
				continue
			}
			if bestName == "" || installedAt.After(bestTime) {
				bestName, bestTime = name, installedAt
			}
		}
	}
	if bestName == "" {
		return "", fmt.Errorf("no webserver installed — run: phpv server install caddy")
	}
	return bestName, nil
}

// serverInstallTime approximates when a server version finished installing.
// MarkComplete writes the .state marker last, so prefer it over the
// directory's own mtime.
func serverInstallTime(versionDir string) time.Time {
	var best time.Time
	if info, err := os.Stat(versionDir); err == nil {
		best = info.ModTime()
	}
	if info, err := os.Stat(filepath.Join(versionDir, ".state")); err == nil && info.ModTime().After(best) {
		best = info.ModTime()
	}
	return best
}

// serverSelectionSource explains why a webserver was picked, so the command
// can tell the user when it is falling back.
type serverSelectionSource string

const (
	serverFromFlag          serverSelectionSource = "flag"
	serverFromPHPPreference serverSelectionSource = "php-preference"
	serverFromConfig        serverSelectionSource = "config"
)

// serverConfiguredFor reports whether a webserver is configured for a PHP
// version. A version is configured when `phpv server install` recorded a
// server for it, or when it is the default version and the global `server`
// config names an installed server.
func (h *PHPHandler) serverConfiguredFor(phpVersion string) bool {
	installed := h.installedServers()
	if len(installed) == 0 {
		return false
	}
	if pref, ok := h.serverPreference(phpVersion); ok && containsString(installed, pref) {
		return true
	}
	defaultVersion, _ := h.siloSvc.GetDefault()
	isDefaultPHP := defaultVersion == "" || domain.CompareVersions(defaultVersion, phpVersion) == 0
	if cfg, _ := h.configSvc.Get("server"); isDefaultPHP && cfg != "" && containsString(installed, cfg) {
		return true
	}
	return false
}

// suggestServer returns the webserver to recommend in "install first" hints.
// The most recently installed server best reflects the user's latest intent.
func (h *PHPHandler) suggestServer() string {
	installed := h.installedServers()
	if len(installed) == 0 {
		return domain.ServerCaddy
	}
	if newest, err := h.newestInstalledServer(); err == nil {
		return newest
	}
	if cfg, _ := h.configSvc.Get("server"); containsString(installed, cfg) {
		return cfg
	}
	return installed[0]
}

// noteMissingServer prints a one-line hint when a PHP version has no
// webserver configured for it. It goes to stderr because `phpv use --print`
// output is eval'd by the shell.
func (h *PHPHandler) noteMissingServer(phpVersion string) {
	if h.serverConfiguredFor(phpVersion) {
		return
	}
	fmt.Fprintf(os.Stderr, "note: no webserver configured for PHP %s — run: phpv server install %s\n", phpVersion, h.suggestServer())
}

// resolveServerForPHP picks the webserver for a run. Servers are only used
// for a PHP version that has one configured, so switching versions never
// silently reuses a server chosen for another version. Resolution order:
//
//	--server flag → preference recorded for this PHP version →
//	global config (default PHP only) → error telling the user to install one.
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

	defaultVersion, _ := h.siloSvc.GetDefault()
	isDefaultPHP := defaultVersion == "" || domain.CompareVersions(defaultVersion, phpVersion) == 0
	if cfg, _ := h.configSvc.Get("server"); isDefaultPHP && cfg != "" && containsString(installed, cfg) {
		return cfg, serverFromConfig, nil
	}

	return "", "", fmt.Errorf(
		"no webserver configured for PHP %s — run: phpv server install %s --php %s (or pass --server)",
		phpVersion, h.suggestServer(), phpVersion,
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
