package download

import (
	"github.com/supanadit/phpv/domain"
)

type DownloadRepository interface {
	Download(url, destination string) (*domain.Download, error)
	DownloadWithFallbacks(urls []string, destination string, options ...DownloadOption) (*domain.Download, error)
}

type DownloadOption func(*DownloadOptions)

type DownloadOptions struct {
	Checksum   string
	MaxRetries int
	RetryDelay int // milliseconds
}

func WithChecksum(checksum string) DownloadOption {
	return func(o *DownloadOptions) {
		o.Checksum = checksum
	}
}

func WithMaxRetries(retries int) DownloadOption {
	return func(o *DownloadOptions) {
		o.MaxRetries = retries
	}
}

func WithRetryDelay(delayMs int) DownloadOption {
	return func(o *DownloadOptions) {
		o.RetryDelay = delayMs
	}
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

func (s *Service) DownloadWithFallbacks(urls []string, destination string, options ...DownloadOption) (*domain.Download, error) {
	return s.downloadRepository.DownloadWithFallbacks(urls, destination, options...)
}
