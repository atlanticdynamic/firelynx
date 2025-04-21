package client

import "errors"

var (
	ErrInvalidAddressFormat = errors.New("invalid server address format")
	ErrInvalidTCPFormat     = errors.New("invalid TCP address format")
	ErrUnsupportedNetwork   = errors.New("unsupported network type")
)
