'use client'

import { useState, useEffect, useCallback } from 'react'
import { api, type ApiResponse } from '@/lib/api'
import { useAuth } from '@/contexts/auth-context'

export function useApi<T>(
  apiCall: () => Promise<ApiResponse<T>>,
  dependencies: any[] = []
) {
  const [data, setData] = useState<T | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const { isAuthenticated } = useAuth()

  const refetch = useCallback(async () => {
    // Don't make API calls if not authenticated
    if (!isAuthenticated) {
      setLoading(false)
      setData(null)
      setError(null) // Don't show error for unauthenticated state
      return
    }

    setLoading(true)
    setError(null)

    try {
      const response = await apiCall()
      if (response.success) {
        setData(response.data || null)
      } else {
        setError(response.error || 'Unknown error')
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error')
    } finally {
      setLoading(false)
    }
  }, [apiCall, isAuthenticated])

  useEffect(() => {
    refetch()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isAuthenticated, ...dependencies])

  return { data, loading, error, refetch }
}

// Specific hooks for common API calls
export function useApplications() {
  return useApi(() => api.getApplications())
}

export function useApplication(name: string | null) {
  return useApi(
    () => name ? api.getApplication(name) : Promise.resolve({ success: true, data: null }),
    [name]
  )
}

export function useWorkflows(appName?: string) {
  return useApi(() => api.getWorkflows(appName), [appName])
}

export function useWorkflow(id: string | null) {
  return useApi(
    () => id ? api.getWorkflow(id) : Promise.resolve({ success: true, data: null }),
    [id]
  )
}

export function useResourceGraph(appName: string | null) {
  return useApi(
    () => appName ? api.getResourceGraph(appName) : Promise.resolve({ success: true, data: null }),
    [appName]
  )
}

export function useStats() {
  return useApi(() => api.getStats())
}

export function useResources(appName?: string) {
  return useApi(() => api.getResources(appName), [appName])
}

export function useResource(id: number | null) {
  return useApi(
    () => id ? api.getResource(id) : Promise.resolve({ success: true, data: null }),
    [id]
  )
}

// Mutation hook for actions that modify data
export function useApiMutation<T, P = any>(
  apiCall: (params: P) => Promise<ApiResponse<T>>
) {
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const mutate = useCallback(async (params: P): Promise<ApiResponse<T>> => {
    setLoading(true)
    setError(null)

    try {
      const response = await apiCall(params)
      if (!response.success) {
        setError(response.error || 'Unknown error')
      }
      return response
    } catch (err) {
      const error = err instanceof Error ? err.message : 'Unknown error'
      setError(error)
      return { success: false, error }
    } finally {
      setLoading(false)
    }
  }, [apiCall])

  return { mutate, loading, error }
}

// Specific mutation hooks
export function useDeployApplication() {
  return useApiMutation((scoreSpec: string) => api.deployApplication(scoreSpec))
}

export function useDeleteApplication() {
  return useApiMutation((name: string) => api.deleteApplication(name))
}

// Demo environment hooks
export function useDemoStatus() {
  return useApi(() => api.getDemoStatus())
}

export function useDemoActions() {
  return {
    runDemoTime: useApiMutation(() => api.runDemoTime()),
    runDemoNuke: useApiMutation(() => api.runDemoNuke())
  }
}