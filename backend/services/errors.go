package services

import "errors"

var (
	ErrPluginNotFound    = errors.New("plugin not found")
	ErrPluginDisabled    = errors.New("plugin disabled")
	ErrPluginRuntime     = errors.New("plugin runtime unavailable")
	ErrPluginFailedFetch = errors.New("failed to fetch title from plugin")
	ErrTitleAlreadySaved = errors.New("title already saved")
)
