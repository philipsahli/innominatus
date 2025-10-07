import 'server-only';
import fs from 'fs';
import path from 'path';
import * as yaml from 'yaml';

// Re-export types for convenience
export type {
  WorkflowStep,
  WorkflowSpec,
  WorkflowMetadata,
  WorkflowDefinition,
} from './workflow-types';
export { getStepStyle, getStepDescription } from './workflow-types';
import type { WorkflowDefinition } from './workflow-types';

/**
 * Parse a workflow YAML file and extract steps (SERVER-SIDE ONLY)
 */
export async function parseWorkflowFile(workflowPath: string): Promise<WorkflowDefinition | null> {
  try {
    const fullPath = path.join(process.cwd(), '..', workflowPath.replace(/^\.\//, ''));

    if (!fs.existsSync(fullPath)) {
      console.warn(`Workflow file not found: ${fullPath}`);
      return null;
    }

    const fileContents = fs.readFileSync(fullPath, 'utf8');
    const workflow = yaml.parse(fileContents) as WorkflowDefinition;

    return workflow;
  } catch (error) {
    console.error(`Error parsing workflow file ${workflowPath}:`, error);
    return null;
  }
}
