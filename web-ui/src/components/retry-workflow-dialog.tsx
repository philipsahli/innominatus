'use client';

import { useState } from 'react';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { RotateCcw, Upload, AlertCircle, CheckCircle } from 'lucide-react';
import { api } from '@/lib/api';

interface RetryWorkflowDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  workflowId: string;
  workflowName: string;
  onSuccess?: () => void;
}

export function RetryWorkflowDialog({
  open,
  onOpenChange,
  workflowId,
  workflowName,
  onSuccess,
}: RetryWorkflowDialogProps) {
  const [mode, setMode] = useState<'auto' | 'manual'>('auto');
  const [file, setFile] = useState<File | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const selectedFile = e.target.files?.[0];
    if (selectedFile) {
      setFile(selectedFile);
      setError(null);
    }
  };

  const handleRetry = async () => {
    setLoading(true);
    setError(null);
    setSuccess(false);

    try {
      let workflowSpec = undefined;

      if (mode === 'manual') {
        if (!file) {
          setError('Please select a workflow file');
          setLoading(false);
          return;
        }

        // Read and parse the YAML/JSON file
        const fileContent = await file.text();
        try {
          // Try parsing as JSON first
          workflowSpec = JSON.parse(fileContent);
        } catch {
          // If not JSON, pass as-is and let the server handle YAML parsing
          setError('Please upload a valid JSON workflow file');
          setLoading(false);
          return;
        }
      }

      // Call the retry API (empty body for automatic retry)
      const response = await api.retryWorkflow(workflowId, workflowSpec);

      if (response.success) {
        setSuccess(true);
        setTimeout(() => {
          onOpenChange(false);
          onSuccess?.();
          // Reset state after closing
          setTimeout(() => {
            setMode('auto');
            setFile(null);
            setSuccess(false);
            setError(null);
          }, 300);
        }, 1500);
      } else {
        setError(response.error || 'Failed to retry workflow');
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to retry workflow');
    } finally {
      setLoading(false);
    }
  };

  const handleClose = () => {
    if (!loading) {
      onOpenChange(false);
      // Reset state after closing
      setTimeout(() => {
        setMode('auto');
        setFile(null);
        setSuccess(false);
        setError(null);
      }, 300);
    }
  };

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>Retry Workflow</DialogTitle>
          <DialogDescription>
            Retry the failed workflow <span className="font-semibold text-white">{workflowName}</span>
          </DialogDescription>
        </DialogHeader>

        {success ? (
          <div className="space-y-4">
            <div className="flex items-center gap-3 p-4 bg-green-900/20 border border-green-700 rounded-lg">
              <CheckCircle className="w-5 h-5 text-green-400 flex-shrink-0" />
              <div className="text-sm text-green-200">
                Workflow retry initiated successfully!
              </div>
            </div>
          </div>
        ) : (
          <div className="space-y-4">
            {/* Mode Selection */}
            <div className="space-y-2">
              <label className="text-sm font-medium text-gray-300">Retry Mode</label>
              <div className="grid grid-cols-2 gap-3">
                <button
                  type="button"
                  onClick={() => setMode('auto')}
                  disabled={loading}
                  className={`flex items-center justify-center gap-2 p-3 rounded-lg border-2 transition-colors ${
                    mode === 'auto'
                      ? 'border-blue-500 bg-blue-900/20 text-blue-300'
                      : 'border-gray-600 bg-gray-700/50 text-gray-400 hover:border-gray-500'
                  } ${loading ? 'opacity-50 cursor-not-allowed' : 'cursor-pointer'}`}
                >
                  <RotateCcw className="w-4 h-4" />
                  <span className="text-sm font-medium">Automatic</span>
                </button>
                <button
                  type="button"
                  onClick={() => setMode('manual')}
                  disabled={loading}
                  className={`flex items-center justify-center gap-2 p-3 rounded-lg border-2 transition-colors ${
                    mode === 'manual'
                      ? 'border-blue-500 bg-blue-900/20 text-blue-300'
                      : 'border-gray-600 bg-gray-700/50 text-gray-400 hover:border-gray-500'
                  } ${loading ? 'opacity-50 cursor-not-allowed' : 'cursor-pointer'}`}
                >
                  <Upload className="w-4 h-4" />
                  <span className="text-sm font-medium">Manual</span>
                </button>
              </div>
            </div>

            {/* Mode Description */}
            <div className="text-sm text-gray-400 bg-gray-800/50 p-3 rounded-lg">
              {mode === 'auto' ? (
                <p>
                  Retry the workflow from the first failed step using the original workflow specification.
                </p>
              ) : (
                <p>
                  Upload an updated workflow specification file (JSON format) to retry with modified steps
                  or configuration.
                </p>
              )}
            </div>

            {/* File Upload for Manual Mode */}
            {mode === 'manual' && (
              <div className="space-y-2">
                <label className="text-sm font-medium text-gray-300">Workflow File (JSON)</label>
                <div className="flex items-center gap-2">
                  <input
                    type="file"
                    accept=".json,.yaml,.yml"
                    onChange={handleFileChange}
                    disabled={loading}
                    className="block w-full text-sm text-gray-400
                      file:mr-4 file:py-2 file:px-4
                      file:rounded-lg file:border-0
                      file:text-sm file:font-semibold
                      file:bg-blue-900/20 file:text-blue-300
                      hover:file:bg-blue-900/30
                      file:cursor-pointer
                      disabled:opacity-50 disabled:cursor-not-allowed"
                  />
                </div>
                {file && (
                  <p className="text-xs text-gray-500">
                    Selected: {file.name} ({(file.size / 1024).toFixed(1)} KB)
                  </p>
                )}
              </div>
            )}

            {/* Error Message */}
            {error && (
              <div className="flex items-start gap-3 p-4 bg-red-900/20 border border-red-700 rounded-lg">
                <AlertCircle className="w-5 h-5 text-red-400 flex-shrink-0 mt-0.5" />
                <div className="text-sm text-red-200">{error}</div>
              </div>
            )}

            {/* Action Buttons */}
            <div className="flex justify-end gap-3 pt-2">
              <Button
                variant="outline"
                onClick={handleClose}
                disabled={loading}
                className="border-gray-600 hover:bg-gray-700"
              >
                Cancel
              </Button>
              <Button
                onClick={handleRetry}
                disabled={loading || (mode === 'manual' && !file)}
                className="bg-blue-600 hover:bg-blue-700"
              >
                {loading ? (
                  <>
                    <RefreshCw className="w-4 h-4 mr-2 animate-spin" />
                    Retrying...
                  </>
                ) : (
                  <>
                    <RotateCcw className="w-4 h-4 mr-2" />
                    Retry Workflow
                  </>
                )}
              </Button>
            </div>
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}

function RefreshCw(props: React.SVGProps<SVGSVGElement>) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      {...props}
    >
      <path d="M3 12a9 9 0 0 1 9-9 9.75 9.75 0 0 1 6.74 2.74L21 8" />
      <path d="M21 3v5h-5" />
      <path d="M21 12a9 9 0 0 1-9 9 9.75 9.75 0 0 1-6.74-2.74L3 16" />
      <path d="M3 21v-5h5" />
    </svg>
  );
}
