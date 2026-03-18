package disk

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type ForgeRepository interface {
	PHPVRoot() string
	VersionsPath() string
	BuildPrefix(version string) string
	SourcePath(version string) string
	LedgerPath(version string) string
	GetConfigureFlags(version string) ([]string, bool)
	ExpandConfigureFlags(version string) ([]string, bool)
}

type ConfigureFlags map[string][]string

var DefaultFlags = ConfigureFlags{
	"4.4.0": {
		"CC=zig cc -target x86_64-linux-gnu",
		"CFLAGS=-Wno-date-time -Wno-error",
		"--prefix={prefix}",
		"--disable-all",
		"--enable-cli",
		"LDFLAGS=-Wl,-rpath,'$ORIGIN/../lib'",
	},
	"5.0.0": {
		"CC=zig cc -target x86_64-linux-gnu",
		"CFLAGS=-Wno-date-time -Wno-error",
		"--prefix={prefix}",
		"--disable-all",
		"--enable-cli",
		"LDFLAGS=-Wl,-rpath,'$ORIGIN/../lib'",
	},
	"5.1.0": {
		"CC=zig cc -target x86_64-linux-gnu",
		"CFLAGS=-Wno-date-time -Wno-error",
		"--prefix={prefix}",
		"--disable-all",
		"--enable-cli",
		"LDFLAGS=-Wl,-rpath,'$ORIGIN/../lib'",
	},
	"5.2.0": {
		"CC=zig cc -target x86_64-linux-gnu",
		"CFLAGS=-Wno-date-time -Wno-error",
		"--prefix={prefix}",
		"--disable-all",
		"--enable-cli",
		"LDFLAGS=-Wl,-rpath,'$ORIGIN/../lib'",
	},
	"5.3.0": {
		"CC=zig cc -target x86_64-linux-gnu",
		"CFLAGS=-Wno-date-time -Wno-error",
		"--prefix={prefix}",
		"--disable-all",
		"--enable-cli",
		"LDFLAGS=-Wl,-rpath,'$ORIGIN/../lib'",
	},
	"5.4.0": {
		"CC=zig cc -target x86_64-linux-gnu",
		"CFLAGS=-Wno-date-time -Wno-error",
		"--prefix={prefix}",
		"--disable-all",
		"--enable-cli",
		"LDFLAGS=-Wl,-rpath,'$ORIGIN/../lib'",
	},
	"5.5.0": {
		"CC=zig cc -target x86_64-linux-gnu",
		"CFLAGS=-Wno-date-time -Wno-error",
		"--prefix={prefix}",
		"--disable-all",
		"--enable-cli",
		"LDFLAGS=-Wl,-rpath,'$ORIGIN/../lib'",
	},
	"5.6.0": {
		"CC=zig cc -target x86_64-linux-gnu",
		"CFLAGS=-Wno-date-time -Wno-error",
		"--prefix={prefix}",
		"--disable-all",
		"--enable-cli",
		"LDFLAGS=-Wl,-rpath,'$ORIGIN/../lib'",
	},
	"7.0.0": {
		"CC=zig cc -target x86_64-linux-gnu",
		"CFLAGS=-Wno-date-time -Wno-error",
		"--prefix={prefix}",
		"--disable-all",
		"--enable-cli",
		"LDFLAGS=-Wl,-rpath,'$ORIGIN/../lib'",
	},
	"7.1.0": {
		"CC=zig cc -target x86_64-linux-gnu",
		"CFLAGS=-Wno-date-time -Wno-error",
		"--prefix={prefix}",
		"--disable-all",
		"--enable-cli",
		"LDFLAGS=-Wl,-rpath,'$ORIGIN/../lib'",
	},
	"7.2.0": {
		"CC=zig cc -target x86_64-linux-gnu",
		"CFLAGS=-Wno-date-time -Wno-error",
		"--prefix={prefix}",
		"--disable-all",
		"--enable-cli",
		"LDFLAGS=-Wl,-rpath,'$ORIGIN/../lib'",
	},
	"7.3.0": {
		"CC=zig cc -target x86_64-linux-gnu",
		"CFLAGS=-Wno-date-time -Wno-error",
		"--prefix={prefix}",
		"--disable-all",
		"--enable-cli",
		"LDFLAGS=-Wl,-rpath,'$ORIGIN/../lib'",
	},
	"7.4.0": {
		"CC=zig cc -target x86_64-linux-gnu",
		"CFLAGS=-Wno-date-time -Wno-error",
		"--prefix={prefix}",
		"--disable-all",
		"--enable-cli",
		"LDFLAGS=-Wl,-rpath,'$ORIGIN/../lib'",
	},
	"8.0.0": {
		"CC=zig cc -target x86_64-linux-gnu",
		"CFLAGS=-Wno-date-time -Wno-error",
		"--prefix={prefix}",
		"--disable-all",
		"--enable-cli",
		"LDFLAGS=-Wl,-rpath,'$ORIGIN/../lib'",
	},
	"8.1.0": {
		"CC=zig cc -target x86_64-linux-gnu",
		"CFLAGS=-Wno-date-time -Wno-error",
		"--prefix={prefix}",
		"--disable-all",
		"--enable-cli",
		"LDFLAGS=-Wl,-rpath,'$ORIGIN/../lib'",
	},
	"8.2.0": {
		"CC=zig cc -target x86_64-linux-gnu",
		"CFLAGS=-Wno-date-time -Wno-error",
		"--prefix={prefix}",
		"--disable-all",
		"--enable-cli",
		"LDFLAGS=-Wl,-rpath,'$ORIGIN/../lib'",
	},
	"8.3.0": {
		"CC=zig cc -target x86_64-linux-gnu",
		"CFLAGS=-Wno-date-time -Wno-error",
		"--prefix={prefix}",
		"--disable-all",
		"--enable-cli",
		"LDFLAGS=-Wl,-rpath,'$ORIGIN/../lib'",
	},
	"8.4.0": {
		"CC=zig cc -target x86_64-linux-gnu",
		"CFLAGS=-Wno-date-time -Wno-error",
		"--prefix={prefix}",
		"--disable-all",
		"--disable-opcache",
		"--enable-cli",
		"LDFLAGS=-Wl,-rpath,'$ORIGIN/../lib'",
	},
	"8.5.0": {
		"CC=zig cc -target x86_64-linux-gnu",
		"CFLAGS=-Wno-date-time -Wno-error",
		"--prefix={prefix}",
		"--disable-all",
		"--disable-opcache",
		"--enable-cli",
		"LDFLAGS=-Wl,-rpath,'$ORIGIN/../lib'",
	},
	"8.5.4": {
		"CC=zig cc -target x86_64-linux-gnu",
		"CFLAGS=-Wno-date-time -Wno-error",
		"--prefix={prefix}",
		"--disable-all",
		"--disable-opcache",
		"--enable-cli",
		"LDFLAGS=-Wl,-rpath,'$ORIGIN/../lib'",
	},
}

