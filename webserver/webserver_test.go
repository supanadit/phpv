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

func TestRenderFPMConfLoadsCLIExtensions(t *testing.T) {
	dir := t.TempDir()
	ini := filepath.Join(dir, "php.ini")
	content := "; comment\nmemory_limit = 256M\ndisplay_errors = On\nextension_dir = /opt/php/lib/php/extensions/no-debug-non-zts-x\nextension = curl.so\nextension = xdebug.so\nzend_extension = opcache.so\n"
	if err := os.WriteFile(ini, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	conf, err := RenderFPMConf(FPMConfig{
		Socket:       "/tmp/sock",
		PIDFile:      filepath.Join(dir, "fpm.pid"),
		ErrorLog:     filepath.Join(dir, "err.log"),
		SlowLog:      filepath.Join(dir, "slow.log"),
		StateDir:     dir,
		PHPIniPath:   ini,
		ExtensionDir: "/opt/php/lib/php/extensions/no-debug-non-zts-x",
		Extensions:   []string{"curl.so", "xdebug.so"},
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
		"php_admin_value[extension_dir] = /opt/php/lib/php/extensions/no-debug-non-zts-x",
		"php_admin_value[extension] = curl.so",
		"php_admin_value[extension] = xdebug.so",
		"php_admin_value[memory_limit] = 256M",
		"php_admin_flag[display_errors] = on",
		"daemonize = no",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("php-fpm.conf missing %q\ngot:\n%s", want, s)
		}
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
	conf, err := d.WriteConfig(inst, "")
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
	conf, err := d.WriteConfig(inst, "")
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
	conf, err := d.WriteConfig(inst, "")
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
	got := InstanceID("nginx", "5.6.40", 8080)
	if got != "nginx-8080-php5.6.40" {
		t.Errorf("InstanceID = %q", got)
	}
}
