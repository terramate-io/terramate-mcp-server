package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestNewServer_RequiresConfig(t *testing.T) {
	_, err := newServer(nil)
	if err == nil || err.Error() != "config is required" {
		t.Fatalf("expected config required error, got: %v", err)
	}
}

func TestNewServer_ValidatesAPIKey(t *testing.T) {
	// When no API key or credential file is provided, should error
	_, err := newServer(&Config{
		APIKey:         "",
		CredentialFile: "/nonexistent/path/credentials.json",
		Region:         "eu",
		BaseURL:        "https://api.terramate.io",
	})
	if err == nil {
		t.Fatalf("expected error for missing credentials")
	}
}

func TestNewServer_WithJWT(t *testing.T) {
	// Create a temporary credential file
	tmpDir := t.TempDir()
	credFile := filepath.Join(tmpDir, "credentials.tmrc.json")

	// Create a test JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iss": "https://accounts.google.com",
		"sub": "test-user",
		"exp": time.Now().Add(1 * time.Hour).Unix(),
		"iat": time.Now().Unix(),
	})
	tokenString, _ := token.SignedString([]byte("test-secret"))

	// Write credential file
	cred := map[string]string{
		"provider":      "Google",
		"id_token":      tokenString,
		"refresh_token": "refresh-token",
	}
	data, _ := json.Marshal(cred)
	if err := os.WriteFile(credFile, data, 0o600); err != nil {
		t.Fatalf("failed to write credential file: %v", err)
	}

	// Create server with JWT credential
	s, err := newServer(&Config{
		APIKey:         "",
		CredentialFile: credFile,
		Region:         "eu",
		BaseURL:        "https://api.terramate.io",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s == nil {
		t.Fatal("expected server instance")
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
