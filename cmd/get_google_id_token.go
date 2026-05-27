package cmd

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"CredChain_Golang/config"
	infraLogger "CredChain_Golang/infrastructure/logger"

	"github.com/spf13/cobra"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func init() {
	rootCmd.AddCommand(getGoogleIdTokenCmd)
}

// getGoogleIdTokenResponse represents Google's OAuth 2.0 token endpoint response.
// See: https://developers.google.com/identity/protocols/oauth2/web-server#exchange-authorization-code
type getGoogleIdTokenResponse struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope"`
	TokenType    string `json:"token_type"`
	IDToken      string `json:"id_token"`
}

// getGoogleIdTokenExchangeCodeForTokens exchanges an authorization code for OAuth tokens from Google.
//
// This function performs the "Authorization Code Exchange" step of the OAuth 2.0 flow:
// 1. Constructs a POST request to Google's token endpoint
// 2. Sends the authorization code, client credentials, and redirect URI
// 3. Parses the response containing access token, refresh token, and ID token
//
// The ID token is an OpenID Connect JWT that contains the user's identity claims.
func getGoogleIdTokenExchangeCodeForTokens(clientID, clientSecret, redirectURI, code string) (*getGoogleIdTokenResponse, error) {
	const tokenEndpoint = "https://oauth2.googleapis.com/token"

	// Build form-encoded request body
	formData := url.Values{}
	formData.Set("code", code)
	formData.Set("client_id", clientID)
	formData.Set("client_secret", clientSecret)
	formData.Set("redirect_uri", redirectURI)
	formData.Set("grant_type", "authorization_code")

	// Create HTTP request
	req, err := http.NewRequest(http.MethodPost, tokenEndpoint, strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Execute request with timeout
	httpClient := &http.Client{Timeout: 30 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute token request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read token response: %w", err)
	}

	// Check for HTTP error
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	// Parse JSON response
	var tokenResp getGoogleIdTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	// Validate ID token is present (required for our use case)
	if tokenResp.IDToken == "" {
		return nil, fmt.Errorf("no id_token in response — ensure 'openid' scope is included in authorization URL")
	}

	return &tokenResp, nil
}

// getGoogleIdTokenOpenBrowser opens the given URL in the user's default web browser.
// Supports macOS (open), Linux (xdg-open), and Windows (start).
func getGoogleIdTokenOpenBrowser(targetURL string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", targetURL)
	case "linux":
		cmd = exec.Command("xdg-open", targetURL)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", targetURL)
	default:
		return fmt.Errorf("unsupported platform %s — please open the URL manually", runtime.GOOS)
	}

	return cmd.Start()
}

// getGoogleIdTokenCmd is the Cobra command for obtaining a Google ID token.
var getGoogleIdTokenCmd = &cobra.Command{
	Use:   "get-google-id-token",
	Short: "Obtain a Google ID token via OAuth 2.0 Authorization Code flow",
	Long: `Starts a local HTTP server, opens the browser for Google login,
and exchanges the authorization code for tokens. Prints the ID token
to stdout for use with Postman or other testing tools.

The Google Client ID, Secret, and Redirect URI are read from .env.
The server listens on the port specified in GOOGLE_REDIRECT_URI (default: :3000).

Workflow:
  1. Local server starts on configured port
  2. Browser opens to Google's authorization page
  3. User logs in and consents
  4. Google redirects to configured callback URL with authorization code
  5. Server receives the code and exchanges it for tokens
  6. ID token is printed to stdout`,
	Run: func(cmd *cobra.Command, args []string) {
		fx.New(
			infraLogger.Module,
			fx.Provide(NewConfigFromCmd(cmd)),
			fx.Invoke(getGoogleIdToken),
		).Run()
	},
}

