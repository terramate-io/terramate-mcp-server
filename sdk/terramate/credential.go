package terramate

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/golang-jwt/jwt/v5"
)

const (
	// Provider names
	providerGoogle        = "Google"
	providerGitHubActions = "GitHub Actions"
	providerGitLab        = "GitLab"

	// firebaseAuthAPIKey is the public Firebase Auth API key used for token refresh.
	// This is a public client ID that is safe to expose in client applications.
	// It's used to identify the Firebase project and is not a secret credential.
	firebaseAuthAPIKey = "AIzaSyAXJ6bqssXF4_W4dL6LwDVR7LEGVUZxnO0"
)

// Credential represents an authentication credential for Terramate Cloud
type Credential interface {
	// ApplyCredentials applies the credential to an HTTP request
	ApplyCredentials(req *http.Request) error

	// Name returns a human-readable name for the credential type
	Name() string
}

// RefreshableCredential represents a credential that can be refreshed
type RefreshableCredential interface {
	Credential
	// Refresh refreshes the credential using its refresh token
	Refresh(ctx context.Context) error
}

// JWTCredential implements Credential for JWT tokens loaded from credentials file.
// It supports automatic token refresh and file watching for external updates.
type JWTCredential struct {
	idToken        string
	refreshToken   string
	provider       string
	credentialPath string

	// Synchronization
	mu sync.RWMutex

	// File watching
	watcher     *fsnotify.Watcher
	stopWatcher chan struct{}

	// Refresh state
	refreshing     bool
	lastRefreshErr error
	refreshCond    *sync.Cond // Condition variable to wait for refresh completion

	// Testing: injected HTTP client and endpoint (only used in tests)
	httpClient      *http.Client
	refreshEndpoint string
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

// refreshResponse represents the response from Firebase Auth token refresh endpoint
type refreshResponse struct {
	IDToken      string `json:"id_token"`
	RefreshToken string `json:"refresh_token"`
}

// LoadJWTFromFile loads JWT credentials from a file (typically ~/.terramate.d/credentials.tmrc.json)
// and optionally starts watching the file for external updates (e.g., from Terramate CLI).
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

	cred := &JWTCredential{
		idToken:        cached.IDToken,
		refreshToken:   cached.RefreshToken,
		provider:       provider,
		credentialPath: credentialPath,
		stopWatcher:    make(chan struct{}),
	}
	// Initialize condition variable for waiting on refresh completion
	cred.refreshCond = sync.NewCond(&cred.mu)
	return cred, nil
}

// StartWatching starts watching the credential file for external updates (e.g., from Terramate CLI).
// This enables automatic token reload when the CLI refreshes the token.
// Call StopWatching() to clean up the file watcher.
func (j *JWTCredential) StartWatching(ctx context.Context) error {
	j.mu.Lock()
	defer j.mu.Unlock()

	if j.credentialPath == "" {
		return fmt.Errorf("cannot watch credential file: path not set")
	}

	if j.watcher != nil {
		return nil // Already watching
	}

	// Reinitialize stopWatcher if it was previously closed (e.g., after StopWatching())
	if j.stopWatcher == nil {
		j.stopWatcher = make(chan struct{})
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create file watcher: %w", err)
	}

	if err := watcher.Add(j.credentialPath); err != nil {
		_ = watcher.Close()
		return fmt.Errorf("failed to watch credential file: %w", err)
	}

	j.watcher = watcher

	go j.watchCredentialFile(ctx, watcher)

	return nil
}

// watchCredentialFile runs the file watcher loop in a separate goroutine.
// This method reduces cyclomatic complexity by extracting the watcher logic.
func (j *JWTCredential) watchCredentialFile(ctx context.Context, watcher *fsnotify.Watcher) {
	defer func() { _ = watcher.Close() }()

	for {
		// Check stopWatcher with lock to avoid race
		j.mu.RLock()
		stopCh := j.stopWatcher
		j.mu.RUnlock()

		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			j.handleFileEvent(event)

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Printf("File watcher error: %v", err)

		case <-stopCh:
			return

		case <-ctx.Done():
			return
		}
	}
}

