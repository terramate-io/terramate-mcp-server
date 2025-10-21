package main

import (
	"testing"
)

func TestNewServer_RequiresConfig(t *testing.T) {
	_, err := newServer(nil)
	if err == nil || err.Error() != "config is required" {
		t.Fatalf("expected config required error, got: %v", err)
	}
}

func TestNewServer_ValidatesAPIKey(t *testing.T) {
	_, err := newServer(&Config{
		APIKey:  "",
		Region:  "eu",
		BaseURL: "https://api.terramate.io",
	})
	if err == nil {
		t.Fatalf("expected error for empty API key")
	}
}

func TestNewServer_Success(t *testing.T) {
	s, err := newServer(&Config{
		APIKey:  "test-key",
		Region:  "eu",
		BaseURL: "https://api.terramate.io",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s == nil {
		t.Fatal("expected server instance")
		return
	}
	if s.mcp == nil {
		t.Fatal("expected MCP server to be initialized")
	}
	if s.toolHandlers == nil {
		t.Fatal("expected tool handlers to be initialized")
	}
}

func TestConfig_Struct(t *testing.T) {
	cfg := &Config{
		APIKey:  "key",
		Region:  "us",
		BaseURL: "https://api.us.terramate.io",
	}
	if cfg.APIKey != "key" || cfg.Region != "us" {
		t.Fatalf("config fields not set correctly")
	}
}
