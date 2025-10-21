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
      // Always use relative URL since we're served from the same origin
      const response = await fetch('/api/profile', {
        headers: {
          Authorization: `Bearer ${authToken}`,
        },
        credentials: 'include',
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
      // Optimistically set as authenticated to prevent flashing
      setToken(storedToken);
      setIsAuthenticated(true);

      // Validate token - wait for it to complete before allowing page to render
      fetchUserProfile(storedToken)
        .then((success) => {
          if (!success) {
            // Only clear auth if profile fetch explicitly fails (not network errors)
            // This prevents logout on transient network issues
            console.warn('Session validation failed - token may be expired');
            localStorage.removeItem('auth-token');
            setToken(null);
            setUser(null);
            setIsAuthenticated(false);
          }
        })
        .catch((error) => {
          // Network error - keep user logged in, they'll get 401 on next API call
          console.warn('Failed to validate session (network error):', error);
          // Don't clear auth on network errors
        })
        .finally(() => {
          // Always stop loading after validation attempt
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
      await fetch('/logout', {
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
