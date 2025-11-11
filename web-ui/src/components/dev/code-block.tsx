'use client';

import { useState } from 'react';
import { Copy, Check } from 'lucide-react';
import { cn } from '@/lib/utils';

export function CodeBlock({
  code,
  language = 'yaml',
  className,
}: {
  code: string;
  language?: 'yaml' | 'json' | 'bash' | 'text';
  className?: string;
}) {
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    await navigator.clipboard.writeText(code);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div
      className={cn(
        'group relative rounded-lg border border-zinc-200 bg-zinc-50 dark:border-zinc-800 dark:bg-zinc-900',
        className
      )}
    >
      {/* Copy Button */}
      <button
        onClick={handleCopy}
        className="absolute right-2 top-2 flex h-7 w-7 items-center justify-center rounded border border-zinc-300 bg-white opacity-0 transition-opacity hover:bg-zinc-100 group-hover:opacity-100 dark:border-zinc-700 dark:bg-zinc-800 dark:hover:bg-zinc-700"
        aria-label="Copy code"
      >
        {copied ? (
          <Check size={14} className="text-green-600" />
        ) : (
          <Copy size={14} className="text-zinc-600 dark:text-zinc-400" />
        )}
      </button>

      {/* Code Content */}
      <pre className="overflow-x-auto p-4 text-xs leading-relaxed">
        <code className="font-mono text-zinc-800 dark:text-zinc-200">{code}</code>
      </pre>
    </div>
  );
}

// Inline code for small snippets
export function InlineCode({ children }: { children: React.ReactNode }) {
  return (
    <code className="rounded bg-zinc-100 px-1.5 py-0.5 font-mono text-xs text-zinc-800 dark:bg-zinc-800 dark:text-zinc-200">
      {children}
    </code>
  );
}

// Copyable text field (for connection strings, URLs, etc.)
export function CopyableText({ text, label }: { text: string; label?: string }) {
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    await navigator.clipboard.writeText(text);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="flex items-center gap-2 rounded-lg border border-zinc-200 bg-zinc-50 px-3 py-2 dark:border-zinc-800 dark:bg-zinc-900">
      {label && <span className="text-xs font-medium text-zinc-500">{label}:</span>}
      <code className="flex-1 font-mono text-xs text-zinc-800 dark:text-zinc-200">{text}</code>
      <button
        onClick={handleCopy}
        className="flex h-6 w-6 items-center justify-center rounded hover:bg-zinc-200 dark:hover:bg-zinc-700"
        aria-label="Copy"
      >
        {copied ? (
          <Check size={12} className="text-green-600" />
        ) : (
          <Copy size={12} className="text-zinc-500" />
        )}
      </button>
    </div>
  );
}
