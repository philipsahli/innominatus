package security

import (
	"fmt"
	"net/url"
	"strings"
)

// ValidateArgoCDURL validates that an ArgoCD URL is safe to use
// This prevents SSRF (Server-Side Request Forgery) attacks
func ValidateArgoCDURL(argoCDURL string) error {
	// Parse the URL
	parsedURL, err := url.Parse(argoCDURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	// Only allow HTTP and HTTPS schemes
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("invalid URL scheme: %s (only http and https allowed)", parsedURL.Scheme)
	}

	// Check for forbidden hosts (localhost, internal IPs, etc.)
	hostname := strings.ToLower(parsedURL.Hostname())

	// Block access to localhost and loopback addresses (except explicitly allowed)
	forbiddenHosts := []string{
		"localhost",
		"127.0.0.1",
		"::1",
		"0.0.0.0",
		"169.254.", // link-local addresses
		"10.",      // private network
		"172.16.",  // private network
		"192.168.", // private network
	}

	// Allow specific local services for demo environment
	allowedLocalHosts := []string{
		"argocd.localtest.me",
		"argocd-server.argocd.svc.cluster.local",
		"argocd-server.argocd",
	}

	isAllowedLocal := false
	for _, allowed := range allowedLocalHosts {
		if hostname == allowed {
			isAllowedLocal = true
			break
		}
	}

	if !isAllowedLocal {
		for _, forbidden := range forbiddenHosts {
			if hostname == forbidden || strings.HasPrefix(hostname, forbidden) {
				return fmt.Errorf("access to internal/private addresses not allowed: %s", hostname)
			}
		}
	}

	return nil
}

// ValidateExternalURL validates URLs for external services
func ValidateExternalURL(serviceURL string) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	// Only allow HTTP and HTTPS
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("invalid URL scheme: %s (only http and https allowed)", parsedURL.Scheme)
	}

	return nil
}
