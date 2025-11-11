import { useMemo } from 'react';
import { AlertCircle, FileCode } from 'lucide-react';
import { WizardData } from './types';
import { generateScoreYaml } from './yaml-generator';

interface YamlPreviewProps {
  data: WizardData;
}

export function YamlPreview({ data }: YamlPreviewProps) {
  const { yaml, error } = useMemo(() => {
    try {
      const generatedYaml = generateScoreYaml(data);
      return { yaml: generatedYaml, error: null };
    } catch (err) {
      return {
        yaml: '',
        error: err instanceof Error ? err.message : 'Invalid configuration',
      };
    }
  }, [data]);

  return (
    <div className="flex h-full flex-col rounded-lg border border-zinc-200 bg-zinc-50 dark:border-zinc-800 dark:bg-zinc-900">
      {/* Header */}
      <div className="flex items-center gap-2 border-b border-zinc-200 px-4 py-3 dark:border-zinc-800">
        <FileCode size={18} className="text-zinc-600 dark:text-zinc-400" />
        <h3 className="text-sm font-medium text-zinc-900 dark:text-white">
          Score Specification Preview
        </h3>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto p-4">
        {error ? (
          <div className="flex items-start gap-2 rounded-lg border border-red-200 bg-red-50 p-3 dark:border-red-900 dark:bg-red-950">
            <AlertCircle
              size={18}
              className="mt-0.5 flex-shrink-0 text-red-600 dark:text-red-400"
            />
            <div>
              <p className="text-sm font-medium text-red-600 dark:text-red-400">Validation Error</p>
              <p className="mt-1 text-xs text-red-600 dark:text-red-400">{error}</p>
            </div>
          </div>
        ) : yaml ? (
          <pre className="overflow-x-auto rounded-lg border border-zinc-200 bg-white p-3 text-xs dark:border-zinc-700 dark:bg-zinc-950">
            <code className="text-zinc-900 dark:text-white">{yaml}</code>
          </pre>
        ) : (
          <div className="flex h-full items-center justify-center text-sm text-zinc-500 dark:text-zinc-400">
            Fill in the form to see the generated YAML
          </div>
        )}
      </div>

      {/* Footer hint */}
      {!error && yaml && (
        <div className="border-t border-zinc-200 px-4 py-2 dark:border-zinc-800">
          <p className="text-xs text-zinc-500 dark:text-zinc-400">
            This YAML will be submitted when you deploy
          </p>
        </div>
      )}
    </div>
  );
}
