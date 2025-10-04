// API client for IDP Orchestrator backend
const API_BASE_URL = process.env.NODE_ENV === 'production' ? '/api' : 'http://localhost:8081/api';

export interface ApiResponse<T = any> {
  success: boolean;
  data?: T;
  error?: string;
}

export interface Application {
  name: string;
  status: 'running' | 'failed' | 'pending';
  environment: string;
  lastUpdated: string;
  resources: number;
}

export interface WorkflowExecution {
  id: string;
  name: string;
  status: 'completed' | 'running' | 'failed' | 'pending';
  duration: string;
  timestamp: string;
  app_name?: string;
}

export interface WorkflowStepExecution {
  id: number;
  step_number: number;
  step_name: string;
  step_type: string;
  status: 'completed' | 'running' | 'failed' | 'pending';
  started_at: string;
  completed_at?: string;
  duration_ms?: number;
  error_message?: string;
  output_logs?: string;
}

export interface WorkflowExecutionDetail {
  id: number;
  application_name: string;
  workflow_name: string;
  status: 'completed' | 'running' | 'failed' | 'pending';
  started_at: string;
  completed_at?: string;
  total_steps: number;
  error_message?: string;
  steps: WorkflowStepExecution[];
}

// Backend API response interface
interface WorkflowExecutionApiResponse {
  id: number;
  application_name: string;
  workflow_name: string;
  status: 'completed' | 'running' | 'failed' | 'pending';
  started_at: string;
  completed_at?: string;
  total_steps: number;
  completed_steps: number;
  failed_steps: number;
  duration_ms?: number;
}

export interface GraphData {
  app_name: string;
  nodes: GraphNode[];
  edges: GraphEdge[];
  timestamp: string;
}

export interface GraphNode {
  id: string;
  name: string;
  type: string;
  status: string;
  description: string;
}

export interface GraphEdge {
  id: string;
  source_id: string;
  target_id: string;
  relationship: string;
}

export interface Stats {
  applications: number;
  workflows: number;
  resources: number;
  users: number;
}

export interface DemoComponent {
  name: string;
  url: string;
  status: boolean;
  credentials: string;
}

export interface DemoStatusResponse {
  components: DemoComponent[];
  timestamp: string;
}

export interface ResourceInstance {
  id: number;
  application_name: string;
  resource_name: string;
  resource_type: string;
  state:
    | 'requested'
    | 'provisioning'
    | 'active'
    | 'scaling'
    | 'updating'
    | 'degraded'
    | 'terminating'
    | 'terminated'
    | 'failed';
  health_status: string;
  configuration: Record<string, any>;
  provider_id?: string;
  provider_metadata?: Record<string, any>;
  workflow_execution_id?: number;
  created_at: string;
  updated_at: string;
  last_health_check?: string;
  error_message?: string;
}

export interface UserProfile {
  username: string;
  team: string;
  role: string;
}

export interface APIKeyInfo {
  name: string;
  masked_key: string;
  created_at: string;
  last_used_at?: string;
  expires_at: string;
}

export interface APIKeyFull {
  key: string;
  name: string;
  created_at: string;
  expires_at: string;
}

class ApiClient {
  private getAuthToken(): string | null {
    if (typeof window === 'undefined') return null;
    return localStorage.getItem('auth-token');
  }

  private async request<T>(endpoint: string, options: RequestInit = {}): Promise<ApiResponse<T>> {
    try {
      const token = this.getAuthToken();
      const headers: Record<string, string> = {
        'Content-Type': 'application/json',
        ...(options.headers as Record<string, string>),
      };

      if (token) {
        headers['Authorization'] = `Bearer ${token}`;
      }

      const response = await fetch(`${API_BASE_URL}${endpoint}`, {
        headers,
        ...options,
      });

      if (!response.ok) {
        if (response.status === 401) {
          // Clear invalid token
          if (typeof window !== 'undefined') {
            localStorage.removeItem('auth-token');
          }
        }
        return {
          success: false,
          error: `HTTP ${response.status}: ${response.statusText}`,
        };
      }

      const data = await response.json();
      return {
        success: true,
        data,
      };
    } catch (error) {
      return {
        success: false,
        error: error instanceof Error ? error.message : 'Unknown error',
      };
    }
  }

  // Applications
  async getApplications(): Promise<ApiResponse<Application[]>> {
    try {
      const response = await this.request<Record<string, any>>('/specs');

      if (response.success && response.data) {
        // Transform specs data to Application format
        const applications: Application[] = Object.keys(response.data).map((name) => {
          const spec = response.data![name] || {};
          return {
            name,
            status: 'running' as const, // Default status since specs don't have status
            environment: spec.environment?.Type || 'unknown',
            lastUpdated: new Date().toISOString(), // Use current time as fallback
            resources: Object.keys(spec.resources || {}).length,
          };
        });

        return {
          success: true,
          data: applications,
        };
      }

      return {
        success: false,
        error: 'No data received from server',
      };
    } catch (error) {
      return {
        success: false,
        error: error instanceof Error ? error.message : 'Unknown error',
      };
    }
  }

  async getSpecs(): Promise<ApiResponse<Record<string, any>>> {
    return this.request<Record<string, any>>('/specs');
  }

  async getApplication(name: string): Promise<ApiResponse<Application>> {
    return this.request<Application>(`/apps/${name}`);
  }

