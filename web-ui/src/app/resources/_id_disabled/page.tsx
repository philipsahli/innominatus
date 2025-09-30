'use client'

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import {
  Package,
  RefreshCw,
  ArrowLeft,
  Database,
  Server,
  HardDrive,
  Activity,
  CheckCircle,
  XCircle,
  Clock,
  AlertTriangle,
  Trash2,
  Calendar,
  Zap,
  Settings,
  Eye,
  History
} from "lucide-react"
import { ProtectedRoute } from "@/components/protected-route"
import { useResource } from "@/hooks/use-api"
import { useParams, useRouter } from "next/navigation"

function getStatusBadge(state: string, healthStatus: string) {
  const isHealthy = healthStatus === 'healthy'
  const isDegraded = healthStatus === 'degraded'

  switch (state) {
    case 'active':
      if (isHealthy) {
        return <Badge variant="default" className="bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200">
          <CheckCircle className="w-3 h-3 mr-1" />
          Active
        </Badge>
      } else if (isDegraded) {
        return <Badge variant="secondary" className="bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200">
          <AlertTriangle className="w-3 h-3 mr-1" />
          Degraded
        </Badge>
      } else {
        return <Badge variant="destructive">
          <XCircle className="w-3 h-3 mr-1" />
          Unhealthy
        </Badge>
      }
    case 'provisioning':
    case 'scaling':
    case 'updating':
      return <Badge variant="secondary" className="bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200">
        <Zap className="w-3 h-3 mr-1" />
        {state.charAt(0).toUpperCase() + state.slice(1)}
      </Badge>
    case 'requested':
    case 'pending':
      return <Badge variant="outline">
        <Clock className="w-3 h-3 mr-1" />
        Pending
      </Badge>
    case 'terminating':
      return <Badge variant="secondary" className="bg-orange-100 text-orange-800 dark:bg-orange-900 dark:text-orange-200">
        <Trash2 className="w-3 h-3 mr-1" />
        Terminating
      </Badge>
    case 'terminated':
      return <Badge variant="outline" className="text-gray-500">
        <XCircle className="w-3 h-3 mr-1" />
        Terminated
      </Badge>
    case 'failed':
      return <Badge variant="destructive">
        <XCircle className="w-3 h-3 mr-1" />
        Failed
      </Badge>
    default:
      return <Badge variant="outline">
        <Clock className="w-3 h-3 mr-1" />
        {state}
      </Badge>
  }
}

function getResourceTypeIcon(type: string) {
  switch (type.toLowerCase()) {
    case 'postgres':
    case 'postgresql':
    case 'database':
      return <Database className="w-6 h-6" />
    case 'redis':
    case 'cache':
      return <Server className="w-6 h-6" />
    case 'volume':
    case 'storage':
      return <HardDrive className="w-6 h-6" />
    default:
      return <Package className="w-6 h-6" />
  }
}

function formatTimestamp(timestamp: string) {
  return new Date(timestamp).toLocaleString()
}


