package domain

type Forge struct {
	Prefix string            `json:"prefix"`
	Env    map[string]string `json:"env"`
}
