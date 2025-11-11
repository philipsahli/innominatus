-- Migration: Add workflow retry support
-- Description: Adds columns to track workflow retry relationships and state
-- Date: 2025-10-17

-- Add retry-related columns to workflow_executions table
ALTER TABLE workflow_executions ADD COLUMN IF NOT EXISTS parent_execution_id BIGINT REFERENCES workflow_executions(id) ON DELETE SET NULL;
ALTER TABLE workflow_executions ADD COLUMN IF NOT EXISTS retry_count INTEGER DEFAULT 0 NOT NULL;
ALTER TABLE workflow_executions ADD COLUMN IF NOT EXISTS is_retry BOOLEAN DEFAULT FALSE NOT NULL;
ALTER TABLE workflow_executions ADD COLUMN IF NOT EXISTS resume_from_step INTEGER;

-- Add index for parent execution lookups
CREATE INDEX IF NOT EXISTS idx_workflow_executions_parent_id ON workflow_executions(parent_execution_id);

-- Add index for finding retry chains
CREATE INDEX IF NOT EXISTS idx_workflow_executions_is_retry ON workflow_executions(is_retry);

-- Add index for application + workflow name lookups (for finding latest execution)
CREATE INDEX IF NOT EXISTS idx_workflow_executions_app_workflow ON workflow_executions(application_name, workflow_name, created_at DESC);

-- Add comment explaining retry flow
COMMENT ON COLUMN workflow_executions.parent_execution_id IS 'References the original/parent execution when this is a retry';
COMMENT ON COLUMN workflow_executions.retry_count IS 'Number of retry attempts (0 for original execution)';
COMMENT ON COLUMN workflow_executions.is_retry IS 'True if this execution is a retry of a previous failed execution';
COMMENT ON COLUMN workflow_executions.resume_from_step IS 'Step number to resume from (NULL means start from beginning)';