export default function ResourceDetailsPage() {
  const params = useParams()
  const router = useRouter()
  const resourceId = params.id ? parseInt(params.id as string) : null
  const { data: resource, loading: resourceLoading, error: resourceError, refetch: refetchResource } = useResource(resourceId)

  const handleRefresh = () => {
    refetchResource()
  }

  const handleBack = () => {
    router.push('/resources')
  }

  if (resourceLoading) {
    return (
      <ProtectedRoute>
        <div className="min-h-screen bg-white dark:bg-gray-900 p-6">
          <div className="flex items-center justify-center p-12">
            <RefreshCw className="w-6 h-6 animate-spin mr-2 text-muted-foreground" />
            <span className="text-muted-foreground">Loading resource details...</span>
          </div>
        </div>
      </ProtectedRoute>
    )
  }

  if (resourceError || !resource) {
    return (
      <ProtectedRoute>
        <div className="min-h-screen bg-white dark:bg-gray-900 p-6">
          <div className="space-y-6">
            <div className="flex items-center gap-4">
              <Button variant="outline" onClick={handleBack}>
                <ArrowLeft className="w-4 h-4 mr-2" />
                Back to Resources
              </Button>
            </div>
            <Card className="bg-gray-50 dark:bg-gray-800 border-gray-200 dark:border-gray-700">
              <CardContent className="pt-6">
                <p className="text-gray-800 dark:text-gray-200 text-sm">
                  Resource not found or error loading resource details: {resourceError}
                </p>
              </CardContent>
            </Card>
          </div>
        </div>
      </ProtectedRoute>
    )
  }

  return (
    <ProtectedRoute>
      <div className="min-h-screen bg-white dark:bg-gray-900 p-6">
        <div className="relative space-y-6">
          {/* Header */}
          <div className="space-y-4">
            <div className="flex items-center gap-4">
              <Button variant="outline" onClick={handleBack}>
                <ArrowLeft className="w-4 h-4 mr-2" />
                Back to Resources
              </Button>
              <Button
                variant="outline"
                onClick={handleRefresh}
                disabled={resourceLoading}
              >
                <RefreshCw className={`w-4 h-4 mr-2 ${resourceLoading ? 'animate-spin' : ''}`} />
                Refresh
              </Button>
            </div>

            <div className="flex items-center gap-4">
              <div className="p-3 rounded-lg bg-gray-200 dark:bg-gray-700">
                {getResourceTypeIcon(resource.resource_type)}
              </div>
              <div>
                <h1 className="text-3xl font-bold tracking-tight text-gray-900 dark:text-gray-100">
                  {resource.resource_name}
                </h1>
                <p className="text-gray-600 dark:text-gray-400">
                  {resource.resource_type} resource in {resource.application_name}
                </p>
              </div>
            </div>
          </div>

          {/* Status Overview */}
          <div className="grid gap-4 md:grid-cols-4">
            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">State</CardTitle>
                <Activity className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                {getStatusBadge(resource.state, resource.health_status)}
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">Health</CardTitle>
                <CheckCircle className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <Badge variant="outline" className="text-xs">
                  {resource.health_status}
                </Badge>
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">Created</CardTitle>
                <Calendar className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-sm">{formatTimestamp(resource.created_at)}</div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">Last Updated</CardTitle>
                <History className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-sm">{formatTimestamp(resource.updated_at)}</div>
              </CardContent>
            </Card>
          </div>

          {/* Main Content */}
          <div className="grid gap-6 lg:grid-cols-2">
            {/* Basic Information */}
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <Eye className="w-4 h-4" />
                  Basic Information
                </CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="grid gap-2">
                  <label className="text-sm font-medium">Resource ID</label>
                  <div className="text-sm text-muted-foreground">{resource.id}</div>
                </div>
                <div className="grid gap-2">
                  <label className="text-sm font-medium">Resource Name</label>
                  <div className="text-sm">{resource.resource_name}</div>
                </div>
                <div className="grid gap-2">
                  <label className="text-sm font-medium">Resource Type</label>
                  <Badge variant="secondary">{resource.resource_type}</Badge>
                </div>
                <div className="grid gap-2">
                  <label className="text-sm font-medium">Application</label>
                  <Badge variant="outline">{resource.application_name}</Badge>
                </div>
                {resource.provider_id && (
                  <div className="grid gap-2">
                    <label className="text-sm font-medium">Provider ID</label>
                    <div className="text-sm text-muted-foreground font-mono">{resource.provider_id}</div>
                  </div>
                )}
                {resource.last_health_check && (
                  <div className="grid gap-2">
                    <label className="text-sm font-medium">Last Health Check</label>
                    <div className="text-sm text-muted-foreground">{formatTimestamp(resource.last_health_check)}</div>
                  </div>
                )}
                {resource.error_message && (
                  <div className="grid gap-2">
                    <label className="text-sm font-medium">Error Message</label>
                    <div className="text-sm text-red-600 dark:text-red-400 bg-red-50 dark:bg-red-900/20 p-2 rounded">
                      {resource.error_message}
                    </div>
                  </div>
                )}
              </CardContent>
            </Card>

            {/* Configuration */}
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <Settings className="w-4 h-4" />
                  Configuration
                </CardTitle>
              </CardHeader>
              <CardContent>
                {Object.keys(resource.configuration).length > 0 ? (
                  <div className="space-y-2">
                    {Object.entries(resource.configuration).map(([key, value]) => (
                      <div key={key} className="flex justify-between items-center py-1">
                        <span className="text-sm font-medium">{key}:</span>
                        <span className="text-sm text-muted-foreground font-mono">
                          {typeof value === 'object' ? JSON.stringify(value) : String(value)}
                        </span>
                      </div>
                    ))}
                  </div>
                ) : (
                  <div className="text-center text-muted-foreground py-4">
                    No configuration data available
                  </div>
                )}
              </CardContent>
            </Card>

            {/* Provider Metadata */}
            {resource.provider_metadata && Object.keys(resource.provider_metadata).length > 0 && (
              <Card className="lg:col-span-2">
                <CardHeader>
                  <CardTitle className="flex items-center gap-2">
                    <Server className="w-4 h-4" />
                    Provider Metadata
                  </CardTitle>
                  <CardDescription>
                    Information from the resource provider
                  </CardDescription>
                </CardHeader>
                <CardContent>
                  <div className="grid gap-2 md:grid-cols-2">
                    {Object.entries(resource.provider_metadata).map(([key, value]) => (
                      <div key={key} className="flex justify-between items-center py-1">
                        <span className="text-sm font-medium">{key}:</span>
                        <span className="text-sm text-muted-foreground font-mono">
                          {typeof value === 'object' ? JSON.stringify(value) : String(value)}
                        </span>
                      </div>
                    ))}
                  </div>
                </CardContent>
              </Card>
            )}
          </div>
        </div>
      </div>
    </ProtectedRoute>
  )
}