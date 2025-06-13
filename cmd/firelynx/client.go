package main

import (
	"context"
	"time"

	"github.com/atlanticdynamic/firelynx/cmd/firelynx/client"
	"github.com/urfave/cli/v3"
)

// Shared flag definitions
var (
	serverFlag = &cli.StringFlag{
		Name:     "server",
		Usage:    "Server address (tcp://host:port or unix:///path/to/socket)",
		Aliases:  []string{"s"},
		Required: true,
		Value:    "localhost:9999",
	}

	formatConfigFlag = &cli.StringFlag{
		Name:    "format",
		Usage:   "Output format: text (summary), json (config data), toml (config data)",
		Aliases: []string{"f"},
		Value:   "toml",
	}

	formatStorageFlag = &cli.StringFlag{
		Name:    "format",
		Usage:   "Output format: text (summary), json (config data), toml (config data)",
		Aliases: []string{"f"},
		Value:   "text",
	}

	pageSizeFlag = &cli.IntFlag{
		Name:  "page-size",
		Usage: "Number of transactions per page",
		Value: 10,
	}

	pageTokenFlag = &cli.StringFlag{
		Name:  "page-token",
		Usage: "Token for pagination",
	}

	stateFlag = &cli.StringFlag{
		Name:  "state",
		Usage: "Filter by transaction state",
	}

	sourceFlag = &cli.StringFlag{
		Name:  "source",
		Usage: "Filter by transaction source",
	}

	transactionIDFlag = &cli.StringFlag{
		Name:     "id",
		Usage:    "Transaction ID",
		Required: true,
	}

	keepLastFlag = &cli.IntFlag{
		Name:  "keep-last",
		Usage: "Number of recent transactions to keep",
		Value: 1,
	}
)

var clientCmd = &cli.Command{
	Name:  "client",
	Usage: "Client operations for firelynx server",
	Description: `Interact with a running firelynx server via gRPC.

  Examples:
    firelynx client apply --config myconfig.toml --server localhost:9999
    firelynx client config current --server localhost:9999 --output config.toml
    firelynx client config current --server localhost:9999 --format json
    firelynx client config rollback --server localhost:9999 --id <TRANSACTION_ID>
    firelynx client config storage list --server localhost:9999 --page-size 5
    firelynx client config storage clear --server localhost:9999 --keep-last 3`,
	Commands: []*cli.Command{
		{
			Name:  "apply",
			Usage: "Apply configuration to the server",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     "config",
					Usage:    "Path to TOML configuration file",
					Aliases:  []string{"c"},
					Required: true,
				},
				serverFlag,
				&cli.IntFlag{
					Name:    "timeout",
					Usage:   "Timeout for the operation in seconds",
					Aliases: []string{"t"},
					Value:   5,
				},
			},
			Action: clientApplyAction,
		},
		{
			Name:        "config",
			Usage:       "Configuration operations",
			Description: `Manage server configuration and transaction storage.`,
			Commands: []*cli.Command{
				{
					Name:  "current",
					Usage: "Get current configuration from the server",
					Description: `Show the current active configuration.

  Examples:
    firelynx client config current --server localhost:9999
    firelynx client config current --server localhost:9999 --format json
    firelynx client config current --server localhost:9999 --output config.toml`,
					Flags: []cli.Flag{
						serverFlag,
						formatConfigFlag,
						&cli.StringFlag{
							Name:    "output",
							Usage:   "Path to save configuration (if not provided, print to stdout)",
							Aliases: []string{"o"},
						},
					},
					Action: configCurrentAction,
				},
				{
					Name:  "rollback",
					Usage: "Roll back to a previous configuration transaction",
					Description: `Apply the configuration from a previous transaction.

  Examples:
    firelynx client config rollback --server localhost:9999 --id <TRANSACTION_ID>`,
					Flags: []cli.Flag{
						serverFlag,
						transactionIDFlag,
					},
					Action: storageRollbackAction,
				},
				{
					Name:        "storage",
					Usage:       "Configuration transaction storage operations",
					Description: `Manage configuration transaction history.`,
					Commands: []*cli.Command{
						{
							Name:  "list",
							Usage: "List configuration transactions",
							Description: `List configuration transaction history.

  Examples:
    firelynx client config storage list --server localhost:9999
    firelynx client config storage list --server localhost:9999 --page-size 5
    firelynx client config storage list --server localhost:9999 --state completed
    firelynx client config storage list --server localhost:9999 --format json`,
							Flags: []cli.Flag{
								serverFlag,
								pageSizeFlag,
								pageTokenFlag,
								stateFlag,
								sourceFlag,
								formatStorageFlag,
							},
							Action: storageListAction,
						},
						{
							Name:  "get",
							Usage: "Get a specific configuration transaction by ID",
							Description: `Get details for a specific transaction.

  Examples:
    firelynx client config storage get --server localhost:9999 --id <ID>
    firelynx client config storage get --server localhost:9999 --id <ID> --format toml > backup.toml`,
							Flags: []cli.Flag{
								serverFlag,
								transactionIDFlag,
								formatStorageFlag,
							},
							Action: storageGetAction,
						},
						{
							Name:  "clear",
							Usage: "Clear old configuration transactions",
							Description: `Remove old transaction records.

  Examples:
    firelynx client config storage clear --server localhost:9999 --keep-last 3
    firelynx client config storage clear --server localhost:9999 --keep-last 0`,
							Flags: []cli.Flag{
								serverFlag,
								keepLastFlag,
							},
							Action: storageClearAction,
						},
					},
				},
			},
		},
	},
}

