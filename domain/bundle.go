package domain

import "time"

type BundleManifest struct {
	FormatVersion int              `json:"format_version"`
	Package       string           `json:"package"`
	Version       string           `json:"version"`
	OS            string           `json:"os"`
	Arch          string           `json:"arch"`
	BuildDate     time.Time        `json:"build_date"`
	Builder       BundleBuilder    `json:"builder"`
	RuntimeDeps   []BundleRuntimeDep `json:"runtime_deps"`
	BuildDeps     []BundleBuildDep `json:"build_deps"`
	Extensions    []BundleExtension `json:"extensions"`
	TotalSize     int64            `json:"total_size"`
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
