package system

import (
	"bufio"
	"os"
	"strings"
)

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
