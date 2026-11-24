package patches

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeTree materializes a patch tree: keys are slash-separated relative
// paths, values are file contents.
func writeTree(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for rel, content := range files {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestUserLocalShadowsSyncedDirectory(t *testing.T) {
	synced := filepath.Join(t.TempDir(), "patches")
	local := filepath.Join(t.TempDir(), "patches-local")
	writeTree(t, synced, map[string]string{
		"php/8.0.0-8.1.0/0001-synced.patch":   "--- a/configure\n+++ b/configure\n",
		"php/8.0.0-8.1.0/flags.yaml":          "extraCFlags:\n  - \"-Dsynced\"\n",
		"php/7.0.0-8.0.0/0001-synced74.patch": "--- a/configure\n+++ b/configure\n",
	})
	writeTree(t, local, map[string]string{
		"php/8.0.0-8.1.0/0001-local.patch": "--- a/configure\n+++ b/configure\n",
		"php/8.0.0-8.1.0/flags.yaml":       "extraCFlags:\n  - \"-Dlocal\"\n",
	})

	repo := NewRepository(synced, local)
	entries := repo.EntriesFor("php", "8.0.30")
	if len(entries) != 1 {
		t.Fatalf("EntriesFor = %d entries, want 1 (local shadows synced dir)", len(entries))
	}
	var e80 *Entry
	for i := range entries {
		if entries[i].RangeDir == "8.0.0-8.1.0" {
			e80 = &entries[i]
		}
	}
	if e80 == nil {
		t.Fatal("no entry for 8.0.0-8.1.0")
	}
	if e80.Source != SourceLocal {
		t.Errorf("source = %q, want user-local (shadowing)", e80.Source)
	}
	if len(e80.PatchFiles) != 1 || filepath.Base(e80.PatchFiles[0]) != "0001-local.patch" {
		t.Errorf("patch files = %v, want only the local patch", e80.PatchFiles)
	}
	if e80.Manifest == nil || len(e80.Manifest.ExtraCFlags) != 1 || e80.Manifest.ExtraCFlags[0] != "-Dlocal" {
		t.Errorf("manifest = %+v, want local flags (directory owned wholly by local tree)", e80.Manifest)
	}
	// 7.4 directory must not match 8.0.30.
	for _, e := range entries {
		if e.RangeDir == "7.0.0-8.0.0" {
			t.Errorf("7.0.0-8.0.0 selected for 8.0.30")
		}
	}
}

func TestUnclaimedSyncedDirectoryApplies(t *testing.T) {
	synced := filepath.Join(t.TempDir(), "patches")
	local := filepath.Join(t.TempDir(), "patches-local")
	writeTree(t, synced, map[string]string{
		"oniguruma/6.0.0+/0001-synced.patch": "--- a/st.h\n+++ b/st.h\n",
	})
	writeTree(t, local, map[string]string{
		"oniguruma/6.9.7/0001-hotfix.patch": "--- a/st.h\n+++ b/st.h\n",
	})

	repo := NewRepository(synced, local)
	entries := repo.EntriesFor("oniguruma", "6.9.7")
	if len(entries) != 2 {
		t.Fatalf("EntriesFor = %d entries, want 2", len(entries))
	}
	sources := map[string]string{}
	for _, e := range entries {
		sources[e.RangeDir] = e.Source
	}
	if sources["6.0.0+"] != SourceSynced || sources["6.9.7"] != SourceLocal {
		t.Errorf("sources = %v, want synced for 6.0.0+ and user-local for 6.9.7", sources)
	}
}

func TestManifestOnlyDirectorySelectsFlags(t *testing.T) {
	synced := filepath.Join(t.TempDir(), "patches")
	writeTree(t, synced, map[string]string{
		"libzip/any/flags.yaml": "configureFlags:\n  - \"-DZLIB_ROOT={{dep:zlib}}\"\n",
	})
	repo := NewRepository(synced, "")
	patches := repo.PatchesFor("libzip", "1.11.3")
	if len(patches) != 1 {
		t.Fatalf("PatchesFor = %d, want 1", len(patches))
	}
	p := patches[0]
	if p.Apply != nil {
		t.Error("flags-only entry must not attempt patch application")
	}
	if len(p.ConfigureFlags) != 1 || p.ConfigureFlags[0] != "-DZLIB_ROOT={{dep:zlib}}" {
		t.Errorf("ConfigureFlags = %v", p.ConfigureFlags)
	}
	if p.VersionRange != "" {
		t.Errorf("VersionRange = %q, want empty (any)", p.VersionRange)
	}
}

func TestRangeMatchNoMatchAndOrdering(t *testing.T) {
	synced := filepath.Join(t.TempDir(), "patches")
	writeTree(t, synced, map[string]string{
		"php/7.0.0-8.0.0/0002-second.patch": "--- a/configure\n+++ b/configure\n",
		"php/7.0.0-8.0.0/0001-first.patch":  "--- a/configure\n+++ b/configure\n",
		"php/8.0.0-8.1.0/0001-eight.patch":  "--- a/configure\n+++ b/configure\n",
	})
	repo := NewRepository(synced, "")

	// One directory yields one Patch whose Apply runs its files in
	// ordinal order (0001-first, then 0002-second).
	patches := repo.PatchesFor("php", "7.4.33")
	if len(patches) != 1 {
		t.Fatalf("PatchesFor(7.4.33) = %d, want 1", len(patches))
	}
	if patches[0].Name != "0001-first" {
		t.Errorf("Name = %s, want 0001-first (first ordinal)", patches[0].Name)
	}
	p := patches[0]
	if p.VersionRange != ">=7.0.0,<8.0.0" {
		t.Errorf("VersionRange = %q", p.VersionRange)
	}
	if got := repo.PatchesFor("php", "8.0.30"); len(got) != 1 || got[0].Name != "0001-eight" {
		t.Errorf("PatchesFor(8.0.30) = %+v", got)
	}
	if got := repo.PatchesFor("php", "8.1.0"); len(got) != 0 {
		t.Errorf("8.1.0 matched %d patches, want 0 (outside both ranges)", len(got))
	}
}

func TestPatchesForNoTreeIsEmptyNotError(t *testing.T) {
	repo := NewRepository(filepath.Join(t.TempDir(), "does-not-exist"), "")
	if got := repo.PatchesFor("php", "8.0.0"); len(got) != 0 {
		t.Fatalf("PatchesFor on missing tree = %d patches, want 0", len(got))
	}
}

// TestPublishedCurlLegacyRangeHasEraCFlags reads the real publication tree
// (../../..//patches) and verifies the legacy-curl range directory selects
// era CFLAGS for the archeology-era versions while leaving modern releases
// untouched. Without -std=gnu89 (or equivalent), GCC 14+ rejects curl 7.2x's
// K&R conftest programs (implicit-int main) and configure aborts with
// "libs available at link-time are not available run-time".
func TestPublishedCurlLegacyRangeHasEraCFlags(t *testing.T) {
	pubRoot := filepath.Join("..", "..", "..", "patches")
	if _, err := os.Stat(pubRoot); err != nil {
		t.Skipf("publication tree not found at %s: %v", pubRoot, err)
	}
	repo := NewRepository(pubRoot, "")

	legacy := repo.PatchesFor("curl", "7.20.0")
	var cflags []string
	for _, p := range legacy {
		cflags = append(cflags, p.ExtraCFlags...)
	}
	joined := strings.Join(cflags, " ")
	if !strings.Contains(joined, "-std=gnu89") {
		t.Errorf("curl 7.20.0 ExtraCFlags = %q, want -std=gnu89 (GCC 14+ K&R conftest compat)", joined)
	}

	for _, modern := range []string{"7.80.0", "8.10.1"} {
		var modernFlags []string
		for _, p := range repo.PatchesFor("curl", modern) {
			modernFlags = append(modernFlags, p.ExtraCFlags...)
		}
		if joined := strings.Join(modernFlags, " "); strings.Contains(joined, "-std=gnu89") {
			t.Errorf("curl %s should not receive legacy era flags, got %q", modern, joined)
		}
	}
}

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
	err := applyPatches(patchSet{pkg: "php", version: "8.0.30", rangeDir: "8.0.0-8.1.0", files: []string{patch}, sourceDir: root})
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
	if err := applyPatches(patchSet{pkg: "php", version: "8.0.30", rangeDir: "any", files: []string{first, second}, sourceDir: root}); err != nil {
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

func TestDirRangeExpr(t *testing.T) {
	tests := []struct {
		dir  string
		want string
	}{
		{"any", ""},
		{"8.0.0-8.1.0", ">=8.0.0,<8.1.0"},
		{"6.0.0+", ">=6.0.0"},
		{"6.9.7", "=6.9.7"},
		{"5.0.0-7.0.0", ">=5.0.0,<7.0.0"},
		{"6.9.9-6.9.10", ">=6.9.9,<6.9.10"},
	}
	for _, tt := range tests {
		if got := DirRangeExpr(tt.dir); got != tt.want {
			t.Errorf("DirRangeExpr(%q) = %q, want %q", tt.dir, got, tt.want)
		}
	}
}

func TestDirRangeExprMatchesVersions(t *testing.T) {
	// End-to-end through the real matcher, pinning the semantics.
	tests := []struct {
		dir     string
		version string
		want    bool
	}{
		{"8.0.0-8.1.0", "8.0.30", true},
		{"8.0.0-8.1.0", "8.1.0", false},
		{"8.0.0-8.1.0", "7.4.33", false},
		{"6.0.0+", "6.9.9", true},
		{"6.0.0+", "6.0.0", true},
		{"6.0.0+", "5.9.6", false},
		{"6.9.7", "6.9.7", true},
		{"6.9.7", "6.9.8", false},
	}
	for _, tt := range tests {
		if got := entryMatches(Entry{RangeDir: tt.dir}, tt.version); got != tt.want {
			t.Errorf("dir %q vs %s = %v, want %v", tt.dir, tt.version, got, tt.want)
		}
	}
}

func TestParseManifestPreservesPlaceholders(t *testing.T) {
	data := []byte(`extraCFlags:
  - "-Wno-error=incompatible-pointer-types"
  - "-Wno-incompatible-pointer-types"
configureFlags:
  - "-DZLIB_ROOT={{dep:zlib}}"
  - "--with-openssl={{dep:openssl}}"
  - "--prefix={{prefix}}"
  - "{{source}}"
`)
	m, err := ParseManifest(data)
	if err != nil {
		t.Fatalf("ParseManifest: %v", err)
	}
	wantCFlags := []string{"-Wno-error=incompatible-pointer-types", "-Wno-incompatible-pointer-types"}
	if len(m.ExtraCFlags) != len(wantCFlags) {
		t.Fatalf("ExtraCFlags = %v, want %v", m.ExtraCFlags, wantCFlags)
	}
	for i, w := range wantCFlags {
		if m.ExtraCFlags[i] != w {
			t.Errorf("ExtraCFlags[%d] = %q, want %q", i, m.ExtraCFlags[i], w)
		}
	}
	wantConfigure := []string{
		"-DZLIB_ROOT={{dep:zlib}}",
		"--with-openssl={{dep:openssl}}",
		"--prefix={{prefix}}",
		"{{source}}",
	}
	if len(m.ConfigureFlags) != len(wantConfigure) {
		t.Fatalf("ConfigureFlags = %v, want %v", m.ConfigureFlags, wantConfigure)
	}
	for i, w := range wantConfigure {
		if m.ConfigureFlags[i] != w {
			t.Errorf("ConfigureFlags[%d] = %q, want %q", i, m.ConfigureFlags[i], w)
		}
	}
}

func TestParseManifestEmpty(t *testing.T) {
	m, err := ParseManifest([]byte(""))
	if err != nil {
		t.Fatalf("ParseManifest: %v", err)
	}
	if m == nil || len(m.ExtraCFlags) != 0 || len(m.ConfigureFlags) != 0 {
		t.Fatalf("expected empty manifest, got %+v", m)
	}
}

func TestParseManifestInvalidYAML(t *testing.T) {
	if _, err := ParseManifest([]byte("extraCFlags: [unclosed")); err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestLoadManifestMissingIsNil(t *testing.T) {
	dir := t.TempDir()
	m, err := LoadManifest(filepath.Join(dir, ManifestFileName))
	if err != nil {
		t.Fatalf("LoadManifest on missing file: %v", err)
	}
	if m != nil {
		t.Fatalf("expected nil manifest, got %+v", m)
	}
}

func TestLoadManifestFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ManifestFileName)
	if err := os.WriteFile(path, []byte("configureFlags:\n  - \"-DZLIB_ROOT={{dep:zlib}}\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m, err := LoadManifest(path)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	if len(m.ConfigureFlags) != 1 || m.ConfigureFlags[0] != "-DZLIB_ROOT={{dep:zlib}}" {
		t.Fatalf("ConfigureFlags = %v", m.ConfigureFlags)
	}
}

// TestNoEmbeddedPatchData pins the "zero baked-in patch data" rule: the
// binary ships no patches, so no go:embed directive may exist in this
// package (or anywhere it could pull patch content from) and patch
// content must only be reachable through injected filesystem roots.
func TestNoEmbeddedPatchData(t *testing.T) {
	err := filepath.WalkDir(".", func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || strings.HasSuffix(path, "_test.go") || !strings.HasSuffix(path, ".go") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(data), "go:embed") {
			t.Errorf("%s contains a go:embed directive; patches must never be embedded", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
