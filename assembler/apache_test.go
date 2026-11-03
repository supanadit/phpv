package assembler

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/supanadit/phpv/compiler"
	"github.com/supanadit/phpv/internal/repository/memory"
	"github.com/supanadit/phpv/registry"
)

func newApacheTestService(t *testing.T, root string) *ApacheService {
	t.Helper()
	regSvc := registry.NewService(memory.NewRegistryRepository())
	return NewApacheService(&Service{reg: regSvc}, root)
}

func TestEnsureApacheModule_ExistingSkipsProgress(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PHPV_ROOT", root)
	profile := writeFakeCLIProfile(t, root, "7.4.33")

	prefix := filepath.Join(root, "packages", "php-apache", "7.4.33")
	module := filepath.Join(prefix, "modules", "libphp.so")
	if err := os.MkdirAll(filepath.Dir(module), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(module, []byte("ELF"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeVariantMarker(prefix, VariantMarker{
		SAPI:        "modphp",
		PHPVersion:  "7.4.33",
		ProfileHash: profile.Hash("modphp") + ":" + filepath.Base("/no-httpd") + ":",
	}); err != nil {
		t.Fatal(err)
	}

	var stages []string
	got, err := newApacheTestService(t, root).EnsureApacheModule(
		context.Background(), "7.4.33", "/no-httpd", false, 1, compiler.Toolchain{},
		func(stage, _ string) { stages = append(stages, stage) },
	)
	if err != nil {
		t.Fatalf("EnsureApacheModule: %v", err)
	}
	if got != module {
		t.Fatalf("module = %q, want %q", got, module)
	}
	if len(stages) != 0 {
		t.Fatalf("progress stages = %v, want none", stages)
	}
}

func TestEnsureApacheModule_RequiresApxs(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PHPV_ROOT", root)
	writeFakeCLIProfile(t, root, "7.4.33")

	_, err := newApacheTestService(t, root).EnsureApacheModule(
		context.Background(), "7.4.33", "/no-httpd", false, 1, compiler.Toolchain{}, nil,
	)
	if err == nil || !strings.Contains(err.Error(), "apxs") {
		t.Fatalf("error = %v, want an apxs hint", err)
	}
}

func TestFindApacheModule(t *testing.T) {
	src := t.TempDir()
	if err := os.MkdirAll(filepath.Join(src, "libs"), 0o755); err != nil {
		t.Fatal(err)
	}
	module := filepath.Join(src, "libs", "libphp.so")
	if err := os.WriteFile(module, []byte("ELF"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := findApacheModule(src)
	if err != nil {
		t.Fatalf("findApacheModule: %v", err)
	}
	if got != module {
		t.Fatalf("module = %q, want %q", got, module)
	}
}

func TestApacheNeedsRebuild(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PHPV_ROOT", root)
	profile := writeFakeCLIProfile(t, root, "7.4.33")
	svc := newApacheTestService(t, root)
	httpdPrefix := "/no-httpd"

	if !svc.NeedsRebuild("7.4.33", httpdPrefix, compiler.Toolchain{}) {
		t.Fatal("missing module must need a rebuild")
	}

	prefix := filepath.Join(root, "packages", "php-apache", "7.4.33")
	module := filepath.Join(prefix, "modules", "libphp.so")
	if err := os.MkdirAll(filepath.Dir(module), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(module, []byte("ELF"), 0o755); err != nil {
		t.Fatal(err)
	}
	hash := profile.Hash("modphp") + ":" + filepath.Base(httpdPrefix) + ":"
	if err := writeVariantMarker(prefix, VariantMarker{SAPI: "modphp", PHPVersion: "7.4.33", ProfileHash: hash}); err != nil {
		t.Fatal(err)
	}
	if svc.NeedsRebuild("7.4.33", httpdPrefix, compiler.Toolchain{}) {
		t.Fatal("matching marker must not need a rebuild")
	}
}
