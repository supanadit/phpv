package domain

// Forge is the result of a successful build. It carries the installation
// prefix and any environment variables that downstream packages need to
// link against this dependency (PKG_CONFIG_PATH, CPPFLAGS, LDFLAGS, etc.).
type Forge struct {
	Prefix string
	Env    map[string]string
}