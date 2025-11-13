'use client';

import React, { useState } from 'react';
import Link from 'next/link';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import {
  X,
  Database,
  Server,
  Clock,
  ExternalLink,
  FileCode,
  Settings,
  GitBranch,
  Download,
  Link2,
  Terminal,
  Lock,
  Globe,
  Copy,
  CheckCheck,
  Check,
  Activity,
  AlertCircle,
  CheckCircle2,
} from 'lucide-react';
import { ResourceInstance } from '@/lib/api';

// ============================================================================
// Types & Utilities
// ============================================================================

interface ResourceDetailsPaneProps {
  resource: ResourceInstance | null;
  onClose: () => void;
}

type BadgeVariant = 'default' | 'secondary' | 'destructive' | 'outline';

interface ResourceHint {
  label: string;
  value: string;
  type?: string;
  icon?: string;
}

const STATE_CONFIG = {
  active: {
    badgeVariant: 'default' as BadgeVariant,
    icon: <CheckCircle2 className="w-4 h-4 text-green-500" />,
    color: 'text-green-600 dark:text-green-400',
  },
  provisioning: {
    badgeVariant: 'secondary' as BadgeVariant,
    icon: <Clock className="w-4 h-4 text-blue-500" />,
    color: 'text-blue-600 dark:text-blue-400',
  },
  scaling: {
    badgeVariant: 'secondary' as BadgeVariant,
    icon: <Activity className="w-4 h-4 text-blue-500" />,
    color: 'text-blue-600 dark:text-blue-400',
  },
  updating: {
    badgeVariant: 'secondary' as BadgeVariant,
    icon: <Activity className="w-4 h-4 text-blue-500" />,
    color: 'text-blue-600 dark:text-blue-400',
  },
  failed: {
    badgeVariant: 'destructive' as BadgeVariant,
    icon: <AlertCircle className="w-4 h-4 text-red-500" />,
    color: 'text-red-600 dark:text-red-400',
  },
  degraded: {
    badgeVariant: 'destructive' as BadgeVariant,
    icon: <AlertCircle className="w-4 h-4 text-orange-500" />,
    color: 'text-orange-600 dark:text-orange-400',
  },
} as const;

const HEALTH_CONFIG = {
  healthy: { color: 'text-green-600 dark:text-green-400', dotColor: 'bg-green-500' },
  ok: { color: 'text-green-600 dark:text-green-400', dotColor: 'bg-green-500' },
  degraded: { color: 'text-yellow-600 dark:text-yellow-400', dotColor: 'bg-yellow-500' },
  warning: { color: 'text-yellow-600 dark:text-yellow-400', dotColor: 'bg-yellow-500' },
  unhealthy: { color: 'text-red-600 dark:text-red-400', dotColor: 'bg-red-500' },
  critical: { color: 'text-red-600 dark:text-red-400', dotColor: 'bg-red-500' },
} as const;

const getStateConfig = (state: string) => {
  return STATE_CONFIG[state as keyof typeof STATE_CONFIG] || STATE_CONFIG.provisioning;
};

const getHealthConfig = (health: string) => {
  const key = health?.toLowerCase() as keyof typeof HEALTH_CONFIG;
  return (
    HEALTH_CONFIG[key] || { color: 'text-gray-600 dark:text-gray-400', dotColor: 'bg-gray-400' }
  );
};

const getHintIcon = (icon?: string) => {
  const iconClass = 'h-5 w-5 text-gray-500 dark:text-gray-400';
  const icons: Record<string, JSX.Element> = {
    'git-branch': <GitBranch className={iconClass} />,
    download: <Download className={iconClass} />,
    settings: <Settings className={iconClass} />,
    terminal: <Terminal className={iconClass} />,
    database: <Database className={iconClass} />,
    lock: <Lock className={iconClass} />,
    'external-link': <ExternalLink className={iconClass} />,
    globe: <Globe className={iconClass} />,
  };
  return icons[icon || ''] || <Link2 className={iconClass} />;
};

// ============================================================================
// Sub-Components
// ============================================================================

interface ResourceHeaderProps {
  resource: ResourceInstance;
  onClose: () => void;
}

