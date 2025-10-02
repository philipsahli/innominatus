'use client';

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { ProtectedRoute } from '@/components/protected-route';
import { BookOpen, FileText, ArrowUpRight, Search } from 'lucide-react';
import { useRouter } from 'next/navigation';
import { useState } from 'react';
import { Input } from '@/components/ui/input';

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

  const categoryTitles: Record<string, string> = {
    'getting-started': 'Getting Started',
    guides: 'Guides',
    features: 'Features',
    cli: 'CLI Reference',
    api: 'API Reference',
    General: 'General Documentation',
  };

  return (
    <ProtectedRoute>
      <div className="min-h-screen bg-white dark:bg-gray-900">
        <div className="p-6 space-y-6">
          {/* Header */}
          <div className="p-8 rounded-2xl border bg-white dark:bg-gray-900">
            <div className="flex items-center justify-between">
              <div className="space-y-2">
                <div className="flex items-center gap-3">
                  <div className="p-2 rounded-lg bg-gray-200 dark:bg-gray-700">
                    <BookOpen className="w-6 h-6 text-gray-900 dark:text-gray-100" />
                  </div>
                  <h1 className="text-4xl font-bold text-gray-900 dark:text-gray-100">
                    Documentation
                  </h1>
                </div>
                <p className="text-lg text-gray-600 dark:text-gray-400 max-w-2xl">
                  Learn about innominatus - Score-based platform orchestration, workflows, golden
                  paths, and more
                </p>
              </div>
            </div>
          </div>

          {/* Search */}
          <Card className="bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700 shadow-lg">
            <CardContent className="pt-6">
              <div className="relative">
                <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-gray-500 dark:text-gray-400" />
                <Input
                  placeholder="Search documentation..."
                  value={searchTerm}
                  onChange={(e) => setSearchTerm(e.target.value)}
                  className="pl-10 bg-white dark:bg-gray-800"
                />
              </div>
            </CardContent>
          </Card>

          {/* Documentation Categories */}
          {Object.entries(filteredCategories).map(([category, docs]) => (
            <div key={category} className="space-y-4">
              <div className="flex items-center gap-3">
                <div className="p-1.5 rounded-lg bg-gray-200 dark:bg-gray-700">
                  <FileText className="w-4 h-4 text-gray-900 dark:text-gray-100" />
                </div>
                <h2 className="text-2xl font-bold text-gray-900 dark:text-gray-100">
                  {categoryTitles[category] || category}
                </h2>
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
                        <ArrowUpRight className="w-4 h-4 text-gray-400 group-hover:text-indigo-600 dark:group-hover:text-indigo-400 transition-colors" />
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
          ))}

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

          {/* Quick Links */}
          <Card className="bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700 shadow-lg">
            <CardHeader>
              <CardTitle className="text-xl">Quick Links</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="grid gap-3 md:grid-cols-2">
                <Button
                  variant="outline"
                  onClick={() => router.push('/docs/getting-started/quick-start')}
                  className="justify-start h-auto py-4 bg-white dark:bg-gray-800 hover:bg-gray-50 dark:hover:bg-gray-700"
                >
                  <div className="text-left">
                    <div className="font-semibold">Quick Start Guide</div>
                    <div className="text-xs text-gray-500 dark:text-gray-400">
                      Get started in 5 minutes
                    </div>
                  </div>
                </Button>
                <Button
                  variant="outline"
                  onClick={() => router.push('/docs/getting-started/concepts')}
                  className="justify-start h-auto py-4 bg-white dark:bg-gray-800 hover:bg-gray-50 dark:hover:bg-gray-700"
                >
                  <div className="text-left">
                    <div className="font-semibold">Core Concepts</div>
                    <div className="text-xs text-gray-500 dark:text-gray-400">
                      Understand the architecture
                    </div>
                  </div>
                </Button>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    </ProtectedRoute>
  );
}
