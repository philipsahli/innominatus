'use client'

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Label } from "@/components/ui/label"
import {
  AlertTriangle,
  Clock,
  Lightbulb,
  Upload,
  RefreshCw,
  CheckCircle,
  ArrowRight,
  Zap,
  FileText,
  Activity,
  GitBranch,
  Target,
  AlertCircle,
  TrendingUp,
  Database,
  Box
} from "lucide-react"
import { ProtectedRoute } from "@/components/protected-route"
import { useState, useRef } from "react"
import { api } from "@/lib/api"

interface WorkflowAnalysis {
  summary: {
    totalSteps: number
    totalResources: number
    parallelSteps: number
    criticalPath: string[]
    estimatedTime: string
    complexityScore: number
    riskLevel: string
  }
  executionPlan: {
    phases: Array<{
      name: string
      order: number
      parallelGroups: Array<{
        steps: Array<{
          name: string
          type: string
          estimatedTime: string
        }>
        estimatedTime: string
      }>
      estimatedTime: string
    }>
    totalTime: string
    maxParallel: number
  }
  resourceGraph: {
    nodes: Array<{
      id: string
      name: string
      type: string
      level: number
    }>
    edges: Array<{
      from: string
      to: string
      dependencyType: string
    }>
  }
  dependencies: Array<{
    stepName: string
    stepType: string
    dependsOn: string[]
    blocks: string[]
    canRunInParallel: boolean
    estimatedDuration: string
    phase: string
  }>
  warnings: string[]
  recommendations: string[]
}

