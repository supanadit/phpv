// Package webserver installs and runs local webservers (caddy, nginx,
// apache) that serve PHP applications through FastCGI or Apache's embedded
// mod_php. Instances are fully isolated per PHP version: each run pins one
// PHP version and gets its own runtime (FPM master + socket, or a mod_php
// module), config set and log directory, so several PHP versions can be
// served simultaneously.
package webserver

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	vsn "github.com/supanadit/phpv/pkg/version"
	webidentity "github.com/supanadit/phpv/pkg/webidentity"

	"github.com/supanadit/phpv/domain"
)

// Deps is what the Manager needs from the rest of phpv.
type Deps struct {
	// Root is the phpv root ($PHPV_ROOT).
	Root string
	// ResolveVersion resolves "5.6"/"" to an exact installed php version.
	ResolveVersion func(constraint string) (string, error)
	// EnsureFPM installs (when needed) and returns the php-fpm binary.
	EnsureFPM func(ctx context.Context, phpVersion string, verbose bool, jobs int) (string, error)
	// EnsureApacheModule builds/returns the mod_php libphp.so for the PHP
	// version, rebuilding httpd with a prefork MPM when required.
	EnsureApacheModule func(ctx context.Context, phpVersion, httpdPrefix string, opts RunOptions) (string, error)
	// CLIPrefix returns the install prefix of the CLI php for a version.
	CLIPrefix func(phpVersion string) string
	// InstalledServerVersion resolves a server version constraint against
	// installed servers, or "" when none match.
	InstalledServerVersion func(server, constraint string) (string, error)
	// ServerPrefix returns the install prefix of an installed server.
	ServerPrefix func(server, version string) string
}

// Manager installs, starts, stops and inspects server instances.
type Manager struct {
	deps    Deps
	store   *Store
	drivers map[string]driver
}

// NewManager wires a Manager for the given deps.
func NewManager(deps Deps) *Manager {
	serversDir := StateDir(deps.Root)
	m := &Manager{
		deps:    deps,
		store:   NewStore(serversDir),
		drivers: make(map[string]driver),
	}
	for _, name := range webidentity.ServerNames() {
		m.drivers[name] = driver{name: name}
	}
	return m
}

// Store exposes the instance store.
func (m *Manager) Store() *Store { return m.store }

// driverFor returns the driver for a server name.
func (m *Manager) driverFor(name string) (driver, error) {
	d, ok := m.drivers[name]
	if !ok {
		return driver{}, fmt.Errorf("unknown server %q (available: %s)", name, strings.Join(webidentity.ServerNames(), ", "))
	}
	return d, nil
}

// RunOptions describes one `phpv server run` invocation.
type RunOptions = domain.ServerRunOptions

