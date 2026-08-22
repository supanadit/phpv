package apache

import (
	"strings"
	"testing"

	"github.com/supanadit/phpv/domain"
)

func TestRenderVhost_Basic(t *testing.T) {
	s := &ApacheService{}
	v := domain.Vhost{
		ServerName:   "test.local",
		DocumentRoot: "/home/user/projects/test",
		Port:         8080,
		Enabled:      true,
	}
	out := s.renderVhost(v)
	if !strings.Contains(out, "<VirtualHost *:8080>") {
		t.Errorf("missing VirtualHost line: %s", out)
	}
	if !strings.Contains(out, "ServerName test.local") {
		t.Errorf("missing ServerName: %s", out)
	}
	if !strings.Contains(out, "DocumentRoot /home/user/projects/test") {
		t.Errorf("missing DocumentRoot: %s", out)
	}
	if !strings.Contains(out, "AllowOverride All") {
		t.Errorf("missing AllowOverride: %s", out)
	}
	if !strings.Contains(out, "Require all granted") {
		t.Errorf("missing Require: %s", out)
	}
}

func TestRenderVhost_PerVhostFPM(t *testing.T) {
	s := &ApacheService{}
	v := domain.Vhost{
		ServerName:   "app.local",
		DocumentRoot: "/srv/app",
		PHPVersion:   "8.4",
		FPMPort:      9001,
		Enabled:      true,
	}
	out := s.renderVhost(v)
	if !strings.Contains(out, `SetHandler "proxy:fcgi://127.0.0.1:9001"`) {
		t.Errorf("missing per-vhost FPM handler: %s", out)
	}
}

func TestRenderVhost_Aliases(t *testing.T) {
	s := &ApacheService{}
	v := domain.Vhost{
		ServerName:   "test.local",
		DocumentRoot: "/srv/test",
		Aliases:      []string{"www.test.local", "api.test.local"},
		Port:         80,
	}
	out := s.renderVhost(v)
	if !strings.Contains(out, "ServerAlias www.test.local") {
		t.Errorf("missing ServerAlias: %s", out)
	}
	if !strings.Contains(out, "ServerAlias api.test.local") {
		t.Errorf("missing ServerAlias 2: %s", out)
	}
}

func TestDefaultMPM(t *testing.T) {
	if got := defaultMPM(domain.ConnectorModPHP, ""); got != "prefork" {
		t.Errorf("mod_php default MPM = %q, want prefork", got)
	}
	if got := defaultMPM(domain.ConnectorFPM, ""); got != "event" {
		t.Errorf("fpm default MPM = %q, want event", got)
	}
	if got := defaultMPM(domain.ConnectorFPM, "worker"); got != "worker" {
		t.Errorf("requested worker MPM = %q, want worker", got)
	}
}
