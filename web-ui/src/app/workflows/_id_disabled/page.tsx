'use client'

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import {
  Activity,
  RefreshCw,
  ArrowLeft,
  Clock,
  CheckCircle,
  XCircle,
  Zap,
  Calendar,
  Timer,
  Play,
  FileText
} from "lucide-react"
import { ProtectedRoute } from "@/components/protected-route"
import { useWorkflow } from "@/hooks/use-api"
import { useParams, useRouter } from "next/navigation"


function getStatusBadge(status: string) {
  switch (status) {
    case 'completed':
      return <Badge variant="default" className="bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200">
        <CheckCircle className="w-3 h-3 mr-1" />
        Completed
      </Badge>
    case 'running':
      return <Badge variant="secondary" className="bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200">
        <Zap className="w-3 h-3 mr-1" />
        Running
      </Badge>
    case 'failed':
      return <Badge variant="destructive">
        <XCircle className="w-3 h-3 mr-1" />
        Failed
      </Badge>
    case 'pending':
      return <Badge variant="outline">
        <Clock className="w-3 h-3 mr-1" />
        Pending
      </Badge>
    default:
      return <Badge variant="outline">
        <Clock className="w-3 h-3 mr-1" />
        {status}
      </Badge>
  }
}

function getStepStatusIcon(status: string) {
  switch (status) {
    case 'completed':
      return <CheckCircle className="w-4 h-4 text-green-500" />
    case 'running':
      return <Zap className="w-4 h-4 text-blue-500" />
    case 'failed':
      return <XCircle className="w-4 h-4 text-red-500" />
    case 'pending':
      return <Clock className="w-4 h-4 text-gray-400" />
    default:
      return <Clock className="w-4 h-4 text-gray-400" />
  }
}

function formatTimestamp(timestamp: string) {
  return new Date(timestamp).toLocaleString()
}

function formatDuration(startedAt: string, completedAt?: string) {
  if (!completedAt) {
    const now = new Date()
    const start = new Date(startedAt)
    const diffMs = now.getTime() - start.getTime()
    const diffSeconds = Math.floor(diffMs / 1000)
    const diffMinutes = Math.floor(diffSeconds / 60)
    const remainingSeconds = diffSeconds % 60

    if (diffMinutes > 0) {
      return `${diffMinutes}m ${remainingSeconds}s (running)`
    } else {
      return `${diffSeconds}s (running)`
    }
  }

  const start = new Date(startedAt)
  const end = new Date(completedAt)
  const diffMs = end.getTime() - start.getTime()
  const diffSeconds = Math.floor(diffMs / 1000)
  const diffMinutes = Math.floor(diffSeconds / 60)
  const remainingSeconds = diffSeconds % 60

  if (diffMinutes > 0) {
    return `${diffMinutes}m ${remainingSeconds}s`
  } else {
    return `${diffSeconds}s`
  }
}