// normalizeSAPI canonicalizes a user-supplied SAPI value.
func NormalizeSAPI(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "fpm":
		return "fpm"
	case "modphp", "mod_php", "apache":
		return "modphp"
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

// hasSharedMPM reports whether an installed httpd ships a shared MPM module.
func hasSharedMPM(prefix, name string) bool {
	_, err := os.Stat(filepath.Join(prefix, "modules", "mod_mpm_"+name+".so"))
	return err == nil
}

// Start brings up an instance: PHP runtime, then the server, then records state.
// Foreground runs block until interrupted (processes are stopped on the
// way out); detached runs return the instance once serving.
func (m *Manager) Start(ctx context.Context, opts RunOptions) (domain.ServerInstance, error) {
	if st, err := os.Stat(opts.Dir); err != nil || !st.IsDir() {
		return domain.ServerInstance{}, fmt.Errorf("document root %q is not a directory", opts.Dir)
	}
	// A public/ subdirectory replaces the `php -S localhost:... -t public`
	// workflow this command substitutes for; when present it becomes the
	// document root (like every mainstream app skeleton).
	if pub := filepath.Join(opts.Dir, "public"); opts.Dir != pub {
		if st, err := os.Stat(pub); err == nil && st.IsDir() {
			opts.Dir = pub
		}
	}

	// Resolve the PHP version and the server before selecting the SAPI.
	phpVersion, err := m.deps.ResolveVersion(opts.PHPVersion)
	if err != nil {
		return domain.ServerInstance{}, fmt.Errorf("resolve php version: %w", err)
	}
	sapi := NormalizeSAPI(opts.SAPI)
	if sapi != "fpm" && sapi != "modphp" {
		return domain.ServerInstance{}, fmt.Errorf("unknown SAPI %q (available: fpm, modphp)", opts.SAPI)
	}
	if sapi == "modphp" && opts.Server != domain.ServerApache {
		return domain.ServerInstance{}, fmt.Errorf("--sapi modphp is only supported with apache (got %s)", opts.Server)
	}

	drv, err := m.driverFor(opts.Server)
	if err != nil {
		return domain.ServerInstance{}, err
	}
	serverVersion, err := m.deps.InstalledServerVersion(opts.Server, opts.ServerVersion)
	if err != nil {
		return domain.ServerInstance{}, fmt.Errorf("resolve %s version: %w", opts.Server, err)
	}
	prefix := m.deps.ServerPrefix(opts.Server, serverVersion)
	serverBin := drv.Binary(prefix)
	if serverBin == "" {
		return domain.ServerInstance{}, fmt.Errorf("%s binary not found in %s (install it: phpv server install %s)", opts.Server, prefix, opts.Server)
	}

	// Ensure the PHP runtime for the selected SAPI.
	var (
		fpmBin       string
		apacheModule string
	)
	if sapi == "modphp" {
		apacheModule, err = m.deps.EnsureApacheModule(ctx, phpVersion, prefix, opts)
		if err != nil {
			return domain.ServerInstance{}, err
		}
	} else {
		fpmBin, err = m.deps.EnsureFPM(ctx, phpVersion, opts.Verbose, opts.Jobs)
		if err != nil {
			return domain.ServerInstance{}, err
		}
	}

	port, err := PickFreePort(opts.Port)
	if err != nil {
		return domain.ServerInstance{}, err
	}

	id := InstanceID(opts.Server, phpVersion, port, sapi)
	inst := domain.ServerInstance{
		ID:            id,
		Server:        opts.Server,
		ServerVersion: serverVersion,
		PHPVersion:    phpVersion,
		SAPI:          sapi,
		Dir:           opts.Dir,
		Host:          opts.Host,
		Port:          port,
		Socket:        filepath.Join(StateDir(m.deps.Root), "state", id+".sock"),
		StateDir:      filepath.Join(StateDir(m.deps.Root), "instances", id),
		Detached:      opts.Detach,
		StartedAt:     time.Now(),
	}

	// Refuse double-binds: same server+port already recorded and alive.
	if existing, err := m.store.Load(inst.ID); err == nil && existing.ID != "" && m.store.Running(existing) {
		return domain.ServerInstance{}, fmt.Errorf("instance %s is already running (phpv server stop %s)", existing.ID, existing.ID)
	}

	// --host: vhost + /etc/hosts.
	if opts.Host != "" && opts.Host != "localhost" {
		if err := EnsureHostEntry(opts.Host); err != nil {
			var permErr *HostsPermissionError
			if asHostsPermission(err, &permErr) {
				fmt.Printf("  note: %s\n        %s\n", permErr.Error(), permErr.ManualInstruction())
			} else {
				return domain.ServerInstance{}, err
			}
		}
	}

	// Prepare state dirs and socket hygiene. The socket lives in
	// servers/state/ (alongside instance records), so that directory must
	// exist before php-fpm binds. mod_php has no socket.
	for _, d := range []string{filepath.Dir(inst.Socket), inst.StateDir, filepath.Join(inst.StateDir, "logs"), filepath.Join(inst.StateDir, "tmp")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return domain.ServerInstance{}, err
		}
	}
	if sapi != "modphp" {
		_ = os.Remove(inst.Socket)
	}

	iniDir, extDir := CLIRuntime(m.deps.CLIPrefix(phpVersion))

	var fpmConf string
	if sapi != "modphp" {
		// Render and start the FPM master. The CLI php.ini and
		// extension_dir are passed so FPM matches `php -S`.
		fpmConf, err = m.renderFPM(inst, fpmBin)
		if err != nil {
			return domain.ServerInstance{}, err
		}
		fpmLogPath := filepath.Join(inst.StateDir, "fpm.log")
		fpmCmd := execFPM(fpmBin, fpmConf, iniDir, extDir)
		if opts.Detach {
			fpmPID, err := Detach(fpmCmd, fpmLogPath)
			if err != nil {
				return domain.ServerInstance{}, fmt.Errorf("start php-fpm: %w", err)
			}
			inst.FPMPID = fpmPID
		} else {
			logFile, logErr := os.OpenFile(fpmLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
			if logErr != nil {
				return domain.ServerInstance{}, logErr
			}
			fpmCmd.Stdout = logFile
			fpmCmd.Stderr = logFile
			if err := fpmCmd.Start(); err != nil {
				logFile.Close()
				return domain.ServerInstance{}, fmt.Errorf("start php-fpm: %w", err)
			}
			inst.FPMPID = fpmCmd.Process.Pid
			go func() { _ = fpmCmd.Wait(); _ = logFile.Close() }()
		}
		if err := waitSocket(inst.Socket, 15*time.Second); err != nil {
			m.Stop(inst)
			return domain.ServerInstance{}, fmt.Errorf("php-fpm did not open %s: %w", inst.Socket, err)
		}
	}

	mpm := ""
	if opts.Server == domain.ServerApache {
		if sapi == "modphp" {
			mpm = "prefork"
		} else if hasSharedMPM(prefix, "event") {
			mpm = "event"
		}
	}
	runtime := Runtime{
		SAPI:         sapi,
		FPMConf:      fpmConf,
		ApacheModule: apacheModule,
		PHPIniDir:    iniDir,
		ExtensionDir: extDir,
		MPM:          mpm,
	}

	// Start the webserver.
	confPath, err := drv.WriteConfig(inst, runtime)
	if err != nil {
		m.Stop(inst)
		return domain.ServerInstance{}, err
	}
	args := drv.StartArgs(inst, confPath)
	srvCmd := command(serverBin, args...)
	if opts.Detach {
		srvPID, err := Detach(srvCmd, filepath.Join(inst.StateDir, "server.log"))
		if err != nil {
			m.Stop(inst)
			return domain.ServerInstance{}, fmt.Errorf("start %s: %w", opts.Server, err)
		}
		inst.ServerPID = srvPID
	} else {
		if err := srvCmd.Start(); err != nil {
			m.Stop(inst)
			return domain.ServerInstance{}, fmt.Errorf("start %s: %w", opts.Server, err)
		}
		inst.ServerPID = srvCmd.Process.Pid
		go func() { _ = srvCmd.Wait() }()
	}
	if err := WaitPort(port, 15*time.Second); err != nil {
		m.Stop(inst)
		return domain.ServerInstance{}, err
	}

	if err := m.store.Save(inst); err != nil {
		m.Stop(inst)
		return domain.ServerInstance{}, err
	}
	return inst, nil
}

// renderFPM builds the FPMConfig for an instance. Runtime ini settings and
// extension_dir are supplied on the php-fpm command line so FPM reads the
// exact same php.ini as the CLI install.
func (m *Manager) renderFPM(inst domain.ServerInstance, fpmBin string) (string, error) {
	legacy := vsn.CompareVersions(inst.PHPVersion, "7.0.0") < 0
	conf, err := RenderFPMConf(FPMConfig{
		Binary:     fpmBin,
		PHPVersion: inst.PHPVersion,
		Socket:     inst.Socket,
		PIDFile:    filepath.Join(inst.StateDir, "fpm.pid"),
		ErrorLog:   filepath.Join(inst.StateDir, "logs", "fpm-error.log"),
		SlowLog:    filepath.Join(inst.StateDir, "logs", "fpm-slow.log"),
		StateDir:   inst.StateDir,
		Legacy:     legacy,
	})
	return conf, err
}

// Stop terminates a running instance: server first, then FPM.
func (m *Manager) Stop(inst domain.ServerInstance) error {
	_ = SignalTerm(inst.ServerPID)
	AwaitExit(inst.ServerPID, stopGrace)
	_ = StopFPM(filepath.Join(inst.StateDir, "fpm.pid"), inst.FPMPID)
	AwaitExit(inst.FPMPID, stopGrace)
	_ = os.Remove(inst.Socket)
	if inst.Host != "" && inst.Host != "localhost" {
		_ = RemoveHostEntry(inst.Host)
	}
	return m.store.Remove(inst.ID)
}

// StopByID stops a recorded instance by id.
func (m *Manager) StopByID(id string) (domain.ServerInstance, error) {
	inst, err := m.store.Load(id)
	if err != nil {
		return inst, err
	}
	if inst.ID == "" {
		return inst, fmt.Errorf("no such instance: %s", id)
	}
	return inst, m.Stop(inst)
}

// List returns all stored instances, marking liveness.
func (m *Manager) List() ([]domain.ServerInstance, error) {
	instances, err := m.store.List()
	if err != nil {
		return nil, err
	}
	sort.Slice(instances, func(i, j int) bool {
		if instances[i].StartedAt.Equal(instances[j].StartedAt) {
			return instances[i].ID < instances[j].ID
		}
		return instances[i].StartedAt.Before(instances[j].StartedAt)
	})
	return instances, nil
}

// Running returns only live instances.
func (m *Manager) Running() ([]domain.ServerInstance, error) {
	all, err := m.List()
	if err != nil {
		return nil, err
	}
	var live []domain.ServerInstance
	for _, inst := range all {
		if m.store.Running(inst) {
			live = append(live, inst)
		}
	}
	return live, nil
}

// Prune removes records whose processes are all dead.
func (m *Manager) Prune() (int, error) {
	all, err := m.List()
	if err != nil {
		return 0, err
	}
	pruned := 0
	for _, inst := range all {
		if !m.store.Running(inst) {
			if err := m.store.Remove(inst.ID); err == nil {
				pruned++
			}
		}
	}
	return pruned, nil
}

// Runtime describes how PHP is executed for an instance.
type Runtime struct {
	// SAPI is "fpm" (default) or "modphp".
	SAPI string
	// FPMConf is the generated php-fpm.conf (FPM only).
	FPMConf string
	// ApacheModule is the absolute libphp.so path (mod_php only).
	ApacheModule string
	// PHPIniDir is the CLI install's php.ini directory.
	PHPIniDir string
	// ExtensionDir is the CLI install's extension_dir.
	ExtensionDir string
	// MPM is the Apache MPM to load when built as a shared module
	// ("event" or "prefork"); empty means the MPM is static.
	MPM string
}

// driver is the per-server behavior: binary lookup, config generation and
// start arguments. The Manager selects one by its domain server name.
type driver struct{ name string }

func (d driver) Binary(prefix string) string {
	switch d.name {
	case domain.ServerCaddy:
		return FindBinary(prefix, "caddy")
	case domain.ServerNginx:
		return FindBinary(prefix, "nginx")
	default:
		return FindBinary(prefix, "httpd")
	}
}

func (d driver) WriteConfig(inst domain.ServerInstance, rt Runtime) (string, error) {
	switch d.name {
	case domain.ServerCaddy:
		return writeCaddyConfig(inst)
	case domain.ServerNginx:
		return writeNginxConfig(inst)
	default:
		return writeApacheConfig(inst, rt)
	}
}

func (d driver) StartArgs(inst domain.ServerInstance, confPath string) []string {
	switch d.name {
	case domain.ServerCaddy:
		return []string{"run", "--config", confPath, "--adapter", "caddyfile"}
	case domain.ServerNginx:
		return []string{"-c", confPath}
	default:
		return []string{"-f", confPath, "-D", "FOREGROUND"}
	}
}

// writeCaddyConfig serves PHP through php_fastcgi (reverse_proxy to FastCGI).
func writeCaddyConfig(inst domain.ServerInstance) (string, error) {
	if err := os.MkdirAll(inst.StateDir, 0o755); err != nil {
		return "", err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "{\n  admin off\n  auto_https off\n}\n\n")
	b.WriteString("{PORT} {\n")
	host := inst.Host
	if host != "" && host != "localhost" {
		// Serve on the vhost name; port binding comes from the site label.
		fmt.Fprintf(&b, "  # vhost: %s\n", host)
	}
	fmt.Fprintf(&b, "  root * %s\n", inst.Dir)
	b.WriteString("  encode gzip\n")
	fmt.Fprintf(&b, "  php_fastcgi unix/%s\n", inst.Socket)
	b.WriteString("  file_server\n")
	b.WriteString("}\n")
	conf := filepath.Join(inst.StateDir, "Caddyfile")
	rendered := strings.ReplaceAll(b.String(), "{PORT}", ":"+fmt.Sprintf("%d", inst.Port))
	if err := os.WriteFile(conf, []byte(rendered), 0o644); err != nil {
		return "", fmt.Errorf("write Caddyfile: %w", err)
	}
	return conf, nil
}

// writeNginxConfig serves PHP via FastCGI pass to the instance socket.
func writeNginxConfig(inst domain.ServerInstance) (string, error) {
	if err := os.MkdirAll(filepath.Join(inst.StateDir, "logs"), 0o755); err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Join(inst.StateDir, "tmp", "client"), 0o755); err != nil {
		return "", err
	}
	// nginx resolves bare `include` paths against its --prefix/conf dir,
	// which an isolated silo install does not populate; ship the params
	// file next to the generated config instead.
	if err := os.WriteFile(filepath.Join(inst.StateDir, "fastcgi_params"), []byte(fastcgiParams), 0o644); err != nil {
		return "", fmt.Errorf("write fastcgi_params: %w", err)
	}
	var b strings.Builder
	b.WriteString("worker_processes 1;\n")
	b.WriteString("daemon off;\n")
	b.WriteString("pid " + inst.StateDir + "/nginx.pid;\n")
	b.WriteString("error_log " + inst.StateDir + "/logs/error.log info;\n\n")
	b.WriteString("events { worker_connections 256; }\n\n")
	b.WriteString("http {\n")
	b.WriteString("  access_log " + inst.StateDir + "/logs/access.log;\n")
	b.WriteString("  client_body_temp_path " + inst.StateDir + "/tmp/client;\n")
	b.WriteString("  proxy_temp_path " + inst.StateDir + "/tmp/proxy;\n")
	b.WriteString("  fastcgi_temp_path " + inst.StateDir + "/tmp/fastcgi;\n")
	b.WriteString("  uwsgi_temp_path " + inst.StateDir + "/tmp/uwsgi;\n")
	b.WriteString("  scgi_temp_path " + inst.StateDir + "/tmp/scgi;\n\n")
	// The silo install does not ship nginx's conf/mime.types, so declare
	// the common web types inline; anything else falls back to
	// application/octet-stream instead of text/plain.
	b.WriteString("  default_type application/octet-stream;\n")
	b.WriteString("  types {\n")
	for _, m := range nginxMimeTypes {
		fmt.Fprintf(&b, "    %s;\n", m)
	}
	b.WriteString("  }\n\n")
	fmt.Fprintf(&b, "  server {\n")
	fmt.Fprintf(&b, "    listen %d;\n", inst.Port)
	if inst.Host != "" && inst.Host != "localhost" {
		fmt.Fprintf(&b, "    server_name %s;\n", inst.Host)
	}
	fmt.Fprintf(&b, "    root %s;\n", inst.Dir)
	b.WriteString("    index index.php index.html;\n\n")
	b.WriteString("    location / {\n      try_files $uri $uri/ /index.php?$query_string;\n    }\n\n")
	b.WriteString("    location ~ \\.php$ {\n")
	b.WriteString("      try_files $uri =404;\n")
	fmt.Fprintf(&b, "      fastcgi_pass unix:%s;\n", inst.Socket)
	b.WriteString("      fastcgi_index index.php;\n")
	b.WriteString("      include " + filepath.Join(inst.StateDir, "fastcgi_params") + ";\n")
	b.WriteString("      fastcgi_param SCRIPT_FILENAME $document_root$fastcgi_script_name;\n")
	b.WriteString("    }\n")
	b.WriteString("  }\n")
	b.WriteString("}\n")
	conf := filepath.Join(inst.StateDir, "nginx.conf")
	if err := os.WriteFile(conf, []byte(b.String()), 0o644); err != nil {
		return "", fmt.Errorf("write nginx.conf: %w", err)
	}
	return conf, nil
}

