package system

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/supanadit/phpv/domain"
)

// SystemCatalog is the swappable source of distro package data and
// host-level package-manager mechanics. Memory, REST or SQL drivers can
// implement it without changing this module.
type SystemCatalog interface {
	Detect() (Distro, error)
	PackagesForDistro(distro string) map[string]string
	BuildToolsForDistro(distro string) map[string]string
	AdvisoryLibraries() []string
	InstallArgs(pm string, names []string) []string
}

type Service struct{ catalog SystemCatalog }

func NewService(catalog SystemCatalog) *Service {
	return &Service{catalog: catalog}
}

// Check reports which of the given system packages are installed. Dependencies
// the distro maps to a different package name are resolved through the distro
// table; those with no mapping are skipped. The Required flag and any other
// caller-supplied field is preserved on the returned packages.
func (s *Service) Check(deps []Package) (*CheckResult, error) {
	distro, err := s.catalog.Detect()
	if err != nil {
		return nil, fmt.Errorf("detect distro: %w", err)
	}

	pkgs := s.catalog.PackagesForDistro(distro.Name)
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
	distro, err := s.catalog.Detect()
	if err != nil {
		return nil, fmt.Errorf("detect distro: %w", err)
	}

	pkgs := s.catalog.BuildToolsForDistro(distro.Name)
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
	distro, err := s.catalog.Detect()
	if err != nil {
		return err
	}

	var names []string
	for _, p := range packages {
		names = append(names, p.SystemName)
	}

	args := s.catalog.InstallArgs(distro.PM, names)
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
	distro, _ := s.catalog.Detect()
	var names []string
	for _, p := range packages {
		names = append(names, p.SystemName)
	}
	args := s.catalog.InstallArgs(distro.PM, names)
	return strings.Join(args, " ")
}

func (s *Service) DistroInfo() Distro {
	d, _ := s.catalog.Detect()
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

// from the system, and no version constraint is checked for them. Libraries an

// DependencySet returns the system packages to check for a build of the given
// extensions. It is the single source of truth shared by `phpv install`,
// `phpv extension add` and `phpv doctor`, so the three cannot disagree about
// what a build needs.
//
// A package is marked Required when an extension names it in RequiresPackage,
// because that field also feeds the source-build dependency resolver: the
// package really is part of the build plan and its version is checked before
// use. Packages from RequiresPackages and s.catalog.AdvisoryLibraries() are advisory —
// dev libraries the build can consume but whose version is not gated.
//
// Only packages the build can actually consume are listed: a library with no
// buildable package definition and no extension asking for it is not included.
// DependencySet exposes the dependency derivation as a service method.
func (s *Service) DependencySet(exts []domain.ExtensionDef) []Package {
	return dependencySet(exts, s.catalog.AdvisoryLibraries())
}

// DependencySet is the standalone helper used when no catalog is injected.
func DependencySet(exts []domain.ExtensionDef) []Package {
	return dependencySet(exts, []string{"sqlite3", "readline", "pcre2", "sodium"})
}

func dependencySet(exts []domain.ExtensionDef, advisory []string) []Package {
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
	for _, name := range advisory {
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

// Aliases keep the system package's vocabulary while the canonical data lives
// in domain, so consumers can declare interfaces without importing this module.
type (
	Distro      = domain.Distro
	Package     = domain.SystemPackage
	CheckResult = domain.SystemCheckResult
)
