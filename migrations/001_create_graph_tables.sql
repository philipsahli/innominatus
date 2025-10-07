-- Migration: Create innominatus-graph SDK tables
-- Description: Creates the schema required for graph tracking integration
-- Date: 2025-10-04

-- Create apps table
-- This table stores application metadata for graph tracking
-- Note: Using CHAR(36) for UUIDs to match GORM model expectations
CREATE TABLE IF NOT EXISTS apps (
    id CHAR(36) PRIMARY KEY,
    name VARCHAR(255) UNIQUE NOT NULL,
    description TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_apps_name ON apps(name);

-- Create nodes table
-- This table stores graph nodes (workflow, step, resource, spec)
CREATE TABLE IF NOT EXISTS nodes (
    id VARCHAR(255) PRIMARY KEY,
    app_id CHAR(36) NOT NULL REFERENCES apps(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    state VARCHAR(50) NOT NULL DEFAULT 'waiting',
    properties TEXT DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_nodes_app_id ON nodes(app_id);
CREATE INDEX IF NOT EXISTS idx_nodes_type ON nodes(type);
CREATE INDEX IF NOT EXISTS idx_nodes_state ON nodes(state);

-- Create edges table
-- This table stores relationships between nodes
CREATE TABLE IF NOT EXISTS edges (
    id VARCHAR(255) PRIMARY KEY,
    app_id CHAR(36) NOT NULL REFERENCES apps(id) ON DELETE CASCADE,
    from_node_id VARCHAR(255) NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    to_node_id VARCHAR(255) NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL,
    description TEXT,
    properties TEXT DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_edges_app_id ON edges(app_id);
CREATE INDEX IF NOT EXISTS idx_edges_from_node_id ON edges(from_node_id);
CREATE INDEX IF NOT EXISTS idx_edges_to_node_id ON edges(to_node_id);
CREATE INDEX IF NOT EXISTS idx_edges_type ON edges(type);

-- Create graph_runs table
-- This table stores graph execution metadata
CREATE TABLE IF NOT EXISTS graph_runs (
    id CHAR(36) PRIMARY KEY,
    app_id CHAR(36) NOT NULL REFERENCES apps(id) ON DELETE CASCADE,
    version INTEGER NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    started_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    error_message TEXT,
    execution_plan TEXT,
    metadata TEXT DEFAULT '{}'
);

CREATE INDEX IF NOT EXISTS idx_graph_runs_app_id ON graph_runs(app_id);
CREATE INDEX IF NOT EXISTS idx_graph_runs_status ON graph_runs(status);
