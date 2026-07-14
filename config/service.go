package config

import (
	"fmt"
	"strings"
	"sync"
)

type Data struct {
	CacheDir     string `toml:"cache_dir,omitempty"`
	Concurrency  int    `toml:"concurrency,omitempty"`
	Compiler     string `toml:"compiler,omitempty"`
	Mirror       string `toml:"mirror,omitempty"`
	StaticLibGCC bool   `toml:"static_libgcc,omitempty"`
}

type ConfigRepository interface {
	Path() string
	Load() (Data, error)
	Save(data Data) error
}

type keyDef struct {
	key  string
	typ  string
	desc string
}

var knownKeys = []keyDef{
	{key: "cache_dir", typ: "string", desc: "Download cache directory path"},
	{key: "concurrency", typ: "int", desc: "Number of parallel build jobs (default: CPU count)"},
	{key: "compiler", typ: "string", desc: "C compiler to use (e.g., gcc, clang)"},
	{key: "mirror", typ: "string", desc: "PHP download mirror URL"},
	{key: "static_libgcc", typ: "bool", desc: "Statically link libgcc in static builds"},
}

var knownMap map[string]keyDef

func init() {
	knownMap = make(map[string]keyDef, len(knownKeys))
	for _, k := range knownKeys {
		knownMap[k.key] = k
	}
}

type Service struct {
	mu   sync.Mutex
	repo ConfigRepository
	data Data
}

func NewService(repo ConfigRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) load() error {
	data, err := s.repo.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	s.data = data
	return nil
}

func (s *Service) save() error {
	return s.repo.Save(s.data)
}

func (s *Service) Get(key string) (string, error) {
	def, ok := knownMap[key]
	if !ok {
		return "", fmt.Errorf("unknown config key: %q (use `phpv config list` to see available keys)", key)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.load(); err != nil {
		return "", err
	}

	switch def.typ {
	case "string":
		switch key {
		case "cache_dir":
			return s.data.CacheDir, nil
		case "compiler":
			return s.data.Compiler, nil
		case "mirror":
			return s.data.Mirror, nil
		}
	case "int":
		switch key {
		case "concurrency":
			if s.data.Concurrency == 0 {
				return "", nil
			}
			return fmt.Sprintf("%d", s.data.Concurrency), nil
		}
	case "bool":
		switch key {
		case "static_libgcc":
			return fmt.Sprintf("%t", s.data.StaticLibGCC), nil
		}
	}
	return "", nil
}

func (s *Service) Set(key, value string) error {
	def, ok := knownMap[key]
	if !ok {
		return fmt.Errorf("unknown config key: %q (use `phpv config list` to see available keys)", key)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.load(); err != nil {
		return err
	}

	switch def.typ {
	case "string":
		if value == "" {
			return fmt.Errorf("value for %q must not be empty", key)
		}
		switch key {
		case "cache_dir":
			s.data.CacheDir = value
		case "compiler":
			s.data.Compiler = value
		case "mirror":
			s.data.Mirror = value
		}
	case "int":
		n, err := parseInt(value)
		if err != nil {
			return fmt.Errorf("value for %q must be an integer: %w", key, err)
		}
		if n < 1 {
			return fmt.Errorf("value for %q must be positive", key)
		}
		switch key {
		case "concurrency":
			s.data.Concurrency = n
		}
	case "bool":
		b, err := parseBool(value)
		if err != nil {
			return fmt.Errorf("value for %q must be true or false: %w", key, err)
		}
		switch key {
		case "static_libgcc":
			s.data.StaticLibGCC = b
		}
	}

	return s.save()
}

func (s *Service) List() ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.load(); err != nil {
		return nil, err
	}

	var lines []string
	for _, k := range knownKeys {
		v := ""
		switch k.typ {
		case "string":
			switch k.key {
			case "cache_dir":
				v = s.data.CacheDir
			case "compiler":
				v = s.data.Compiler
			case "mirror":
				v = s.data.Mirror
			}
		case "int":
			switch k.key {
			case "concurrency":
				if s.data.Concurrency > 0 {
					v = fmt.Sprintf("%d", s.data.Concurrency)
				}
			}
		case "bool":
			switch k.key {
			case "static_libgcc":
				v = fmt.Sprintf("%t", s.data.StaticLibGCC)
			}
		}
		if v == "" {
			v = "(unset)"
		}
		lines = append(lines, fmt.Sprintf("%-20s %-10s %s", k.key, v, k.desc))
	}
	return lines, nil
}

func parseInt(s string) (int, error) {
	var n int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("invalid integer: %q", s)
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
}

func parseBool(s string) (bool, error) {
	switch strings.ToLower(s) {
	case "true", "yes", "1", "on":
		return true, nil
	case "false", "no", "0", "off":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean: %q", s)
	}
}
