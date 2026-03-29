package domain

type Dependency struct {
	Name     string
	Version  string
	Optional bool
}

type DependencyGraph map[string][]Dependency
