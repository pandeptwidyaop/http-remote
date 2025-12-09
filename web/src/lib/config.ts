// Dynamic path prefix configuration
// Path prefix will be detected from current URL base path (before the #)
// Example: http://localhost:8080/devops/#/dashboard
//          Path prefix = /devops
//          Hash route = /dashboard

export function getPathPrefix(): string {
  // Check if path prefix is set in window object (injected by backend)
  if (typeof window !== 'undefined' && (window as any).__PATH_PREFIX__) {
    return (window as any).__PATH_PREFIX__;
  }

  // Extract from current URL pathname (everything before # or at root)
  if (typeof window !== 'undefined') {
    const pathname = window.location.pathname;
    // Remove trailing slash if present
    return pathname.endsWith('/') && pathname.length > 1
      ? pathname.slice(0, -1)
      : pathname === '/'
      ? ''
      : pathname;
  }

  // Fallback for SSR or development
  return import.meta.env.VITE_PATH_PREFIX || '';
}

export function getApiUrl(path: string): string {
  const prefix = getPathPrefix();
  const cleanPath = path.startsWith('/') ? path : `/${path}`;

  if (!prefix || prefix === '/') {
    return cleanPath;
  }

  return `${prefix}${cleanPath}`;
}

export function getBaseUrl(): string {
  const prefix = getPathPrefix();
  return prefix || '/';
}

export const API_ENDPOINTS = {
  // Auth
  login: '/api/auth/login',
  logout: '/api/auth/logout',
  me: '/api/auth/me',

  // Apps
  apps: '/api/apps',
  app: (id: string) => `/api/apps/${id}`,
  regenerateToken: (id: string) => `/api/apps/${id}/regenerate-token`,
  appCommands: (id: string) => `/api/apps/${id}/commands`,
  reorderCommands: (id: string) => `/api/apps/${id}/commands/reorder`,

  // Commands
  commands: '/api/commands',
  command: (id: string) => `/api/commands/${id}`,
  executeCommand: (id: string) => `/api/commands/${id}/execute`,

  // Executions
  executions: '/api/executions',
  execution: (id: string) => `/api/executions/${id}`,
  executionStream: (id: string) => `/api/executions/${id}/stream`,

  // Deploy
  deploy: (appId: string) => `/deploy/${appId}`,
  deployStatus: (appId: string, execId: string) => `/deploy/${appId}/status/${execId}`,
  deployStream: (appId: string, execId: string) => `/deploy/${appId}/stream/${execId}`,

  // Audit Logs
  auditLogs: '/api/audit-logs',

  // Version
  version: '/api/version',
  versionCheck: '/api/version/check',

  // Metrics
  metricsSystem: '/api/metrics/system',
  metricsDocker: '/api/metrics/docker',
  metricsDockerContainer: (id: string) => `/api/metrics/docker/${id}`,
  metricsDockerContainerHistory: (id: string) => `/api/metrics/docker/${id}/history`,
  metricsSummary: '/api/metrics/summary',
  metricsHistory: '/api/metrics/history',
  metricsStorage: '/api/metrics/storage',
  metricsStream: '/api/metrics/stream',
  metricsPrune: '/api/metrics/prune',
  metricsVacuum: '/api/metrics/vacuum',
} as const;