function ResourceHeader({ resource, onClose }: ResourceHeaderProps) {
  const stateConfig = getStateConfig(resource.state);

  return (
    <div className="flex items-center justify-between p-4 border-b">
      <div className="flex-1">
        <div className="flex items-center gap-2 mb-2">
          {stateConfig.icon}
          <h3 className="text-lg font-semibold">{resource.resource_name}</h3>
          <Badge variant={stateConfig.badgeVariant}>{resource.state}</Badge>
        </div>
        <p className="text-sm text-muted-foreground mb-1">Type: {resource.resource_type}</p>
        <Link
          href={`/graph/${resource.application_name}`}
          className="text-sm text-blue-600 hover:text-blue-800 dark:text-blue-400 dark:hover:text-blue-300 flex items-center gap-1 w-fit"
        >
          Application: {resource.application_name}
          <ExternalLink className="w-3 h-3" />
        </Link>
      </div>
      <Button variant="ghost" size="sm" onClick={onClose}>
        <X className="w-4 h-4" />
      </Button>
    </div>
  );
}

interface QuickAccessCardProps {
  hint: ResourceHint;
  onHintClick: (hint: ResourceHint) => void;
  isCopied: boolean;
}

function QuickAccessCard({ hint, onHintClick, isCopied }: QuickAccessCardProps) {
  const isUrl = hint.type === 'url' || hint.type === 'dashboard';

  return (
    <Card
      className="relative overflow-hidden border shadow-sm bg-white dark:bg-gray-800 hover:shadow-md transition-shadow cursor-pointer"
      onClick={() => onHintClick(hint)}
    >
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2 relative z-10">
        <CardTitle className="text-sm font-medium text-gray-700 dark:text-gray-300">
          {hint.label}
        </CardTitle>
        {getHintIcon(hint.icon)}
      </CardHeader>
      <CardContent className="relative z-10">
        <div className="text-xs font-mono text-gray-600 dark:text-gray-400 break-all pr-8">
          {hint.value}
        </div>
        {isCopied && (
          <div className="absolute top-2 right-2 flex items-center gap-1 bg-green-100 dark:bg-green-900 text-green-800 dark:text-green-200 px-2 py-1 rounded text-xs">
            <CheckCheck className="w-3 h-3" />
            Copied!
          </div>
        )}
        <div className="absolute bottom-2 right-2">
          {isUrl ? (
            <ExternalLink className="w-3 h-3 text-gray-400" />
          ) : (
            <Copy className="w-3 h-3 text-gray-400" />
          )}
        </div>
      </CardContent>
    </Card>
  );
}

interface ResourceInfoCardProps {
  resource: ResourceInstance;
}

function ResourceInfoCard({ resource }: ResourceInfoCardProps) {
  const stateConfig = getStateConfig(resource.state);
  const healthConfig = getHealthConfig(resource.health_status);

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base flex items-center gap-2">
          <Database className="w-4 h-4" />
          Resource Information
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        <div className="grid grid-cols-2 gap-3 text-sm">
          <div className="text-muted-foreground">ID:</div>
          <div className="font-mono text-xs break-all">{resource.id}</div>

          <div className="text-muted-foreground">Name:</div>
          <div className="font-medium">{resource.resource_name}</div>

          <div className="text-muted-foreground">Type:</div>
          <div>{resource.resource_type}</div>

          <div className="text-muted-foreground">Application:</div>
          <div>
            <Link
              href={`/graph/${resource.application_name}`}
              className="text-blue-600 hover:text-blue-800 dark:text-blue-400 dark:hover:text-blue-300 flex items-center gap-1"
            >
              {resource.application_name}
              <ExternalLink className="w-3 h-3" />
            </Link>
          </div>

          <div className="text-muted-foreground">State:</div>
          <div className="flex items-center gap-2">
            {stateConfig.icon}
            <Badge variant={stateConfig.badgeVariant}>{resource.state}</Badge>
          </div>

          <div className="text-muted-foreground">Health Status:</div>
          <div className="flex items-center gap-2">
            <div className={`w-2 h-2 rounded-full ${healthConfig.dotColor}`} />
            <span className={healthConfig.color}>{resource.health_status || 'Unknown'}</span>
          </div>

          {resource.workflow_execution_id && (
            <>
              <div className="text-muted-foreground">Workflow ID:</div>
              <div>
                <Link
                  href={`/workflows?id=${resource.workflow_execution_id}`}
                  className="text-blue-600 hover:text-blue-800 dark:text-blue-400 dark:hover:text-blue-300 font-mono text-xs"
                >
                  #{resource.workflow_execution_id}
                </Link>
              </div>
            </>
          )}

          {resource.provider_id && (
            <>
              <div className="text-muted-foreground">Provider:</div>
              <div className="font-mono text-xs break-all">{resource.provider_id}</div>
            </>
          )}
        </div>

        {resource.error_message && (
          <div className="mt-3 p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded">
            <p className="text-sm font-medium text-red-800 dark:text-red-200 mb-1 flex items-center gap-2">
              <AlertCircle className="w-4 h-4" />
              Error Message
            </p>
            <p className="text-sm text-red-600 dark:text-red-300">{resource.error_message}</p>
          </div>
        )}
      </CardContent>
    </Card>
  );
}

