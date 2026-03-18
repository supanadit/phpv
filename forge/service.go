package forge

import (
	"github.com/supanadit/phpv/internal/repository/disk"
)

type Service struct {
	forgeRepository disk.ForgeRepository
	buildRepository disk.BuildRepository
}

func NewService(forgeRepository disk.ForgeRepository, buildRepository disk.BuildRepository) *Service {
	return &Service{
		forgeRepository: forgeRepository,
		buildRepository: buildRepository,
	}
}

func (s *Service) GetConfigureFlags(version string) ([]string, bool) {
	return s.forgeRepository.GetConfigureFlags(version)
}

func (s *Service) ExpandConfigureFlags(version string) ([]string, bool) {
	return s.forgeRepository.ExpandConfigureFlags(version)
}

func (s *Service) GetBuildPrefix(version string) string {
	return s.forgeRepository.BuildPrefix(version)
}

func (s *Service) GetSourcePath(version string) string {
	return s.forgeRepository.SourcePath(version)
}

func (s *Service) Configure(sourceDir string, flags []string) error {
	return s.buildRepository.Configure(sourceDir, flags)
}

func (s *Service) Make(sourceDir string, jobs int) error {
	return s.buildRepository.Make(sourceDir, jobs)
}

func (s *Service) Install(sourceDir string) error {
	return s.buildRepository.Install(sourceDir)
}

func (s *Service) Distclean(sourceDir string) error {
	return s.buildRepository.Distclean(sourceDir)
}
