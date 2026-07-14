package disk

import (
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
	"github.com/supanadit/phpv/config"
)

type ConfigRepository struct {
	root string
}

func NewConfigRepository() *ConfigRepository {
	root := resolveRoot()
	return &ConfigRepository{root: root}
}

func (r *ConfigRepository) Path() string {
	return filepath.Join(r.root, "config.toml")
}

func (r *ConfigRepository) Load() (config.Data, error) {
	data, err := os.ReadFile(r.Path())
	if err != nil {
		if os.IsNotExist(err) {
			return config.Data{}, nil
		}
		return config.Data{}, err
	}
	var cfg config.Data
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return config.Data{}, err
	}
	return cfg, nil
}

func (r *ConfigRepository) Save(data config.Data) error {
	raw, err := toml.Marshal(data)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(r.Path()), 0o755); err != nil {
		return err
	}
	return os.WriteFile(r.Path(), raw, 0o644)
}
