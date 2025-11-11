package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
)

// HandleListProviders returns a list of all loaded providers
func (s *Server) HandleListProviders(w http.ResponseWriter, r *http.Request) {
	// Check if provider registry is available
	if s.providerRegistry == nil {
		http.Error(w, "Provider registry not available", http.StatusServiceUnavailable)
		return
	}

	// Get all providers
	providers := s.providerRegistry.ListProviders()

	// Transform to response format
	type WorkflowSummary struct {
		Name        string   `json:"name"`
		Description string   `json:"description"`
		Category    string   `json:"category"`
		Version     string   `json:"version,omitempty"`
		Tags        []string `json:"tags,omitempty"`
	}

	type ProviderSummary struct {
		Name         string            `json:"name"`
		Version      string            `json:"version"`
		Category     string            `json:"category"`
		Description  string            `json:"description"`
		Provisioners int               `json:"provisioners"`
		GoldenPaths  int               `json:"golden_paths"`
		Workflows    []WorkflowSummary `json:"workflows"`
	}

	response := make([]ProviderSummary, len(providers))
	for i, p := range providers {
		// Collect workflows
		workflows := make([]WorkflowSummary, 0, len(p.Workflows))
		for _, w := range p.Workflows {
			workflows = append(workflows, WorkflowSummary{
				Name:        w.Name,
				Description: w.Description,
				Category:    w.Category,
				Version:     w.Version,
				Tags:        w.Tags,
			})
		}

		response[i] = ProviderSummary{
			Name:         p.Metadata.Name,
			Version:      p.Metadata.Version,
			Category:     p.Metadata.Category,
			Description:  p.Metadata.Description,
			Provisioners: len(p.Provisioners),
			GoldenPaths:  len(p.GoldenPaths),
			Workflows:    workflows,
		}
	}

	// Sort providers alphabetically by name
	sort.Slice(response, func(i, j int) bool {
		return response[i].Name < response[j].Name
	})

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode response: %v\n", err)
	}
}

// HandleProviderStats returns statistics about loaded providers
func (s *Server) HandleProviderStats(w http.ResponseWriter, r *http.Request) {
	// Check if provider registry is available
	if s.providerRegistry == nil {
		http.Error(w, "Provider registry not available", http.StatusServiceUnavailable)
		return
	}

	// Get counts
	providerCount, provisionerCount := s.providerRegistry.Count()

	response := map[string]interface{}{
		"providers":    providerCount,
		"provisioners": provisionerCount,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode response: %v\n", err)
	}
}
