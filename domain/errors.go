package domain

import "errors"

var (
	ErrInternalServerError = errors.New("internal server error")
	ErrNotFound            = errors.New("resource not found")
	ErrConflict            = errors.New("resource already exists")
	ErrBadParamInput       = errors.New("invalid parameter")
	ErrDownloadFailed      = errors.New("download failed")
	ErrBuildFailed         = errors.New("build failed")
	ErrExtensionConflict   = errors.New("extension conflict")
	ErrUnknownExtension    = errors.New("unknown extension")
)