// writeApacheConfig serves PHP either through mod_proxy_fcgi (FPM) or the
// embedded mod_php module, depending on the instance SAPI.
func writeApacheConfig(inst domain.ServerInstance, rt Runtime) (string, error) {
	if err := os.MkdirAll(filepath.Join(inst.StateDir, "logs"), 0o755); err != nil {
		return "", err
	}
	sapi := rt.SAPI
	if sapi == "" {
		sapi = "fpm"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "ServerName %s\n", hostOrLocalhost(inst.Host))
	fmt.Fprintf(&b, "Listen %d\n", inst.Port)
	b.WriteString("PidFile " + inst.StateDir + "/httpd.pid\n")
	b.WriteString("ErrorLog " + inst.StateDir + "/logs/error.log\n")
	b.WriteString("CustomLog " + inst.StateDir + "/logs/access.log combined\n\n")

	// MPM: phpv builds httpd with shared event and prefork modules when
	// available. mod_php requires prefork (it is not thread-safe).
	if rt.MPM != "" {
		fmt.Fprintf(&b, "LoadModule mpm_%s_module modules/mod_mpm_%s.so\n", rt.MPM, rt.MPM)
	}
	b.WriteString("LoadModule authz_core_module modules/mod_authz_core.so\n")
	b.WriteString("LoadModule unixd_module modules/mod_unixd.so\n")
	b.WriteString("LoadModule log_config_module modules/mod_log_config.so\n")
	b.WriteString("LoadModule mime_module modules/mod_mime.so\n")
	b.WriteString("LoadModule dir_module modules/mod_dir.so\n")
	b.WriteString("LoadModule setenvif_module modules/mod_setenvif.so\n")
	// mod_rewrite: AllowOverride All is set below, so apps shipping a
	// .htaccess with rewrite rules (WordPress, Laravel, most PHP apps)
	// 500 with "Invalid command 'RewriteEngine'" without it.
	b.WriteString("LoadModule rewrite_module modules/mod_rewrite.so\n")

	switch sapi {
	case "modphp":
		fmt.Fprintf(&b, "LoadModule %s %s\n", apachePHPModuleName(inst.PHPVersion), rt.ApacheModule)
		if rt.PHPIniDir != "" {
			fmt.Fprintf(&b, "PHPIniDir \"%s\"\n", rt.PHPIniDir)
		}
		if rt.ExtensionDir != "" {
			fmt.Fprintf(&b, "php_admin_value extension_dir \"%s\"\n", rt.ExtensionDir)
		}
		b.WriteString("AddHandler application/x-httpd-php .php\n\n")
	default: // fpm
		b.WriteString("LoadModule proxy_module modules/mod_proxy.so\n")
		b.WriteString("LoadModule proxy_fcgi_module modules/mod_proxy_fcgi.so\n")
	}
	b.WriteString("\nTypesConfig conf/mime.types\n")
	b.WriteString("DirectoryIndex index.php index.html\n\n")
	fmt.Fprintf(&b, "DocumentRoot \"%s\"\n", inst.Dir)
	fmt.Fprintf(&b, "<Directory \"%s\">\n", inst.Dir)
	b.WriteString("  Options Indexes FollowSymLinks\n")
	b.WriteString("  AllowOverride All\n")
	b.WriteString("  Require all granted\n")
	b.WriteString("</Directory>\n\n")
	if sapi == "modphp" {
		b.WriteString("<FilesMatch \\.php$>\n")
		b.WriteString("  SetHandler application/x-httpd-php\n")
		b.WriteString("</FilesMatch>\n")
	} else {
		b.WriteString("<FilesMatch \\.php$>\n")
		fmt.Fprintf(&b, "  SetHandler \"proxy:unix:%s|fcgi://localhost\"\n", inst.Socket)
		b.WriteString("</FilesMatch>\n")
	}
	conf := filepath.Join(inst.StateDir, "httpd.conf")
	if err := os.WriteFile(conf, []byte(b.String()), 0o644); err != nil {
		return "", fmt.Errorf("write httpd.conf: %w", err)
	}
	return conf, nil
}

