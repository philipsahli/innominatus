package cli

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// generateCodeVerifier creates a random 43-character code verifier for PKCE
// SECURITY: Now properly checks rand.Read errors
func generateCodeVerifier() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// generateCodeChallenge creates SHA256 hash of code verifier for PKCE
func generateCodeChallenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

// callbackServerResult holds the result of the OAuth callback
type callbackServerResult struct {
	code       string
	err        error
	shutdownFn func()
}

// generateRandomState creates a random state for CSRF protection
func generateRandomState() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// startCallbackServer starts a local HTTP server to receive OAuth callback
// SECURITY: Validates state parameter to prevent CSRF attacks
func startCallbackServer(expectedState string) (port int, callbackURL string, resultChan chan callbackServerResult) {
	resultChan = make(chan callbackServerResult, 1)

	// Use fixed port for Keycloak registration
	listener, err := net.Listen("tcp", "127.0.0.1:8082")
	if err != nil {
		resultChan <- callbackServerResult{err: fmt.Errorf("failed to start callback server: %w", err)}
		return
	}
	port = listener.Addr().(*net.TCPAddr).Port
	callbackURL = fmt.Sprintf("http://127.0.0.1:%d/callback", port)

	// Create HTTP server
	mux := http.NewServeMux()
	server := &http.Server{
		Handler: mux,
	}

	// Handle callback
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		// SECURITY: Validate state parameter (CSRF protection)
		receivedState := r.URL.Query().Get("state")
		if receivedState != expectedState {
			resultChan <- callbackServerResult{
				err: fmt.Errorf("invalid state parameter (CSRF protection)"),
				shutdownFn: func() { //nolint:errcheck // Server cleanup, error not actionable
					_ = server.Close()
				},
			}
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`
				<!DOCTYPE html>
				<html>
				<head><title>Authentication Failed</title></head>
				<body style="font-family: system-ui; padding: 40px; text-align: center;">
					<h1 style="color: #dc2626;">❌ Authentication Failed</h1>
					<p>Invalid state parameter. This may be a CSRF attack.</p>
					<p>You can close this window.</p>
				</body>
				</html>
			`))
			return
		}

		code := r.URL.Query().Get("code")
		errorParam := r.URL.Query().Get("error")
		errorDesc := r.URL.Query().Get("error_description")

		if errorParam != "" {
			msg := fmt.Sprintf("OAuth error: %s", errorParam)
			if errorDesc != "" {
				msg = fmt.Sprintf("%s - %s", msg, errorDesc)
			}
			resultChan <- callbackServerResult{
				err: fmt.Errorf("%s", msg),
				shutdownFn: func() { //nolint:errcheck // Server cleanup, error not actionable
					_ = server.Close()
				},
			}
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`
				<!DOCTYPE html>
				<html>
				<head><title>Authentication Failed</title></head>
				<body style="font-family: system-ui; padding: 40px; text-align: center;">
					<h1 style="color: #dc2626;">❌ Authentication Failed</h1>
					<p>You can close this window.</p>
				</body>
				</html>
			`))
			return
		}

		if code == "" {
			resultChan <- callbackServerResult{
				err: fmt.Errorf("no authorization code received"),
				shutdownFn: func() { //nolint:errcheck // Server cleanup, error not actionable
					_ = server.Close()
				},
			}
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`
				<!DOCTYPE html>
				<html>
				<head><title>Authentication Failed</title></head>
				<body style="font-family: system-ui; padding: 40px; text-align: center;">
					<h1 style="color: #dc2626;">❌ Authentication Failed</h1>
					<p>No authorization code received.</p>
					<p>You can close this window.</p>
				</body>
				</html>
			`))
			return
		}

		resultChan <- callbackServerResult{
			code: code,
			shutdownFn: func() { //nolint:errcheck // Server cleanup, error not actionable
				_ = server.Close()
			},
		}

		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`
			<!DOCTYPE html>
			<html>
			<head><title>Authentication Successful</title></head>
			<body style="font-family: system-ui; padding: 40px; text-align: center;">
				<h1 style="color: #16a34a;">✓ Authentication Successful</h1>
				<p>You can close this window and return to your terminal.</p>
			</body>
			</html>
		`))
	})

	// Start server in background
	go func() {
		_ = server.Serve(listener) //nolint:errcheck // Background server, error handled via context
	}()

	return port, callbackURL, resultChan
}

