import { getApiUrl } from '@/lib/config';
import type { ErrorResponse } from '@/types';

// CSRF token management
function getCSRFToken(): string {
  const cookies = document.cookie.split(';');
  for (const cookie of cookies) {
    const [name, value] = cookie.trim().split('=');
    if (name === 'csrf_token') {
      return value;
    }
  }
  return '';
}

export class ApiError extends Error {
  constructor(
    public status: number,
    public statusText: string,
    message: string
  ) {
    super(message);
    this.name = 'ApiError';
  }
}

export interface FetchOptions extends RequestInit {
  timeout?: number;
}

async function fetchWithTimeout(
  url: string,
  options: FetchOptions = {}
): Promise<Response> {
  const { timeout = 30000, ...fetchOptions } = options;

  const controller = new AbortController();
  const timeoutId = setTimeout(() => controller.abort(), timeout);

  // Build headers with CSRF token for state-changing methods
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...fetchOptions.headers as Record<string, string>,
  };

  // Add CSRF token for non-GET methods
  const method = fetchOptions.method || 'GET';
  if (method !== 'GET' && method !== 'HEAD' && method !== 'OPTIONS') {
    const csrfToken = getCSRFToken();
    if (csrfToken) {
      headers['X-CSRF-Token'] = csrfToken;
    }
  }

  try {
    const response = await fetch(url, {
      ...fetchOptions,
      signal: controller.signal,
      credentials: 'include', // Important for session cookies
      headers,
    });

    clearTimeout(timeoutId);
    return response;
  } catch (error) {
    clearTimeout(timeoutId);
    if (error instanceof Error && error.name === 'AbortError') {
      throw new ApiError(408, 'Request Timeout', 'Request timed out');
    }
    throw error;
  }
}

async function handleResponse<T>(response: Response): Promise<T> {
  if (!response.ok) {
    let errorMessage = response.statusText;

    try {
      const errorData: ErrorResponse = await response.json();
      errorMessage = errorData.error || errorMessage;
    } catch {
      // If JSON parsing fails, use statusText
    }

    throw new ApiError(response.status, response.statusText, errorMessage);
  }

  // Handle empty responses (204 No Content)
  if (response.status === 204) {
    return {} as T;
  }

  return response.json();
}

export const api = {
  async get<T>(endpoint: string, options?: FetchOptions): Promise<T> {
    const url = getApiUrl(endpoint);
    const response = await fetchWithTimeout(url, {
      ...options,
      method: 'GET',
    });
    return handleResponse<T>(response);
  },

  async post<T>(endpoint: string, data?: any, options?: FetchOptions): Promise<T> {
    const url = getApiUrl(endpoint);
    const response = await fetchWithTimeout(url, {
      ...options,
      method: 'POST',
      body: data ? JSON.stringify(data) : undefined,
    });
    return handleResponse<T>(response);
  },

  async put<T>(endpoint: string, data?: any, options?: FetchOptions): Promise<T> {
    const url = getApiUrl(endpoint);
    const response = await fetchWithTimeout(url, {
      ...options,
      method: 'PUT',
      body: data ? JSON.stringify(data) : undefined,
    });
    return handleResponse<T>(response);
  },

  async delete<T>(endpoint: string, options?: FetchOptions): Promise<T> {
    const url = getApiUrl(endpoint);
    const response = await fetchWithTimeout(url, {
      ...options,
      method: 'DELETE',
    });
    return handleResponse<T>(response);
  },

  // Special method for SSE streams
  createEventSource(endpoint: string): EventSource {
    const url = getApiUrl(endpoint);
    return new EventSource(url, { withCredentials: true });
  },
};
