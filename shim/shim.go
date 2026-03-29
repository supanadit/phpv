package shim

import (
	"fmt"
	"os"
	"path/filepath"
)

const phpShimTemplate = `#!/bin/bash
# Shim for PHP %s
PHPV_VERSION="%s"
PHPV_OUTPUT="%s"
export PHPV_CURRENT="${PHPV_VERSION}"
export LD_LIBRARY_PATH="${PHPV_OUTPUT}/lib:${LD_LIBRARY_PATH}"
exec "${PHPV_OUTPUT}/bin/php" "$@"
`

const phpizeShimTemplate = `#!/bin/bash
# Shim for phpize %s
PHPV_VERSION="%s"
PHPV_OUTPUT="%s"
export PHPV_CURRENT="${PHPV_VERSION}"
export LD_LIBRARY_PATH="${PHPV_OUTPUT}/lib:${LD_LIBRARY_PATH}"
exec "${PHPV_OUTPUT}/bin/phpize" "$@"
`

const phpConfigShimTemplate = `#!/bin/bash
# Shim for php-config %s
PHPV_VERSION="%s"
PHPV_OUTPUT="%s"
export PHPV_CURRENT="${PHPV_VERSION}"
export LD_LIBRARY_PATH="${PHPV_OUTPUT}/lib:${LD_LIBRARY_PATH}"
exec "${PHPV_OUTPUT}/bin/php-config" "$@"
`

const phpCgiShimTemplate = `#!/bin/bash
# Shim for php-cgi %s
PHPV_VERSION="%s"
PHPV_OUTPUT="%s"
export PHPV_CURRENT="${PHPV_VERSION}"
export LD_LIBRARY_PATH="${PHPV_OUTPUT}/lib:${LD_LIBRARY_PATH}"
exec "${PHPV_OUTPUT}/bin/php-cgi" "$@"
`

func WriteShims(binPath, version, outputPath string) error {
	shims := map[string]string{
		"php":        fmt.Sprintf(phpShimTemplate, version, version, outputPath),
		"phpize":     fmt.Sprintf(phpizeShimTemplate, version, version, outputPath),
		"php-config": fmt.Sprintf(phpConfigShimTemplate, version, version, outputPath),
		"php-cgi":    fmt.Sprintf(phpCgiShimTemplate, version, version, outputPath),
	}

	for name, content := range shims {
		shimPath := filepath.Join(binPath, name)
		if err := os.WriteFile(shimPath, []byte(content), 0755); err != nil {
			return fmt.Errorf("failed to write shim %s: %w", name, err)
		}
	}

	return nil
}
