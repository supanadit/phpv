package advisor

import "github.com/supanadit/phpv/domain"

type AdvisorRepository interface {
	Check(name string, version string) (domain.AdvisorCheck, error)
}

type Service struct {
	repo AdvisorRepository
}

func NewAdvisorService(repo AdvisorRepository) *Service {
	return &Service{
		repo: repo,
	}
}

func (s *Service) Check(name string, version string) (domain.AdvisorCheck, error) {
	return s.repo.Check(name, version)
}
