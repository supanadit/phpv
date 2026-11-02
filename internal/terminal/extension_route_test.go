package terminal

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/pecl"
)

// fakePECLForRouting lets routing tests assert which PECL methods the
// handler calls, without any network or build activity.
type fakePECLForRouting struct {
	PECLService
	installCompatibleCalled int
	resolveReleaseCalled    int
	installCalled           int
	lastINI                 []string
}

func (f *fakePECLForRouting) InstallCompatible(ctx context.Context, name, phpVersion string, jobs int, extraINI []string) (*pecl.InstallResult, error) {
	f.installCompatibleCalled++
	f.lastINI = extraINI
	return &pecl.InstallResult{Name: name, Version: "1.2.3"}, nil
}

func (f *fakePECLForRouting) ResolveCompatibleRelease(name, phpVersion string) (string, error) {
	f.resolveReleaseCalled++
	return "1.2.3", nil
}

func (f *fakePECLForRouting) Install(ctx context.Context, source, phpVersion string, jobs int) (*pecl.InstallResult, error) {
	f.installCalled++
	return &pecl.InstallResult{Name: source}, nil
}

func (f *fakePECLForRouting) List(phpVersion string) ([]domain.ExtensionState, error) {
	return nil, nil
}

func (f *fakePECLForRouting) Uninstall(name, phpVersion string) error { return nil }

// fakeAssemblerForRouting stubs the assembler surface the routing path
// touches. Anything unknown to its defs map routes to PECL.
type fakeAssemblerForRouting struct {
	AssemblerService
	defs             map[string]domain.ExtensionDef
	installExtension int
}

func (f *fakeAssemblerForRouting) GetExtensionDef(name string) (domain.ExtensionDef, bool) {
	def, ok := f.defs[name]
	return def, ok
}

func (f *fakeAssemblerForRouting) InstallExtension(ctx context.Context, phpVersion, extName, phpSourceDir, phpPrefix string, jobs int) error {
	f.installExtension++
	return errBundledBuildAttempted
}

var errBundledBuildAttempted = fmt.Errorf("bundled build attempted (routing test sentinel)")