export default function WorkflowAnalyzePage() {
  const [analysis, setAnalysis] = useState<WorkflowAnalysis | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [specContent, setSpecContent] = useState('')
  const fileInputRef = useRef<HTMLInputElement>(null)

  const analyzeWorkflow = async (yamlContent: string) => {
    setLoading(true)
    setError(null)

    try {
      const result = await api.analyzeWorkflow(yamlContent)

      if (result.success) {
        setAnalysis(result.data)
      } else {
        throw new Error(result.error || 'Failed to analyze workflow')
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to analyze workflow')
    } finally {
      setLoading(false)
    }
  }

  const handleFileUpload = (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0]
    if (file) {
      const reader = new FileReader()
      reader.onload = (e) => {
        const content = e.target?.result as string
        setSpecContent(content)
        analyzeWorkflow(content)
      }
      reader.readAsText(file)
    }
  }

  const handleAnalyze = () => {
    if (specContent.trim()) {
      analyzeWorkflow(specContent)
    }
  }

  const getRiskColor = (riskLevel: string) => {
    switch (riskLevel) {
      case 'low': return 'text-green-600 bg-green-100 dark:text-green-400 dark:bg-green-900/20'
      case 'medium': return 'text-yellow-600 bg-yellow-100 dark:text-yellow-400 dark:bg-yellow-900/20'
      case 'high': return 'text-red-600 bg-red-100 dark:text-red-400 dark:bg-red-900/20'
      default: return 'text-gray-600 bg-gray-100 dark:text-gray-400 dark:bg-gray-900/20'
    }
  }

  const getStepTypeIcon = (type: string) => {
    switch (type) {
      case 'terraform': return <Box className="w-4 h-4" />
      case 'kubernetes': return <Activity className="w-4 h-4" />
      case 'ansible': return <FileText className="w-4 h-4" />
      case 'validation': return <CheckCircle className="w-4 h-4" />
      case 'security': return <AlertTriangle className="w-4 h-4" />
      case 'monitoring': return <TrendingUp className="w-4 h-4" />
      case 'database': return <Database className="w-4 h-4" />
      default: return <Activity className="w-4 h-4" />
    }
  }

  return (
    <ProtectedRoute>
      <div className="min-h-screen bg-white dark:bg-gray-900 p-6">
        <div className="relative space-y-6 max-w-7xl mx-auto">
          {/* Header */}
          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <div className="p-2 rounded-lg bg-blue-100 dark:bg-blue-900/20">
                  <GitBranch className="w-6 h-6 text-blue-600 dark:text-blue-400" />
                </div>
                <div>
                  <h1 className="text-3xl font-bold tracking-tight text-gray-900 dark:text-gray-100">
                    Workflow Analysis
                  </h1>
                  <p className="text-gray-600 dark:text-gray-400">
                    Analyze Score specifications for workflow dependencies and execution planning
                  </p>
                </div>
              </div>
            </div>
          </div>

          {/* File Upload and Input */}
          <Card className="bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700">
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Upload className="w-5 h-5" />
                Score Specification Input
              </CardTitle>
              <CardDescription>
                Upload a Score specification file or paste the YAML content below
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="file-upload">Upload Score Spec File</Label>
                <div className="flex items-center gap-4">
                  <Button
                    variant="outline"
                    onClick={() => fileInputRef.current?.click()}
                    className="flex items-center gap-2"
                  >
                    <Upload className="w-4 h-4" />
                    Choose File
                  </Button>
                  <input
                    ref={fileInputRef}
                    type="file"
                    accept=".yaml,.yml"
                    onChange={handleFileUpload}
                    className="hidden"
                    id="file-upload"
                  />
                  <span className="text-sm text-gray-500">
                    Supports .yaml and .yml files
                  </span>
                </div>
              </div>

              <div className="space-y-2">
                <Label htmlFor="spec-content">Or paste YAML content</Label>
                <textarea
                  id="spec-content"
                  value={specContent}
                  onChange={(e) => setSpecContent(e.target.value)}
                  placeholder={`apiVersion: score.dev/v1b1
metadata:
  name: my-app
containers:
  web:
    image: nginx:latest
resources:
  postgres:
    type: postgres
    params:
      version: "13"
`}
                  className="w-full h-64 p-3 text-sm font-mono bg-gray-50 dark:bg-gray-900 border border-gray-300 dark:border-gray-600 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>

              <Button
                onClick={handleAnalyze}
                disabled={loading || !specContent.trim()}
                className="w-full"
              >
                {loading ? (
                  <>
                    <RefreshCw className="w-4 h-4 mr-2 animate-spin" />
                    Analyzing...
                  </>
                ) : (
                  <>
                    <GitBranch className="w-4 h-4 mr-2" />
                    Analyze Workflow
                  </>
                )}
              </Button>
            </CardContent>
          </Card>

          {/* Error Display */}
          {error && (
            <Card className="border-red-200 dark:border-red-800 bg-red-50 dark:bg-red-900/10">
              <CardContent className="pt-6">
                <div className="flex items-center gap-2 text-red-800 dark:text-red-200">
                  <AlertCircle className="w-4 h-4" />
                  <span className="font-medium">Analysis Failed</span>
                </div>
                <p className="mt-2 text-sm text-red-700 dark:text-red-300">{error}</p>
              </CardContent>
            </Card>
          )}

          {/* Analysis Results */}
          {analysis && (
            <div className="space-y-6">
              {/* Summary Cards */}
              <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
                <Card className="bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700">
                  <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                    <CardTitle className="text-sm font-medium">Complexity Score</CardTitle>
                    <Target className="h-4 w-4 text-muted-foreground" />
                  </CardHeader>
                  <CardContent>
                    <div className="text-2xl font-bold">{analysis.summary.complexityScore}</div>
                    <div className={`inline-flex items-center px-2 py-1 rounded-full text-xs font-medium ${getRiskColor(analysis.summary.riskLevel)}`}>
                      {analysis.summary.riskLevel} risk
                    </div>
                  </CardContent>
                </Card>

                <Card className="bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700">
                  <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                    <CardTitle className="text-sm font-medium">Estimated Time</CardTitle>
                    <Clock className="h-4 w-4 text-muted-foreground" />
                  </CardHeader>
                  <CardContent>
                    <div className="text-2xl font-bold">{analysis.summary.estimatedTime}</div>
                    <p className="text-xs text-muted-foreground">Total execution time</p>
                  </CardContent>
                </Card>

                <Card className="bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700">
                  <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                    <CardTitle className="text-sm font-medium">Total Steps</CardTitle>
                    <Activity className="h-4 w-4 text-muted-foreground" />
                  </CardHeader>
                  <CardContent>
                    <div className="text-2xl font-bold">{analysis.summary.totalSteps}</div>
                    <p className="text-xs text-muted-foreground">
                      {analysis.summary.parallelSteps} can run in parallel
                    </p>
                  </CardContent>
                </Card>

                <Card className="bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700">
                  <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                    <CardTitle className="text-sm font-medium">Resources</CardTitle>
                    <Box className="h-4 w-4 text-muted-foreground" />
                  </CardHeader>
                  <CardContent>
                    <div className="text-2xl font-bold">{analysis.summary.totalResources}</div>
                    <p className="text-xs text-muted-foreground">Dependencies to provision</p>
                  </CardContent>
                </Card>
              </div>

              {/* Execution Plan */}
              <Card className="bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700">
                <CardHeader>
                  <CardTitle className="flex items-center gap-2">
                    <Zap className="w-5 h-5" />
                    Execution Plan
                  </CardTitle>
                  <CardDescription>
                    Optimized workflow execution phases with parallelization opportunities
                  </CardDescription>
                </CardHeader>
                <CardContent>
                  <div className="space-y-6">
                    {analysis.executionPlan.phases.map((phase, phaseIndex) => (
                      <div key={phaseIndex} className="space-y-3">
                        <div className="flex items-center gap-3">
                          <div className="flex items-center justify-center w-8 h-8 rounded-full bg-blue-100 dark:bg-blue-900/20 text-blue-600 dark:text-blue-400 text-sm font-medium">
                            {phase.order}
                          </div>
                          <div>
                            <h3 className="font-medium capitalize">{phase.name.replace('-', ' ')}</h3>
                            <p className="text-sm text-muted-foreground">
                              Estimated time: {phase.estimatedTime}
                            </p>
                          </div>
                        </div>

                        <div className="ml-4 pl-4 border-l-2 border-gray-200 dark:border-gray-700 space-y-4">
                          {phase.parallelGroups.map((group, groupIndex) => (
                            <div key={groupIndex} className="space-y-2">
                              {group.steps.length > 1 && (
                                <div className="flex items-center gap-2 text-sm text-muted-foreground">
                                  <Zap className="w-3 h-3" />
                                  <span>Parallel group ({group.estimatedTime})</span>
                                </div>
                              )}
                              <div className={`space-y-2 ${group.steps.length > 1 ? 'ml-4' : ''}`}>
                                {group.steps.map((step, stepIndex) => (
                                  <div
                                    key={stepIndex}
                                    className="flex items-center gap-3 p-3 rounded-lg bg-gray-50 dark:bg-gray-800/50"
                                  >
                                    <div className="flex items-center justify-center w-6 h-6 rounded bg-white dark:bg-gray-700">
                                      {getStepTypeIcon(step.type)}
                                    </div>
                                    <div className="flex-1">
                                      <div className="font-medium text-sm">{step.name}</div>
                                      <div className="text-xs text-muted-foreground">
                                        {step.type} • {step.estimatedTime}
                                      </div>
                                    </div>
                                  </div>
                                ))}
                              </div>
                            </div>
                          ))}
                        </div>

                        {phaseIndex < analysis.executionPlan.phases.length - 1 && (
                          <div className="flex justify-center py-2">
                            <ArrowRight className="w-4 h-4 text-muted-foreground" />
                          </div>
                        )}
                      </div>
                    ))}
                  </div>
                </CardContent>
              </Card>

              {/* Critical Path */}
              {analysis.summary.criticalPath.length > 0 && (
                <Card className="bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700">
                  <CardHeader>
                    <CardTitle className="flex items-center gap-2">
                      <Target className="w-5 h-5" />
                      Critical Path
                    </CardTitle>
                    <CardDescription>
                      The longest sequence of dependent steps that determines minimum execution time
                    </CardDescription>
                  </CardHeader>
                  <CardContent>
                    <div className="flex flex-wrap items-center gap-2">
                      {analysis.summary.criticalPath.map((step, index) => (
                        <div key={index} className="flex items-center gap-2">
                          <Badge variant="outline" className="text-xs">
                            {step}
                          </Badge>
                          {index < analysis.summary.criticalPath.length - 1 && (
                            <ArrowRight className="w-3 h-3 text-muted-foreground" />
                          )}
                        </div>
                      ))}
                    </div>
                  </CardContent>
                </Card>
              )}

              {/* Warnings and Recommendations */}
              <div className="grid gap-4 md:grid-cols-2">
                {analysis.warnings.length > 0 && (
                  <Card className="bg-yellow-50 dark:bg-yellow-900/10 border-yellow-200 dark:border-yellow-800">
                    <CardHeader>
                      <CardTitle className="flex items-center gap-2 text-yellow-800 dark:text-yellow-200">
                        <AlertTriangle className="w-5 h-5" />
                        Warnings
                      </CardTitle>
                    </CardHeader>
                    <CardContent>
                      <ul className="space-y-2">
                        {analysis.warnings.map((warning, index) => (
                          <li key={index} className="flex items-start gap-2 text-sm">
                            <AlertTriangle className="w-4 h-4 text-yellow-600 dark:text-yellow-400 mt-0.5 flex-shrink-0" />
                            <span className="text-yellow-800 dark:text-yellow-200">{warning}</span>
                          </li>
                        ))}
                      </ul>
                    </CardContent>
                  </Card>
                )}

                {analysis.recommendations.length > 0 && (
                  <Card className="bg-blue-50 dark:bg-blue-900/10 border-blue-200 dark:border-blue-800">
                    <CardHeader>
                      <CardTitle className="flex items-center gap-2 text-blue-800 dark:text-blue-200">
                        <Lightbulb className="w-5 h-5" />
                        Recommendations
                      </CardTitle>
                    </CardHeader>
                    <CardContent>
                      <ul className="space-y-2">
                        {analysis.recommendations.map((recommendation, index) => (
                          <li key={index} className="flex items-start gap-2 text-sm">
                            <Lightbulb className="w-4 h-4 text-blue-600 dark:text-blue-400 mt-0.5 flex-shrink-0" />
                            <span className="text-blue-800 dark:text-blue-200">{recommendation}</span>
                          </li>
                        ))}
                      </ul>
                    </CardContent>
                  </Card>
                )}
              </div>

              {/* Resource Dependencies */}
              {analysis.resourceGraph.nodes.length > 0 && (
                <Card className="bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700">
                  <CardHeader>
                    <CardTitle className="flex items-center gap-2">
                      <GitBranch className="w-5 h-5" />
                      Resource Dependencies
                    </CardTitle>
                    <CardDescription>
                      Visual representation of resource dependencies and levels
                    </CardDescription>
                  </CardHeader>
                  <CardContent>
                    <div className="space-y-4">
                      {Array.from(new Set(analysis.resourceGraph.nodes.map(n => n.level)))
                        .sort((a, b) => a - b)
                        .map(level => (
                          <div key={level} className="space-y-2">
                            <h4 className="text-sm font-medium text-muted-foreground">
                              Level {level}
                            </h4>
                            <div className="flex flex-wrap gap-2">
                              {analysis.resourceGraph.nodes
                                .filter(node => node.level === level)
                                .map(node => (
                                  <Badge
                                    key={node.id}
                                    variant="outline"
                                    className="text-xs"
                                  >
                                    {node.name} ({node.type})
                                  </Badge>
                                ))}
                            </div>
                          </div>
                        ))}
                    </div>
                  </CardContent>
                </Card>
              )}
            </div>
          )}
        </div>
      </div>
    </ProtectedRoute>
  )
}