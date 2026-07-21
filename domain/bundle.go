package domain

import "time"

type BundleManifest struct {
	FormatVersion int                `json:"format_version"`
	Package       string             `json:"package"`
	Version       string             `json:"version"`
	OS            string             `json:"os"`
	Arch          string             `json:"arch"`
	Libc          string             `json:"libc,omitempty"`
	GlibcVersion  string             `json:"glibc_version,omitempty"`
	PhpApiVersion string             `json:"php_api_version,omitempty"`
	BuildDate     time.Time          `json:"build_date"`
	Builder       BundleBuilder      `json:"builder"`
	RuntimeDeps   []BundleRuntimeDep `json:"runtime_deps"`
	BuildDeps     []BundleBuildDep   `json:"build_deps"`
	Extensions    []BundleExtension  `json:"extensions"`
	ExtPool       []BundleExtArtifact `json:"ext_pool,omitempty"`
	Toolchain     BundleToolchain    `json:"toolchain,omitempty"`
	TotalSize     int64              `json:"total_size"`
}

type BundleBuilder struct {
	PHPVVersion    string   `json:"phpv_version"`
	Compiler       string   `json:"compiler"`
	ConfigureFlags []string `json:"configure_flags"`
	Static         bool     `json:"static"`
	Libc           string   `json:"libc"`
}

type BundleRuntimeDep struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type BundleBuildDep struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Prefix  string `json:"prefix"`
}

type BundleExtension struct {
	Name          string   `json:"name"`
	Version       string   `json:"version"`
	SOFile        string   `json:"so_file"`
	ConfigureArgs []string `json:"configure_args"`
}

type BundleExtArtifact struct {
	Name          string   `json:"name"`
	Version       string   `json:"version"`
	SOFile        string   `json:"so_file"`
	PhpApiVersion string   `json:"php_api_version"`
	ConfigureArgs []string `json:"configure_args"`
}

type BundleToolchain struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Arch    string `json:"arch"`
	URL     string `json:"url"`
	SHA256  string `json:"sha256"`
}

// BundleMeta is stored in the PHP prefix after importing a portable bundle.
// It records the libc type and PHP API version for use by InstallExtension.
type BundleMeta struct {
	Libc          string `json:"libc"`
	PhpApiVersion string `json:"php_api_version"`
}
