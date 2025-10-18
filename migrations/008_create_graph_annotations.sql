-- Migration: Create graph annotations table
-- Description: Store user annotations/notes on graph nodes for collaboration

CREATE TABLE IF NOT EXISTS graph_annotations (
    id SERIAL PRIMARY KEY,
    application_name VARCHAR(255) NOT NULL,
    node_id VARCHAR(255) NOT NULL,
    node_name VARCHAR(255) NOT NULL,
    annotation_text TEXT NOT NULL,
    created_by VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    -- Index for fast lookups by application
    CONSTRAINT idx_annotations_app_node UNIQUE (application_name, node_id, id)
);

-- Create index for efficient queries
CREATE INDEX IF NOT EXISTS idx_annotations_app ON graph_annotations(application_name);
CREATE INDEX IF NOT EXISTS idx_annotations_node ON graph_annotations(node_id);
CREATE INDEX IF NOT EXISTS idx_annotations_created_at ON graph_annotations(created_at DESC);
