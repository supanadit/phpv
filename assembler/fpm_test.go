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

// Regression: the php-fpm build used to print plain lines with no spinner.
// Progress must be reported through the callback so `server run` can animate.
func TestEnsureFPM_ReportsPrepareProgress(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PHPV_ROOT", root)

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

// An existing FPM binary needs no work and must not animate or emit stages.
func TestEnsureFPM_ExistingBinarySkipsProgress(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PHPV_ROOT", root)

	bin := filepath.Join(root, "packages", "php-fpm", "7.4.33", "sbin", "php-fpm")
	if err := os.MkdirAll(filepath.Dir(bin), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o755); err != nil {
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
