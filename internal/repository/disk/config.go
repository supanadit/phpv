package disk

import (
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/internal/repository"
)

type ConfigRepository struct {
	root string
}

func NewConfigRepository() *ConfigRepository {
	root := repository.ResolveRoot()
	return &ConfigRepository{root: root}
}

func (r *ConfigRepository) Path() string {
	return filepath.Join(r.root, "config.toml")
}

func (r *ConfigRepository) Load() (domain.ConfigData, error) {
	data, err := os.ReadFile(r.Path())
	if err != nil {
		if os.IsNotExist(err) {
			return domain.ConfigData{}, nil
		}
		return domain.ConfigData{}, err
	}
	var cfg domain.ConfigData
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return domain.ConfigData{}, err
	}
	return cfg, nil
}

func (r *ConfigRepository) Save(data domain.ConfigData) error {
	raw, err := toml.Marshal(data)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(r.Path()), 0o755); err != nil {
		return err
	}
	return os.WriteFile(r.Path(), raw, 0o644)
}
