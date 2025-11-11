package terramate

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestJWTCredential_ApplyCredentials(t *testing.T) {
	jwtToken := generateTestJWT(time.Now().Add(1 * time.Hour))
	cred, err := NewJWTCredential(jwtToken, "")
	if err != nil {
		t.Fatalf("NewJWTCredential() error = %v", err)
	}

	req, err := http.NewRequest("GET", "https://api.terramate.io/v1/stacks", nil)
	if err != nil {
		t.Fatalf("http.NewRequest() error = %v", err)
	}

	// ApplyCredentials should always succeed - no client-side expiration checking
	err = cred.ApplyCredentials(req)
	if err != nil {
		t.Errorf("ApplyCredentials() error = %v, expected no error", err)
	}

	// Verify Authorization header is set correctly
	authHeader := req.Header.Get("Authorization")
	if authHeader != "Bearer "+jwtToken {
		t.Errorf("Authorization header = %v, want Bearer %v", authHeader, jwtToken)
	}
}

func TestJWTCredential_Name(t *testing.T) {
	tests := []struct {
		name         string
		provider     string
		issuer       string
		wantProvider string
	}{
		{
			name:         "explicit provider",
			provider:     "Google",
			issuer:       "https://accounts.google.com",
			wantProvider: "Google",
		},
		{
			name:         "provider from issuer - Google",
			provider:     "",
			issuer:       "https://accounts.google.com",
			wantProvider: "Google",
		},
		{
			name:         "provider from issuer - GitHub",
			provider:     "",
			issuer:       "https://token.actions.githubusercontent.com",
			wantProvider: "GitHub Actions",
		},
		{
			name:         "provider from issuer - GitLab",
			provider:     "",
			issuer:       "https://gitlab.com",
			wantProvider: "GitLab",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := generateTestJWTWithIssuer(time.Now().Add(1*time.Hour), tt.issuer)
			cred, err := NewJWTCredential(token, tt.provider)
			if err != nil {
				t.Fatalf("NewJWTCredential() error = %v", err)
			}

			if got := cred.Name(); got != tt.wantProvider {
				t.Errorf("Name() = %v, want %v", got, tt.wantProvider)
			}
		})
	}
}

func TestAPIKeyCredential_ApplyCredentials(t *testing.T) {
	apiKey := "test-api-key-123"
	cred := NewAPIKeyCredential(apiKey)

	req, err := http.NewRequest("GET", "https://api.terramate.io/v1/stacks", nil)
	if err != nil {
		t.Fatalf("http.NewRequest() error = %v", err)
	}

	err = cred.ApplyCredentials(req)
	if err != nil {
		t.Errorf("ApplyCredentials() error = %v", err)
	}

	// Check Basic Auth
	username, password, ok := req.BasicAuth()
	if !ok {
		t.Error("Basic Auth not set")
	}
	if username != apiKey {
		t.Errorf("Basic Auth username = %v, want %v", username, apiKey)
	}
	if password != "" {
		t.Errorf("Basic Auth password = %v, want empty string", password)
	}
}

func TestAPIKeyCredential_Name(t *testing.T) {
	cred := NewAPIKeyCredential("test-key")
	if got := cred.Name(); got != "API Key" {
		t.Errorf("Name() = %v, want API Key", got)
	}
}

func TestLoadJWTFromFile(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		fileContent string
		fileName    string
		expectError bool
		checkFunc   func(t *testing.T, cred *JWTCredential)
	}{
		{
			name:     "valid credential file",
			fileName: "credentials.tmrc.json",
			fileContent: createTestCredentialFile(
				"Google",
				generateTestJWT(time.Now().Add(1*time.Hour)),
				"refresh-token-123",
			),
			expectError: false,
			checkFunc: func(t *testing.T, cred *JWTCredential) {
				if cred.Name() != "Google" {
					t.Errorf("Name() = %v, want Google", cred.Name())
				}
			},
		},
		{
			name:     "missing provider field",
			fileName: "credentials2.tmrc.json",
			fileContent: createTestCredentialFile(
				"",
				generateTestJWTWithIssuer(time.Now().Add(1*time.Hour), "https://accounts.google.com"),
				"refresh-token-123",
			),
			expectError: false,
			checkFunc: func(t *testing.T, cred *JWTCredential) {
				// Provider should be extracted from issuer
				if cred.Name() != "Google" {
					t.Errorf("Name() = %v, want Google (from issuer)", cred.Name())
				}
			},
		},
		{
			name:        "file does not exist",
			fileName:    "nonexistent.json",
			fileContent: "",
			expectError: true,
		},
		{
			name:        "invalid JSON",
			fileName:    "invalid.json",
			fileContent: `{invalid json`,
			expectError: true,
		},
		{
			name:        "missing id_token field",
			fileName:    "missing-token.json",
			fileContent: `{"provider": "Google"}`,
			expectError: true,
		},
		{
			name:        "invalid JWT token",
			fileName:    "invalid-jwt.json",
			fileContent: `{"provider": "Google", "id_token": "not-a-valid-jwt", "refresh_token": "refresh"}`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := filepath.Join(tmpDir, tt.fileName)

			// Write test file if content is provided
			if tt.fileContent != "" {
				if err := os.WriteFile(filePath, []byte(tt.fileContent), 0o600); err != nil {
					t.Fatalf("failed to write test file: %v", err)
				}
			}

			cred, err := LoadJWTFromFile(filePath)
			if (err != nil) != tt.expectError {
				t.Errorf("LoadJWTFromFile() error = %v, expectError %v", err, tt.expectError)
				return
			}

			if !tt.expectError && tt.checkFunc != nil {
				tt.checkFunc(t, cred)
			}
		})
	}
}

