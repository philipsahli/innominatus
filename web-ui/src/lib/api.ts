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
  step_config?: Record<string, any>;
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

// Paginated response interface
export interface PaginatedWorkflowsResponse {
  data: WorkflowExecution[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
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
  step_number?: number;
  total_steps?: number;
  workflow_id?: number;
  duration_ms?: number;
  execution_order?: number;
  created_at?: string;
  updated_at?: string;
  metadata?: any;
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

export interface ResourceHint {
  type: string; // "url", "connection_string", "dashboard", "docs", "api_endpoint", "git_clone", "command"
  label: string; // Display name: "Repository URL", "Admin Dashboard", etc.
  value: string; // Actual value: URL, connection string, command, etc.
  icon?: string; // Optional icon: "external-link", "database", "lock", "terminal", "git-branch"
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
  hints?: ResourceHint[]; // Multiple contextual hints for the resource
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

export interface AdminConfig {
  admin: {
    defaultCostCenter: string;
    defaultRuntime: string;
    splunkIndex: string;
  };
  resourceDefinitions: Record<string, string>;
  policies: {
    enforceBackups: boolean;
    allowedEnvironments: string[];
  };
  gitea: {
    url: string;
    internalURL: string;
    username: string;
    password: string; // Will be "****" from backend
    orgName: string;
  };
  argocd: {
    url: string;
    username: string;
    password: string; // Will be "****" from backend
  };
  vault: {
    url: string;
    token: string; // Will be "****" from backend
    namespace: string;
  };
  keycloak: {
    url: string;
    adminUser: string;
    adminPassword: string; // Will be "****" from backend
    realm: string;
  };
  minio: {
    url: string;
    consoleURL: string;
    accessKey: string;
    secretKey: string; // Will be "****" from backend
  };
  prometheus: {
    url: string;
  };
  grafana: {
    url: string;
    username: string;
    password: string; // Will be "****" from backend
  };
  kubernetesDashboard: {
    url: string;
  };
  workflowPolicies: {
    workflowsRoot: string;
    requiredPlatformWorkflows: string[];
    allowedProductWorkflows: string[];
    maxWorkflowDuration: string;
    maxConcurrentWorkflows: number;
    maxStepsPerWorkflow: number;
    allowedStepTypes: string[];
    workflowOverrides: {
      platform: boolean;
      product: boolean;
    };
    security: {
      requireApproval: string[];
      allowedExecutors: string[];
      secretsAccess: Record<string, string>;
    };
  };
}

export interface AuthConfig {
  oidc_enabled: boolean;
  oidc_provider_name: string;
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
        credentials: 'include', // Include cookies for session-based auth
        ...options,
      });

      if (!response.ok) {
        if (response.status === 401) {
          // Check if this is a permission check vs authentication failure
          // Don't redirect on endpoints that are used for status checks
          const isStatusCheck =
            endpoint.includes('/impersonate') ||
            (endpoint.includes('/admin/') && options.method === 'GET');

          if (!isStatusCheck && typeof window !== 'undefined') {
            // Session expired - clear token and redirect to login
            localStorage.removeItem('auth-token');
            // Give user feedback about session expiration
            console.warn('Session expired - redirecting to login');
            // Redirect to login page
            window.location.href = '/login';
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
      const response = await this.request<Record<string, any>>('/applications');

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
    // Kept for backward compatibility, now calls /applications
    return this.request<Record<string, any>>('/applications');
  }

  async getApplication(name: string): Promise<ApiResponse<Application>> {
    return this.request<Application>(`/applications/${name}`);
  }

  async deployApplication(scoreSpec: string): Promise<ApiResponse<{ message: string }>> {
    return this.request('/applications', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/yaml',
      },
      body: scoreSpec,
    });
  }

  async deleteApplication(name: string): Promise<ApiResponse<{ message: string }>> {
    return this.request(`/applications/${name}`, {
      method: 'DELETE',
    });
  }

  async deprovisionApplication(name: string): Promise<ApiResponse<{ message: string }>> {
    return this.request(`/applications/${name}/deprovision`, {
      method: 'POST',
    });
  }

  // Environments
  async getEnvironments(): Promise<ApiResponse<Record<string, any>>> {
    return this.request<Record<string, any>>('/environments');
  }

  // Workflows
  async getWorkflows(
    appName?: string,
    search?: string,
    status?: string,
    page: number = 1,
    limit: number = 50
  ): Promise<ApiResponse<PaginatedWorkflowsResponse>> {
    const params = new URLSearchParams();
    if (appName) params.append('app', appName);
    if (search) params.append('search', search);
    if (status && status !== 'all') params.append('status', status);
    params.append('page', page.toString());
    params.append('limit', limit.toString());

    const query = params.toString() ? `?${params.toString()}` : '';
    const response = await this.request<{
      data: WorkflowExecutionApiResponse[];
      total: number;
      page: number;
      page_size: number;
      total_pages: number;
    }>(`/workflows${query}`);

    if (response.success && response.data) {
      // Transform backend response to frontend interface
      const transformedData: WorkflowExecution[] = response.data.data.map((workflow) => {
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
        data: {
          data: transformedData,
          total: response.data.total,
          page: response.data.page,
          page_size: response.data.page_size,
          total_pages: response.data.total_pages,
        },
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

  async retryWorkflow(
    id: string,
    workflow?: any
  ): Promise<ApiResponse<{ success: boolean; message: string }>> {
    const options: RequestInit = {
      method: 'POST',
    };

    // Only include body if workflow is provided (for manual retry with updated spec)
    // If workflow is undefined/null, send empty body for automatic retry
    if (workflow) {
      options.body = JSON.stringify(workflow);
    }

    return this.request<{ success: boolean; message: string }>(`/workflows/${id}/retry`, options);
  }

  // Resource Graph
  async getResourceGraph(appName: string): Promise<ApiResponse<GraphData>> {
    return this.request<GraphData>(`/graph/${appName}`);
  }

  async getWorkflowDetailsForGraph(
    appName: string,
    workflowId: string
  ): Promise<ApiResponse<WorkflowExecutionDetail>> {
    return this.request<WorkflowExecutionDetail>(`/graph/${appName}/workflow/${workflowId}`);
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
  async getAuthConfig(): Promise<ApiResponse<AuthConfig>> {
    return this.request<AuthConfig>('/auth/config');
  }

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

  async runDemoReset(): Promise<
    ApiResponse<{
      success: boolean;
      tables_truncated: number;
      tasks_stopped: number;
      message: string;
      timestamp: string;
    }>
  > {
    return this.request('/admin/demo/reset', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ confirm: true }),
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

  async createResource(
    applicationName: string,
    resourceName: string,
    resourceType: string,
    configuration?: Record<string, any>
  ): Promise<ApiResponse<ResourceInstance>> {
    return this.request<ResourceInstance>('/resources', {
      method: 'POST',
      body: JSON.stringify({
        application_name: applicationName,
        resource_name: resourceName,
        resource_type: resourceType,
        configuration: configuration || {},
      }),
    });
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

  // Users Management
  async listUsers(): Promise<ApiResponse<any>> {
    return this.request('/users');
  }

  // Admin Configuration
  async getAdminConfig(): Promise<ApiResponse<AdminConfig>> {
    return this.request<AdminConfig>('/admin/config');
  }

  // AI Assistant
  async getAIStatus(): Promise<ApiResponse<AIStatusResponse>> {
    return this.request<AIStatusResponse>('/ai/status');
  }

  async sendAIChat(
    message: string,
    conversationHistory?: ConversationMessage[],
    context?: string
  ): Promise<ApiResponse<AIChatResponse>> {
    return this.request('/ai/chat', {
      method: 'POST',
      body: JSON.stringify({ message, conversation_history: conversationHistory, context }),
    });
  }

  async generateSpec(
    description: string,
    metadata?: Record<string, string>
  ): Promise<ApiResponse<AIGenerateSpecResponse>> {
    return this.request('/ai/generate-spec', {
      method: 'POST',
      body: JSON.stringify({ description, metadata }),
    });
  }

  // Impersonation
  async startImpersonation(username: string): Promise<ApiResponse<any>> {
    return this.request('/impersonate', {
      method: 'POST',
      body: JSON.stringify({ username }),
    });
  }

  async stopImpersonation(): Promise<ApiResponse<any>> {
    return this.request('/impersonate', {
      method: 'DELETE',
    });
  }

  async getImpersonationStatus(): Promise<ApiResponse<ImpersonationStatus>> {
    return this.request('/impersonate');
  }

  async getProviders(): Promise<ApiResponse<ProviderSummary[]>> {
    return this.request('/providers');
  }

  async getProviderStats(): Promise<ApiResponse<ProviderStats>> {
    return this.request('/providers/stats');
  }

  // Alias for deployApplication - used by deploy wizard
  async submitSpec(scoreSpec: string): Promise<ApiResponse<{ message: string }>> {
    return this.deployApplication(scoreSpec);
  }
}

export interface ConversationMessage {
  role: 'user' | 'assistant';
  content: string;
  timestamp: string;
  spec?: string;
  citations?: string[];
}

export interface AIStatusResponse {
  enabled: boolean;
  llm_provider: string;
  documents_loaded: number;
  status: string;
  message?: string;
  missing_keys?: string[];
}

export interface AIChatResponse {
  message: string;
  generated_spec?: string;
  citations?: string[];
  tokens_used?: number;
  timestamp: string;
}

export interface AIGenerateSpecResponse {
  spec: string;
  explanation: string;
  citations?: string[];
  tokens_used?: number;
}

export interface ImpersonationStatus {
  is_impersonating: boolean;
  original_user?: {
    username: string;
    team: string;
    role: string;
  };
  impersonated_user?: {
    username: string;
    team: string;
    role: string;
  };
}

export interface WorkflowSummary {
  name: string;
  description: string;
  category: string;
  version?: string;
  tags?: string[];
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

export interface ProviderStats {
  providers: number;
  provisioners: number;
}

export const api = new ApiClient();
