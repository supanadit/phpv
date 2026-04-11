package disk

import (
	"os"
	"os/exec"
	"strings"

	"github.com/supanadit/phpv/internal/utils"
)

type defaultExecutor struct{}

func (e *defaultExecutor) Which(cmd string) (string, error) {
	out, err := exec.Command("which", cmd).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func (e *defaultExecutor) PkgConfig(pkg string) (string, string, error) {
	out, err := exec.Command("pkg-config", "--libs", "--cflags", pkg).Output()
	if err != nil {
		return "", "", err
	}
	parts := strings.SplitN(strings.TrimSpace(string(out)), " ", 2)
	cflags := ""
	ldflags := ""
	if len(parts) >= 1 {
		ldflags = parts[0]
	}
	if len(parts) >= 2 {
		cflags = parts[1]
	}
	return cflags, ldflags, nil
}

func (e *defaultExecutor) PkgConfigModVersion(pkg string) (string, error) {
	out, err := exec.Command("pkg-config", "--modversion", pkg).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func (e *defaultExecutor) PkgConfigExists(pkg string) bool {
	env := os.Environ()
	pkgConfigPath := os.Getenv("PKG_CONFIG_PATH")
	standardPaths := utils.GetSystemPkgConfigPaths()
	for _, p := range standardPaths {
		if pkgConfigPath == "" {
			pkgConfigPath = p
		} else {
			pkgConfigPath = p + ":" + pkgConfigPath
		}
	}
	for i, v := range env {
		if strings.HasPrefix(v, "PKG_CONFIG_PATH=") {
			env[i] = "PKG_CONFIG_PATH=" + pkgConfigPath
			break
		}
	}
	if !strings.Contains(strings.Join(env, ""), "PKG_CONFIG_PATH") {
		env = append(env, "PKG_CONFIG_PATH="+pkgConfigPath)
	}
	cmd := exec.Command("pkg-config", "--exists", pkg)
	cmd.Env = env
	return cmd.Run() == nil
}

func (e *defaultExecutor) PathExists(path string) bool {
	_, err := exec.Command("test", "-f", path).CombinedOutput()
	return err == nil
}

func (e *defaultExecutor) GetVersion(name string) string {
	cmd := exec.Command(name, "--version")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	parser := buildToolVersionParsers[name]
	if parser == nil {
		return ""
	}
	return parser(strings.TrimSpace(string(out)))
}
