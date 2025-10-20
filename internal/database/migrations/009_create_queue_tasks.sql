-- Create queue_tasks table for async workflow execution
CREATE TABLE IF NOT EXISTS queue_tasks (
    id SERIAL PRIMARY KEY,
    task_id VARCHAR(255) UNIQUE NOT NULL,
    app_name VARCHAR(255) NOT NULL,
    workflow_name VARCHAR(255) NOT NULL,
    workflow_spec JSONB NOT NULL,
    metadata JSONB,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    error_message TEXT,
    enqueued_at TIMESTAMP NOT NULL DEFAULT NOW(),
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    updated_at TIMESTAMP DEFAULT NOW(),
    created_at TIMESTAMP DEFAULT NOW()
);

-- Create indexes for efficient queries
CREATE INDEX IF NOT EXISTS idx_queue_tasks_status ON queue_tasks(status);
CREATE INDEX IF NOT EXISTS idx_queue_tasks_app_name ON queue_tasks(app_name);
CREATE INDEX IF NOT EXISTS idx_queue_tasks_enqueued_at ON queue_tasks(enqueued_at DESC);
CREATE INDEX IF NOT EXISTS idx_queue_tasks_task_id ON queue_tasks(task_id);

-- Add comments for documentation
COMMENT ON TABLE queue_tasks IS 'Async workflow execution queue';
COMMENT ON COLUMN queue_tasks.task_id IS 'Unique task identifier';
COMMENT ON COLUMN queue_tasks.workflow_spec IS 'Full workflow specification as JSON';
COMMENT ON COLUMN queue_tasks.metadata IS 'Additional task metadata (e.g., user, source)';
COMMENT ON COLUMN queue_tasks.status IS 'Task status: pending, running, completed, failed';
