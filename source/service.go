package source

import "github.com/supanadit/phpv/domain"

type SourceRepository interface {
	GetVersions() ([]domain.Source, error)
}

type Service struct {
	sourceRepository SourceRepository
}

func NewService(sourceRepository SourceRepository) *Service {
	return &Service{
		sourceRepository: sourceRepository,
	}
}

func (s *Service) GetVersions() ([]domain.Source, error) {
	return s.sourceRepository.GetVersions()
}