// handleFileEvent processes file system events from the watcher.
func (j *JWTCredential) handleFileEvent(event fsnotify.Event) {
	// React to file writes, creates, and renames
	// Write: Direct file writes (e.g., when CLI updates the token)
	// Create: Atomic file replacement via os.Rename on some platforms (e.g., macOS)
	// Rename: Atomic file replacement via os.Rename on Linux with inotify
	// When a watched file is atomically replaced via os.Rename, different platforms
	// may emit different events. We handle all three to ensure cross-platform reliability.
	if event.Op&fsnotify.Write == fsnotify.Write ||
		event.Op&fsnotify.Create == fsnotify.Create ||
		event.Op&fsnotify.Rename == fsnotify.Rename {
		// Debounce rapid writes
		time.Sleep(100 * time.Millisecond)

		if err := j.reloadFromFile(); err != nil {
			log.Printf("Warning: failed to reload JWT credential from file: %v", err)
		} else {
			log.Printf("JWT credential reloaded from file")
		}
	}
}

// StopWatching stops watching the credential file and cleans up resources.
func (j *JWTCredential) StopWatching() {
	j.mu.Lock()
	defer j.mu.Unlock()

	if j.stopWatcher != nil {
		close(j.stopWatcher)
		j.stopWatcher = nil
	}

	if j.watcher != nil {
		_ = j.watcher.Close()
		j.watcher = nil
	}
}

// reloadFromFile reloads the credential from the file.
// This is called when the file watcher detects changes.
func (j *JWTCredential) reloadFromFile() error {
	// Check file permissions before reading (security: prevent loading from insecure files)
	fileInfo, err := os.Stat(j.credentialPath)
	if err != nil {
		return fmt.Errorf("failed to stat credential file: %w", err)
	}

	if permErr := checkCredentialFilePermissions(j.credentialPath, fileInfo); permErr != nil {
		return fmt.Errorf("credential file has insecure permissions, refusing to reload: %w", permErr)
	}

	data, err := os.ReadFile(j.credentialPath)
	if err != nil {
		return fmt.Errorf("failed to read credential file: %w", err)
	}

	var cached cachedCredential
	if err := json.Unmarshal(data, &cached); err != nil {
		return fmt.Errorf("failed to parse credential file: %w", err)
	}

	if cached.IDToken == "" {
		return fmt.Errorf("credential file is missing id_token field")
	}

	j.mu.Lock()
	defer j.mu.Unlock()

	j.idToken = cached.IDToken
	if cached.RefreshToken != "" {
		j.refreshToken = cached.RefreshToken
	}
	if cached.Provider != "" {
		j.provider = cached.Provider
	}

	return nil
}

