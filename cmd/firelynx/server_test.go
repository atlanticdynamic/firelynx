package main

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"
)

// TestServerCmd_EmptyFlags verifies that running the server with no flags returns an error
func TestServerCmd_EmptyFlags(t *testing.T) {
	t.Parallel()
	// Create a command with empty flags
	cmd := &cli.Command{
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "config"},
			&cli.StringFlag{Name: "listen"},
		},
	}

	// Call the command action directly
	result := serverCmd.Action(context.Background(), cmd)

	// Verify we get the expected error
	var exitErr cli.ExitCoder
	ok := errors.As(result, &exitErr)
	require.True(t, ok, "Expected cli.ExitCoder, got %T", result)
	assert.Equal(t, 1, exitErr.ExitCode())
	assert.Equal(t, "either --config or --listen flag is required", exitErr.Error())
}