func NewForgeRepository() ForgeRepository {
	return &forgeRepository{}
}

type forgeRepository struct{}

func (r *forgeRepository) PHPVRoot() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.Getenv("HOME")
	}
	if home == "" {
		return "~/.phpv"
	}
	return filepath.Join(home, ".phpv")
}

func (r *forgeRepository) VersionsPath() string {
	return r.PHPVRoot() + "/versions"
}

func (r *forgeRepository) BuildPrefix(version string) string {
	return r.VersionsPath() + "/" + version
}

func (r *forgeRepository) SourcePath(version string) string {
	return r.PHPVRoot() + "/sources/" + version + "/php"
}

func (r *forgeRepository) LedgerPath(version string) string {
	return r.PHPVRoot() + "/ledger/" + version + ".json"
}

func (r *forgeRepository) GetConfigureFlags(version string) ([]string, bool) {
	flags, ok := DefaultFlags[version]
	return flags, ok
}

func (r *forgeRepository) ExpandConfigureFlags(version string) ([]string, bool) {
	prefix := r.BuildPrefix(version)
	flags, ok := DefaultFlags[version]
	if !ok {
		return nil, false
	}

	expanded := make([]string, len(flags))
	for i, flag := range flags {
		expanded[i] = expandPrefix(flag, prefix)
	}
	return expanded, true
}

