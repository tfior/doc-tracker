import { get, post, del, ApiError } from './client';

export async function getSession(): Promise<{ authenticated: boolean }> {
  try {
    await get<{ authenticated: boolean }>('/auth/session');
    return { authenticated: true };
  } catch (err) {
    if (err instanceof ApiError && err.status === 401) {
      return { authenticated: false };
    }
    throw err;
  }
}

export async function login(email: string, password: string): Promise<void> {
  await post('/auth/session', { email, password });
}

export async function logout(): Promise<void> {
  await del('/auth/session');
}
