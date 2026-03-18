package domain

type Unload struct {
	Source      string
	Destination string
	Extracted   int
}

const (
	UnloadFormatTarGz = ".tar.gz"
	UnloadFormatTarXz = ".tar.xz"
	UnloadFormatZip   = ".zip"
)
