package disk

import (
	"github.com/supanadit/phpv/bundler"
	"github.com/supanadit/phpv/domain"
)

type bundlerRepository struct {
	service *bundler.BundlerService
}

func NewBundlerRepository(cfg bundler.BundlerServiceConfig) bundler.BundlerRepository {
	return bundler.NewBundlerService(cfg)
}

func BundlerServiceConfigFromSilo(silo *domain.Silo) bundler.BundlerServiceConfig {
	return bundler.BundlerServiceConfig{
		Silo: silo,
	}
}
