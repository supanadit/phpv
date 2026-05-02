package shim

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/supanadit/phpv/internal/config"
)

const dynamicShimTemplate = `#!/bin/bash
# Dynamic shim - resolves PHP version at runtime
# Resolution order: PHPV_CURRENT → .phpvrc → composer.json → $PHPV_ROOT/default → system
PHPV_ROOT="${PHPV_ROOT:-$HOME/.phpv}"
if [ -f "$PHPV_ROOT/.phpv_system" ]; then
    PHP_PATH="$(command -v %s 2>/dev/null)"
    if [ -z "$PHP_PATH" ]; then
        echo "Error: System %s not found" >&2
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
exec "${PHPV_OUTPUT}/bin/%s" "$@"
`

const composerShimTemplate = `#!/bin/bash
# Dynamic shim for composer - runs system composer through phpv PHP
PHPV_ROOT="${PHPV_ROOT:-$HOME/.phpv}"
if [ -f "$PHPV_ROOT/.phpv_system" ]; then
    PHP_PATH="$(command -v php 2>/dev/null)"
    if [ -z "$PHP_PATH" ]; then
        echo "Error: System PHP not found" >&2
        exit 1
    fi
    COMPOSER_PATH="{{ .ComposerPath }}"
    if [ -z "$COMPOSER_PATH" ]; then
        echo "Error: composer not found. Please install composer first." >&2
        echo "Hint: https://getcomposer.org/download/" >&2
        exit 1
    fi
    exec "$PHP_PATH" "$COMPOSER_PATH" "$@"
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
COMPOSER_PATH="{{ .ComposerPath }}"
if [ -z "$COMPOSER_PATH" ]; then
    echo "Error: composer not found. Please install composer first." >&2
    echo "Hint: https://getcomposer.org/download/" >&2
    exit 1
fi
exec "${PHPV_OUTPUT}/bin/php" "$COMPOSER_PATH" "$@"
`

const pieShimTemplate = `#!/bin/bash
# Dynamic shim for pie - runs pie phar through phpv PHP
PHPV_ROOT="${PHPV_ROOT:-$HOME/.phpv}"
if [ -f "$PHPV_ROOT/.phpv_system" ]; then
    PHP_PATH="$(command -v php 2>/dev/null)"
    if [ -z "$PHP_PATH" ]; then
        echo "Error: System PHP not found" >&2
        exit 1
    fi
    PIE_PATH="{{ .PiePath }}"
    if [ -z "$PIE_PATH" ]; then
        echo "Error: pie not found. Please install pie first." >&2
        echo "Hint: phpv phar install pie" >&2
        exit 1
    fi
    exec "$PHP_PATH" "$PIE_PATH" "$@"
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
PIE_PATH="{{ .PiePath }}"
if [ -z "$PIE_PATH" ]; then
    echo "Error: pie not found. Please install pie first." >&2
    echo "Hint: phpv phar install pie" >&2
    exit 1
fi
exec "${PHPV_OUTPUT}/bin/php" "$PIE_PATH" "$@"
`

type ShimConfig struct {
	BinPath      string
	ComposerPath string
	PiePath      string
}

func DetectComposerPath() string {
	cfg := config.Get()
	phpvRoot := cfg.RootDir()
	phpvBin := cfg.BinPath()
	phpvPhar := cfg.PharPath()

	localComposer := filepath.Join(phpvPhar, "composer.phar")
	if _, err := os.Stat(localComposer); err == nil {
		return localComposer
	}

	pathEnv := os.Getenv("PATH")
	var filteredParts []string
	for _, part := range strings.Split(pathEnv, ":") {
		if part != phpvBin && !strings.HasPrefix(part, phpvRoot+"/") {
			filteredParts = append(filteredParts, part)
		}
	}
	filteredPath := strings.Join(filteredParts, ":")

	cmd := exec.Command("which", "composer")
	cmd.Env = append(os.Environ(), "PATH="+filteredPath)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func DetectPiePath() string {
	cfg := config.Get()
	phpvRoot := cfg.RootDir()
	phpvBin := cfg.BinPath()
	phpvPhar := cfg.PharPath()

	localPie := filepath.Join(phpvPhar, "pie.phar")
	if _, err := os.Stat(localPie); err == nil {
		return localPie
	}

	pathEnv := os.Getenv("PATH")
	var filteredParts []string
	for _, part := range strings.Split(pathEnv, ":") {
		if part != phpvBin && !strings.HasPrefix(part, phpvRoot+"/") {
			filteredParts = append(filteredParts, part)
		}
	}
	filteredPath := strings.Join(filteredParts, ":")

	cmd := exec.Command("which", "pie")
	cmd.Env = append(os.Environ(), "PATH="+filteredPath)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func WriteShims(cfg ShimConfig) error {
	phpShims := []string{
		"php",
		"phpize",
		"php-config",
		"php-cgi",
	}

	for _, name := range phpShims {
		shimPath := filepath.Join(cfg.BinPath, name)
		content := fmt.Sprintf(dynamicShimTemplate, name, name, name)
		if err := os.WriteFile(shimPath, []byte(content), 0755); err != nil {
			return fmt.Errorf("failed to write shim %s: %w", name, err)
		}
	}

	composerPath := cfg.ComposerPath
	shimPath := filepath.Join(cfg.BinPath, "composer")
	content := strings.ReplaceAll(composerShimTemplate, "{{ .ComposerPath }}", composerPath)
	if err := os.WriteFile(shimPath, []byte(content), 0755); err != nil {
		return fmt.Errorf("failed to write shim composer: %w", err)
	}

	if cfg.PiePath != "" {
		piePath := cfg.PiePath
		shimPath := filepath.Join(cfg.BinPath, "pie")
		content := strings.ReplaceAll(pieShimTemplate, "{{ .PiePath }}", piePath)
		if err := os.WriteFile(shimPath, []byte(content), 0755); err != nil {
			return fmt.Errorf("failed to write shim pie: %w", err)
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
