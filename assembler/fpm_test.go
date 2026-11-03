package assembler

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/supanadit/phpv/compiler"
	"github.com/supanadit/phpv/internal/repository/memory"
	"github.com/supanadit/phpv/registry"
)

func newFPMTestService(t *testing.T, root string) *FPMService {
	t.Helper()
	regSvc := registry.NewService(memory.NewRegistryRepository())
	return NewFPMService(&Service{reg: regSvc}, root)
}

func writeFakeCLIProfile(t *testing.T, root, version string) BuildProfile {
	t.Helper()
	prefix := filepath.Join(root, "packages", "php", version)
	if err := os.MkdirAll(prefix, 0o755); err != nil {
		t.Fatal(err)
	}
	profile := BuildProfile{ConfigureArgs: []string{
		"--prefix=" + prefix,
		"--disable-all",
		"--enable-cli",
		"--with-zlib",
	}}
	if err := saveBuildProfile(prefix, profile); err != nil {
		t.Fatal(err)
	}
	return profile
}

// Regression: the php-fpm build used to print plain lines with no spinner.
// Progress must be reported through the callback so `server run` can animate.
func TestEnsureFPM_ReportsPrepareProgress(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PHPV_ROOT", root)
	writeFakeCLIProfile(t, root, "0.0.0")

	var stages []string
	_, err := newFPMTestService(t, root).EnsureFPM(
		context.Background(), "0.0.0", false, 1, compiler.Toolchain{}, "",
		func(stage, _ string) { stages = append(stages, stage) },
	)
	if err == nil {
		t.Fatal("expected an error for an unknown PHP version")
	}
	if len(stages) == 0 || stages[0] != "prepare" {
		t.Fatalf("progress stages = %v, want prepare first", stages)
	}
}

// An existing FPM binary with a matching profile marker needs no work and must
// not animate or emit stages.
func TestEnsureFPM_ExistingBinarySkipsProgress(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PHPV_ROOT", root)
	profile := writeFakeCLIProfile(t, root, "7.4.33")

	prefix := filepath.Join(root, "packages", "php-fpm", "7.4.33")
	bin := filepath.Join(prefix, "sbin", "php-fpm")
	if err := os.MkdirAll(filepath.Dir(bin), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeVariantMarker(prefix, VariantMarker{
		SAPI:        "fpm",
		PHPVersion:  "7.4.33",
		ProfileHash: profile.Hash("fpm"),
	}); err != nil {
		t.Fatal(err)
	}

	var stages []string
	got, err := newFPMTestService(t, root).EnsureFPM(
		context.Background(), "7.4.33", false, 1, compiler.Toolchain{}, "",
		func(stage, _ string) { stages = append(stages, stage) },
	)
	if err != nil {
		t.Fatalf("EnsureFPM: %v", err)
	}
	if got != bin {
		t.Fatalf("binary = %q, want %q", got, bin)
	}
	if len(stages) != 0 {
		t.Fatalf("progress stages = %v, want none", stages)
	}
}

// A stale marker (CLI rebuilt with different extensions) must force a rebuild.
func TestEnsureFPM_StaleMarkerRebuilds(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PHPV_ROOT", root)
	writeFakeCLIProfile(t, root, "0.0.0")

	prefix := filepath.Join(root, "packages", "php-fpm", "0.0.0")
	bin := filepath.Join(prefix, "sbin", "php-fpm")
	if err := os.MkdirAll(filepath.Dir(bin), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeVariantMarker(prefix, VariantMarker{
		SAPI:        "fpm",
		PHPVersion:  "0.0.0",
		ProfileHash: "stale",
	}); err != nil {
		t.Fatal(err)
	}

	var stages []string
	if _, err := newFPMTestService(t, root).EnsureFPM(
		context.Background(), "0.0.0", false, 1, compiler.Toolchain{}, "",
		func(stage, _ string) { stages = append(stages, stage) },
	); err == nil {
		t.Fatal("expected the stale build to attempt a rebuild and fail without php sources")
	}
	if len(stages) == 0 || stages[0] != "prepare" {
		t.Fatalf("progress stages = %v, want a rebuild starting at prepare", stages)
	}
}

func TestFPMNeedsRebuild(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PHPV_ROOT", root)
	profile := writeFakeCLIProfile(t, root, "7.4.33")
	svc := newFPMTestService(t, root)

	if !svc.NeedsRebuild("7.4.33", compiler.Toolchain{}) {
		t.Fatal("missing FPM binary must need a rebuild")
	}

	prefix := filepath.Join(root, "packages", "php-fpm", "7.4.33")
	bin := filepath.Join(prefix, "sbin", "php-fpm")
	if err := os.MkdirAll(filepath.Dir(bin), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeVariantMarker(prefix, VariantMarker{SAPI: "fpm", PHPVersion: "7.4.33", ProfileHash: profile.Hash("fpm")}); err != nil {
		t.Fatal(err)
	}
	if svc.NeedsRebuild("7.4.33", compiler.Toolchain{}) {
		t.Fatal("matching marker must not need a rebuild")
	}
	if err := writeVariantMarker(prefix, VariantMarker{SAPI: "fpm", PHPVersion: "7.4.33", ProfileHash: "stale"}); err != nil {
		t.Fatal(err)
	}
	if !svc.NeedsRebuild("7.4.33", compiler.Toolchain{}) {
		t.Fatal("stale marker must need a rebuild")
	}
}
