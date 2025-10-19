package server

import (
	"encoding/json"
	"net/http"
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
	type ProviderSummary struct {
		Name         string `json:"name"`
		Version      string `json:"version"`
		Category     string `json:"category"`
		Description  string `json:"description"`
		Provisioners int    `json:"provisioners"`
		GoldenPaths  int    `json:"golden_paths"`
	}

	response := make([]ProviderSummary, len(providers))
	for i, p := range providers {
		response[i] = ProviderSummary{
			Name:         p.Metadata.Name,
			Version:      p.Metadata.Version,
			Category:     p.Metadata.Category,
			Description:  p.Metadata.Description,
			Provisioners: len(p.Provisioners),
			GoldenPaths:  len(p.GoldenPaths),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
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
	json.NewEncoder(w).Encode(response)
}