// Refresh refreshes the JWT token using the refresh token.
// This method is called automatically when the API returns 401 Unauthorized.
// It exchanges the refresh_token for a new id_token via Firebase Auth API.
func (j *JWTCredential) Refresh(ctx context.Context) error {
	if !j.acquireRefreshLock() {
		return j.waitForRefresh(ctx)
	}
	defer j.releaseRefreshLock()

	// Copy refresh token while holding the lock to avoid data race with reloadFromFile()
	j.mu.RLock()
	refreshToken := j.refreshToken
	j.mu.RUnlock()

	if refreshToken == "" {
		return j.setRefreshError(fmt.Errorf("cannot refresh token: refresh_token not available"))
	}

	resp, body, err := j.makeRefreshRequest(ctx, refreshToken)
	if err != nil {
		return j.setRefreshError(err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return j.handleRefreshError(resp.StatusCode, body)
	}

	result, err := j.parseRefreshResponse(body)
	if err != nil {
		return j.setRefreshError(err)
	}

	j.updateCredentials(result)
	j.updateCredentialFileIfNeeded()

	log.Printf("JWT token refreshed successfully")
	return nil
}

// ensureRefreshCond ensures the refresh condition variable is initialized.
// This handles cases where JWTCredential is created manually (e.g., in tests).
func (j *JWTCredential) ensureRefreshCond() {
	if j.refreshCond == nil {
		j.refreshCond = sync.NewCond(&j.mu)
	}
}

// acquireRefreshLock acquires the refresh lock, returning true if successful.
func (j *JWTCredential) acquireRefreshLock() bool {
	j.mu.Lock()
	defer j.mu.Unlock()

	j.ensureRefreshCond()

	if j.refreshing {
		return false
	}
	j.refreshing = true
	return true
}

// waitForRefresh waits for an ongoing refresh to complete.
// It blocks until the refresh operation finishes (either successfully or with an error)
// or the context is canceled. Returns context error if context expires before refresh completes.
func (j *JWTCredential) waitForRefresh(ctx context.Context) error {
	j.mu.Lock()
	j.ensureRefreshCond()

	// Channel to signal when refresh completes
	refreshDone := make(chan error, 1)

	// Start a goroutine to wait for the refresh to complete
	// This goroutine will wait on the condition variable
	go func() {
		// Check context before acquiring lock to avoid unnecessary blocking
		if ctx.Err() != nil {
			// Context already canceled, exit immediately
			return
		}

		j.mu.Lock()
		defer j.mu.Unlock()

		// Wait until refresh completes (refreshing becomes false) or context is canceled
		for j.refreshing {
			// Check if context was canceled before waiting
			// This check happens while holding the lock, but we need to check before Wait()
			if ctx.Err() != nil {
				// Context was canceled, exit early to avoid leaking the goroutine
				return
			}
			j.refreshCond.Wait()
			// Check again after Wait() returns (it may have been woken up by Broadcast)
			// This handles the case where context was canceled while waiting
			if ctx.Err() != nil {
				// Context was canceled while waiting, exit early
				return
			}
		}

		// Refresh has completed, send the result (only if context wasn't canceled)
		if ctx.Err() == nil {
			select {
			case refreshDone <- j.lastRefreshErr:
			default:
				// Channel already closed or receiver gave up (context canceled)
			}
		}
	}()

	j.mu.Unlock()

	// Wait for either refresh completion or context cancellation
	select {
	case err := <-refreshDone:
		// Refresh completed, return the result
		return err
	case <-ctx.Done():
		// Context was canceled or expired - signal the condition variable to wake up
		// the waiting goroutine so it can exit early and release the mutex
		j.mu.Lock()
		j.ensureRefreshCond()
		j.refreshCond.Broadcast() // Wake up the waiting goroutine
		j.mu.Unlock()
		return ctx.Err()
	}
}

// releaseRefreshLock releases the refresh lock and signals waiting goroutines.
func (j *JWTCredential) releaseRefreshLock() {
	j.mu.Lock()
	j.refreshing = false
	j.ensureRefreshCond() // Ensure condition is initialized before signaling
	j.mu.Unlock()
	// Signal all waiting goroutines that refresh has completed
	j.refreshCond.Broadcast()
}

// setRefreshError sets the refresh error and returns it.
func (j *JWTCredential) setRefreshError(err error) error {
	j.mu.Lock()
	j.lastRefreshErr = err
	j.mu.Unlock()
	return err
}

// makeRefreshRequest makes the HTTP request to Firebase Auth.
func (j *JWTCredential) makeRefreshRequest(ctx context.Context, refreshToken string) (*http.Response, []byte, error) {
	// Use injected endpoint if available (for testing), otherwise use default Firebase endpoint
	endpoint := j.refreshEndpoint
	if endpoint == "" {
		endpoint = fmt.Sprintf("https://securetoken.googleapis.com/v1/token?key=%s", firebaseAuthAPIKey)
	}

	payload := map[string]string{
		"grant_type":    "refresh_token",
		"refresh_token": refreshToken,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal refresh payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(payloadBytes))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create refresh request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Use injected HTTP client if available (for testing), otherwise create default client
	client := j.httpClient
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		_ = resp.Body.Close()
		return nil, nil, fmt.Errorf("failed to read refresh response: %w", err)
	}

	return resp, body, nil
}

