'use client';

import React, { useEffect, useState } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Clock, GitCompare, X } from 'lucide-react';

interface HistorySnapshot {
  id: number;
  workflow_name: string;
  status: string;
  started_at: string;
  completed_at?: string;
  duration_seconds?: number;
  total_steps: number;
  completed_steps: number;
  failed_steps: number;
}

interface HistoryResponse {
  application: string;
  snapshots: HistorySnapshot[];
  count: number;
}

interface GraphDiffProps {
  app: string;
  onClose: () => void;
}

export function GraphDiff({ app, onClose }: GraphDiffProps) {
  const [history, setHistory] = useState<HistorySnapshot[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selectedSnapshot1, setSelectedSnapshot1] = useState<number | null>(null);
  const [selectedSnapshot2, setSelectedSnapshot2] = useState<number | null>(null);
  const [showComparison, setShowComparison] = useState(false);

  useEffect(() => {
    fetchHistory();
  }, [app]);

  const fetchHistory = async () => {
    setLoading(true);
    setError(null);

    try {
      const apiKey = localStorage.getItem('api_key') || '';
      const response = await fetch(`/api/graph/${app}/history?limit=20`, {
        headers: {
          Authorization: `Bearer ${apiKey}`,
        },
      });

      if (!response.ok) {
        throw new Error(`Failed to fetch history: ${response.statusText}`);
      }

      const data: HistoryResponse = await response.json();
      setHistory(data.snapshots);

      // Auto-select first two snapshots for quick comparison
      if (data.snapshots.length >= 2) {
        setSelectedSnapshot1(data.snapshots[0].id);
        setSelectedSnapshot2(data.snapshots[1].id);
      } else if (data.snapshots.length === 1) {
        setSelectedSnapshot1(data.snapshots[0].id);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
    } finally {
      setLoading(false);
    }
  };

  const formatDate = (dateStr: string) => {
    const date = new Date(dateStr);
    return date.toLocaleString();
  };

  const formatDuration = (seconds?: number) => {
    if (!seconds) return 'N/A';
    if (seconds < 1) return `${(seconds * 1000).toFixed(0)}ms`;
    if (seconds < 60) return `${seconds.toFixed(2)}s`;
    const minutes = Math.floor(seconds / 60);
    const remainingSeconds = (seconds % 60).toFixed(0);
    return `${minutes}m ${remainingSeconds}s`;
  };

  const getStatusBadgeColor = (status: string) => {
    switch (status) {
      case 'succeeded':
      case 'completed':
        return 'bg-green-500';
      case 'failed':
        return 'bg-red-500';
      case 'running':
        return 'bg-yellow-500';
      default:
        return 'bg-gray-500';
    }
  };

  const compareSnapshots = () => {
    if (!selectedSnapshot1 || !selectedSnapshot2) {
      return;
    }
    setShowComparison(true);
  };

  const snap1 = history.find((s) => s.id === selectedSnapshot1);
  const snap2 = history.find((s) => s.id === selectedSnapshot2);

  const getDurationDiff = () => {
    if (!snap1?.duration_seconds || !snap2?.duration_seconds) return null;
    return snap1.duration_seconds - snap2.duration_seconds;
  };

  const getPerformanceIndicator = (diff: number | null) => {
    if (diff === null) return null;
    if (diff < 0) return { text: 'faster', color: 'text-green-600', icon: '↓' };
    if (diff > 0) return { text: 'slower', color: 'text-red-600', icon: '↑' };
    return { text: 'same', color: 'text-gray-600', icon: '=' };
  };

  if (loading) {
    return (
      <Card className="w-full">
        <CardHeader>
          <CardTitle className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Clock className="h-5 w-5" />
              Workflow History
            </div>
            <Button variant="ghost" size="sm" onClick={onClose}>
              <X className="h-4 w-4" />
            </Button>
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="text-center py-8 text-gray-500">Loading history...</div>
        </CardContent>
      </Card>
    );
  }

  if (error) {
    return (
      <Card className="w-full">
        <CardHeader>
          <CardTitle className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Clock className="h-5 w-5" />
              Workflow History
            </div>
            <Button variant="ghost" size="sm" onClick={onClose}>
              <X className="h-4 w-4" />
            </Button>
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="text-center py-8 text-red-500">Error: {error}</div>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card className="w-full">
      <CardHeader>
        <CardTitle className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Clock className="h-5 w-5" />
            Workflow History - {app}
          </div>
          <Button variant="ghost" size="sm" onClick={onClose}>
            <X className="h-4 w-4" />
          </Button>
        </CardTitle>
      </CardHeader>
      <CardContent>
        <div className="space-y-4">
          {/* Comparison Controls */}
          <div className="flex items-center gap-4 p-4 bg-gray-50 rounded-lg">
            <GitCompare className="h-5 w-5 text-gray-600" />
            <span className="text-sm text-gray-600">Select two snapshots to compare:</span>
            <Button
              onClick={compareSnapshots}
              disabled={!selectedSnapshot1 || !selectedSnapshot2}
              size="sm"
            >
              Compare
            </Button>
          </div>

          {/* History List */}
          <div className="space-y-2 max-h-96 overflow-y-auto">
            {history.length === 0 ? (
              <div className="text-center py-8 text-gray-500">No history available</div>
            ) : (
              history.map((snapshot, index) => (
                <div
                  key={snapshot.id}
                  className={`p-4 border rounded-lg cursor-pointer transition-colors ${
                    selectedSnapshot1 === snapshot.id || selectedSnapshot2 === snapshot.id
                      ? 'border-blue-500 bg-blue-50'
                      : 'border-gray-200 hover:border-gray-300'
                  }`}
                  onClick={() => {
                    if (selectedSnapshot1 === snapshot.id) {
                      setSelectedSnapshot1(null);
                    } else if (selectedSnapshot2 === snapshot.id) {
                      setSelectedSnapshot2(null);
                    } else if (!selectedSnapshot1) {
                      setSelectedSnapshot1(snapshot.id);
                    } else if (!selectedSnapshot2) {
                      setSelectedSnapshot2(snapshot.id);
                    } else {
                      // Replace the second selection
                      setSelectedSnapshot2(snapshot.id);
                    }
                  }}
                >
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-3">
                      <div className="flex items-center gap-2">
                        <input
                          type="checkbox"
                          checked={
                            selectedSnapshot1 === snapshot.id || selectedSnapshot2 === snapshot.id
                          }
                          readOnly
                          className="h-4 w-4"
                        />
                        <Badge className={getStatusBadgeColor(snapshot.status)}>
                          {snapshot.status}
                        </Badge>
                      </div>
                      <div>
                        <div className="font-medium text-sm">{snapshot.workflow_name}</div>
                        <div className="text-xs text-gray-500">
                          ID: {snapshot.id} • {formatDate(snapshot.started_at)}
                        </div>
                      </div>
                    </div>
                    <div className="text-right text-sm">
                      <div className="text-gray-600">
                        {formatDuration(snapshot.duration_seconds)}
                      </div>
                      <div className="text-xs text-gray-500">
                        {snapshot.completed_steps}/{snapshot.total_steps} steps
                        {snapshot.failed_steps > 0 && (
                          <span className="text-red-500"> • {snapshot.failed_steps} failed</span>
                        )}
                      </div>
                    </div>
                  </div>
                </div>
              ))
            )}
          </div>

          {/* Comparison View */}
          {showComparison && snap1 && snap2 && (
            <div className="mt-6 p-6 border-2 border-blue-500 rounded-lg bg-blue-50">
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-lg font-semibold flex items-center gap-2">
                  <GitCompare className="h-5 w-5" />
                  Comparison Results
                </h3>
                <Button variant="ghost" size="sm" onClick={() => setShowComparison(false)}>
                  <X className="h-4 w-4" />
                </Button>
              </div>

              {/* Side-by-side comparison */}
              <div className="grid grid-cols-2 gap-6 mb-6">
                {/* Snapshot 1 */}
                <div className="bg-white p-4 rounded-lg border">
                  <div className="text-sm font-medium text-gray-500 mb-2">Snapshot #{snap1.id}</div>
                  <div className="space-y-2 text-sm">
                    <div>
                      <span className="text-gray-600">Workflow:</span>{' '}
                      <span className="font-medium">{snap1.workflow_name}</span>
                    </div>
                    <div>
                      <span className="text-gray-600">Status:</span>{' '}
                      <Badge className={getStatusBadgeColor(snap1.status)}>{snap1.status}</Badge>
                    </div>
                    <div>
                      <span className="text-gray-600">Started:</span>{' '}
                      <span className="font-medium">{formatDate(snap1.started_at)}</span>
                    </div>
                    <div>
                      <span className="text-gray-600">Duration:</span>{' '}
                      <span className="font-medium">{formatDuration(snap1.duration_seconds)}</span>
                    </div>
                    <div>
                      <span className="text-gray-600">Steps:</span>{' '}
                      <span className="font-medium">
                        {snap1.completed_steps}/{snap1.total_steps}
                        {snap1.failed_steps > 0 && (
                          <span className="text-red-600"> ({snap1.failed_steps} failed)</span>
                        )}
                      </span>
                    </div>
                  </div>
                </div>

                {/* Snapshot 2 */}
                <div className="bg-white p-4 rounded-lg border">
                  <div className="text-sm font-medium text-gray-500 mb-2">Snapshot #{snap2.id}</div>
                  <div className="space-y-2 text-sm">
                    <div>
                      <span className="text-gray-600">Workflow:</span>{' '}
                      <span className="font-medium">{snap2.workflow_name}</span>
                    </div>
                    <div>
                      <span className="text-gray-600">Status:</span>{' '}
                      <Badge className={getStatusBadgeColor(snap2.status)}>{snap2.status}</Badge>
                    </div>
                    <div>
                      <span className="text-gray-600">Started:</span>{' '}
                      <span className="font-medium">{formatDate(snap2.started_at)}</span>
                    </div>
                    <div>
                      <span className="text-gray-600">Duration:</span>{' '}
                      <span className="font-medium">{formatDuration(snap2.duration_seconds)}</span>
                    </div>
                    <div>
                      <span className="text-gray-600">Steps:</span>{' '}
                      <span className="font-medium">
                        {snap2.completed_steps}/{snap2.total_steps}
                        {snap2.failed_steps > 0 && (
                          <span className="text-red-600"> ({snap2.failed_steps} failed)</span>
                        )}
                      </span>
                    </div>
                  </div>
                </div>
              </div>

              {/* Differences */}
              <div className="bg-white p-4 rounded-lg border">
                <h4 className="font-semibold mb-3">Key Differences</h4>
                <div className="space-y-2 text-sm">
                  {/* Duration Difference */}
                  {(() => {
                    const durationDiff = getDurationDiff();
                    const indicator = getPerformanceIndicator(durationDiff);
                    return durationDiff !== null && indicator ? (
                      <div className="flex items-center justify-between p-2 bg-gray-50 rounded">
                        <span className="text-gray-600">Duration:</span>
                        <span className={`font-medium ${indicator.color}`}>
                          {indicator.icon} {Math.abs(durationDiff).toFixed(2)}s {indicator.text}
                        </span>
                      </div>
                    ) : null;
                  })()}

                  {/* Status Comparison */}
                  {snap1.status !== snap2.status && (
                    <div className="flex items-center justify-between p-2 bg-gray-50 rounded">
                      <span className="text-gray-600">Status Change:</span>
                      <span className="font-medium">
                        {snap1.status} → {snap2.status}
                      </span>
                    </div>
                  )}

                  {/* Steps Comparison */}
                  {(() => {
                    const completedDiff = snap1.completed_steps - snap2.completed_steps;
                    const failedDiff = snap1.failed_steps - snap2.failed_steps;
                    return (
                      <>
                        {completedDiff !== 0 && (
                          <div className="flex items-center justify-between p-2 bg-gray-50 rounded">
                            <span className="text-gray-600">Completed Steps:</span>
                            <span
                              className={`font-medium ${completedDiff > 0 ? 'text-green-600' : 'text-red-600'}`}
                            >
                              {completedDiff > 0 ? '+' : ''}
                              {completedDiff}
                            </span>
                          </div>
                        )}
                        {failedDiff !== 0 && (
                          <div className="flex items-center justify-between p-2 bg-gray-50 rounded">
                            <span className="text-gray-600">Failed Steps:</span>
                            <span
                              className={`font-medium ${failedDiff > 0 ? 'text-red-600' : 'text-green-600'}`}
                            >
                              {failedDiff > 0 ? '+' : ''}
                              {failedDiff}
                            </span>
                          </div>
                        )}
                      </>
                    );
                  })()}

                  {/* Performance Summary */}
                  <div className="mt-4 p-3 bg-blue-100 rounded border border-blue-200">
                    <div className="text-xs font-medium text-blue-800 mb-1">Summary</div>
                    <div className="text-sm text-blue-700">
                      {(() => {
                        const durationDiff = getDurationDiff();
                        if (durationDiff === null) return 'Performance data not available';
                        if (durationDiff < -1)
                          return '🎉 Significant performance improvement detected!';
                        if (durationDiff > 1) return '⚠️ Performance regression detected';
                        return '✓ Performance is stable';
                      })()}
                    </div>
                  </div>
                </div>
              </div>
            </div>
          )}
        </div>
      </CardContent>
    </Card>
  );
}
