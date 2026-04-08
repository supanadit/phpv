package shim

import (
	"fmt"
	"os"
	"path/filepath"
)

const dynamicShimTemplate = `#!/bin/bash
# Dynamic shim - resolves PHP version at runtime
PHPV_ROOT="${PHPV_ROOT:-$HOME/.phpv}"
PHPV_VERSION="${PHPV_CURRENT:-$(cat "$PHPV_ROOT/default" 2>/dev/null)}"
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
export LD_LIBRARY_PATH="$PHPV_OUTPUT/lib:${LD_LIBRARY_PATH}"
exec "${PHPV_OUTPUT}/bin/%s" "$@"
`

func WriteShims(binPath string) error {
	shims := []string{
		"php",
		"phpize",
		"php-config",
		"php-cgi",
	}

	for _, name := range shims {
		shimPath := filepath.Join(binPath, name)
		content := fmt.Sprintf(dynamicShimTemplate, name)
		if err := os.WriteFile(shimPath, []byte(content), 0755); err != nil {
			return fmt.Errorf("failed to write shim %s: %w", name, err)
		}
	}

	return nil
}
