package system

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/supanadit/phpv/domain"
)

type Service struct{}

func NewService() *Service {
	return &Service{}
}

// Check reports which of the given system packages are installed. Dependencies
// the distro maps to a different package name are resolved through the distro
// table; those with no mapping are skipped. The Required flag and any other
// caller-supplied field is preserved on the returned packages.
func (s *Service) Check(deps []Package) (*CheckResult, error) {
	distro, err := Detect()
	if err != nil {
		return nil, fmt.Errorf("detect distro: %w", err)
	}

	pkgs := packagesForDistro(distro.Name)
	if pkgs == nil {
		return &CheckResult{Distro: distro}, nil
	}

	result := &CheckResult{Distro: distro}
	for _, dep := range deps {
		sysName, ok := pkgs[dep.Name]
		if !ok {
			continue
		}
		installed := isInstalled(distro.PM, sysName)
		p := dep
		p.SystemName = sysName
		p.Installed = installed
		if installed {
			p.Version = getVersion(distro.PM, sysName)
			result.Available = append(result.Available, p)
		} else {
			p.Version = ""
			result.Missing = append(result.Missing, p)
		}
	}
	return result, nil
}

func (s *Service) CheckBuildTools(toolNames []string) (*CheckResult, error) {
	distro, err := Detect()
	if err != nil {
		return nil, fmt.Errorf("detect distro: %w", err)
	}

	pkgs := buildToolsForDistro(distro.Name)
	result := &CheckResult{Distro: distro}

	for _, name := range toolNames {
		sysName, ok := pkgs[name]
		if !ok {
			sysName = name
		}
		installed := isInstalled(distro.PM, sysName)
		p := Package{
			Name:       name,
			SystemName: sysName,
			Installed:  installed,
		}
		if installed {
			result.Available = append(result.Available, p)
		} else {
			result.Missing = append(result.Missing, p)
		}
	}
	return result, nil
}

