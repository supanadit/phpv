package assembler

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const variantMarkerFile = ".phpv-variant.json"

// VariantMarker records which SAPI/profile a cached variant binary was built
// with, so changing the CLI build or the SAPI triggers an automatic rebuild.
type VariantMarker struct {
	SAPI        string `json:"sapi"`
	PHPVersion  string `json:"php_version"`
	ProfileHash string `json:"profile_hash"`
}

func variantMarkerPath(prefix string) string {
	return filepath.Join(prefix, variantMarkerFile)
}

func readVariantMarker(prefix string) (VariantMarker, bool) {
	var marker VariantMarker
	data, err := os.ReadFile(variantMarkerPath(prefix))
	if err != nil {
		return marker, false
	}
	if err := json.Unmarshal(data, &marker); err != nil {
		return marker, false
	}
	return marker, true
}

func writeVariantMarker(prefix string, marker VariantMarker) error {
	if err := os.MkdirAll(prefix, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(marker, "", "  ")
	if err != nil {
		return err
	}
	tmp := variantMarkerPath(prefix) + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, variantMarkerPath(prefix))
}

func variantMatches(prefix, sapi, phpVersion, profileHash string) bool {
	marker, ok := readVariantMarker(prefix)
	return ok &&
		marker.SAPI == sapi &&
		marker.PHPVersion == phpVersion &&
		marker.ProfileHash == profileHash
}

// safeRebuild replaces a cached prefix without losing a working binary when
// the new build fails: the old prefix is renamed to .bak, restored on error,
// and removed on success.
func safeRebuild(prefix string, build func() error) error {
	if err := os.MkdirAll(filepath.Dir(prefix), 0o755); err != nil {
		return err
	}
	bak := prefix + ".bak"
	_ = os.RemoveAll(bak)
	hadOld := false
	if _, err := os.Stat(prefix); err == nil {
		if err := os.Rename(prefix, bak); err != nil {
			return fmt.Errorf("move old build aside: %w", err)
		}
		hadOld = true
	}
	if err := build(); err != nil {
		_ = os.RemoveAll(prefix)
		if hadOld {
			_ = os.Rename(bak, prefix)
		}
		return err
	}
	_ = os.RemoveAll(bak)
	return nil
}
