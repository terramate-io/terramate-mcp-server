package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/urfave/cli/v2"
)

var (
	apiKeyFlag = &cli.StringFlag{
		Name:    "api-key",
		Usage:   "Terramate Cloud API key",
		EnvVars: []string{"TERRAMATE_API_KEY"},
	}

	credentialFileFlag = &cli.StringFlag{
		Name:    "credential-file",
		Usage:   "Path to JWT credentials file (default: ~/.terramate.d/credentials.tmrc.json)",
		EnvVars: []string{"TERRAMATE_CREDENTIAL_FILE"},
	}

	regionFlag = &cli.StringFlag{
		Name:     "region",
		Usage:    "Terramate Cloud region (eu or us)",
		EnvVars:  []string{"TERRAMATE_REGION"},
		Required: false,
	}

	baseURLFlag = &cli.StringFlag{
		Name:    "base-url",
		Usage:   "Terramate Cloud API base URL",
		EnvVars: []string{"TERRAMATE_BASE_URL"},
		Value:   "https://api.terramate.io",
	}
)

func main() {
	app := &cli.App{
		Name:        "terramate-mcp-server",
		Usage:       "Terramate MCP Server",
		Description: "Terramate MCP server to manage Terramate Cloud and CLI with natural language",
		Flags:       []cli.Flag{apiKeyFlag, credentialFileFlag, regionFlag, baseURLFlag},
		Action: func(c *cli.Context) error {
			apiKey := c.String(apiKeyFlag.Name)
			credentialFile := c.String(credentialFileFlag.Name)
			region := c.String(regionFlag.Name)
			baseURL := c.String(baseURLFlag.Name)

			// Only validate region if provided and using default base URL
			if baseURL == "https://api.terramate.io" && region != "" && region != "eu" && region != "us" {
				return fmt.Errorf("invalid region: %s (must be 'eu' or 'us')", region)
			}

			config := &Config{
				APIKey:         apiKey,
				CredentialFile: credentialFile,
				Region:         region,
				BaseURL:        baseURL,
			}

			server, err := newServer(config)
			if err != nil {
				return fmt.Errorf("failed to create MCP server: %w", err)
			}

			ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
			defer stop()

			errChan := make(chan error, 1)
			go func() {
				if err := server.start(ctx); err != nil {
					errChan <- err
				}
			}()

			var serverErr error
			select {
			case <-ctx.Done():
				log.Println("Received signal, shutting down...")
			case serverErr = <-errChan:
				log.Println("Server error, shutting down...")
				stop()
			}

			// Use context.Background() for shutdown timeout to ensure it's not already canceled
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer shutdownCancel()

			server.stop(shutdownCtx)

			log.Println("Terramate MCP server shut down")

			return serverErr
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatalf("Failed to run application: %v", err)
	}
}
