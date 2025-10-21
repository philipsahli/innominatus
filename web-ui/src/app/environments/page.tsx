'use client';

import React, { useState, useEffect } from 'react';
import { ProtectedRoute } from '@/components/protected-route';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Globe, RefreshCw, CheckCircle, XCircle } from 'lucide-react';
import { api } from '@/lib/api';

interface Environment {
  Type: string;
  [key: string]: any;
}

export default function EnvironmentsPage() {
  const [environments, setEnvironments] = useState<Record<string, Environment>>({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchEnvironments = async () => {
    setLoading(true);
    setError(null);

    try {
      const response = await api.getEnvironments();

      if (response.success && response.data) {
        setEnvironments(response.data);
      } else {
        setError(response.error || 'Failed to fetch environments');
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch environments');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchEnvironments();
  }, []);

  const environmentList = Object.entries(environments);

  return (
    <ProtectedRoute>
      <div className="container mx-auto py-8 px-4">
        {/* Header */}
        <div className="flex items-center justify-between mb-8">
          <div>
            <h1 className="text-3xl font-bold text-gray-900 dark:text-gray-100 flex items-center gap-3">
              <Globe className="w-8 h-8 text-blue-500" />
              Environments
            </h1>
            <p className="text-gray-600 dark:text-gray-400 mt-2">
              Deployment targets for applications
            </p>
          </div>
          <Button onClick={fetchEnvironments} variant="outline" disabled={loading}>
            <RefreshCw className={`w-4 h-4 mr-2 ${loading ? 'animate-spin' : ''}`} />
            Refresh
          </Button>
        </div>

        {/* Statistics Cards */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
          <Card>
            <CardHeader className="pb-3">
              <CardTitle className="text-sm font-medium text-gray-600 dark:text-gray-400">
                Total Environments
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold text-gray-900 dark:text-gray-100">
                {environmentList.length}
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="pb-3">
              <CardTitle className="text-sm font-medium text-gray-600 dark:text-gray-400">
                Active
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold text-green-600 dark:text-green-400 flex items-center gap-2">
                <CheckCircle className="w-5 h-5" />
                {environmentList.length}
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="pb-3">
              <CardTitle className="text-sm font-medium text-gray-600 dark:text-gray-400">
                Status
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold text-blue-600 dark:text-blue-400">Operational</div>
            </CardContent>
          </Card>
        </div>

        {/* Environments Table */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Globe className="w-5 h-5" />
              Available Environments
              <span className="text-sm font-normal text-gray-500">({environmentList.length})</span>
            </CardTitle>
            <CardDescription>Environments available for application deployment</CardDescription>
          </CardHeader>
          <CardContent>
            {loading ? (
              <div className="flex items-center justify-center py-8">
                <RefreshCw className="w-6 h-6 animate-spin text-gray-400" />
                <span className="ml-2 text-gray-600 dark:text-gray-400">
                  Loading environments...
                </span>
              </div>
            ) : error ? (
              <div className="text-center py-8">
                <div className="flex items-center justify-center gap-2 text-red-600 dark:text-red-400 mb-4">
                  <XCircle className="w-6 h-6" />
                  <p>{error}</p>
                </div>
                <Button onClick={fetchEnvironments} variant="outline">
                  Try Again
                </Button>
              </div>
            ) : environmentList.length === 0 ? (
              <div className="text-center py-8 text-gray-500 dark:text-gray-400">
                <Globe className="w-12 h-12 mx-auto mb-4 text-gray-300 dark:text-gray-600" />
                <p className="text-lg font-medium">No environments found</p>
                <p className="text-sm mt-1">Configure environments to enable deployments</p>
              </div>
            ) : (
              <div className="overflow-x-auto">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Name</TableHead>
                      <TableHead>Type</TableHead>
                      <TableHead>Status</TableHead>
                      <TableHead>Description</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {environmentList.map(([name, env]) => (
                      <TableRow key={name} className="hover:bg-gray-50 dark:hover:bg-gray-800">
                        <TableCell>
                          <div className="font-medium text-gray-900 dark:text-gray-100 flex items-center gap-2">
                            <Globe className="w-4 h-4 text-blue-500" />
                            {name}
                          </div>
                        </TableCell>
                        <TableCell>
                          <Badge variant="outline">{env.Type || 'unknown'}</Badge>
                        </TableCell>
                        <TableCell>
                          <Badge
                            variant="default"
                            className="bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-100"
                          >
                            <CheckCircle className="w-3 h-3 mr-1" />
                            Active
                          </Badge>
                        </TableCell>
                        <TableCell>
                          <div className="text-sm text-gray-600 dark:text-gray-400">
                            {getEnvironmentDescription(name)}
                          </div>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            )}
          </CardContent>
        </Card>

        {/* Information Card */}
        <Card className="mt-6">
          <CardHeader>
            <CardTitle className="text-base">About Environments</CardTitle>
          </CardHeader>
          <CardContent className="text-sm text-gray-600 dark:text-gray-400 space-y-2">
            <p>
              <strong>Environments</strong> represent deployment targets for your applications. Each
              environment has its own configuration, policies, and resources.
            </p>
            <ul className="list-disc list-inside space-y-1 ml-2">
              <li>
                <strong>Development:</strong> For local development and testing
              </li>
              <li>
                <strong>Staging:</strong> Pre-production environment for final testing
              </li>
              <li>
                <strong>Production:</strong> Live environment serving end users
              </li>
              <li>
                <strong>Ephemeral:</strong> Temporary environments (PR previews, feature branches)
              </li>
            </ul>
            <p className="pt-2">
              When deploying an application, you can specify the target environment in your Score
              specification or through golden path parameters.
            </p>
          </CardContent>
        </Card>
      </div>
    </ProtectedRoute>
  );
}

function getEnvironmentDescription(name: string): string {
  const descriptions: Record<string, string> = {
    development: 'Local development and testing environment',
    dev: 'Development environment for feature development',
    staging: 'Pre-production environment for final testing',
    production: 'Live production environment',
    prod: 'Production environment serving end users',
    ephemeral: 'Temporary environments for PR previews',
    test: 'Automated testing environment',
    qa: 'Quality assurance and testing environment',
  };

  return descriptions[name.toLowerCase()] || 'Application deployment target';
}
