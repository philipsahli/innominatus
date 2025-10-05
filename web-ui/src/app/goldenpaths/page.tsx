'use client';

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Input } from '@/components/ui/input';
import {
  Activity,
  Search,
  Clock,
  Play,
  FileText,
  Tag,
  Layers,
  Info,
  Github,
  ExternalLink,
} from 'lucide-react';
import { ProtectedRoute } from '@/components/protected-route';
import { useState } from 'react';
import { useRouter } from 'next/navigation';
import {
  getAllGoldenPaths,
  getCategoryColor,
  getIconColor,
  getGitHubWorkflowUrl,
} from '@/lib/goldenpaths';

export default function GoldenPathsPage() {
  const router = useRouter();
  const [searchTerm, setSearchTerm] = useState('');
  const [categoryFilter, setCategoryFilter] = useState('all');

  const GOLDEN_PATHS = getAllGoldenPaths();

  // Filter golden paths based on search term and category
  const filteredPaths = GOLDEN_PATHS.filter((path) => {
    const matchesSearch =
      path.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      path.description.toLowerCase().includes(searchTerm.toLowerCase()) ||
      path.tags.some((tag) => tag.toLowerCase().includes(searchTerm.toLowerCase()));
    const matchesCategory = categoryFilter === 'all' || path.category === categoryFilter;
    return matchesSearch && matchesCategory;
  });

  // Get unique categories
  const categories = Array.from(new Set(GOLDEN_PATHS.map((path) => path.category)));

  // Get category statistics
  const stats = {
    total: GOLDEN_PATHS.length,
    deployment: GOLDEN_PATHS.filter((p) => p.category === 'deployment').length,
    cleanup: GOLDEN_PATHS.filter((p) => p.category === 'cleanup').length,
    environment: GOLDEN_PATHS.filter((p) => p.category === 'environment').length,
    database: GOLDEN_PATHS.filter((p) => p.category === 'database').length,
    observability: GOLDEN_PATHS.filter((p) => p.category === 'observability').length,
  };

  return (
    <ProtectedRoute>
      <div className="min-h-screen bg-white dark:bg-gray-900 p-6">
        <div className="relative space-y-6">
          {/* Header */}
          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <div className="p-2 rounded-lg bg-gray-200 dark:bg-gray-700">
                  <Activity className="w-6 h-6 text-gray-900 dark:text-gray-100" />
                </div>
                <div>
                  <h1 className="text-3xl font-bold tracking-tight text-gray-900 dark:text-gray-100">
                    Golden Paths
                  </h1>
                  <p className="text-gray-600 dark:text-gray-400">
                    Pre-defined workflow patterns for common platform operations
                  </p>
                </div>
              </div>
            </div>
          </div>

          {/* Statistics Cards */}
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-6">
            <Card className="bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700 shadow-lg">
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">Total</CardTitle>
                <Layers className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">{stats.total}</div>
              </CardContent>
            </Card>

            <Card className="bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700 shadow-lg">
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">Deployment</CardTitle>
                <Activity className="h-4 w-4 text-blue-500" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold text-gray-900 dark:text-gray-100">
                  {stats.deployment}
                </div>
              </CardContent>
            </Card>

            <Card className="bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700 shadow-lg">
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">Cleanup</CardTitle>
                <Activity className="h-4 w-4 text-red-500" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold text-gray-900 dark:text-gray-100">
                  {stats.cleanup}
                </div>
              </CardContent>
            </Card>

            <Card className="bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700 shadow-lg">
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">Environment</CardTitle>
                <Activity className="h-4 w-4 text-green-500" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold text-gray-900 dark:text-gray-100">
                  {stats.environment}
                </div>
              </CardContent>
            </Card>

            <Card className="bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700 shadow-lg">
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">Database</CardTitle>
                <Activity className="h-4 w-4 text-purple-500" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold text-gray-900 dark:text-gray-100">
                  {stats.database}
                </div>
              </CardContent>
            </Card>

            <Card className="bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700 shadow-lg">
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">Observability</CardTitle>
                <Activity className="h-4 w-4 text-orange-500" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold text-gray-900 dark:text-gray-100">
                  {stats.observability}
                </div>
              </CardContent>
            </Card>
          </div>

          {/* Filters and Search */}
          <Card className="bg-white/80 dark:bg-slate-900/80 backdrop-blur-sm border-white/20 shadow-lg">
            <CardHeader className="pb-4">
              <CardTitle className="text-lg">Filters</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="flex flex-col sm:flex-row gap-4">
                <div className="flex-1">
                  <div className="relative">
                    <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                    <Input
                      placeholder="Search golden paths, descriptions, or tags..."
                      value={searchTerm}
                      onChange={(e) => setSearchTerm(e.target.value)}
                      className="pl-10"
                    />
                  </div>
                </div>
                <div className="sm:w-48">
                  <select
                    value={categoryFilter}
                    onChange={(e) => setCategoryFilter(e.target.value)}
                    className="w-full h-10 px-3 py-2 text-sm bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  >
                    <option value="all">All Categories</option>
                    {categories.map((category) => (
                      <option key={category} value={category}>
                        {category.charAt(0).toUpperCase() + category.slice(1)}
                      </option>
                    ))}
                  </select>
                </div>
              </div>
            </CardContent>
          </Card>

          {/* Golden Paths Grid */}
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
            {filteredPaths.length === 0 ? (
              <div className="col-span-full flex items-center justify-center p-12 text-center">
                <div className="space-y-2">
                  <Activity className="w-8 h-8 text-muted-foreground mx-auto" />
                  <p className="text-muted-foreground">
                    {searchTerm || categoryFilter !== 'all'
                      ? 'No golden paths match your filters'
                      : 'No golden paths available'}
                  </p>
                </div>
              </div>
            ) : (
              filteredPaths.map((path) => (
                <Card
                  key={path.name}
                  className="bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700 hover:border-blue-400 dark:hover:border-blue-500 transition-colors shadow-lg cursor-pointer"
                  onClick={() => router.push(`/goldenpaths/${path.name}`)}
                >
                  <CardHeader className="pb-3">
                    <div className="flex items-start justify-between">
                      <div className="flex items-center gap-2 flex-1">
                        <FileText className={`w-4 h-4 ${getIconColor(path.category)}`} />
                        <CardTitle className="text-lg">{path.name}</CardTitle>
                      </div>
                      <div className="flex items-center gap-2">
                        <a
                          href={getGitHubWorkflowUrl(path.workflow)}
                          target="_blank"
                          rel="noopener noreferrer"
                          onClick={(e) => e.stopPropagation()}
                          className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-200"
                        >
                          <Github className="w-4 h-4" />
                        </a>
                        <Badge className={`${getCategoryColor(path.category)} text-xs`}>
                          {path.category}
                        </Badge>
                      </div>
                    </div>
                  </CardHeader>
                  <CardContent className="space-y-4">
                    <CardDescription className="text-sm">{path.description}</CardDescription>

                    {/* Tags */}
                    <div className="flex flex-wrap gap-1">
                      {path.tags.map((tag) => (
                        <Badge key={tag} variant="secondary" className="text-xs">
                          <Tag className="w-2 h-2 mr-1" />
                          {tag}
                        </Badge>
                      ))}
                    </div>

                    {/* Duration */}
                    <div className="flex items-center gap-2 text-sm text-muted-foreground">
                      <Clock className="w-3 h-3" />
                      <span>{path.estimated_duration}</span>
                    </div>

                    {/* Parameters count */}
                    {path.parameters && Object.keys(path.parameters).length > 0 && (
                      <div className="flex items-center gap-2 text-sm text-muted-foreground">
                        <Info className="w-3 h-3" />
                        <span>{Object.keys(path.parameters).length} configurable parameters</span>
                      </div>
                    )}

                    {/* Actions */}
                    <div className="flex gap-2 pt-2">
                      <Button
                        size="sm"
                        className="flex-1"
                        disabled
                        onClick={(e) => e.stopPropagation()}
                      >
                        <Play className="w-3 h-3 mr-2" />
                        Run Path
                      </Button>
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={(e) => {
                          e.stopPropagation();
                          router.push(`/goldenpaths/${path.name}`);
                        }}
                      >
                        <Info className="w-3 h-3 mr-2" />
                        Details
                      </Button>
                    </div>
                  </CardContent>
                </Card>
              ))
            )}
          </div>

          {/* Info Card */}
          <Card className="bg-white/80 dark:bg-slate-900/80 backdrop-blur-sm border-white/20 shadow-lg">
            <CardHeader>
              <CardTitle className="text-lg">What are Golden Paths?</CardTitle>
            </CardHeader>
            <CardContent className="space-y-2">
              <p className="text-sm text-muted-foreground">
                Golden Paths are pre-defined, curated workflows that represent the recommended way
                to accomplish common platform operations. They encapsulate best practices,
                standardized configurations, and proven patterns for deploying applications,
                managing infrastructure, and maintaining platform components.
              </p>
              <p className="text-sm text-muted-foreground">Each golden path includes:</p>
              <ul className="list-disc list-inside text-sm text-muted-foreground space-y-1">
                <li>Detailed description of what the workflow accomplishes</li>
                <li>Categorization for easy discovery (deployment, database, etc.)</li>
                <li>Searchable tags for filtering and organization</li>
                <li>Estimated completion time</li>
                <li>Configurable parameters with defaults and validation</li>
              </ul>
            </CardContent>
          </Card>
        </div>
      </div>
    </ProtectedRoute>
  );
}
