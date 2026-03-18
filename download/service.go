package download

type DownloadRepository interface{}

type Service struct {
	downloadRepository DownloadRepository
}

func NewService(downloadRepository DownloadRepository) *Service {
	return &Service{
		downloadRepository: downloadRepository,
	}
}
