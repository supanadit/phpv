package patches

import (
	"os"
	"path/filepath"
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
	entries := repo.(*Repository).EntriesFor("php", "8.0.30")
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
	entries := repo.(*Repository).EntriesFor("oniguruma", "6.9.7")
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
	if got := repo.PatchesFor("php", "8.0.30"); len(got) != 0 {
		t.Errorf("missing tree returned %v, want empty", got)
	}
}
