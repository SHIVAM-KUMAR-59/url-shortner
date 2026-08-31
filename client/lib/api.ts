const BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL || 'http://localhost:8080';

export class ApiError extends Error {
  status: number;
  constructor(message: string, status: number) {
    super(message);
    this.status = status;
  }
}

async function request<T>(
  path: string,
  options: RequestInit = {},
  apiKey?: string
): Promise<T> {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(options.headers as Record<string, string>),
  };

  if (apiKey) {
    headers['Authorization'] = `Bearer ${apiKey}`;
  }

  const res = await fetch(`${BASE_URL}${path}`, { ...options, headers });

  if (!res.ok) {
    let message = `Request failed (${res.status})`;
    try {
      const body = await res.json();
      message = body.error || message;
    } catch {
      // response wasn't JSON — keep the generic message
    }
    throw new ApiError(message, res.status);
  }

  if (res.status === 204) {
    return undefined as T;
  }

  return res.json();
}

export interface RegisterResponse {
  user_id: number;
  email: string;
  api_key: string;
}

export interface ShortenResponse {
  short_code: string;
}

export interface StatsResponse {
  short_code: string;
  long_url: string;
  created_at: string;
  total_clicks: number;
  last_clicked?: string;
}

export function registerUser(email: string): Promise<RegisterResponse> {
  return request<RegisterResponse>('/api/v1/users', {
    method: 'POST',
    body: JSON.stringify({ email }),
  });
}

export function shortenUrl(
  longUrl: string,
  apiKey: string
): Promise<ShortenResponse> {
  return request<ShortenResponse>(
    '/api/v1/shorten',
    { method: 'POST', body: JSON.stringify({ long_url: longUrl }) },
    apiKey
  );
}

export function getStats(
  shortCode: string,
  apiKey: string
): Promise<StatsResponse> {
  return request<StatsResponse>(
    `/api/v1/${shortCode}/stats`,
    { method: 'GET' },
    apiKey
  );
}

export function getRedirectUrl(shortCode: string): string {
  return `${BASE_URL}/${shortCode}`;
}