func (s *Service) Install(packages []Package) error {
	if len(packages) == 0 {
		return nil
	}
	distro, err := Detect()
	if err != nil {
		return err
	}

	var names []string
	for _, p := range packages {
		names = append(names, p.SystemName)
	}

	args := installArgs(distro.PM, names)
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (s *Service) InstallCommand(packages []Package) string {
	if len(packages) == 0 {
		return ""
	}
	distro, _ := Detect()
	var names []string
	for _, p := range packages {
		names = append(names, p.SystemName)
	}
	args := installArgs(distro.PM, names)
	return strings.Join(args, " ")
}

func (s *Service) DistroInfo() Distro {
	d, _ := Detect()
	return d
}

func isInstalled(pm, name string) bool {
	var args []string
	switch pm {
	case "apt":
		args = []string{"dpkg", "-s", name}
	case "dnf":
		args = []string{"rpm", "-q", "--whatprovides", name}
	case "apk":
		args = []string{"apk", "info", "-e", name}
	case "pacman":
		args = []string{"pacman", "-Q", name}
	default:
		return false
	}
	cmd := exec.Command(args[0], args[1:]...)
	return cmd.Run() == nil
}

// getVersion returns the installed version of a system package.
// The returned format is normalized to "major.minor.patch" where possible,
// so it can be used with repository.MatchVersionRange.
func getVersion(pm, name string) string {
	switch pm {
	case "apt":
		// dpkg -s libssl-dev | grep ^Version: | awk '{print $2}'
		cmd := exec.Command("dpkg-query", "-W", "-f=${Version}", name)
		out, err := cmd.Output()
		if err != nil {
			return ""
		}
		return normalizeVersion(strings.TrimSpace(string(out)))
	case "dnf":
		// rpm -q --queryformat '%{VERSION}' openssl-devel
		cmd := exec.Command("rpm", "-q", "--queryformat", "%{VERSION}", name)
		out, err := cmd.Output()
		if err != nil {
			return ""
		}
		return normalizeVersion(strings.TrimSpace(string(out)))
	case "apk":
		// apk info -v openssl-dev | sed 's/^openssl-dev-//'
		cmd := exec.Command("apk", "info", "-v", name)
		out, err := cmd.Output()
		if err != nil {
			return ""
		}
		line := strings.TrimSpace(string(out))
		// apk info -v outputs: "package-name-version\n"
		if idx := strings.Index(line, "\n"); idx != -1 {
			line = line[:idx]
		}
		// Strip package name prefix (e.g. "openssl-dev-3.1.4-r0" -> "3.1.4-r0")
		if strings.HasPrefix(line, name+"-") {
			line = strings.TrimPrefix(line, name+"-")
		}
		return normalizeVersion(line)
	case "pacman":
		// pacman -Q openssl | awk '{print $2}'  -> "3.6.3-1"
		cmd := exec.Command("pacman", "-Q", name)
		out, err := cmd.Output()
		if err != nil {
			return ""
		}
		fields := strings.Fields(strings.TrimSpace(string(out)))
		if len(fields) >= 2 {
			return normalizeVersion(fields[1])
		}
		return ""
	default:
		return ""
	}
}

// normalizeVersion strips trailing distro-specific suffixes (e.g. "-1", "-r0",
// "+deb12u1", "ubuntu1", ".el9") and returns a semver-ish string that
// repository.MatchVersionRange can parse.
func normalizeVersion(v string) string {
	if v == "" {
		return ""
	}
	// Strip leading/trailing whitespace.
	v = strings.TrimSpace(v)
	// Remove epoch prefix (e.g. Arch "1:1.3.2" -> "1.3.2").
	if idx := strings.Index(v, ":"); idx != -1 {
		v = v[idx+1:]
	}
	// Keep only the leading digits-and-dots portion. Everything from the first
	// non-digit, non-dot character onward is a distro package release suffix.
	var end int
	for end = 0; end < len(v); end++ {
		c := v[end]
		if (c >= '0' && c <= '9') || c == '.' {
			continue
		}
		break
	}
	return v[:end]
}

func installArgs(pm string, names []string) []string {
	if os.Geteuid() == 0 {
		switch pm {
		case "apt":
			return append([]string{"apt-get", "install", "-y"}, names...)
		case "dnf":
			return append([]string{"dnf", "install", "-y"}, names...)
		case "apk":
			return append([]string{"apk", "add"}, names...)
		case "pacman":
			return append([]string{"pacman", "-S", "--noconfirm"}, names...)
		default:
			return nil
		}
	}
	switch pm {
	case "apt":
		return append([]string{"sudo", "apt-get", "install", "-y"}, names...)
	case "dnf":
		return append([]string{"sudo", "dnf", "install", "-y"}, names...)
	case "apk":
		return append([]string{"sudo", "apk", "add"}, names...)
	case "pacman":
		return append([]string{"sudo", "pacman", "-S", "--noconfirm"}, names...)
	default:
		return nil
	}
}

// baseAdvisoryLibraries are development libraries PHP can use at build time but
// which are not modelled as buildable dependencies: they are expected to come
// from the system, and no version constraint is checked for them. Libraries an
// extension asks for arrive through ExtensionDef.RequiresPackages instead.
var baseAdvisoryLibraries = []string{"sqlite3", "readline", "pcre2", "sodium"}

// DependencySet returns the system packages to check for a build of the given
// extensions. It is the single source of truth shared by `phpv install`,
// `phpv extension add` and `phpv doctor`, so the three cannot disagree about
// what a build needs.
//
// A package is marked Required when an extension names it in RequiresPackage,
// because that field also feeds the source-build dependency resolver: the
// package really is part of the build plan and its version is checked before
// use. Packages from RequiresPackages and baseAdvisoryLibraries are advisory —
// dev libraries the build can consume but whose version is not gated.
//
// Only packages the build can actually consume are listed: a library with no
// buildable package definition and no extension asking for it is not included.
// DependencySet exposes the dependency derivation as a service method.
func (s *Service) DependencySet(exts []domain.ExtensionDef) []Package {
	return DependencySet(exts)
}

func DependencySet(exts []domain.ExtensionDef) []Package {
	seen := make(map[string]bool)
	var pkgs []Package

	add := func(name string, required bool) {
		if name == "" || seen[name] {
			return
		}
		seen[name] = true
		pkgs = append(pkgs, Package{Name: name, SystemName: name, Required: required})
	}

	for _, def := range exts {
		add(def.RequiresPackage, true)
	}
	for _, def := range exts {
		for _, name := range def.RequiresPackages {
			add(name, false)
		}
	}
	for _, name := range baseAdvisoryLibraries {
		add(name, false)
	}
	return pkgs
}

// Names returns the dependency names of pkgs, in order. It is the form
// Service.Check expects.
func Names(pkgs []Package) []string {
	names := make([]string, 0, len(pkgs))
	for _, p := range pkgs {
		names = append(names, p.Name)
	}
	return names
}

func Detect() (Distro, error) {
	f, err := os.Open("/etc/os-release")
	if err != nil {
		return Distro{Name: "unknown", PM: "unknown"}, nil
	}
	defer f.Close()

	var d Distro
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "ID=") {
			d.Name = strings.Trim(strings.TrimPrefix(line, "ID="), `"`)
		}
		if strings.HasPrefix(line, "VERSION_ID=") {
			d.Version = strings.Trim(strings.TrimPrefix(line, "VERSION_ID="), `"`)
		}
	}

	switch d.Name {
	case "ubuntu", "debian":
		d.PM = "apt"
	case "fedora", "rhel", "centos":
		d.PM = "dnf"
	case "alpine":
		d.PM = "apk"
	case "arch":
		d.PM = "pacman"
	default:
		d.PM = "unknown"
	}

	return d, nil
}

type pkgMap map[string]string

var fedoraPackages = pkgMap{
	"openssl":   "openssl-devel",
	"libxml2":   "libxml2-devel",
	"zlib":      "zlib-devel",
	"oniguruma": "oniguruma-devel",
	"curl":      "libcurl-devel",
	"sqlite3":   "sqlite-devel",
	"readline":  "readline-devel",
	"icu":       "libicu-devel",
	"pcre2":     "pcre2-devel",
	"argon2":    "libargon2-devel",
	"sodium":    "libsodium-devel",
	"libpng":    "libpng-devel",
	"libjpeg":   "libjpeg-turbo-devel",
	"freetype":  "freetype-devel",
	"libzip":    "libzip-devel",
	"libpq":     "postgresql-devel",
}

