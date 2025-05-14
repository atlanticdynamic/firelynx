package toml

import (
	"errors"

	"github.com/atlanticdynamic/firelynx/internal/config/errz"
)

var (
	ErrNoSourceData         = errors.New("no source data provided")
	ErrParseToml            = errors.New("failed to parse TOML")
	ErrJsonConversion       = errors.New("failed to convert TOML to JSON")
	ErrUnmarshalProto       = errors.New("failed to unmarshal proto")
	ErrPostProcessConfig    = errors.New("failed to post-process config")
	ErrUnsupportedConfigVer = errz.ErrUnsupportedConfigVer
)
