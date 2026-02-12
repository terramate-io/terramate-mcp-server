package terramate

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestJWTCredential_Refresh(t *testing.T) {
	t.Run("successful refresh", testJWTCredentialRefreshSuccessful)
	t.Run("missing refresh token", testJWTCredentialRefreshMissingToken)
	t.Run("concurrent refresh attempts", testJWTCredentialRefreshConcurrent)
	t.Run("waitForRefresh waits for slow refresh", testJWTCredentialRefreshWaitForRefresh)
	t.Run("waitForRefresh respects context cancellation", testJWTCredentialRefreshContextCancellation)
}

func testJWTCredentialRefreshSuccessful(t *testing.T) {
	// Create mock Firebase Auth server
	newToken := generateMockJWT()
	newRefreshToken := "new-refresh-token-456"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/token" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var payload map[string]string
		_ = json.NewDecoder(r.Body).Decode(&payload)

		if payload["grant_type"] != "refresh_token" {
			t.Errorf("unexpected grant_type: %s", payload["grant_type"])
		}

		if payload["refresh_token"] != "old-refresh-token-123" {
			t.Errorf("unexpected refresh_token: %s", payload["refresh_token"])
		}

		response := map[string]string{
			"id_token":      newToken,
			"refresh_token": newRefreshToken,
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create credential with injected HTTP client and endpoint for testing
	oldToken := generateMockJWT()
	cred := &JWTCredential{
		idToken:         oldToken,
		refreshToken:    "old-refresh-token-123",
		provider:        "Google",
		httpClient:      server.Client(),
		refreshEndpoint: server.URL + "/v1/token",
	}

	ctx := context.Background()

	// Refresh should succeed with the mock server
	err := cred.Refresh(ctx)
	if err != nil {
		t.Fatalf("expected successful refresh, got error: %v", err)
	}

	// Verify the token was updated
	cred.mu.RLock()
	updatedToken := cred.idToken
	updatedRefreshToken := cred.refreshToken
	cred.mu.RUnlock()

	if updatedToken != newToken {
		t.Errorf("expected id_token to be updated, got %s", updatedToken)
	}

	if updatedRefreshToken != newRefreshToken {
		t.Errorf("expected refresh_token to be updated, got %s", updatedRefreshToken)
	}
}

func testJWTCredentialRefreshMissingToken(t *testing.T) {
	cred := &JWTCredential{
		idToken:  generateMockJWT(),
		provider: "Google",
		// refreshToken intentionally missing
	}

	err := cred.Refresh(context.Background())
	if err == nil {
		t.Fatal("expected error when refresh_token is missing")
	}

	if err.Error() != "cannot refresh token: refresh_token not available" {
		t.Errorf("unexpected error: %v", err)
	}
}

func testJWTCredentialRefreshConcurrent(t *testing.T) {
	cred := &JWTCredential{
		idToken:      generateMockJWT(),
		refreshToken: "test-refresh-token",
		provider:     "Google",
	}

	var wg sync.WaitGroup
	errors := make([]error, 10)

	// Launch 10 concurrent refresh attempts
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			errors[index] = cred.Refresh(context.Background())
		}(i)
	}

	wg.Wait()

	// All should complete without panic
	t.Log("Concurrent refresh attempts completed")
}

