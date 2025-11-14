-- Migration: Add workflow-related columns to resource_instances table
-- Description: Adds columns to support workflow execution tracking and CRUD operations
-- Date: 2025-11-14

-- Add workflow_execution_id to track which workflow is provisioning this resource
ALTER TABLE resource_instances
ADD COLUMN IF NOT EXISTS workflow_execution_id BIGINT NULL;

-- Add desired_operation to support CRUD operation routing
ALTER TABLE resource_instances
ADD COLUMN IF NOT EXISTS desired_operation VARCHAR(50) NULL;

-- Add workflow_override to allow explicit workflow selection
ALTER TABLE resource_instances
ADD COLUMN IF NOT EXISTS workflow_override VARCHAR(255) NULL;

-- Add workflow_tags for workflow disambiguation
ALTER TABLE resource_instances
ADD COLUMN IF NOT EXISTS workflow_tags JSONB DEFAULT '[]';

-- Add index for workflow execution lookups
CREATE INDEX IF NOT EXISTS idx_resource_instances_workflow_execution_id ON resource_instances(workflow_execution_id);

-- Add index for desired operation filtering
CREATE INDEX IF NOT EXISTS idx_resource_instances_desired_operation ON resource_instances(desired_operation);

-- Add comments for documentation
COMMENT ON COLUMN resource_instances.workflow_execution_id IS 'ID of the workflow execution provisioning this resource';
COMMENT ON COLUMN resource_instances.desired_operation IS 'CRUD operation: create, read, update, delete';
COMMENT ON COLUMN resource_instances.workflow_override IS 'Explicit workflow name to use instead of auto-resolution';
COMMENT ON COLUMN resource_instances.workflow_tags IS 'Tags for workflow disambiguation when multiple workflows match';
