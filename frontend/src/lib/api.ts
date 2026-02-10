import type {
  OutgoingEndpoint,
  OutgoingEndpointRequest,
  IncomingRoute,
  IncomingRouteRequest,
  AuthConfig,
  AuthConfigRequest,
  TokenStatus,
  MetricsResponse,
  MetricsSnapshot,
  IncomingMetricsSnapshot,
  ControlStatus,
  ControlRequest,
  EndpointControlRequest,
  OutgoingSettings,
  ApiResponse,
} from '@/types/api';

const BASE_URL = '';

class ApiError extends Error {
  status: number;
  
  constructor(status: number, message: string) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
  }
}

async function request<T>(
  path: string,
  options: RequestInit = {}
): Promise<T> {
  const url = `${BASE_URL}${path}`;
  const response = await fetch(url, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...options.headers,
    },
  });

  if (!response.ok) {
    const text = await response.text();
    let message = `HTTP ${response.status}`;
    try {
      const json = JSON.parse(text);
      message = json.error || json.message || message;
    } catch {
      message = text || message;
    }
    throw new ApiError(response.status, message);
  }

  // Handle empty responses
  const text = await response.text();
  if (!text) return {} as T;
  
  return JSON.parse(text);
}

async function requestText(
  path: string,
  options: RequestInit = {}
): Promise<string> {
  const url = `${BASE_URL}${path}`;
  const response = await fetch(url, {
    ...options,
    headers: {
      ...options.headers,
    },
  });

  if (!response.ok) {
    const text = await response.text();
    let message = `HTTP ${response.status}`;
    try {
      const json = JSON.parse(text);
      message = json.error || json.message || message;
    } catch {
      message = text || message;
    }
    throw new ApiError(response.status, message);
  }

  return response.text();
}

// ============================================================================
// Metrics API
// ============================================================================

export const metricsApi = {
  getOverview: () => request<MetricsResponse>('/api/metrics'),
  getOutgoing: () => request<MetricsSnapshot>('/api/metrics/outgoing'),
  getIncoming: () => request<IncomingMetricsSnapshot>('/api/metrics/incoming'),
  resetAll: () => request<void>('/api/metrics/reset', { method: 'POST' }),
  resetOutgoing: () => request<void>('/api/metrics/outgoing/reset', { method: 'POST' }),
  resetIncoming: () => request<void>('/api/metrics/incoming/reset', { method: 'POST' }),
};

// ============================================================================
// Scheduler Control API
// ============================================================================

export const controlApi = {
  getStatus: () => request<ControlStatus>('/api/outgoing/control'),
  control: (action: ControlRequest['action']) =>
    request<void>('/api/outgoing/control', {
      method: 'POST',
      body: JSON.stringify({ action }),
    }),
  setEndpointEnabled: (name: string, enabled: boolean) =>
    request<void>('/api/outgoing/control/endpoint', {
      method: 'POST',
      body: JSON.stringify({ name, enabled } as EndpointControlRequest),
    }),
  setAllEndpointsEnabled: (enabled: boolean) =>
    request<void>('/api/outgoing/control/endpoints/all', {
      method: 'POST',
      body: JSON.stringify({ enabled }),
    }),
  bulkSetEndpointsEnabled: (names: string[], enabled: boolean) =>
    request<void>('/api/outgoing/control/endpoints/bulk', {
      method: 'POST',
      body: JSON.stringify({ names, enabled }),
    }),
};

// ============================================================================
// Outgoing Settings API
// ============================================================================

export const settingsApi = {
  getAll: () => request<OutgoingSettings>('/api/outgoing/settings'),
  getMultiplier: () => request<{ global_multiplier: number }>('/api/outgoing/settings/multiplier')
    .then(res => res.global_multiplier),
  setMultiplier: (value: number) =>
    request<void>('/api/outgoing/settings/multiplier', {
      method: 'POST',
      body: JSON.stringify({ multiplier: value }),
    }),
  getConcurrency: () => request<{ concurrent_requests: number }>('/api/outgoing/settings/concurrency')
    .then(res => res.concurrent_requests),
  setConcurrency: (value: number) =>
    request<void>('/api/outgoing/settings/concurrency', {
      method: 'POST',
      body: JSON.stringify({ concurrent: value }),
    }),
  getLogRequests: () => request<{ log_all_requests: boolean }>('/api/outgoing/settings/log-requests')
    .then(res => res.log_all_requests),
  setLogRequests: (value: boolean) =>
    request<void>('/api/outgoing/settings/log-requests', {
      method: 'POST',
      body: JSON.stringify({ log_requests: value }),
    }),
};

// ============================================================================
// Outgoing Endpoints API
// ============================================================================

