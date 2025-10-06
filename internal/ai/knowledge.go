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
func (kl *KnowledgeLoader) loadDocs() ([]Document, error) {
	var docs []Document

	err := filepath.Walk(kl.docsPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Only process markdown files
		if !info.IsDir() && (strings.HasSuffix(path, ".md") || strings.HasSuffix(path, ".MD")) {
			// #nosec G304 - File path comes from filepath.Walk within trusted docs directory
			content, err := os.ReadFile(path)
			if err != nil {
				log.Warn().Err(err).Str("file", path).Msg("Failed to read documentation file")
				return nil
			}

			// Get relative path from docs root
			relPath, _ := filepath.Rel(kl.docsPath, path)

			docs = append(docs, Document{
				ID:      fmt.Sprintf("doc-%s", strings.ReplaceAll(relPath, "/", "-")),
				Content: string(content),
				Metadata: map[string]string{
					"type":   "documentation",
					"source": relPath,
					"format": "markdown",
				},
			})
		}

		return nil
	})

	return docs, err
}

// loadWorkflows loads workflow YAML files
func (kl *KnowledgeLoader) loadWorkflows() ([]Document, error) {
	var docs []Document

	err := filepath.Walk(kl.workflowsPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Only process YAML files
		if !info.IsDir() && (strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml")) {
			// #nosec G304 - File path comes from filepath.Walk within trusted workflows directory
			content, err := os.ReadFile(path)
			if err != nil {
				log.Warn().Err(err).Str("file", path).Msg("Failed to read workflow file")
				return nil
			}

			// Parse YAML to extract workflow name and description
			var workflow struct {
				Name        string `yaml:"name"`
				Description string `yaml:"description"`
			}
			_ = yaml.Unmarshal(content, &workflow)

			// Get relative path from workflows root
			relPath, _ := filepath.Rel(kl.workflowsPath, path)

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
		}

		return nil
	})

	return docs, err
}

// loadRootDocs loads README.md and CLAUDE.md
func (kl *KnowledgeLoader) loadRootDocs() ([]Document, error) {
	var docs []Document

	files := []string{"README.md", "CLAUDE.md"}
	for _, filename := range files {
		// #nosec G304 - Fixed list of trusted root documentation files
		content, err := os.ReadFile(filename)
		if err != nil {
			log.Warn().Err(err).Str("file", filename).Msg("Failed to read root documentation")
			continue
		}

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

	return docs, nil
}

// loadGoldenPaths loads golden paths configuration
func (kl *KnowledgeLoader) loadGoldenPaths() ([]Document, error) {
	var docs []Document

	content, err := os.ReadFile("goldenpaths.yaml")
	if err != nil {
		return docs, err
	}

	// Parse golden paths YAML
	var config struct {
		GoldenPaths map[string]string `yaml:"goldenpaths"`
	}
	if err := yaml.Unmarshal(content, &config); err != nil {
		return docs, err
	}

	// Create a readable document about golden paths
	var builder strings.Builder
	builder.WriteString("# Golden Paths Configuration\n\n")
	builder.WriteString("Available golden path workflows:\n\n")

	for name, path := range config.GoldenPaths {
		builder.WriteString(fmt.Sprintf("- **%s**: %s\n", name, path))
	}

	builder.WriteString("\n\n```yaml\n")
	builder.Write(content)
	builder.WriteString("\n```")

	docs = append(docs, Document{
		ID:      "golden-paths-config",
		Content: builder.String(),
		Metadata: map[string]string{
			"type":   "configuration",
			"source": "goldenpaths.yaml",
			"format": "yaml",
		},
	})

	return docs, nil
}
