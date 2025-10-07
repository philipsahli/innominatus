'use client';

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import {
  ArrowLeft,
  Clock,
  Play,
  FileText,
  Tag,
  Info,
  ExternalLink,
  Github,
  Code,
  Settings,
  Globe,
  GitGraph,
} from 'lucide-react';
import { useRouter } from 'next/navigation';
import {
  getCategoryColor,
  getIconColor,
  getParameterTypeBadgeColor,
  type GoldenPath,
} from '@/lib/goldenpaths';
import { WorkflowDiagram } from '@/components/workflow-diagram';
import { WorkflowStep } from '@/lib/workflow-types';

interface GoldenPathDetailClientProps {
  goldenPath: GoldenPath;
  workflowUrl: string;
  configUrl: string;
  workflowSteps: WorkflowStep[];
}

export function GoldenPathDetailClient({
  goldenPath,
  workflowUrl,
  configUrl,
  workflowSteps,
}: GoldenPathDetailClientProps) {
  const router = useRouter();

  return (
    <div className="min-h-screen bg-white dark:bg-gray-900 p-6">
      <div className="max-w-6xl mx-auto space-y-6">
        {/* Back Navigation */}
        <Button variant="ghost" onClick={() => router.push('/goldenpaths')} className="mb-4">
          <ArrowLeft className="w-4 h-4 mr-2" />
          Back to Golden Paths
        </Button>

        {/* Header */}
        <div className="space-y-4">
          <div className="flex items-start justify-between">
            <div className="flex-1">
              <div className="flex items-center gap-3 mb-2">
                <FileText className={`w-8 h-8 ${getIconColor(goldenPath.category)}`} />
                <h1 className="text-4xl font-bold tracking-tight text-gray-900 dark:text-gray-100">
                  {goldenPath.name}
                </h1>
                <Badge className={`${getCategoryColor(goldenPath.category)} text-sm`}>
                  {goldenPath.category}
                </Badge>
              </div>
              <p className="text-lg text-gray-600 dark:text-gray-400 mt-2">
                {goldenPath.description}
              </p>
            </div>
          </div>
        </div>

        {/* Quick Info Cards */}
        <div className="grid gap-4 md:grid-cols-3">
          <Card className="bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700">
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Duration</CardTitle>
              <Clock className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{goldenPath.estimated_duration}</div>
            </CardContent>
          </Card>

          <Card className="bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700">
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Category</CardTitle>
              <Tag className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold capitalize">{goldenPath.category}</div>
            </CardContent>
          </Card>

          <Card className="bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700">
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Parameters</CardTitle>
              <Settings className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">
                {goldenPath.parameters ? Object.keys(goldenPath.parameters).length : 0}
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Tags */}
        <Card className="bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700">
          <CardHeader>
            <CardTitle className="text-lg flex items-center gap-2">
              <Tag className="w-4 h-4" />
              Tags
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex flex-wrap gap-2">
              {goldenPath.tags.map((tag) => (
                <Badge key={tag} variant="secondary" className="text-sm">
                  {tag}
                </Badge>
              ))}
            </div>
          </CardContent>
        </Card>

        {/* Workflow Visualization */}
        {workflowSteps.length > 0 && (
          <Card className="bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700">
            <CardHeader>
              <CardTitle className="text-lg flex items-center gap-2">
                <GitGraph className="w-4 h-4" />
                Workflow Visualization
              </CardTitle>
              <CardDescription>Step-by-step execution flow for this golden path</CardDescription>
            </CardHeader>
            <CardContent>
              <WorkflowDiagram steps={workflowSteps} />
              <div className="mt-4 p-3 bg-blue-50 dark:bg-blue-900/20 rounded-lg border border-blue-200 dark:border-blue-800">
                <p className="text-xs text-blue-900 dark:text-blue-100">
                  <strong>Legend:</strong> Solid borders indicate required steps. Dashed borders
                  indicate conditional steps that execute based on the &quot;if&quot; condition.
                </p>
              </div>
            </CardContent>
          </Card>
        )}

        {/* GitHub Source Links */}
        <Card className="bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700">
          <CardHeader>
            <CardTitle className="text-lg flex items-center gap-2">
              <Github className="w-4 h-4" />
              Source Code
            </CardTitle>
            <CardDescription>
              View the source code and configuration for this golden path
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            <a
              href={workflowUrl}
              target="_blank"
              rel="noopener noreferrer"
              className="flex items-center justify-between p-3 rounded-lg border border-gray-200 dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors"
            >
              <div className="flex items-center gap-3">
                <Code className="w-5 h-5 text-blue-500" />
                <div>
                  <div className="font-medium text-gray-900 dark:text-gray-100">
                    Workflow Definition
                  </div>
                  <div className="text-sm text-gray-500 dark:text-gray-400">
                    {goldenPath.workflow}
                  </div>
                </div>
              </div>
              <ExternalLink className="w-4 h-4 text-gray-400" />
            </a>

            <a
              href={configUrl}
              target="_blank"
              rel="noopener noreferrer"
              className="flex items-center justify-between p-3 rounded-lg border border-gray-200 dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors"
            >
              <div className="flex items-center gap-3">
                <FileText className="w-5 h-5 text-green-500" />
                <div>
                  <div className="font-medium text-gray-900 dark:text-gray-100">
                    Golden Path Configuration
                  </div>
                  <div className="text-sm text-gray-500 dark:text-gray-400">goldenpaths.yaml</div>
                </div>
              </div>
              <ExternalLink className="w-4 h-4 text-gray-400" />
            </a>
          </CardContent>
        </Card>

        {/* Parameters */}
        {goldenPath.parameters && Object.keys(goldenPath.parameters).length > 0 && (
          <Card className="bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700">
            <CardHeader>
              <CardTitle className="text-lg flex items-center gap-2">
                <Settings className="w-4 h-4" />
                Parameters
              </CardTitle>
              <CardDescription>Configurable parameters for this golden path</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                {Object.entries(goldenPath.parameters).map(([paramName, schema]) => (
                  <div
                    key={paramName}
                    className="p-4 rounded-lg border border-gray-200 dark:border-gray-700"
                  >
                    <div className="flex items-start justify-between mb-2">
                      <div className="flex items-center gap-2">
                        <code className="text-sm font-mono font-semibold text-gray-900 dark:text-gray-100">
                          {paramName}
                        </code>
                        <Badge className={getParameterTypeBadgeColor(schema.type)}>
                          {schema.type}
                        </Badge>
                        {schema.required ? (
                          <Badge className="bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-300">
                            Required
                          </Badge>
                        ) : (
                          <Badge variant="outline">Optional</Badge>
                        )}
                      </div>
                    </div>

                    {schema.description && (
                      <p className="text-sm text-gray-600 dark:text-gray-400 mb-2">
                        {schema.description}
                      </p>
                    )}

                    <div className="grid grid-cols-2 gap-3 text-sm mt-3">
                      {schema.default !== undefined && (
                        <div>
                          <span className="text-gray-500 dark:text-gray-400">Default:</span>{' '}
                          <code className="text-xs bg-gray-100 dark:bg-gray-700 px-1 py-0.5 rounded">
                            {schema.default || '""'}
                          </code>
                        </div>
                      )}

                      {schema.allowed_values && (
                        <div className="col-span-2">
                          <span className="text-gray-500 dark:text-gray-400">Allowed values:</span>{' '}
                          <div className="flex flex-wrap gap-1 mt-1">
                            {schema.allowed_values.map((value) => (
                              <code
                                key={value}
                                className="text-xs bg-gray-100 dark:bg-gray-700 px-2 py-1 rounded"
                              >
                                {value}
                              </code>
                            ))}
                          </div>
                        </div>
                      )}

                      {schema.min !== undefined && (
                        <div>
                          <span className="text-gray-500 dark:text-gray-400">Min:</span>{' '}
                          <code className="text-xs bg-gray-100 dark:bg-gray-700 px-1 py-0.5 rounded">
                            {schema.min}
                          </code>
                        </div>
                      )}

                      {schema.max !== undefined && (
                        <div>
                          <span className="text-gray-500 dark:text-gray-400">Max:</span>{' '}
                          <code className="text-xs bg-gray-100 dark:bg-gray-700 px-1 py-0.5 rounded">
                            {schema.max}
                          </code>
                        </div>
                      )}

                      {schema.pattern && (
                        <div className="col-span-2">
                          <span className="text-gray-500 dark:text-gray-400">Pattern:</span>{' '}
                          <code className="text-xs bg-gray-100 dark:bg-gray-700 px-1 py-0.5 rounded">
                            {schema.pattern}
                          </code>
                        </div>
                      )}
                    </div>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>
        )}

        {/* Usage Examples */}
        <Card className="bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700">
          <CardHeader>
            <CardTitle className="text-lg flex items-center gap-2">
              <Code className="w-4 h-4" />
              Usage Examples
            </CardTitle>
            <CardDescription>How to run this golden path from the CLI</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div>
              <p className="text-sm text-gray-600 dark:text-gray-400 mb-2">Basic usage:</p>
              <pre className="bg-gray-100 dark:bg-gray-900 p-4 rounded-lg overflow-x-auto">
                <code className="text-sm font-mono">
                  ./innominatus-ctl run {goldenPath.name} score-spec.yaml
                </code>
              </pre>
            </div>

            {goldenPath.parameters && Object.keys(goldenPath.parameters).length > 0 && (
              <div>
                <p className="text-sm text-gray-600 dark:text-gray-400 mb-2">
                  With custom parameters:
                </p>
                <pre className="bg-gray-100 dark:bg-gray-900 p-4 rounded-lg overflow-x-auto">
                  <code className="text-sm font-mono">
                    ./innominatus-ctl run {goldenPath.name} score-spec.yaml \{'\n'}
                    {'  '}--param {Object.keys(goldenPath.parameters)[0]}=custom-value
                    {Object.keys(goldenPath.parameters).length > 1 && (
                      <>
                        {' \\\n'}
                        {'  '}--param {Object.keys(goldenPath.parameters)[1]}=another-value
                      </>
                    )}
                  </code>
                </pre>
              </div>
            )}
          </CardContent>
        </Card>

        {/* API Usage */}
        <Card className="bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700">
          <CardHeader>
            <CardTitle className="text-lg flex items-center gap-2">
              <Globe className="w-4 h-4" />
              API Usage
            </CardTitle>
            <CardDescription>Execute this golden path via REST API</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div>
              <p className="text-sm text-gray-600 dark:text-gray-400 mb-2">Endpoint:</p>
              <pre className="bg-gray-100 dark:bg-gray-900 p-4 rounded-lg overflow-x-auto">
                <code className="text-sm font-mono">
                  POST /api/workflows/golden-paths/{goldenPath.name}/execute
                </code>
              </pre>
            </div>

            <div>
              <p className="text-sm text-gray-600 dark:text-gray-400 mb-2">Example using curl:</p>
              <pre className="bg-gray-100 dark:bg-gray-900 p-4 rounded-lg overflow-x-auto">
                <code className="text-sm font-mono">
                  curl -X POST http://localhost:8081/api/workflows/golden-paths/{goldenPath.name}
                  /execute \{'\n'}
                  {'  '}-H &quot;Content-Type: application/yaml&quot; \{'\n'}
                  {'  '}-H &quot;Authorization: Bearer $API_TOKEN&quot; \{'\n'}
                  {'  '}--data-binary @score-spec.yaml
                </code>
              </pre>
            </div>

            {goldenPath.parameters && Object.keys(goldenPath.parameters).length > 0 && (
              <div className="mt-3 p-3 bg-blue-50 dark:bg-blue-900/20 rounded-lg border border-blue-200 dark:border-blue-800">
                <p className="text-xs text-blue-900 dark:text-blue-100">
                  <strong>Note:</strong> Parameters can be passed via query string or embedded in
                  the Score spec metadata.
                </p>
              </div>
            )}
          </CardContent>
        </Card>

        {/* Action Buttons */}
        <div className="flex gap-3">
          <Button size="lg" disabled className="flex-1">
            <Play className="w-4 h-4 mr-2" />
            Run Golden Path
          </Button>
          <Button size="lg" variant="outline" onClick={() => window.open(workflowUrl, '_blank')}>
            <Github className="w-4 h-4 mr-2" />
            View in GitHub
          </Button>
        </div>

        {/* Info Footer */}
        <Card className="bg-blue-50 dark:bg-blue-900/20 border-blue-200 dark:border-blue-800">
          <CardContent className="pt-6">
            <div className="flex items-start gap-3">
              <Info className="w-5 h-5 text-blue-600 dark:text-blue-400 mt-0.5" />
              <div className="text-sm text-blue-900 dark:text-blue-100">
                <p className="font-medium mb-1">About Golden Paths</p>
                <p className="text-blue-800 dark:text-blue-200">
                  Golden Paths are pre-defined, curated workflows that represent the recommended way
                  to accomplish common platform operations. They encapsulate best practices,
                  standardized configurations, and proven patterns.
                </p>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
