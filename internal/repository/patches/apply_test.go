package patches

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const samplePatch = `--- a/hello.txt
+++ b/hello.txt
@@ -1,1 +1,1 @@
-hello
+world
`

func writeSource(t *testing.T, nested bool, content string) string {
	t.Helper()
	root := t.TempDir()
	src := root
	if nested {
		src = filepath.Join(root, "pkg-1.2.3")
		if err := os.MkdirAll(src, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(src, "hello.txt"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestApplyCleanOnNestedRoot(t *testing.T) {
	root := writeSource(t, true, "hello\n")
	patch := filepath.Join(t.TempDir(), "0001-test.patch")
	if err := os.WriteFile(patch, []byte(samplePatch), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := applyPatchFile(patch, root); err != nil {
		t.Fatalf("applyPatchFile: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(root, "pkg-1.2.3", "hello.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "world\n" {
		t.Errorf("content = %q, want patched", data)
	}
}

func TestApplyContextMismatchNamesFileAndRoot(t *testing.T) {
	root := writeSource(t, true, "different\n")
	patch := filepath.Join(t.TempDir(), "0001-intl-cxx17.patch")
	if err := os.WriteFile(patch, []byte(samplePatch), 0o644); err != nil {
		t.Fatal(err)
	}
	err := applyPatches("php", "8.0.30", "8.0.0-8.1.0", []string{patch}, root)
	if err == nil {
		t.Fatal("expected loud failure on context mismatch")
	}
	msg := err.Error()
	for _, want := range []string{"0001-intl-cxx17.patch", "php@8.0.30"} {
		if !strings.Contains(msg, want) {
			t.Errorf("error %q does not mention %q", msg, want)
		}
	}
}

func TestApplyAlreadyAppliedTolerated(t *testing.T) {
	root := writeSource(t, true, "world\n") // patch already applied
	patch := filepath.Join(t.TempDir(), "0001-test.patch")
	if err := os.WriteFile(patch, []byte(samplePatch), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := applyPatchFile(patch, root); err != nil {
		t.Fatalf("already-applied patch must be tolerated, got: %v", err)
	}
}

func TestApplyOrdinalOrder(t *testing.T) {
	root := writeSource(t, false, "hello\n")
	dir := t.TempDir()
	first := filepath.Join(dir, "0001-first.patch")
	second := filepath.Join(dir, "0002-second.patch")
	// Second patch builds on the first's output.
	if err := os.WriteFile(first, []byte(samplePatch), 0o644); err != nil {
		t.Fatal(err)
	}
	secondPatch := strings.Replace(samplePatch, "-hello\n+world\n", "-world\n+final\n", 1)
	if err := os.WriteFile(second, []byte(secondPatch), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := applyPatches("php", "8.0.30", "any", []string{first, second}, root); err != nil {
		t.Fatalf("applyPatches: %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(root, "hello.txt"))
	if string(data) != "final\n" {
		t.Errorf("content = %q, want ordinal application result", data)
	}
}

func TestMissingPatchToolPreflight(t *testing.T) {
	t.Setenv("PATH", t.TempDir()) // empty PATH: no `patch` binary
	err := preflightPatchTool()
	if err == nil {
		t.Skip("patch binary still reachable; environment does not honor empty PATH")
	}
	if !strings.Contains(err.Error(), "patch") || !strings.Contains(err.Error(), "install") {
		t.Errorf("preflight error %q lacks tool name and install hint", err)
	}
}

func TestResolveSourceRootFallsBackToSourceDir(t *testing.T) {
	root := t.TempDir() // empty tree
	patch := filepath.Join(t.TempDir(), "p.patch")
	if err := os.WriteFile(patch, []byte(samplePatch), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := resolveSourceRoot(patch, root)
	if err != nil {
		t.Fatal(err)
	}
	if got != root {
		t.Errorf("resolveSourceRoot = %q, want fallback %q", got, root)
	}
}

func TestFirstDiffTarget(t *testing.T) {
	cases := []struct {
		name, diff, want string
		wantErr          bool
	}{
		{"git", "--- a/ext/standard/scanf.c\n+++ b/ext/standard/scanf.c\n", "ext/standard/scanf.c", false},
		{"timestamp", "--- a/configure\t2024-01-01\n+++ b/configure\t2024-01-01\n", "configure", false},
		{"created-file", "--- /dev/null\n+++ b/new.c\n", "new.c", false},
		{"no-target", "some random text\n", "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := filepath.Join(t.TempDir(), "p.patch")
			if err := os.WriteFile(p, []byte(tc.diff), 0o644); err != nil {
				t.Fatal(err)
			}
			got, err := firstDiffTarget(p)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("want error, got %q", got)
				}
				return
			}
			if err != nil || got != tc.want {
				t.Fatalf("firstDiffTarget = %q, %v; want %q", got, err, tc.want)
			}
		})
	}
}
