package webserver

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/supanadit/phpv/domain"
)

func TestStoreSaveLoadListRemove(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	inst := domain.ServerInstance{
		ID:            "caddy-8080-php8.3.36",
		Server:        "caddy",
		ServerVersion: "2.10.2",
		PHPVersion:    "8.3.36",
		Dir:           "/tmp/app",
		Port:          8080,
		Socket:        "/tmp/app.sock",
		StateDir:      "/tmp/state",
		StartedAt:     mustTime(t, "2026-11-02T10:00:00Z"),
	}
	if err := store.Save(inst); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := store.Load(inst.ID)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Server != "caddy" || got.Port != 8080 || got.PHPVersion != "8.3.36" {
		t.Fatalf("Load returned %+v", got)
	}

	list, err := store.List()
	if err != nil || len(list) != 1 {
		t.Fatalf("List = %v, %v", list, err)
	}

	if err := store.Remove(inst.ID); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	list, _ = store.List()
	if len(list) != 0 {
		t.Fatalf("List after Remove = %d entries, want 0", len(list))
	}
}

func TestRenderFPMConf(t *testing.T) {
	dir := t.TempDir()
	conf, err := RenderFPMConf(FPMConfig{
		Socket:   "/tmp/sock",
		PIDFile:  filepath.Join(dir, "fpm.pid"),
		ErrorLog: filepath.Join(dir, "err.log"),
		SlowLog:  filepath.Join(dir, "slow.log"),
		StateDir: dir,
	})
	if err != nil {
		t.Fatalf("RenderFPMConf: %v", err)
	}
	data, err := os.ReadFile(conf)
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	for _, want := range []string{
		"listen = /tmp/sock",
		"php_admin_value[error_log] = " + filepath.Join(dir, "php-error.log"),
		"php_admin_flag[log_errors] = on",
		"daemonize = no",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("php-fpm.conf missing %q\ngot:\n%s", want, s)
		}
	}
}

func TestExecFPMIncludesCLIIniAndExtensionDir(t *testing.T) {
	cmd := execFPM("/opt/php-fpm", "/tmp/fpm.conf", "/opt/php/lib", "/opt/php/lib/php/extensions/api")
	args := strings.Join(cmd.Args, " ")
	for _, want := range []string{
		"-F",
		"-y /tmp/fpm.conf",
		"-c /opt/php/lib",
		"-d extension_dir=/opt/php/lib/php/extensions/api",
	} {
		if !strings.Contains(args, want) {
			t.Errorf("php-fpm args missing %q: %v", want, cmd.Args)
		}
	}
}

func TestCLIRuntimeReadsPHPConfig(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	script := "#!/bin/sh\ncase \"$1\" in\n--ini-path) echo \"$0-ini\";;\n--extension-dir) echo \"$0-ext\";;\nesac\n"
	phpConfig := filepath.Join(dir, "bin", "php-config")
	if err := os.WriteFile(phpConfig, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	iniDir, extDir := CLIRuntime(dir)
	if iniDir != phpConfig+"-ini" {
		t.Errorf("iniDir = %q, want php-config --ini-path output", iniDir)
	}
	if extDir != phpConfig+"-ext" {
		t.Errorf("extDir = %q, want php-config --extension-dir output", extDir)
	}
}

func TestCLIRuntimeFallsBackToExtensionAPIDir(t *testing.T) {
	dir := t.TempDir()
	apiDir := filepath.Join(dir, "lib", "php", "extensions", "no-debug-non-zts-20250925")
	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		t.Fatal(err)
	}
	_, extDir := CLIRuntime(dir)
	if extDir != apiDir {
		t.Errorf("extDir = %q, want %q", extDir, apiDir)
	}
}

func TestRenderFPMConfLegacySkipsSlowLog(t *testing.T) {
	dir := t.TempDir()
	conf, err := RenderFPMConf(FPMConfig{
		Socket:   "/tmp/sock",
		StateDir: dir,
		Legacy:   true,
	})
	if err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(conf)
	if strings.Contains(string(data), "slowlog") || strings.Contains(string(data), "request_terminate_timeout") {
		t.Error("legacy php-fpm.conf contains PHP 7+ pool directives")
	}
}

