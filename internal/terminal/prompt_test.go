package terminal

import (
	"os"
	"strings"
	"testing"

	"github.com/supanadit/phpv/system"
)

const versionGateSentence = "satisfies the build requirement"

func TestPromptAndInstall_DependencyPromptStatesVersionGate(t *testing.T) {
	sysSvc := &fakeSystemService{
		checkResults: []*system.CheckResult{missingResult("curl")},
	}
	h := &PHPHandler{systemSvc: sysSvc}

	out := captureStdout(t, func() {
		// dryRun so no install runs; the prompt text is what we assert on.
		if _, err := h.promptAndInstall([]system.Package{{Name: "curl"}}, false, true, true); err != nil {
			t.Errorf("promptAndInstall: %v", err)
		}
	})

	if !strings.Contains(out, versionGateSentence) {
		t.Errorf("dependency prompt does not explain the version gate.\noutput:\n%s", out)
	}
}

func TestPromptAndInstall_BuildToolPromptOmitsVersionGate(t *testing.T) {
	sysSvc := &fakeSystemService{}
	h := &PHPHandler{systemSvc: sysSvc}

	out := captureStdout(t, func() {
		if _, err := h.promptAndInstall([]system.Package{{Name: "make"}}, false, true, false); err != nil {
			t.Errorf("promptAndInstall: %v", err)
		}
	})

	if strings.Contains(out, versionGateSentence) {
		t.Errorf("build-tool prompt must not mention a version gate; none applies.\noutput:\n%s", out)
	}
}

func TestPromptAndInstall_MarksAdvisoryPackages(t *testing.T) {
	h := &PHPHandler{systemSvc: &fakeSystemService{}}
	pkgs := []system.Package{
		{Name: "libzip", SystemName: "libzip-devel", Required: true},
		{Name: "pcre2", SystemName: "pcre2-devel", Required: false},
	}

	out := captureStdout(t, func() {
		if _, err := h.promptAndInstall(pkgs, false, true, true); err != nil {
			t.Errorf("promptAndInstall: %v", err)
		}
	})

	reqLine := lineContaining(out, "libzip")
	advLine := lineContaining(out, "pcre2")
	if reqLine == "" || advLine == "" {
		t.Fatalf("both packages should be listed.\noutput:\n%s", out)
	}
	if strings.Contains(reqLine, "[optional]") {
		t.Errorf("required package rendered as optional: %q", reqLine)
	}
	if !strings.Contains(advLine, "[optional]") {
		t.Errorf("advisory package not marked optional: %q", advLine)
	}
}

func lineContaining(s, sub string) string {
	for _, line := range strings.Split(s, "\n") {
		if strings.Contains(line, sub) {
			return line
		}
	}
	return ""
}

func TestPromptAndInstall_NonInteractiveSkipsInstall(t *testing.T) {
	sysSvc := &fakeSystemService{}
	h := &PHPHandler{systemSvc: sysSvc}

	// Replace stdin with a pipe (not a char device), which is the
	// non-interactive case.
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	orig := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = orig; r.Close(); w.Close() }()

	out := captureStdout(t, func() {
		installed, err := h.promptAndInstall([]system.Package{{Name: "curl"}}, false, false, true)
		if err != nil {
			t.Errorf("promptAndInstall: %v", err)
		}
		if installed {
			t.Error("installed = true for a non-interactive session, want false")
		}
	})

	if len(sysSvc.installed) != 0 {
		t.Errorf("Install called %d times in a non-interactive session, want 0", len(sysSvc.installed))
	}
	if !strings.Contains(out, "non-interactive") {
		t.Errorf("expected a non-interactive notice.\noutput:\n%s", out)
	}
}
