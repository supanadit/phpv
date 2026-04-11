package flagresolver

import "errors"

var ErrUnknownExtension = errors.New("unknown extension")
var ErrExtensionConflict = errors.New("extension conflict")

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
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
