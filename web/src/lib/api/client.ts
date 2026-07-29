import type {
  LogsResponse,
  ContextsResponse,
  FieldsResponse,
  SearchParams,
  ApiError,
  Context,
} from './types';

// Port injected by VS Code webview
declare global {
  interface Window {
    LOGVIEWER_PORT?: number;
  }
}

function getBaseUrl(): string {
  const port = window.LOGVIEWER_PORT || 8080;
  return `http://localhost:${port}`;
}

class ApiClient {
  private baseUrl: string;

  constructor() {
    this.baseUrl = getBaseUrl();
  }

  private async request<T>(
    endpoint: string,
    options: RequestInit = {}
  ): Promise<T> {
    const url = `${this.baseUrl}${endpoint}`;

    const response = await fetch(url, {
      ...options,
      headers: {
        'Content-Type': 'application/json',
        ...options.headers,
      },
    });

    if (!response.ok) {
      const error: ApiError = await response.json();
      throw new Error(error.error || `HTTP ${response.status}`);
    }

    return response.json();
  }

  // GET /contexts - List all available contexts
  async getContexts(): Promise<ContextsResponse> {
    return this.request<ContextsResponse>('/contexts');
  }

  // GET /contexts/:id - Get specific context
  async getContext(contextId: string): Promise<Context> {
    return this.request<Context>(`/contexts/${encodeURIComponent(contextId)}`);
  }

  // POST /query/logs - Query log entries
  async queryLogs(params: SearchParams): Promise<LogsResponse> {
    return this.request<LogsResponse>('/query/logs', {
      method: 'POST',
      body: JSON.stringify({
        contextId: params.contextId,
        search: {
          fields: params.fields,
          range: params.range,
          size: params.size || 100,
        },
        ...(params.nativeQuery && { nativeQuery: params.nativeQuery }),
        ...(params.pageToken && { pageToken: params.pageToken }),
      }),
    });
  }

  // POST /query/fields - Discover available fields
  async queryFields(contextId: string): Promise<FieldsResponse> {
    return this.request<FieldsResponse>('/query/fields', {
      method: 'POST',
      body: JSON.stringify({
        contextId,
        search: {
          range: { last: '1h' },
          size: 1000,
        },
      }),
    });
  }

  // GET /health - Health check
  async healthCheck(): Promise<{ status: string }> {
    return this.request<{ status: string }>('/health');
  }
}

// Singleton instance
export const api = new ApiClient();
