package domain

// ConfigData is the persisted configuration document.
type ConfigData struct {
	CacheDir      string `toml:"cache_dir,omitempty"`
	Concurrency   int    `toml:"concurrency,omitempty"`
	Compiler      string `toml:"compiler,omitempty"`
	Mirror        string `toml:"mirror,omitempty"`
	StaticLibGCC  bool   `toml:"static_libgcc,omitempty"`
	Server        string `toml:"server,omitempty"`
	ServerVersion string `toml:"server_version,omitempty"`
	ServerSAPI    string `toml:"server_sapi,omitempty"`
}