func hostOrLocalhost(h string) string {
	if h == "" {
		return "localhost"
	}
	return h
}

// apachePHPModuleName returns the LoadModule name for mod_php on this PHP
// version (php_module since PHP 8, php7_module on 7.x, php5_module on 5.x).
func apachePHPModuleName(phpVersion string) string {
	switch {
	case vsn.CompareVersions(phpVersion, "8.0.0") >= 0:
		return "php_module"
	case vsn.CompareVersions(phpVersion, "7.0.0") >= 0:
		return "php7_module"
	default:
		return "php5_module"
	}
}

// nginxMimeTypes maps extensions to MIME types for the inline `types`
// block written into every generated nginx.conf (format: "type ext ext").
var nginxMimeTypes = []string{
	"text/html html htm",
	"text/css css",
	"text/javascript js mjs",
	"text/plain txt",
	"text/markdown md",
	"application/json json map",
	"application/manifest+json webmanifest",
	"application/wasm wasm",
	"application/xml xml",
	"application/pdf pdf",
	"image/png png",
	"image/jpeg jpg jpeg",
	"image/gif gif",
	"image/svg+xml svg svgz",
	"image/webp webp",
	"image/avif avif",
	"image/x-icon ico",
	"font/woff woff",
	"font/woff2 woff2",
	"font/otf otf",
	"font/ttf ttf",
	"video/mp4 mp4",
	"video/webm webm",
	"audio/mpeg mp3",
	"audio/ogg ogg",
}

