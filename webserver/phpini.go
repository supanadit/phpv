package webserver

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// CLIRuntime returns the CLI install's php.ini directory and extension_dir.
// It is the single source of truth for both FPM (`-c`, `-d extension_dir`)
// and the Apache module (PHPIniDir, extension_dir), so `php -S` and the
// webserver load the same configuration and shared extensions.
func CLIRuntime(cliPrefix string) (iniDir, extDir string) {
	iniDir = filepath.Join(cliPrefix, "lib")
	phpConfig := filepath.Join(cliPrefix, "bin", "php-config")

	if out, err := exec.Command(phpConfig, "--ini-path").Output(); err == nil {
		if value := strings.TrimSpace(string(out)); value != "" {
			iniDir = value
		}
	}
	if out, err := exec.Command(phpConfig, "--extension-dir").Output(); err == nil {
		if value := strings.TrimSpace(string(out)); value != "" {
			extDir = value
		}
	}
	if extDir == "" {
		// Legacy fallback: the API directory is the only child of
		// lib/php/extensions.
		matches, _ := filepath.Glob(filepath.Join(cliPrefix, "lib", "php", "extensions", "*"))
		for _, match := range matches {
			if st, err := os.Stat(match); err == nil && st.IsDir() {
				extDir = match
				break
			}
		}
	}
	return iniDir, extDir
}