export default function WorkflowExecutionDetailsPage() {
  const params = useParams()
  const router = useRouter()
  const workflowId = params.id as string
  const { data: workflowDetail, loading: workflowLoading, error: workflowError, refetch: refetchWorkflow } = useWorkflow(workflowId)

  const handleRefresh = () => {
    refetchWorkflow()
  }

  const handleBack = () => {
    router.push('/workflows')
  }

  if (workflowLoading) {
    return (
      <ProtectedRoute>
        <div className="min-h-screen bg-white dark:bg-gray-900 p-6">
          <div className="flex items-center justify-center p-12">
            <RefreshCw className="w-6 h-6 animate-spin mr-2 text-muted-foreground" />
            <span className="text-muted-foreground">Loading workflow execution details...</span>
          </div>
        </div>
      </ProtectedRoute>
    )
  }

  if (workflowError || !workflowDetail) {
    return (
      <ProtectedRoute>
        <div className="min-h-screen bg-white dark:bg-gray-900 p-6">
          <div className="space-y-6">
            <div className="flex items-center gap-4">
              <Button variant="outline" onClick={handleBack}>
                <ArrowLeft className="w-4 h-4 mr-2" />
                Back to Workflows
              </Button>
            </div>
            <Card className="bg-gray-50 dark:bg-gray-800 border-gray-200 dark:border-gray-700">
              <CardContent className="pt-6">
                <p className="text-gray-800 dark:text-gray-200 text-sm">
                  Workflow execution not found or error loading details: {workflowError}
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
                Back to Workflows
              </Button>
              <Button
                variant="outline"
                onClick={handleRefresh}
                disabled={workflowLoading}
              >
                <RefreshCw className={`w-4 h-4 mr-2 ${workflowLoading ? 'animate-spin' : ''}`} />
                Refresh
              </Button>
            </div>

            <div className="flex items-center gap-4">
              <div className="p-3 rounded-lg bg-gray-200 dark:bg-gray-700">
                <Activity className="w-6 h-6 text-gray-900 dark:text-gray-100" />
              </div>
              <div>
                <h1 className="text-3xl font-bold tracking-tight text-gray-900 dark:text-gray-100">
                  {workflowDetail.workflow_name}
                </h1>
                <p className="text-gray-600 dark:text-gray-400">
                  Workflow execution #{workflowDetail.id} {workflowDetail.application_name && `for ${workflowDetail.application_name}`}
                </p>
              </div>
            </div>
          </div>

          {/* Status Overview */}
          <div className="grid gap-4 md:grid-cols-4">
            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">Status</CardTitle>
                <Activity className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                {getStatusBadge(workflowDetail.status)}
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">Duration</CardTitle>
                <Timer className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-lg font-semibold">{formatDuration(workflowDetail.started_at, workflowDetail.completed_at)}</div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">Started</CardTitle>
                <Calendar className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-sm">{formatTimestamp(workflowDetail.started_at)}</div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">Steps</CardTitle>
                <FileText className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-lg font-semibold">{workflowDetail.steps.length}</div>
              </CardContent>
            </Card>
          </div>

          {/* Main Content */}
          <div className="grid gap-6">
            {/* Workflow Steps */}
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <Play className="w-4 h-4" />
                  Execution Steps
                </CardTitle>
                <CardDescription>
                  Step-by-step progress of the workflow execution
                </CardDescription>
              </CardHeader>
              <CardContent>
                <div className="space-y-4">
                  {workflowDetail.steps.map((step) => (
                    <div key={step.id} className="border rounded-lg p-4">
                      <div className="flex items-center justify-between mb-3">
                        <div className="flex items-center gap-3">
                          {getStepStatusIcon(step.status)}
                          <div>
                            <h4 className="font-medium">
                              Step {step.step_number}: {step.step_name}
                            </h4>
                            <p className="text-sm text-muted-foreground">
                              Type: {step.step_type}
                            </p>
                          </div>
                        </div>
                        <div className="text-right">
                          {step.duration_ms && (
                            <div className="text-sm font-mono">
                              {Math.floor(step.duration_ms / 1000)}s
                            </div>
                          )}
                          <div className="text-xs text-muted-foreground">
                            {formatTimestamp(step.started_at)}
                          </div>
                        </div>
                      </div>

                      {step.output_logs && (
                        <div className="mt-3">
                          <h5 className="text-sm font-medium mb-2">Output:</h5>
                          <div className="bg-gray-50 dark:bg-gray-800 rounded-lg p-3 max-h-64 overflow-y-auto">
                            <pre className="text-xs text-gray-800 dark:text-gray-200 whitespace-pre-wrap">
                              {step.output_logs}
                            </pre>
                          </div>
                        </div>
                      )}
                    </div>
                  ))}
                </div>
              </CardContent>
            </Card>

            {/* Workflow Metadata */}
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <FileText className="w-4 h-4" />
                  Execution Metadata
                </CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="grid gap-2">
                  <label className="text-sm font-medium">Execution ID</label>
                  <div className="text-sm text-muted-foreground font-mono">{workflowDetail.id}</div>
                </div>
                <div className="grid gap-2">
                  <label className="text-sm font-medium">Workflow Template</label>
                  <Badge variant="secondary">{workflowDetail.workflow_name}</Badge>
                </div>
                <div className="grid gap-2">
                  <label className="text-sm font-medium">Application</label>
                  <Badge variant="outline">{workflowDetail.application_name}</Badge>
                </div>
                <div className="grid gap-2">
                  <label className="text-sm font-medium">Execution Status</label>
                  {getStatusBadge(workflowDetail.status)}
                </div>
                <div className="grid gap-2">
                  <label className="text-sm font-medium">Total Steps</label>
                  <div className="text-sm text-muted-foreground">{workflowDetail.total_steps}</div>
                </div>
                {workflowDetail.completed_at && (
                  <div className="grid gap-2">
                    <label className="text-sm font-medium">Completed At</label>
                    <div className="text-sm text-muted-foreground">{formatTimestamp(workflowDetail.completed_at)}</div>
                  </div>
                )}
                {workflowDetail.error_message && (
                  <div className="grid gap-2">
                    <label className="text-sm font-medium text-red-600">Error Message</label>
                    <div className="text-sm text-red-600">{workflowDetail.error_message}</div>
                  </div>
                )}
              </CardContent>
            </Card>
          </div>
        </div>
      </div>
    </ProtectedRoute>
  )
}