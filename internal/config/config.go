package config

import (
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

// Config holds all configuration for phpv.
// All environment variable handling should go through this package.
type Config struct {
	Root    string // PHPV_ROOT
	HomeDir string
	OS      string
	Arch    string
}

var (
	cfg       *Config
	once      sync.Once
	overrides *Config // For testing - allows overriding the global config
)

// Get returns the global Config singleton.
// Thread-safe, initialized only once.
func Get() *Config {
	if overrides != nil {
		return overrides
	}
	once.Do(initConfig)
	return cfg
}

// MustGet returns the global Config, panicking if initialization fails.
func MustGet() *Config {
	c := Get()
	if c == nil {
		panic("config: failed to initialize")
	}
	return c
}

// SetForTesting sets a config override for testing purposes.
// This should only be used in tests.
func SetForTesting(root string) {
	overrides = &Config{
		Root:    root,
		HomeDir: osGetHomeDir(),
		OS:      runtime.GOOS,
		Arch:    runtime.GOARCH,
	}
}

// ResetForTesting clears any test override and resets the singleton.
func ResetForTesting() {
	overrides = nil
	cfg = nil
	once = sync.Once{}
}

// RootDir returns the phpv root directory.
// Respects PHPV_ROOT env var, falls back to $HOME/.phpv.
func (c *Config) RootDir() string {
	return c.Root
}

// BinPath returns the bin directory within phpv root.
func (c *Config) BinPath() string {
	return filepath.Join(c.Root, "bin")
}

// VersionsPath returns the versions directory.
func (c *Config) VersionsPath() string {
	return filepath.Join(c.Root, "versions")
}

// CachePath returns the cache directory.
func (c *Config) CachePath() string {
	return filepath.Join(c.Root, "cache")
}

// SourcePath returns the source directory.
func (c *Config) SourcePath() string {
	return filepath.Join(c.Root, "sources")
}

// PharPath returns the phar directory.
func (c *Config) PharPath() string {
	return filepath.Join(c.Root, "phar")
}

// DefaultFilePath returns the path to the default version file.
func (c *Config) DefaultFilePath() string {
	return filepath.Join(c.Root, "default")
}

func initConfig() {
	cfg = &Config{
		HomeDir: osGetHomeDir(),
		OS:      runtime.GOOS,
		Arch:    runtime.GOARCH,
	}

	// PHPV_ROOT env var takes precedence
	if root := os.Getenv("PHPV_ROOT"); root != "" {
		cfg.Root = root
	} else {
		cfg.Root = filepath.Join(cfg.HomeDir, ".phpv")
	}
}

func osGetHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		// Fallback for systems where UserHomeDir fails
		home = os.Getenv("HOME")
		if home == "" {
			home = "/root"
		}
	}
	return home
}
