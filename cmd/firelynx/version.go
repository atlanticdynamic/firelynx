package main

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
)

// Version is set during build using ldflags
var Version = "dev"

var versionCmd = &cli.Command{
	Name:  "version",
	Usage: "Print the version information",
	Action: func(ctx context.Context, cmd *cli.Command) error {
		fmt.Printf("firelynx version %s\n", cmd.Root().Version)
		return nil
	},
}
