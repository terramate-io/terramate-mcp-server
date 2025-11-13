package terramate

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

const (
	// Provider names
	providerGoogle        = "Google"
	providerGitHubActions = "GitHub Actions"
	providerGitLab        = "GitLab"
)

// Credential represents an authentication credential for Terramate Cloud
type Credential interface {
	// ApplyCredentials applies the credential to an HTTP request
	ApplyCredentials(req *http.Request) error

	// Name returns a human-readable name for the credential type
	Name() string
}

// JWTCredential implements Credential for JWT tokens loaded from credentials file
type JWTCredential struct {
	idToken  string
	provider string
}

// APIKeyCredential implements Credential for organizational API keys
type APIKeyCredential struct {
	apiKey string
}

// cachedCredential represents the structure stored in credentials.tmrc.json
type cachedCredential struct {
	Provider     string `json:"provider"`
	IDToken      string `json:"id_token"`
	RefreshToken string `json:"refresh_token"`
}

// LoadJWTFromFile loads JWT credentials from a file (typically ~/.terramate.d/credentials.tmrc.json)
func LoadJWTFromFile(credentialPath string) (*JWTCredential, error) {
	// Expand home directory if path starts with ~
	if strings.HasPrefix(credentialPath, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to determine home directory: %w", err)
		}
		credentialPath = filepath.Join(home, credentialPath[1:])
	}

	// Check if file exists
	fileInfo, err := os.Stat(credentialPath)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf(
			"credential file not found at %s\n\n"+
				"To authenticate:\n"+
				"  1. Run 'terramate cloud login' in your terminal\n"+
				"  2. Or provide an API key with --api-key flag\n"+
				"  3. Or set TERRAMATE_API_KEY environment variable",
			credentialPath,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to stat credential file: %w", err)
	}

	// Validate file permissions (OS-specific)
	if permErr := checkCredentialFilePermissions(credentialPath, fileInfo); permErr != nil {
		return nil, permErr
	}

	// Read the file
	data, err := os.ReadFile(credentialPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read credential file: %w", err)
	}

	// Parse JSON
	var cached cachedCredential
	if unmarshalErr := json.Unmarshal(data, &cached); unmarshalErr != nil {
		return nil, fmt.Errorf("failed to parse credential file: %w", unmarshalErr)
	}

	// Validate required fields
	if cached.IDToken == "" {
		return nil, fmt.Errorf("credential file is missing id_token field")
	}

	// Parse JWT to extract provider info (for display purposes only)
	// Note: We do NOT validate expiration client-side - the API server is the source of truth
	detectedProvider, err := parseJWTToken(cached.IDToken)
	if err != nil {
		return nil, fmt.Errorf(
			"invalid JWT token: %w\n\n"+
				"The credential file may be corrupted.\n"+
				"To fix:\n"+
				"  1. Delete %s\n"+
				"  2. Run 'terramate cloud login' again",
			err, credentialPath,
		)
	}

	// Use cached provider if available, otherwise use detected provider
	provider := cached.Provider
	if provider == "" {
		provider = detectedProvider
	}

	return &JWTCredential{
		idToken:  cached.IDToken,
		provider: provider,
	}, nil
}

// NewJWTCredential creates a new JWT credential from a raw token string
func NewJWTCredential(jwtToken string, provider string) (*JWTCredential, error) {
	if jwtToken == "" {
		return nil, fmt.Errorf("JWT token is required")
	}

	// Parse JWT to extract provider info (for display purposes only)
	// Note: We do NOT validate expiration client-side - the API server is the source of truth
	detectedProvider, err := parseJWTToken(jwtToken)
	if err != nil {
		return nil, fmt.Errorf("invalid JWT token: %w", err)
	}

	// Use provided provider if available, otherwise use detected provider
	if provider == "" {
		provider = detectedProvider
	}

	return &JWTCredential{
		idToken:  jwtToken,
		provider: provider,
	}, nil
}

// ApplyCredentials applies the JWT credential to an HTTP request
// Note: We do NOT check expiration client-side - the API server validates the token
func (j *JWTCredential) ApplyCredentials(req *http.Request) error {
	req.Header.Set("Authorization", "Bearer "+j.idToken)
	return nil
}

// Name returns the provider name for the credential
func (j *JWTCredential) Name() string {
	return j.provider
}

// NewAPIKeyCredential creates a new API key credential
func NewAPIKeyCredential(apiKey string) *APIKeyCredential {
	return &APIKeyCredential{apiKey: apiKey}
}

// ApplyCredentials applies the API key to an HTTP request using Basic Auth
func (a *APIKeyCredential) ApplyCredentials(req *http.Request) error {
	req.SetBasicAuth(a.apiKey, "")
	return nil
}

// Name returns the credential type name
func (a *APIKeyCredential) Name() string {
	return "API Key"
}

// parseJWTToken parses a JWT token and extracts provider information for display purposes
// Note: This does NOT verify the signature or validate expiration - the API server is the source of truth
// We only extract the issuer to provide a friendly provider name to users
func parseJWTToken(token string) (provider string, err error) {
	parser := &jwt.Parser{}
	parsedToken, _, parseErr := parser.ParseUnverified(token, jwt.MapClaims{})
	if parseErr != nil {
		return "", fmt.Errorf("failed to parse JWT: %w", parseErr)
	}

	claims := parsedToken.Claims

	// Extract provider from issuer (for display purposes only)
	provider = "unknown"
	if iss, issErr := claims.GetIssuer(); issErr == nil && iss != "" {
		provider = extractProviderFromIssuer(iss)
	}

	return provider, nil
}

// extractProviderFromIssuer extracts a friendly provider name from JWT issuer
func extractProviderFromIssuer(issuer string) string {
	// Common issuer patterns
	switch {
	case issuer == "https://accounts.google.com" || issuer == "accounts.google.com":
		return providerGoogle
	case issuer == "https://token.actions.githubusercontent.com" || issuer == "token.actions.githubusercontent.com":
		return providerGitHubActions
	case issuer == "https://gitlab.com":
		return providerGitLab
	default:
		return issuer
	}
}

// GetDefaultCredentialPath returns the default path for the credential file
func GetDefaultCredentialPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to determine home directory: %w", err)
	}
	return filepath.Join(home, ".terramate.d", "credentials.tmrc.json"), nil
}
