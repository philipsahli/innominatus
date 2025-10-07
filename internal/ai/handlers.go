package ai

import (
	"encoding/json"
	"net/http"

	"github.com/rs/zerolog/log"
)

// HandleChat handles chat requests from the web UI
func (s *Service) HandleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !s.enabled {
		http.Error(w, "AI service is not enabled. Set OPENAI_API_KEY and ANTHROPIC_API_KEY.", http.StatusServiceUnavailable)
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Extract auth token from request header for tool execution
	authHeader := r.Header.Get("Authorization")
	var authToken string
	if authHeader != "" {
		// Extract token from "Bearer <token>"
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			authToken = authHeader[7:]
		}
	}
	req.AuthToken = authToken

	// Process chat request
	response, err := s.Chat(r.Context(), req)
	if err != nil {
		log.Error().Err(err).Msg("Failed to process chat request")
		http.Error(w, "Failed to generate AI response", http.StatusInternalServerError)
		return
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Error().Err(err).Msg("Failed to encode chat response")
	}
}

// HandleGenerateSpec handles spec generation requests
func (s *Service) HandleGenerateSpec(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !s.enabled {
		http.Error(w, "AI service is not enabled. Set OPENAI_API_KEY and ANTHROPIC_API_KEY.", http.StatusServiceUnavailable)
		return
	}

	var req GenerateSpecRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Generate spec
	response, err := s.GenerateSpec(r.Context(), req)
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate spec")
		http.Error(w, "Failed to generate specification", http.StatusInternalServerError)
		return
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Error().Err(err).Msg("Failed to encode spec generation response")
	}
}

// HandleStatus handles AI service status requests
func (s *Service) HandleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status := s.GetStatus(r.Context())

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(status); err != nil {
		log.Error().Err(err).Msg("Failed to encode status response")
	}
}
