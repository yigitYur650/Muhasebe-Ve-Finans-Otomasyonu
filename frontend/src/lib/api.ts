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
  isRetry?: boolean;
}

/**
 * Centralized API client for communicating with the Go Backend.
 * Automatically injects Supabase Bearer token, handles session refresh on 401, and retries seamlessly.
 */
export async function apiFetch<T>(endpoint: string, options: ApiOptions = {}): Promise<ApiEnvelope<T>> {
  const { tenantId, userId, userRole, idempotencyKey, authToken, isRetry, headers, ...customConfig } = options;

  const defaultHeaders: Record<string, string> = {
    'Content-Type': 'application/json',
  };

  // Automatically attach Supabase JWT access token if available
  let token = authToken;
  let supabaseClient: ReturnType<typeof createClient> | null = null;
  if (typeof window !== 'undefined') {
    try {
      supabaseClient = createClient();
      const { data } = await supabaseClient.auth.getSession();
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

  // Attach optional parameters if explicitly provided
  if (tenantId) defaultHeaders['X-Tenant-ID'] = tenantId;
  if (userId) defaultHeaders['X-User-ID'] = userId;
  if (userRole) defaultHeaders['X-User-Role'] = userRole;
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

    // Handle 401 Unauthorized with automatic session refresh & retry
    if (response.status === 401 && !isRetry && typeof window !== 'undefined' && supabaseClient) {
      try {
        const { data: refreshData, error: refreshErr } = await supabaseClient.auth.refreshSession();
        if (!refreshErr && refreshData?.session?.access_token) {
          // Retry the request with the new fresh token
          return await apiFetch<T>(endpoint, {
            ...options,
            authToken: refreshData.session.access_token,
            isRetry: true,
          });
        }
      } catch {
        // Refresh failed
      }
    }

    const data: ApiEnvelope<T> = await response.json();

    // If still 401 and in browser, sign out stale local session to avoid infinite loop
    if (response.status === 401 && typeof window !== 'undefined' && supabaseClient && !window.location.pathname.includes('/login')) {
      try {
        await supabaseClient.auth.signOut();
      } catch {
        // Ignore signout error
      }
    }

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
