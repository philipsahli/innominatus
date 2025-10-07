package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

// HandleQueueStats returns queue statistics
func (s *Server) HandleQueueStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.workflowQueue == nil {
		http.Error(w, "Queue not available", http.StatusServiceUnavailable)
		return
	}

	stats := s.workflowQueue.GetQueueStats()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(stats); err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode queue stats: %v\n", err)
	}
}

// HandleActiveTasks returns currently executing tasks
func (s *Server) HandleActiveTasks(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.workflowQueue == nil {
		http.Error(w, "Queue not available", http.StatusServiceUnavailable)
		return
	}

	activeTasks := s.workflowQueue.GetActiveTasks()

	response := map[string]interface{}{
		"count": len(activeTasks),
		"tasks": activeTasks,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode active tasks: %v\n", err)
	}
}
