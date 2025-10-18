'use client';

import React, { useEffect, useState } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Activity, TrendingUp, TrendingDown, Clock, CheckCircle, XCircle } from 'lucide-react';

interface StepMetric {
  step_name: string;
  step_type: string;
  execution_count: number;
  success_count: number;
  failure_count: number;
  success_rate_percent: number;
  average_duration_seconds: number;
  max_duration_seconds: number;
}

interface TimeSeriesPoint {
  timestamp: string;
  duration_seconds: number;
  status: string;
}

interface PerformanceMetrics {
  application: string;
  total_executions: number;
  success_rate_percent: number;
  failure_rate_percent: number;
  average_duration_seconds: number;
  median_duration_seconds: number;
  min_duration_seconds: number;
  max_duration_seconds: number;
  step_metrics: StepMetric[];
  time_series_data: TimeSeriesPoint[];
  last_execution_time: string;
  calculated_at: string;
}

interface PerformanceMetricsProps {
  app: string;
  onClose: () => void;
}

export default function PerformanceMetrics({ app, onClose }: PerformanceMetricsProps) {
  const [metrics, setMetrics] = useState<PerformanceMetrics | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetchMetrics();
  }, [app]);

  const fetchMetrics = async () => {
    setLoading(true);
    setError(null);

    try {
      const token = localStorage.getItem('token') || sessionStorage.getItem('token');
      const response = await fetch(`/api/graph/${app}/metrics`, {
        headers: {
          Authorization: `Bearer ${token}`,
        },
      });

      if (!response.ok) {
        throw new Error(`Failed to fetch metrics: ${response.statusText}`);
      }

      const data = await response.json();
      setMetrics(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
    } finally {
      setLoading(false);
    }
  };

  const formatDuration = (seconds: number): string => {
    if (seconds < 1) {
      return `${(seconds * 1000).toFixed(0)}ms`;
    }
    if (seconds < 60) {
      return `${seconds.toFixed(2)}s`;
    }
    const minutes = Math.floor(seconds / 60);
    const remainingSeconds = (seconds % 60).toFixed(0);
    return `${minutes}m ${remainingSeconds}s`;
  };

  const formatTimestamp = (timestamp: string): string => {
    return new Date(timestamp).toLocaleString();
  };

  if (loading) {
    return (
      <div className="bg-white border rounded-lg shadow-lg p-6">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-xl font-bold">Performance Metrics</h2>
          <button onClick={onClose} className="text-gray-500 hover:text-gray-700">
            ✕
          </button>
        </div>
        <div className="text-center py-8">Loading metrics...</div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="bg-white border rounded-lg shadow-lg p-6">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-xl font-bold">Performance Metrics</h2>
          <button onClick={onClose} className="text-gray-500 hover:text-gray-700">
            ✕
          </button>
        </div>
        <div className="text-center py-8 text-red-500">{error}</div>
      </div>
    );
  }

  if (!metrics) {
    return null;
  }

  return (
    <div className="bg-white border rounded-lg shadow-lg p-6 max-h-[600px] overflow-y-auto">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h2 className="text-xl font-bold">Performance Metrics</h2>
          <p className="text-sm text-gray-500">Application: {metrics.application}</p>
        </div>
        <button onClick={onClose} className="text-gray-500 hover:text-gray-700">
          ✕
        </button>
      </div>

      {/* Summary Cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4 mb-6">
        <Card>
          <CardHeader className="pb-3">
            <div className="flex items-center justify-between">
              <CardTitle className="text-sm font-medium">Total Executions</CardTitle>
              <Activity className="h-4 w-4 text-gray-500" />
            </div>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{metrics.total_executions}</div>
            <p className="text-xs text-gray-500 mt-1">
              Last: {formatTimestamp(metrics.last_execution_time)}
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-3">
            <div className="flex items-center justify-between">
              <CardTitle className="text-sm font-medium">Success Rate</CardTitle>
              {metrics.success_rate_percent > 50 ? (
                <CheckCircle className="h-4 w-4 text-green-500" />
              ) : (
                <XCircle className="h-4 w-4 text-red-500" />
              )}
            </div>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{metrics.success_rate_percent.toFixed(1)}%</div>
            <div className="flex items-center gap-2 text-xs text-gray-500 mt-1">
              <span className="text-green-600">
                {metrics.success_rate_percent.toFixed(0)}% success
              </span>
              <span className="text-red-600">
                {metrics.failure_rate_percent.toFixed(0)}% failed
              </span>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-3">
            <div className="flex items-center justify-between">
              <CardTitle className="text-sm font-medium">Avg Duration</CardTitle>
              <Clock className="h-4 w-4 text-gray-500" />
            </div>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {formatDuration(metrics.average_duration_seconds)}
            </div>
            <p className="text-xs text-gray-500 mt-1">
              Min: {formatDuration(metrics.min_duration_seconds)} / Max:{' '}
              {formatDuration(metrics.max_duration_seconds)}
            </p>
          </CardContent>
        </Card>
      </div>

      {/* Step Metrics */}
      {metrics.step_metrics && metrics.step_metrics.length > 0 && (
        <div className="mb-6">
          <h3 className="text-lg font-semibold mb-3">Step Performance</h3>
          <div className="space-y-3">
            {metrics.step_metrics.map((step, idx) => (
              <Card key={idx}>
                <CardHeader className="pb-2">
                  <div className="flex items-center justify-between">
                    <CardTitle className="text-sm">{step.step_name}</CardTitle>
                    <span className="text-xs text-gray-500">{step.step_type}</span>
                  </div>
                </CardHeader>
                <CardContent>
                  <div className="grid grid-cols-2 gap-3 text-sm">
                    <div>
                      <span className="text-gray-500">Executions:</span>{' '}
                      <span className="font-medium">{step.execution_count}</span>
                    </div>
                    <div>
                      <span className="text-gray-500">Success Rate:</span>{' '}
                      <span
                        className={`font-medium ${
                          step.success_rate_percent > 50 ? 'text-green-600' : 'text-red-600'
                        }`}
                      >
                        {step.success_rate_percent.toFixed(0)}%
                      </span>
                    </div>
                    <div>
                      <span className="text-gray-500">Avg Duration:</span>{' '}
                      <span className="font-medium">
                        {formatDuration(step.average_duration_seconds)}
                      </span>
                    </div>
                    <div>
                      <span className="text-gray-500">Max Duration:</span>{' '}
                      <span className="font-medium">
                        {formatDuration(step.max_duration_seconds)}
                      </span>
                    </div>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        </div>
      )}

      {/* Time Series */}
      {metrics.time_series_data && metrics.time_series_data.length > 0 && (
        <div>
          <h3 className="text-lg font-semibold mb-3">Recent Executions</h3>
          <div className="space-y-2">
            {metrics.time_series_data.slice(0, 10).map((point, idx) => (
              <div
                key={idx}
                className="flex items-center justify-between p-2 bg-gray-50 rounded border"
              >
                <div className="flex items-center gap-3">
                  {point.status === 'succeeded' || point.status === 'completed' ? (
                    <CheckCircle className="h-4 w-4 text-green-500" />
                  ) : (
                    <XCircle className="h-4 w-4 text-red-500" />
                  )}
                  <span className="text-sm">{formatTimestamp(point.timestamp)}</span>
                </div>
                <div className="flex items-center gap-2">
                  <span className="text-sm font-medium">
                    {formatDuration(point.duration_seconds)}
                  </span>
                  <span
                    className={`text-xs px-2 py-1 rounded ${
                      point.status === 'succeeded' || point.status === 'completed'
                        ? 'bg-green-100 text-green-700'
                        : 'bg-red-100 text-red-700'
                    }`}
                  >
                    {point.status}
                  </span>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Calculated timestamp */}
      <div className="mt-6 text-xs text-gray-500 text-center">
        Calculated at: {formatTimestamp(metrics.calculated_at)}
      </div>
    </div>
  );
}