// newRoutingTestHandler builds a handler with a fake PHP install, a fake
// assembler, and the fake PECL service, plus a fake `php` binary inside
// the prefix.
func newRoutingTestHandler(t *testing.T) (*PHPHandler, *fakePECLForRouting, *fakeAssemblerForRouting, string) {
	t.Helper()
	root := t.TempDir()
	t.Setenv("PHPV_ROOT", root)
	version := "8.2.0"
	createFakePHPInstall(t, root, version)

	h := newTestPHPHandler(root)
	fakePECL := &fakePECLForRouting{}
	h.peclSvc = fakePECL
	fakeAsm := &fakeAssemblerForRouting{defs: map[string]domain.ExtensionDef{}}
	h.assemblerSvc = fakeAsm

	// PHP source tree with a bundled extension dir, so bundled routing can
	// be exercised without downloading real PHP source. source.Find needs a
	// build file at the tree root (configure) and a single top-level dir.
	treeRoot := filepath.Join(root, "sources", "php", version, "php-"+version)
	srcDir := filepath.Join(treeRoot, "ext", "bz2")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(treeRoot, "configure"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return h, fakePECL, fakeAsm, version
}

// runExtensionAdd builds a command carrying all `extension add` flags,
// parses the argv (flags + positionals), and invokes the handler with the
// positional extension names only.
func runExtensionAdd(t *testing.T, h *PHPHandler, args ...string) error {
	t.Helper()
	cmd := &cobra.Command{Use: "add", RunE: func(*cobra.Command, []string) error { return nil }}
	cmd.Flags().String("version", "", "")
	cmd.Flags().Bool("auto-deps", false, "")
	cmd.Flags().Bool("no-system", false, "")
	cmd.Flags().Bool("dry-run", false, "")
	cmd.Flags().Bool("force", false, "")
	cmd.Flags().Int("jobs", 0, "")
	cmd.Flags().StringArray("ini", nil, "")
	if err := cmd.ParseFlags(args); err != nil {
		return err
	}
	positional := cmd.Flags().Args()
	return h.extensionAdd(cmd, positional)
}

// parseINITFlagsForTest mirrors parseINITFlags with a plain cobra command
// carrying the given --ini values.
func parseINITFlagsForTest(values ...string) ([]string, error) {
	cmd := newCmd()
	cmd.Flags().StringArray("ini", nil, "")
	for _, v := range values {
		if err := cmd.Flags().Set("ini", v); err != nil {
			return nil, err
		}
	}
	return parseINITFlags(cmd)
}

func TestExtensionAdd_UnknownExtRoutesToPECL(t *testing.T) {
	h, fake, _, version := newRoutingTestHandler(t)

	// Source tree exists but has no ext/xdebug dir → PECL route.
	if err := runExtensionAdd(t, h, "xdebug", "--version", version); err != nil {
		t.Fatalf("extensionAdd xdebug: %v", err)
	}
	if fake.installCompatibleCalled != 1 {
		t.Fatalf("expected InstallCompatible to be called once, got %d", fake.installCompatibleCalled)
	}
	if fake.installCalled != 0 {
		t.Errorf("legacy Install should not be called by routing")
	}
}

func TestExtensionAdd_BundledExtStaysOnAssembler(t *testing.T) {
	h, fake, asm, version := newRoutingTestHandler(t)
	asm.defs["bz2"] = domain.ExtensionDef{Name: "bz2", Flag: "--with-bz2"}

	// bz2 exists in the fake source tree → bundled route. The stubbed
	// assembler returns a sentinel error we can assert on.
	err := runExtensionAdd(t, h, "bz2", "--version", version, "--no-system")
	if err == nil || !strings.Contains(err.Error(), "bundled build attempted") {
		t.Fatalf("expected bundled build to be attempted, got: %v", err)
	}
	if asm.installExtension != 1 {
		t.Errorf("assembler InstallExtension should be called once, got %d", asm.installExtension)
	}
	if fake.installCompatibleCalled != 0 {
		t.Errorf("PECL must not be called for bundled extensions")
	}
}

func TestExtensionAdd_DryRunResolvesOnly(t *testing.T) {
	h, fake, _, version := newRoutingTestHandler(t)

	if err := runExtensionAdd(t, h, "xdebug", "--version", version, "--dry-run"); err != nil {
		t.Fatalf("extensionAdd --dry-run xdebug: %v", err)
	}
	if fake.resolveReleaseCalled != 1 {
		t.Errorf("dry-run should resolve the release once, got %d", fake.resolveReleaseCalled)
	}
	if fake.installCompatibleCalled != 0 {
		t.Errorf("dry-run must not install")
	}
}

func TestExtensionAdd_INIFlagsPassedThrough(t *testing.T) {
	h, fake, _, version := newRoutingTestHandler(t)

	err := runExtensionAdd(t, h, "xdebug", "--version", version, "--ini", "xdebug.mode=debug", "--ini", "xdebug.start_with_request=yes")
	if err != nil {
		t.Fatalf("extensionAdd with --ini: %v", err)
	}
	want := []string{"xdebug.mode=debug", "xdebug.start_with_request=yes"}
	if len(fake.lastINI) != len(want) {
		t.Fatalf("INI settings = %v, want %v", fake.lastINI, want)
	}
	for i := range want {
		if fake.lastINI[i] != want[i] {
			t.Errorf("INI[%d] = %q, want %q", i, fake.lastINI[i], want[i])
		}
	}
}

func TestParseINITFlags_Validation(t *testing.T) {
	// Valid settings pass through.
	out, err := parseINITFlagsForTest("xdebug.mode=debug", "a.b=x")
	if err != nil || len(out) != 2 {
		t.Fatalf("parseINITFlags valid = %v, %v", out, err)
	}
	// Missing '=' rejected.
	if _, err := parseINITFlagsForTest("xdebug.mode"); err == nil {
		t.Error("expected error for setting without '='")
	}
	// Key with space rejected.
	if _, err := parseINITFlagsForTest("bad key=value"); err == nil {
		t.Error("expected error for key with whitespace")
	}
	// Empty value allowed (disabling a setting is legitimate).
	out, err = parseINITFlagsForTest("xdebug.mode=")
	if err != nil || len(out) != 1 {
		t.Fatalf("empty value should be allowed, got %v, %v", out, err)
	}
}