func TestLoadJWTFromFile_TildeExpansion(t *testing.T) {
	// Create test file in temp directory
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "credentials.tmrc.json")
	content := createTestCredentialFile(
		"Google",
		generateTestJWT(time.Now().Add(1*time.Hour)),
		"refresh-token",
	)
	if err := os.WriteFile(testFile, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Test with tilde path (this should work if we manually set HOME)
	t.Setenv("HOME", tmpDir)
	home, _ := os.UserHomeDir()

	// Create .terramate.d directory
	tmDir := filepath.Join(home, ".terramate.d")
	if err := os.MkdirAll(tmDir, 0o755); err != nil {
		t.Fatalf("failed to create .terramate.d directory: %v", err)
	}

	testFile2 := filepath.Join(tmDir, "credentials.tmrc.json")
	if err := os.WriteFile(testFile2, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	cred, err := LoadJWTFromFile("~/.terramate.d/credentials.tmrc.json")
	if err != nil {
		t.Errorf("LoadJWTFromFile() with tilde path error = %v", err)
	}
	if cred == nil {
		t.Error("LoadJWTFromFile() returned nil credential")
	}
}

func TestGetDefaultCredentialPath(t *testing.T) {
	path, err := GetDefaultCredentialPath()
	if err != nil {
		t.Errorf("GetDefaultCredentialPath() error = %v", err)
	}
	if path == "" {
		t.Error("GetDefaultCredentialPath() returned empty path")
	}
	// Should end with .terramate.d/credentials.tmrc.json
	if !filepath.IsAbs(path) {
		t.Error("GetDefaultCredentialPath() should return absolute path")
	}
}

func TestParseJWTToken(t *testing.T) {
	tests := []struct {
		name             string
		token            string
		expectError      bool
		expectedProvider string
	}{
		{
			name:             "valid JWT with Google issuer",
			token:            generateTestJWTWithIssuer(time.Now().Add(1*time.Hour), "https://accounts.google.com"),
			expectError:      false,
			expectedProvider: "Google",
		},
		{
			name:             "valid JWT with GitHub issuer",
			token:            generateTestJWTWithIssuer(time.Now().Add(1*time.Hour), "https://token.actions.githubusercontent.com"),
			expectError:      false,
			expectedProvider: "GitHub Actions",
		},
		{
			name:             "valid JWT with GitLab issuer",
			token:            generateTestJWTWithIssuer(time.Now().Add(1*time.Hour), "https://gitlab.com"),
			expectError:      false,
			expectedProvider: "GitLab",
		},
		{
			name:             "valid JWT with unknown issuer",
			token:            generateTestJWTWithIssuer(time.Now().Add(1*time.Hour), "https://custom-issuer.com"),
			expectError:      false,
			expectedProvider: "https://custom-issuer.com",
		},
		{
			name:        "invalid JWT - not enough parts",
			token:       "invalid.token",
			expectError: true,
		},
		{
			name:        "invalid JWT - garbage",
			token:       "not-a-jwt-at-all",
			expectError: true,
		},
		{
			name:        "empty token",
			token:       "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := parseJWTToken(tt.token)
			if (err != nil) != tt.expectError {
				t.Errorf("parseJWTToken() error = %v, expectError %v", err, tt.expectError)
				return
			}

			if !tt.expectError {
				if provider == "" {
					t.Error("provider should not be empty for valid token")
				}
				if tt.expectedProvider != "" && provider != tt.expectedProvider {
					t.Errorf("expected provider %q, got %q", tt.expectedProvider, provider)
				}
			}
		})
	}
}

// Helper functions

func generateTestJWT(expiration time.Time) string {
	return generateTestJWTWithIssuer(expiration, "https://accounts.google.com")
}

func generateTestJWTWithIssuer(expiration time.Time, issuer string) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iss":   issuer,
		"sub":   "test-user-123",
		"email": "test@example.com",
		"exp":   expiration.Unix(),
		"iat":   time.Now().Unix(),
	})

	// Sign with a test secret (doesn't matter for our tests since we don't verify)
	tokenString, _ := token.SignedString([]byte("test-secret"))
	return tokenString
}

func createTestCredentialFile(provider, idToken, refreshToken string) string {
	cred := cachedCredential{
		Provider:     provider,
		IDToken:      idToken,
		RefreshToken: refreshToken,
	}
	data, _ := json.Marshal(cred)
	return string(data)
}
