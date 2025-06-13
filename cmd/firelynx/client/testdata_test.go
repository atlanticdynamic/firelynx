//go:build e2e
// +build e2e

package client

import _ "embed"

//go:embed testdata/test_config.toml
var testConfigContent string

//go:embed testdata/updated_config.toml
var updatedConfigContent string
