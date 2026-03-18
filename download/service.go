package download

import (
	"github.com/supanadit/phpv/domain"
)

type DownloadRepository interface {
	Exists(url string) (*domain.FileInfo, error)
	Download(url, destination string) (*domain.Download, error)
}

type Service struct {
	downloadRepository DownloadRepository
}

func NewService(downloadRepository DownloadRepository) *Service {
	return &Service{
		downloadRepository: downloadRepository,
	}
}

func (s *Service) Exists(url string) (*domain.FileInfo, error) {
	return s.downloadRepository.Exists(url)
}

func (s *Service) Download(url, destination string) (*domain.Download, error) {
	return s.downloadRepository.Download(url, destination)
}
