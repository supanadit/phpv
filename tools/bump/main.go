// Command bump is the single writer of phpv's PHP version registry. It is run
// by .github/workflows/autobump.yaml, never by hand.
//
//	go run ./tools/bump -check            # exit 1 if upstream has versions phpv lacks
//	go run ./tools/bump -apply            # edit registry_other.go, registry_test.go, test.yaml
//
// -apply is idempotent: applying the same upstream state twice is a no-op.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"go/format"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	distributions = "https://www.php.net/distributions/php-%s.tar.gz"
)

// File targets are vars so tests can redirect them to temp copies.
var (
	registryFile = "internal/repository/memory/registry_other.go"
	testFile     = "internal/repository/memory/registry_test.go"
	workflowFile = ".github/workflows/test.yaml"
)

// stableTag matches GA release tags; RC/alpha/beta/preview tags never match.
var stableTag = regexp.MustCompile(`^php-(\d+)\.(\d+)\.(\d+)$`)

type version struct{ major, minor, patch int }

func (v version) String() string { return fmt.Sprintf("%d.%d.%d", v.major, v.minor, v.patch) }

func main() {
	check := flag.Bool("check", false, "report whether upstream has versions missing from the registry")
	apply := flag.Bool("apply", false, "rewrite the registry to the latest upstream state")
	matrix := flag.Bool("matrix", false, "print the changed major.minor list as a JSON array for the CI matrix")
	flag.Parse()
	switch {
	case *check && !*apply && !*matrix, *apply && !*check && !*matrix, *matrix && !*check && !*apply:
	default:
		fatalf("pass exactly one of -check, -apply or -matrix")
	}

	upstream, err := latestStable()
	if err != nil {
		fatalf("%v", err)
	}
	state, err := readRegistryState()
	if err != nil {
		fatalf("%v", err)
	}

	changed := changed(state, upstream)
	changes := diff(state, upstream)
	if len(changes) == 0 {
		fmt.Println("registry is up to date")
		if *matrix {
			fmt.Println("[]")
		}
		return
	}
	if *matrix {
		// Compile the newest patch of each changed minor (e.g. "8.6"), which
		// is what `phpv install` resolves, so CI builds exactly what changed.
		minors := make([]string, 0, len(changed))
		for k := range changed {
			minors = append(minors, k)
		}
		sortStrings(minors)
		b, _ := json.Marshal(minors)
		fmt.Println(string(b))
		return
	}
	for _, c := range changes {
		fmt.Printf("%s\n", c)
	}
	if *check {
		os.Exit(1)
	}
	if err := applyChanges(changed); err != nil {
		fatalf("%v", err)
	}
}

// latestStable returns, per major.minor, the highest stable patch on php-src.
func latestStable() (map[string]int, error) {
	out := map[string]int{}
	page := 1
	for {
		var tags []struct {
			Name string `json:"name"`
		}
		if err := githubJSON(fmt.Sprintf("https://api.github.com/repos/php/php-src/tags?per_page=100&page=%d", page), &tags); err != nil {
			return nil, err
		}
		if len(tags) == 0 {
			break
		}
		for _, t := range tags {
			m := stableTag.FindStringSubmatch(t.Name)
			if m == nil {
				continue
			}
			major, _ := strconv.Atoi(m[1])
			minor, _ := strconv.Atoi(m[2])
			patch, _ := strconv.Atoi(m[3])
			key := fmt.Sprintf("%d.%d", major, minor)
			if patch > out[key] {
				out[key] = patch
			}
		}
		if len(tags) < 100 {
			break
		}
		page++
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no stable tags found")
	}
	return out, nil
}

func githubJSON(url string, v any) error {
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	if tok := os.Getenv("GITHUB_TOKEN"); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(v)
}

// readRegistryState parses PatchEnd out of the PHP minor-range table.
func readRegistryState() (map[string]int, error) {
	b, err := os.ReadFile(registryFile)
	if err != nil {
		return nil, err
	}
	re := regexp.MustCompile(`\{Minor: (\d+), PatchEnd: (\d+)\}`)
	// Only the major-8 block is relevant; find it via the BuildMinorRanges(8, …) call.
	idx := strings.Index(string(b), "vsn.BuildMinorRanges(8, []vsn.MinorRange{")
	if idx < 0 {
		return nil, fmt.Errorf("major-8 range table not found in %s", registryFile)
	}
	rest := string(b[idx:])
	end := strings.Index(rest, "}),")
	out := map[string]int{}
	for _, m := range re.FindAllStringSubmatch(rest[:end], -1) {
		minor, _ := strconv.Atoi(m[1])
		patch, _ := strconv.Atoi(m[2])
		out[fmt.Sprintf("8.%d", minor)] = patch
	}
	return out, nil
}

