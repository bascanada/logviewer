// Log entry from the backend
export interface LogEntry {
  timestamp: string; // RFC3339 format
  message: string;
  level: string;
  fields: Record<string, unknown>;
  context_id: string;
}

// Query response metadata
export interface QueryMeta {
  queryTime: string;
  resultCount: number;
  contextUsed: string;
  clientType: string;
  nextPageToken?: string;
}

// Log query response
export interface LogsResponse {
  logs: LogEntry[];
  meta: QueryMeta;
}

// Context definition
export interface Context {
  id: string;
  client: string;
  description?: string;
  searchInherit?: string[];
}

export interface ContextsResponse {
  contexts: Context[];
}

// Fields discovery response
export interface FieldsResponse {
  fields: Record<string, string[]>;
  meta: QueryMeta;
}

// Search parameters
export interface SearchParams {
  contextId: string;
  fields?: Record<string, string>;
  nativeQuery?: string;
  range?: {
    last?: string; // e.g., "1h", "24h"
    gte?: string; // RFC3339
    lte?: string; // RFC3339
  };
  size?: number;
  pageToken?: string;
}

// API error response
export interface ApiError {
  error: string;
  code: string;
  details?: Record<string, unknown>;
}
