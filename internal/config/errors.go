package config

import "errors"

var (
	ErrFailedToLoadConfig     = errors.New("failed to load config")
	ErrFailedToValidateConfig = errors.New("failed to validate config")
	ErrUnsupportedConfigVer   = errors.New("unsupported config version")
)