// fastcgiParams is the standard nginx FastCGI parameter set (from the
// nginx distribution's fastcgi_params file, minus the SCRIPT_FILENAME we
// set explicitly).
const fastcgiParams = `fastcgi_param  QUERY_STRING       $query_string;
fastcgi_param  REQUEST_METHOD     $request_method;
fastcgi_param  CONTENT_TYPE       $content_type;
fastcgi_param  CONTENT_LENGTH     $content_length;
fastcgi_param  REQUEST_URI        $request_uri;
fastcgi_param  DOCUMENT_URI       $document_uri;
fastcgi_param  DOCUMENT_ROOT      $document_root;
fastcgi_param  SERVER_PROTOCOL    $server_protocol;
fastcgi_param  REQUEST_SCHEME     $scheme;
fastcgi_param  HTTPS              $https if_not_empty;
fastcgi_param  GATEWAY_INTERFACE  CGI/1.1;
fastcgi_param  SERVER_SOFTWARE    nginx;
fastcgi_param  REMOTE_ADDR        $remote_addr;
fastcgi_param  REMOTE_PORT        $remote_port;
fastcgi_param  SERVER_ADDR        $server_addr;
fastcgi_param  SERVER_PORT        $server_port;
fastcgi_param  SERVER_NAME        $server_name;
fastcgi_param  REDIRECT_STATUS    200;
`

