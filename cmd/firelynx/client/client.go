package client

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/client"
	"github.com/atlanticdynamic/firelynx/internal/config/loader"
	"github.com/pelletier/go-toml/v2"
)

// ApplyConfig applies a configuration file to the server
func ApplyConfig(ctx context.Context, configPath, serverAddr string, timeout time.Duration) error {
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	logger := slog.Default()

	configLoader, err := loader.NewLoaderFromFilePath(configPath)
	if err != nil {
		return err
	}

	firelynxClient := client.New(client.Config{
		Logger:     logger,
		ServerAddr: serverAddr,
	})

	return firelynxClient.ApplyConfig(ctx, configLoader)
}

// GetCurrentConfig retrieves the current configuration with flexible output formats
func GetCurrentConfig(ctx context.Context, serverAddr, format, outputPath string) error {
	if format == "toml" && outputPath != "" {
		return GetConfig(ctx, serverAddr, outputPath)
	}
	return GetCurrentTransaction(ctx, serverAddr, format)
}

// GetConfig retrieves the current configuration from the server
func GetConfig(ctx context.Context, serverAddr, outputPath string) error {
	logger := slog.Default()

	firelynxClient := client.New(client.Config{
		Logger:     logger,
		ServerAddr: serverAddr,
	})

	config, err := firelynxClient.GetConfig(ctx)
	if err != nil {
		return err
	}

	if outputPath != "" {
		return firelynxClient.SaveConfig(config, outputPath)
	}

	configStr, err := firelynxClient.FormatConfig(config)
	if err != nil {
		return err
	}
	fmt.Println(configStr)

	return nil
}

// GetCurrentTransaction gets the current configuration transaction
func GetCurrentTransaction(ctx context.Context, serverAddr, format string) error {
	logger := slog.Default()

	firelynxClient := client.New(client.Config{
		Logger:     logger,
		ServerAddr: serverAddr,
	})

	transaction, err := firelynxClient.GetCurrentConfigTransaction(ctx)
	if err != nil {
		return err
	}

	if transaction == nil {
		fmt.Println("No current transaction found")
		return nil
	}

	switch format {
	case "json":
		if transaction.GetConfig() != nil {
			jsonBytes, err := json.MarshalIndent(transaction.GetConfig(), "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal config to JSON: %w", err)
			}
			fmt.Println(string(jsonBytes))
		} else {
			fmt.Println("{}")
		}
	case "toml":
		if transaction.GetConfig() != nil {
			tomlBytes, err := toml.Marshal(transaction.GetConfig())
			if err != nil {
				return fmt.Errorf("failed to marshal config to TOML: %w", err)
			}
			fmt.Print(string(tomlBytes))
		}
	default: // text format
		fmt.Printf("Transaction ID: %s\n", transaction.GetId())
		fmt.Printf("Source: %s\n", transaction.GetSource())
		fmt.Printf("Source Detail: %s\n", transaction.GetSourceDetail())
		fmt.Printf("Request ID: %s\n", transaction.GetRequestId())
		fmt.Printf("State: %s\n", transaction.GetState())
		fmt.Printf("Valid: %t\n", transaction.GetIsValid())
		if transaction.GetCreatedAt() != nil {
			fmt.Printf("Created: %s\n", transaction.GetCreatedAt().AsTime().Format(time.RFC3339))
		}
		if transaction.GetConfig() != nil {
			fmt.Printf("Config Version: %s\n", transaction.GetConfig().GetVersion())
		}
	}

	return nil
}

