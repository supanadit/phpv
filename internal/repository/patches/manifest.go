package patches

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ManifestFileName is the optional flags manifest inside a patch range
// directory. A directory containing only this file is still a valid
// selection: it contributes build flags without any patch application.
const ManifestFileName = "flags.yaml"

// Manifest declares the build flags contributed by a patch range
// directory. Strings pass through unchanged so assembler-side
// placeholder resolution ({{prefix}}, {{source}}, {{dep:NAME}}) is
// preserved exactly as inline patch declarations resolve today.
type Manifest struct {
	// ExtraCFlags are additional CFLAGS injected into the package's
	// build environment (e.g., to relax strict warnings).
	ExtraCFlags []string `yaml:"extraCFlags"`
	// ConfigureFlags are appended to the package's ./configure (or
	// cmake) invocation.
	ConfigureFlags []string `yaml:"configureFlags"`
}

// ParseManifest decodes a flags.yaml manifest from its raw bytes.
// Placeholder strings such as "{{dep:zlib}}" are preserved verbatim.
func ParseManifest(data []byte) (*Manifest, error) {
	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse flags manifest: %w", err)
	}
	return &m, nil
}

// LoadManifest reads and parses the flags manifest at the given path.
// Returns nil (no error) when the file does not exist, so callers can
// probe for a manifest without treating absence as a failure.
func LoadManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read %s: %w", filepath.Base(path), err)
	}
	m, err := ParseManifest(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", filepath.Base(path), err)
	}
	return m, nil
}
