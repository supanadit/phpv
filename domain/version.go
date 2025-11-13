package domain

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// PHPVersion represents a PHP version entity with all its metadata
type PHPVersion struct {
	Version       string    `json:"version"`
	Major         int       `json:"major"`
	Minor         int       `json:"minor"`
	Patch         int       `json:"patch"`
	ReleaseType   string    `json:"release_type"`   // stable, rc, alpha, beta
	ReleaseNumber int       `json:"release_number"` // for rc1, rc2, etc.
	DownloadURL   string    `json:"download_url"`
	SHA256        string    `json:"sha256"`
	ReleasedAt    time.Time `json:"released_at"`
}

// Installation represents an installed PHP version on the system
type Installation struct {
	Version     PHPVersion `json:"version"`
	Path        string     `json:"path"`
	IsActive    bool       `json:"is_active"`
	InstalledAt time.Time  `json:"installed_at"`
}

// ParseVersion parses a version string like "8.1.0" or "8.2.0-rc1" into a PHPVersion struct
func ParseVersion(versionStr string) (PHPVersion, error) {
	versionStr = strings.TrimSpace(versionStr)
	if versionStr == "" {
		return PHPVersion{}, ErrBadParamInput
	}

	// Regex to match version patterns like:
	// 8.1.0 (stable)
	// 8.2.0-rc1 (release candidate)
	// 8.3.0-alpha2 (alpha)
	// 8.3.0-beta1 (beta)
	re := regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)(?:-([a-zA-Z]+)(\d+))?$`)
	matches := re.FindStringSubmatch(versionStr)

	if len(matches) == 0 {
		return PHPVersion{}, ErrBadParamInput
	}

	major, err := strconv.Atoi(matches[1])
	if err != nil {
		return PHPVersion{}, ErrBadParamInput
	}

	minor, err := strconv.Atoi(matches[2])
	if err != nil {
		return PHPVersion{}, ErrBadParamInput
	}

	patch, err := strconv.Atoi(matches[3])
	if err != nil {
		return PHPVersion{}, ErrBadParamInput
	}

	version := PHPVersion{
		Version:     versionStr,
		Major:       major,
		Minor:       minor,
		Patch:       patch,
		ReleaseType: "stable", // default to stable
	}

	// Check if there's a release type (rc, alpha, beta)
	if len(matches) > 4 && matches[4] != "" {
		releaseType := strings.ToLower(matches[4])
		if releaseType != "rc" && releaseType != "alpha" && releaseType != "beta" {
			return PHPVersion{}, ErrBadParamInput
		}
		version.ReleaseType = releaseType

		if len(matches) > 5 && matches[5] != "" {
			releaseNumber, err := strconv.Atoi(matches[5])
			if err != nil {
				return PHPVersion{}, ErrBadParamInput
			}
			version.ReleaseNumber = releaseNumber
		}
	}

	return version, nil
}

// String returns the string representation of the version
func (v PHPVersion) String() string {
	if v.ReleaseType == "stable" {
		return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
	}
	return fmt.Sprintf("%d.%d.%d-%s%d", v.Major, v.Minor, v.Patch, v.ReleaseType, v.ReleaseNumber)
}

// IsStable returns true if this is a stable release
func (v PHPVersion) IsStable() bool {
	return v.ReleaseType == "stable"
}

// Compare compares two versions, returns -1 if v < other, 0 if equal, 1 if v > other
func (v PHPVersion) Compare(other PHPVersion) int {
	if v.Major != other.Major {
		if v.Major < other.Major {
			return -1
		}
		return 1
	}
	if v.Minor != other.Minor {
		if v.Minor < other.Minor {
			return -1
		}
		return 1
	}
	if v.Patch != other.Patch {
		if v.Patch < other.Patch {
			return -1
		}
		return 1
	}
	// Compare release types: stable > rc > beta > alpha
	releaseOrder := map[string]int{"stable": 4, "rc": 3, "beta": 2, "alpha": 1}
	vOrder := releaseOrder[v.ReleaseType]
	otherOrder := releaseOrder[other.ReleaseType]
	if vOrder != otherOrder {
		if vOrder < otherOrder {
			return -1
		}
		return 1
	}
	if v.ReleaseNumber != other.ReleaseNumber {
		if v.ReleaseNumber < other.ReleaseNumber {
			return -1
		}
		return 1
	}
	return 0
}

// Activate marks this installation as active
func (i *Installation) Activate() {
	i.IsActive = true
}

// Deactivate marks this installation as inactive
func (i *Installation) Deactivate() {
	i.IsActive = false
}

// IsInstalled checks if the installation exists at the path
func (i Installation) IsInstalled() bool {
	// TODO: implement file system check
	return true
}

// Validate checks if the PHPVersion is valid
func (v PHPVersion) Validate() error {
	if v.Major < 0 || v.Minor < 0 || v.Patch < 0 {
		return ErrBadParamInput
	}

	if v.ReleaseType != "stable" && v.ReleaseType != "rc" && v.ReleaseType != "alpha" && v.ReleaseType != "beta" {
		return ErrBadParamInput
	}

	if v.ReleaseNumber < 0 {
		return ErrBadParamInput
	}

	if v.ReleaseType == "stable" && v.ReleaseNumber != 0 {
		return ErrBadParamInput
	}

	return nil
}

// Validate checks if the Installation is valid
func (i Installation) Validate() error {
	if err := i.Version.Validate(); err != nil {
		return err
	}

	if strings.TrimSpace(i.Path) == "" {
		return ErrBadParamInput
	}

	return nil
}