var ubuntuPackages = pkgMap{
	"openssl":   "libssl-dev",
	"libxml2":   "libxml2-dev",
	"zlib":      "zlib1g-dev",
	"oniguruma": "libonig-dev",
	"curl":      "libcurl4-openssl-dev",
	"sqlite3":   "libsqlite3-dev",
	"readline":  "libreadline-dev",
	"icu":       "libicu-dev",
	"pcre2":     "libpcre2-dev",
	"argon2":    "libargon2-dev",
	"sodium":    "libsodium-dev",
	"libpng":    "libpng-dev",
	"libjpeg":   "libjpeg-dev",
	"freetype":  "libfreetype-dev",
	"libzip":    "libzip-dev",
	"libpq":     "libpq-dev",
}

var alpinePackages = pkgMap{
	"openssl":   "openssl-dev",
	"libxml2":   "libxml2-dev",
	"zlib":      "zlib-dev",
	"oniguruma": "oniguruma-dev",
	"curl":      "curl-dev",
	"sqlite3":   "sqlite-dev",
	"readline":  "readline-dev",
	"icu":       "icu-dev",
	"pcre2":     "pcre2-dev",
	"argon2":    "argon2-dev",
	"sodium":    "libsodium-dev",
	"libpng":    "libpng-dev",
	"libjpeg":   "libjpeg-dev",
	"freetype":  "freetype-dev",
	"libzip":    "libzip-dev",
	"libpq":     "libpq-dev",
}

var archPackages = pkgMap{
	"openssl":   "openssl",
	"libxml2":   "libxml2",
	"zlib":      "zlib",
	"oniguruma": "oniguruma",
	"curl":      "curl",
	"sqlite3":   "sqlite",
	"readline":  "readline",
	"icu":       "icu",
	"pcre2":     "pcre2",
	"argon2":    "argon2",
	"sodium":    "libsodium",
	"libpng":    "libpng",
	"libjpeg":   "libjpeg-turbo",
	"freetype":  "freetype2",
	"libzip":    "libzip",
	"libpq":     "postgresql-libs",
}

var fedoraBuildTools = pkgMap{
	"gcc":        "gcc",
	"g++":        "gcc-c++",
	"make":       "make",
	"cmake":      "cmake",
	"autoconf":   "autoconf",
	"automake":   "automake",
	"m4":         "m4",
	"perl":       "perl",
	"bison":      "bison",
	"re2c":       "re2c",
	"flex":       "flex",
	"pkg-config": "pkgconf-pkg-config",
	"xz":         "xz-devel",
}

var ubuntuBuildTools = pkgMap{
	"gcc":        "gcc",
	"g++":        "g++",
	"make":       "make",
	"cmake":      "cmake",
	"autoconf":   "autoconf",
	"automake":   "automake",
	"m4":         "m4",
	"perl":       "perl",
	"bison":      "bison",
	"re2c":       "re2c",
	"flex":       "flex",
	"pkg-config": "pkg-config",
	"xz":         "xz-utils",
}

var alpineBuildTools = pkgMap{
	"gcc":        "gcc",
	"g++":        "g++",
	"make":       "make",
	"cmake":      "cmake",
	"autoconf":   "autoconf",
	"automake":   "automake",
	"m4":         "m4",
	"perl":       "perl",
	"bison":      "bison",
	"re2c":       "re2c",
	"flex":       "flex",
	"pkg-config": "pkgconfig",
	"xz":         "xz",
}

var archBuildTools = pkgMap{
	"gcc":        "gcc",
	"g++":        "gcc",
	"make":       "make",
	"cmake":      "cmake",
	"autoconf":   "autoconf",
	"automake":   "automake",
	"m4":         "m4",
	"perl":       "perl",
	"bison":      "bison",
	"re2c":       "re2c",
	"flex":       "flex",
	"pkg-config": "pkg-config",
	"xz":         "xz",
}

func packagesForDistro(distro string) pkgMap {
	switch distro {
	case "fedora", "rhel", "centos":
		return fedoraPackages
	case "ubuntu", "debian":
		return ubuntuPackages
	case "alpine":
		return alpinePackages
	case "arch":
		return archPackages
	default:
		return nil
	}
}

func buildToolsForDistro(distro string) pkgMap {
	switch distro {
	case "fedora", "rhel", "centos":
		return fedoraBuildTools
	case "ubuntu", "debian":
		return ubuntuBuildTools
	case "alpine":
		return alpineBuildTools
	case "arch":
		return archBuildTools
	default:
		return nil
	}
}

// Aliases keep the system package's vocabulary while the canonical data lives
// in domain, so consumers can declare interfaces without importing this module.
type (
	Distro      = domain.Distro
	Package     = domain.SystemPackage
	CheckResult = domain.SystemCheckResult
)
