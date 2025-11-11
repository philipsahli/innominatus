-- Migration 011: Add workflow-related columns to resource_instances
-- These columns enable orchestration engine to track workflow execution and CRUD operations

-- Add workflow_execution_id to track which workflow is provisioning this resource
ALTER TABLE resource_instances
ADD COLUMN workflow_execution_id INTEGER REFERENCES workflow_executions(id) ON DELETE SET NULL;

-- Add desired_operation to support CRUD operations (create, read, update, delete)
ALTER TABLE resource_instances
ADD COLUMN desired_operation VARCHAR(10) CHECK (desired_operation IN ('create', 'read', 'update', 'delete'));

-- Add workflow_override to allow explicit workflow selection
ALTER TABLE resource_instances
ADD COLUMN workflow_override VARCHAR(255);

-- Add workflow_tags for workflow disambiguation when multiple workflows handle same operation
ALTER TABLE resource_instances
ADD COLUMN workflow_tags TEXT[];

-- Create indexes for orchestration engine queries
CREATE INDEX idx_resource_instances_workflow_execution ON resource_instances(workflow_execution_id);

-- Create composite index for orchestration engine polling query
-- This dramatically improves performance: WHERE state = 'requested' AND workflow_execution_id IS NULL
CREATE INDEX idx_resource_instances_orchestration
ON resource_instances(state, workflow_execution_id, created_at)
WHERE state IN ('requested', 'pending');  -- Partial index for efficiency

-- Create index for desired_operation queries
CREATE INDEX idx_resource_instances_desired_operation ON resource_instances(desired_operation)
WHERE desired_operation IS NOT NULL;

-- Create GIN index for workflow_tags array queries
CREATE INDEX idx_resource_instances_workflow_tags ON resource_instances USING GIN(workflow_tags)
WHERE workflow_tags IS NOT NULL;

-- Add comment explaining the columns
COMMENT ON COLUMN resource_instances.workflow_execution_id IS 'ID of the workflow execution that is provisioning/managing this resource';
COMMENT ON COLUMN resource_instances.desired_operation IS 'CRUD operation to perform: create, read, update, or delete';
COMMENT ON COLUMN resource_instances.workflow_override IS 'Explicit workflow name to use instead of auto-resolution';
COMMENT ON COLUMN resource_instances.workflow_tags IS 'Tags for workflow disambiguation when multiple workflows handle the same operation';
