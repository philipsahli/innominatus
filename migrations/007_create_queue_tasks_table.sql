-- Migration: Create queue_tasks table for async workflow execution
-- Created: 2025-10-14
-- Description: Stores async workflow tasks with status tracking

CREATE TABLE IF NOT EXISTS queue_tasks (
    task_id VARCHAR(255) PRIMARY KEY,
    app_name VARCHAR(255) NOT NULL,
    workflow_name VARCHAR(255) NOT NULL,
    workflow_spec TEXT NOT NULL,  -- JSON serialized workflow
    metadata TEXT DEFAULT '{}',   -- JSON serialized metadata
    status VARCHAR(50) NOT NULL DEFAULT 'pending',  -- pending, running, completed, failed
    error_message TEXT,
    enqueued_at TIMESTAMP NOT NULL,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_queue_tasks_app_name ON queue_tasks(app_name);
CREATE INDEX IF NOT EXISTS idx_queue_tasks_status ON queue_tasks(status);
CREATE INDEX IF NOT EXISTS idx_queue_tasks_enqueued_at ON queue_tasks(enqueued_at DESC);
CREATE INDEX IF NOT EXISTS idx_queue_tasks_workflow_name ON queue_tasks(workflow_name);

-- Composite index for common queries
CREATE INDEX IF NOT EXISTS idx_queue_tasks_app_status ON queue_tasks(app_name, status);

-- Comments
COMMENT ON TABLE queue_tasks IS 'Async workflow execution queue with status tracking';
COMMENT ON COLUMN queue_tasks.task_id IS 'Unique task identifier (UUID)';
COMMENT ON COLUMN queue_tasks.workflow_spec IS 'JSON serialized workflow specification';
COMMENT ON COLUMN queue_tasks.metadata IS 'Additional task metadata (JSON)';
COMMENT ON COLUMN queue_tasks.status IS 'Task status: pending, running, completed, failed';
