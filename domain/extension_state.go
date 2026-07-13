package domain

import "time"

const (
	ExtensionTypeBuiltin = "built-in"
	ExtensionTypePECL    = "pecl"
)

type ExtensionState struct {
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	Version     string    `json:"version,omitempty"`
	InstalledAt time.Time `json:"installed_at"`
	SoPath      string    `json:"so_path"`
}

type ExtensionManifest struct {
	PHPVersion string           `json:"php_version"`
	Extensions []ExtensionState `json:"extensions"`
}
