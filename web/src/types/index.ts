// Types matching Go models from backend

export type UserRole = 'admin' | 'operator' | 'viewer';

export interface User {
  id: number;
  username: string;
  is_admin: boolean;
  role: UserRole;
  totp_enabled?: boolean;
  created_at: string;
  updated_at: string;
}

export interface App {
  id: string;
  name: string;
  description: string;
  working_dir: string;
  token?: string;
  command_count?: number;
  created_at: string;
  updated_at: string;
}

export interface Command {
  id: string;
  app_id: string;
  name: string;
  description: string;
  command: string;
  timeout_seconds: number;
  sort_order: number;
  created_at: string;
}

export type ExecutionStatus = 'pending' | 'running' | 'success' | 'failed';

export interface Execution {
  id: string;
  command_id: string;
  user_id: number;
  status: ExecutionStatus;
  output: string;
  exit_code?: number;
  started_at?: string;
  finished_at?: string;
  created_at: string;
}

export interface ExecutionWithDetails extends Execution {
  command_name: string;
  app_name: string;
  username: string;
}

export interface AuditLog {
  id: number;
  user_id?: number;
  username: string;
  action: string;
  resource_type: string;
  resource_id?: string;
  ip_address: string;
  user_agent: string;
  details?: string;
  created_at: string;
}

export interface VersionInfo {
  version: string;
  build_time: string;
  git_commit: string;
  latest_version?: string;
  update_available: boolean;
}

// API Request/Response types
export interface LoginRequest {
  username: string;
  password: string;
  totp_code?: string;
}

export interface LoginResponse {
  message: string;
  user: User;
}

export interface CreateAppRequest {
  name: string;
  description: string;
  working_dir: string;
}

export interface UpdateAppRequest {
  name?: string;
  description?: string;
  working_dir?: string;
}

export interface CreateCommandRequest {
  name: string;
  description: string;
  command: string;
  timeout_seconds: number;
}

export interface UpdateCommandRequest {
  name?: string;
  description?: string;
  command?: string;
  timeout_seconds?: number;
}

export interface ExecuteCommandResponse {
  message: string;
  execution_id: string;
  app_id: string;
  app_name: string;
  stream_url: string;
  status_url: string;
}

export interface DeployRequest {
  command_id?: string;
}

export interface DeployResponse {
  message: string;
  execution_id: string;
  app_id: string;
  app_name: string;
  stream_url: string;
  status_url: string;
}

export interface ErrorResponse {
  error: string;
}

// API Configuration
export interface ApiConfig {
  baseUrl: string;
  pathPrefix: string;
}