// ListTransactions lists configuration transactions with pagination and filtering
func ListTransactions(
	ctx context.Context,
	serverAddr string,
	pageSize int32,
	pageToken, state, source, format string,
) error {
	logger := slog.Default()

	firelynxClient := client.New(client.Config{
		Logger:     logger,
		ServerAddr: serverAddr,
	})

	transactions, nextPageToken, err := firelynxClient.ListConfigTransactions(
		ctx,
		pageToken,
		pageSize,
		state,
		source,
	)
	if err != nil {
		return err
	}

	if len(transactions) == 0 {
		fmt.Println("No transactions found")
		return nil
	}

	switch format {
	case "json":
		response := struct {
			Transactions  any    `json:"transactions"`
			NextPageToken string `json:"nextPageToken,omitempty"`
		}{
			Transactions:  transactions,
			NextPageToken: nextPageToken,
		}
		jsonBytes, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal transactions to JSON: %w", err)
		}
		fmt.Println(string(jsonBytes))
	case "toml":
		response := struct {
			Transactions  any    `toml:"transactions"`
			NextPageToken string `toml:"next_page_token,omitempty"`
		}{
			Transactions:  transactions,
			NextPageToken: nextPageToken,
		}
		tomlBytes, err := toml.Marshal(response)
		if err != nil {
			return fmt.Errorf("failed to marshal transactions to TOML: %w", err)
		}
		fmt.Print(string(tomlBytes))
	default: // text format
		// Print header
		fmt.Printf("%-36s %-10s %-12s %-20s %-10s\n", "ID", "SOURCE", "STATE", "CREATED", "VALID")
		fmt.Println(strings.Repeat("-", 88))

		// Print transactions
		for _, tx := range transactions {
			createdTime := "N/A"
			if tx.GetCreatedAt() != nil {
				createdTime = tx.GetCreatedAt().AsTime().Format("2006-01-02 15:04:05")
			}

			fmt.Printf("%-36s %-10s %-12s %-20s %-10t\n",
				tx.GetId(),
				tx.GetSource(),
				tx.GetState(),
				createdTime,
				tx.GetIsValid(),
			)
		}

		if nextPageToken != "" {
			fmt.Printf("\nNext page token: %s\n", nextPageToken)
		}
	}

	return nil
}

// GetTransaction gets a specific configuration transaction by ID
func GetTransaction(ctx context.Context, serverAddr, transactionID, format string) error {
	logger := slog.Default()

	firelynxClient := client.New(client.Config{
		Logger:     logger,
		ServerAddr: serverAddr,
	})

	transaction, err := firelynxClient.GetConfigTransaction(ctx, transactionID)
	if err != nil {
		return err
	}

	if transaction == nil {
		fmt.Printf("Transaction %s not found\n", transactionID)
		return nil
	}

	switch format {
	case "json":
		if transaction.GetConfig() != nil {
			jsonBytes, err := json.MarshalIndent(transaction.GetConfig(), "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal config to JSON: %w", err)
			}
			fmt.Println(string(jsonBytes))
		} else {
			fmt.Println("{}")
		}
	case "toml":
		if transaction.GetConfig() != nil {
			tomlBytes, err := toml.Marshal(transaction.GetConfig())
			if err != nil {
				return fmt.Errorf("failed to marshal config to TOML: %w", err)
			}
			fmt.Print(string(tomlBytes))
		}
	default: // text format
		fmt.Printf("Transaction ID: %s\n", transaction.GetId())
		fmt.Printf("Source: %s\n", transaction.GetSource())
		fmt.Printf("Source Detail: %s\n", transaction.GetSourceDetail())
		fmt.Printf("Request ID: %s\n", transaction.GetRequestId())
		fmt.Printf("State: %s\n", transaction.GetState())
		fmt.Printf("Valid: %t\n", transaction.GetIsValid())
		if transaction.GetCreatedAt() != nil {
			fmt.Printf("Created: %s\n", transaction.GetCreatedAt().AsTime().Format(time.RFC3339))
		}
		if transaction.GetConfig() != nil {
			fmt.Printf("Config Version: %s\n", transaction.GetConfig().GetVersion())
			fmt.Printf("Config Listeners: %d\n", len(transaction.GetConfig().GetListeners()))
			fmt.Printf("Config Endpoints: %d\n", len(transaction.GetConfig().GetEndpoints()))
			fmt.Printf("Config Apps: %d\n", len(transaction.GetConfig().GetApps()))
		}

		// Show log count
		if len(transaction.GetLogs()) > 0 {
			fmt.Printf("Log Entries: %d\n", len(transaction.GetLogs()))
		}
	}

	return nil
}

// RollbackToTransaction rolls back to a previous configuration transaction
func RollbackToTransaction(ctx context.Context, serverAddr, transactionID string) error {
	logger := slog.Default()

	firelynxClient := client.New(client.Config{
		Logger:     logger,
		ServerAddr: serverAddr,
	})

	return firelynxClient.ApplyConfigFromTransaction(ctx, transactionID)
}

// ClearTransactions clears configuration transaction history
func ClearTransactions(ctx context.Context, serverAddr string, keepLast int32) error {
	logger := slog.Default()

	firelynxClient := client.New(client.Config{
		Logger:     logger,
		ServerAddr: serverAddr,
	})

	clearedCount, err := firelynxClient.ClearConfigTransactions(ctx, keepLast)
	if err != nil {
		return err
	}

	fmt.Printf("Cleared %d transaction(s), keeping last %d\n", clearedCount, keepLast)

	return nil
}
