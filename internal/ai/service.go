package ai

import (
	"context"
	"fmt"
	"os"

	"github.com/philipsahli/innominatus-ai-sdk/pkg/platformai"
	"github.com/philipsahli/innominatus-ai-sdk/pkg/platformai/rag"
	"github.com/rs/zerolog/log"
)

// Service provides AI assistance functionality
type Service struct {
	sdk     *platformai.SDK
	enabled bool
}

// Config holds AI service configuration
type Config struct {
	OpenAIKey     string
	AnthropicKey  string
	DocsPath      string // Path to docs directory for knowledge base
	WorkflowsPath string // Path to workflows directory
}

// NewService creates a new AI service
func NewService(ctx context.Context, cfg Config) (*Service, error) {
	log.Debug().Msg("Initializing AI service")

	// Check if AI is enabled (require both API keys)
	if cfg.OpenAIKey == "" || cfg.AnthropicKey == "" {
		log.Warn().
			Bool("has_openai_key", cfg.OpenAIKey != "").
			Bool("has_anthropic_key", cfg.AnthropicKey != "").
			Msg("AI service disabled: missing API keys (OPENAI_API_KEY and/or ANTHROPIC_API_KEY)")
		return &Service{enabled: false}, nil
	}

	log.Debug().
		Str("llm_provider", "anthropic").
		Str("llm_model", "claude-sonnet-4-5-20250929").
		Str("embedding_provider", "openai").
		Str("embedding_model", "text-embedding-3-small").
		Msg("Initializing AI SDK")

	// Initialize Platform AI SDK
	sdk, err := platformai.New(ctx, &platformai.Config{
		LLM: platformai.LLMConfig{
			Provider:    "anthropic",
			APIKey:      cfg.AnthropicKey,
			Model:       "claude-sonnet-4-5-20250929",
			Temperature: 0.7,
			MaxTokens:   4096,
		},
		RAG: &rag.Config{
			EmbeddingProvider: "openai",
			APIKey:            cfg.OpenAIKey,
			Model:             "text-embedding-3-small",
		},
	})
	if err != nil {
		log.Error().
			Err(err).
			Msg("Failed to initialize AI SDK")
		return nil, fmt.Errorf("failed to initialize AI SDK: %w", err)
	}

	log.Debug().Msg("Initialized AI SDK")

	service := &Service{
		sdk:     sdk,
		enabled: true,
	}

	// Load knowledge base
	if err := service.loadKnowledgeBase(ctx, cfg); err != nil {
		log.Warn().
			Err(err).
			Msg("Failed to load knowledge base, AI will work with limited context")
	}

	log.Info().
		Str("llm_provider", "anthropic").
		Str("llm_model", "claude-sonnet-4-5").
		Str("embedding_provider", "openai").
		Str("embedding_model", "text-embedding-3-small").
		Msg("Initialized AI service")

	return service, nil
}

// IsEnabled returns whether AI service is enabled
func (s *Service) IsEnabled() bool {
	return s.enabled
}

// GetStatus returns the current AI service status
func (s *Service) GetStatus(ctx context.Context) StatusResponse {
	log.Debug().Msg("Getting AI service status")

	if !s.enabled {
		log.Debug().Msg("AI service is not enabled")

		// Check which keys are missing
		var missingKeys []string
		if os.Getenv("OPENAI_API_KEY") == "" {
			missingKeys = append(missingKeys, "OPENAI_API_KEY")
		}
		if os.Getenv("ANTHROPIC_API_KEY") == "" {
			missingKeys = append(missingKeys, "ANTHROPIC_API_KEY")
		}

		message := "AI service is disabled. Set OPENAI_API_KEY and ANTHROPIC_API_KEY to enable."
		if len(missingKeys) > 0 {
			message = fmt.Sprintf("AI service is disabled. Missing environment variables: %v", missingKeys)
		}

		return StatusResponse{
			Enabled:     false,
			Status:      "not_configured",
			Message:     message,
			MissingKeys: missingKeys,
		}
	}

	docCount, err := s.sdk.RAG().Count(ctx)
	if err != nil {
		log.Warn().
			Err(err).
			Msg("Failed to get document count from RAG for status")
		docCount = 0
	}

	log.Debug().
		Int("document_count", docCount).
		Msg("AI service status retrieved")

	return StatusResponse{
		Enabled:         true,
		LLMProvider:     "anthropic",
		EmbeddingModel:  "text-embedding-3-small",
		DocumentsLoaded: docCount,
		Status:          "ready",
		Message:         fmt.Sprintf("AI assistant is ready with %d documents in knowledge base", docCount),
	}
}

// GetSDK returns the underlying Platform AI SDK (for direct access)
func (s *Service) GetSDK() *platformai.SDK {
	return s.sdk
}

// loadKnowledgeBase loads documentation and examples into the RAG system
func (s *Service) loadKnowledgeBase(ctx context.Context, cfg Config) error {
	if s.sdk.RAG() == nil {
		log.Error().Msg("RAG module not initialized")
		return fmt.Errorf("RAG module not initialized")
	}

	log.Debug().
		Str("docs_path", cfg.DocsPath).
		Str("workflows_path", cfg.WorkflowsPath).
		Msg("Loading knowledge base")

	// Load documents from various sources
	loader := NewKnowledgeLoader(cfg.DocsPath, cfg.WorkflowsPath)
	documents, err := loader.LoadAll(ctx)
	if err != nil {
		log.Error().
			Err(err).
			Msg("Failed to load documents from sources")
		return fmt.Errorf("failed to load documents: %w", err)
	}

	log.Debug().
		Int("documents_loaded", len(documents)).
		Msg("Adding documents to RAG index")

	// Add documents to RAG in batches to avoid OpenAI token limits
	// OpenAI text-embedding-3-small has 8192 tokens context limit per document
	// Some docs are very large, so use conservative batch size
	// Estimate ~2000 tokens per doc average = ~20 docs per batch to be safe
	batchSize := 20
	totalDocs := len(documents)

	for i := 0; i < totalDocs; i += batchSize {
		end := i + batchSize
		if end > totalDocs {
			end = totalDocs
		}

		batch := documents[i:end]
		batchNum := (i / batchSize) + 1
		totalBatches := (totalDocs + batchSize - 1) / batchSize

		log.Debug().
			Int("batch", batchNum).
			Int("total_batches", totalBatches).
			Int("batch_size", len(batch)).
			Msg("Adding document batch to RAG index")

		if err := s.sdk.RAG().AddDocuments(ctx, batch); err != nil {
			log.Error().
				Err(err).
				Int("batch", batchNum).
				Int("batch_size", len(batch)).
				Msg("Failed to add document batch to RAG index")
			return fmt.Errorf("failed to add documents batch %d to RAG: %w", batchNum, err)
		}
	}

	log.Info().
		Int("total_documents", totalDocs).
		Msg("Successfully added all documents to RAG index")

	count, err := s.sdk.RAG().Count(ctx)
	if err != nil {
		log.Warn().
			Err(err).
			Msg("Failed to get document count from RAG")
	}

	log.Info().
		Int("document_count", count).
		Msg("Loaded knowledge base")

	return nil
}

// NewServiceFromEnv creates a new AI service from environment variables
func NewServiceFromEnv(ctx context.Context) (*Service, error) {
	return NewService(ctx, Config{
		OpenAIKey:     os.Getenv("OPENAI_API_KEY"),
		AnthropicKey:  os.Getenv("ANTHROPIC_API_KEY"),
		DocsPath:      "docs",
		WorkflowsPath: "workflows",
	})
}
