package shim

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

// PharTool defines a PHAR-based tool that gets a shell shim.
type PharTool struct {
	Name     string // display + shim name: composer, pie, wp
	PharFile string // phar filename: composer.phar, pie.phar, wp-cli.phar
	BinName  string // system binary name for --system detection: composer, pie, wp
}

// DefaultPharTools is the canonical list of PHAR tools phpv manages.
var DefaultPharTools = []PharTool{
	{Name: "composer", PharFile: "composer.phar", BinName: "composer"},
	{Name: "pie", PharFile: "pie.phar", BinName: "pie"},
	{Name: "wp", PharFile: "wp-cli.phar", BinName: "wp"},
}

// pharShimTmpl is a single reusable bash shim for any PHAR tool.
var pharShimTmpl = template.Must(template.New("phar").Parse(`#!/bin/bash
# phpv shim for {{.Name}} - resolves PHP version at runtime
# Resolution order: PHPV_CURRENT -> .phpvrc -> composer.json -> $PHPV_ROOT/default -> system
PHPV_ROOT="${PHPV_ROOT:-$HOME/.phpv}"
if [ -f "$PHPV_ROOT/.phpv_system" ]; then
    PHP_PATH="$(command -v php 2>/dev/null)"
    if [ -z "$PHP_PATH" ]; then
        echo "Error: System PHP not found" >&2
        exit 1
    fi
    PHAR_PATH="$PHPV_ROOT/phar/{{.PharFile}}"
    if [ ! -f "$PHAR_PATH" ]; then
        echo "Error: {{.Name}} not found. Please install {{.Name}} first." >&2
        echo "Hint: phpv phar install {{.Name}}" >&2
        exit 1
    fi
    exec "$PHP_PATH" "$PHAR_PATH" "$@"
fi
if [ -n "$PHPV_CURRENT" ]; then
    PHPV_VERSION="$PHPV_CURRENT"
elif [ -f .phpvrc ] && [ -s .phpvrc ]; then
    PHPV_VERSION="$(phpv auto-detect-resolve "$(cat .phpvrc)" 2>/dev/null)"
    if [ -z "$PHPV_VERSION" ]; then
        PHPV_VERSION="$(cat .phpvrc)"
    fi
else
    PHPV_VERSION="$(phpv auto-detect-resolve 2>/dev/null || cat "$PHPV_ROOT/default" 2>/dev/null)"
fi
if [ -z "$PHPV_VERSION" ]; then
    echo "Error: No PHP version selected. Run 'phpv use <version>' first." >&2
    exit 1
fi
PHPV_OUTPUT="$PHPV_ROOT/versions/$PHPV_VERSION/output"
if [ ! -d "$PHPV_OUTPUT" ]; then
    echo "Error: PHP version $PHPV_VERSION not found. Run 'phpv install $PHPV_VERSION' first." >&2
    exit 1
fi
export PHPV_CURRENT="$PHPV_VERSION"
PHPV_DEPS="$PHPV_ROOT/versions/$PHPV_VERSION/dependency"
if [ -d "$PHPV_DEPS" ]; then
    for dep_lib in "$PHPV_DEPS"/*/*/lib; do
        [ -d "$dep_lib" ] && LD_LIBRARY_PATH="$dep_lib:$LD_LIBRARY_PATH"
    done
fi
export LD_LIBRARY_PATH="$PHPV_OUTPUT/lib:$LD_LIBRARY_PATH"
PHAR_PATH="$PHPV_ROOT/versions/$PHPV_VERSION/phar/{{.PharFile}}"
if [ ! -f "$PHAR_PATH" ]; then
    echo "Error: {{.Name}} not installed for PHP $PHPV_VERSION" >&2
    echo "Hint: phpv phar install {{.Name}}" >&2
    exit 1
fi
exec "${PHPV_OUTPUT}/bin/php" "$PHAR_PATH" "$@"
`))

