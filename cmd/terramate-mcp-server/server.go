package main

import (
	"context"
	"fmt"
	"log"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/terramate-io/terramate-mcp-server/sdk/terramate"
	"github.com/terramate-io/terramate-mcp-server/tools"
)

// Server implements the MCP server to extend its functionality
type Server struct {
	mcp          *server.MCPServer
	toolHandlers *tools.ToolHandlers
	config       *Config
}

type Config struct {
	APIKey  string
	Region  string
	BaseURL string
}

// newServer creates a new server instance
func newServer(config *Config) (*Server, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}

	// Create Terramate Cloud API client
	tmcClient, err := terramate.NewClient(
		config.APIKey,
		terramate.WithBaseURL(config.BaseURL),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Terramate client: %w", err)
	}

	// Create tool handlers
	toolHandlers := tools.New(tmcClient)

	// Create server
	s := &Server{
		toolHandlers: toolHandlers,
		config:       config,
	}

	// Create MCP server
	s.mcp = server.NewMCPServer(
		"terramate-mcp-server",
		"1.0.0",
		server.WithToolCapabilities(false),
		server.WithLogging(),
		// server.WithInstructions(instructions.Get()),
	)

	// Register MCP tools
	for _, tool := range toolHandlers.Tools() {
		s.mcp.AddTool(tool.Tool, tool.Handler)
		log.Printf("Registered MCP tool: %s", tool.Tool.Name)
	}

	return s, nil
}

// start starts the server with the given configuration
func (s *Server) start(ctx context.Context) error {
	log.Printf("Starting Terramate MCP server in stdio mode")

	// Start server in a goroutine so we can handle context cancellation
	errChan := make(chan error, 1)
	go func() {
		errChan <- server.ServeStdio(s.mcp)
	}()

	// Wait for context cancellation or server error
	select {
	case <-ctx.Done():
		log.Println("Context cancelled, shutting down stdio server")
		return ctx.Err()
	case err := <-errChan:
		return err
	}
}

// stop gracefully shuts down the server
func (s *Server) stop(ctx context.Context) {
	log.Println("terramate-mcp-server stopped")
}

// AddTool registers an MCP tool handler
func (s *Server) AddTool(tool mcp.Tool, handler func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)) {
	s.mcp.AddTool(tool, handler)
}
