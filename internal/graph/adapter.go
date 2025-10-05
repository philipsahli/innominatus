package graph

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"idp-orchestrator/pkg/export"
	sdk "idp-orchestrator/pkg/graph"
	"idp-orchestrator/pkg/storage"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Adapter wraps the innominatus-graph SDK repository
// and provides a clean API for the orchestrator
type Adapter struct {
	Repo   *storage.Repository
	gormDB *gorm.DB
}

// NewAdapter creates a new graph adapter using database connection parameters
// It creates a separate GORM connection to avoid prepared statement issues
func NewAdapter(sqlDB *sql.DB) (*Adapter, error) {
	if sqlDB == nil {
		return nil, fmt.Errorf("sql.DB cannot be nil")
	}

	// Get database connection string from environment
	// Create a NEW GORM connection instead of wrapping existing sql.DB
	// This avoids prepared statement parameter binding issues
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "postgres")
	password := getEnv("DB_PASSWORD", "")
	// Graph tracking now uses same database with graph_ prefixed tables
	dbname := getEnv("DB_NAME", "idp_orchestrator")
	sslmode := getEnv("DB_SSLMODE", "disable")

	// Build connection string - omit password if empty to avoid lib/pq default behavior
	dsn := fmt.Sprintf("host=%s port=%s user=%s dbname=%s sslmode=%s search_path=public",
		host, port, user, dbname, sslmode)
	if password != "" {
		dsn = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s search_path=public",
			host, port, user, password, dbname, sslmode)
	}

	gormDB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		SkipDefaultTransaction:                   true,
		PrepareStmt:                              false,
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create GORM connection: %w", err)
	}

	// Skip AutoMigrate - tables are created via SQL migration
	// See: migrations/001_create_graph_tables.sql

	repo := storage.NewRepository(gormDB)

	return &Adapter{
		Repo:   repo,
		gormDB: gormDB,
	}, nil
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultVal string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultVal
}

// AddNode adds a node to the graph for a specific run/application
func (a *Adapter) AddNode(runID string, node *sdk.Node) error {
	if node == nil {
		return fmt.Errorf("node cannot be nil")
	}

	// Load or create graph for this run
	graph, err := a.loadOrCreateGraph(runID)
	if err != nil {
		return fmt.Errorf("failed to load/create graph: %w", err)
	}

	// Add node to graph
	if err := graph.AddNode(node); err != nil {
		return fmt.Errorf("failed to add node to graph: %w", err)
	}

	// Persist graph
	if err := a.Repo.SaveGraph(runID, graph); err != nil {
		return fmt.Errorf("failed to save graph: %w", err)
	}

	log.Printf("Graph: added node %s (type: %s, state: %s) to %s", node.ID, node.Type, node.State, runID)
	return nil
}

// AddEdge adds an edge between two nodes in the graph
func (a *Adapter) AddEdge(runID string, edge *sdk.Edge) error {
	if edge == nil {
		return fmt.Errorf("edge cannot be nil")
	}

	// Load graph for this run
	graph, err := a.Repo.LoadGraph(runID)
	if err != nil {
		return fmt.Errorf("failed to load graph: %w", err)
	}

	// Add edge to graph
	if err := graph.AddEdge(edge); err != nil {
		return fmt.Errorf("failed to add edge to graph: %w", err)
	}

	// Persist graph
	if err := a.Repo.SaveGraph(runID, graph); err != nil {
		return fmt.Errorf("failed to save graph: %w", err)
	}

	log.Printf("Graph: added edge %s (%s → %s, type: %s) to %s",
		edge.ID, edge.FromNodeID, edge.ToNodeID, edge.Type, runID)
	return nil
}

// UpdateNodeState updates the state of a node in the graph
// This will trigger state propagation according to SDK rules
func (a *Adapter) UpdateNodeState(runID, nodeID string, state sdk.NodeState) error {
	// Load graph for this run
	graph, err := a.Repo.LoadGraph(runID)
	if err != nil {
		return fmt.Errorf("failed to load graph: %w", err)
	}

	// Get current node state for logging
	node, exists := graph.GetNode(nodeID)
	if !exists {
		return fmt.Errorf("node %s not found in graph %s", nodeID, runID)
	}

	oldState := node.State

	// Update node state (with automatic propagation)
	if err := graph.UpdateNodeState(nodeID, state); err != nil {
		return fmt.Errorf("failed to update node state: %w", err)
	}

	// Persist graph
	if err := a.Repo.SaveGraph(runID, graph); err != nil {
		return fmt.Errorf("failed to save graph: %w", err)
	}

	log.Printf("Graph: updated node %s state: %s → %s in %s", nodeID, oldState, state, runID)

	// If state changed to failed, check if parent workflow was also updated
	if state == sdk.NodeStateFailed && node.Type == sdk.NodeTypeStep {
		if parent, err := graph.GetParentWorkflow(nodeID); err == nil {
			if parent.State == sdk.NodeStateFailed {
				log.Printf("Graph: propagated failure to parent workflow %s", parent.ID)
			}
		}
	}

	return nil
}

// ExportGraph exports the graph in the specified format (dot, svg, png)
func (a *Adapter) ExportGraph(runID, format string) ([]byte, error) {
	// Load graph for this run
	graph, err := a.Repo.LoadGraph(runID)
	if err != nil {
		return nil, fmt.Errorf("failed to load graph: %w", err)
	}

	// Export using SDK exporter
	exporter := export.NewExporter()
	defer exporter.Close()

	// Convert format string to export.Format type
	var exportFormat export.Format
	switch format {
	case "svg":
		exportFormat = export.FormatSVG
	case "png":
		exportFormat = export.FormatPNG
	case "dot":
		exportFormat = export.FormatDOT
	default:
		exportFormat = export.FormatSVG // default to SVG
	}

	data, err := exporter.ExportGraph(graph, exportFormat)
	if err != nil {
		return nil, fmt.Errorf("failed to export graph: %w", err)
	}

	return data, nil
}

// GetGraph retrieves the full graph for a run
func (a *Adapter) GetGraph(runID string) (*sdk.Graph, error) {
	graph, err := a.Repo.LoadGraph(runID)
	if err != nil {
		return nil, fmt.Errorf("failed to load graph for %s: %w", runID, err)
	}
	return graph, nil
}

// GetNodesByState returns all nodes in a specific state for a run
func (a *Adapter) GetNodesByState(runID string, state sdk.NodeState) ([]*sdk.Node, error) {
	graph, err := a.Repo.LoadGraph(runID)
	if err != nil {
		return nil, fmt.Errorf("failed to load graph: %w", err)
	}

	return graph.GetNodesByState(state), nil
}

// GetNodesByType returns all nodes of a specific type for a run
func (a *Adapter) GetNodesByType(runID string, nodeType sdk.NodeType) ([]*sdk.Node, error) {
	graph, err := a.Repo.LoadGraph(runID)
	if err != nil {
		return nil, fmt.Errorf("failed to load graph: %w", err)
	}

	return graph.GetNodesByType(nodeType), nil
}

// loadOrCreateGraph loads an existing graph or creates a new one if it doesn't exist
func (a *Adapter) loadOrCreateGraph(runID string) (*sdk.Graph, error) {
	graph, err := a.Repo.LoadGraph(runID)
	if err != nil {
		// Graph doesn't exist, create a new one
		graph = sdk.NewGraph(runID)
		log.Printf("Graph: created new graph for %s", runID)
	}
	return graph, nil
}

// Close closes the adapter (if needed in the future)
func (a *Adapter) Close() error {
	// GORM DB shares the underlying sql.DB, so we don't close it here
	return nil
}
