/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

const BASE = '/api/v1';

export interface FieldError {
  field: string;
  code: string;
  message: string;
}

export class ApiError extends Error {
  readonly status: number;
  readonly code: string;
  readonly fields: FieldError[];

  constructor(status: number, code: string, message: string, fields: FieldError[] = []) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.code = code;
    this.fields = fields;
  }
}

// Called when any request other than the session check or login is refused for
// want of a session, so the app can send the user back to the login screen.
let onUnauthorized: () => void = () => {};

export function setUnauthorizedHandler(handler: () => void) {
  onUnauthorized = handler;
}

async function parseError(res: Response): Promise<ApiError> {
  const data = await res.json().catch(() => ({})) as {
    error?: { code: string; message: string; fields?: FieldError[] };
  };
  return new ApiError(
    res.status,
    data?.error?.code ?? 'unknown',
    data?.error?.message ?? `The server responded with an error (${res.status}).`,
    data?.error?.fields ?? [],
  );
}

async function request<T>(method: string, path: string, body?: unknown, params?: Record<string, string | number>): Promise<T> {
  const url = new URL(BASE + path, window.location.origin);
  for (const [key, value] of Object.entries(params ?? {})) {
    url.searchParams.set(key, String(value));
  }

  let res: Response;
  try {
    res = await fetch(url.toString(), {
      method,
      headers: body !== undefined ? { 'Content-Type': 'application/json' } : {},
      body: body !== undefined ? JSON.stringify(body) : undefined,
    });
  } catch {
    throw new ApiError(0, 'network', 'Could not reach the server. Check your connection and try again.');
  }

  if (!res.ok) {
    const err = await parseError(res);
    if (res.status === 401 && path !== '/auth/session') onUnauthorized();
    throw err;
  }
  if (res.status === 204) return undefined as T;
  return res.json() as Promise<T>;
}

function get<T>(path: string, params?: Record<string, string | number>): Promise<T> {
  return request<T>('GET', path, undefined, params);
}

function post<T>(path: string, body?: unknown): Promise<T> {
  return request<T>('POST', path, body);
}

function patch<T>(path: string, body: unknown): Promise<T> {
  return request<T>('PATCH', path, body);
}

function del(path: string, params?: Record<string, string | number>): Promise<void> {
  return request<void>('DELETE', path, undefined, params);
}

/** A list returned in full. Lists inside a case are never paginated. */
export interface Items<T> {
  items: T[];
  total: number;
}

export { get, post, patch, del };