// getGoogleIdToken orchestrates the full OAuth 2.0 Authorization Code flow.
//
// Flow:
//  1. Start local HTTP server to receive OAuth callback
//  2. Open browser to Google's authorization URL
//  3. Wait for callback with authorization code
//  4. Exchange code for token (access token, refresh token, ID token)
//  5. Print ID token to stdout
//  6. Shutdown server and exit
func getGoogleIdToken(cfg *config.Config, logger *zap.Logger) error {
	// Determine redirect URI (use config or default)
	redirectURI := *cfg.GoogleRedirectURI

	// Channels for communication between HTTP handler and main goroutine
	codeCh := make(chan string, 1) // Receives authorization code from callback
	errCh := make(chan error, 1)   // Receives errors from callback
	var once sync.Once             // Ensures only one result is sent

	// Generate random state for CSRF protection
	stateBytes := make([]byte, 16)
	if _, err := rand.Read(stateBytes); err != nil {
		return fmt.Errorf("failed to generate state: %w", err)
	}
	state := hex.EncodeToString(stateBytes)

	// Create HTTP handler for OAuth callback endpoint
	mux := http.NewServeMux()
	mux.HandleFunc("/google/callback", func(w http.ResponseWriter, r *http.Request) {
		// Verify state parameter (CSRF protection)
		if r.URL.Query().Get("state") != state {
			once.Do(func() {
				errCh <- fmt.Errorf("invalid state parameter — possible CSRF attack")
			})
			http.Error(w, "Invalid state parameter", http.StatusBadRequest)
			return
		}

		// Check for OAuth error (user denied consent, etc.)
		if errParam := r.URL.Query().Get("error"); errParam != "" {
			errDesc := r.URL.Query().Get("error_description")
			once.Do(func() {
				errCh <- fmt.Errorf("OAuth error: %s — %s", errParam, errDesc)
			})
			http.Error(w, "OAuth error: "+errParam, http.StatusBadRequest)
			return
		}

		// Extract authorization code from query parameters
		code := r.URL.Query().Get("code")
		if code == "" {
			once.Do(func() {
				errCh <- fmt.Errorf("no authorization code received in callback")
			})
			http.Error(w, "No authorization code received", http.StatusBadRequest)
			return
		}

		// Show success page to user
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`
			<html><body style="font-family:sans-serif;text-align:center;padding:40px;">
			<h2>Login successful!</h2>
			<p>You can close this window and return to the terminal.</p>
			</body></html>
		`))

		// Send code to main goroutine
		once.Do(func() {
			codeCh <- code
		})
	})

	// Extract port from redirect URI for server binding
	serverAddr := ":3000" // default
	if parsedURL, err := url.Parse(redirectURI); err == nil && parsedURL.Port() != "" {
		serverAddr = ":" + parsedURL.Port()
	}

	// Start local HTTP server
	server := &http.Server{
		Addr:    serverAddr,
		Handler: mux,
	}

	go func() {
		logger.Info("starting callback server", zap.String("addr", serverAddr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			once.Do(func() {
				errCh <- fmt.Errorf("callback server error: %w", err)
			})
		}
	}()

	// Brief delay to ensure server is ready before opening browser
	time.Sleep(500 * time.Millisecond)

	// Construct Google OAuth 2.0 authorization URL
	// Scope: openid (required for ID token), email, profile
	// access_type=offline: requests refresh token
	// prompt=consent: forces consent screen (ensures refresh token is returned)
	authURL := fmt.Sprintf(
		"https://accounts.google.com/o/oauth2/v2/auth?"+
			"client_id=%s&"+
			"redirect_uri=%s&"+
			"response_type=code&"+
			"scope=openid%%20email%%20profile&"+
			"access_type=offline&"+
			"prompt=consent&"+
			"state=%s",
		*cfg.GoogleClientID,
		redirectURI,
		state,
	)

	logger.Info("opening browser for Google login...")

	// Open browser (warn if it fails)
	if err := getGoogleIdTokenOpenBrowser(authURL); err != nil {
		logger.Warn("failed to open browser automatically", zap.Error(err))
		logger.Info("please open this URL manually", zap.String("url", authURL))
	}

	// Wait for result: authorization code, error, or timeout
	select {
	case code := <-codeCh:
		logger.Info("authorization code received, exchanging for tokens...")

		// Exchange authorization code for tokens
		tokenResp, err := getGoogleIdTokenExchangeCodeForTokens(
			*cfg.GoogleClientID,
			*cfg.GoogleClientSecret,
			redirectURI,
			code,
		)
		if err != nil {
			return fmt.Errorf("token exchange failed: %w", err)
		}

		logger.Info("Google ID token obtained", zap.String("token", tokenResp.IDToken))

		logger.Info("ID token obtained successfully")

		// Graceful server shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(ctx)

		return nil

	case err := <-errCh:
		server.Close()
		return err

	case <-time.After(5 * time.Minute):
		server.Close()
		return fmt.Errorf("timed out waiting for authorization code (5 minutes)")
	}
}
