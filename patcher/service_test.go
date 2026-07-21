package patcher

import (
	"testing"
)

type mockPatcherRepo struct {
	patches []Patch
}

func (m *mockPatcherRepo) PatchesFor(name, version string) []Patch {
	return m.patches
}

func TestService_PatchesFor_FiltersByVersion(t *testing.T) {
	mock := &mockPatcherRepo{
		patches: []Patch{
			{Name: "scanf-fix", VersionRange: ">=8.0.0,<8.4.0"},
			{Name: "icu-fix", VersionRange: ">=8.0.0"},
			{Name: "always", VersionRange: ""},
		},
	}
	svc := NewService(mock)

	patches := svc.PatchesFor("php", "8.3.0")
	if len(patches) != 3 {
		t.Fatalf("expected 3 patches for 8.3.0, got %d", len(patches))
	}

	patches = svc.PatchesFor("php", "8.4.0")
	if len(patches) != 2 {
		t.Fatalf("expected 2 patches for 8.4.0, got %d", len(patches))
	}
	for _, p := range patches {
		if p.Name == "scanf-fix" {
			t.Fatal("scanf-fix should not apply to 8.4.0")
		}
	}
}

func TestService_Prepare_AppliesPatches(t *testing.T) {
	applied := false
	mock := &mockPatcherRepo{
		patches: []Patch{
			{
				Name:           "test-patch",
				Apply:          func(dir string) error { applied = true; return nil },
				ExtraCFlags:    []string{"-Wno-deprecated"},
				ConfigureFlags: []string{"--with-test={{prefix}}"},
			},
		},
	}
	svc := NewService(mock)

	result, err := svc.Prepare("php", "8.3.0", "/src")
	if err != nil {
		t.Fatalf("Prepare returned error: %v", err)
	}
	if !applied {
		t.Fatal("patch was not applied")
	}
	if len(result.Applied) != 1 || result.Applied[0] != "test-patch" {
		t.Fatalf("expected test-patch in Applied, got %v", result.Applied)
	}
	if len(result.ExtraCFlags) != 1 || result.ExtraCFlags[0] != "-Wno-deprecated" {
		t.Fatalf("unexpected ExtraCFlags: %v", result.ExtraCFlags)
	}
	if len(result.ConfigureFlags) != 1 || result.ConfigureFlags[0] != "--with-test={{prefix}}" {
		t.Fatalf("unexpected ConfigureFlags: %v", result.ConfigureFlags)
	}
}

func TestService_Prepare_NoPatches(t *testing.T) {
	mock := &mockPatcherRepo{patches: nil}
	svc := NewService(mock)

	result, err := svc.Prepare("php", "8.3.0", "/src")
	if err != nil {
		t.Fatalf("Prepare returned error: %v", err)
	}
	if len(result.Applied) != 0 {
		t.Fatalf("expected 0 applied, got %d", len(result.Applied))
	}
}
