'use client';

import React from 'react';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { GraphVisualization } from '@/components/graph-visualization';
import { Button } from '@/components/ui/button';
import { ExternalLink } from 'lucide-react';
import { useRouter } from 'next/navigation';

interface GraphModalProps {
  appName: string;
  isOpen: boolean;
  onClose: () => void;
}

export function GraphModal({ appName, isOpen, onClose }: GraphModalProps) {
  const router = useRouter();

  const handleOpenFullPage = () => {
    onClose();
    router.push(`/graph/${appName}`);
  };

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="max-w-[95vw] max-h-[95vh] h-[90vh] p-0">
        <DialogHeader className="p-6 pb-4">
          <div className="flex items-center justify-between">
            <div>
              <DialogTitle className="text-2xl">Workflow Graph</DialogTitle>
              <DialogDescription className="mt-1">
                Real-time visualization for <span className="font-semibold">{appName}</span>
              </DialogDescription>
            </div>
            <Button variant="outline" size="sm" onClick={handleOpenFullPage} className="gap-2">
              <ExternalLink className="w-4 h-4" />
              Open Full Page
            </Button>
          </div>
        </DialogHeader>
        <div className="flex-1 overflow-hidden px-6 pb-6">
          <div className="h-full border rounded-lg overflow-hidden bg-white dark:bg-gray-900">
            <GraphVisualization app={appName} />
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
