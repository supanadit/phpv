package shell

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/spf13/viper"
)

var shimTemplate = template.Must(template.New("shim").Parse(`#!/usr/bin/env bash
# PHPV Shim - auto-generated, do not edit

set -e

if [ -z "${PHPV_VERSION:-}" ]; then
  if [ -f "${PHPV_ROOT}/version" ]; then
    PHPV_VERSION=$(cat "${PHPV_ROOT}/version")
  fi
fi

if [ -z "${PHPV_VERSION:-}" ]; then
  if [ -f ".php-version" ]; then
    PHPV_VERSION=$(cat .php-version)
  fi
fi

if [ -z "${PHPV_VERSION:-}" ]; then
  echo "phpv: no version selected" >&2
  exit 1
fi

VERSION_DIR="${PHPV_ROOT}/versions/${PHPV_VERSION}"
BINARY="${VERSION_DIR}/bin/{{.Binary}}"

if [ ! -x "$BINARY" ]; then
  echo "phpv: version ${PHPV_VERSION} is not installed" >&2
  exit 1
fi

exec "$BINARY" "$@"
`))

type ShimBinary struct {
	Binary string
}

func (s *Service) EnsureShims() error {
	root := viper.GetString("PHPV_ROOT")
	shimsDir := filepath.Join(root, "bin")

	binaries := []string{
		"php",
		"php-cgi",
		"php-config",
		"phpize",
		"phpdbg",
	}

	for _, binary := range binaries {
		shimPath := filepath.Join(shimsDir, binary)

		exists, err := fileExists(shimPath)
		if err != nil {
			return err
		}

		if !exists {
			if err := s.writeShim(shimPath, binary); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *Service) writeShim(path string, binary string) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("failed to create shim %s: %w", path, err)
	}
	defer f.Close()

	shimContent := fmt.Sprintf(`#!/usr/bin/env bash
# PHPV Shim - auto-generated, do not edit

set -e

if [ -z "${PHPV_VERSION:-}" ]; then
  if [ -f "${PHPV_ROOT}/version" ]; then
    PHPV_VERSION=$(cat "${PHPV_ROOT}/version")
  fi
fi

if [ -z "${PHPV_VERSION:-}" ]; then
  if [ -f ".php-version" ]; then
    PHPV_VERSION=$(cat .php-version)
  fi
fi

if [ -z "${PHPV_VERSION:-}" ]; then
  echo "phpv: no version selected" >&2
  exit 1
fi

VERSION_DIR="${PHPV_ROOT}/versions/${PHPV_VERSION}"
BINARY="${VERSION_DIR}/bin/%s"

if [ ! -x "$BINARY" ]; then
  echo "phpv: version ${PHPV_VERSION} is not installed" >&2
  exit 1
fi

exec "$BINARY" "$@
`, binary)

	if _, err := f.WriteString(shimContent); err != nil {
		return fmt.Errorf("failed to write shim %s: %w", path, err)
	}

	return nil
}

func fileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func ShimsDir() string {
	root := viper.GetString("PHPV_ROOT")
	return filepath.Join(root, "bin")
}
