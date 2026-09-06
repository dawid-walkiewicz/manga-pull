package db

import "errors"

var (
	ErrNotFound               = errors.New("not found")
	ErrInvalidStateTransition = errors.New("invalid state transition")
)
