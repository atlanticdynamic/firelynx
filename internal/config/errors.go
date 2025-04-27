package config

import "errors"

var (
	ErrFailedToLoadConfig     = errors.New("failed to load config")
	ErrFailedToConvertConfig  = errors.New("failed to convert config from proto")
	ErrFailedToValidateConfig = errors.New("failed to validate config")
	ErrUnsupportedConfigVer   = errors.New("unsupported config version")
)
