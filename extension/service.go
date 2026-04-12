package extension

import (
	"errors"

	"github.com/supanadit/phpv/domain"
)

var ErrUnknownExtension = errors.New("unknown extension")
var ErrExtensionConflict = errors.New("extension conflict")

type Repository interface {
	GetExtensionDef(name string) (domain.ExtensionDef, bool)
	IsExtensionValidForPHPVersion(name string, phpVersion string) bool
	GetConflictingExtensions(name string) []string
	GetExtensionDependency(name string) (string, bool)
	GetExtensionDependencyWithVersion(extName, phpVersion string) (string, string, bool)
	ValidateExtensions(extensions []string, phpVersion string) ([]string, error)
	CheckExtensionConflicts(extensions []string) ([]string, [][]string)
	ListExtensions() []domain.ExtensionInfo
	ListExtensionsForPHP(phpVersion string) []domain.ExtensionInfo
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetExtensionDef(name string) (domain.ExtensionDef, bool) {
	return s.repo.GetExtensionDef(name)
}

func (s *Service) IsExtensionValidForPHPVersion(name string, phpVersion string) bool {
	return s.repo.IsExtensionValidForPHPVersion(name, phpVersion)
}

func (s *Service) GetConflictingExtensions(name string) []string {
	return s.repo.GetConflictingExtensions(name)
}

func (s *Service) GetExtensionDependency(name string) (string, bool) {
	return s.repo.GetExtensionDependency(name)
}

func (s *Service) GetExtensionDependencyWithVersion(extName, phpVersion string) (string, string, bool) {
	return s.repo.GetExtensionDependencyWithVersion(extName, phpVersion)
}

func (s *Service) ValidateExtensions(extensions []string, phpVersion string) error {
	unknown, err := s.repo.ValidateExtensions(extensions, phpVersion)
	if err != nil {
		return err
	}
	if len(unknown) > 0 {
		return ErrUnknownExtension
	}
	return nil
}

func (s *Service) CheckExtensionConflicts(extensions []string) ([]string, [][]string, error) {
	conflicts, conflictPairs := s.repo.CheckExtensionConflicts(extensions)
	if len(conflicts) > 0 {
		return conflicts, conflictPairs, ErrExtensionConflict
	}
	return nil, nil, nil
}
