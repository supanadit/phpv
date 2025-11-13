package domain

import "errors"

var (
	// ErrInternalServerError will throw if any the Internal Server Error happen
	ErrInternalServerError = errors.New("internal server error")
	// ErrNotFound will throw if the requested item is not exists
	ErrNotFound = errors.New("your requested item is not found")
	// ErrConflict will throw if the current action already exists
	ErrConflict = errors.New("your item already exist")
	// ErrBadParamInput will throw if the given request-body or params is not valid
	ErrBadParamInput = errors.New("given param is not valid")
	// ErrVersionNotInstalled will throw if the version is not installed
	ErrVersionNotInstalled = errors.New("version is not installed")
	// ErrVersionAlreadyActive will throw if trying to activate an already active version
	ErrVersionAlreadyActive = errors.New("version is already active")
)