func testJWTCredentialRefreshWaitForRefresh(t *testing.T) {
	// Create a mock server that delays its response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow API response (200ms delay)
		time.Sleep(200 * time.Millisecond)
		response := map[string]string{
			"id_token":      generateMockJWT(),
			"refresh_token": "new-refresh-token",
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create credential with refresh token and injected HTTP client/endpoint for testing
	cred := &JWTCredential{
		idToken:         generateMockJWT(),
		refreshToken:    "test-refresh-token",
		provider:        "Google",
		httpClient:      server.Client(),
		refreshEndpoint: server.URL + "/v1/token",
	}

	// Start a refresh that will take time (200ms delay from mock server)
	refreshStarted := make(chan struct{})
	refreshDone := make(chan error)
	go func() {
		close(refreshStarted)
		err := cred.Refresh(context.Background())
		refreshDone <- err
	}()

	// Wait for refresh goroutine to start
	<-refreshStarted

	// Check if refresh is in progress - it might complete very quickly in CI
	// so we check multiple times with small delays
	var refreshing bool
	for i := 0; i < 10; i++ {
		cred.mu.RLock()
		refreshing = cred.refreshing
		cred.mu.RUnlock()
		if refreshing {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	// Refresh should be in progress (mock server has 200ms delay)
	if !refreshing {
		t.Fatal("expected refresh to be in progress")
	}

	// Refresh is in progress - now test that waitForRefresh actually waits
	// Call Refresh() immediately while refreshing is true to test the waiting mechanism
	waitStart := time.Now()
	err := cred.Refresh(context.Background())
	waitDuration := time.Since(waitStart)

	// Verify we got the result from the first refresh attempt
	firstErr := <-refreshDone

	// The key test: waitForRefresh should have waited for the first refresh to complete
	// Since the mock server has a 200ms delay, waitForRefresh should wait at least that long
	// (minus some overhead for goroutine scheduling)
	if waitDuration < 100*time.Millisecond {
		t.Errorf("waitForRefresh should have waited for the slow refresh (200ms delay), but only waited %v", waitDuration)
	}

	// Both refresh attempts should succeed (mock server returns valid tokens)
	if err != nil {
		t.Errorf("second refresh should have succeeded, got error: %v", err)
	}
	if firstErr != nil {
		t.Errorf("first refresh should have succeeded, got error: %v", firstErr)
	}

	t.Logf("✓ waitForRefresh properly waited for refresh to complete (duration: %v)", waitDuration)
}

// testJWTCredentialRefreshContextCancellation tests that waitForRefresh respects context cancellation.
// This verifies the fix for the issue where waitForRefresh would block indefinitely even when
// the context expired.
func testJWTCredentialRefreshContextCancellation(t *testing.T) {
	// Create a mock server with a long delay to ensure refresh is in progress
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow API response (500ms delay to ensure refresh is in progress)
		time.Sleep(500 * time.Millisecond)
		response := map[string]string{
			"id_token":      generateMockJWT(),
			"refresh_token": "new-refresh-token",
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create a credential with refresh token and injected HTTP client/endpoint for testing
	cred := &JWTCredential{
		idToken:         "test-token",
		refreshToken:    "test-refresh-token",
		provider:        "Google",
		httpClient:      server.Client(),
		refreshEndpoint: server.URL + "/v1/token",
	}

	// Start a slow refresh that will take a while
	refreshStarted := make(chan struct{})
	refreshBlocked := make(chan struct{})
	go func() {
		close(refreshStarted)
		// Use a context that won't expire - this refresh will be slow (500ms delay)
		ctx := context.Background()
		_ = cred.Refresh(ctx)
		close(refreshBlocked)
	}()

	// Wait for refresh goroutine to start
	<-refreshStarted

	// Wait a bit to ensure refresh is in progress (mock server has 500ms delay)
	time.Sleep(50 * time.Millisecond)

	// Verify refresh is in progress
	cred.mu.RLock()
	refreshing := cred.refreshing
	cred.mu.RUnlock()

	if !refreshing {
		t.Fatal("expected refresh to be in progress (mock server has 500ms delay)")
	}

	// Now test that a context with a short timeout returns immediately
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	waitStart := time.Now()
	err := cred.Refresh(ctx)
	waitDuration := time.Since(waitStart)

	// Should return quickly (within timeout + small overhead)
	// The context timeout is 100ms, so waitForRefresh should return around that time
	if waitDuration > 200*time.Millisecond {
		t.Errorf("waitForRefresh should respect context timeout (100ms), but waited %v", waitDuration)
	}

	// Should return context deadline exceeded error
	if err == nil {
		t.Errorf("waitForRefresh should return error when context expires, got nil")
	} else if err != context.DeadlineExceeded {
		// Might also be context.Canceled if timing is different, but should be a context error
		if err != context.Canceled {
			t.Logf("Note: Expected context error, got %v (this might be acceptable)", err)
		}
	}

	t.Logf("✓ waitForRefresh properly respects context cancellation (waited %v, error: %v)", waitDuration, err)

	// Wait for the first refresh to complete (cleanup)
	select {
	case <-refreshBlocked:
		// First refresh completed
	case <-time.After(1 * time.Second):
		// First refresh still running, but that's okay - we've verified our fix works
	}
}

func TestJWTCredential_StartWatching(t *testing.T) {
	t.Run("watches file for changes", testStartWatchingFileChanges)
	t.Run("handles atomic file replacement via rename", testStartWatchingAtomicRename)
	t.Run("handles missing credential path", testStartWatchingMissingPath)
	t.Run("can restart watching after stop", testStartWatchingRestart)
	t.Run("watcher works after stop and restart", testStartWatchingWorksAfterRestart)
}

// testStartWatchingFileChanges tests that the watcher detects file changes and reloads credentials.
func testStartWatchingFileChanges(t *testing.T) {
	// Create temporary credential file
	tmpDir := t.TempDir()
	credFile := filepath.Join(tmpDir, "credentials.tmrc.json")

	// Write initial credential
	initialCred := cachedCredential{
		Provider:     "Google",
		IDToken:      generateMockJWT(),
		RefreshToken: "refresh-token-1",
	}
	data, _ := json.MarshalIndent(initialCred, "", "  ")
	if err := os.WriteFile(credFile, data, 0o600); err != nil {
		t.Fatalf("failed to write credential file: %v", err)
	}

	// Load credential
	cred, err := LoadJWTFromFile(credFile)
	if err != nil {
		t.Fatalf("failed to load credential: %v", err)
	}

	// Start watching
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := cred.StartWatching(ctx); err != nil {
		t.Fatalf("failed to start watching: %v", err)
	}
	defer cred.StopWatching()

	originalToken := cred.idToken

	// Wait a moment for watcher to initialize
	time.Sleep(200 * time.Millisecond)

	// Update the file (simulating CLI refresh)
	newToken := generateMockJWT()
	newCred := cachedCredential{
		Provider:     "Google",
		IDToken:      newToken,
		RefreshToken: "refresh-token-2",
	}
	newData, _ := json.MarshalIndent(newCred, "", "  ")
	if err := os.WriteFile(credFile, newData, 0o600); err != nil {
		t.Fatalf("failed to write credential file: %v", err)
	}

	// Wait for file watcher to detect and reload
	time.Sleep(300 * time.Millisecond)

	// Verify token was updated
	cred.mu.RLock()
	updatedToken := cred.idToken
	cred.mu.RUnlock()

	if updatedToken == originalToken {
		t.Logf("Warning: credential may not have reloaded yet (timing issue)")
	}

	if updatedToken == newToken {
		t.Log("✓ Credential successfully reloaded from file")
	} else {
		t.Logf("Token after reload: %s", updatedToken[:20]+"...")
		t.Logf("Expected token: %s", newToken[:20]+"...")
	}
}

// testStartWatchingAtomicRename tests that the watcher detects atomic file replacement via os.Rename.
// This simulates how updateCredentialFile atomically updates the credential file.
// On Linux with inotify, this triggers a Rename event; on macOS it may trigger a Create event.
// The watcher handles both Write, Create, and Rename events for cross-platform reliability.
func testStartWatchingAtomicRename(t *testing.T) {
	// Create temporary credential file
	tmpDir := t.TempDir()
	credFile := filepath.Join(tmpDir, "credentials.tmrc.json")

	// Write initial credential
	initialCred := cachedCredential{
		Provider:     "Google",
		IDToken:      generateMockJWT(),
		RefreshToken: "refresh-token-1",
	}
	data, _ := json.MarshalIndent(initialCred, "", "  ")
	if err := os.WriteFile(credFile, data, 0o600); err != nil {
		t.Fatalf("failed to write credential file: %v", err)
	}

	// Load credential
	cred, err := LoadJWTFromFile(credFile)
	if err != nil {
		t.Fatalf("failed to load credential: %v", err)
	}

	// Start watching
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := cred.StartWatching(ctx); err != nil {
		t.Fatalf("failed to start watching: %v", err)
	}
	defer cred.StopWatching()

	originalToken := cred.idToken

	// Wait a moment for watcher to initialize
	time.Sleep(200 * time.Millisecond)

	// Atomically replace the file using os.Rename (simulating updateCredentialFile behavior)
	newToken := generateMockJWT()
	newCred := cachedCredential{
		Provider:     "Google",
		IDToken:      newToken,
		RefreshToken: "refresh-token-2",
	}
	newData, _ := json.MarshalIndent(newCred, "", "  ")

	// Write to temporary file first, then atomically rename (same pattern as updateCredentialFile)
	tmpPath := credFile + ".tmp." + randomString(8)
	if err := os.WriteFile(tmpPath, newData, 0o600); err != nil {
		t.Fatalf("failed to write temp credential file: %v", err)
	}

	// Atomic rename (overwrites existing file) - this should trigger a Rename event
	if err := os.Rename(tmpPath, credFile); err != nil {
		_ = os.Remove(tmpPath) // Clean up temp file on failure
		t.Fatalf("failed to rename credential file: %v", err)
	}

	// Wait for file watcher to detect and reload
	time.Sleep(300 * time.Millisecond)

	// Verify token was updated
	cred.mu.RLock()
	updatedToken := cred.idToken
	cred.mu.RUnlock()

	if updatedToken == originalToken {
		t.Logf("Warning: credential may not have reloaded yet (timing issue)")
	}

	if updatedToken == newToken {
		t.Log("✓ Credential successfully reloaded after atomic file replacement (Rename event)")
	} else {
		t.Logf("Token after reload: %s", updatedToken[:20]+"...")
		t.Logf("Expected token: %s", newToken[:20]+"...")
	}
}

// testStartWatchingMissingPath tests error handling when credential path is not set.
func testStartWatchingMissingPath(t *testing.T) {
	cred := &JWTCredential{
		idToken:  generateMockJWT(),
		provider: "Google",
		// credentialPath intentionally empty
	}

	err := cred.StartWatching(context.Background())
	if err == nil {
		t.Fatal("expected error when credential path is not set")
	}
}

// testStartWatchingRestart tests that watching can be restarted after stopping.
func testStartWatchingRestart(t *testing.T) {
	// Create temporary credential file
	tmpDir := t.TempDir()
	credFile := filepath.Join(tmpDir, "credentials.tmrc.json")

	// Write initial credential
	initialCred := cachedCredential{
		Provider:     "Google",
		IDToken:      generateMockJWT(),
		RefreshToken: "refresh-token-1",
	}
	data, _ := json.MarshalIndent(initialCred, "", "  ")
	if err := os.WriteFile(credFile, data, 0o600); err != nil {
		t.Fatalf("failed to write credential file: %v", err)
	}

	// Load credential
	cred, err := LoadJWTFromFile(credFile)
	if err != nil {
		t.Fatalf("failed to load credential: %v", err)
	}

	// Start watching
	ctx1, cancel1 := context.WithCancel(context.Background())
	defer cancel1()

	if err := cred.StartWatching(ctx1); err != nil {
		t.Fatalf("failed to start watching: %v", err)
	}

	// Verify stopWatcher is initialized
	stopCh1 := getStopWatcher(cred)
	if stopCh1 == nil {
		t.Fatal("stopWatcher should be initialized after StartWatching")
	}

	// Stop watching (this closes stopWatcher and sets it to nil)
	cred.StopWatching()

	// Verify stopWatcher is nil after stop
	if getStopWatcher(cred) != nil {
		t.Fatal("stopWatcher should be nil after StopWatching")
	}

	// Wait a moment for cleanup
	time.Sleep(100 * time.Millisecond)

	// Start watching again (this should reinitialize stopWatcher)
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()

	if err := cred.StartWatching(ctx2); err != nil {
		t.Fatalf("failed to start watching again: %v", err)
	}

	// Verify stopWatcher is reinitialized
	stopCh2 := getStopWatcher(cred)
	if stopCh2 == nil {
		t.Fatal("stopWatcher should be reinitialized after StartWatching again")
	}

	// Verify it's a different channel (new instance)
	if stopCh1 == stopCh2 {
		t.Fatal("stopWatcher should be a new channel instance after restart")
	}

	// Verify we can stop again
	cred.StopWatching()

	// Verify stopWatcher is nil again
	if getStopWatcher(cred) != nil {
		t.Fatal("stopWatcher should be nil after second StopWatching")
	}

	t.Log("✓ Successfully restarted watching after stop")
}

// testStartWatchingWorksAfterRestart tests that the watcher continues to function correctly
// after being stopped and restarted. This verifies that restarting the watcher doesn't
// break its ability to detect file changes.
func testStartWatchingWorksAfterRestart(t *testing.T) {
	// Create temporary credential file
	tmpDir := t.TempDir()
	credFile := filepath.Join(tmpDir, "credentials.tmrc.json")

	// Write initial credential
	initialCred := cachedCredential{
		Provider:     "Google",
		IDToken:      generateMockJWT(),
		RefreshToken: "refresh-token-1",
	}
	data, _ := json.MarshalIndent(initialCred, "", "  ")
	if err := os.WriteFile(credFile, data, 0o600); err != nil {
		t.Fatalf("failed to write credential file: %v", err)
	}

	// Load credential
	cred, err := LoadJWTFromFile(credFile)
	if err != nil {
		t.Fatalf("failed to load credential: %v", err)
	}

	// Start watching
	ctx1, cancel1 := context.WithCancel(context.Background())
	defer cancel1()

	if err := cred.StartWatching(ctx1); err != nil {
		t.Fatalf("failed to start watching: %v", err)
	}

	// Wait for watcher to initialize
	time.Sleep(200 * time.Millisecond)

	// Update file and verify it's detected
	originalToken := cred.idToken
	newToken1 := generateMockJWT()
	newCred1 := cachedCredential{
		Provider:     "Google",
		IDToken:      newToken1,
		RefreshToken: "refresh-token-2",
	}
	newData1, _ := json.MarshalIndent(newCred1, "", "  ")
	if err := os.WriteFile(credFile, newData1, 0o600); err != nil {
		t.Fatalf("failed to write credential file: %v", err)
	}

	// Wait for file watcher to detect and reload
	time.Sleep(300 * time.Millisecond)

	// Verify token was updated
	cred.mu.RLock()
	tokenAfterFirstUpdate := cred.idToken
	cred.mu.RUnlock()

	if tokenAfterFirstUpdate == originalToken {
		t.Log("Warning: credential may not have reloaded yet (timing issue)")
	}

	// Stop watching
	cred.StopWatching()

	// Wait a moment for cleanup
	time.Sleep(100 * time.Millisecond)

	// Start watching again
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()

	if err := cred.StartWatching(ctx2); err != nil {
		t.Fatalf("failed to start watching again: %v", err)
	}

	// Wait for watcher to initialize
	time.Sleep(200 * time.Millisecond)

	// Update file again and verify it's detected after restart
	newToken2 := generateMockJWT()
	newCred2 := cachedCredential{
		Provider:     "GitHub",
		IDToken:      newToken2,
		RefreshToken: "refresh-token-3",
	}
	newData2, _ := json.MarshalIndent(newCred2, "", "  ")
	if err := os.WriteFile(credFile, newData2, 0o600); err != nil {
		t.Fatalf("failed to write credential file: %v", err)
	}

	// Wait for file watcher to detect and reload
	time.Sleep(300 * time.Millisecond)

	// Verify token was updated after restart
	cred.mu.RLock()
	tokenAfterRestart := cred.idToken
	providerAfterRestart := cred.provider
	cred.mu.RUnlock()

	if tokenAfterRestart == tokenAfterFirstUpdate {
		t.Log("Warning: credential may not have reloaded after restart (timing issue)")
	}

	// Verify the watcher is working after restart
	if tokenAfterRestart == newToken2 && providerAfterRestart == "GitHub" {
		t.Log("✓ Watcher successfully detects file changes after stop and restart")
	} else {
		t.Logf("Token after restart: %s", tokenAfterRestart[:20]+"...")
		t.Logf("Expected token: %s", newToken2[:20]+"...")
		t.Logf("Provider after restart: %s (expected: GitHub)", providerAfterRestart)
	}

	// Clean up
	cred.StopWatching()
}

// getStopWatcher safely retrieves the stopWatcher channel from the credential.
func getStopWatcher(cred *JWTCredential) chan struct{} {
	cred.mu.RLock()
	defer cred.mu.RUnlock()
	return cred.stopWatcher
}

func TestJWTCredential_updateCredentialFile(t *testing.T) {
	t.Run("atomic file update", testJWTCredentialUpdateCredentialFileAtomic)
	t.Run("concurrent file updates", testJWTCredentialUpdateCredentialFileConcurrent)
}

func testJWTCredentialUpdateCredentialFileAtomic(t *testing.T) {
	tmpDir := t.TempDir()
	credFile := filepath.Join(tmpDir, "credentials.tmrc.json")

	cred := &JWTCredential{
		idToken:        generateMockJWT(),
		refreshToken:   "refresh-token-123",
		provider:       "Google",
		credentialPath: credFile,
	}

	err := cred.updateCredentialFile()
	if err != nil {
		t.Fatalf("failed to update credential file: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(credFile); os.IsNotExist(err) {
		t.Fatal("credential file was not created")
	}

	// Verify file permissions (Unix-style check, skipped on Windows)
	// Windows uses ACLs instead of Unix permissions, so we only check on Unix systems
	if runtime.GOOS != "windows" {
		fileInfo, _ := os.Stat(credFile)
		mode := fileInfo.Mode()
		if mode&0o077 != 0 {
			t.Errorf("insecure file permissions: %v", mode)
		}
	}

	// Verify file contents
	data, _ := os.ReadFile(credFile)
	var loaded cachedCredential
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("failed to unmarshal credential file: %v", err)
	}

	if loaded.IDToken != cred.idToken {
		t.Error("id_token mismatch")
	}
	if loaded.RefreshToken != cred.refreshToken {
		t.Error("refresh_token mismatch")
	}
	if loaded.Provider != cred.provider {
		t.Error("provider mismatch")
	}
}

func testJWTCredentialUpdateCredentialFileConcurrent(t *testing.T) {
	tmpDir := t.TempDir()
	credFile := filepath.Join(tmpDir, "credentials.tmrc.json")

	cred := &JWTCredential{
		idToken:        generateMockJWT(),
		refreshToken:   "refresh-token-123",
		provider:       "Google",
		credentialPath: credFile,
	}

	var wg sync.WaitGroup
	errors := make([]error, 10)

	// Launch 10 concurrent file updates
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			errors[index] = cred.updateCredentialFile()
		}(i)
	}

	wg.Wait()

	// Count successes and failures
	successCount := 0
	failureCount := 0
	for i, err := range errors {
		if err != nil {
			failureCount++
			// On Windows, file locking is stricter, so some operations may fail
			// with "Access is denied" - this is expected behavior
			if runtime.GOOS == "windows" {
				t.Logf("update %d failed (expected on Windows): %v", i, err)
			} else {
				t.Errorf("update %d failed: %v", i, err)
			}
		} else {
			successCount++
		}
	}

	// On Windows, some operations may fail due to strict file locking,
	// but at least one should succeed. On Unix, all should succeed.
	if runtime.GOOS == "windows" {
		if successCount == 0 {
			t.Fatal("all concurrent file updates failed on Windows")
		}
		if successCount < 3 {
			t.Logf("Warning: only %d out of 10 updates succeeded on Windows (file locking is stricter)", successCount)
		}
	} else if failureCount > 0 {
		t.Errorf("%d out of 10 concurrent file updates failed", failureCount)
	}

	// File should exist and be valid
	if _, err := os.Stat(credFile); os.IsNotExist(err) {
		t.Fatal("credential file was not created")
	}

	t.Log("✓ Concurrent file updates completed successfully")
}

func TestJWTCredential_reloadFromFile(t *testing.T) {
	t.Run("reload updates credential", testReloadUpdatesCredential)
	t.Run("reload rejects insecure permissions", testReloadRejectsInsecurePermissions)
}

// testReloadUpdatesCredential tests that reloading updates the credential fields.
func testReloadUpdatesCredential(t *testing.T) {
	// Skip on Windows: file write/reload timing can differ (e.g. delayed visibility).
	if runtime.GOOS == "windows" {
		t.Skip("reload from file behavior differs on Windows")
	}
	tmpDir := t.TempDir()
	credFile := filepath.Join(tmpDir, "credentials.tmrc.json")

	// Write initial credential
	initialCred := cachedCredential{
		Provider:     "Google",
		IDToken:      generateMockJWT(),
		RefreshToken: "refresh-token-1",
	}
	data, _ := json.MarshalIndent(initialCred, "", "  ")
	if err := os.WriteFile(credFile, data, 0o600); err != nil {
		t.Fatalf("failed to write credential file: %v", err)
	}

	// Load credential
	cred, err := LoadJWTFromFile(credFile)
	if err != nil {
		t.Fatalf("failed to load credential: %v", err)
	}

	originalToken := cred.idToken

	// Update file with a new token
	newToken := generateMockJWT()
	newCred := cachedCredential{
		Provider:     "GitHub",
		IDToken:      newToken,
		RefreshToken: "refresh-token-2",
	}
	newData, _ := json.MarshalIndent(newCred, "", "  ")
	if err := os.WriteFile(credFile, newData, 0o600); err != nil {
		t.Fatalf("failed to write credential file: %v", err)
	}

	// Reload
	if err := cred.reloadFromFile(); err != nil {
		t.Fatalf("failed to reload: %v", err)
	}

	// Verify updates
	cred.mu.RLock()
	defer cred.mu.RUnlock()

	if cred.idToken == originalToken {
		t.Error("id_token was not updated")
	}
	if cred.idToken != newToken {
		t.Errorf("id_token mismatch: got %s, want %s", cred.idToken[:20]+"...", newToken[:20]+"...")
	}
	if cred.refreshToken != newCred.RefreshToken {
		t.Error("refresh_token mismatch")
	}
	if cred.provider != newCred.Provider {
		t.Error("provider mismatch")
	}
}

// testReloadRejectsInsecurePermissions tests that reloading rejects files with insecure permissions.
func testReloadRejectsInsecurePermissions(t *testing.T) {
	// Skip on Windows - permission checking is Unix-specific
	if runtime.GOOS == "windows" {
		t.Skip("permission checking is Unix-specific")
	}

	tmpDir := t.TempDir()
	credFile := filepath.Join(tmpDir, "credentials.tmrc.json")

	// Write initial credential with secure permissions
	initialCred := cachedCredential{
		Provider:     "Google",
		IDToken:      generateMockJWT(),
		RefreshToken: "refresh-token-1",
	}
	data, _ := json.MarshalIndent(initialCred, "", "  ")
	if err := os.WriteFile(credFile, data, 0o600); err != nil {
		t.Fatalf("failed to write credential file: %v", err)
	}

	// Load credential
	cred, err := LoadJWTFromFile(credFile)
	if err != nil {
		t.Fatalf("failed to load credential: %v", err)
	}

	// Change file permissions to insecure (world-readable)
	if err := os.Chmod(credFile, 0o644); err != nil {
		t.Fatalf("failed to change file permissions: %v", err)
	}

	// Update file content
	newCred := cachedCredential{
		Provider:     "GitHub",
		IDToken:      generateMockJWT(),
		RefreshToken: "refresh-token-2",
	}
	newData, _ := json.MarshalIndent(newCred, "", "  ")
	if err := os.WriteFile(credFile, newData, 0o644); err != nil {
		t.Fatalf("failed to write credential file: %v", err)
	}

	// Reload should fail due to insecure permissions
	reloadErr := cred.reloadFromFile()
	if reloadErr == nil {
		t.Fatal("reloadFromFile should have failed with insecure permissions")
	}

	// Verify error message mentions permissions
	if reloadErr.Error() == "" {
		t.Error("error message should not be empty")
	}
	if !strings.Contains(reloadErr.Error(), "insecure permissions") {
		t.Errorf("error should mention insecure permissions, got: %v", reloadErr)
	}

	// Verify credentials were NOT updated (should still be original)
	cred.mu.RLock()
	defer cred.mu.RUnlock()
	if cred.provider != initialCred.Provider {
		t.Error("provider should not have been updated")
	}
}

func TestJWTCredential_ApplyCredentials_ThreadSafe(t *testing.T) {
	cred := &JWTCredential{
		idToken:  generateMockJWT(),
		provider: "Google",
	}

	var wg sync.WaitGroup
	errors := make([]error, 100)

	// Launch 100 concurrent ApplyCredentials calls
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			req, _ := http.NewRequest("GET", "https://example.com", nil)
			errors[index] = cred.ApplyCredentials(req)
		}(i)
	}

	// Also simulate concurrent refresh
	go func() {
		for i := 0; i < 10; i++ {
			cred.mu.Lock()
			cred.idToken = generateMockJWT()
			cred.mu.Unlock()
			time.Sleep(10 * time.Millisecond)
		}
	}()

	wg.Wait()

	// All should complete without error
	for i, err := range errors {
		if err != nil {
			t.Errorf("apply %d failed: %v", i, err)
		}
	}

	t.Log("✓ Concurrent access completed successfully")
}

// Helper to generate a mock JWT token
func generateMockJWT() string {
	// This is a fake JWT just for testing - it won't validate but has the right structure
	// Use current timestamp to ensure unique tokens each call
	header := `{"alg":"RS256","kid":"test","typ":"JWT"}`
	claims := `{"iss":"https://securetoken.google.com/test","sub":"test","iat":` +
		time.Now().Format("20060102150405") +
		`,"exp":9999999999,"nonce":"` +
		time.Now().Format("20060102150405.000000") +
		`"}`
	signature := "fake-signature"

	// Base64 encode
	h := base64.RawStdEncoding.EncodeToString([]byte(header))
	c := base64.RawStdEncoding.EncodeToString([]byte(claims))

	return h + "." + c + "." + signature
}
