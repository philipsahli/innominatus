'use client'

import React, { createContext, useContext, useState, useEffect, ReactNode } from 'react'
import { useRouter } from 'next/navigation'

interface AuthContextType {
  isAuthenticated: boolean
  token: string | null
  login: (token: string) => void
  logout: () => void
  checkAuth: () => boolean
}

const AuthContext = createContext<AuthContextType | undefined>(undefined)

export function useAuth() {
  const context = useContext(AuthContext)
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider')
  }
  return context
}

interface AuthProviderProps {
  children: ReactNode
}

export function AuthProvider({ children }: AuthProviderProps) {
  const [token, setToken] = useState<string | null>(null)
  const [isAuthenticated, setIsAuthenticated] = useState(false)
  const [isLoading, setIsLoading] = useState(true)
  const router = useRouter()

  useEffect(() => {
    // Check for existing token on mount
    const storedToken = localStorage.getItem('auth-token')
    if (storedToken) {
      // Validate token by making a test API call
      fetch('http://localhost:8081/api/stats', {
        headers: {
          'Authorization': `Bearer ${storedToken}`
        }
      })
      .then(response => {
        if (response.ok) {
          setToken(storedToken)
          setIsAuthenticated(true)
        } else {
          // Token is invalid, remove it
          localStorage.removeItem('auth-token')
          setToken(null)
          setIsAuthenticated(false)
        }
      })
      .catch(() => {
        // Network error or invalid token
        localStorage.removeItem('auth-token')
        setToken(null)
        setIsAuthenticated(false)
      })
      .finally(() => {
        setIsLoading(false)
      })
    } else {
      // No token found
      setIsAuthenticated(false)
      setIsLoading(false)
    }
  }, [])

  const login = (newToken: string) => {
    localStorage.setItem('auth-token', newToken)
    setToken(newToken)
    setIsAuthenticated(true)
  }

  const logout = () => {
    localStorage.removeItem('auth-token')
    setToken(null)
    setIsAuthenticated(false)
    router.push('/login')
  }

  const checkAuth = (): boolean => {
    const currentToken = localStorage.getItem('auth-token')
    if (!currentToken) {
      setIsAuthenticated(false)
      setToken(null)
      return false
    }
    return true
  }

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="w-8 h-8 border-4 border-blue-500/30 border-t-blue-500 rounded-full animate-spin"></div>
      </div>
    )
  }

  return (
    <AuthContext.Provider value={{
      isAuthenticated,
      token,
      login,
      logout,
      checkAuth
    }}>
      {children}
    </AuthContext.Provider>
  )
}