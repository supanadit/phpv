package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// preBumpRegistry is the registry state before 2026-11-24, when 8.6.0 /
// 8.5.13 / 8.4.28 / 8.3.37 / 8.2.36 landed and 8.6 had to be added to CI.
const preBumpRegistry = `case "php":
		result = BuildRegistries(PackageConfig{
			Name: "php",
			Ranges: vsn.BuildRanges(
				vsn.BuildMinorRanges(8, []vsn.MinorRange{
					{Minor: 0, PatchEnd: 30},
					{Minor: 1, PatchEnd: 33},
					{Minor: 2, PatchEnd: 35},
					{Minor: 3, PatchEnd: 36},
					{Minor: 4, PatchEnd: 27},
					{Minor: 5, PatchEnd: 12},
				}),
			),
			Checksums: []Checksum{
				{
					Version: "8.5.12",
					Type:    "sha256",
					Value:   "81a0f450349e84b08eb43faa85f8a4c835684f3cc1c50471775d1317a046c525",
				},
			},
		})`

const preBumpWorkflow = `        php_version:
          - "7.0"
          - "8.4"
          - "8.5"

    env:`

// The diff must name every change upstream has that we lack.
func TestDiff_ReportsPatchBumpsAndNewMinor(t *testing.T) {
	state := map[string]int{"8.2": 35, "8.3": 36, "8.4": 27, "8.5": 12}
	up := map[string]int{"8.2": 36, "8.3": 37, "8.4": 28, "8.5": 13, "8.6": 0, "7.4": 33}
	got := diff(state, up)
	want := []string{
		"NEW MINOR  8.6.0 (registry has none)",
		"PATCH BUMP 8.2: 35 -> 36",
		"PATCH BUMP 8.3: 36 -> 37",
		"PATCH BUMP 8.4: 27 -> 28",
		"PATCH BUMP 8.5: 12 -> 13",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("diff = %q, want %q", got, want)
	}
	if d := diff(state, state); len(d) != 0 {
		t.Fatalf("diff with identical state = %q, want none", d)
	}
}

// Only lagging/new 8.x entries are "changed"; nothing else may trigger a
// download — old majors live on museum.php.net and are not touched.
func TestChanged_OnlyLaggingEightX(t *testing.T) {
	state := map[string]int{"8.4": 28, "8.5": 13, "8.6": 0}
	up := map[string]int{"8.4": 28, "8.5": 13, "8.6": 0, "8.1": 34, "5.1": 6, "7.4": 33}
	got := changed(state, up)
	if len(got) != 1 || got["8.1"] != 34 {
		t.Fatalf("changed = %v, want only 8.1:34", got)
	}
	if c := changed(state, state); len(c) != 0 {
		t.Fatalf("changed with identical state = %v, want none", c)
	}
}

func TestStableTag_ExcludesPrereleases(t *testing.T) {
	for tag, ok := range map[string]bool{
		"php-8.6.0": true, "php-8.5.13": true, "php-7.4.33": true,
		"php-8.6.0RC1": false, "php-8.6.0alpha1": false, "php-8.6.0beta2": false,
		"php-8.6.0RC4": false, "php-8.6.0preview": false, "php-8.4.0RC1": false,
		"v8.6.0": false, "php-8.6": false, "php-8.6.0.1": false,
	} {
		if got := stableTag.MatchString(tag); got != ok {
			t.Errorf("stableTag.MatchString(%q) = %v, want %v", tag, got, ok)
		}
	}
}

