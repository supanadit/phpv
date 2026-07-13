package shim

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/supanadit/phpv/silo"
)

type shimKind int

const (
	kindBinary shimKind = iota
	kindPhar
)

type shimDef struct {
	Name    string
	Kind    shimKind
	PharRel string
}

var versionRegex = regexp.MustCompile(`^[0-9]+(\.[0-9]+)*$`)

type Service struct {
	silo *silo.Service
	bin  string
}

func NewService(siloSvc *silo.Service) *Service {
	return &Service{
		silo: siloSvc,
		bin:  filepath.Join(siloSvc.GetSilo().Root, "bin"),
	}
}

func (s *Service) IsSystemMode() bool {
	return s.silo.IsSystemMode()
}

func (s *Service) SetSystemMode(enabled bool) error {
	return s.silo.SetSystemMode(enabled)
}

func (s *Service) WriteShim(def shimDef) error {
	content, err := s.renderShim(def)
	if err != nil {
		return err
	}
	path := filepath.Join(s.bin, def.Name)
	if err := os.MkdirAll(s.bin, 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o755)
}

func (s *Service) renderShim(def shimDef) (string, error) {
	var systemBlock, execLine string
	switch def.Kind {
	case kindBinary:
		systemBlock = strings.ReplaceAll(systemBinaryBlock, "{{NAME}}", def.Name)
		execLine = strings.ReplaceAll(binaryExec, "{{NAME}}", def.Name)
	case kindPhar:
		systemBlock = strings.ReplaceAll(systemPharBlock, "{{NAME}}", def.Name)
		systemBlock = strings.ReplaceAll(systemBlock, "{{PHAR_REL}}", def.PharRel)
		execLine = strings.ReplaceAll(pharExec, "{{PHAR_REL}}", def.PharRel)
	default:
		return "", fmt.Errorf("unknown shim kind")
	}
	out := strings.Replace(shimTemplate, "{{SYSTEM_BLOCK}}", systemBlock, 1)
	out = strings.Replace(out, "{{EXEC}}", execLine, 1)
	return out, nil
}

func (s *Service) WriteAll() error {
	for _, def := range defaultShims() {
		if err := s.WriteShim(def); err != nil {
			return fmt.Errorf("write shim %s: %w", def.Name, err)
		}
	}
	return nil
}

func (s *Service) WritePhar(name, pharRelPath string) error {
	return s.WriteShim(shimDef{Name: name, Kind: kindPhar, PharRel: pharRelPath})
}

func (s *Service) RegenerateAll() error {
	return s.WriteAll()
}

func defaultShims() []shimDef {
	return []shimDef{
		{Name: "php", Kind: kindBinary},
		{Name: "phpize", Kind: kindBinary},
		{Name: "php-config", Kind: kindBinary},
		{Name: "php-cgi", Kind: kindBinary},
		{Name: "phpdbg", Kind: kindBinary},
	}
}

func IsValidVersion(v string) bool {
	return versionRegex.MatchString(v)
}