// changed returns only the upstream entries that the registry lacks or lags
// behind, so -apply touches (and downloads checksums for) just those.
func changed(state, upstream map[string]int) map[string]int {
	out := map[string]int{}
	for key, up := range upstream {
		if !strings.HasPrefix(key, "8.") {
			continue
		}
		if have, ok := state[key]; !ok || up > have {
			out[key] = up
		}
	}
	return out
}

func diff(state, upstream map[string]int) []string {
	var out []string
	for key, up := range upstream {
		// Only track majors the registry already carries (4.x–8.x); never
		// invent a new major on the fly.
		if !strings.HasPrefix(key, "8.") {
			continue
		}
		have, ok := state[key]
		if !ok {
			out = append(out, fmt.Sprintf("NEW MINOR  %s.0 (registry has none)", key))
		} else if up > have {
			out = append(out, fmt.Sprintf("PATCH BUMP %s: %d -> %d", key, have, up))
		}
	}
	sortStrings(out)
	return out
}

// applyChanges rewrites the registry, test and CI matrix. Only `changed`
// entries get new checksums (each requires downloading its tarball); the
// range table and matrix are reconciled against the full upstream picture.
func applyChanges(changed map[string]int) error {
	if err := updateRangeTable(changed); err != nil {
		return err
	}
	if err := updateChecksums(changed); err != nil {
		return err
	}
	if err := updateTestCount(); err != nil {
		return err
	}
	if err := updateWorkflowMatrix(changed); err != nil {
		return err
	}
	// gofmt-align the edited Go files so the committed tree is canonical.
	for _, f := range []string{registryFile, testFile} {
		b, err := os.ReadFile(f)
		if err != nil {
			return err
		}
		out, err := format.Source(b)
		if err != nil {
			return fmt.Errorf("%s does not parse after edit: %w", f, err)
		}
		if err := os.WriteFile(f, out, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func updateRangeTable(upstream map[string]int) error {
	b, err := os.ReadFile(registryFile)
	if err != nil {
		return err
	}
	s := string(b)
	start := strings.Index(s, "vsn.BuildMinorRanges(8, []vsn.MinorRange{")
	if start < 0 {
		return fmt.Errorf("major-8 range table not found")
	}
	bodyStart := start + len("vsn.BuildMinorRanges(8, []vsn.MinorRange{")
	end := strings.Index(s[bodyStart:], "}),")
	table := s[bodyStart : bodyStart+end]
	// The slice runs up to the closing "})," and may carry the indentation
	// newline that precedes it; normalise so rows are appended cleanly.
	closeIndent := ""
	if nl := strings.LastIndex(table, "\n"); nl >= 0 && strings.TrimSpace(table[nl:]) == "" {
		closeIndent = table[nl:]
		table = table[:nl]
	}

	// Rewrite existing rows in place; append new minors at the end.
	rowRe := regexp.MustCompile(`(\{Minor: (\d+), PatchEnd: )(\d+)(\},)`)
	var maxMinor int
	newTable := rowRe.ReplaceAllStringFunc(table, func(row string) string {
		m := rowRe.FindStringSubmatch(row)
		minor, _ := strconv.Atoi(m[2])
		if minor > maxMinor {
			maxMinor = minor
		}
		if up, ok := upstream[fmt.Sprintf("8.%d", minor)]; ok && up >= 0 {
			return m[1] + strconv.Itoa(up) + m[4]
		}
		return row
	})
	for key, up := range upstream {
		minor, _ := strconv.Atoi(strings.TrimPrefix(key, "8."))
		if !strings.Contains(key, "8.") || strings.Contains(newTable, fmt.Sprintf("{Minor: %d,", minor)) {
			continue
		}
		if minor > maxMinor {
			newTable += "\n\t\t\t\t\t" + fmt.Sprintf("{Minor: %d, PatchEnd: %d},", minor, up)
		}
	}
	out := s[:bodyStart] + newTable + closeIndent + s[bodyStart+end:]
	return os.WriteFile(registryFile, []byte(out), 0o644)
}

func updateChecksums(changed map[string]int) error {
	b, err := os.ReadFile(registryFile)
	if err != nil {
		return err
	}
	s := string(b)
	for key, up := range changed {
		v := key + "." + strconv.Itoa(up)
		if strings.Contains(s, `Version: "`+v+`"`) {
			continue
		}
		sum, err := sha256OfTarball(v)
		if err != nil {
			return fmt.Errorf("checksum for %s: %w", v, err)
		}
		entry := fmt.Sprintf("\t\t\t\t{\n\t\t\t\t\tVersion: \"%s\",\n\t\t\t\t\tType:    \"sha256\",\n\t\t\t\t\tValue:   \"%s\",\n\t\t\t\t},\n", v, sum)
		marker := "\t\t\tChecksums: []Checksum{\n"
		i := strings.Index(s, marker)
		if i < 0 {
			return fmt.Errorf("Checksums block not found")
		}
		i += len(marker)
		s = s[:i] + entry + s[i:]
	}
	return os.WriteFile(registryFile, []byte(s), 0o644)
}

func sha256OfTarball(v string) (string, error) {
	resp, err := http.Get(fmt.Sprintf(distributions, v))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GET %s: %s", resp.Request.URL, resp.Status)
	}
	h := sha256.New()
	if _, err := io.Copy(h, resp.Body); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// updateTestCount syncs registry_test.go with the registry: the count
// assertion and the expected sha256 map must list every checksummed version,
// otherwise the test fails in CI after a bump.
func updateTestCount() error {
	reg, err := os.ReadFile(registryFile)
	if err != nil {
		return err
	}
	entryRe := regexp.MustCompile(`Version: "(8\.[0-9]+\.[0-9]+)",\s*Type:\s*"sha256",\s*Value:\s*"([0-9a-f]{64})",`)
	// Keep the registry's own order so a bump's diff is one added line per
	// new version rather than a reshuffle of existing entries.
	var entries []string
	for _, m := range entryRe.FindAllStringSubmatch(string(reg), -1) {
		entries = append(entries, fmt.Sprintf("\t\t%q: %q,", m[1], m[2]))
	}
	if len(entries) == 0 {
		return fmt.Errorf("no php checksums found in %s", registryFile)
	}

	b, err := os.ReadFile(testFile)
	if err != nil {
		return err
	}
	s := string(b)
	countRe := regexp.MustCompile(`len\(onlyWithChecksum\) != (\d+)`)
	if !countRe.MatchString(s) {
		return fmt.Errorf("checksum count assertion not found in %s", testFile)
	}
	s = countRe.ReplaceAllString(s, fmt.Sprintf("len(onlyWithChecksum) != %d", len(entries)))

	// Scope to the PHP checksum test: other tests (curl) also have a
	// wantChecksums map and must not be touched.
	fn := strings.Index(s, "func TestRegistryRepository_List_PHP_ChecksumFilter(")
	if fn < 0 {
		return fmt.Errorf("PHP checksum filter test not found in %s", testFile)
	}
	const open = "wantChecksums := map[string]string{\n"
	i := strings.Index(s[fn:], open)
	if i < 0 {
		return fmt.Errorf("wantChecksums map not found in %s", testFile)
	}
	i += fn + len(open)
	j := strings.Index(s[i:], "\n\t}")
	if j < 0 {
		return fmt.Errorf("wantChecksums map not closed in %s", testFile)
	}
	s = s[:i] + strings.Join(entries, "\n") + s[i+j:]
	return os.WriteFile(testFile, []byte(s), 0o644)
}

func updateWorkflowMatrix(upstream map[string]int) error {
	b, err := os.ReadFile(workflowFile)
	if err != nil {
		return err
	}
	maxMinor := -1
	for key := range upstream {
		m, _ := strconv.Atoi(strings.TrimPrefix(key, "8."))
		if m > maxMinor {
			maxMinor = m
		}
	}
	if maxMinor < 0 {
		return nil
	}
	s := string(b)
	// Each matrix lists minors as `- "8.N"` rows; add the newest minor after
	// the previous newest one in every matrix that already carries it.
	prev := fmt.Sprintf("          - \"8.%d\"\n", maxMinor-1)
	next := fmt.Sprintf("          - \"8.%d\"\n", maxMinor)
	if !strings.Contains(s, next) {
		s = strings.ReplaceAll(s, prev, prev+next)
	}
	return os.WriteFile(workflowFile, []byte(s), 0o644)
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "bump: "+format+"\n", args...)
	os.Exit(1)
}
