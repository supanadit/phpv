package disk

import (
	"os/exec"
	"testing"
)

func TestLibraryPackagesMap(t *testing.T) {
	expectedLibraries := map[string]string{
		"libxml2":   "libxml-2.0",
		"openssl":   "openssl",
		"curl":      "libcurl",
		"zlib":      "zlib",
		"oniguruma": "oniguruma",
	}

	for pkgName, pkgConfigName := range expectedLibraries {
		if got, ok := libraryPackages[pkgName]; !ok {
			t.Errorf("libraryPackages missing expected package: %s", pkgName)
		} else if got != pkgConfigName {
			t.Errorf("libraryPackages[%s] = %s, want %s", pkgName, got, pkgConfigName)
		}
	}
}

func TestCheckSystemPackage_Libraries(t *testing.T) {
	repo := &AdvisorRepository{
		exec: &defaultExecutor{},
	}

	libraryTests := []struct {
		name          string
		pkgName       string
		wantAvailable bool
	}{
		{"libxml2 is a library", "libxml2", true},
		{"openssl is a library", "openssl", true},
		{"curl is a library", "curl", true},
		{"zlib is a library", "zlib", true},
		{"m4 is not a library", "m4", true},
		{"autoconf is not a library", "autoconf", true},
	}

	for _, tt := range libraryTests {
		t.Run(tt.name, func(t *testing.T) {
			available, path, _ := repo.checkSystemPackage(tt.pkgName)
			t.Logf("checkSystemPackage(%q) = available=%v, path=%q", tt.pkgName, available, path)

			if available != tt.wantAvailable {
				t.Errorf("checkSystemPackage(%q) available = %v, want %v", tt.pkgName, available, tt.wantAvailable)
			}
		})
	}

	t.Run("oniguruma detection depends on system", func(t *testing.T) {
		available, path, _ := repo.checkSystemPackage("oniguruma")
		t.Logf("oniguruma: available=%v, path=%q", available, path)
	})
}

func TestCheckSystemLibrary_WithRealSystem(t *testing.T) {
	repo := &AdvisorRepository{
		exec: &defaultExecutor{},
	}

	t.Run("libxml2 detected via headers", func(t *testing.T) {
		available, path, _ := repo.checkSystemLibrary("libxml2", "libxml-2.0")
		if !available {
			t.Log("libxml2 not found via pkg-config or headers")
		} else {
			t.Logf("libxml2 detected via: %s", path)
		}
	})

	t.Run("openssl detected via pkg-config", func(t *testing.T) {
		available, path, _ := repo.checkSystemLibrary("openssl", "openssl")
		if !available {
			t.Error("openssl should be available via pkg-config")
		} else {
			t.Logf("openssl detected via: %s", path)
		}
	})
}

func TestAdvisorRepository_ExecutableDetection(t *testing.T) {
	repo := &AdvisorRepository{
		exec: &defaultExecutor{},
	}

	executables := []string{"make", "gcc", "bison", "flex"}

	for _, exe := range executables {
		t.Run(exe, func(t *testing.T) {
			available, _, _ := repo.checkSystemPackage(exe)
			if !available {
				t.Errorf("Expected %s to be available on system", exe)
			}
		})
	}
}

func TestAdvisorRepository_NonExistentExecutable(t *testing.T) {
	repo := &AdvisorRepository{
		exec: &defaultExecutor{},
	}

	available, _, _ := repo.checkSystemPackage("nonexistent_command_xyz_123")
	if available {
		t.Error("nonexistent command should not be available")
	}
}

func TestAdvisorRepository_LibraryPaths(t *testing.T) {
	repo := &AdvisorRepository{
		exec: &defaultExecutor{},
	}

	libs := []string{"libxml-2.0", "openssl", "libcurl", "zlib"}

	for _, lib := range libs {
		t.Run(lib, func(t *testing.T) {
			available, _, _ := repo.checkSystemLibrary(lib, lib)
			t.Logf("Library %s: available=%v", lib, available)
		})
	}
}

func TestAdvisorRepository_CheckSystemLibrary_NotFound(t *testing.T) {
	repo := &AdvisorRepository{
		exec: &defaultExecutor{},
	}

	available, _, _ := repo.checkSystemLibrary("nonexistent_library_xyz", "nonexistent_library_xyz")
	if available {
		t.Error("nonexistent library should not be available")
	}
}

func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}
