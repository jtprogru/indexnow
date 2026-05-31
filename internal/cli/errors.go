package cli

import "errors"

var (
	ErrSourceConflict = errors.New("indexnow cli: choose exactly one of args, --file, --stdin")
	ErrNoSource       = errors.New("indexnow cli: no URL source provided")
	ErrNoURLs         = errors.New("indexnow cli: no URLs found in input")
	ErrInvalidOutput  = errors.New("indexnow cli: invalid --output value")
	ErrInvalidFailOn  = errors.New("indexnow cli: invalid --fail-on value")
)
