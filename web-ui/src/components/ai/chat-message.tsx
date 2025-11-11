'use client';

import React, { useState } from 'react';
import { Bot, User, Copy, Check } from 'lucide-react';
import { cn } from '@/lib/utils';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import rehypeHighlight from 'rehype-highlight';
import 'highlight.js/styles/github-dark.css';

export interface ChatMessageProps {
  role: 'user' | 'assistant';
  content: string;
  timestamp?: string;
  citations?: string[];
}

export function ChatMessage({ role, content, timestamp, citations }: ChatMessageProps) {
  const isUser = role === 'user';
  const [copiedCode, setCopiedCode] = useState<string | null>(null);

  // Format ISO timestamp for display
  const formatTimestamp = (ts?: string) => {
    if (!ts) return '';
    try {
      return new Date(ts).toLocaleTimeString();
    } catch {
      return ts; // Fallback to original string if parsing fails
    }
  };

  // Copy code block to clipboard
  const handleCopyCode = (code: string, blockId: string) => {
    navigator.clipboard.writeText(code);
    setCopiedCode(blockId);
    setTimeout(() => setCopiedCode(null), 2000);
  };

  return (
    <div className={cn('flex gap-3 mb-4', isUser ? 'justify-end' : 'justify-start')}>
      {!isUser && (
        <div className="flex-shrink-0 w-8 h-8 rounded-full bg-blue-500 dark:bg-blue-600 flex items-center justify-center">
          <Bot className="w-5 h-5 text-white" />
        </div>
      )}
      <div
        className={cn('flex flex-col gap-1', isUser ? 'items-end' : 'items-start', 'max-w-[80%]')}
      >
        <div
          className={cn(
            'rounded-lg px-4 py-2 text-sm',
            isUser
              ? 'bg-blue-500 dark:bg-blue-600 text-white'
              : 'bg-gray-100 dark:bg-gray-800 text-gray-900 dark:text-gray-100'
          )}
        >
          {isUser ? (
            <p className="whitespace-pre-wrap">{content}</p>
          ) : (
            <ReactMarkdown
              remarkPlugins={[remarkGfm]}
              rehypePlugins={[rehypeHighlight]}
              components={{
                p: ({ children }) => <p className="mb-2 last:mb-0">{children}</p>,
                ul: ({ children }) => (
                  <ul className="list-disc list-inside mb-2 space-y-1">{children}</ul>
                ),
                ol: ({ children }) => (
                  <ol className="list-decimal list-inside mb-2 space-y-1">{children}</ol>
                ),
                li: ({ children }) => <li className="ml-2">{children}</li>,
                h1: ({ children }) => <h1 className="text-xl font-bold mb-2 mt-2">{children}</h1>,
                h2: ({ children }) => <h2 className="text-lg font-bold mb-2 mt-2">{children}</h2>,
                h3: ({ children }) => <h3 className="text-base font-bold mb-1 mt-1">{children}</h3>,
                code: ({ inline, className, children, ...props }: any) => {
                  const codeString = String(children).replace(/\n$/, '');
                  const language = className?.replace(/language-/, '') || '';
                  const blockId = `code-${Math.random().toString(36).substr(2, 9)}`;

                  if (inline) {
                    return (
                      <code
                        className="bg-gray-200 dark:bg-gray-700 px-1.5 py-0.5 rounded text-xs font-mono text-gray-900 dark:text-gray-100"
                        {...props}
                      >
                        {children}
                      </code>
                    );
                  }

                  return (
                    <div className="relative group my-2">
                      {language && (
                        <div className="flex items-center justify-between bg-gray-800 dark:bg-black px-3 py-1 rounded-t text-xs">
                          <span className="text-gray-400 font-mono">{language}</span>
                          <button
                            onClick={() => handleCopyCode(codeString, blockId)}
                            className="text-gray-400 hover:text-gray-200 transition-colors"
                            title="Copy code"
                          >
                            {copiedCode === blockId ? (
                              <Check className="w-4 h-4" />
                            ) : (
                              <Copy className="w-4 h-4" />
                            )}
                          </button>
                        </div>
                      )}
                      <code
                        className={cn(
                          'block bg-gray-900 dark:bg-black text-gray-100 p-3 text-xs font-mono overflow-x-auto',
                          language ? 'rounded-b' : 'rounded',
                          !language && 'relative'
                        )}
                        {...props}
                      >
                        {!language && (
                          <button
                            onClick={() => handleCopyCode(codeString, blockId)}
                            className="absolute top-2 right-2 text-gray-400 hover:text-gray-200 transition-colors opacity-0 group-hover:opacity-100"
                            title="Copy code"
                          >
                            {copiedCode === blockId ? (
                              <Check className="w-4 h-4" />
                            ) : (
                              <Copy className="w-4 h-4" />
                            )}
                          </button>
                        )}
                        {children}
                      </code>
                    </div>
                  );
                },
                pre: ({ children }) => <>{children}</>,
                blockquote: ({ children }) => (
                  <blockquote className="border-l-4 border-gray-300 dark:border-gray-600 pl-3 italic my-2">
                    {children}
                  </blockquote>
                ),
                a: ({ children, href }) => (
                  <a
                    href={href}
                    className="text-blue-600 dark:text-blue-400 underline hover:no-underline"
                    target="_blank"
                    rel="noopener noreferrer"
                  >
                    {children}
                  </a>
                ),
                table: ({ children }) => (
                  <div className="overflow-x-auto my-2">
                    <table className="min-w-full border border-gray-300 dark:border-gray-600">
                      {children}
                    </table>
                  </div>
                ),
                thead: ({ children }) => (
                  <thead className="bg-gray-200 dark:bg-gray-700">{children}</thead>
                ),
                th: ({ children }) => (
                  <th className="border border-gray-300 dark:border-gray-600 px-2 py-1 text-left">
                    {children}
                  </th>
                ),
                td: ({ children }) => (
                  <td className="border border-gray-300 dark:border-gray-600 px-2 py-1">
                    {children}
                  </td>
                ),
                strong: ({ children }) => <strong className="font-bold">{children}</strong>,
                em: ({ children }) => <em className="italic">{children}</em>,
              }}
            >
              {content}
            </ReactMarkdown>
          )}
        </div>
        {timestamp && (
          <span className="text-xs text-muted-foreground px-2">{formatTimestamp(timestamp)}</span>
        )}
        {citations && citations.length > 0 && (
          <div className="px-2 mt-1">
            <p className="text-xs text-muted-foreground mb-1">Sources:</p>
            <ul className="text-xs text-muted-foreground space-y-0.5">
              {citations.map((citation, idx) => (
                <li key={idx} className="truncate">
                  • {citation}
                </li>
              ))}
            </ul>
          </div>
        )}
      </div>
      {isUser && (
        <div className="flex-shrink-0 w-8 h-8 rounded-full bg-gray-500 dark:bg-gray-600 flex items-center justify-center">
          <User className="w-5 h-5 text-white" />
        </div>
      )}
    </div>
  );
}
