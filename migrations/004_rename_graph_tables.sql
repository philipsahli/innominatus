-- Migration: Rename graph tables with graph_ prefix for single database architecture
-- Description: Consolidates graph tracking tables in main database with clear naming
-- Date: 2025-10-05

-- Drop existing unprefixed graph tables if they exist (from previous two-database setup)
DROP TABLE IF EXISTS edges CASCADE;
DROP TABLE IF EXISTS nodes CASCADE;
DROP TABLE IF EXISTS graph_runs CASCADE;
DROP TABLE IF EXISTS apps CASCADE;

-- Create graph_apps table (renamed from apps)
CREATE TABLE IF NOT EXISTS graph_apps (
    id CHAR(36) PRIMARY KEY,
    name VARCHAR(255) UNIQUE NOT NULL,
    description TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_graph_apps_name ON graph_apps(name);

-- Create graph_nodes table (renamed from nodes)
CREATE TABLE IF NOT EXISTS graph_nodes (
    id VARCHAR(255) PRIMARY KEY,
    app_id CHAR(36) NOT NULL REFERENCES graph_apps(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    state VARCHAR(50) NOT NULL DEFAULT 'waiting',
    properties TEXT DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_graph_nodes_app_id ON graph_nodes(app_id);
CREATE INDEX IF NOT EXISTS idx_graph_nodes_type ON graph_nodes(type);
CREATE INDEX IF NOT EXISTS idx_graph_nodes_state ON graph_nodes(state);

-- Create graph_edges table (renamed from edges)
CREATE TABLE IF NOT EXISTS graph_edges (
    id VARCHAR(255) PRIMARY KEY,
    app_id CHAR(36) NOT NULL REFERENCES graph_apps(id) ON DELETE CASCADE,
    from_node_id VARCHAR(255) NOT NULL REFERENCES graph_nodes(id) ON DELETE CASCADE,
    to_node_id VARCHAR(255) NOT NULL REFERENCES graph_nodes(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL,
    description TEXT,
    properties TEXT DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_graph_edges_app_id ON graph_edges(app_id);
CREATE INDEX IF NOT EXISTS idx_graph_edges_from_node_id ON graph_edges(from_node_id);
CREATE INDEX IF NOT EXISTS idx_graph_edges_to_node_id ON graph_edges(to_node_id);
CREATE INDEX IF NOT EXISTS idx_graph_edges_type ON graph_edges(type);

-- Create graph_runs table (already has graph_ prefix)
CREATE TABLE IF NOT EXISTS graph_runs (
    id CHAR(36) PRIMARY KEY,
    app_id CHAR(36) NOT NULL REFERENCES graph_apps(id) ON DELETE CASCADE,
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