func expandPrefix(flag, prefix string) string {
	return strings.ReplaceAll(flag, "{prefix}", prefix)
}

type BuildRepository interface {
	Configure(sourceDir string, flags []string) error
	Make(sourceDir string, jobs int) error
	Install(sourceDir string) error
	Distclean(sourceDir string) error
}

func NewBuildRepository() BuildRepository {
	return &buildRepository{}
}

type buildRepository struct{}

func (r *buildRepository) Configure(sourceDir string, flags []string) error {
	configurePath := filepath.Join(sourceDir, "configure")

	bashPath, err := exec.LookPath("bash")
	if err != nil {
		return fmt.Errorf("bash not found: %w", err)
	}

	if stat, err := os.Stat(configurePath); err == nil && stat.Size() == 0 {
		fmt.Printf("configure is empty, running buildconf...\n")
		buildconfPath := filepath.Join(sourceDir, "buildconf")
		if err := os.Chmod(buildconfPath, 0o755); err != nil {
			return fmt.Errorf("failed to make buildconf executable: %w", err)
		}

		cmd := exec.Command(bashPath, buildconfPath)
		cmd.Dir = sourceDir
		cmd.Env = os.Environ()
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("buildconf failed: %w", err)
		}
	}

	if _, err := os.Stat(configurePath); os.IsNotExist(err) {
		return fmt.Errorf("configure script not found at %s", configurePath)
	}

	if err := os.Chmod(configurePath, 0o755); err != nil {
		return fmt.Errorf("failed to make configure executable: %w", err)
	}

	args := []string{}
	env := os.Environ()
	for _, flag := range flags {
		if strings.Contains(flag, "=") {
			parts := strings.SplitN(flag, "=", 2)
			if parts[0] == "CC" || parts[0] == "CFLAGS" || parts[0] == "LDFLAGS" {
				env = append(env, flag)
				continue
			}
		}
		args = append(args, flag)
	}

	cmd := exec.Command(bashPath, append([]string{configurePath}, args...)...)
	cmd.Dir = sourceDir
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("Running configure in %s...\n", sourceDir)
	return cmd.Run()
}

func (r *buildRepository) Make(sourceDir string, jobs int) error {
	if jobs <= 0 {
		jobs = 1
	}

	makePath, err := exec.LookPath("make")
	if err != nil {
		return fmt.Errorf("make not found: %w", err)
	}

	cmd := exec.Command(makePath, "-j", fmt.Sprintf("%d", jobs))
	cmd.Dir = sourceDir
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("Running make -j%d in %s...\n", jobs, sourceDir)
	return cmd.Run()
}

func (r *buildRepository) Install(sourceDir string) error {
	makePath, err := exec.LookPath("make")
	if err != nil {
		return fmt.Errorf("make not found: %w", err)
	}

	cmd := exec.Command(makePath, "install")
	cmd.Dir = sourceDir
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("Running make install in %s...\n", sourceDir)
	return cmd.Run()
}

func (r *buildRepository) Distclean(sourceDir string) error {
	makePath, err := exec.LookPath("make")
	if err != nil {
		return fmt.Errorf("make not found: %w", err)
	}

	cmd := exec.Command(makePath, "distclean")
	cmd.Dir = sourceDir
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("Running make distclean in %s...\n", sourceDir)
	if err := cmd.Run(); err != nil {
		fmt.Println("distclean failed (this is normal if not previously configured)")
	}
	return nil
}
