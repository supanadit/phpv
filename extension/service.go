// Package extension owns the extension knowledge use case: which extensions
// exist, which ones apply to a PHP version, and what flags they need.
package extension

import (
	"github.com/supanadit/phpv/domain"
)

// ExtensionCatalog is the extension-catalog port. The memory driver implements it;
// a future REST or SQL driver can replace it without changing this module.
type ExtensionCatalog interface {
	GetExtensionDef(name string) (domain.ExtensionDef, bool)
	IsExtensionValidForPHPVersion(name string, phpVersion string) bool
	GetConflictingExtensions(name string) []string
	GetExtensionDependency(name string) (string, bool)
	GetExtensionDependencyWithVersion(extName string, phpVersion string) (string, string, bool)
	ValidateExtensions(extensions []string, phpVersion string) ([]string, error)
	CheckExtensionConflicts(extensions []string) ([]string, [][]string)
	ListExtensions() []domain.ExtensionInfo
	ListExtensionsForPHP(phpVersion string) []domain.ExtensionInfo
	ExpandImplied(extensions []string) (expanded []string, added []string)
	DefaultExtensions(phpVersion string) (included []string, skipped []string)
	SharedOnlyExtensions(phpVersion string, requested []string) []string
	GetPHPConfigureFlags(phpVersion string, extensions []string) []string
	GetExtensionConfigureFlags(name string, phpVersion string) []string
}

// Service exposes extension knowledge to the rest of the application.
type Service struct{ repo ExtensionCatalog }

// NewService creates an extension service over the given catalog.
func NewService(repo ExtensionCatalog) *Service { return &Service{repo: repo} }

func (s *Service) GetExtensionDef(name string) (domain.ExtensionDef, bool) {
	return s.repo.GetExtensionDef(name)
}
func (s *Service) IsExtensionValidForPHPVersion(name, phpVersion string) bool {
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
func (s *Service) ValidateExtensions(extensions []string, phpVersion string) ([]string, error) {
	return s.repo.ValidateExtensions(extensions, phpVersion)
}
func (s *Service) CheckExtensionConflicts(extensions []string) ([]string, [][]string) {
	return s.repo.CheckExtensionConflicts(extensions)
}
func (s *Service) ListExtensions() []domain.ExtensionInfo { return s.repo.ListExtensions() }
func (s *Service) ListExtensionsForPHP(phpVersion string) []domain.ExtensionInfo {
	return s.repo.ListExtensionsForPHP(phpVersion)
}
func (s *Service) ExpandImplied(extensions []string) (expanded []string, added []string) {
	return s.repo.ExpandImplied(extensions)
}
func (s *Service) DefaultExtensions(phpVersion string) ([]string, []string) {
	return s.repo.DefaultExtensions(phpVersion)
}
func (s *Service) SharedOnlyExtensions(phpVersion string, requested []string) []string {
	return s.repo.SharedOnlyExtensions(phpVersion, requested)
}
func (s *Service) GetPHPConfigureFlags(phpVersion string, extensions []string) []string {
	return s.repo.GetPHPConfigureFlags(phpVersion, extensions)
}
func (s *Service) GetExtensionConfigureFlags(name, phpVersion string) []string {
	return s.repo.GetExtensionConfigureFlags(name, phpVersion)
}
