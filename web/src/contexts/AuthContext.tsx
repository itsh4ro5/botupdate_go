import React, { createContext, useContext, useState, useEffect, type ReactNode } from 'react';
import { api } from '../services/api';

export interface User {
  id: string;
  username: string;
  role: string;
  must_change_password?: boolean;
}

interface AuthState {
  authenticated: boolean;
  user: User | null;
  loading: boolean;
}

interface AuthContextType extends AuthState {
  login: (user: User) => void;
  logout: () => void;
  refreshSession: () => Promise<void>;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export const AuthProvider: React.FC<{ children: ReactNode }> = ({ children }) => {
  const [state, setState] = useState<AuthState>({
    authenticated: false,
    user: null,
    loading: true,
  });

  const refreshSession = async () => {
    try {
      const data = await api.get<{ authenticated: boolean; user: User }>('/auth/me');
      setState({
        authenticated: data.authenticated,
        user: data.user,
        loading: false,
      });
    } catch (error) {
      setState({
        authenticated: false,
        user: null,
        loading: false,
      });
    }
  };

  useEffect(() => {
    refreshSession();
  }, []);

  const login = (user: User) => {
    setState({ authenticated: true, user, loading: false });
  };

  const logout = async () => {
    try {
      await api.post('/auth/logout', {});
    } catch (e) {
      // ignore
    } finally {
      setState({ authenticated: false, user: null, loading: false });
    }
  };

  return (
    <AuthContext.Provider value={{ ...state, login, logout, refreshSession }}>
      {children}
    </AuthContext.Provider>
  );
};

export const useAuth = () => {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
};