func TestCaddyfileRendering(t *testing.T) {
	dir := t.TempDir()
	inst := domain.ServerInstance{
		ID:       "caddy-8080-php8.3.36",
		Server:   "caddy",
		Dir:      "/srv/app/public",
		Port:     8080,
		Socket:   "/tmp/x.sock",
		StateDir: dir,
	}
	var d CaddyDriver
	conf, err := d.WriteConfig(inst, Runtime{})
	if err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(conf)
	s := string(data)
	for _, want := range []string{
		"root * /srv/app/public",
		"php_fastcgi unix//tmp/x.sock",
		":8080 {",
		"file_server",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("Caddyfile missing %q\ngot:\n%s", want, s)
		}
	}

	// The site label must render the concrete port, not the {PORT} token.
	if strings.Contains(s, ":{PORT}") {
		t.Error("Caddyfile still contains unresolved {PORT} placeholder")
	}
}

func TestNginxConfRendering(t *testing.T) {
	dir := t.TempDir()
	inst := domain.ServerInstance{
		ID:       "nginx-8081-php5.6.40",
		Server:   "nginx",
		Dir:      "/srv/app",
		Host:     "legacy.local",
		Port:     8081,
		Socket:   "/tmp/n.sock",
		StateDir: dir,
	}
	var d NginxDriver
	conf, err := d.WriteConfig(inst, Runtime{})
	if err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(conf)
	s := string(data)
	for _, want := range []string{
		"listen 8081;",
		"server_name legacy.local;",
		"fastcgi_pass unix:/tmp/n.sock;",
		"fastcgi_param SCRIPT_FILENAME $document_root$fastcgi_script_name;",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("nginx.conf missing %q\ngot:\n%s", want, s)
		}
	}
}

func TestApacheConfRendering(t *testing.T) {
	dir := t.TempDir()
	inst := domain.ServerInstance{
		ID:       "apache-8082-php8.3.36",
		Server:   "apache",
		Dir:      "/srv/app",
		Port:     8082,
		Socket:   "/tmp/a.sock",
		StateDir: dir,
	}
	var d ApacheDriver
	conf, err := d.WriteConfig(inst, Runtime{})
	if err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(conf)
	s := string(data)
	for _, want := range []string{
		"Listen 8082",
		"LoadModule proxy_fcgi_module modules/mod_proxy_fcgi.so",
		"SetHandler \"proxy:unix:/tmp/a.sock|fcgi://localhost\"",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("httpd.conf missing %q\ngot:\n%s", want, s)
		}
	}
}

