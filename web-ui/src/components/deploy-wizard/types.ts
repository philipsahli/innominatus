export interface WizardData {
  // Step 1: Basic Info
  appName: string;
  environment: 'kubernetes' | 'production' | 'staging';
  ttl: string; // '1h' | '24h' | '168h'

  // Step 2: Container
  container: {
    image: string;
    port: number;
    envVars: Record<string, string>;
    cpuRequest?: string;
    memoryRequest?: string;
  };

  // Step 3: Resources
  resources: ResourceRequest[];
}

export interface ResourceRequest {
  name: string; // Resource identifier (e.g., 'db', 'storage')
  type: string; // Resource type (e.g., 'postgres', 's3')
  properties: Record<string, any>;
}

export interface ProviderSummary {
  name: string;
  version: string;
  category: string;
  description: string;
  provisioners: number;
  golden_paths: number;
  workflows: WorkflowSummary[];
}

export interface WorkflowSummary {
  name: string;
  description: string;
  category: string;
  version?: string;
  tags?: string[];
  parameters?: WorkflowParameter[];
}

export interface WorkflowParameter {
  name: string;
  type: 'string' | 'number' | 'boolean';
  required: boolean;
  default?: any;
  description?: string;
  enum?: string[];
}

export interface StepProps {
  data: WizardData;
  onChange: (data: Partial<WizardData>) => void;
  onNext: () => void;
  onPrev: () => void;
}
