import { createClient } from './supabase/client';

const rawApiUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api/v1';

export function getApiBaseUrl(): string {
  let url = rawApiUrl.trim().replace(/\/+$/, '');
  if (!url.endsWith('/api/v1')) {
    if (url.endsWith('/api')) {
      url += '/v1';
    } else {
      url += '/api/v1';
    }
  }
  return url;
}

export function getApiUrl(endpoint: string): string {
  const baseUrl = getApiBaseUrl();
  const path = endpoint.startsWith('/') ? endpoint : `/${endpoint}`;
  return `${baseUrl}${path}`;
}

export interface ApiEnvelope<T> {
  success: boolean;
  data?: T;
  error?: {
    code: string;
    message: string;
  };
}

export interface ApiOptions extends RequestInit {
  tenantId?: string;
  userId?: string;
  userRole?: string;
  idempotencyKey?: string;
  authToken?: string;
}

/**
 * Centralized API client for communicating with the Go Backend.
 * Automatically injects Supabase Bearer token and tenant headers.
 */
export async function apiFetch<T>(endpoint: string, options: ApiOptions = {}): Promise<ApiEnvelope<T>> {
  const { tenantId, userId, userRole, idempotencyKey, authToken, headers, ...customConfig } = options;

  const defaultHeaders: Record<string, string> = {
    'Content-Type': 'application/json',
  };

  // Automatically attach Supabase JWT access token if available
  let token = authToken;
  if (!token && typeof window !== 'undefined') {
    try {
      const supabase = createClient();
      const { data } = await supabase.auth.getSession();
      if (data?.session?.access_token) {
        token = data.session.access_token;
      }
    } catch {
      // Fallback
    }
  }

  if (token) {
    defaultHeaders['Authorization'] = `Bearer ${token}`;
  }

  // Ensure default tenant and user headers exist if not explicitly provided
  defaultHeaders['X-Tenant-ID'] = tenantId || '00000000-0000-0000-0000-000000000001';
  defaultHeaders['X-User-ID'] = userId || '00000000-0000-0000-0000-000000000002';
  defaultHeaders['X-User-Role'] = userRole || 'admin';
  if (idempotencyKey) defaultHeaders['Idempotency-Key'] = idempotencyKey;

  const config: RequestInit = {
    ...customConfig,
    headers: {
      ...defaultHeaders,
      ...headers,
    },
  };

  try {
    const response = await fetch(getApiUrl(endpoint), config);
    const data: ApiEnvelope<T> = await response.json();
    return data;
  } catch (error: any) {
    return {
      success: false,
      error: {
        code: 'NETWORK_ERROR',
        message: error?.message || 'Sunucuya bağlanılamadı.',
      },
    };
  }
}
