package disk

import (
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/internal/utils"
)

func TestBundlerRepository_FreshClean(t *testing.T) {
	baseDir := t.TempDir()
	silo := &domain.Silo{Root: baseDir}
	fs := afero.NewOsFs()

	exactVersion := "8.0.30"

	versionPath := utils.PHPVersionPath(silo, exactVersion)
	sourcePath := utils.GetSourcePath(silo, "php", exactVersion)

	if err := fs.MkdirAll(versionPath, 0755); err != nil {
		t.Fatalf("failed to create version path: %v", err)
	}
	if err := fs.MkdirAll(sourcePath, 0755); err != nil {
		t.Fatalf("failed to create source path: %v", err)
	}

	if err := afero.WriteFile(fs, filepath.Join(versionPath, "some-file"), []byte("test"), 0644); err != nil {
		t.Fatalf("failed to write version file: %v", err)
	}
	if err := afero.WriteFile(fs, filepath.Join(sourcePath, "source.tar.gz"), []byte("test"), 0644); err != nil {
		t.Fatalf("failed to write source file: %v", err)
	}

	cachePath := utils.GetArchivePath(silo, "php", exactVersion)
	if err := fs.MkdirAll(filepath.Dir(cachePath), 0755); err != nil {
		t.Fatalf("failed to create cache dir: %v", err)
	}
	if err := afero.WriteFile(fs, cachePath, []byte("cached-archive"), 0644); err != nil {
		t.Fatalf("failed to write cache file: %v", err)
	}

	repo := &bundlerRepository{
		silo: silo,
		fs:   fs,
	}

	if err := repo.freshClean("php", exactVersion); err != nil {
		t.Fatalf("freshClean failed: %v", err)
	}

	if exists, _ := afero.Exists(fs, versionPath); exists {
		t.Errorf("version path %s should have been removed", versionPath)
	}

	if exists, _ := afero.Exists(fs, sourcePath); exists {
		t.Errorf("source path %s should have been removed", sourcePath)
	}

	if exists, _ := afero.Exists(fs, cachePath); !exists {
		t.Errorf("cache path %s should have been preserved", cachePath)
	}
}

func TestBundlerRepository_FreshClean_NonExistent(t *testing.T) {
	baseDir := t.TempDir()
	silo := &domain.Silo{Root: baseDir}
	fs := afero.NewOsFs()

	repo := &bundlerRepository{
		silo: silo,
		fs:   fs,
	}

	if err := repo.freshClean("php", "9.0.0"); err != nil {
		t.Errorf("freshClean should not fail for non-existent paths: %v", err)
	}
}

func TestBundlerRepository_FreshClean_OnlyRemovesPHPVersion(t *testing.T) {
	baseDir := t.TempDir()
	silo := &domain.Silo{Root: baseDir}
	fs := afero.NewOsFs()

	exactVersion := "8.0.30"
	otherVersion := "8.1.33"

	versionPath8 := utils.PHPVersionPath(silo, exactVersion)
	versionPath81 := utils.PHPVersionPath(silo, otherVersion)

	if err := fs.MkdirAll(versionPath8, 0755); err != nil {
		t.Fatalf("failed to create version path 8.0: %v", err)
	}
	if err := fs.MkdirAll(versionPath81, 0755); err != nil {
		t.Fatalf("failed to create version path 8.1: %v", err)
	}

	repo := &bundlerRepository{
		silo: silo,
		fs:   fs,
	}

	if err := repo.freshClean("php", exactVersion); err != nil {
		t.Fatalf("freshClean failed: %v", err)
	}

	if exists, _ := afero.Exists(fs, versionPath8); exists {
		t.Errorf("version path %s should have been removed", versionPath8)
	}

	if exists, _ := afero.Exists(fs, versionPath81); !exists {
		t.Errorf("other version path %s should have been preserved", versionPath81)
	}
}

