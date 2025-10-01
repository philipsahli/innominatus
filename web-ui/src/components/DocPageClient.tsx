'use client'

import { Card, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { ProtectedRoute } from '@/components/protected-route'
import { BookOpen, ChevronLeft, ChevronRight } from 'lucide-react'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import rehypeHighlight from 'rehype-highlight'
import rehypeSlug from 'rehype-slug'
import rehypeAutolinkHeadings from 'rehype-autolink-headings'
import { ComponentPropsWithoutRef } from 'react'
import { useRouter } from 'next/navigation'
import 'highlight.js/styles/github-dark.css'

interface Doc {
  slug: string
  content: string
  metadata: {
    title: string
    description?: string
  }
}

interface DocPageClientProps {
  doc: Doc
  prevDoc: Doc | null
  nextDoc: Doc | null
}

export function DocPageClient({ doc, prevDoc, nextDoc }: DocPageClientProps) {
  const router = useRouter()

  return (
    <ProtectedRoute>
      <div className="min-h-screen bg-white dark:bg-gray-900">
        <div className="p-6 space-y-6">
          {/* Header */}
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <div className="p-2 rounded-lg bg-gray-200 dark:bg-gray-700">
                <BookOpen className="w-6 h-6 text-gray-900 dark:text-gray-100" />
              </div>
              <div>
                <h1 className="text-3xl font-bold tracking-tight text-gray-900 dark:text-gray-100">
                  {doc.metadata.title}
                </h1>
                {doc.metadata.description && (
                  <p className="text-gray-600 dark:text-gray-400 mt-1">
                    {doc.metadata.description}
                  </p>
                )}
              </div>
            </div>
            <Button
              variant="outline"
              onClick={() => router.push('/docs')}
              className="bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700"
            >
              <ChevronLeft className="w-4 h-4 mr-2" />
              Back to Docs
            </Button>
          </div>

          {/* Content Card */}
          <Card className="bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700 shadow-lg">
            <CardContent className="pt-8 px-8 pb-8">
              <div className="prose prose-slate dark:prose-invert max-w-none
                prose-headings:text-gray-900 dark:prose-headings:text-gray-100
                prose-p:text-gray-700 dark:prose-p:text-gray-300
                prose-li:text-gray-700 dark:prose-li:text-gray-300
                prose-strong:text-gray-900 dark:prose-strong:text-gray-100
                prose-code:text-indigo-600 dark:prose-code:text-indigo-400
                prose-pre:bg-gray-900 dark:prose-pre:bg-gray-950
                prose-a:text-indigo-600 dark:prose-a:text-indigo-400">
                <ReactMarkdown
                  remarkPlugins={[remarkGfm]}
                  rehypePlugins={[
                    rehypeHighlight,
                    rehypeSlug,
                    [rehypeAutolinkHeadings, { behavior: 'wrap' }],
                  ]}
                  components={{
                    code({ inline, className, children, ...props }: ComponentPropsWithoutRef<'code'> & { inline?: boolean }) {
                      const match = /language-(\w+)/.exec(className || '')
                      const lang = match ? match[1] : ''

                      if (inline) {
                        return (
                          <code className="px-2 py-0.5 rounded bg-gray-100 dark:bg-gray-700 text-indigo-600 dark:text-indigo-400 text-sm font-mono" {...props}>
                            {children}
                          </code>
                        )
                      }

                      return (
                        <div className="relative group my-4 rounded-lg overflow-hidden border border-gray-200 dark:border-gray-700">
                          {lang && (
                            <div className="absolute top-2 right-2 px-3 py-1 text-xs font-medium text-gray-400 bg-gray-800 rounded z-10">
                              {lang}
                            </div>
                          )}
                          <pre className="!my-0 !p-4 overflow-x-auto bg-gray-900 dark:bg-gray-950">
                            <code className={className} {...props}>
                              {children}
                            </code>
                          </pre>
                        </div>
                      )
                    },

                    pre({ children }) {
                      return <>{children}</>
                    },

                    a({ children, href, ...props }) {
                      const isExternal = href?.startsWith('http')
                      return (
                        <a
                          href={href}
                          className="underline hover:no-underline"
                          target={isExternal ? '_blank' : undefined}
                          rel={isExternal ? 'noopener noreferrer' : undefined}
                          {...props}
                        >
                          {children}
                        </a>
                      )
                    },

                    blockquote({ children, ...props }) {
                      return (
                        <blockquote
                          className="border-l-4 border-gray-300 dark:border-gray-600 pl-4 py-2 my-4 italic"
                          {...props}
                        >
                          {children}
                        </blockquote>
                      )
                    },

                    table({ children, ...props }) {
                      return (
                        <div className="overflow-x-auto my-6 rounded-lg border border-gray-200 dark:border-gray-700">
                          <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700" {...props}>
                            {children}
                          </table>
                        </div>
                      )
                    },

                    th({ children, ...props }) {
                      return (
                        <th
                          className="px-4 py-3 text-left text-xs font-semibold text-gray-900 dark:text-gray-100 bg-gray-100 dark:bg-gray-800"
                          {...props}
                        >
                          {children}
                        </th>
                      )
                    },

                    td({ children, ...props }) {
                      return (
                        <td className="px-4 py-3 text-sm text-gray-700 dark:text-gray-300" {...props}>
                          {children}
                        </td>
                      )
                    },
                  }}
                >
                  {doc.content}
                </ReactMarkdown>
              </div>
            </CardContent>
          </Card>

          {/* Navigation */}
          {(prevDoc || nextDoc) && (
            <div className="flex items-center justify-between gap-4">
              <div className="flex-1">
                {prevDoc && (
                  <Button
                    variant="outline"
                    onClick={() => router.push(`/docs/${prevDoc.slug}`)}
                    className="w-full bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-700"
                  >
                    <ChevronLeft className="w-4 h-4 mr-2" />
                    <div className="text-left flex-1">
                      <div className="text-xs text-gray-500 dark:text-gray-400">Previous</div>
                      <div className="text-sm font-medium">{prevDoc.metadata.title}</div>
                    </div>
                  </Button>
                )}
              </div>
              <div className="flex-1">
                {nextDoc && (
                  <Button
                    variant="outline"
                    onClick={() => router.push(`/docs/${nextDoc.slug}`)}
                    className="w-full bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-700"
                  >
                    <div className="text-right flex-1">
                      <div className="text-xs text-gray-500 dark:text-gray-400">Next</div>
                      <div className="text-sm font-medium">{nextDoc.metadata.title}</div>
                    </div>
                    <ChevronRight className="w-4 h-4 ml-2" />
                  </Button>
                )}
              </div>
            </div>
          )}
        </div>
      </div>
    </ProtectedRoute>
  )
}
