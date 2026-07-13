package domain

import "time"

type ExtensionState struct {
	Name        string    `json:"name"`
	InstalledAt time.Time `json:"installed_at"`
	SoPath      string    `json:"so_path"`
}

type ExtensionManifest struct {
	PHPVersion string           `json:"php_version"`
	Extensions []ExtensionState `json:"extensions"`
}
