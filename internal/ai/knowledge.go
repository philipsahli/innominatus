package ai

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

// KnowledgeLoader loads documentation and examples for the RAG system
type KnowledgeLoader struct {
	docsPath      string
	workflowsPath string
}

// Document represents a document to be loaded into RAG
type Document struct {
	ID       string
	Content  string
	Metadata map[string]string
}

// NewKnowledgeLoader creates a new knowledge base loader
func NewKnowledgeLoader(docsPath, workflowsPath string) *KnowledgeLoader {
	return &KnowledgeLoader{
		docsPath:      docsPath,
		workflowsPath: workflowsPath,
	}
}

// LoadAll loads all documents from all sources
func (kl *KnowledgeLoader) LoadAll(ctx context.Context) ([]struct {
	ID       string
	Content  string
	Metadata map[string]string
}, error) {
	var allDocs []struct {
		ID       string
		Content  string
		Metadata map[string]string
	}

	// Load main documentation files
	docs, err := kl.loadDocs()
	if err != nil {
		log.Warn().Err(err).Msg("Failed to load documentation files")
	} else {
		for _, doc := range docs {
			allDocs = append(allDocs, struct {
				ID       string
				Content  string
				Metadata map[string]string
			}{
				ID:       doc.ID,
				Content:  doc.Content,
				Metadata: doc.Metadata,
			})
		}
	}

	// Load workflow definitions
	workflows, err := kl.loadWorkflows()
	if err != nil {
		log.Warn().Err(err).Msg("Failed to load workflow files")
	} else {
		for _, doc := range workflows {
			allDocs = append(allDocs, struct {
				ID       string
				Content  string
				Metadata map[string]string
			}{
				ID:       doc.ID,
				Content:  doc.Content,
				Metadata: doc.Metadata,
			})
		}
	}

	// Load README and CLAUDE.md
	rootDocs, err := kl.loadRootDocs()
	if err != nil {
		log.Warn().Err(err).Msg("Failed to load root documentation")
	} else {
		for _, doc := range rootDocs {
			allDocs = append(allDocs, struct {
				ID       string
				Content  string
				Metadata map[string]string
			}{
				ID:       doc.ID,
				Content:  doc.Content,
				Metadata: doc.Metadata,
			})
		}
	}

	// Load golden paths configuration
	goldenPaths, err := kl.loadGoldenPaths()
	if err != nil {
		log.Warn().Err(err).Msg("Failed to load golden paths")
	} else {
		for _, doc := range goldenPaths {
			allDocs = append(allDocs, struct {
				ID       string
				Content  string
				Metadata map[string]string
			}{
				ID:       doc.ID,
				Content:  doc.Content,
				Metadata: doc.Metadata,
			})
		}
	}

	log.Info().Int("total_documents", len(allDocs)).Msg("Loaded documents for knowledge base")

	return allDocs, nil
}

// loadDocs loads all markdown files from the docs directory
// with size and pattern-based filtering to stay within OpenAI token limits
func (kl *KnowledgeLoader) loadDocs() ([]Document, error) {
	var docs []Document
	var skippedPattern, skippedSize, loaded int

	log.Debug().
		Str("docs_path", kl.docsPath).
		Msg("Loading documentation files")

	// Exclude patterns to reduce token usage
	excludePatterns := []string{
		"saas-agent-architecture.md",   // Very large file (1928 lines)
		"kubernetes-deployment.md",     // Large deployment guide
		"tool-calling-architecture.md", // Large technical doc
	}

	// Maximum file size in bytes (roughly 2000 lines)
	maxFileSize := int64(100000) // ~100KB

	err := filepath.Walk(kl.docsPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Warn().
				Err(err).
				Str("path", path).
				Msg("Failed to walk documentation directory")
			return err
		}

		// Only process markdown files
		if !info.IsDir() && (strings.HasSuffix(path, ".md") || strings.HasSuffix(path, ".MD")) {
			// Get relative path from docs root
			relPath, _ := filepath.Rel(kl.docsPath, path)

			// Skip excluded patterns
			for _, pattern := range excludePatterns {
				if strings.Contains(relPath, pattern) {
					log.Debug().
						Str("file", relPath).
						Str("pattern", pattern).
						Msg("Skipping file by pattern")
					skippedPattern++
					return nil
				}
			}

			// Skip files that are too large
			if info.Size() > maxFileSize {
				log.Debug().
					Str("file", relPath).
					Int64("size_bytes", info.Size()).
					Int64("max_size_bytes", maxFileSize).
					Msg("Skipping file by size limit")
				skippedSize++
				return nil
			}

			// #nosec G304 - File path comes from filepath.Walk within trusted docs directory
			content, err := os.ReadFile(path)
			if err != nil {
				log.Warn().
					Err(err).
					Str("file", relPath).
					Msg("Failed to read documentation file")
				return nil
			}

			log.Debug().
				Str("file", relPath).
				Int("content_length", len(content)).
				Int64("size_bytes", info.Size()).
				Msg("Loaded documentation file")

			docs = append(docs, Document{
				ID:      fmt.Sprintf("doc-%s", strings.ReplaceAll(relPath, "/", "-")),
				Content: string(content),
				Metadata: map[string]string{
					"type":   "documentation",
					"source": relPath,
					"format": "markdown",
				},
			})
			loaded++
		}

		return nil
	})

	log.Debug().
		Int("loaded", loaded).
		Int("skipped_pattern", skippedPattern).
		Int("skipped_size", skippedSize).
		Int("total_docs", len(docs)).
		Msg("Loaded documentation files")

	return docs, err
}

