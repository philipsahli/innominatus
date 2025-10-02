import fs from 'fs';
import path from 'path';
import matter from 'gray-matter';

export interface DocMetadata {
  title: string;
  description?: string;
  category?: string;
  order?: number;
}

export interface Doc {
  slug: string;
  content: string;
  metadata: DocMetadata;
  path: string;
}

export interface DocNavItem {
  title: string;
  slug: string;
  children?: DocNavItem[];
}

const DOCS_PATH = path.join(process.cwd(), '..', 'docs');

/**
 * Get all documentation files recursively
 */
export function getAllDocs(): Doc[] {
  const docs: Doc[] = [];

  function readDocsRecursive(dir: string, baseSlug: string = '') {
    const entries = fs.readdirSync(dir, { withFileTypes: true });

    for (const entry of entries) {
      const fullPath = path.join(dir, entry.name);

      if (entry.isDirectory()) {
        readDocsRecursive(fullPath, baseSlug + entry.name + '/');
      } else if (entry.name.endsWith('.md')) {
        const fileContent = fs.readFileSync(fullPath, 'utf-8');
        const { data, content } = matter(fileContent);

        const slug = (baseSlug + entry.name.replace(/\.md$/, '')).replace(/^\//, '');

        docs.push({
          slug,
          content,
          metadata: {
            title: data.title || extractTitleFromContent(content) || slug,
            description: data.description,
            category: data.category,
            order: data.order,
          },
          path: fullPath,
        });
      }
    }
  }

  readDocsRecursive(DOCS_PATH);
  return docs;
}

/**
 * Get a single documentation page by slug
 */
export function getDocBySlug(slug: string): Doc | null {
  const allDocs = getAllDocs();
  return allDocs.find((doc) => doc.slug === slug) || null;
}

/**
 * Generate navigation structure from docs
 */
export function getDocsNavigation(): DocNavItem[] {
  const navigation: DocNavItem[] = [
    {
      title: 'Getting Started',
      slug: 'getting-started',
      children: [
        { title: 'Quick Start', slug: 'getting-started/quick-start' },
        { title: 'Core Concepts', slug: 'getting-started/concepts' },
      ],
    },
    {
      title: 'Guides',
      slug: 'guides',
      children: [
        { title: 'Workflows', slug: 'guides/workflows' },
        { title: 'Golden Paths', slug: 'guides/golden-paths' },
        { title: 'Score Specs', slug: 'guides/score-specs' },
      ],
    },
    {
      title: 'Features',
      slug: 'features',
      children: [
        { title: 'Conditional Execution', slug: 'features/conditional-execution' },
        { title: 'Parallel Execution', slug: 'features/parallel-execution' },
        { title: 'Variable Interpolation', slug: 'features/context-variables' },
        { title: 'Health & Monitoring', slug: 'features/health-monitoring' },
      ],
    },
    {
      title: 'CLI Reference',
      slug: 'cli',
      children: [
        { title: 'Overview', slug: 'cli/overview' },
        { title: 'Commands', slug: 'cli/commands' },
        { title: 'Golden Paths', slug: 'cli/golden-paths' },
        { title: 'Output Formatting', slug: 'cli/output-formatting' },
      ],
    },
    {
      title: 'API Reference',
      slug: 'api',
      children: [
        { title: 'REST API', slug: 'api/rest-api' },
        { title: 'Swagger/OpenAPI', slug: 'api/swagger' },
      ],
    },
  ];

  return navigation;
}

/**
 * Extract title from markdown content (first H1)
 */
function extractTitleFromContent(content: string): string | null {
  const match = content.match(/^#\s+(.+)$/m);
  return match ? match[1] : null;
}

/**
 * Get breadcrumbs for a doc slug
 */
export function getBreadcrumbs(slug: string): { title: string; slug: string }[] {
  const parts = slug.split('/');
  const breadcrumbs: { title: string; slug: string }[] = [];

  let currentSlug = '';
  for (let i = 0; i < parts.length; i++) {
    currentSlug += (i > 0 ? '/' : '') + parts[i];
    const title = parts[i]
      .split('-')
      .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
      .join(' ');

    breadcrumbs.push({
      title,
      slug: currentSlug,
    });
  }

  return breadcrumbs;
}

/**
 * Search documentation
 */
export function searchDocs(query: string): Doc[] {
  const allDocs = getAllDocs();
  const lowerQuery = query.toLowerCase();

  return allDocs.filter((doc) => {
    const titleMatch = doc.metadata.title.toLowerCase().includes(lowerQuery);
    const contentMatch = doc.content.toLowerCase().includes(lowerQuery);
    return titleMatch || contentMatch;
  });
}
