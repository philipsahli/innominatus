-- Migration: Add delegated resources support to resource_instances table
-- Date: 2025-10-06
-- Description: Adds columns to support delegated (externally-managed) and external (read-only) resources

-- Add type column to distinguish native, delegated, and external resources
ALTER TABLE resource_instances
ADD COLUMN IF NOT EXISTS type VARCHAR(50) NOT NULL DEFAULT 'native';

-- Add provider column to track external system managing the resource
ALTER TABLE resource_instances
ADD COLUMN IF NOT EXISTS provider VARCHAR(100) NULL;

-- Add reference_url column to store links to external resources (PR, build, etc.)
ALTER TABLE resource_instances
ADD COLUMN IF NOT EXISTS reference_url TEXT NULL;

-- Add external_state column to track external provisioning status
ALTER TABLE resource_instances
ADD COLUMN IF NOT EXISTS external_state VARCHAR(50) NULL;

-- Add last_sync column to track when external state was last updated
ALTER TABLE resource_instances
ADD COLUMN IF NOT EXISTS last_sync TIMESTAMP WITH TIME ZONE NULL;

-- Create indexes for efficient filtering
CREATE INDEX IF NOT EXISTS idx_resource_instances_type ON resource_instances(type);
CREATE INDEX IF NOT EXISTS idx_resource_instances_provider ON resource_instances(provider);
CREATE INDEX IF NOT EXISTS idx_resource_instances_external_state ON resource_instances(external_state);

-- Add comments for documentation
COMMENT ON COLUMN resource_instances.type IS 'Resource type: native (managed by orchestrator), delegated (managed externally), external (read-only reference)';
COMMENT ON COLUMN resource_instances.provider IS 'External provider managing the resource (e.g., gitops, terraform-enterprise)';
COMMENT ON COLUMN resource_instances.reference_url IS 'URL to external resource (GitHub PR, Terraform Cloud run, etc.)';
COMMENT ON COLUMN resource_instances.external_state IS 'External resource state: WaitingExternal, BuildingExternal, Healthy, Error, Unknown';
COMMENT ON COLUMN resource_instances.last_sync IS 'Timestamp of last sync with external system';
