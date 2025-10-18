package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"
)

// GraphAnnotation represents a user annotation on a graph node
type GraphAnnotation struct {
	ID              int64     `json:"id"`
	ApplicationName string    `json:"application_name"`
	NodeID          string    `json:"node_id"`
	NodeName        string    `json:"node_name"`
	AnnotationText  string    `json:"annotation_text"`
	CreatedBy       string    `json:"created_by"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// handleGraphAnnotations handles /api/graph/<app>/annotations requests
func (s *Server) handleGraphAnnotations(w http.ResponseWriter, r *http.Request, appName string) {
	switch r.Method {
	case "GET":
		s.handleListAnnotations(w, r, appName)
	case "POST":
		s.handleCreateAnnotation(w, r, appName)
	case "DELETE":
		s.handleDeleteAnnotation(w, r, appName)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleListAnnotations returns all annotations for an application
func (s *Server) handleListAnnotations(w http.ResponseWriter, r *http.Request, appName string) {
	if s.db == nil {
		http.Error(w, "Database not available", http.StatusServiceUnavailable)
		return
	}

	// Optional node_id filter
	nodeID := r.URL.Query().Get("node_id")

	var rows *sql.Rows
	var err error

	if nodeID != "" {
		rows, err = s.db.DB().Query(`
			SELECT id, application_name, node_id, node_name, annotation_text, created_by, created_at, updated_at
			FROM graph_annotations
			WHERE application_name = $1 AND node_id = $2
			ORDER BY created_at DESC
		`, appName, nodeID)
	} else {
		rows, err = s.db.DB().Query(`
			SELECT id, application_name, node_id, node_name, annotation_text, created_by, created_at, updated_at
			FROM graph_annotations
			WHERE application_name = $1
			ORDER BY created_at DESC
		`, appName)
	}

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to query annotations: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	annotations := []GraphAnnotation{}
	for rows.Next() {
		var a GraphAnnotation
		err := rows.Scan(&a.ID, &a.ApplicationName, &a.NodeID, &a.NodeName,
			&a.AnnotationText, &a.CreatedBy, &a.CreatedAt, &a.UpdatedAt)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error scanning annotation: %v\n", err)
			continue
		}
		annotations = append(annotations, a)
	}

	response := map[string]interface{}{
		"application": appName,
		"annotations": annotations,
		"count":       len(annotations),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode response: %v\n", err)
	}
}

// handleCreateAnnotation creates a new annotation
func (s *Server) handleCreateAnnotation(w http.ResponseWriter, r *http.Request, appName string) {
	if s.db == nil {
		http.Error(w, "Database not available", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		NodeID         string `json:"node_id"`
		NodeName       string `json:"node_name"`
		AnnotationText string `json:"annotation_text"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.NodeID == "" || req.AnnotationText == "" {
		http.Error(w, "node_id and annotation_text are required", http.StatusBadRequest)
		return
	}

	// Get user from context
	user := s.getUserFromContext(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Insert annotation
	var id int64
	err := s.db.DB().QueryRow(`
		INSERT INTO graph_annotations (application_name, node_id, node_name, annotation_text, created_by)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, appName, req.NodeID, req.NodeName, req.AnnotationText, user.Username).Scan(&id)

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create annotation: %v", err), http.StatusInternalServerError)
		return
	}

	// Return created annotation
	var annotation GraphAnnotation
	err = s.db.DB().QueryRow(`
		SELECT id, application_name, node_id, node_name, annotation_text, created_by, created_at, updated_at
		FROM graph_annotations
		WHERE id = $1
	`, id).Scan(&annotation.ID, &annotation.ApplicationName, &annotation.NodeID, &annotation.NodeName,
		&annotation.AnnotationText, &annotation.CreatedBy, &annotation.CreatedAt, &annotation.UpdatedAt)

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch created annotation: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(annotation); err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode response: %v\n", err)
	}
}

// handleDeleteAnnotation deletes an annotation by ID
func (s *Server) handleDeleteAnnotation(w http.ResponseWriter, r *http.Request, appName string) {
	if s.db == nil {
		http.Error(w, "Database not available", http.StatusServiceUnavailable)
		return
	}

	// Get annotation ID from query parameter
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		http.Error(w, "annotation id is required", http.StatusBadRequest)
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid annotation id", http.StatusBadRequest)
		return
	}

	// Get user from context for authorization
	user := s.getUserFromContext(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Delete annotation (only if created by the user, unless admin)
	var result sql.Result
	if user.Role == "admin" {
		result, err = s.db.DB().Exec(`
			DELETE FROM graph_annotations
			WHERE id = $1 AND application_name = $2
		`, id, appName)
	} else {
		result, err = s.db.DB().Exec(`
			DELETE FROM graph_annotations
			WHERE id = $1 AND application_name = $2 AND created_by = $3
		`, id, appName, user.Username)
	}

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete annotation: %v", err), http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "Annotation not found or unauthorized", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