interface ConfigurationCardProps {
  configuration: Record<string, any>;
  onCopy: () => void;
  isCopied: boolean;
}

function ConfigurationCard({ configuration, onCopy, isCopied }: ConfigurationCardProps) {
  const hasConfig = configuration && Object.keys(configuration).length > 0;

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <CardTitle className="text-base flex items-center gap-2">
            <Settings className="w-4 h-4" />
            Resource Configuration
          </CardTitle>
          {hasConfig && (
            <Button variant="ghost" size="sm" onClick={onCopy} className="h-7 text-xs">
              {isCopied ? (
                <>
                  <Check className="w-3 h-3 mr-1 text-green-500" />
                  Copied
                </>
              ) : (
                <>
                  <Copy className="w-3 h-3 mr-1" />
                  Copy
                </>
              )}
            </Button>
          )}
        </div>
      </CardHeader>
      <CardContent>
        {hasConfig ? (
          <pre className="bg-gray-100 dark:bg-gray-900 p-3 rounded text-xs overflow-x-auto">
            <code>{JSON.stringify(configuration, null, 2)}</code>
          </pre>
        ) : (
          <p className="text-sm text-muted-foreground">No configuration data available</p>
        )}
      </CardContent>
    </Card>
  );
}

interface TimelineCardProps {
  resource: ResourceInstance;
}

function TimelineCard({ resource }: TimelineCardProps) {
  const healthConfig = getHealthConfig(resource.health_status);

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base flex items-center gap-2">
          <Clock className="w-4 h-4" />
          Resource Timeline
        </CardTitle>
      </CardHeader>
      <CardContent>
        <div className="space-y-3">
          <div className="flex items-start gap-3 pb-3 border-b">
            <div className="w-2 h-2 rounded-full bg-green-500 mt-1.5 flex-shrink-0" />
            <div className="flex-1 min-w-0">
              <p className="text-sm font-medium">Created</p>
              <p className="text-xs text-muted-foreground">
                {new Date(resource.created_at).toLocaleString()}
              </p>
            </div>
          </div>

          <div className="flex items-start gap-3 pb-3 border-b">
            <div className="w-2 h-2 rounded-full bg-blue-500 mt-1.5 flex-shrink-0" />
            <div className="flex-1 min-w-0">
              <p className="text-sm font-medium">Last Updated</p>
              <p className="text-xs text-muted-foreground">
                {new Date(resource.updated_at).toLocaleString()}
              </p>
            </div>
          </div>

          {resource.last_health_check && (
            <div className="flex items-start gap-3 pb-3 border-b">
              <div
                className={`w-2 h-2 rounded-full mt-1.5 flex-shrink-0 ${healthConfig.dotColor}`}
              />
              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium">Last Health Check</p>
                <p className="text-xs text-muted-foreground">
                  {new Date(resource.last_health_check).toLocaleString()}
                </p>
                <p className={`text-xs font-medium mt-1 ${healthConfig.color}`}>
                  Status: {resource.health_status}
                </p>
              </div>
            </div>
          )}

          {resource.workflow_execution_id && (
            <div className="flex items-start gap-3">
              <div className="w-2 h-2 rounded-full bg-purple-500 mt-1.5 flex-shrink-0" />
              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium">Related Workflow</p>
                <Link
                  href={`/workflows?id=${resource.workflow_execution_id}`}
                  className="text-xs text-blue-600 hover:text-blue-800 dark:text-blue-400 dark:hover:text-blue-300 flex items-center gap-1"
                >
                  Workflow #{resource.workflow_execution_id}
                  <ExternalLink className="w-3 h-3" />
                </Link>
              </div>
            </div>
          )}
        </div>
      </CardContent>
    </Card>
  );
}

