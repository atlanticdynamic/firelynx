package mcp_test

import (
	_ "embed"
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/stretchr/testify/require"
)

//go:embed mcp-typed-builtins.toml
var typedBuiltinsConfig []byte

// TestTypedBuiltinsConfigLoading verifies the typed-builtins example config
// loads and validates without modification.
func TestTypedBuiltinsConfigLoading(t *testing.T) {
	t.Parallel()

	cfg, err := config.NewConfigFromBytes(typedBuiltinsConfig)
	require.NoError(t, err, "should load typed-builtins config from embedded bytes")
	require.NotNil(t, cfg, "config should not be nil")
	require.NoError(t, cfg.Validate(), "typed-builtins config should validate")
}
