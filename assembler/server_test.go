package assembler

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/supanadit/phpv/compiler"
	"github.com/supanadit/phpv/internal/repository/disk"
	"github.com/supanadit/phpv/internal/repository/memory"
	"github.com/supanadit/phpv/registry"
	"github.com/supanadit/phpv/silo"
)

func newServerTestService(t *testing.T) *Service {
	t.Helper()
	regSvc := registry.NewService(memory.NewRegistryRepository())
	siloSvc := silo.NewService(disk.NewSiloRepository(), regSvc)
	return &Service{silo: siloSvc, reg: regSvc}
}

func writeFakeServerBinary(t *testing.T, prefix, sub, name string) {
	t.Helper()
	dir := filepath.Join(prefix, sub)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
}

// Regression: servers have no include/ tree, so the old isDepInstalled check
// always reported them missing and re-downloaded/rebuilt on every install.
func TestAssembleServer_SkipsInstalledStateWithoutIncludeDir(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PHPV_ROOT", root)

	prefix := filepath.Join(root, "packages", "nginx", "1.28.0")
	writeFakeServerBinary(t, prefix, "sbin", "nginx")
	if err := os.WriteFile(filepath.Join(prefix, ".state"), []byte("installed"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(prefix, "include")); err == nil {
		t.Fatal("test setup must not create include/, servers do not ship one")
	}

	var stages []string
	got, err := newServerTestService(t).AssembleServer(
		context.Background(), "nginx", "1.28.0", false, 1, compiler.Toolchain{},
		func(stage, _ string) { stages = append(stages, stage) },
	)
	if err != nil {
		t.Fatalf("AssembleServer: %v", err)
	}
	if got != prefix {
		t.Fatalf("prefix = %q, want %q", got, prefix)
	}
	want := []string{"resolve", "skip"}
	if !reflect.DeepEqual(stages, want) {
		t.Fatalf("progress stages = %v, want %v", stages, want)
	}
}

// Legacy installs predate state markers; the installed binary alone must be
// enough to skip a rebuild.
func TestAssembleServer_SkipsLegacyBinaryWithoutState(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PHPV_ROOT", root)

	prefix := filepath.Join(root, "packages", "nginx", "1.28.0")
	writeFakeServerBinary(t, prefix, "sbin", "nginx")

	var stages []string
	if _, err := newServerTestService(t).AssembleServer(
		context.Background(), "nginx", "1.28.0", false, 1, compiler.Toolchain{},
		func(stage, _ string) { stages = append(stages, stage) },
	); err != nil {
		t.Fatalf("AssembleServer: %v", err)
	}
	want := []string{"resolve", "skip"}
	if !reflect.DeepEqual(stages, want) {
		t.Fatalf("progress stages = %v, want %v", stages, want)
	}
}

func TestServerInstalled_NotInstalled(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PHPV_ROOT", root)

	svc := newServerTestService(t)
	prefix := filepath.Join(root, "packages", "nginx", "1.28.0")
	if err := os.MkdirAll(prefix, 0o755); err != nil {
		t.Fatal(err)
	}
	if svc.serverInstalled("nginx", "nginx", "1.28.0", prefix) {
		t.Fatal("serverInstalled = true for an empty prefix")
	}
}

// Progress must be reported even when assembly fails early, so callers can
// show what the command is doing.
func TestAssembleServer_EmitsResolveProgressOnError(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PHPV_ROOT", root)

	var stages []string
	_, err := newServerTestService(t).AssembleServer(
		context.Background(), "nginx", "0.0.0-nope", false, 1, compiler.Toolchain{},
		func(stage, _ string) { stages = append(stages, stage) },
	)
	if err == nil {
		t.Fatal("expected an error for an unknown version")
	}
	if len(stages) == 0 || stages[0] != "resolve" {
		t.Fatalf("progress stages = %v, want resolve first", stages)
	}
}