// FPMConfig is everything needed to render and run one php-fpm master for
// an instance. The CLI php.ini and extension_dir are supplied on the command
// line (see execFPM), so FPM runs with exactly the CLI's configuration.
type FPMConfig struct {
	Binary     string // php-fpm binary
	PHPVersion string
	Socket     string // unix socket path
	PIDFile    string
	ErrorLog   string
	SlowLog    string
	StateDir   string
	// PHP 5.3-era FPM lacks some pool directives; guard by version.
	Legacy bool // PHP < 7.0
}

// RenderFPMConf writes php-fpm.conf (global + pool) into stateDir and
// returns its path. The pool listens on the instance's unix socket and
// loads the CLI install's shared extensions via php_admin_value.
func RenderFPMConf(c FPMConfig) (string, error) {
	if err := os.MkdirAll(c.StateDir, 0o755); err != nil {
		return "", fmt.Errorf("create fpm state dir: %w", err)
	}
	var b strings.Builder
	b.WriteString("[global]\n")
	fmt.Fprintf(&b, "pid = %s\n", c.PIDFile)
	fmt.Fprintf(&b, "error_log = %s\n", c.ErrorLog)
	b.WriteString("daemonize = no\n\n")
	b.WriteString("[phpv]\n")
	fmt.Fprintf(&b, "listen = %s\n", c.Socket)
	b.WriteString("listen.mode = 0666\n")
	b.WriteString("pm = dynamic\n")
	b.WriteString("pm.max_children = 10\n")
	b.WriteString("pm.start_servers = 2\n")
	b.WriteString("pm.min_spare_servers = 1\n")
	b.WriteString("pm.max_spare_servers = 3\n")
	if !c.Legacy {
		fmt.Fprintf(&b, "slowlog = %s\n", c.SlowLog)
		b.WriteString("request_terminate_timeout = 120s\n")
	}
	fmt.Fprintf(&b, "php_admin_value[error_log] = %s\n", filepath.Join(c.StateDir, "php-error.log"))
	b.WriteString("php_admin_flag[log_errors] = on\n")
	b.WriteString("php_admin_value[error_reporting] = E_ALL\n")
	confPath := filepath.Join(c.StateDir, "php-fpm.conf")
	if err := os.WriteFile(confPath, []byte(b.String()), 0o644); err != nil {
		return "", fmt.Errorf("write php-fpm.conf: %w", err)
	}
	return confPath, nil
}

// StopFPM gracefully stops a php-fpm master via its pid file, falling back
// to SIGTERM on the recorded pid.
func StopFPM(pidFile string, pid int) error {
	if data, err := os.ReadFile(pidFile); err == nil {
		if p := parseInt(string(data)); p > 0 {
			pid = p
		}
	}
	return SignalTerm(pid)
}

func parseInt(s string) int {
	s = strings.TrimSpace(s)
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + int(c-'0')
	}
	return n
}

// execFPM builds the php-fpm foreground command for a rendered config. It
// reads the CLI install's php.ini (`-c`) and pins extension_dir so shared
// extensions resolve exactly like they do for `php -S`.
func execFPM(fpmBin, confPath, iniDir, extDir string) *exec.Cmd {
	args := []string{"-F", "-y", confPath}
	if iniDir != "" {
		args = append(args, "-c", iniDir)
	}
	if extDir != "" {
		args = append(args, "-d", "extension_dir="+extDir)
	}
	return exec.Command(fpmBin, args...)
}

// waitSocket polls until the unix socket accepts connections.
func waitSocket(socket string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(socket); err == nil {
			c, err := net.Dial("unix", socket)
			if err == nil {
				c.Close()
				return nil
			}
		}
		time.Sleep(150 * time.Millisecond)
	}
	return fmt.Errorf("socket %s not ready within %s", filepath.Base(socket), timeout)
}

// asHostsPermission wraps errors.As for *HostsPermissionError.
func asHostsPermission(err error, target **HostsPermissionError) bool {
	return errors.As(err, target)
}

const hostsMarker = "# managed by phpv server"

// hostsPath is a variable so tests can redirect it.
var hostsPath = "/etc/hosts"

// HostsHasEntry reports whether /etc/hosts already maps host.
func HostsHasEntry(host string) bool {
	f, err := os.Open(hostsPath)
	if err != nil {
		return false
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		for _, f := range fields[1:] {
			if f == host {
				return true
			}
		}
	}
	return false
}

// HostsLine returns the exact line phpv would add for host.
func HostsLine(host string) string {
	return fmt.Sprintf("127.0.0.1 %s %s", host, hostsMarker)
}

