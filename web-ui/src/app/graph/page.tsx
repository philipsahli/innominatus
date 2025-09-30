"use client"

import { useState, useEffect, useRef, useCallback } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Badge } from "@/components/ui/badge"
import { Download, RefreshCw, FileText, Copy, Check } from "lucide-react"
import { useApplications, useResourceGraph } from "@/hooks/use-api"
import type { GraphNode, GraphData } from "@/lib/api"
import { api } from "@/lib/api"
import { ProtectedRoute } from "@/components/protected-route"


export default function ResourceGraphPage() {
  const [selectedApp, setSelectedApp] = useState<string>("")
  const [selectedNode, setSelectedNode] = useState<GraphNode | null>(null)
  const [showManifest, setShowManifest] = useState(false)
  const [appSpec, setAppSpec] = useState<any>(null)
  const [copiedManifest, setCopiedManifest] = useState(false)
  const networkRef = useRef<HTMLDivElement>(null)
  const networkInstance = useRef<any>(null)

  // API hooks
  const { data: applications, loading: appsLoading, error: appsError } = useApplications()
  const { data: graphData, loading: graphLoading, error: graphError, refetch: refetchGraph } = useResourceGraph(selectedApp || null)

  // Load vis.js dynamically
  useEffect(() => {
    const loadVisJs = async () => {
      if (typeof window !== 'undefined') {
        const script = document.createElement('script')
        script.src = 'https://unpkg.com/vis-network/standalone/umd/vis-network.min.js'
        script.async = true
        document.head.appendChild(script)
      }
    }
    loadVisJs()
  }, [])

  const displayData = graphData
  const appNames = applications?.map(app => app.name) || []

  // Fetch application spec when selectedApp changes
  useEffect(() => {
    const fetchAppSpec = async () => {
      if (selectedApp) {
        try {
          const response = await api.getSpecs()
          if (response.success && response.data) {
            setAppSpec(response.data[selectedApp] || null)
          } else {
            console.error('Error fetching app spec:', response.error)
            setAppSpec(null)
          }
        } catch (error) {
          console.error('Error fetching app spec:', error)
          setAppSpec(null)
        }
      } else {
        setAppSpec(null)
      }
    }
    fetchAppSpec()
  }, [selectedApp])

  const copyManifestToClipboard = async () => {
    if (appSpec) {
      try {
        await navigator.clipboard.writeText(JSON.stringify(appSpec, null, 2))
        setCopiedManifest(true)
        setTimeout(() => setCopiedManifest(false), 2000)
      } catch (error) {
        console.error('Failed to copy manifest:', error)
      }
    }
  }

  const renderGraph = useCallback((data: GraphData) => {
    if (!networkRef.current || !(window as any).vis) return

    const nodes = new (window as any).vis.DataSet(
      data.nodes.map(node => ({
        id: node.id,
        label: node.name,
        title: `${node.type}: ${node.name}\\nStatus: ${node.status}\\nDescription: ${node.description}`,
        color: {
          background: getNodeColor(node.type, node.status),
          border: getNodeBorderColor(node.status),
          highlight: {
            background: '#f0f9ff',
            border: '#0ea5e9'
          }
        },
        font: { color: '#374151', size: 12 },
        borderWidth: 2,
        shadow: true
      }))
    )

    const edges = new (window as any).vis.DataSet(
      data.edges.map(edge => ({
        id: edge.id,
        from: edge.source_id,
        to: edge.target_id,
        label: edge.relationship,
        arrows: 'to',
        color: '#6b7280',
        font: { color: '#6b7280', size: 10 }
      }))
    )

    const options = {
      layout: {
        randomSeed: 2,
        improvedLayout: true
      },
      physics: {
        stabilization: { iterations: 100 },
        barnesHut: {
          gravitationalConstant: -8000,
          springConstant: 0.001,
          springLength: 200
        }
      },
      interaction: {
        hover: true,
        selectConnectedEdges: false
      },
      nodes: {
        shape: 'dot',
        size: 25,
        borderWidth: 2,
        shadow: true
      },
      edges: {
        width: 2,
        shadow: true,
        smooth: {
          type: 'continuous',
          roundness: 0.5
        }
      }
    }

    if (networkInstance.current) {
      networkInstance.current.destroy()
    }

    networkInstance.current = new (window as any).vis.Network(
      networkRef.current,
      { nodes, edges },
      options
    )

    // Handle node selection
    networkInstance.current.on('selectNode', (params: any) => {
      if (params.nodes.length > 0) {
        const nodeId = params.nodes[0]
        const node = data.nodes.find(n => n.id === nodeId)
        if (node) {
          setSelectedNode(node)
        }
      }
    })

    networkInstance.current.on('deselectNode', () => {
      setSelectedNode(null)
    })
  }, [])

  function getNodeColor(type: string, status: string) {
    if (status === 'failed') return '#fef2f2'

    switch (type) {
      case 'app': return '#dbeafe'
      case 'postgres': return '#f0fdf4'
      case 'redis': return '#fef3c7'
      case 'service': return '#f3e8ff'
      default: return '#f9fafb'
    }
  }

  function getNodeBorderColor(status: string) {
    switch (status) {
      case 'running': return '#10b981'
      case 'failed': return '#ef4444'
      case 'pending': return '#f59e0b'
      default: return '#6b7280'
    }
  }

  const exportGraph = () => {
    if (!displayData) return

    const exportData = {
      name: displayData.app_name,
      graph: displayData,
      timestamp: displayData.timestamp
    }

    const blob = new Blob([JSON.stringify(exportData, null, 2)], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `${displayData.app_name}-graph.json`
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
  }

  useEffect(() => {
    if (displayData && (window as any).vis) {
      renderGraph(displayData)
    }
  }, [displayData, renderGraph])

  return (
    <ProtectedRoute>
      <div className="p-6 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Resource Graph</h1>
          <p className="text-muted-foreground">
            Visualize application dependencies and resource relationships
          </p>
        </div>
        <div className="flex gap-2">
          <Button
            variant="outline"
            onClick={() => refetchGraph()}
            disabled={!selectedApp || graphLoading}
          >
            <RefreshCw className={`w-4 h-4 mr-2 ${graphLoading ? 'animate-spin' : ''}`} />
            Refresh
          </Button>
          <Button variant="outline" onClick={exportGraph} disabled={!displayData}>
            <Download className="w-4 h-4 mr-2" />
            Export
          </Button>
        </div>
      </div>

      {/* Error Display */}
      {(appsError || graphError) && (
        <Card className="border-gray-200 bg-gray-50 dark:border-gray-700 dark:bg-gray-800">
          <CardContent className="pt-6">
            <p className="text-gray-800 dark:text-gray-200 text-sm">
              Using offline mode - graph data may not be current.
              {appsError && ` Apps: ${appsError}`}
              {graphError && ` Graph: ${graphError}`}
            </p>
          </CardContent>
        </Card>
      )}

      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle>Application Selector</CardTitle>
            <Select value={selectedApp} onValueChange={setSelectedApp} disabled={appsLoading}>
              <SelectTrigger className="w-[200px]">
                <SelectValue placeholder={appsLoading ? "Loading..." : "Select application"} />
              </SelectTrigger>
              <SelectContent>
                {appNames.map((app) => (
                  <SelectItem key={app} value={app}>
                    {app}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </CardHeader>
      </Card>

      <div className="grid gap-6 lg:grid-cols-4">
        {/* Graph Visualization */}
        <Card className="lg:col-span-3">
          <CardHeader>
            <CardTitle>
              {displayData ? `${displayData.app_name} Resource Graph` : 'Resource Graph'}
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div
              ref={networkRef}
              className="w-full h-[500px] border rounded-lg bg-white relative"
              style={{ minHeight: '500px' }}
            >
              {!selectedApp ? (
                <div className="flex items-center justify-center h-full text-muted-foreground">
                  Select an application to view its resource graph
                </div>
              ) : graphLoading ? (
                <div className="flex items-center justify-center h-full text-muted-foreground">
                  <RefreshCw className="w-6 h-6 animate-spin mr-2" />
                  Loading resource graph...
                </div>
              ) : null}
            </div>
          </CardContent>
        </Card>

        {/* Resource Details Panel */}
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle>Resource Details</CardTitle>
              {selectedApp && appSpec && (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setShowManifest(!showManifest)}
                  className="gap-2"
                >
                  <FileText className="w-4 h-4" />
                  {showManifest ? 'Hide' : 'Show'} Manifest
                </Button>
              )}
            </div>
          </CardHeader>
          <CardContent>
            {selectedNode ? (
              <div className="space-y-4">
                <div>
                  <h3 className="font-semibold text-lg">{selectedNode.name}</h3>
                  <p className="text-sm text-muted-foreground">{selectedNode.description}</p>
                </div>

                <div className="space-y-2">
                  <div className="flex justify-between items-center">
                    <span className="text-sm font-medium">Type:</span>
                    <Badge variant="outline">{selectedNode.type}</Badge>
                  </div>
                  <div className="flex justify-between items-center">
                    <span className="text-sm font-medium">Status:</span>
                    <Badge variant="outline">
                      {selectedNode.status}
                    </Badge>
                  </div>
                </div>

                {displayData && (
                  <div className="pt-4 border-t">
                    <h4 className="font-medium mb-2">Graph Info</h4>
                    <div className="text-sm space-y-1">
                      <p><strong>Application:</strong> {displayData.app_name}</p>
                      <p><strong>Total Nodes:</strong> {displayData.nodes.length}</p>
                      <p><strong>Total Edges:</strong> {displayData.edges.length}</p>
                      <p><strong>Last Updated:</strong> {new Date(displayData.timestamp).toLocaleString()}</p>
                    </div>
                  </div>
                )}

                {/* Manifest Section */}
                {showManifest && selectedApp && appSpec && (
                  <div className="pt-4 border-t">
                    <div className="flex items-center justify-between mb-2">
                      <h4 className="font-medium">Application Manifest</h4>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={copyManifestToClipboard}
                        className="gap-2"
                      >
                        {copiedManifest ? (
                          <>
                            <Check className="w-3 h-3" />
                            Copied
                          </>
                        ) : (
                          <>
                            <Copy className="w-3 h-3" />
                            Copy
                          </>
                        )}
                      </Button>
                    </div>
                    <div className="bg-gray-50 dark:bg-gray-800 rounded-lg p-3 max-h-64 overflow-y-auto">
                      <pre className="text-xs text-gray-800 dark:text-gray-200 whitespace-pre-wrap">
                        {JSON.stringify(appSpec, null, 2)}
                      </pre>
                    </div>
                  </div>
                )}
              </div>
            ) : (
              <div className="text-center text-muted-foreground py-8">
                {selectedApp ? 'Click on a node in the graph to see details here' : 'Select an application to view resource details and manifest'}
              </div>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Legend */}
      {displayData && (
        <Card>
          <CardHeader>
            <CardTitle>Legend</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex flex-wrap gap-4">
              <div className="flex items-center gap-2">
                <div className="w-4 h-4 bg-gray-100 border-2 border-gray-500 rounded-full"></div>
                <span className="text-sm">App</span>
              </div>
              <div className="flex items-center gap-2">
                <div className="w-4 h-4 bg-gray-200 border-2 border-gray-500 rounded-full"></div>
                <span className="text-sm">PostgreSQL</span>
              </div>
              <div className="flex items-center gap-2">
                <div className="w-4 h-4 bg-gray-300 border-2 border-gray-500 rounded-full"></div>
                <span className="text-sm">Redis</span>
              </div>
              <div className="flex items-center gap-2">
                <div className="w-4 h-4 bg-gray-400 border-2 border-gray-500 rounded-full"></div>
                <span className="text-sm">Service</span>
              </div>
              <div className="flex items-center gap-2">
                <div className="w-4 h-4 bg-gray-500 border-2 border-gray-600 rounded-full"></div>
                <span className="text-sm">Failed Status</span>
              </div>
            </div>
          </CardContent>
        </Card>
      )}
      </div>
    </ProtectedRoute>
  )
}