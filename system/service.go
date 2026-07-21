package system

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) Check(phpDeps []string) (*CheckResult, error) {
	distro, err := Detect()
	if err != nil {
		return nil, fmt.Errorf("detect distro: %w", err)
	}

	pkgs := packagesForDistro(distro.Name)
	if pkgs == nil {
		return &CheckResult{Distro: distro}, nil
	}

	result := &CheckResult{Distro: distro}
	for _, dep := range phpDeps {
		sysName, ok := pkgs[dep]
		if !ok {
			continue
		}
		installed := isInstalled(distro.PM, sysName)
		p := Package{
			Name:       dep,
			SystemName: sysName,
			Installed:  installed,
		}
		if installed {
			p.Version = getVersion(distro.PM, sysName)
		}
		if installed {
			result.Available = append(result.Available, p)
		} else {
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
		result.Available = append(result.Available, Package{
			Name:       name,
			SystemName: sysName,
			Installed:  installed,
		})
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
