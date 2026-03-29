package flagresolver

import "github.com/supanadit/phpv/domain"

type Service struct {
	repo domain.FlagResolverRepository
}

func NewService(repo domain.FlagResolverRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetConfigureFlags(name string) []string {
	return s.repo.GetConfigureFlags(name)
}

func (s *Service) GetPHPConfigureFlags(phpVersion string, extensions []string) []string {
	return s.repo.GetPHPConfigureFlags(phpVersion, extensions)
}
