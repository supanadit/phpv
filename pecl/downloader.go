package pecl

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type peclPackage struct {
	Name    string
	Version string
	URL     string
}

type Downloader interface {
	ResolveLatest(name string) (peclPackage, error)
	ResolveCompatible(name, phpVersion string) (peclPackage, error)
	Download(pkg peclPackage, destDir string) (string, error)
}

type peclDownloader struct {
	client *http.Client
	// base defaults to https://pecl.php.net; tests override it.
	base string
}

const peclDefaultBase = "https://pecl.php.net"

func newPECLDownloader() *peclDownloader {
	return &peclDownloader{client: http.DefaultClient, base: peclDefaultBase}
}

type peclRestResponse struct {
	Version string `xml:"v"`
}

func (d *peclDownloader) ResolveLatest(name string) (peclPackage, error) {
	url := fmt.Sprintf("%s/rest/r/%s/latest.xml", d.base, name)
	resp, err := d.client.Get(url)
	if err != nil {
		return peclPackage{}, fmt.Errorf("pecl.net resolve %s: %w", name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return peclPackage{}, fmt.Errorf("pecl.net resolve %s: HTTP %d", name, resp.StatusCode)
	}

	var r peclRestResponse
	if err := xml.NewDecoder(resp.Body).Decode(&r); err != nil {
		return peclPackage{}, fmt.Errorf("pecl.net parse response for %s: %w", name, err)
	}
	if r.Version == "" {
		return peclPackage{}, fmt.Errorf("pecl.net: no version found for %s", name)
	}

	return peclPackage{
		Name:    name,
		Version: r.Version,
		URL:     fmt.Sprintf("https://pecl.php.net/get/%s-%s.tgz", name, r.Version),
	}, nil
}

// allreleasesResponse models pecl.php.net/rest/r/<name>/allreleases.xml.
// Note: <v> and <s> carry element text, not attributes:
//
//	<r><v>3.5.3</v><s>stable</s></r>
type allreleasesResponse struct {
	Releases []struct {
		Version string `xml:"v"`
		Stable  string `xml:"s"`
	} `xml:"r"`
}

// depsConstraint is the PHP version range from a release's deps endpoint,
// e.g. xdebug 3.3.2 requires min 8.0.0 / max 8.3.99.
type depsConstraint struct {
	Min string
	Max string
}

// parseDepsPHP parses the PHP-serialized dependency payload published at
// pecl.php.net/rest/r/<name>/deps.<version>.txt, e.g.:
//
//	a:1:{s:8:"required";a:2:{s:3:"php";a:2:{s:3:"min";s:5:"8.0.0";s:3:"max";s:6:"8.3.99";}s:13:"pearinstaller";a:1:{s:3:"min";s:5:"1.9.1";}}}
//
// Only the required.php min/max pair is needed, so a targeted scan is used
// instead of a general PHP unserializer.
func parseDepsPHP(body string) (depsConstraint, error) {
	idx := strings.Index(body, `"php"`)
	if idx < 0 {
		return depsConstraint{}, fmt.Errorf("no php dependency in payload")
	}
	rest := body[idx:]
	var c depsConstraint
	if m := serializedStringAfter(rest, `"min"`); m != "" {
		c.Min = m
	}
	if m := serializedStringAfter(rest, `"max"`); m != "" {
		c.Max = m
	}
	if c.Min == "" && c.Max == "" {
		return depsConstraint{}, fmt.Errorf("no min/max found for php dependency")
	}
	return c, nil
}

// serializedStringAfter finds key (e.g. `"min"`) in a PHP-serialized
// payload and returns the string value that follows it, as in
// s:3:"min";s:5:"8.0.0";
func serializedStringAfter(s, key string) string {
	i := strings.Index(s, key)
	if i < 0 {
		return ""
	}
	tail := s[i+len(key):]
	// Expect: ;s:<len>:"<value>";
	semi := strings.Index(tail, `s:`)
	if semi < 0 {
		return ""
	}
	tail = tail[semi+2:]
	colon := strings.Index(tail, `:"`)
	if colon < 0 {
		return ""
	}
	tail = tail[colon+2:]
	end := strings.Index(tail, `"`)
	if end < 0 {
		return ""
	}
	return tail[:end]
}

// ResolveCompatible walks PECL releases newest-first (stable only) and
// returns the first release whose PHP constraint admits the target PHP
// version (e.g. xdebug 3.x for PHP 8, xdebug 2.x for PHP 7.4).
func (d *peclDownloader) ResolveCompatible(name, phpVersion string) (peclPackage, error) {
	url := fmt.Sprintf("%s/rest/r/%s/allreleases.xml", d.base, name)
	resp, err := d.client.Get(url)
	if err != nil {
		return peclPackage{}, fmt.Errorf("pecl.net allreleases %s: %w", name, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return peclPackage{}, fmt.Errorf("pecl.net allreleases %s: HTTP %d", name, resp.StatusCode)
	}

	var all allreleasesResponse
	if err := xml.NewDecoder(resp.Body).Decode(&all); err != nil {
		return peclPackage{}, fmt.Errorf("pecl.net parse allreleases for %s: %w", name, err)
	}

	var tried []string
	for _, rel := range all.Releases {
		if !strings.EqualFold(strings.TrimSpace(rel.Stable), "stable") {
			continue // pre-release: skip deterministically
		}
		pkg := peclPackage{
			Name:    name,
			Version: rel.Version,
			URL:     fmt.Sprintf("%s/get/%s-%s.tgz", d.base, name, rel.Version),
		}
		ok, err := d.releaseSupportsPHP(name, rel.Version, phpVersion)
		if err != nil {
			// A release we cannot inspect (network hiccup, unexpected
			// payload) is skipped, not fatal: the next-older release may
			// be the right one.
			tried = append(tried, fmt.Sprintf("%s (check failed: %v)", rel.Version, err))
			continue
		}
		if ok {
			return pkg, nil
		}
		tried = append(tried, rel.Version)
	}
	if len(tried) == 0 {
		return peclPackage{}, fmt.Errorf("no stable releases of %s found on pecl.php.net", name)
	}
	return peclPackage{}, fmt.Errorf("no stable release of %s supports PHP %s (tried: %s)", name, phpVersion, strings.Join(tried, ", "))
}

// releaseSupportsPHP fetches the release's deps.<ver>.txt and reports
// whether its PHP constraint admits phpVersion.
func (d *peclDownloader) releaseSupportsPHP(name, releaseVersion, phpVersion string) (bool, error) {
	resp, err := d.client.Get(fmt.Sprintf("%s/rest/r/%s/deps.%s.txt", d.base, name, releaseVersion))
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}
	c, err := parseDepsPHP(string(body))
	if err != nil {
		return false, err
	}
	return versionInRange(phpVersion, c.Min, c.Max), nil
}

// versionInRange reports whether v satisfies [min, max] with empty bounds
// meaning unbounded. Comparison is numeric segment-wise. PECL phrases
// inclusive maxima as X.Y.99, which numeric comparison handles directly.
func versionInRange(v, min, max string) bool {
	if min != "" && compareVersions(v, min) < 0 {
		return false
	}
	if max != "" && compareVersions(v, max) > 0 {
		return false
	}
	return true
}

// compareVersions compares two dotted version strings segment by segment,
// numerically on each segment. Non-numeric suffixes (RC, beta, ...) compare
// as their longest numeric prefix.
func compareVersions(a, b string) int {
	aSegs := strings.Split(strings.Split(a, "-")[0], ".")
	bSegs := strings.Split(strings.Split(b, "-")[0], ".")
	n := len(aSegs)
	if len(bSegs) > n {
		n = len(bSegs)
	}
	for i := 0; i < n; i++ {
		var av, bv int
		if i < len(aSegs) {
			av, _ = strconv.Atoi(aSegs[i])
		}
		if i < len(bSegs) {
			bv, _ = strconv.Atoi(bSegs[i])
		}
		if av != bv {
			if av < bv {
				return -1
			}
			return 1
		}
	}
	return 0
}

func (d *peclDownloader) Download(pkg peclPackage, destDir string) (string, error) {
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", fmt.Errorf("create pecl cache dir: %w", err)
	}

	destPath := filepath.Join(destDir, pkg.Name+"-"+pkg.Version+".tgz")

	if _, err := os.Stat(destPath); err == nil {
		return destPath, nil
	}

	resp, err := d.client.Get(pkg.URL)
	if err != nil {
		return "", fmt.Errorf("download %s: %w", pkg.URL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download %s: HTTP %d", pkg.URL, resp.StatusCode)
	}

	f, err := os.Create(destPath)
	if err != nil {
		return "", fmt.Errorf("create %s: %w", destPath, err)
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		os.Remove(destPath)
		return "", fmt.Errorf("write %s: %w", destPath, err)
	}

	return destPath, nil
}
