package domain

import (
	"time"
)

const (
	DefaultPHPVRoot = ".phpv"
	VersionsDir     = "versions"
	SourceDirName   = "src"
	LedgerDir       = "ledger"
)

type Build struct {
	Version     string     `json:"version"`
	Prefix      string     `json:"prefix"`
	SourceDir   string     `json:"source_dir"`
	Configured  bool       `json:"configured"`
	Compiled    bool       `json:"compiled"`
	Installed   bool       `json:"installed"`
	StartedAt   time.Time  `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

type Ledger struct {
	Version    string `json:"version"`
	Configured bool   `json:"configured"`
	Compiled   bool   `json:"compiled"`
	Installed  bool   `json:"installed"`
}
