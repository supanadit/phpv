package download

import (
	"github.com/supanadit/phpv/domain"
)

type DownloadRepository interface {
	Download(url, destination string) (*domain.Download, error)
	DownloadWithFallbacks(urls []string, destination string) (*domain.Download, error)
}

type Service struct {
	downloadRepository DownloadRepository
}

func NewService(downloadRepository DownloadRepository) *Service {
	return &Service{
		downloadRepository: downloadRepository,
	}
}

func (s *Service) Download(url, destination string) (*domain.Download, error) {
	return s.downloadRepository.Download(url, destination)
}

func (s *Service) DownloadWithFallbacks(urls []string, destination string) (*domain.Download, error) {
	return s.downloadRepository.DownloadWithFallbacks(urls, destination)
}
