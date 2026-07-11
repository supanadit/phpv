package silo

import (
	"runtime"

	"github.com/supanadit/phpv/registry"
)

// SiloRepository is the low-level downloader responsible for fetching a
// file from the given URL and storing it in the silo (local storage).
// When checksumType and checksumValue are non-empty the implementation is
// expected to verify the downloaded file against them.
//
// Download returns true when the file was actually fetched from the network,
// and false when the file already existed (skipped).
type SiloRepository interface {
	Download(url string, checksumType string, checksumValue string) (downloaded bool, err error)
}

type Service struct {
	siloRep     SiloRepository
	registryRep registry.RegistryRepository
}

func NewService(sr SiloRepository, rr registry.RegistryRepository) *Service {
	return &Service{
		siloRep:     sr,
		registryRep: rr,
	}
}

// Download resolves the registry entry for the given name and version and
// then delegates the actual download to the SiloRepository. The registry
// entry provides the download URL and, when available, the checksum used
// to verify the integrity of the downloaded file.
func (s *Service) Download(name string, version string) (bool, error) {
	// checksum=false for now — we skip verification until checksums
	// are populated for all packages.
	r, err := s.registryRep.Get(name, version, false, runtime.GOOS)
	if err != nil {
		return false, err
	}
	return s.siloRep.Download(r.URL, r.ChecksumType, r.ChecksumValue)
}