// loadWorkflows loads workflow YAML files
func (kl *KnowledgeLoader) loadWorkflows() ([]Document, error) {
	var docs []Document
	var loaded int

	log.Debug().
		Str("workflows_path", kl.workflowsPath).
		Msg("Loading workflow files")

	err := filepath.Walk(kl.workflowsPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Warn().
				Err(err).
				Str("path", path).
				Msg("Failed to walk workflows directory")
			return err
		}

		// Only process YAML files
		if !info.IsDir() && (strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml")) {
			// Get relative path from workflows root
			relPath, _ := filepath.Rel(kl.workflowsPath, path)

			// #nosec G304 - File path comes from filepath.Walk within trusted workflows directory
			content, err := os.ReadFile(path)
			if err != nil {
				log.Warn().
					Err(err).
					Str("file", relPath).
					Msg("Failed to read workflow file")
				return nil
			}

			// Parse YAML to extract workflow name and description
			var workflow struct {
				Name        string `yaml:"name"`
				Description string `yaml:"description"`
			}
			_ = yaml.Unmarshal(content, &workflow)

			log.Debug().
				Str("file", relPath).
				Str("workflow_name", workflow.Name).
				Int("content_length", len(content)).
				Msg("Loaded workflow file")

			// Create a more readable content format
			readableContent := fmt.Sprintf("# Workflow: %s\n\nFile: %s\n\nDescription: %s\n\n```yaml\n%s\n```",
				workflow.Name,
				relPath,
				workflow.Description,
				string(content))

			docs = append(docs, Document{
				ID:      fmt.Sprintf("workflow-%s", strings.ReplaceAll(relPath, "/", "-")),
				Content: readableContent,
				Metadata: map[string]string{
					"type":     "workflow",
					"source":   relPath,
					"format":   "yaml",
					"name":     workflow.Name,
					"category": filepath.Dir(relPath),
				},
			})
			loaded++
		}

		return nil
	})

	log.Debug().
		Int("loaded", loaded).
		Int("total_workflows", len(docs)).
		Msg("Loaded workflow files")

	return docs, err
}

// loadRootDocs loads README.md (skip CLAUDE.md to reduce token usage)
func (kl *KnowledgeLoader) loadRootDocs() ([]Document, error) {
	var docs []Document

	log.Debug().Msg("Loading root documentation files")

	// Only load README.md - CLAUDE.md is too large and causes OpenAI token limit issues
	files := []string{"README.md"}
	for _, filename := range files {
		// #nosec G304 - Fixed list of trusted root documentation files
		content, err := os.ReadFile(filename)
		if err != nil {
			log.Warn().
				Err(err).
				Str("file", filename).
				Msg("Failed to read root documentation file")
			continue
		}

		log.Debug().
			Str("file", filename).
			Int("content_length", len(content)).
			Msg("Loaded root documentation file")

		docs = append(docs, Document{
			ID:      fmt.Sprintf("root-%s", strings.ToLower(strings.TrimSuffix(filename, ".md"))),
			Content: string(content),
			Metadata: map[string]string{
				"type":   "root-documentation",
				"source": filename,
				"format": "markdown",
			},
		})
	}

	log.Debug().
		Int("loaded", len(docs)).
		Msg("Loaded root documentation files")

	return docs, nil
}

// loadGoldenPaths loads golden paths configuration
// NOTE: Skipped to reduce token usage - workflow files are already loaded separately
func (kl *KnowledgeLoader) loadGoldenPaths() ([]Document, error) {
	var docs []Document

	// Skip loading goldenpaths.yaml to reduce token count
	// The individual workflow YAML files in workflows/ directory are already loaded
	// which provides the same information without the configuration overhead
	log.Info().Msg("Skipping goldenpaths.yaml to reduce token usage (workflow files loaded separately)")

	return docs, nil
}
