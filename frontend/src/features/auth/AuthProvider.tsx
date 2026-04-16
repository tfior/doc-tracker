import { createContext, useContext, useEffect, useState } from 'react';
import type { ReactNode } from 'react';
import { getSession, login as apiLogin, logout as apiLogout } from '../../api/auth';

interface AuthState {
  authenticated: boolean;
  checking: boolean;
}

interface AuthContextValue extends AuthState {
  login: (email: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [state, setState] = useState<AuthState>({ authenticated: false, checking: true });

  useEffect(() => {
    getSession()
      .then(({ authenticated }) => setState({ authenticated, checking: false }))
      .catch(() => setState({ authenticated: false, checking: false }));
  }, []);

  async function login(email: string, password: string) {
    await apiLogin(email, password);
    setState({ authenticated: true, checking: false });
  }

  async function logout() {
    await apiLogout().catch(() => {});
    setState({ authenticated: false, checking: false });
  }

  return (
    <AuthContext.Provider value={{ ...state, login, logout }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth must be used within AuthProvider');
  return ctx;
}