func TestBundlerRepository_FreshClean_PreservesCache(t *testing.T) {
	baseDir := t.TempDir()
	silo := &domain.Silo{Root: baseDir}
	fs := afero.NewOsFs()

	exactVersion := "8.0.30"

	cachePath := utils.GetArchivePath(silo, "php", exactVersion)
	cacheDir := filepath.Dir(cachePath)
	if err := fs.MkdirAll(cacheDir, 0755); err != nil {
		t.Fatalf("failed to create cache dir: %v", err)
	}
	if err := afero.WriteFile(fs, cachePath, []byte("cached"), 0644); err != nil {
		t.Fatalf("failed to write cache file: %v", err)
	}

	repo := &bundlerRepository{
		silo: silo,
		fs:   fs,
	}

	if err := repo.freshClean("php", exactVersion); err != nil {
		t.Fatalf("freshClean failed: %v", err)
	}

	if exists, _ := afero.Exists(fs, cachePath); !exists {
		t.Errorf("cache should be preserved at %s", cachePath)
	}

	cacheContents, err := afero.ReadFile(fs, cachePath)
	if err != nil {
		t.Fatalf("failed to read cache: %v", err)
	}
	if string(cacheContents) != "cached" {
		t.Errorf("cache contents should be preserved, got %s", string(cacheContents))
	}
}

func TestBundlerRepository_FreshClean_PreservesOtherSources(t *testing.T) {
	baseDir := t.TempDir()
	silo := &domain.Silo{Root: baseDir}
	fs := afero.NewOsFs()

	exactVersion := "8.0.30"

	phpSourcePath := utils.GetSourcePath(silo, "php", exactVersion)
	libxml2SourcePath := utils.GetSourcePath(silo, "libxml2", "2.9.14")

	if err := fs.MkdirAll(phpSourcePath, 0755); err != nil {
		t.Fatalf("failed to create php source path: %v", err)
	}
	if err := fs.MkdirAll(libxml2SourcePath, 0755); err != nil {
		t.Fatalf("failed to create libxml2 source path: %v", err)
	}

	repo := &bundlerRepository{
		silo: silo,
		fs:   fs,
	}

	if err := repo.freshClean("php", exactVersion); err != nil {
		t.Fatalf("freshClean failed: %v", err)
	}

	if exists, _ := afero.Exists(fs, phpSourcePath); exists {
		t.Errorf("php source path %s should have been removed", phpSourcePath)
	}

	if exists, _ := afero.Exists(fs, libxml2SourcePath); !exists {
		t.Errorf("libxml2 source path %s should have been preserved", libxml2SourcePath)
	}
}

func TestBundlerRepository_FreshClean_PreservesBuildTools(t *testing.T) {
	baseDir := t.TempDir()
	silo := &domain.Silo{Root: baseDir}
	fs := afero.NewOsFs()

	exactVersion := "8.0.30"

	versionPath := utils.PHPVersionPath(silo, exactVersion)
	buildToolsPath := utils.BuildToolsPath(silo)
	buildToolPath := utils.BuildToolPath(silo, "m4", "1.4.19")

	if err := fs.MkdirAll(versionPath, 0755); err != nil {
		t.Fatalf("failed to create version path: %v", err)
	}
	if err := fs.MkdirAll(buildToolPath, 0755); err != nil {
		t.Fatalf("failed to create build tool path: %v", err)
	}

	repo := &bundlerRepository{
		silo: silo,
		fs:   fs,
	}

	if err := repo.freshClean("php", exactVersion); err != nil {
		t.Fatalf("freshClean failed: %v", err)
	}

	if exists, _ := afero.Exists(fs, versionPath); exists {
		t.Errorf("version path should be removed")
	}

	if exists, _ := afero.Exists(fs, buildToolsPath); !exists {
		t.Errorf("build-tools path %s should be preserved", buildToolsPath)
	}

	if exists, _ := afero.Exists(fs, buildToolPath); !exists {
		t.Errorf("build tool path %s should be preserved", buildToolPath)
	}
}
