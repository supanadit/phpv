package pecl

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/silo"
)

type InstallResult struct {
	Name       string
	Version    string
	InstallDir string
}

type Service struct {
	silo       *silo.Service
	downloader Downloader
}

func NewService(siloSvc *silo.Service) *Service {
	return &Service{
		silo:       siloSvc,
		downloader: newPECLDownloader(),
	}
}

// Install installs from a package name (resolves the latest stable release,
// without PHP compatibility checks) or a local archive path.
func (s *Service) Install(ctx context.Context, source, phpVersion string, jobs int) (*InstallResult, error) {
	if isLocalArchive(source) {
		extName, extVersion, err := parseNameVersion(source)
		if err != nil {
			return nil, fmt.Errorf("parse archive name: %w", err)
		}
		return s.installArchive(ctx, source, extName, extVersion, phpVersion, jobs, nil)
	}

	pkg, err := s.downloader.ResolveLatest(source)
	if err != nil {
		return nil, fmt.Errorf("resolve pecl package %s: %w", source, err)
	}
	return s.installPackage(ctx, pkg, phpVersion, jobs, nil)
}

// InstallCompatible resolves the newest stable release of name that
// supports phpVersion (per its published PHP dependency range), then
// downloads, builds, and enables it. extraINI lines (key=value) are
// appended to php.ini after a successful install.
func (s *Service) InstallCompatible(ctx context.Context, name, phpVersion string, jobs int, extraINI []string) (*InstallResult, error) {
	pkg, err := s.downloader.ResolveCompatible(name, phpVersion)
	if err != nil {
		return nil, fmt.Errorf("resolve pecl package %s: %w", name, err)
	}
	return s.installPackage(ctx, pkg, phpVersion, jobs, extraINI)
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

func (s *Service) installPackage(ctx context.Context, pkg peclPackage, phpVersion string, jobs int, extraINI []string) (*InstallResult, error) {
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

	return s.installArchive(ctx, archivePath, pkg.Name, pkg.Version, phpVersion, jobs, extraINI)
}

func (s *Service) installArchive(ctx context.Context, archivePath, extName, extVersion, phpVersion string, jobs int, extraINI []string) (*InstallResult, error) {
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

	iniDir := filepath.Join(prefix, "etc")
	_ = iniDir // legacy fallback; the real path is resolved below
	iniPath := resolveIniPath(prefix, phpConfig)
	if err := os.MkdirAll(filepath.Dir(iniPath), 0755); err != nil {
		cleanup()
		return nil, fmt.Errorf("create ini dir: %w", err)
	}

	if _, err := enableExtension(phpBin, iniPath, extName, knownZendExtensions[extName]); err != nil {
		cleanup()
		return nil, err
	}
	for _, setting := range extraINI {
		if err := writeINIEntry(iniPath, setting); err != nil {
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

// knownZendExtensions lists extensions that must be loaded via
// zend_extension= rather than extension=. The table is only a hint —
// enableExtension verifies the module actually loads and flips the
// directive if needed, so unknown Zend extensions are handled too.
var knownZendExtensions = map[string]bool{
	"xdebug": true,
}

// iniEntryFor returns the php.ini load directive for an extension.
func iniEntryFor(extName string, zend bool) string {
	if zend {
		return "zend_extension=" + extName + ".so"
	}
	return "extension=" + extName + ".so"
}

// moduleLoads runs php -m and reports whether extName is listed and PHP
// started cleanly. stderr is returned for error reporting.
func moduleLoads(phpBin, extName string) (bool, string) {
	cmd := exec.Command(phpBin, "-m")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return false, stderr.String()
	}
	for _, mod := range strings.Split(string(out), "\n") {
		if strings.EqualFold(strings.TrimSpace(mod), extName) {
			return true, ""
		}
	}
	return false, stderr.String()
}

// writeINIEntry appends the line to php.ini unless it is already present.
func writeINIEntry(iniPath, line string) error {
	if data, err := os.ReadFile(iniPath); err == nil {
		for _, l := range strings.Split(string(data), "\n") {
			if strings.TrimSpace(l) == line {
				return nil
			}
		}
	}
	f, err := os.OpenFile(iniPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open php.ini: %w", err)
	}
	defer f.Close()
	if _, err := f.WriteString(line + "\n"); err != nil {
		return fmt.Errorf("write php.ini: %w", err)
	}
	return nil
}

// removeINIEntry strips an exact line from php.ini (best effort).
func removeINIEntry(iniPath, line string) {
	data, err := os.ReadFile(iniPath)
	if err != nil {
		return
	}
	var kept []string
	for _, l := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(l) != line {
			kept = append(kept, l)
		}
	}
	os.WriteFile(iniPath, []byte(strings.Join(kept, "\n")), 0644)
}

// resolveIniPath returns the php.ini path PHP actually loads: query
// php-config --ini-path (the directory) and append php.ini. Falls back to
// <prefix>/etc/php.ini when php-config fails or answers nothing. This
// mirrors the assembler's getPHPIniDir semantics; phpv-built PHP uses
// <prefix>/lib, NOT <prefix>/etc.
func resolveIniPath(prefix, phpConfigBin string) string {
	out, err := exec.Command(phpConfigBin, "--ini-path").Output()
	if err == nil {
		if dir := strings.TrimSpace(string(out)); dir != "" {
			return filepath.Join(dir, "php.ini")
		}
	}
	return filepath.Join(prefix, "etc", "php.ini")
}

// iniDiagnostic captures php --ini output so a "module does not load"
// failure can name the file PHP actually reads.
func iniDiagnostic(phpBin string) string {
	out, err := exec.Command(phpBin, "--ini").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// enableExtension writes the load directive for extName into iniPath and
// verifies the module actually loads via php -m. If the initial form
// (chosen from zendHint) does not load, it flips to the other form once.
// If neither loads, both entries are removed and an error carrying PHP's
// diagnostics is returned. Returns the entry that verified.
func enableExtension(phpBin, iniPath, extName string, zendHint bool) (string, error) {
	zend := zendHint
	entry := iniEntryFor(extName, zend)
	if err := writeINIEntry(iniPath, entry); err != nil {
		return "", err
	}
	if ok, phpErr := moduleLoads(phpBin, extName); !ok {
		removeINIEntry(iniPath, entry)
		alt := iniEntryFor(extName, !zend)
		if err := writeINIEntry(iniPath, alt); err != nil {
			return "", err
		}
		if ok, phpErr2 := moduleLoads(phpBin, extName); !ok {
			removeINIEntry(iniPath, alt)
			detail := phpErr2
			if detail == "" {
				detail = phpErr
			}
			detail = strings.TrimSpace(detail)
			if detail == "" {
				// PHP starts cleanly but never saw the directive — name the
				// ini file it actually reads so a path mismatch diagnoses
				// itself.
				detail = iniDiagnostic(phpBin)
			}
			return "", fmt.Errorf("extension %s built and installed but does not load (PHP reports: %s)", extName, detail)
		}
		return alt, nil
	}
	return entry, nil
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
		resolveIniPath(prefix, filepath.Join(prefix, "bin", "php-config")),
		filepath.Join(prefix, "etc", "php.ini"),
	} {
		removeINIEntry(iniPath, "extension="+name+".so")
		removeINIEntry(iniPath, "zend_extension="+name+".so")
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
	return strings.HasSuffix(path, ".tgz") ||
		strings.HasSuffix(path, ".tar.gz") ||
		strings.HasSuffix(path, ".tar.bz2")
}
