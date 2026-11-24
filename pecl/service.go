package pecl

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/supanadit/phpv/domain"
)

// PECLSilo is the storage capability pecl consumes, so app/main.go can
// inject the concrete silo service without a module import.
//
//go:generate mockery --name PECLSilo
type PECLSilo interface {
	PackagePrefix(name, version string) string
	PECLArchivePath(name, version string) string
	GetExtensionManifest(phpVersion string) (*domain.ExtensionManifest, error)
	SaveExtensionManifest(phpVersion string, manifest *domain.ExtensionManifest) error
}

// IniEditor is the php.ini editing capability pecl consumes.
//
//go:generate mockery --name IniEditor
type IniEditor interface {
	Resolve(prefix, phpConfigBin string) string
	Enable(phpBin, iniPath, extName string, zendHint bool) (string, error)
	Write(iniPath, line string) error
	Remove(iniPath, line string)
}

type InstallResult = domain.PECLInstallResult

// installOpts groups the shared install options for private helpers.
type installOpts struct {
	phpVersion string
	jobs       int
	extraINI   []string
}

// archiveSpec groups the inputs for extracting and enabling an archive.
type archiveSpec struct {
	archivePath, extName, extVersion, phpVersion string
	jobs                                         int
	extraINI                                     []string
}

type Service struct {
	silo       PECLSilo
	downloader Downloader
	ini        IniEditor
}

func NewService(siloSvc PECLSilo, ini IniEditor) *Service {
	return &Service{silo: siloSvc, downloader: newPECLDownloader(), ini: ini}
}

// Install installs from a package name (resolves the latest stable release,
// without PHP compatibility checks) or a local archive path.
func (s *Service) Install(ctx context.Context, source, phpVersion string, jobs int) (*InstallResult, error) {
	if isLocalArchive(source) {
		extName, extVersion, err := parseNameVersion(source)
		if err != nil {
			return nil, fmt.Errorf("parse archive name: %w", err)
		}
		return s.installArchive(ctx, archiveSpec{source, extName, extVersion, phpVersion, jobs, nil})
	}

	pkg, err := s.downloader.ResolveLatest(source)
	if err != nil {
		return nil, fmt.Errorf("resolve pecl package %s: %w", source, err)
	}
	return s.installPackage(ctx, pkg, installOpts{phpVersion, jobs, nil})
}

// InstallCompatible resolves the newest stable release of name that
// supports phpVersion (per its published PHP dependency range), then
// downloads, builds, and enables it. extraINI lines (key=value) are
// appended to php.ini after a successful install.
func (s *Service) InstallCompatible(ctx context.Context, req domain.PECLInstallRequest) (*InstallResult, error) {
	pkg, err := s.downloader.ResolveCompatible(req.Name, req.PHPVersion)
	if err != nil {
		return nil, fmt.Errorf("resolve pecl package %s: %w", req.Name, err)
	}
	return s.installPackage(ctx, pkg, installOpts{req.PHPVersion, req.Jobs, req.ExtraINI})
}

// ResolveCompatibleRelease returns just the version InstallCompatible
// would pick, for dry-run reporting.
func (s *Service) ResolveCompatibleRelease(name, phpVersion string) (string, error) {
	pkg, err := s.downloader.ResolveCompatible(name, phpVersion)
	if err != nil {
		return "", err
	}
	return pkg.Version, nil
}

func (s *Service) installPackage(ctx context.Context, pkg peclPackage, opts installOpts) (*InstallResult, error) {
	phpVersion, jobs, extraINI := opts.phpVersion, opts.jobs, opts.extraINI
	prefix := s.silo.PackagePrefix("php", phpVersion)
	phpBin := filepath.Join(prefix, "bin", "php")
	if _, err := os.Stat(phpBin); os.IsNotExist(err) {
		return nil, fmt.Errorf("PHP %s is not installed at %s", phpVersion, prefix)
	}

	cacheDir := filepath.Dir(s.silo.PECLArchivePath(pkg.Name, pkg.Version))
	archivePath, err := s.downloader.Download(pkg, cacheDir)
	if err != nil {
		return nil, fmt.Errorf("download pecl package %s: %w", pkg.Name, err)
	}

	return s.installArchive(ctx, archiveSpec{archivePath, pkg.Name, pkg.Version, phpVersion, jobs, extraINI})
}

