package assembler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseConfigureOptions(t *testing.T) {
	args, env := parseConfigureOptions("--prefix=/x --disable-all --with-curl PKG_CONFIG_PATH=/a:/b")
	if len(args) != 3 {
		t.Fatalf("args = %v, want 3", args)
	}
	if env["PKG_CONFIG_PATH"] != "/a:/b" {
		t.Fatalf("env = %v, want PKG_CONFIG_PATH", env)
	}
}

func TestVariantArgsFPMKeepsExtensions(t *testing.T) {
	profile := BuildProfile{ConfigureArgs: []string{
		"--prefix=/cli", "--disable-all", "--enable-cli", "--with-curl",
		"--with-pdo-mysql=mysqlnd", "--enable-mbstring",
	}}
	args, err := profile.VariantArgs("fpm", "/fpm", "")
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(args, " ")
	for _, want := range []string{
		"--prefix=/fpm", "--enable-fpm", "--with-curl",
		"--with-pdo-mysql=mysqlnd", "--enable-mbstring",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("fpm args missing %q: %v", want, args)
		}
	}
	for _, unwanted := range []string{"--prefix=/cli", "--enable-cli"} {
		if strings.Contains(joined, unwanted) {
			t.Errorf("fpm args still contain %q: %v", unwanted, args)
		}
	}
}

func TestVariantArgsModPHPRequiresApxs(t *testing.T) {
	profile := BuildProfile{ConfigureArgs: []string{"--disable-all"}}
	if _, err := profile.VariantArgs("modphp", "/apache-php", "/no-httpd"); err == nil {
		t.Fatal("expected an error when apxs is missing")
	}

	httpd := t.TempDir()
	if err := os.MkdirAll(filepath.Join(httpd, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	apxs := filepath.Join(httpd, "bin", "apxs")
	if err := os.WriteFile(apxs, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	args, err := profile.VariantArgs("modphp", "/apache-php", httpd)
	if err != nil {
		t.Fatalf("VariantArgs: %v", err)
	}
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "--with-apxs2="+apxs) || !strings.Contains(joined, "--disable-fpm") {
		t.Fatalf("modphp args = %v", args)
	}
}

func TestProfileHashChangesWithSAPI(t *testing.T) {
	profile := BuildProfile{ConfigureArgs: []string{"--with-curl"}}
	if profile.Hash("fpm") == profile.Hash("modphp") {
		t.Fatal("profile hash must differ by SAPI")
	}
	if profile.Hash("fpm") == (BuildProfile{ConfigureArgs: []string{"--with-curl", "--enable-gd"}}).Hash("fpm") {
		t.Fatal("profile hash must change when extensions change")
	}
}

func TestCLIExtensionDir(t *testing.T) {
	prefix := t.TempDir()
	if err := os.MkdirAll(filepath.Join(prefix, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	phpConfig := filepath.Join(prefix, "bin", "php-config")
	if err := os.WriteFile(phpConfig, []byte("#!/bin/sh\necho /opt/php/ext\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := cliExtensionDir(prefix); got != "/opt/php/ext" {
		t.Fatalf("cliExtensionDir = %q, want /opt/php/ext", got)
	}
}

func TestBuildProfileRoundTrip(t *testing.T) {
	prefix := t.TempDir()
	want := BuildProfile{
		ConfigureArgs: []string{"--disable-all", "--enable-pdo"},
		Env:           map[string]string{"PKG_CONFIG_PATH": "/a:/b", "CFLAGS": "-O2"},
	}
	if err := saveBuildProfile(prefix, want); err != nil {
		t.Fatalf("saveBuildProfile: %v", err)
	}
	got, err := readBuildProfile(prefix)
	if err != nil {
		t.Fatalf("readBuildProfile: %v", err)
	}
	if strings.Join(got.ConfigureArgs, " ") != strings.Join(want.ConfigureArgs, " ") {
		t.Fatalf("args = %v, want %v", got.ConfigureArgs, want.ConfigureArgs)
	}
	if got.Env["PKG_CONFIG_PATH"] != "/a:/b" || got.Env["CFLAGS"] != "-O2" {
		t.Fatalf("env = %v", got.Env)
	}
}