export const endpointsApi = {
  list: () => request<{ count: number; endpoints: OutgoingEndpoint[] }>('/api/outgoing/endpoints')
    .then(res => res.endpoints),
  get: (name: string) => request<OutgoingEndpoint>(`/api/outgoing/endpoints/${encodeURIComponent(name)}`),
  create: (endpoint: OutgoingEndpointRequest) =>
    request<OutgoingEndpoint>('/api/outgoing/endpoints', {
      method: 'POST',
      body: JSON.stringify(endpoint),
    }),
  update: (name: string, endpoint: OutgoingEndpointRequest) =>
    request<OutgoingEndpoint>(`/api/outgoing/endpoints/${encodeURIComponent(name)}`, {
      method: 'PUT',
      body: JSON.stringify(endpoint),
    }),
  delete: (name: string) =>
    request<void>(`/api/outgoing/endpoints/${encodeURIComponent(name)}`, {
      method: 'DELETE',
    }),
  bulkCreate: (endpoints: OutgoingEndpointRequest[]) =>
    request<OutgoingEndpoint[]>('/api/outgoing/endpoints/bulk', {
      method: 'POST',
      body: JSON.stringify(endpoints),
    }),
  bulkDelete: (names: string[]) =>
    request<void>('/api/outgoing/endpoints/bulk', {
      method: 'DELETE',
      body: JSON.stringify({ names }),
    }),
};

// ============================================================================
// Incoming Routes API
// ============================================================================

export const routesApi = {
  list: () => request<{ count: number; enabled: boolean; routes: IncomingRoute[] }>('/api/incoming/routes')
    .then(res => res.routes),
  get: (name: string) => request<IncomingRoute>(`/api/incoming/routes/${encodeURIComponent(name)}`),
  create: (route: IncomingRouteRequest) =>
    request<IncomingRoute>('/api/incoming/routes', {
      method: 'POST',
      body: JSON.stringify(route),
    }),
  update: (name: string, route: IncomingRouteRequest) =>
    request<IncomingRoute>(`/api/incoming/routes/${encodeURIComponent(name)}`, {
      method: 'PUT',
      body: JSON.stringify(route),
    }),
  delete: (name: string) =>
    request<void>(`/api/incoming/routes/${encodeURIComponent(name)}`, {
      method: 'DELETE',
    }),
  reload: () => request<void>('/api/incoming/routes/reload', { method: 'POST' }),
  getControl: () => request<{ enabled: boolean; enabled_routes: number; total_routes: number }>('/api/incoming/control')
    .then(res => ({
      enabled: res.enabled,
      active_routes: res.enabled_routes,
      total_routes: res.total_routes,
    })),
  setEnabled: (enabled: boolean) =>
    request<void>('/api/incoming/control', {
      method: 'POST',
      body: JSON.stringify({ enabled }),
    }),
  setRouteEnabled: (name: string, enabled: boolean) =>
    request<void>('/api/incoming/control/route', {
      method: 'POST',
      body: JSON.stringify({ name, enabled }),
    }),
};

// ============================================================================
// Auth Configs API
// ============================================================================

export const authApi = {
  list: () => request<{ auth_configs: Record<string, AuthConfig> }>('/api/outgoing/auth-configs')
    .then(res => Object.values(res.auth_configs)),
  get: (name: string) => request<AuthConfig>(`/api/outgoing/auth-configs/${encodeURIComponent(name)}`),
  create: (config: AuthConfigRequest) =>
    request<AuthConfig>('/api/outgoing/auth-configs', {
      method: 'POST',
      body: JSON.stringify(config),
    }),
  update: (name: string, config: AuthConfigRequest) =>
    request<AuthConfig>(`/api/outgoing/auth-configs/${encodeURIComponent(name)}`, {
      method: 'PUT',
      body: JSON.stringify(config),
    }),
  delete: (name: string) =>
    request<void>(`/api/outgoing/auth-configs/${encodeURIComponent(name)}`, {
      method: 'DELETE',
    }),
  getTokenStatus: (name: string) =>
    request<TokenStatus>(`/api/outgoing/auth-configs/${encodeURIComponent(name)}/status`),
  refreshToken: (name: string) =>
    request<void>(`/api/outgoing/auth-configs/${encodeURIComponent(name)}/refresh`, {
      method: 'POST',
    }),
  setToken: (name: string, token: string, expiresIn?: number) =>
    request<void>(`/api/outgoing/auth-configs/${encodeURIComponent(name)}/token`, {
      method: 'POST',
      body: JSON.stringify({ token, expires_in: expiresIn }),
    }),
};

// ============================================================================
// Config Import/Export API
// ============================================================================

export const configApi = {
  exportYaml: () => requestText('/api/config/export'),
  importYaml: (yamlText: string) =>
    request<ApiResponse<unknown>>('/api/config/import', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/x-yaml',
      },
      body: yamlText,
    }),
};

export { ApiError };
