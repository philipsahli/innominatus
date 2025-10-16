package auth

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// OIDCConfig holds OIDC configuration
type OIDCConfig struct {
	Enabled      bool
	IssuerURL    string
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

// OIDCAuthenticator handles OIDC authentication
type OIDCAuthenticator struct {
	provider     *oidc.Provider
	verifier     *oidc.IDTokenVerifier
	oauth2Config *oauth2.Config
	enabled      bool
}

// UserInfo contains user information from OIDC token
type UserInfo struct {
	Subject           string
	Email             string
	EmailVerified     bool
	PreferredUsername string
	Name              string
	GivenName         string
	FamilyName        string
	Roles             []string
}

// LoadOIDCConfig loads OIDC configuration from environment variables
func LoadOIDCConfig() OIDCConfig {
	return OIDCConfig{
		Enabled:      os.Getenv("OIDC_ENABLED") == "true",
		IssuerURL:    getEnvOrDefault("OIDC_ISSUER_URL", "http://keycloak.localtest.me/realms/demo-realm"),
		ClientID:     getEnvOrDefault("OIDC_CLIENT_ID", "innominatus"),
		ClientSecret: getEnvOrDefault("OIDC_CLIENT_SECRET", "innominatus-client-secret"),
		RedirectURL:  getEnvOrDefault("OIDC_REDIRECT_URL", "http://localhost:8081/auth/callback"),
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// NewOIDCAuthenticator creates a new OIDC authenticator
func NewOIDCAuthenticator(cfg OIDCConfig) (*OIDCAuthenticator, error) {
	if !cfg.Enabled {
		return &OIDCAuthenticator{enabled: false}, nil
	}

	// Use a timeout context to prevent blocking indefinitely
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Initialize provider
	provider, err := oidc.NewProvider(ctx, cfg.IssuerURL)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize OIDC provider: %w", err)
	}

	// Create ID token verifier
	verifier := provider.Verifier(&oidc.Config{
		ClientID: cfg.ClientID,
	})

	// Configure OAuth2
	oauth2Config := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email", "roles"},
	}

	return &OIDCAuthenticator{
		provider:     provider,
		verifier:     verifier,
		oauth2Config: oauth2Config,
		enabled:      true,
	}, nil
}

// IsEnabled returns whether OIDC is enabled
func (a *OIDCAuthenticator) IsEnabled() bool {
	return a.enabled
}

// AuthCodeURL generates the authorization URL
func (a *OIDCAuthenticator) AuthCodeURL(state string) string {
	if !a.enabled {
		return ""
	}
	return a.oauth2Config.AuthCodeURL(state)
}

// Exchange exchanges the authorization code for a token
func (a *OIDCAuthenticator) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
	if !a.enabled {
		return nil, fmt.Errorf("OIDC not enabled")
	}
	return a.oauth2Config.Exchange(ctx, code)
}

// VerifyIDToken verifies and parses the ID token
func (a *OIDCAuthenticator) VerifyIDToken(ctx context.Context, rawIDToken string) (*UserInfo, error) {
	if !a.enabled {
		return nil, fmt.Errorf("OIDC not enabled")
	}

	// Verify ID token
	idToken, err := a.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, fmt.Errorf("failed to verify ID token: %w", err)
	}

	// Extract claims
	var claims struct {
		Email             string   `json:"email"`
		EmailVerified     bool     `json:"email_verified"`
		PreferredUsername string   `json:"preferred_username"`
		Name              string   `json:"name"`
		GivenName         string   `json:"given_name"`
		FamilyName        string   `json:"family_name"`
		Roles             []string `json:"roles"`
	}

	if err := idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("failed to parse claims: %w", err)
	}

	return &UserInfo{
		Subject:           idToken.Subject,
		Email:             claims.Email,
		EmailVerified:     claims.EmailVerified,
		PreferredUsername: claims.PreferredUsername,
		Name:              claims.Name,
		GivenName:         claims.GivenName,
		FamilyName:        claims.FamilyName,
		Roles:             claims.Roles,
	}, nil
}
