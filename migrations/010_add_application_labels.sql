-- Migration: Add labels to applications
-- Created: 2025-10-20

-- Add labels column to applications table
ALTER TABLE applications ADD COLUMN IF NOT EXISTS labels TEXT[] DEFAULT '{}';

-- Create GIN index for efficient label filtering
CREATE INDEX IF NOT EXISTS idx_applications_labels ON applications USING GIN (labels);

-- Add comment for documentation
COMMENT ON COLUMN applications.labels IS 'Array of labels/tags for organizing and filtering applications';