func (s *Service) installArchive(ctx context.Context, spec archiveSpec) (*InstallResult, error) {
	archivePath, extName, extVersion, phpVersion, jobs, extraINI := spec.archivePath, spec.extName, spec.extVersion, spec.phpVersion, spec.jobs, spec.extraINI
	prefix := s.silo.PackagePrefix("php", phpVersion)
	phpBin := filepath.Join(prefix, "bin", "php")
	if _, err := os.Stat(phpBin); os.IsNotExist(err) {
		return nil, fmt.Errorf("PHP %s is not installed at %s", phpVersion, prefix)
	}

	manifest, err := s.silo.GetExtensionManifest(phpVersion)
	if err != nil {
		return nil, fmt.Errorf("get extension manifest: %w", err)
	}
	for _, e := range manifest.Extensions {
		if e.Name == extName {
			return nil, fmt.Errorf("extension %s is already installed for PHP %s", extName, phpVersion)
		}
	}

	extractDir := filepath.Join(prefix, "lib", "pecl", extName)
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		return nil, fmt.Errorf("create pecl extract dir: %w", err)
	}

	statePath := filepath.Join(extractDir, ".state")
	os.WriteFile(statePath, []byte("in_progress"), 0644)

	cleanup := func() {
		os.WriteFile(statePath, []byte("failed"), 0644)
		os.RemoveAll(extractDir)
	}

	if err := extractArchive(archivePath, extractDir); err != nil {
		cleanup()
		return nil, fmt.Errorf("extract pecl archive: %w", err)
	}

	sourceDir := findSourceDir(extractDir, extName)
	if sourceDir == "" {
		cleanup()
		return nil, fmt.Errorf("could not find extension source in %s (no config.m4 found)", extractDir)
	}

	phpize := filepath.Join(prefix, "bin", "phpize")
	phpConfig := filepath.Join(prefix, "bin", "php-config")

	cmd := exec.CommandContext(ctx, phpize)
	cmd.Dir = sourceDir
	if out, err := cmd.CombinedOutput(); err != nil {
		cleanup()
		return nil, fmt.Errorf("phpize %s: %w\n%s", extName, err, out)
	}

	configure := exec.CommandContext(ctx, "./configure", "--with-php-config="+phpConfig)
	configure.Dir = sourceDir
	if out, err := configure.CombinedOutput(); err != nil {
		cleanup()
		return nil, fmt.Errorf("configure %s: %w\n%s", extName, err, out)
	}

	make := exec.CommandContext(ctx, "make", fmt.Sprintf("-j%d", jobs))
	make.Dir = sourceDir
	if out, err := make.CombinedOutput(); err != nil {
		cleanup()
		return nil, fmt.Errorf("make %s: %w\n%s", extName, err, out)
	}

	install := exec.CommandContext(ctx, "make", "install")
	install.Dir = sourceDir
	if out, err := install.CombinedOutput(); err != nil {
		cleanup()
		return nil, fmt.Errorf("make install %s: %w\n%s", extName, err, out)
	}

	iniPath := s.ini.Resolve(prefix, phpConfig)
	if err := os.MkdirAll(filepath.Dir(iniPath), 0755); err != nil {
		cleanup()
		return nil, fmt.Errorf("create ini dir: %w", err)
	}

	if _, err := s.ini.Enable(phpBin, iniPath, extName, knownZendExtensions[extName]); err != nil {
		cleanup()
		return nil, err
	}
	for _, setting := range extraINI {
		if err := s.ini.Write(iniPath, setting); err != nil {
			cleanup()
			return nil, err
		}
	}

	os.WriteFile(statePath, []byte("installed"), 0644)

	manifest.Extensions = append(manifest.Extensions, domain.ExtensionState{
		Name:        extName,
		Type:        domain.ExtensionTypePECL,
		Version:     extVersion,
		InstalledAt: time.Now(),
	})
	if err := s.silo.SaveExtensionManifest(phpVersion, manifest); err != nil {
		return nil, fmt.Errorf("save extension manifest: %w", err)
	}

	return &InstallResult{
		Name:       extName,
		Version:    extVersion,
		InstallDir: extractDir,
	}, nil
}

// knownZendExtensions lists extensions loaded via zend_extension= rather than
// extension=. It is only a hint — the ini editor verifies the module loads
// and flips the directive if needed, so unknown Zend extensions work too.
var knownZendExtensions = map[string]bool{
	"xdebug": true,
}

