'use client';

import React, { createContext, useContext, useState, useEffect, ReactNode } from 'react';
import { useRouter } from 'next/navigation';

interface UserProfile {
  username: string;
  team: string;
  role: string;
}

interface AuthContextType {
  isAuthenticated: boolean;
  token: string | null;
  user: UserProfile | null;
  isAdmin: boolean;
  login: (token: string) => void;
  logout: () => void;
  checkAuth: () => boolean;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export function useAuth() {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
}

interface AuthProviderProps {
  children: ReactNode;
}

export function AuthProvider({ children }: AuthProviderProps) {
  const [token, setToken] = useState<string | null>(null);
  const [user, setUser] = useState<UserProfile | null>(null);
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
  const router = useRouter();

  // Fetch user profile from API
  const fetchUserProfile = async (authToken: string): Promise<boolean> => {
    try {
      const response = await fetch('http://localhost:8081/api/profile', {
        headers: {
          Authorization: `Bearer ${authToken}`,
        },
      });

      if (response.ok) {
        const profile = await response.json();
        setUser(profile);
        return true;
      } else {
        setUser(null);
        return false;
      }
    } catch (error) {
      console.error('Failed to fetch user profile:', error);
      setUser(null);
      return false;
    }
  };

  useEffect(() => {
    // Check for existing token on mount
    const storedToken = localStorage.getItem('auth-token');
    if (storedToken) {
      // Validate token and fetch user profile
      fetchUserProfile(storedToken)
        .then((success) => {
          if (success) {
            setToken(storedToken);
            setIsAuthenticated(true);
          } else {
            // Token is invalid, remove it
            localStorage.removeItem('auth-token');
            setToken(null);
            setUser(null);
            setIsAuthenticated(false);
          }
        })
        .catch(() => {
          // Network error or invalid token
          localStorage.removeItem('auth-token');
          setToken(null);
          setUser(null);
          setIsAuthenticated(false);
        })
        .finally(() => {
          setIsLoading(false);
        });
    } else {
      // No token found
      setIsAuthenticated(false);
      setIsLoading(false);
    }
  }, []);

  const login = async (newToken: string) => {
    localStorage.setItem('auth-token', newToken);
    setToken(newToken);
    setIsAuthenticated(true);
    // Fetch user profile after login
    await fetchUserProfile(newToken);
  };

  const logout = async () => {
    try {
      // Call backend logout endpoint to clear server-side session
      await fetch('http://localhost:8081/logout', {
        method: 'GET',
        credentials: 'include', // Important: Include cookies for session management
      });
    } catch (error) {
      console.error('Error calling logout endpoint:', error);
      // Continue with client-side logout even if backend call fails
    }

    // Clear client-side state
    localStorage.removeItem('auth-token');
    setToken(null);
    setUser(null);
    setIsAuthenticated(false);
    router.push('/login');
  };

  const checkAuth = (): boolean => {
    const currentToken = localStorage.getItem('auth-token');
    if (!currentToken) {
      setIsAuthenticated(false);
      setToken(null);
      return false;
    }
    return true;
  };

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="w-8 h-8 border-4 border-blue-500/30 border-t-blue-500 rounded-full animate-spin"></div>
      </div>
    );
  }

  return (
    <AuthContext.Provider
      value={{
        isAuthenticated,
        token,
        user,
        isAdmin: user?.role === 'admin',
        login,
        logout,
        checkAuth,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}
