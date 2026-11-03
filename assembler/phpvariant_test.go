package assembler

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/supanadit/phpv/internal/repository/memory"
	"github.com/supanadit/phpv/internal/repository/patches"
	"github.com/supanadit/phpv/internal/source"
	"github.com/supanadit/phpv/patcher"
	"github.com/supanadit/phpv/registry"
	"github.com/supanadit/phpv/silo"
)

// extractSiloRepo simulates a successful extraction.
type extractSiloRepo struct{ mockSiloRepo }

func (r *extractSiloRepo) Extract(archivePath, destDir string) (bool, error) {
	src := filepath.Join(destDir, "php-8.5.12")
	if err := os.MkdirAll(src, 0o755); err != nil {
		return false, err
	}
	if err := os.WriteFile(filepath.Join(src, "configure"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		return false, err
	}
	return true, nil
}

// Regression: an interrupted extraction leaves a non-empty src dir; the
// shared Extract guard would skip re-extracting it forever.
func TestPreparePHPSourceCleansPartialExtraction(t *testing.T) {
	root := t.TempDir()
	work := filepath.Join(root, "build", "php-apache", "8.5.12")

	partial := filepath.Join(work, "src", "php-8.5.12")
	if err := os.MkdirAll(partial, 0o755); err != nil {
		t.Fatal(err)
	}
	// No configure -> source.Find cannot use this tree.

	regSvc := registry.NewService(memory.NewRegistryRepository())
	siloSvc := silo.NewService(&extractSiloRepo{mockSiloRepo{root: root}}, regSvc)
	patcherSvc := patcher.NewService(patches.NewRepository(t.TempDir(), t.TempDir()))
	deps := &Service{reg: regSvc, silo: siloSvc, patcher: patcherSvc}

	if err := preparePHPSource(deps, "8.5.12", work); err != nil {
		t.Fatalf("preparePHPSource: %v", err)
	}
	if source.Find(filepath.Join(work, "src"), "php") == "" {
		t.Fatal("source tree was not re-extracted")
	}
}
