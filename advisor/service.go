package forge

type AdvisorRepository interface {
}

type Service struct {
	advisorRepository AdvisorRepository
}

func NewService(advisorRepository AdvisorRepository) *Service {
	return &Service{
		advisorRepository: advisorRepository,
	}
}
