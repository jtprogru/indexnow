package client

import "errors"

var (
	ErrNotImplemented  = errors.New("indexnow: not implemented")
	ErrUnknownEndpoint = errors.New("indexnow: unknown endpoint")
	ErrInvalidKey      = errors.New("indexnow: invalid key")
	ErrEmptyURLList    = errors.New("indexnow: empty url list")
	ErrMissingHost     = errors.New("indexnow: missing host")
	ErrMissingEndpoint = errors.New("indexnow: missing endpoint")
)