// handleRefreshError handles non-200 responses from Firebase Auth.
func (j *JWTCredential) handleRefreshError(statusCode int, body []byte) error {
	var errResp struct {
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}

	errorMsg := fmt.Sprintf("token refresh failed (status %d)", statusCode)
	if err := json.Unmarshal(body, &errResp); err == nil {
		if errResp.Error != "" {
			if errResp.ErrorDescription != "" {
				errorMsg = fmt.Sprintf("token refresh failed: %s - %s", errResp.Error, errResp.ErrorDescription)
			} else {
				errorMsg = fmt.Sprintf("token refresh failed: %s", errResp.Error)
			}
		}
	}

	return j.setRefreshError(fmt.Errorf("%s", errorMsg))
}

// parseRefreshResponse parses a successful refresh response.
func (j *JWTCredential) parseRefreshResponse(body []byte) (refreshResponse, error) {
	var result refreshResponse

	if err := json.Unmarshal(body, &result); err != nil {
		return result, fmt.Errorf("failed to parse refresh response: %w", err)
	}

	if result.IDToken == "" {
		return result, fmt.Errorf("refresh response missing id_token")
	}

	return result, nil
}

// updateCredentials updates the in-memory credentials.
func (j *JWTCredential) updateCredentials(result refreshResponse) {
	j.mu.Lock()
	defer j.mu.Unlock()

	j.idToken = result.IDToken
	if result.RefreshToken != "" {
		// Firebase may issue a new refresh token (token rotation)
		j.refreshToken = result.RefreshToken
	}
	j.lastRefreshErr = nil
}

// updateCredentialFileIfNeeded updates the credential file if path is set.
func (j *JWTCredential) updateCredentialFileIfNeeded() {
	if j.credentialPath != "" {
		if err := j.updateCredentialFile(); err != nil {
			log.Printf("Warning: failed to update credential file after refresh: %v", err)
			// Don't fail the refresh if file update fails - token is already in memory
		}
	}
}

// updateCredentialFile atomically updates the credential file with the current token.
// This ensures the Terramate CLI can see the refreshed token.
func (j *JWTCredential) updateCredentialFile() error {
	j.mu.RLock()
	defer j.mu.RUnlock()

	if j.credentialPath == "" {
		return fmt.Errorf("credential path not set")
	}

	cached := cachedCredential{
		Provider:     j.provider,
		IDToken:      j.idToken,
		RefreshToken: j.refreshToken,
	}

	data, err := json.MarshalIndent(cached, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}

	// Write to temporary file first
	tmpPath := j.credentialPath + ".tmp." + randomString(8)
	if err := os.WriteFile(tmpPath, data, 0o600); err != nil {
		return fmt.Errorf("failed to write temp credential file: %w", err)
	}

	// Atomic rename (overwrites existing file)
	if err := os.Rename(tmpPath, j.credentialPath); err != nil {
		_ = os.Remove(tmpPath) // Clean up temp file on failure
		return fmt.Errorf("failed to rename credential file: %w", err)
	}

	return nil
}

// randomString generates a random hex string of the specified length.
func randomString(length int) string {
	bytes := make([]byte, length/2)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
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

// ApplyCredentials applies the JWT credential to an HTTP request.
// This method is thread-safe and can be called concurrently.
func (j *JWTCredential) ApplyCredentials(req *http.Request) error {
	j.mu.RLock()
	defer j.mu.RUnlock()
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
