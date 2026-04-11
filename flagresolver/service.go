package flagresolver

import "github.com/supanadit/phpv/domain"

type Service struct {
	repo domain.FlagResolverRepository
}

func NewService(repo domain.FlagResolverRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetConfigureFlags(name string, version string) []string {
	return s.repo.GetConfigureFlags(name, version)
}

func (s *Service) GetPHPConfigureFlags(phpVersion string, extensions []string) []string {
	return s.repo.GetPHPConfigureFlags(phpVersion, extensions)
}

func (s *Service) ValidateExtensions(extensions []string, phpVersion string) error {
	unknown, err := s.repo.ValidateExtensions(extensions, phpVersion)
	if err != nil {
		return err
	}
	if len(unknown) > 0 {
		return &domain.UnknownExtensionError{Extension: unknown[0]}
	}
	return nil
}

func (s *Service) CheckExtensionConflicts(extensions []string) ([]string, [][]string, error) {
	conflicts, conflictPairs := s.repo.CheckExtensionConflicts(extensions)
	if len(conflicts) > 0 {
		return conflicts, conflictPairs, &domain.ExtensionConflictError{
			Extension:   conflicts[0],
			Conflicting: findConflictingFor(conflictPairs, conflicts[0]),
		}
	}
	return nil, nil, nil
}

func (s *Service) GetExtensionDependency(ext string) (string, bool) {
	return s.repo.GetExtensionDependency(ext)
}

func (s *Service) GetExtensionDependencyWithVersion(ext, phpVersion string) (string, string, bool) {
	return s.repo.GetExtensionDependencyWithVersion(ext, phpVersion)
}

func findConflictingFor(pairs [][]string, ext string) []string {
	var result []string
	for _, pair := range pairs {
		if pair[0] == ext {
			result = append(result, pair[1])
		} else if pair[1] == ext {
			result = append(result, pair[0])
		}
	}
	return result
}