func TestHostsParsing(t *testing.T) {
	dir := t.TempDir()
	hosts := filepath.Join(dir, "hosts")
	hostsPath = hosts
	t.Cleanup(func() { hostsPath = "/etc/hosts" })

	if err := os.WriteFile(hosts, []byte("127.0.0.1 localhost\n::1 localhost\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if HostsHasEntry("myapp.local") {
		t.Fatal("myapp.local should not be present yet")
	}
	// Direct write (permissions allow in test).
	if err := appendHostsAsUser(HostsLine("myapp.local")); err != nil {
		t.Fatalf("appendHostsAsUser: %v", err)
	}
	if !HostsHasEntry("myapp.local") {
		t.Fatal("myapp.local should be present after append")
	}
	// RemoveHostEntry drops only the managed line.
	if err := RemoveHostEntry("myapp.local"); err != nil {
		t.Fatalf("RemoveHostEntry: %v", err)
	}
	if HostsHasEntry("myapp.local") {
		t.Fatal("myapp.local should be gone after RemoveHostEntry")
	}
	data, _ := os.ReadFile(hosts)
	if !strings.Contains(string(data), "127.0.0.1 localhost") {
		t.Error("unmanaged localhost line was removed")
	}
}

func TestPickFreePortPreferredBusy(t *testing.T) {
	ln, err := listenLocal()
	if err != nil {
		t.Skip("cannot bind test port")
	}
	defer ln.Close()
	port := ln.Addr().String()
	portNum := 0
	for _, r := range port {
		if r >= '0' && r <= '9' {
			portNum = portNum*10 + int(r-'0')
		}
	}
	if _, err := PickFreePort(portNum); err == nil {
		t.Errorf("PickFreePort(%d) should fail when the port is bound", portNum)
	}
}

func TestInstanceID(t *testing.T) {
	got := InstanceID("nginx", "5.6.40", 8080, "")
	if got != "nginx-8080-php5.6.40" {
		t.Errorf("InstanceID = %q", got)
	}
}

func TestApacheModPHPConfig(t *testing.T) {
	dir := t.TempDir()
	inst := domain.ServerInstance{
		ID:         "apache-modphp-8080-php8.5.12",
		Server:     "apache",
		SAPI:       "modphp",
		PHPVersion: "8.5.12",
		Dir:        "/srv/app",
		Port:       8080,
		StateDir:   dir,
	}
	conf, err := (ApacheDriver{}).WriteConfig(inst, Runtime{
		SAPI:         "modphp",
		ApacheModule: "/opt/php-apache/8.5.12/modules/libphp.so",
		PHPIniDir:    "/opt/php/8.5.12/lib",
		ExtensionDir: "/opt/php/8.5.12/lib/php/extensions/api",
		MPM:          "prefork",
	})
	if err != nil {
		t.Fatalf("WriteConfig: %v", err)
	}
	data, _ := os.ReadFile(conf)
	s := string(data)
	for _, want := range []string{
		"LoadModule mpm_prefork_module modules/mod_mpm_prefork.so",
		"LoadModule php_module /opt/php-apache/8.5.12/modules/libphp.so",
		`PHPIniDir "/opt/php/8.5.12/lib"`,
		`php_admin_value extension_dir "/opt/php/8.5.12/lib/php/extensions/api"`,
		"AddHandler application/x-httpd-php .php",
		"SetHandler application/x-httpd-php",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("httpd.conf missing %q\ngot:\n%s", want, s)
		}
	}
	if strings.Contains(s, "proxy_fcgi") || strings.Contains(s, "proxy:unix:") {
		t.Errorf("mod_php config must not use FastCGI:\n%s", s)
	}
}

func TestApacheFPMConfigUsesEventAndFastCGI(t *testing.T) {
	dir := t.TempDir()
	inst := domain.ServerInstance{
		ID:       "apache-8080-php8.5.12",
		Server:   "apache",
		SAPI:     "fpm",
		Dir:      "/srv/app",
		Port:     8080,
		Socket:   "/tmp/fpm.sock",
		StateDir: dir,
	}
	conf, err := (ApacheDriver{}).WriteConfig(inst, Runtime{SAPI: "fpm", MPM: "event", FPMConf: "/tmp/fpm.conf"})
	if err != nil {
		t.Fatalf("WriteConfig: %v", err)
	}
	data, _ := os.ReadFile(conf)
	s := string(data)
	for _, want := range []string{
		"LoadModule mpm_event_module modules/mod_mpm_event.so",
		"LoadModule proxy_fcgi_module modules/mod_proxy_fcgi.so",
		`SetHandler "proxy:unix:/tmp/fpm.sock|fcgi://localhost"`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("httpd.conf missing %q\ngot:\n%s", want, s)
		}
	}
	if strings.Contains(s, "php_module") {
		t.Error("FPM config must not load the embedded PHP module")
	}
}

func TestInstanceIDModPHP(t *testing.T) {
	if got := InstanceID("apache", "8.5.12", 8080, "modphp"); got != "apache-modphp-8080-php8.5.12" {
		t.Errorf("InstanceID = %q", got)
	}
	if got := InstanceID("apache", "8.5.12", 8080, "fpm"); got != "apache-8080-php8.5.12" {
		t.Errorf("InstanceID = %q", got)
	}
}

func TestApachePHPModuleName(t *testing.T) {
	cases := map[string]string{
		"8.5.12": "php_module",
		"7.4.33": "php7_module",
		"5.6.40": "php5_module",
	}
	for version, want := range cases {
		if got := apachePHPModuleName(version); got != want {
			t.Errorf("apachePHPModuleName(%q) = %q, want %q", version, got, want)
		}
	}
}
