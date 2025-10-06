'use client';

import React, { useState } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { FileCode, Copy, Check, Download, Send } from 'lucide-react';

export interface SpecPreviewProps {
  spec: string;
  explanation?: string;
  citations?: string[];
  onDeploy?: (spec: string) => void;
}

export function SpecPreview({ spec, explanation, citations, onDeploy }: SpecPreviewProps) {
  const [copied, setCopied] = useState(false);

  const handleCopy = () => {
    navigator.clipboard.writeText(spec);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const handleDownload = () => {
    const blob = new Blob([spec], { type: 'text/yaml' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'score-spec.yaml';
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  };

  return (
    <Card className="border-blue-200 dark:border-blue-800 bg-blue-50 dark:bg-blue-950/20">
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <FileCode className="w-5 h-5 text-blue-600 dark:text-blue-400" />
            <CardTitle className="text-base">Generated Score Specification</CardTitle>
          </div>
          <div className="flex gap-2">
            <Button onClick={handleCopy} variant="ghost" size="sm">
              {copied ? <Check className="w-4 h-4" /> : <Copy className="w-4 h-4" />}
            </Button>
            <Button onClick={handleDownload} variant="ghost" size="sm">
              <Download className="w-4 h-4" />
            </Button>
            {onDeploy && (
              <Button
                onClick={() => onDeploy(spec)}
                size="sm"
                className="bg-blue-500 hover:bg-blue-600 text-white"
              >
                <Send className="w-4 h-4 mr-1" />
                Deploy
              </Button>
            )}
          </div>
        </div>
      </CardHeader>
      <CardContent className="space-y-3">
        {explanation && (
          <div className="text-sm text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-900 p-3 rounded-lg">
            <p className="font-semibold mb-1">Explanation:</p>
            <p className="whitespace-pre-wrap">{explanation}</p>
          </div>
        )}
        <pre className="bg-gray-900 dark:bg-black text-gray-100 p-4 rounded-lg overflow-x-auto text-xs">
          <code>{spec}</code>
        </pre>
        {citations && citations.length > 0 && (
          <div className="text-xs text-muted-foreground">
            <p className="font-semibold mb-1">Sources used:</p>
            <ul className="space-y-0.5">
              {citations.map((citation, idx) => (
                <li key={idx}>• {citation}</li>
              ))}
            </ul>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
