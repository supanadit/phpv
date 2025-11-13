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

// CheckCompatibility returns a warning message if the PHP version may have compatibility issues with modern GCC
func (v PHPVersion) CheckCompatibility() string {
	switch v.Major {
	case 4, 5:
		return fmt.Sprintf("PHP %d.x is very old and may not compile with GCC 15.x. PHPV will try GCC 6-11 for compatibility.", v.Major)
	case 7:
		if v.Minor < 4 {
			return fmt.Sprintf("PHP 7.%d may have compatibility issues with GCC 15.x. PHPV will try GCC 9-12 for compatibility.", v.Minor)
		}
	case 8:
		if v.Minor == 0 {
			return "PHP 8.0 may have some compatibility issues with GCC 15.x. PHPV will try GCC 10-14 if compilation fails."
		}
		// PHP 8.1+ should be fine with modern GCC
	}
	return ""
}

// BuildStrategy represents different approaches to building PHP
type BuildStrategy int

const (
	BuildStrategyNative      BuildStrategy = iota // Use system compiler
	BuildStrategyDocker                           // Use Docker container
	BuildStrategySpecificGCC                      // Download and use specific GCC version
)

// String returns the string representation of the build strategy
func (s BuildStrategy) String() string {
	switch s {
	case BuildStrategyNative:
		return "native"
	case BuildStrategyDocker:
		return "docker"
	case BuildStrategySpecificGCC:
		return "specific-gcc"
	default:
		return "unknown"
	}
}

// GetRecommendedBuildStrategy returns the recommended build strategy for this PHP version
func (v PHPVersion) GetRecommendedBuildStrategy() BuildStrategy {
	switch v.Major {
	case 4, 5:
		return BuildStrategySpecificGCC // Use specific GCC for very old versions
	case 7:
		if v.Minor < 4 {
			return BuildStrategySpecificGCC // May need older GCC
		}
	case 8:
		if v.Minor == 0 {
			return BuildStrategySpecificGCC // May need older GCC
		}
	}
	return BuildStrategyNative // Modern versions work with system compiler
}

// GetRecommendedGCCVersion returns the recommended GCC version for this PHP version
func (v PHPVersion) GetRecommendedGCCVersion() string {
	switch v.Major {
	case 4, 5:
		// PHP 4.x/5.x were current around 2000-2018
		// GCC 4.8-6.x would be appropriate, but may not be available in modern distros
		// Try GCC 6 first, then fall back to newer versions
		return "6"
	case 7:
		if v.Minor < 4 {
			// PHP 7.0-7.3 (2015-2018) work well with GCC 7-9
			return "9"
		}
	case 8:
		if v.Minor == 0 {
			// PHP 8.0 (2020) works with GCC 8-10
			return "10"
		}
	}
	return "" // Use system GCC for modern versions
}

// GetRecommendedDockerImage returns the recommended Docker image for building this PHP version
func (v PHPVersion) GetRecommendedDockerImage() string {
	switch v.Major {
	case 4, 5:
		return "ubuntu:16.04" // Ubuntu 16.04 has GCC 5.4, good for PHP 4.x/5.x
	case 7:
		if v.Minor < 4 {
			return "ubuntu:18.04" // Ubuntu 18.04 has GCC 7.5, good for PHP 7.0-7.3
		}
		return "ubuntu:20.04" // Ubuntu 20.04 has GCC 9.3, good for PHP 7.4+
	case 8:
		if v.Minor == 0 {
			return "ubuntu:18.04" // GCC 7.5 for PHP 8.0
		}
		return "ubuntu:22.04" // Modern Ubuntu for PHP 8.1+
	}
	return "ubuntu:24.04" // Latest Ubuntu for modern versions
}

// GetBuildRecommendations returns comprehensive build recommendations for this PHP version
func (v PHPVersion) GetBuildRecommendations() string {
	strategy := v.GetRecommendedBuildStrategy()
	gccVersion := v.GetRecommendedGCCVersion()
	dockerImage := v.GetRecommendedDockerImage()

	var recommendations []string

	switch strategy {
	case BuildStrategyDocker:
		recommendations = append(recommendations,
			fmt.Sprintf("Use Docker with image: %s", dockerImage),
			"This provides an isolated build environment with compatible tools")
	case BuildStrategySpecificGCC:
		recommendations = append(recommendations,
			fmt.Sprintf("Install and use GCC %s for compilation", gccVersion),
			"PHPV will automatically install required build dependencies",
			fmt.Sprintf("Alternative: Use Docker with image %s", dockerImage))
	default:
		recommendations = append(recommendations, "Use system compiler (GCC 11+)")
	}

	if v.Major <= 7 {
		recommendations = append(recommendations,
			"Consider applying compatibility patches if compilation fails",
			"Check PHP documentation for version-specific build requirements")
	}

	return strings.Join(recommendations, "\n")
}
