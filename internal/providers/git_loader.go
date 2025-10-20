package providers

import (
	"fmt"
	"innominatus/internal/logging"
	"innominatus/pkg/sdk"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
)

// GitProviderSource defines a Git repository source for a provider
type GitProviderSource struct {
	Name       string // Provider name
	Repository string // Git repository URL
	Ref        string // Git tag or branch
}

// GitLoader loads provider manifests from Git repositories
type GitLoader struct {
	cacheDir    string // Directory to cache cloned repositories
	coreVersion string // Core version for compatibility checking
	logger      *logging.ZerologAdapter
}

// NewGitLoader creates a new Git provider loader
func NewGitLoader(cacheDir string, coreVersion string) *GitLoader {
	return &GitLoader{
		cacheDir:    cacheDir,
		coreVersion: coreVersion,
		logger:      logging.NewStructuredLogger("providers.git"),
	}
}

// LoadFromGit loads a provider from a Git repository
func (g *GitLoader) LoadFromGit(source GitProviderSource) (*sdk.Provider, error) {
	g.logger.InfoWithFields("Loading provider from Git", map[string]interface{}{
		"name":       source.Name,
		"repository": source.Repository,
		"ref":        source.Ref,
	})

	// Clone or pull repository
	localPath, err := g.cloneOrPull(source)
	if err != nil {
		return nil, fmt.Errorf("failed to clone/pull repository: %w", err)
	}

	// Load provider.yaml (or legacy platform.yaml) from cloned repository
	loader := NewLoader(g.coreVersion)

	// Try provider.yaml first, then platform.yaml for backward compatibility
	providerPath := filepath.Join(localPath, "provider.yaml")
	if _, err := os.Stat(providerPath); os.IsNotExist(err) {
		providerPath = filepath.Join(localPath, "platform.yaml")
	}

	provider, err := loader.LoadFromFile(providerPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load provider manifest: %w", err)
	}

	g.logger.InfoWithFields("Provider loaded successfully", map[string]interface{}{
		"name":    provider.Metadata.Name,
		"version": provider.Metadata.Version,
	})

	return provider, nil
}

// cloneOrPull clones the repository if it doesn't exist, or pulls if it does
func (g *GitLoader) cloneOrPull(source GitProviderSource) (string, error) {
	// Sanitize repository name for directory
	repoName := sanitizeRepoName(source.Repository)
	localPath := filepath.Join(g.cacheDir, source.Name, repoName)

	// Check if repository already exists
	if _, err := os.Stat(filepath.Join(localPath, ".git")); err == nil {
		// Repository exists, pull latest
		g.logger.DebugWithFields("Pulling existing repository", map[string]interface{}{
			"path": localPath,
			"ref":  source.Ref,
		})

		repo, err := git.PlainOpen(localPath)
		if err != nil {
			return "", fmt.Errorf("failed to open repository: %w", err)
		}

		// Fetch updates
		err = repo.Fetch(&git.FetchOptions{
			RefSpecs: []config.RefSpec{"refs/*:refs/*"},
		})
		if err != nil && err != git.NoErrAlreadyUpToDate {
			g.logger.WarnWithFields("Fetch failed, will use existing clone", map[string]interface{}{
				"error": err.Error(),
			})
		}

		// Checkout the specified ref
		worktree, err := repo.Worktree()
		if err != nil {
			return "", fmt.Errorf("failed to get worktree: %w", err)
		}

		checkoutOpts, err := g.getCheckoutOptions(source.Ref)
		if err != nil {
			return "", err
		}

		err = worktree.Checkout(checkoutOpts)
		if err != nil {
			return "", fmt.Errorf("failed to checkout ref %s: %w", source.Ref, err)
		}

		return localPath, nil
	}

	// Repository doesn't exist, clone it
	g.logger.InfoWithFields("Cloning repository", map[string]interface{}{
		"repository": source.Repository,
		"path":       localPath,
	})

	// Ensure parent directory exists (0750 = owner+group read/write/execute)
	if err := os.MkdirAll(filepath.Dir(localPath), 0750); err != nil {
		return "", fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Clone repository
	repo, err := git.PlainClone(localPath, false, &git.CloneOptions{
		URL:      source.Repository,
		Progress: nil, // Could add progress reporting here
	})
	if err != nil {
		return "", fmt.Errorf("failed to clone repository: %w", err)
	}

	// Checkout the specified ref
	worktree, err := repo.Worktree()
	if err != nil {
		return "", fmt.Errorf("failed to get worktree: %w", err)
	}

	checkoutOpts, err := g.getCheckoutOptions(source.Ref)
	if err != nil {
		return "", err
	}

	err = worktree.Checkout(checkoutOpts)
	if err != nil {
		return "", fmt.Errorf("failed to checkout ref %s: %w", source.Ref, err)
	}

	g.logger.InfoWithFields("Repository cloned successfully", map[string]interface{}{
		"path": localPath,
	})

	return localPath, nil
}

// getCheckoutOptions determines checkout options based on ref (tag or branch)
func (g *GitLoader) getCheckoutOptions(ref string) (*git.CheckoutOptions, error) {
	// Try as a tag first
	tagRef := plumbing.NewTagReferenceName(ref)
	opts := &git.CheckoutOptions{
		Branch: tagRef,
	}

	// If ref doesn't start with refs/, try as a branch
	if !strings.HasPrefix(ref, "refs/") {
		// Check if it's a semver tag (starts with v)
		if strings.HasPrefix(ref, "v") {
			opts.Branch = tagRef
		} else {
			// Assume it's a branch
			opts.Branch = plumbing.NewBranchReferenceName(ref)
		}
	}

	return opts, nil
}

// sanitizeRepoName converts a repository URL to a safe directory name
func sanitizeRepoName(repo string) string {
	// Remove protocol
	name := strings.TrimPrefix(repo, "https://")
	name = strings.TrimPrefix(name, "http://")
	name = strings.TrimPrefix(name, "git@")

	// Replace special characters
	name = strings.ReplaceAll(name, ":", "_")
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, ".", "_")

	return name
}
