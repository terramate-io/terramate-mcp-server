// Copyright 2025 Terramate GmbH and contributors
// SPDX-License-Identifier: Apache-2.0

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
		Name:     "api-key",
		Usage:    "Terramate Cloud API key",
		EnvVars:  []string{"TERRAMATE_API_KEY"},
		Required: true,
	}

	regionFlag = &cli.StringFlag{
		Name:     "region",
		Usage:    "Terramate Cloud region (eu or us)",
		EnvVars:  []string{"TERRAMATE_REGION"},
		Required: true,
	}

	apiEndpointFlag = &cli.StringFlag{
		Name:    "api-endpoint",
		Usage:   "Terramate Cloud API endpoint",
		EnvVars: []string{"TERRAMATE_API_ENDPOINT"},
		Value:   "https://api.terramate.io",
	}
)

func main() {
	app := &cli.App{
		Name:        "terramate-mcp-server",
		Usage:       "Terramate MCP Server",
		Description: "Terramate MCP server to manage Terramate Cloud and CLI with natural language",
		Flags:       []cli.Flag{apiKeyFlag, regionFlag, apiEndpointFlag},
		Action: func(c *cli.Context) error {
			apiKey := c.String(apiKeyFlag.Name)
			// Validate region
			region := c.String(regionFlag.Name)
			if region != "eu" && region != "us" {
				return fmt.Errorf("invalid region: %s (must be 'eu' or 'us')", region)
			}

			apiEndpoint := c.String(apiEndpointFlag.Name)

			config := &Config{
				APIKey:      apiKey,
				Region:      region,
				APIEndpoint: apiEndpoint,
			}

			server, err := newServer(config)
			if err != nil {
				return fmt.Errorf("failed to create mcp server: %w", err)
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

			ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()

			server.stop(ctx)

			log.Println("Standalone server shut down")

			return serverErr
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatalf("Failed to run application: %v", err)
	}
}
