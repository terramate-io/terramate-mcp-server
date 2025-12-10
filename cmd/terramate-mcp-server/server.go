package main

import (
	"context"
	"fmt"
	"log"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/terramate-io/terramate-mcp-server/internal/version"
	"github.com/terramate-io/terramate-mcp-server/sdk/terramate"
	"github.com/terramate-io/terramate-mcp-server/tools"
)

// Server implements the MCP server to extend its functionality
type Server struct {
	mcp          *server.MCPServer
	toolHandlers *tools.ToolHandlers
	config       *Config
	jwtCred      *terramate.JWTCredential // Store JWT credential for cleanup
}

// Config holds server configuration values required to initialize dependencies.
type Config struct {
	APIKey         string
	CredentialFile string
	Region         string
	BaseURL        string
}

// newServer creates a new server instance
func newServer(config *Config) (*Server, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}

	// Load credential (precedence: API Key > JWT from file)
	var credential terramate.Credential
	var err error

	// Check API key first (backward compatibility)
	if config.APIKey != "" {
		credential = terramate.NewAPIKeyCredential(config.APIKey)
	} else {
		// Load JWT from credential file
		credPath := config.CredentialFile
		if credPath == "" {
			// Use default path
			credPath, err = terramate.GetDefaultCredentialPath()
			if err != nil {
				return nil, fmt.Errorf("failed to determine default credential path: %w", err)
			}
		}

		credential, err = terramate.LoadJWTFromFile(credPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load credentials: %w", err)
		}
		log.Printf("Using JWT authentication (provider: %s)", credential.Name())
	}

	// Create Terramate Cloud API client with credential
	var opts []terramate.ClientOption
	if config.BaseURL == "" || config.BaseURL == "https://api.terramate.io" {
		opts = append(opts, terramate.WithRegion(config.Region))
	} else {
		opts = append(opts, terramate.WithBaseURL(config.BaseURL))
	}

	tmcClient, err := terramate.NewClient(credential, opts...)
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

	// Store JWT credential if we're using it
	if jwtCred, ok := credential.(*terramate.JWTCredential); ok {
		s.jwtCred = jwtCred
	}

	// Create MCP server
	s.mcp = server.NewMCPServer(
		"terramate-mcp-server",
		version.Version,
		server.WithToolCapabilities(false),
		server.WithLogging(),
		// server.WithInstructions(instructions.Get()),
	)

	// Register MCP tools using AddTools
	s.mcp.AddTools(toolHandlers.Tools()...)
	for _, tool := range toolHandlers.Tools() {
		log.Printf("Registered MCP tool: %s", tool.Tool.Name)
	}

	return s, nil
}

// start starts the server with the given configuration
func (s *Server) start(ctx context.Context) error {
	log.Printf("Starting Terramate MCP server in stdio mode")

	// Start file watching if using JWT credentials
	// Note: We use graceful degradation - if file watching fails, the server continues
	// to work normally. Token refresh will still work via the automatic refresh mechanism
	// when API calls return 401. We don't retry starting the watcher because:
	// 1. File watching is a convenience feature, not critical for functionality
	// 2. Retry logic would add complexity without significant benefit
	// 3. Users can restart the server if file watching is needed
	if s.jwtCred != nil {
		if err := s.jwtCred.StartWatching(ctx); err != nil {
			log.Printf("Warning: failed to start credential file watching: %v", err)
			log.Printf("Automatic token reload from CLI updates will not be available")
		} else {
			log.Printf("Started watching credential file for automatic token reload")
		}
	}

	// Start server in a goroutine so we can handle context cancellation
	errChan := make(chan error, 1)
	go func() {
		errChan <- server.ServeStdio(s.mcp)
	}()

	// Wait for context cancellation or server error
	select {
	case <-ctx.Done():
		log.Println("Context canceled, shutting down stdio server")
		return ctx.Err()
	case err := <-errChan:
		return err
	}
}

// stop gracefully shuts down the server
func (s *Server) stop(_ context.Context) {
	// Stop file watching if active
	if s.jwtCred != nil {
		s.jwtCred.StopWatching()
		log.Println("Stopped credential file watching")
	}

	log.Println("Terramate MCP server stopped")
}

// AddTool registers an MCP tool handler
func (s *Server) AddTool(tool mcp.Tool, handler func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)) {
	s.mcp.AddTool(tool, handler)
}