func (s *Service) List(phpVersion string) ([]domain.ExtensionState, error) {
	manifest, err := s.silo.GetExtensionManifest(phpVersion)
	if err != nil {
		return nil, err
	}
	var peclExts []domain.ExtensionState
	for _, e := range manifest.Extensions {
		if e.Type == domain.ExtensionTypePECL {
			peclExts = append(peclExts, e)
		}
	}
	return peclExts, nil
}

func (s *Service) Uninstall(name, phpVersion string) error {
	manifest, err := s.silo.GetExtensionManifest(phpVersion)
	if err != nil {
		return fmt.Errorf("get extension manifest: %w", err)
	}

	found := false
	for _, e := range manifest.Extensions {
		if e.Name == name && e.Type == domain.ExtensionTypePECL {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("PECL extension %q is not installed for PHP %s", name, phpVersion)
	}

	prefix := s.silo.PackagePrefix("php", phpVersion)

	extDir := filepath.Join(prefix, "lib", "php", "extensions")
	removed := false
	filepath.Walk(extDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.Name() == name+".so" {
			os.Remove(path)
			removed = true
		}
		return nil
	})

	// Strip the load directive from every php.ini phpv could have written:
	// the resolved one (php-config) and the legacy <prefix>/etc location.
	for _, iniPath := range []string{
		s.ini.Resolve(prefix, filepath.Join(prefix, "bin", "php-config")),
		filepath.Join(prefix, "etc", "php.ini"),
	} {
		s.ini.Remove(iniPath, "extension="+name+".so")
		s.ini.Remove(iniPath, "zend_extension="+name+".so")
	}

	peclDir := filepath.Join(prefix, "lib", "pecl", name)
	os.RemoveAll(peclDir)

	var remaining []domain.ExtensionState
	for _, e := range manifest.Extensions {
		if e.Name != name {
			remaining = append(remaining, e)
		}
	}
	manifest.Extensions = remaining
	if err := s.silo.SaveExtensionManifest(phpVersion, manifest); err != nil {
		return fmt.Errorf("save extension manifest: %w", err)
	}

	if !removed {
		return fmt.Errorf("extension %q .so file not found at %s", name, extDir)
	}
	return nil
}

func isLocalArchive(path string) bool {
	return strings.HasSuffix(path, ".tgz") || strings.HasSuffix(path, ".tar.gz") || strings.HasSuffix(path, ".tar.bz2")
}

var peclNameVersionRegex = regexp.MustCompile(`([a-zA-Z0-9_-]+)-(\d+\.\d+\.\d+(?:[a-zA-Z0-9._-]*)?)`)

func parseNameVersion(archivePath string) (name, version string, err error) {
	base := filepath.Base(archivePath)
	base = strings.TrimSuffix(base, ".tar.gz")
	base = strings.TrimSuffix(base, ".tar.bz2")
	base = strings.TrimSuffix(base, ".tgz")

	matches := peclNameVersionRegex.FindStringSubmatch(base)
	if len(matches) >= 3 {
		return matches[1], matches[2], nil
	}

	parts := strings.Split(base, "-")
	if len(parts) >= 2 {
		return parts[0], parts[1], nil
	}

	return base, "unknown", nil
}

func extractArchive(archivePath, destDir string) error {
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("create extract dir: %w", err)
	}

	cmd := exec.Command("tar", "-xzf", archivePath, "-C", destDir)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("extract %s: %w\n%s", archivePath, err, out)
	}
	return nil
}

func findSourceDir(baseDir, extName string) string {
	hasConfigM4 := func(dir string) bool {
		_, err := os.Stat(filepath.Join(dir, "config.m4"))
		return err == nil
	}

	if hasConfigM4(baseDir) {
		return baseDir
	}

	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return ""
	}

	for _, entry := range entries {
		if entry.IsDir() && (entry.Name() == extName || strings.Contains(entry.Name(), extName)) {
			sub := filepath.Join(baseDir, entry.Name())
			if hasConfigM4(sub) {
				return sub
			}
			if nested := findSourceDir(sub, extName); nested != "" {
				return nested
			}
		}
	}

	for _, entry := range entries {
		if entry.IsDir() {
			sub := filepath.Join(baseDir, entry.Name())
			if hasConfigM4(sub) {
				return sub
			}
		}
	}

	return ""
}

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
