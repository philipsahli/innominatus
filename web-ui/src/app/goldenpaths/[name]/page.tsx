import { notFound } from 'next/navigation';
import {
  getAllGoldenPaths,
  getGoldenPathByName,
  getGitHubWorkflowUrl,
  getGitHubGoldenPathConfigUrl,
} from '@/lib/goldenpaths';
import { parseWorkflowFile } from '@/lib/workflow-parser';
import { GoldenPathDetailClient } from '@/components/goldenpath-detail-client';
import { ProtectedRoute } from '@/components/protected-route';

interface PageProps {
  params: Promise<{ name: string }>;
}

// Generate static params for all golden paths
export async function generateStaticParams() {
  const goldenPaths = getAllGoldenPaths();
  return goldenPaths.map((path) => ({
    name: path.name,
  }));
}

export default async function GoldenPathDetailPage({ params }: PageProps) {
  const { name } = await params;
  const goldenPath = getGoldenPathByName(name);

  if (!goldenPath) {
    notFound();
  }

  const workflowUrl = getGitHubWorkflowUrl(goldenPath.workflow);
  const configUrl = getGitHubGoldenPathConfigUrl(goldenPath.name);

  // Parse workflow file to get steps
  const workflowDefinition = await parseWorkflowFile(goldenPath.workflow);
  const workflowSteps = workflowDefinition?.spec?.steps || [];

  return (
    <ProtectedRoute>
      <GoldenPathDetailClient
        goldenPath={goldenPath}
        workflowUrl={workflowUrl}
        configUrl={configUrl}
        workflowSteps={workflowSteps}
      />
    </ProtectedRoute>
  );
}