// phpShimTmpl is the template for PHP binary shims (php, phpize, etc).
var phpShimTmpl = template.Must(template.New("php").Parse(`#!/bin/bash
# phpv shim for {{.Name}} - resolves PHP version at runtime
# Resolution order: PHPV_CURRENT -> .phpvrc -> composer.json -> $PHPV_ROOT/default -> system
PHPV_ROOT="${PHPV_ROOT:-$HOME/.phpv}"
if [ -f "$PHPV_ROOT/.phpv_system" ]; then
    PHP_PATH="$(command -v {{.Name}} 2>/dev/null)"
    if [ -z "$PHP_PATH" ]; then
        echo "Error: System {{.Name}} not found" >&2
        exit 1
    fi
    exec "$PHP_PATH" "$@"
fi
if [ -n "$PHPV_CURRENT" ]; then
    PHPV_VERSION="$PHPV_CURRENT"
elif [ -f .phpvrc ] && [ -s .phpvrc ]; then
    PHPV_VERSION="$(phpv auto-detect-resolve "$(cat .phpvrc)" 2>/dev/null)"
    if [ -z "$PHPV_VERSION" ]; then
        PHPV_VERSION="$(cat .phpvrc)"
    fi
else
    PHPV_VERSION="$(phpv auto-detect-resolve 2>/dev/null || cat "$PHPV_ROOT/default" 2>/dev/null)"
fi
if [ -z "$PHPV_VERSION" ]; then
    echo "Error: No PHP version selected. Run 'phpv use <version>' first." >&2
    exit 1
fi
PHPV_OUTPUT="$PHPV_ROOT/versions/$PHPV_VERSION/output"
if [ ! -d "$PHPV_OUTPUT" ]; then
    echo "Error: PHP version $PHPV_VERSION not found. Run 'phpv install $PHPV_VERSION' first." >&2
    exit 1
fi
export PHPV_CURRENT="$PHPV_VERSION"
PHPV_DEPS="$PHPV_ROOT/versions/$PHPV_VERSION/dependency"
if [ -d "$PHPV_DEPS" ]; then
    for dep_lib in "$PHPV_DEPS"/*/*/lib; do
        [ -d "$dep_lib" ] && LD_LIBRARY_PATH="$dep_lib:$LD_LIBRARY_PATH"
    done
fi
export LD_LIBRARY_PATH="$PHPV_OUTPUT/lib:$LD_LIBRARY_PATH"
exec "${PHPV_OUTPUT}/bin/{{.Name}}" "$@"
`))

// phpBinNames is the list of PHP binaries that get shims.
var phpBinNames = []string{"php", "phpize", "php-config", "php-cgi"}

type shimData struct {
	Name string
}

type pharShimData struct {
	Name     string
	PharFile string
}

type ShimConfig struct {
	BinPath string
}

// DetectPharPath returns the path to a system-installed phar tool.
func DetectPharPath(tool PharTool, phpvRoot string) string {
	phpvBin := filepath.Join(phpvRoot, "bin")
	phpvPhar := filepath.Join(phpvRoot, "phar")

	localPhar := filepath.Join(phpvPhar, tool.PharFile)
	if _, err := os.Stat(localPhar); err == nil {
		return localPhar
	}

	pathEnv := os.Getenv("PATH")
	var filteredParts []string
	for _, part := range strings.Split(pathEnv, ":") {
		if part != phpvBin && !strings.HasPrefix(part, phpvRoot+"/") {
			filteredParts = append(filteredParts, part)
		}
	}
	filteredPath := strings.Join(filteredParts, ":")

	cmd := exec.Command("which", tool.BinName)
	cmd.Env = append(os.Environ(), "PATH="+filteredPath)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func WriteShims(cfg ShimConfig) error {
	// PHP binary shims
	for _, name := range phpBinNames {
		shimPath := filepath.Join(cfg.BinPath, name)
		var buf bytes.Buffer
		if err := phpShimTmpl.Execute(&buf, shimData{Name: name}); err != nil {
			return fmt.Errorf("failed to render shim %s: %w", name, err)
		}
		if err := os.WriteFile(shimPath, buf.Bytes(), 0755); err != nil {
			return fmt.Errorf("failed to write shim %s: %w", name, err)
		}
	}

	// PHAR tool shims
	for _, tool := range DefaultPharTools {
		shimPath := filepath.Join(cfg.BinPath, tool.Name)
		var buf bytes.Buffer
		if err := pharShimTmpl.Execute(&buf, pharShimData{Name: tool.Name, PharFile: tool.PharFile}); err != nil {
			return fmt.Errorf("failed to render shim %s: %w", tool.Name, err)
		}
		if err := os.WriteFile(shimPath, buf.Bytes(), 0755); err != nil {
			return fmt.Errorf("failed to write shim %s: %w", tool.Name, err)
		}
	}

	return nil
}

func SystemMarkerPath(siloRoot string) string {
	return filepath.Join(siloRoot, ".phpv_system")
}

func WriteSystemMarker(siloRoot string) error {
	markerPath := SystemMarkerPath(siloRoot)
	if err := os.WriteFile(markerPath, []byte{}, 0644); err != nil {
		return fmt.Errorf("failed to write system marker: %w", err)
	}
	return nil
}

func IsSystemMode(siloRoot string) bool {
	markerPath := SystemMarkerPath(siloRoot)
	_, err := os.Stat(markerPath)
	return err == nil
}

func RemoveSystemMarker(siloRoot string) error {
	markerPath := SystemMarkerPath(siloRoot)
	if _, err := os.Stat(markerPath); os.IsNotExist(err) {
		return nil
	}
	if err := os.Remove(markerPath); err != nil {
		return fmt.Errorf("failed to remove system marker: %w", err)
	}
	return nil
}