// EnsureHostEntry adds `127.0.0.1 <host>` to /etc/hosts when missing.
//
// Writing /etc/hosts needs root, so this shells out to sudo. When sudo is
// unavailable or non-interactive the caller gets a *HostsPermissionError
// carrying the manual instruction; serving continues regardless.
func EnsureHostEntry(host string) error {
	if host == "" || host == "localhost" {
		return nil
	}
	if HostsHasEntry(host) {
		return nil
	}
	line := HostsLine(host)
	if err := appendHostsAsUser(line); err == nil {
		return nil
	}
	if err := appendHostsViaSudo(line); err != nil {
		return &HostsPermissionError{Host: host, Line: line, Err: err}
	}
	return nil
}

// HostsPermissionError signals that phpv could not edit /etc/hosts itself.
type HostsPermissionError struct {
	Host string
	Line string
	Err  error
}

func (e *HostsPermissionError) Error() string {
	return fmt.Sprintf("could not add %s to %s: %v", e.Host, hostsPath, e.Err)
}

// ManualInstruction returns the copy-pasteable fix for the user.
func (e *HostsPermissionError) ManualInstruction() string {
	return fmt.Sprintf("add this line to %s manually:\n    %s", hostsPath, e.Line)
}

func appendHostsAsUser(line string) error {
	f, err := os.OpenFile(hostsPath, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = fmt.Fprintf(f, "\n%s\n", line)
	return err
}

func appendHostsViaSudo(line string) error {
	if _, err := lookPath("sudo"); err != nil {
		return fmt.Errorf("sudo not available")
	}
	// Use tee so sudo elevates only the write, never a shell.
	cmd := command("sudo", "-n", "tee", "-a", hostsPath)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Start(); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(stdin, "\n%s\n", line); err != nil {
		stdin.Close()
		_ = cmd.Wait()
		return err
	}
	stdin.Close()
	return cmd.Wait()
}

// RemoveHostEntry drops every phpv-managed line for host from /etc/hosts.
// Missing lines and missing permissions are not treated as errors.
func RemoveHostEntry(host string) error {
	if host == "" || host == "localhost" {
		return nil
	}
	data, err := os.ReadFile(hostsPath)
	if err != nil {
		return nil
	}
	var kept []string
	removed := false
	for _, line := range strings.Split(string(data), "\n") {
		if strings.Contains(line, hostsMarker) {
			fields := strings.Fields(line)
			match := false
			for _, f := range fields[1:] {
				if f == host {
					match = true
					break
				}
			}
			if match {
				removed = true
				continue
			}
		}
		kept = append(kept, line)
	}
	if !removed {
		return nil
	}
	tmp := filepath.Join(os.TempDir(), "phpv-hosts")
	if err := os.WriteFile(tmp, []byte(strings.Join(kept, "\n")), 0o644); err != nil {
		return err
	}
	if err := copyIntoHosts(tmp); err != nil {
		return err
	}
	return os.Remove(tmp)
}

// copyFileOver copies src to dst with the given mode (replacing content).
func copyFileOver(src, dst string, mode os.FileMode) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, mode)
}

// copyIntoHosts replaces /etc/hosts content with src (sudo when needed).
func copyIntoHosts(src string) error {
	if err := copyFileOver(src, hostsPath, 0o644); err == nil {
		return nil
	}
	if _, err := lookPath("sudo"); err != nil {
		return fmt.Errorf("sudo not available")
	}
	return command("sudo", "-n", "cp", src, hostsPath).Run()
}

// PickFreePort returns an available TCP port. When preferred is 0 the scan
// starts at the default server port 8080 and walks upward; otherwise only
// the preferred port is probed.
func PickFreePort(preferred int) (int, error) {
	start := preferred
	if start == 0 {
		start = 8080
	}
	for p := start; p < start+100; p++ {
		ln, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(p))
		if err == nil {
			ln.Close()
			return p, nil
		}
		if preferred != 0 {
			return 0, fmt.Errorf("port %d is not available", preferred)
		}
	}
	return 0, fmt.Errorf("no free port found in range %d-%d", start, start+99)
}

// FindBinary locates a command inside an installed package prefix, probing
// the usual layouts (bin/, sbin/, root). Falls back to exec.LookPath when
// the prefix is empty.
func FindBinary(prefix, name string) string {
	if prefix != "" {
		for _, rel := range []string{
			filepath.Join("bin", name),
			filepath.Join("sbin", name),
			name,
		} {
			p := filepath.Join(prefix, rel)
			if fi, err := os.Stat(p); err == nil && !fi.IsDir() && fi.Mode()&0o111 != 0 {
				return p
			}
		}
	}
	if path, err := exec.LookPath(name); err == nil {
		return path
	}
	return ""
}

// WaitPort polls until the TCP port accepts connections or the timeout
// expires. Used after server startup to confirm the instance is serving.
func WaitPort(port int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 500*time.Millisecond)
		if err == nil {
			conn.Close()
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("port %d did not open within %s", port, timeout)
}