// ============================================================================
// Main Component
// ============================================================================

export function ResourceDetailsPane({ resource, onClose }: ResourceDetailsPaneProps) {
  const [copiedHint, setCopiedHint] = useState<string | null>(null);
  const [copiedConfig, setCopiedConfig] = useState(false);

  if (!resource) {
    return null;
  }

  const handleHintClick = (hint: ResourceHint) => {
    if (hint.type === 'url' || hint.type === 'dashboard') {
      window.open(hint.value, '_blank', 'noopener,noreferrer');
    } else {
      copyToClipboard(hint.value, hint.label);
    }
  };

  const copyToClipboard = async (value: string, label: string) => {
    try {
      await navigator.clipboard.writeText(value);
      setCopiedHint(label);
      setTimeout(() => setCopiedHint(null), 2000);
    } catch (err) {
      console.error('Failed to copy:', err);
    }
  };

  const copyConfiguration = async () => {
    try {
      const configText = resource.configuration
        ? JSON.stringify(resource.configuration, null, 2)
        : 'No configuration available';
      await navigator.clipboard.writeText(configText);
      setCopiedConfig(true);
      setTimeout(() => setCopiedConfig(false), 2000);
    } catch (err) {
      console.error('Failed to copy:', err);
    }
  };

  return (
    <div className="w-full h-full flex flex-col bg-white dark:bg-gray-800 border-l">
      <ResourceHeader resource={resource} onClose={onClose} />

      <div className="flex-1 overflow-auto p-4">
        <Tabs defaultValue="overview" className="w-full">
          <TabsList className="grid w-full grid-cols-4">
            <TabsTrigger value="overview">Overview</TabsTrigger>
            <TabsTrigger value="configuration">Config</TabsTrigger>
            <TabsTrigger value="provider">Provider</TabsTrigger>
            <TabsTrigger value="history">History</TabsTrigger>
          </TabsList>

          {/* Overview Tab */}
          <TabsContent value="overview" className="space-y-4 mt-4">
            {/* Quick Access Section */}
            {resource.hints && resource.hints.length > 0 && (
              <div className="space-y-3">
                <h3 className="text-sm font-semibold text-gray-700 dark:text-gray-300 flex items-center gap-2">
                  <Link2 className="w-4 h-4" />
                  Quick Access
                </h3>
                <div className="grid grid-cols-1 gap-3">
                  {resource.hints.map((hint, index) => (
                    <QuickAccessCard
                      key={index}
                      hint={hint}
                      onHintClick={handleHintClick}
                      isCopied={copiedHint === hint.label}
                    />
                  ))}
                </div>
              </div>
            )}

            <ResourceInfoCard resource={resource} />
          </TabsContent>

          {/* Configuration Tab */}
          <TabsContent value="configuration" className="space-y-4 mt-4">
            <ConfigurationCard
              configuration={resource.configuration}
              onCopy={copyConfiguration}
              isCopied={copiedConfig}
            />
          </TabsContent>

          {/* Provider Tab */}
          <TabsContent value="provider" className="space-y-4 mt-4">
            <Card>
              <CardHeader>
                <CardTitle className="text-base flex items-center gap-2">
                  <Server className="w-4 h-4" />
                  Provider Information
                </CardTitle>
              </CardHeader>
              <CardContent className="space-y-3">
                {resource.provider_id && (
                  <div>
                    <p className="text-sm font-medium mb-1">Provider ID</p>
                    <p className="text-xs font-mono bg-gray-100 dark:bg-gray-900 p-2 rounded break-all">
                      {resource.provider_id}
                    </p>
                  </div>
                )}

                {resource.provider_metadata &&
                Object.keys(resource.provider_metadata).length > 0 ? (
                  <div>
                    <p className="text-sm font-medium mb-2">Provider Metadata</p>
                    <pre className="bg-gray-100 dark:bg-gray-900 p-3 rounded text-xs overflow-x-auto">
                      <code>{JSON.stringify(resource.provider_metadata, null, 2)}</code>
                    </pre>
                  </div>
                ) : (
                  <p className="text-sm text-muted-foreground">No provider metadata available</p>
                )}
              </CardContent>
            </Card>
          </TabsContent>

          {/* History Tab */}
          <TabsContent value="history" className="space-y-4 mt-4">
            <TimelineCard resource={resource} />
          </TabsContent>
        </Tabs>
      </div>
    </div>
  );
}
