// Package webserver installs and runs local webservers (caddy, nginx,
// apache) that serve PHP applications through FastCGI or Apache's embedded
// mod_php. Instances are fully isolated per PHP version: each run pins one
// PHP version and gets its own runtime (FPM master + socket, or a mod_php
// module), config set and log directory, so several PHP versions can be
// served simultaneously.
package webserver

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"syscall"
	"time"

	"github.com/supanadit/phpv/domain"
)

// StateDir returns the directory holding server configs, instance state
// and logs under the phpv root.
func StateDir(root string) string {
	return filepath.Join(root, "servers")
}

// Store persists server instances as JSON files under
// <root>/servers/state/. One file per instance ID.
type Store struct {
	dir string
}

// NewStore creates an instance store rooted at the given servers dir.
func NewStore(serversDir string) *Store {
	return &Store{dir: filepath.Join(serversDir, "state")}
}

func (s *Store) path(id string) string {
	return filepath.Join(s.dir, id+".json")
}

// Save writes one instance record.
func (s *Store) Save(inst domain.ServerInstance) error {
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return fmt.Errorf("create state dir: %w", err)
	}
	data, err := json.MarshalIndent(inst, "", "  ")
	if err != nil {
		return fmt.Errorf("encode instance: %w", err)
	}
	return os.WriteFile(s.path(inst.ID), data, 0o644)
}

// Load reads one instance record. Returns (zero, nil) when absent.
func (s *Store) Load(id string) (domain.ServerInstance, error) {
	data, err := os.ReadFile(s.path(id))
	if err != nil {
		if os.IsNotExist(err) {
			return domain.ServerInstance{}, nil
		}
		return domain.ServerInstance{}, fmt.Errorf("read instance %s: %w", id, err)
	}
	var inst domain.ServerInstance
	if err := json.Unmarshal(data, &inst); err != nil {
		return domain.ServerInstance{}, fmt.Errorf("decode instance %s: %w", id, err)
	}
	return inst, nil
}

// List returns all stored instances sorted by start time.
func (s *Store) List() ([]domain.ServerInstance, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("list instances: %w", err)
	}
	var out []domain.ServerInstance
	for _, e := range entries {
		if filepath.Ext(e.Name()) != ".json" {
			continue
		}
		inst, err := s.Load(e.Name()[:len(e.Name())-5])
		if err != nil {
			continue // skip unreadable records
		}
		out = append(out, inst)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].StartedAt.Before(out[j].StartedAt)
	})
	return out, nil
}

// Remove deletes one instance record.
func (s *Store) Remove(id string) error {
	err := os.Remove(s.path(id))
	if err != nil && os.IsNotExist(err) {
		return nil
	}
	return err
}

// ProcessAlive reports whether the given pid refers to a live process.
// A pid of 0 is never alive.
func ProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	// Signal 0 probes for existence without actually signalling. EPERM
	// would mean the process exists but is owned by someone else; treat
	// it as alive to avoid orphaning foreign processes.
	err := syscall.Kill(pid, 0)
	return err == nil || err == syscall.EPERM
}

// Running reports whether the instance's server process is still alive.
func (s *Store) Running(inst domain.ServerInstance) bool {
	return ProcessAlive(inst.ServerPID)
}

// InstanceID builds the deterministic instance id for a run. The SAPI is only
// part of the id when it is not the default FPM one, keeping existing ids.
func InstanceID(server, phpVersion string, port int, sapi string) string {
	if sapi == "" || sapi == "fpm" {
		return fmt.Sprintf("%s-%d-php%s", server, port, phpVersion)
	}
	return fmt.Sprintf("%s-%s-%d-php%s", server, sapi, port, phpVersion)
}

// Timeout for graceful shutdowns before processes get killed.
const stopGrace = 10 * time.Second
