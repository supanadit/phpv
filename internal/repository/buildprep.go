package repository

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func ChmodBuildScripts(sourcePath string) {
	for _, dir := range []string{"build", "ext", "build-aux", "lib", "scripts"} {
		exec.Command("chmod", "-R", "+x", filepath.Join(sourcePath, dir)).Run()
	}
	for _, script := range []string{"install-sh", "depcomp", "ylwrap", "compile",
		"config.guess", "config.sub", "configure", "missing", "mkinstalldirs"} {
		p := filepath.Join(sourcePath, script)
		if _, err := os.Stat(p); err == nil {
			os.Chmod(p, 0o755)
		}
	}
}

func TouchAutotools(sourcePath string) {
	for _, f := range []string{"aclocal.m4", "Makefile.in", "configure", "config.h.in"} {
		p := filepath.Join(sourcePath, f)
		if _, err := os.Stat(p); err == nil {
			os.Chtimes(p, time.Now(), time.Now())
		}
	}
}

func TouchAllGeneratedFiles(sourcePath string) {
	now := time.Now()
	filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".am") {
			os.Chtimes(path, now, now)
		}
		return nil
	})
	time.Sleep(time.Second)
	now = time.Now()
	filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".in") {
			os.Chtimes(path, now, now)
		}
		return nil
	})
}
