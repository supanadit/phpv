package disk

import (
	"github.com/spf13/afero"
	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/download"
	"github.com/supanadit/phpv/source"
	"github.com/supanadit/phpv/unload"
)

type ForgeRepository struct {
	downloadRepo download.DownloadRepository
	unloadRepo   unload.UnloadRepository
	siloRepo     *SiloRepository
	sourceRepo   source.SourceRepository
	fs           afero.Fs
}

func NewForgeRepository(downloadRepo download.DownloadRepository, unloadRepo unload.UnloadRepository, siloRepo *SiloRepository, sourceRepo source.SourceRepository) *ForgeRepository {
	return &ForgeRepository{
		downloadRepo: downloadRepo,
		unloadRepo:   unloadRepo,
		siloRepo:     siloRepo,
		sourceRepo:   sourceRepo,
		fs:           afero.NewOsFs(),
	}
}

func (r *ForgeRepository) Build(config domain.ForgeConfig) (domain.Forge, error) {
	strategy := r.detectStrategy(config.Name, config.Version)
	return r.BuildWithStrategy(config, strategy)
}
