package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type User struct {
	ID      string `json:"id"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

type AuthService struct {
	config *oauth2.Config
}

func NewAuthService() *AuthService {
	config := &oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"),
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}

	return &AuthService{config: config}
}

func (a *AuthService) IsConfigured() bool {
	return a.config.ClientID != "" && a.config.ClientSecret != "" && a.config.RedirectURL != ""
}

func (a *AuthService) GetAuthURL(state string) string {
	return a.config.AuthCodeURL(state)
}

func (a *AuthService) ExchangeCode(ctx context.Context, code string) (*oauth2.Token, error) {
	return a.config.Exchange(ctx, code)
}

func (a *AuthService) GetUserInfo(ctx context.Context, token *oauth2.Token) (*User, error) {
	client := a.config.Client(ctx, token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	var user User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	return &user, nil
}

func GenerateState() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}