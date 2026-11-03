package assembler

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// BuildProfile captures how the CLI PHP was configured so SAPI variants
// (php-fpm, the Apache module) can be built from the exact same extension and
// dependency set. Without it, variants built with --disable-all diverge from
// the CLI install (missing pdo_mysql, curl, mbstring, ...).
type BuildProfile struct {
	ConfigureArgs []string          `json:"configure_args"`
	Env           map[string]string `json:"env,omitempty"`
}

const buildProfileFile = ".build.json"

func buildProfilePath(prefix string) string {
	return filepath.Join(prefix, buildProfileFile)
}

// saveBuildProfile writes the profile next to the CLI install.
func saveBuildProfile(prefix string, profile BuildProfile) error {
	if err := os.MkdirAll(prefix, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return err
	}
	tmp := buildProfilePath(prefix) + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, buildProfilePath(prefix))
}

func readBuildProfile(prefix string) (BuildProfile, error) {
	var profile BuildProfile
	data, err := os.ReadFile(buildProfilePath(prefix))
	if err != nil {
		return profile, err
	}
	if err := json.Unmarshal(data, &profile); err != nil {
		return profile, err
	}
	return profile, nil
}

// loadCLIProfile returns the persisted CLI build profile, or derives one from
// php-config for installs created before profiles existed.
func loadCLIProfile(phpVersion, cliPrefix, root string) (BuildProfile, error) {
	if profile, err := readBuildProfile(cliPrefix); err == nil && len(profile.ConfigureArgs) > 0 {
		return profile, nil
	}

	phpConfig := filepath.Join(cliPrefix, "bin", "php-config")
	out, err := exec.Command(phpConfig, "--configure-options").Output()
	if err != nil {
		return BuildProfile{}, fmt.Errorf("read CLI build profile for PHP %s: %w", phpVersion, err)
	}
	args, env := parseConfigureOptions(string(out))
	return BuildProfile{ConfigureArgs: args, Env: env}, nil
}

// parseConfigureOptions splits `php-config --configure-options` into configure
// arguments and trailing KEY=VALUE environment tokens.
func parseConfigureOptions(raw string) (args []string, env map[string]string) {
	env = map[string]string{}
	for _, token := range strings.Fields(raw) {
		if token == "" {
			continue
		}
		if !strings.HasPrefix(token, "-") {
			if key, value, ok := strings.Cut(token, "="); ok && key != "" {
				env[key] = value
				continue
			}
		}
		args = append(args, token)
	}
	return args, env
}

// Hash fingerprints the profile for a SAPI so stale variant builds can be
// detected and rebuilt automatically.
func (p BuildProfile) Hash(sapi string) string {
	h := sha256.New()
	fmt.Fprintf(h, "sapi=%s\n", sapi)
	for _, arg := range p.ConfigureArgs {
		fmt.Fprintln(h, arg)
	}
	keys := make([]string, 0, len(p.Env))
	for key := range p.Env {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if key == "PATH" {
			// PATH only affects tool lookup, not the built configuration;
			// excluding it avoids spurious rebuilds when the shell changes.
			continue
		}
		fmt.Fprintf(h, "%s=%s\n", key, p.Env[key])
	}
	return hex.EncodeToString(h.Sum(nil))
}

// VariantArgs rewrites a CLI profile for another SAPI: only the prefix and the
// SAPI flags change, every extension/dependency flag is preserved.
func (p BuildProfile) VariantArgs(sapi, prefix, httpdPrefix string) ([]string, error) {
	out := make([]string, 0, len(p.ConfigureArgs)+4)
	for _, arg := range p.ConfigureArgs {
		switch {
		case strings.HasPrefix(arg, "--prefix="),
			arg == "--enable-cli", arg == "--disable-cli",
			arg == "--enable-cgi", arg == "--disable-cgi",
			arg == "--enable-fpm", arg == "--disable-fpm",
			arg == "--enable-phpdbg", arg == "--disable-phpdbg",
			arg == "--enable-embed", arg == "--disable-embed",
			strings.HasPrefix(arg, "--with-apxs2="):
			continue
		}
		out = append(out, arg)
	}
	out = append(out, "--prefix="+prefix)

	switch sapi {
	case "fpm":
		out = append(out, "--enable-fpm", "--disable-cli", "--disable-cgi", "--disable-phpdbg", "--disable-embed")
	case "modphp":
		if httpdPrefix == "" {
			return nil, fmt.Errorf("apache prefix is required for the mod_php SAPI")
		}
		apxs := filepath.Join(httpdPrefix, "bin", "apxs")
		if _, err := os.Stat(apxs); err != nil {
			return nil, fmt.Errorf("apxs not found at %s (install apache first: phpv server install apache)", apxs)
		}
		out = append(out,
			"--with-apxs2="+apxs,
			"--disable-cli", "--disable-cgi", "--disable-fpm", "--disable-phpdbg", "--disable-embed",
		)
	default:
		return nil, fmt.Errorf("unsupported PHP SAPI %q (want fpm or modphp)", sapi)
	}
	return out, nil
}

// cliExtensionDir resolves the CLI install's dynamically loadable extension
// directory. The Apache module bakes this in via EXTENSION_DIR so php.ini's
// `extension=` lines load at startup (php_admin_value is applied too late).
func cliExtensionDir(cliPrefix string) string {
	phpConfig := filepath.Join(cliPrefix, "bin", "php-config")
	if out, err := exec.Command(phpConfig, "--extension-dir").Output(); err == nil {
		if value := strings.TrimSpace(string(out)); value != "" {
			return value
		}
	}
	matches, _ := filepath.Glob(filepath.Join(cliPrefix, "lib", "php", "extensions", "*"))
	for _, match := range matches {
		if st, err := os.Stat(match); err == nil && st.IsDir() {
			return match
		}
	}
	return ""
}

// ApplyEnv layers the profile environment over a base environment.
func (p BuildProfile) ApplyEnv(base []string) []string {
	env := base
	keys := make([]string, 0, len(p.Env))
	for key := range p.Env {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		env = setEnvVar(env, key, p.Env[key])
	}
	return env
}

// profileEnv extracts the build-relevant variables from an os.Environ slice.
func profileEnv(env []string) map[string]string {
	out := map[string]string{}
	for _, entry := range env {
		key, value, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}
		switch key {
		case "CC", "CXX", "CFLAGS", "CXXFLAGS", "CPPFLAGS", "LDFLAGS",
			"PKG_CONFIG_PATH", "PATH", "LD_LIBRARY_PATH", "DYLD_LIBRARY_PATH":
			out[key] = value
		}
	}
	return out
}
