package domain

// Dependency represents a library that PHP depends on
type Dependency struct {
	Name              string
	Version           string
	DownloadURL       string
	ConfigureFlags    []string
	BuildCommands     []string
	RequiresBuildconf bool
	// Dependencies that this dependency needs (for transitive deps)
	Dependencies []string
}

// DependencySet represents all dependencies for a specific PHP version
type DependencySet struct {
	PHPVersion   Version
	Dependencies []Dependency
}