// Replay the 2026-11-24 bump: starting from the real pre-bump files, with
// upstream at the real 2026-11-21 release, -apply must produce the registry
// and CI matrix that were committed by hand. Checksums are injected so the
// test is offline; only the parsing/rewriting is under test here.
func TestApply_ReplaysReal20261124Bump(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) string {
		p := filepath.Join(dir, name)
		os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	reg := write("registry_other.go", preBumpRegistry)
	// Shaped like the real file: the PHP checksum test first, then a curl test
	// that also has a wantChecksums map and must survive untouched.
	test := write("registry_test.go",
		"func TestRegistryRepository_List_PHP_ChecksumFilter(t *testing.T) {\n"+
			"\tif len(onlyWithChecksum) != 7 {\n\t\tt.Fatalf(\"x\")\n\t}\n"+
			"\twantChecksums := map[string]string{\n\t\t\"8.5.12\": \"stale\",\n\t}\n}\n"+
			"func TestRegistryRepository_Get_CURL_Checksums(t *testing.T) {\n"+
			"\twantChecksums := map[string]string{\n\t\t\"7.20.0\": \"curlhash\",\n\t}\n}")
	wf := write("test.yaml", preBumpWorkflow)

	// Redirect the tool's file targets to the temp copies.
	oldReg, oldTest, oldWf := registryFile, testFile, workflowFile
	registryFile, testFile, workflowFile = reg, test, wf
	defer func() { registryFile, testFile, workflowFile = oldReg, oldTest, oldWf }()

	up := map[string]int{"8.2": 36, "8.3": 37, "8.4": 28, "8.5": 13, "8.6": 0}
	if err := updateRangeTable(up); err != nil {
		t.Fatal(err)
	}
	if err := updateTestCount(); err != nil {
		t.Fatal(err)
	}
	if err := updateWorkflowMatrix(map[string]int{"8.6": 0, "8.5": 13}); err != nil {
		t.Fatal(err)
	}

	got := read(t, reg)
	for _, want := range []string{
		"{Minor: 2, PatchEnd: 36},", "{Minor: 3, PatchEnd: 37},",
		"{Minor: 4, PatchEnd: 28},", "{Minor: 5, PatchEnd: 13},",
		"{Minor: 6, PatchEnd: 0},",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("registry missing %q:\n%s", want, got)
		}
	}
	// Rows keep their order and the new minor is last.
	if !regexp.MustCompile(`Minor: 5, PatchEnd: 13\},\s*\{Minor: 6, PatchEnd: 0\},`).MatchString(got) {
		t.Errorf("8.6 row not appended after 8.5:\n%s", got)
	}
	// Unrelated majors (7.x) are untouched — they live in a separate call.
	if strings.Contains(got, "PatchEnd: 99") {
		t.Error("unexpected edit")
	}

	gotTest := read(t, test)
	// The fixture registry has exactly one checksum entry (8.5.12), so the
	// count and map must be derived from the registry, replacing stale values.
	if !strings.Contains(gotTest, "len(onlyWithChecksum) != 1") {
		t.Errorf("test count should be derived from registry checksums (1 here):\n%s", gotTest)
	}
	wantEntry := `"8.5.12": "81a0f450349e84b08eb43faa85f8a4c835684f3cc1c50471775d1317a046c525",`
	if !strings.Contains(gotTest, wantEntry) {
		t.Errorf("checksum map not rewritten from registry, want %s:\n%s", wantEntry, gotTest)
	}
	if strings.Contains(gotTest, `"stale"`) || strings.Contains(gotTest, "!= 7") {
		t.Errorf("stale count/map left behind:\n%s", gotTest)
	}
	if !strings.Contains(gotTest, `"7.20.0": "curlhash",`) {
		t.Errorf("curl wantChecksums map was corrupted:\n%s", gotTest)
	}

	gotWf := read(t, wf)
	if !strings.Contains(gotWf, "- \"8.5\"\n          - \"8.6\"\n") {
		t.Errorf("workflow matrix missing 8.6 after 8.5:\n%s", gotWf)
	}
}

func TestApply_IsIdempotent(t *testing.T) {
	dir := t.TempDir()
	reg := filepath.Join(dir, "r.go")
	os.WriteFile(reg, []byte(preBumpRegistry), 0o644)
	oldReg := registryFile
	registryFile = reg
	defer func() { registryFile = oldReg }()

	up := map[string]int{"8.5": 13, "8.6": 0}
	if err := updateRangeTable(up); err != nil {
		t.Fatal(err)
	}
	first := read(t, reg)
	if err := updateRangeTable(up); err != nil {
		t.Fatal(err)
	}
	second := read(t, reg)
	if first != second {
		t.Fatalf("updateRangeTable is not idempotent:\n--- first\n%s\n--- second\n%s", first, second)
	}
	if strings.Count(second, "{Minor: 6, PatchEnd: 0}") != 1 {
		t.Fatalf("8.6 row duplicated on second apply:\n%s", second)
	}
}

func read(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
