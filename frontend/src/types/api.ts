// API Types for MoxApp Load Test

// ============================================================================
// Scheduler / Control Types
// ============================================================================

export type SchedulerState = 'running' | 'paused' | 'stopped';

export interface ControlStatus {
  global_enabled: boolean;
  paused: boolean;
  scheduler_running: boolean;
  requests_scheduled: number;
  requests_in_flight: number;
  requests_skipped: number;
  total_endpoints: number;
  enabled_endpoints: number;
  disabled_endpoints: number;
}

export interface ControlRequest {
  action: 'pause' | 'resume' | 'emergency_stop';
}

export interface EndpointControlRequest {
  name: string;
  enabled: boolean;
}

// ============================================================================
// Outgoing Endpoint Types
// ============================================================================

export interface OutgoingEndpoint {
  name: string;
  method: 'GET' | 'POST' | 'PUT' | 'DELETE' | 'PATCH' | 'HEAD' | 'OPTIONS';
  url_template: string;
  config_path?: string;
  frequency: number;
  auth?: string | AuthInlineOverride;
  headers?: Record<string, string>;
  body?: unknown;
  timeout?: number;
  enabled: boolean;
}

export interface AuthInlineOverride {
  ref: string;
  header_name?: string;
  query_param?: string;
}

export interface OutgoingEndpointRequest {
  name: string;
  method: string;
  url_template: string;
  config_path?: string;
  frequency: number;
  auth?: string | AuthInlineOverride;
  headers?: Record<string, string>;
  body?: unknown;
  timeout?: number;
  enabled: boolean;
}

// ============================================================================
// Incoming Route Types
// ============================================================================

export interface IncomingRoute {
  name: string;
  path: string;
  method: string;
  responses: IncomingResponseConfig[];
  enabled: boolean;
}

export interface IncomingResponseConfig {
  status: number;
  share: number;
  min_response_ms: number;
  max_response_ms: number;
}

export interface IncomingRouteRequest {
  name: string;
  path: string;
  method: string;
  responses: IncomingResponseConfig[];
  enabled: boolean;
}

// ============================================================================
// Auth Config Types
// ============================================================================

export type AuthType = 'none' | 'bearer' | 'api_key' | 'api_key_query' | 'basic' | 'custom_header';

export interface AuthConfig {
  name: string;
  type: AuthType;
  description?: string;
  header_name?: string;
  query_param?: string;
  env_var?: string;
  username_env?: string;
  password_env?: string;
  token_endpoint?: TokenEndpointConfig;
  refresh_before_expiry?: number;
}

export interface TokenEndpointConfig {
  url?: string;
  url_env: string;
  method: string;
  username_env?: string;
  password_env?: string;
  headers?: Record<string, string>;
  body?: unknown;
  token_path: string;
  expires_path?: string;
}

export interface AuthConfigRequest {
  name: string;
  type: AuthType;
  description?: string;
  header_name?: string;
  query_param?: string;
  env_var?: string;
  username_env?: string;
  password_env?: string;
  token_endpoint?: TokenEndpointConfig;
  refresh_before_expiry?: number;
}

export interface TokenStatus {
  has_token: boolean;
  expires_at?: string;
  is_expired?: boolean;
  expires_in_seconds?: number;
}

// ============================================================================
// Metrics Types
// ============================================================================

export interface MetricsResponse {
  timestamp: string;
  uptime_seconds: number;
  outgoing: {
    total_requests: number;
    total_failures: number;
    requests_per_sec: number;
    success_rate: number;
    endpoint_count: number;
    domain_count: number;
    error_summary: {
      timeout: number;
      dns: number;
      connection: number;
      http: number;
    };
  };
  incoming: {
    available: boolean;
    total_requests?: number;
    requests_per_sec?: number;
    active_routes?: number;
  };
  outgoing_snapshot: MetricsSnapshot;
  incoming_snapshot: IncomingMetricsSnapshot | null;
}

export interface MetricsSnapshot {
  uptime_seconds: number;
  total_requests: number;
  total_successes: number;
  total_failures: number;
  success_rate: number;
  requests_per_second: number;
  collected_at: string;
  endpoints: Record<string, EndpointSnapshot>;
  dns_stats_by_domain: Record<string, DomainSnapshot>;
}

export interface EndpointSnapshot {
  total_requests: number;
  successful: number;
  failed: number;
  success_rate: number;
  timeout_errors: number;
  dns_errors: number;
  connection_errors: number;
  http_errors: number;
  other_errors: number;
  avg_total_time_ms: number;
  avg_dns_time_ms: number;
  avg_connect_time_ms: number;
  p95_total_time_ms: number;
  p99_total_time_ms: number;
  max_total_time_ms: number;
  p95_dns_time_ms: number;
  last_status_code: number;
  last_error: string;
  last_success: string;
  url_pattern: string;
  hostname: string;
}

export interface DomainSnapshot {
  total_lookups: number;
  successful_lookups: number;
  failed_lookups: number;
  avg_resolution_ms: number;
  p95_resolution_ms: number;
  max_resolution_ms: number;
  min_resolution_ms: number;
  last_error: string;
}

export interface IncomingMetricsSnapshot {
  uptime_seconds: number;
  total_requests: number;
  requests_per_second: number;
  collected_at: string;
  routes: Record<string, IncomingRouteSnapshot>;
}

export interface IncomingRouteSnapshot {
  total_requests: number;
  responses_by_status: Record<number, number>;
  avg_response_ms: number;
  p95_response_ms: number;
  p99_response_ms: number;
  max_response_ms: number;
  min_response_ms: number;
  last_request: string;
  route_name: string;
  route_path: string;
}


// ============================================================================
// Settings Types
// ============================================================================

export interface OutgoingSettings {
  global_multiplier: number;
  concurrent_requests: number;
  log_all_requests: boolean;
}

export interface SettingValue<T> {
  value: T;
}

// ============================================================================
// API Response Types
// ============================================================================

export interface ApiResponse<T> {
  data?: T;
  error?: string;
  message?: string;
}

export interface HealthResponse {
  status: string;
  uptime_seconds: number;
  goroutines: number;
  memory_mb: number;
}