// CLIRuntime returns the CLI install's php.ini directory and extension_dir.
// It is the single source of truth for both FPM (`-c`, `-d extension_dir`)
// and the Apache module (PHPIniDir, extension_dir), so `php -S` and the
// webserver load the same configuration and shared extensions.
func CLIRuntime(cliPrefix string) (iniDir, extDir string) {
	iniDir = filepath.Join(cliPrefix, "lib")
	phpConfig := filepath.Join(cliPrefix, "bin", "php-config")

	if out, err := exec.Command(phpConfig, "--ini-path").Output(); err == nil {
		if value := strings.TrimSpace(string(out)); value != "" {
			iniDir = value
		}
	}
	if out, err := exec.Command(phpConfig, "--extension-dir").Output(); err == nil {
		if value := strings.TrimSpace(string(out)); value != "" {
			extDir = value
		}
	}
	if extDir == "" {
		// Legacy fallback: the API directory is the only child of
		// lib/php/extensions.
		matches, _ := filepath.Glob(filepath.Join(cliPrefix, "lib", "php", "extensions", "*"))
		for _, match := range matches {
			if st, err := os.Stat(match); err == nil && st.IsDir() {
				extDir = match
				break
			}
		}
	}
	return iniDir, extDir
}

// SignalTerm sends SIGTERM to a pid, ignoring already-dead targets.
func SignalTerm(pid int) error {
	if pid <= 0 {
		return nil
	}
	err := syscall.Kill(pid, syscall.SIGTERM)
	if err != nil && err != syscall.ESRCH {
		return err
	}
	return nil
}

// AwaitExit waits up to grace for the pid to disappear, then SIGKILLs it.
func AwaitExit(pid int, grace time.Duration) {
	if pid <= 0 {
		return
	}
	deadline := time.Now().Add(grace)
	for time.Now().Before(deadline) {
		if !ProcessAlive(pid) {
			return
		}
		time.Sleep(150 * time.Millisecond)
	}
	_ = syscall.Kill(pid, syscall.SIGKILL)
}

// Detach starts cmd in its own session so it survives the phpv process,
// redirecting stdout/stderr to the given log file. Returns the child pid.
func Detach(cmd *exec.Cmd, logPath string) (int, error) {
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return 0, fmt.Errorf("open log %s: %w", logPath, err)
	}
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.Stdin = nil
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		logFile.Close()
		return 0, err
	}
	pid := cmd.Process.Pid
	// Release our handle: the daemon outlives this phpv invocation.
	go func() { _ = cmd.Wait() }()
	return pid, nil
}

// StateDir returns the directory holding server configs, instance state
// and logs under the phpv root.
func StateDir(root string) string {
	return filepath.Join(root, "servers")
}

// Store persists server instances as JSON files under
// <root>/servers/state/. One file per instance ID.
type Store struct {
	dir string
}

// NewStore creates an instance store rooted at the given servers dir.
func NewStore(serversDir string) *Store {
	return &Store{dir: filepath.Join(serversDir, "state")}
}

func (s *Store) path(id string) string {
	return filepath.Join(s.dir, id+".json")
}

// Save writes one instance record.
func (s *Store) Save(inst domain.ServerInstance) error {
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return fmt.Errorf("create state dir: %w", err)
	}
	data, err := json.MarshalIndent(inst, "", "  ")
	if err != nil {
		return fmt.Errorf("encode instance: %w", err)
	}
	return os.WriteFile(s.path(inst.ID), data, 0o644)
}

// Load reads one instance record. Returns (zero, nil) when absent.
func (s *Store) Load(id string) (domain.ServerInstance, error) {
	data, err := os.ReadFile(s.path(id))
	if err != nil {
		if os.IsNotExist(err) {
			return domain.ServerInstance{}, nil
		}
		return domain.ServerInstance{}, fmt.Errorf("read instance %s: %w", id, err)
	}
	var inst domain.ServerInstance
	if err := json.Unmarshal(data, &inst); err != nil {
		return domain.ServerInstance{}, fmt.Errorf("decode instance %s: %w", id, err)
	}
	return inst, nil
}

// List returns all stored instances sorted by start time.
func (s *Store) List() ([]domain.ServerInstance, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("list instances: %w", err)
	}
	var out []domain.ServerInstance
	for _, e := range entries {
		if filepath.Ext(e.Name()) != ".json" {
			continue
		}
		inst, err := s.Load(e.Name()[:len(e.Name())-5])
		if err != nil {
			continue // skip unreadable records
		}
		out = append(out, inst)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].StartedAt.Before(out[j].StartedAt)
	})
	return out, nil
}

// Remove deletes one instance record.
func (s *Store) Remove(id string) error {
	err := os.Remove(s.path(id))
	if err != nil && os.IsNotExist(err) {
		return nil
	}
	return err
}

// ProcessAlive reports whether the given pid refers to a live process.
// A pid of 0 is never alive.
func ProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	// Signal 0 probes for existence without actually signalling. EPERM
	// would mean the process exists but is owned by someone else; treat
	// it as alive to avoid orphaning foreign processes.
	err := syscall.Kill(pid, 0)
	return err == nil || err == syscall.EPERM
}

// Running reports whether the instance's server process is still alive.
func (s *Store) Running(inst domain.ServerInstance) bool {
	return ProcessAlive(inst.ServerPID)
}

// InstanceID builds the deterministic instance id for a run. The SAPI is only
// part of the id when it is not the default FPM one, keeping existing ids.
func InstanceID(server, phpVersion string, port int, sapi string) string {
	if sapi == "" || sapi == "fpm" {
		return fmt.Sprintf("%s-%d-php%s", server, port, phpVersion)
	}
	return fmt.Sprintf("%s-%s-%d-php%s", server, sapi, port, phpVersion)
}

// Timeout for graceful shutdowns before processes get killed.
const stopGrace = 10 * time.Second

// command and lookPath are indirections so tests can stub process calls.
var (
	command  = exec.Command
	lookPath = exec.LookPath
)
