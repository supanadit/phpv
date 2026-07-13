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

func isInstalled(pm, name string) bool {
	var args []string
	switch pm {
	case "apt":
		args = []string{"dpkg", "-s", name}
	case "dnf":
		args = []string{"rpm", "-q", name}
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
