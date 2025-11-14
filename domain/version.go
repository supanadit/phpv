package domain

type Version struct {
	Major int
	Minor int
	Patch int
	// Examples: RC1, beta2, alpha3
	Extra string
}