  async deployApplication(scoreSpec: string): Promise<ApiResponse<{ message: string }>> {
    return this.request('/apps', {
      method: 'POST',
      body: JSON.stringify({ spec: scoreSpec }),
    });
  }

  async deleteApplication(name: string): Promise<ApiResponse<{ message: string }>> {
    return this.request(`/apps/${name}`, {
      method: 'DELETE',
    });
  }

  // Workflows
  async getWorkflows(appName?: string): Promise<ApiResponse<WorkflowExecution[]>> {
    const query = appName ? `?app=${appName}` : '';
    const response = await this.request<WorkflowExecutionApiResponse[]>(`/workflows${query}`);

    if (response.success && response.data) {
      // Transform backend response to frontend interface
      const transformedData: WorkflowExecution[] = response.data.map((workflow) => {
        // Calculate relative timestamp
        const startTime = new Date(workflow.started_at);
        const now = new Date();
        const diffMs = now.getTime() - startTime.getTime();
        const diffMins = Math.floor(diffMs / 60000);
        const diffHours = Math.floor(diffMs / 3600000);
        const diffDays = Math.floor(diffMs / 86400000);

        let timestamp: string;
        if (diffMins < 1) {
          timestamp = 'just now';
        } else if (diffMins < 60) {
          timestamp = `${diffMins} minute${diffMins === 1 ? '' : 's'} ago`;
        } else if (diffHours < 24) {
          timestamp = `${diffHours} hour${diffHours === 1 ? '' : 's'} ago`;
        } else {
          timestamp = `${diffDays} day${diffDays === 1 ? '' : 's'} ago`;
        }

        // Format duration
        let duration: string;
        if (workflow.duration_ms) {
          const seconds = Math.floor(workflow.duration_ms / 1000);
          const minutes = Math.floor(seconds / 60);
          const remainingSeconds = seconds % 60;
          if (minutes > 0) {
            duration = `${minutes}m ${remainingSeconds}s`;
          } else {
            duration = `${seconds}s`;
          }
        } else {
          duration = '-';
        }

        return {
          id: workflow.id.toString(),
          name: workflow.workflow_name,
          status: workflow.status,
          duration,
          timestamp,
          app_name: workflow.application_name,
        };
      });

      return {
        success: true,
        data: transformedData,
      };
    }

    return {
      success: false,
      error: response.error || 'Failed to get workflows',
    };
  }

  async getWorkflow(id: string): Promise<ApiResponse<WorkflowExecutionDetail>> {
    return this.request<WorkflowExecutionDetail>(`/workflows/${id}`);
  }

  // Resource Graph
  async getResourceGraph(appName: string): Promise<ApiResponse<GraphData>> {
    return this.request<GraphData>(`/graph/${appName}`);
  }

  // Dashboard Stats
  async getStats(): Promise<ApiResponse<Stats>> {
    return this.request<Stats>('/stats');
  }

  // Health Check
  async health(): Promise<ApiResponse<{ status: string }>> {
    return this.request<{ status: string }>('/health');
  }

  // Authentication
  async login(
    username: string,
    password: string
  ): Promise<ApiResponse<{ token: string; user: any }>> {
    return this.request('/login', {
      method: 'POST',
      body: JSON.stringify({ username, password }),
    });
  }

  // Admin
  async getConfig(): Promise<ApiResponse<any>> {
    return this.request('/admin/config');
  }

  // Demo Environment
  async getDemoStatus(): Promise<ApiResponse<DemoStatusResponse>> {
    return this.request<DemoStatusResponse>('/demo/status');
  }

  async runDemoTime(): Promise<ApiResponse<{ message: string }>> {
    return this.request('/demo/time', {
      method: 'POST',
    });
  }

  async runDemoNuke(): Promise<ApiResponse<{ message: string }>> {
    return this.request('/demo/nuke', {
      method: 'POST',
    });
  }

  // Resources
  async getResources(appName?: string): Promise<ApiResponse<Record<string, ResourceInstance[]>>> {
    const query = appName ? `?app=${appName}` : '';
    return this.request<Record<string, ResourceInstance[]>>(`/resources${query}`);
  }

  async getResource(id: number): Promise<ApiResponse<ResourceInstance>> {
    return this.request<ResourceInstance>(`/resources/${id}`);
  }

  // Workflow Analysis
  async analyzeWorkflow(yamlContent: string): Promise<ApiResponse<any>> {
    return this.request('/workflow-analysis', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/yaml',
      },
      body: yamlContent,
    });
  }

  // Profile
  async getProfile(): Promise<ApiResponse<UserProfile>> {
    return this.request<UserProfile>('/profile');
  }

  async getAPIKeys(): Promise<ApiResponse<APIKeyInfo[]>> {
    return this.request<APIKeyInfo[]>('/profile/api-keys');
  }

  async generateAPIKey(name: string, expiryDays: number): Promise<ApiResponse<APIKeyFull>> {
    return this.request('/profile/api-keys', {
      method: 'POST',
      body: JSON.stringify({ name, expiry_days: expiryDays }),
    });
  }

  async revokeAPIKey(name: string): Promise<ApiResponse<void>> {
    return this.request(`/profile/api-keys/${name}`, {
      method: 'DELETE',
    });
  }
}

export const api = new ApiClient();