// oidcConfig holds OIDC configuration from server
type oidcConfig struct {
	AuthURL  string `json:"auth_url"`
	ClientID string `json:"client_id"`
	Enabled  bool   `json:"enabled"`
}

// buildOIDCAuthURL constructs the authorization URL for OIDC authentication
// SECURITY: Includes state parameter for CSRF protection
func buildOIDCAuthURL(serverURL, redirectURI, codeChallenge, state string) (string, error) {
	// Get OIDC configuration from server
	resp, err := http.Get(serverURL + "/api/oidc/config")
	if err != nil {
		return "", fmt.Errorf("failed to fetch OIDC config: %w", err)
	}
	defer func() { _ = resp.Body.Close() }() //nolint:errcheck // Defer close, error not actionable

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OIDC not enabled on server (status: %d)", resp.StatusCode)
	}

	var config oidcConfig
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return "", fmt.Errorf("failed to parse OIDC config: %w", err)
	}

	if !config.Enabled {
		return "", fmt.Errorf("OIDC authentication is not enabled on the server")
	}

	// Extract base URL (everything before the ?)
	baseAuthURL := config.AuthURL
	if idx := strings.Index(baseAuthURL, "?"); idx != -1 {
		baseAuthURL = baseAuthURL[:idx]
	}

	// Build authorization URL with CLI's redirect URI, PKCE, and state parameters
	params := url.Values{
		"client_id":             {config.ClientID},
		"redirect_uri":          {redirectURI},
		"response_type":         {"code"},
		"scope":                 {"openid profile email roles"},
		"code_challenge":        {codeChallenge},
		"code_challenge_method": {"S256"},
		"state":                 {state}, // SECURITY: CSRF protection
	}

	return baseAuthURL + "?" + params.Encode(), nil
}

// tokenResponse holds the response from token exchange
type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Username    string `json:"username"`
}

// exchangeCodeForToken exchanges authorization code for access token using PKCE
func exchangeCodeForToken(serverURL, code, codeVerifier, redirectURI string) (string, string, error) {
	data := map[string]string{
		"code":          code,
		"code_verifier": codeVerifier,
		"redirect_uri":  redirectURI,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := http.Post(
		serverURL+"/api/oidc/token",
		"application/json",
		strings.NewReader(string(jsonData)),
	)
	if err != nil {
		return "", "", fmt.Errorf("token exchange failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }() //nolint:errcheck // Defer close, error not actionable

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("token exchange failed with status %d", resp.StatusCode)
	}

	var tokenResp tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", "", fmt.Errorf("failed to parse token response: %w", err)
	}

	return tokenResp.AccessToken, tokenResp.Username, nil
}

// apiKeyResponse holds the response from API key generation
type apiKeyResponse struct {
	Key       string `json:"key"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
	ExpiresAt string `json:"expires_at"`
}

// generateAPIKeyWithToken generates API key using OIDC access token
func generateAPIKeyWithToken(serverURL, token, keyName string, expiryDays int) (apiKey, name string, expiresAt time.Time, err error) {
	req := map[string]interface{}{
		"name":        keyName,
		"expiry_days": expiryDays,
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest(
		"POST",
		serverURL+"/api/profile/api-keys",
		strings.NewReader(string(jsonData)),
	)
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("API key generation failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }() //nolint:errcheck // Defer close, error not actionable

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", "", time.Time{}, fmt.Errorf("API key generation failed with status %d", resp.StatusCode)
	}

	var apiKeyResp apiKeyResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiKeyResp); err != nil {
		return "", "", time.Time{}, fmt.Errorf("failed to parse API key response: %w", err)
	}

	// Parse expiry time
	expiresAt, err = time.Parse(time.RFC3339, apiKeyResp.ExpiresAt)
	if err != nil {
		// If parsing fails, set default expiry
		expiresAt = time.Now().Add(time.Duration(expiryDays) * 24 * time.Hour)
	}

	return apiKeyResp.Key, apiKeyResp.Name, expiresAt, nil
}

// openBrowser opens the default browser with the given URL
func openBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	return cmd.Start()
}
