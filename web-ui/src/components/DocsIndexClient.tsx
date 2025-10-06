'use client';

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { ProtectedRoute } from '@/components/protected-route';
import {
  BookOpen,
  FileText,
  ArrowUpRight,
  Search,
  User,
  Settings,
  Rocket,
  BookMarked,
  Wrench,
  Terminal,
  Star,
  Clock,
} from 'lucide-react';
import { useRouter } from 'next/navigation';
import { useState } from 'react';

interface Doc {
  slug: string;
  content: string;
  metadata: {
    title: string;
    description?: string;
  };
}

interface DocsIndexClientProps {
  allDocs: Doc[];
}

export function DocsIndexClient({ allDocs }: DocsIndexClientProps) {
  const router = useRouter();
  const [searchTerm, setSearchTerm] = useState('');

  // Group docs by category
  const categories: Record<string, typeof allDocs> = {};
  allDocs.forEach((doc) => {
    const category = doc.slug.includes('/') ? doc.slug.split('/')[0] : 'General';
    if (!categories[category]) {
      categories[category] = [];
    }
    categories[category].push(doc);
  });

  // Filter docs based on search
  const filteredCategories = Object.entries(categories).reduce(
    (acc, [category, docs]) => {
      const filteredDocs = docs.filter(
        (doc) =>
          doc.metadata.title.toLowerCase().includes(searchTerm.toLowerCase()) ||
          (doc.metadata.description &&
            doc.metadata.description.toLowerCase().includes(searchTerm.toLowerCase()))
      );
      if (filteredDocs.length > 0) {
        acc[category] = filteredDocs;
      }
      return acc;
    },
    {} as Record<string, typeof allDocs>
  );

  // Category configuration with icons, titles, and descriptions
  const categoryConfig: Record<
    string,
    { title: string; icon: typeof User; description: string; order: number }
  > = {
    'getting-started': {
      title: 'Getting Started',
      icon: Rocket,
      description: 'Quick start guides and core concepts',
      order: 1,
    },
    'user-guide': {
      title: '👨‍💻 User Guide (For Developers)',
      icon: User,
      description: 'Deploy applications using innominatus',
      order: 2,
    },
    'platform-team-guide': {
      title: '⚙️ Platform Team Guide',
      icon: Settings,
      description: 'Install and operate innominatus',
      order: 3,
    },
    cli: {
      title: '💻 CLI Reference',
      icon: Terminal,
      description: 'Command-line tool documentation',
      order: 4,
    },
    features: {
      title: '⭐ Features',
      icon: Star,
      description: 'Advanced capabilities and workflows',
      order: 5,
    },
    development: {
      title: '🔧 Development Guide',
      icon: Wrench,
      description: 'Contributing to innominatus',
      order: 6,
    },
    guides: {
      title: 'Guides',
      icon: BookMarked,
      description: 'In-depth tutorials and workflows',
      order: 7,
    },
    General: {
      title: 'General Documentation',
      icon: FileText,
      description: 'Other documentation',
      order: 8,
    },
  };

  // Sort categories by order
  const sortedCategories = Object.entries(filteredCategories).sort(([a], [b]) => {
    const orderA = categoryConfig[a]?.order ?? 999;
    const orderB = categoryConfig[b]?.order ?? 999;
    return orderA - orderB;
  });

  // Featured documentation
  const featuredDocs = [
    {
      slug: 'user-guide/getting-started',
      title: 'Getting Started',
      description: 'Connect, install CLI, and deploy',
      duration: '15 minutes',
      icon: Rocket,
    },
    {
      slug: 'user-guide/recipes/nodejs-postgres',
      title: 'Node.js + PostgreSQL',
      description: 'Production-ready API recipe',
      duration: '15 minutes',
      icon: BookMarked,
    },
    {
      slug: 'platform-team-guide/quick-install',
      title: 'Quick Install',
      description: 'Production setup for platform teams',
      duration: '4 hours',
      icon: Settings,
    },
  ];

  return (
    <ProtectedRoute>
      <div className="min-h-screen bg-white dark:bg-gray-900">
        <div className="p-6 space-y-6">
          {/* Persona-Based Hero Section */}
          <div className="p-8 rounded-2xl border bg-gradient-to-br from-indigo-50 to-blue-50 dark:from-gray-800 dark:to-gray-900">
            <div className="space-y-6">
              <div className="space-y-2">
                <div className="flex items-center gap-3">
                  <div className="p-2 rounded-lg bg-white dark:bg-gray-800 shadow-sm">
                    <BookOpen className="w-6 h-6 text-indigo-600 dark:text-indigo-400" />
                  </div>
                  <h1 className="text-4xl font-bold text-gray-900 dark:text-gray-100">
                    Documentation
                  </h1>
                </div>
                <p className="text-lg text-gray-600 dark:text-gray-400 max-w-2xl">
                  Choose your learning path to get started with innominatus
                </p>
              </div>

              {/* Persona Chooser */}
              <div className="grid md:grid-cols-2 gap-4 max-w-3xl">
                <Card
                  className="bg-white dark:bg-gray-800 border-2 border-indigo-200 dark:border-indigo-800 hover:border-indigo-400 dark:hover:border-indigo-600 cursor-pointer transition-all hover:shadow-lg"
                  onClick={() => router.push('/docs/user-guide/README')}
                >
                  <CardContent className="pt-6">
                    <div className="flex items-start gap-4">
                      <div className="p-3 rounded-lg bg-indigo-100 dark:bg-indigo-900">
                        <User className="w-6 h-6 text-indigo-600 dark:text-indigo-400" />
                      </div>
                      <div className="space-y-1 flex-1">
                        <h3 className="font-semibold text-lg text-gray-900 dark:text-gray-100">
                          I&apos;m a Developer
                        </h3>
                        <p className="text-sm text-gray-600 dark:text-gray-400">
                          Deploy applications using the platform
                        </p>
                        <div className="flex items-center gap-2 text-xs text-indigo-600 dark:text-indigo-400 font-medium pt-2">
                          <span>Start learning</span>
                          <ArrowUpRight className="w-3 h-3" />
                        </div>
                      </div>
                    </div>
                  </CardContent>
                </Card>

                <Card
                  className="bg-white dark:bg-gray-800 border-2 border-purple-200 dark:border-purple-800 hover:border-purple-400 dark:hover:border-purple-600 cursor-pointer transition-all hover:shadow-lg"
                  onClick={() => router.push('/docs/platform-team-guide/README')}
                >
                  <CardContent className="pt-6">
                    <div className="flex items-start gap-4">
                      <div className="p-3 rounded-lg bg-purple-100 dark:bg-purple-900">
                        <Settings className="w-6 h-6 text-purple-600 dark:text-purple-400" />
                      </div>
                      <div className="space-y-1 flex-1">
                        <h3 className="font-semibold text-lg text-gray-900 dark:text-gray-100">
                          I&apos;m a Platform Engineer
                        </h3>
                        <p className="text-sm text-gray-600 dark:text-gray-400">
                          Install and operate the platform
                        </p>
                        <div className="flex items-center gap-2 text-xs text-purple-600 dark:text-purple-400 font-medium pt-2">
                          <span>Start learning</span>
                          <ArrowUpRight className="w-3 h-3" />
                        </div>
                      </div>
                    </div>
                  </CardContent>
                </Card>
              </div>
            </div>
          </div>

          {/* Featured Documentation */}
          <div className="space-y-4">
            <div className="flex items-center gap-3">
              <div className="p-1.5 rounded-lg bg-yellow-100 dark:bg-yellow-900">
                <Star className="w-4 h-4 text-yellow-600 dark:text-yellow-400" />
              </div>
              <h2 className="text-2xl font-bold text-gray-900 dark:text-gray-100">
                Featured Documentation
              </h2>
            </div>

            <div className="grid gap-4 md:grid-cols-3">
              {featuredDocs.map((doc) => (
                <Card
                  key={doc.slug}
                  className="bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700 shadow-lg hover:shadow-xl transition-all cursor-pointer group border-l-4 border-l-indigo-500"
                  onClick={() => router.push(`/docs/${doc.slug}`)}
                >
                  <CardContent className="pt-6">
                    <div className="space-y-3">
                      <div className="flex items-start justify-between">
                        <div className="p-2 rounded-lg bg-indigo-100 dark:bg-indigo-900">
                          <doc.icon className="w-5 h-5 text-indigo-600 dark:text-indigo-400" />
                        </div>
                        <ArrowUpRight className="w-4 h-4 text-gray-400 group-hover:text-indigo-600 dark:group-hover:text-indigo-400 transition-colors" />
                      </div>
                      <div className="space-y-1">
                        <h3 className="font-semibold text-gray-900 dark:text-gray-100 group-hover:text-indigo-600 dark:group-hover:text-indigo-400 transition-colors">
                          {doc.title}
                        </h3>
                        <p className="text-sm text-gray-600 dark:text-gray-400">
                          {doc.description}
                        </p>
                        <div className="flex items-center gap-1 text-xs text-gray-500 dark:text-gray-500 pt-1">
                          <Clock className="w-3 h-3" />
                          <span>{doc.duration}</span>
                        </div>
                      </div>
                    </div>
                  </CardContent>
                </Card>
              ))}
            </div>
          </div>

          {/* Search */}
          <Card className="bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700 shadow-lg">
            <CardContent className="pt-6">
              <div className="relative">
                <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-gray-500 dark:text-gray-400" />
                <input
                  type="text"
                  placeholder="Search documentation..."
                  value={searchTerm}
                  onChange={(e) => setSearchTerm(e.target.value)}
                  className="w-full pl-10 pr-4 py-2 rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-indigo-500 focus:border-transparent"
                />
              </div>
            </CardContent>
          </Card>

          {/* Documentation Categories */}
          {sortedCategories.map(([category, docs]) => {
            const config = categoryConfig[category] || categoryConfig.General;
            const IconComponent = config.icon;

            return (
              <div key={category} className="space-y-4">
                <div className="flex items-start gap-3">
                  <div className="p-1.5 rounded-lg bg-gray-200 dark:bg-gray-700">
                    <IconComponent className="w-4 h-4 text-gray-900 dark:text-gray-100" />
                  </div>
                  <div className="space-y-1">
                    <h2 className="text-2xl font-bold text-gray-900 dark:text-gray-100">
                      {config.title}
                    </h2>
                    <p className="text-sm text-gray-600 dark:text-gray-400">{config.description}</p>
                  </div>
                </div>

                <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
                  {docs.map((doc) => (
                    <Card
                      key={doc.slug}
                      className="bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700 shadow-lg hover:shadow-xl transition-shadow cursor-pointer group"
                      onClick={() => router.push(`/docs/${doc.slug}`)}
                    >
                      <CardHeader className="pb-3">
                        <div className="flex items-start justify-between">
                          <CardTitle className="text-lg font-semibold text-gray-900 dark:text-gray-100 group-hover:text-indigo-600 dark:group-hover:text-indigo-400 transition-colors">
                            {doc.metadata.title}
                          </CardTitle>
                          <ArrowUpRight className="w-4 h-4 text-gray-400 group-hover:text-indigo-600 dark:group-hover:text-indigo-400 transition-colors flex-shrink-0" />
                        </div>
                      </CardHeader>
                      <CardContent>
                        {doc.metadata.description && (
                          <p className="text-sm text-gray-600 dark:text-gray-400 line-clamp-2">
                            {doc.metadata.description}
                          </p>
                        )}
                      </CardContent>
                    </Card>
                  ))}
                </div>
              </div>
            );
          })}

          {Object.keys(filteredCategories).length === 0 && (
            <Card className="bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700 shadow-lg">
              <CardContent className="py-12">
                <div className="text-center space-y-2">
                  <FileText className="w-12 h-12 text-gray-400 mx-auto" />
                  <p className="text-gray-600 dark:text-gray-400">
                    No documentation found matching &quot;{searchTerm}&quot;
                  </p>
                </div>
              </CardContent>
            </Card>
          )}

          {/* Quick Links - Persona Based */}
          <Card className="bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700 shadow-lg">
            <CardHeader>
              <CardTitle className="text-xl">Quick Links</CardTitle>
            </CardHeader>
            <CardContent className="space-y-6">
              {/* For Developers */}
              <div className="space-y-3">
                <h3 className="font-semibold text-sm text-gray-700 dark:text-gray-300 flex items-center gap-2">
                  <User className="w-4 h-4" />
                  For Developers
                </h3>
                <div className="grid gap-3 md:grid-cols-2">
                  <Button
                    variant="outline"
                    onClick={() => router.push('/docs/user-guide/getting-started')}
                    className="justify-start h-auto py-4 bg-white dark:bg-gray-800 hover:bg-gray-50 dark:hover:bg-gray-700"
                  >
                    <div className="text-left">
                      <div className="font-semibold">Getting Started</div>
                      <div className="text-xs text-gray-500 dark:text-gray-400">
                        Deploy in 15 minutes
                      </div>
                    </div>
                  </Button>
                  <Button
                    variant="outline"
                    onClick={() => router.push('/docs/user-guide/recipes/nodejs-postgres')}
                    className="justify-start h-auto py-4 bg-white dark:bg-gray-800 hover:bg-gray-50 dark:hover:bg-gray-700"
                  >
                    <div className="text-left">
                      <div className="font-semibold">Node.js + PostgreSQL</div>
                      <div className="text-xs text-gray-500 dark:text-gray-400">
                        Production-ready API
                      </div>
                    </div>
                  </Button>
                  <Button
                    variant="outline"
                    onClick={() => router.push('/docs/user-guide/cli-reference')}
                    className="justify-start h-auto py-4 bg-white dark:bg-gray-800 hover:bg-gray-50 dark:hover:bg-gray-700"
                  >
                    <div className="text-left">
                      <div className="font-semibold">CLI Reference</div>
                      <div className="text-xs text-gray-500 dark:text-gray-400">
                        Complete commands
                      </div>
                    </div>
                  </Button>
                  <Button
                    variant="outline"
                    onClick={() => router.push('/docs/user-guide/troubleshooting')}
                    className="justify-start h-auto py-4 bg-white dark:bg-gray-800 hover:bg-gray-50 dark:hover:bg-gray-700"
                  >
                    <div className="text-left">
                      <div className="font-semibold">Troubleshooting</div>
                      <div className="text-xs text-gray-500 dark:text-gray-400">Common issues</div>
                    </div>
                  </Button>
                </div>
              </div>

              {/* For Platform Teams */}
              <div className="space-y-3">
                <h3 className="font-semibold text-sm text-gray-700 dark:text-gray-300 flex items-center gap-2">
                  <Settings className="w-4 h-4" />
                  For Platform Teams
                </h3>
                <div className="grid gap-3 md:grid-cols-2">
                  <Button
                    variant="outline"
                    onClick={() => router.push('/docs/platform-team-guide/quick-install')}
                    className="justify-start h-auto py-4 bg-white dark:bg-gray-800 hover:bg-gray-50 dark:hover:bg-gray-700"
                  >
                    <div className="text-left">
                      <div className="font-semibold">Quick Install</div>
                      <div className="text-xs text-gray-500 dark:text-gray-400">
                        Production in 4 hours
                      </div>
                    </div>
                  </Button>
                  <Button
                    variant="outline"
                    onClick={() => router.push('/docs/platform-team-guide/authentication')}
                    className="justify-start h-auto py-4 bg-white dark:bg-gray-800 hover:bg-gray-50 dark:hover:bg-gray-700"
                  >
                    <div className="text-left">
                      <div className="font-semibold">Authentication</div>
                      <div className="text-xs text-gray-500 dark:text-gray-400">OIDC/SSO setup</div>
                    </div>
                  </Button>
                  <Button
                    variant="outline"
                    onClick={() => router.push('/docs/platform-team-guide/monitoring')}
                    className="justify-start h-auto py-4 bg-white dark:bg-gray-800 hover:bg-gray-50 dark:hover:bg-gray-700"
                  >
                    <div className="text-left">
                      <div className="font-semibold">Monitoring</div>
                      <div className="text-xs text-gray-500 dark:text-gray-400">
                        Prometheus & Grafana
                      </div>
                    </div>
                  </Button>
                  <Button
                    variant="outline"
                    onClick={() => router.push('/docs/platform-team-guide/operations')}
                    className="justify-start h-auto py-4 bg-white dark:bg-gray-800 hover:bg-gray-50 dark:hover:bg-gray-700"
                  >
                    <div className="text-left">
                      <div className="font-semibold">Operations</div>
                      <div className="text-xs text-gray-500 dark:text-gray-400">
                        Scaling & backup
                      </div>
                    </div>
                  </Button>
                </div>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    </ProtectedRoute>
  );
}