func clientApplyAction(ctx context.Context, cmd *cli.Command) error {
	configPath := cmd.String("config")
	serverAddr := cmd.String("server")
	timeout := time.Duration(cmd.Int("timeout")) * time.Second

	if err := client.ApplyConfig(ctx, configPath, serverAddr, timeout); err != nil {
		return cli.Exit(err.Error(), 1)
	}

	return nil
}

func configCurrentAction(ctx context.Context, cmd *cli.Command) error {
	serverAddr := cmd.String("server")
	format := cmd.String("format")
	outputPath := cmd.String("output")

	if err := client.GetCurrentConfig(ctx, serverAddr, format, outputPath); err != nil {
		return cli.Exit(err.Error(), 1)
	}

	return nil
}

func storageListAction(ctx context.Context, cmd *cli.Command) error {
	serverAddr := cmd.String("server")
	pageSize := int32(cmd.Int("page-size"))
	pageToken := cmd.String("page-token")
	state := cmd.String("state")
	source := cmd.String("source")
	format := cmd.String("format")

	if err := client.ListTransactions(ctx, serverAddr, pageSize, pageToken, state, source, format); err != nil {
		return cli.Exit(err.Error(), 1)
	}

	return nil
}

func storageGetAction(ctx context.Context, cmd *cli.Command) error {
	serverAddr := cmd.String("server")
	transactionID := cmd.String("id")
	format := cmd.String("format")

	if err := client.GetTransaction(ctx, serverAddr, transactionID, format); err != nil {
		return cli.Exit(err.Error(), 1)
	}

	return nil
}

func storageClearAction(ctx context.Context, cmd *cli.Command) error {
	serverAddr := cmd.String("server")
	keepLast := int32(cmd.Int("keep-last"))

	if err := client.ClearTransactions(ctx, serverAddr, keepLast); err != nil {
		return cli.Exit(err.Error(), 1)
	}

	return nil
}

func storageRollbackAction(ctx context.Context, cmd *cli.Command) error {
	serverAddr := cmd.String("server")
	transactionID := cmd.String("id")

	if err := client.RollbackToTransaction(ctx, serverAddr, transactionID); err != nil {
		return cli.Exit(err.Error(), 1)
	}

	return nil
}
