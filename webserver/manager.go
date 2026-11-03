package webserver

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

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
	drivers map[string]Driver
}

// NewManager wires a Manager for the given deps.
func NewManager(deps Deps) *Manager {
	serversDir := StateDir(deps.Root)
	m := &Manager{
		deps:    deps,
		store:   NewStore(serversDir),
		drivers: make(map[string]Driver),
	}
	for _, d := range []Driver{CaddyDriver{}, NginxDriver{}, ApacheDriver{}} {
		m.drivers[d.Name()] = d
	}
	return m
}

// Store exposes the instance store.
func (m *Manager) Store() *Store { return m.store }

// Driver returns the driver for a server name.
func (m *Manager) Driver(name string) (Driver, error) {
	d, ok := m.drivers[name]
	if !ok {
		return nil, fmt.Errorf("unknown server %q (available: %s)", name, strings.Join(domain.ServerNames(), ", "))
	}
	return d, nil
}

// RunOptions describes one `phpv server run` invocation.
type RunOptions struct {
	Server        string // caddy|nginx|apache (resolved upstream when empty)
	ServerVersion string // constraint; empty = config/default/only installed
	PHPVersion    string // constraint; empty = active/default php
	Dir           string // document root (absolute)
	Host          string // optional vhost hostname
	Port          int    // 0 = auto-pick from 8080
	Detach        bool
	Verbose       bool
	Jobs          int
}

// Start brings up an instance: FPM, then the server, then records state.
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

	// Resolve the PHP version and ensure its FPM variant exists.
	phpVersion, err := m.deps.ResolveVersion(opts.PHPVersion)
	if err != nil {
		return domain.ServerInstance{}, fmt.Errorf("resolve php version: %w", err)
	}
	fpmBin, err := m.deps.EnsureFPM(ctx, phpVersion, opts.Verbose, opts.Jobs)
	if err != nil {
		return domain.ServerInstance{}, err
	}

	// Resolve the server and its binary.
	driver, err := m.Driver(opts.Server)
	if err != nil {
		return domain.ServerInstance{}, err
	}
	serverVersion, err := m.deps.InstalledServerVersion(opts.Server, opts.ServerVersion)
	if err != nil {
		return domain.ServerInstance{}, fmt.Errorf("resolve %s version: %w", opts.Server, err)
	}
	prefix := m.deps.ServerPrefix(opts.Server, serverVersion)
	serverBin := driver.Binary(prefix)
	if serverBin == "" {
		return domain.ServerInstance{}, fmt.Errorf("%s binary not found in %s (install it: phpv server install %s)", opts.Server, prefix, opts.Server)
	}

	port, err := PickFreePort(opts.Port)
	if err != nil {
		return domain.ServerInstance{}, err
	}

	id := InstanceID(opts.Server, phpVersion, port)
	inst := domain.ServerInstance{
		ID:            id,
		Server:        opts.Server,
		ServerVersion: serverVersion,
		PHPVersion:    phpVersion,
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
	// exist before php-fpm binds.
	for _, d := range []string{filepath.Dir(inst.Socket), inst.StateDir, filepath.Join(inst.StateDir, "logs"), filepath.Join(inst.StateDir, "tmp")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return domain.ServerInstance{}, err
		}
	}
	_ = os.Remove(inst.Socket)

	// Render the FPM config from the CLI install's extension layout.
	fpmConf, err := m.renderFPM(inst, fpmBin)
	if err != nil {
		return domain.ServerInstance{}, err
	}

	// Start FPM. Its stderr/stdout always land in fpm.log (foreground or
	// detached) so a failed start is diagnosable.
	fpmLogPath := filepath.Join(inst.StateDir, "fpm.log")
	fpmCmd := execFPM(fpmBin, fpmConf)
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

	// Start the webserver.
	confPath, err := driver.WriteConfig(inst, fpmConf)
	if err != nil {
		m.Stop(inst)
		return domain.ServerInstance{}, err
	}
	args := driver.StartArgs(inst, confPath)
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

// renderFPM builds the FPMConfig for an instance from the CLI install.
func (m *Manager) renderFPM(inst domain.ServerInstance, fpmBin string) (string, error) {
	cliPrefix := m.deps.CLIPrefix(inst.PHPVersion)
	iniPath := filepath.Join(cliPrefix, "lib", "php.ini")
	extDir, extensions := scanExtensions(cliPrefix)
	legacy := domain.CompareVersions(inst.PHPVersion, "7.0.0") < 0
	conf, err := RenderFPMConf(FPMConfig{
		Binary:      fpmBin,
		PHPVersion:  inst.PHPVersion,
		Socket:      inst.Socket,
		PIDFile:     filepath.Join(inst.StateDir, "fpm.pid"),
		ErrorLog:    filepath.Join(inst.StateDir, "logs", "fpm-error.log"),
		SlowLog:     filepath.Join(inst.StateDir, "logs", "fpm-slow.log"),
		StateDir:    inst.StateDir,
		PHPIniPath:  iniPath,
		ExtensionDir: extDir,
		Extensions:  extensions,
		Legacy:      legacy,
	})
	return conf, err
}

// scanExtensions returns the extension_dir and extension basenames from the
// CLI install's php.ini, so FPM loads the same shared modules.
func scanExtensions(cliPrefix string) (string, []string) {
	iniPath := filepath.Join(cliPrefix, "lib", "php.ini")
	data, err := os.ReadFile(iniPath)
	if err != nil {
		return "", nil
	}
	var extDir string
	var exts []string
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(strings.ToLower(key))
		val = strings.TrimSpace(val)
		switch key {
		case "extension_dir":
			extDir = val
		case "extension":
			if val != "" {
				exts = append(exts, filepath.Base(val))
			}
		}
	}
	return extDir, exts
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
