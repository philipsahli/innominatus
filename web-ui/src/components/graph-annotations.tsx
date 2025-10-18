'use client';

import React, { useState, useEffect } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { MessageSquare, Plus, Trash2, X } from 'lucide-react';

interface Annotation {
  id: number;
  node_id: string;
  node_name: string;
  annotation_text: string;
  created_by: string;
  created_at: string;
  updated_at: string;
}

interface GraphAnnotationsProps {
  app: string;
  selectedNodeId?: string;
  selectedNodeName?: string;
  onClose: () => void;
}

export function GraphAnnotations({
  app,
  selectedNodeId,
  selectedNodeName,
  onClose,
}: GraphAnnotationsProps) {
  const [annotations, setAnnotations] = useState<Annotation[]>([]);
  const [loading, setLoading] = useState(true);
  const [showAddForm, setShowAddForm] = useState(false);
  const [newAnnotation, setNewAnnotation] = useState('');
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetchAnnotations();
  }, [app, selectedNodeId]);

  const fetchAnnotations = async () => {
    setLoading(true);
    setError(null);

    try {
      const apiKey = localStorage.getItem('api_key') || '';
      const url = selectedNodeId
        ? `/api/graph/${app}/annotations?node_id=${selectedNodeId}`
        : `/api/graph/${app}/annotations`;

      const response = await fetch(url, {
        headers: {
          Authorization: `Bearer ${apiKey}`,
        },
      });

      if (!response.ok) {
        throw new Error(`Failed to fetch annotations: ${response.statusText}`);
      }

      const data = await response.json();
      setAnnotations(data.annotations || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
    } finally {
      setLoading(false);
    }
  };

  const handleAddAnnotation = async () => {
    if (!newAnnotation.trim()) {
      alert('Please enter annotation text');
      return;
    }

    if (!selectedNodeId) {
      alert('Please select a node first');
      return;
    }

    try {
      const apiKey = localStorage.getItem('api_key') || '';
      const response = await fetch(`/api/graph/${app}/annotations`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${apiKey}`,
        },
        body: JSON.stringify({
          node_id: selectedNodeId,
          node_name: selectedNodeName || selectedNodeId,
          annotation_text: newAnnotation,
        }),
      });

      if (!response.ok) {
        throw new Error(`Failed to create annotation: ${response.statusText}`);
      }

      // Refresh annotations and reset form
      await fetchAnnotations();
      setNewAnnotation('');
      setShowAddForm(false);
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Failed to create annotation');
    }
  };

  const handleDeleteAnnotation = async (id: number) => {
    if (!confirm('Are you sure you want to delete this annotation?')) {
      return;
    }

    try {
      const apiKey = localStorage.getItem('api_key') || '';
      const response = await fetch(`/api/graph/${app}/annotations?id=${id}`, {
        method: 'DELETE',
        headers: {
          Authorization: `Bearer ${apiKey}`,
        },
      });

      if (!response.ok) {
        throw new Error(`Failed to delete annotation: ${response.statusText}`);
      }

      // Refresh annotations
      await fetchAnnotations();
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Failed to delete annotation');
    }
  };

  const formatDate = (dateStr: string) => {
    const date = new Date(dateStr);
    return date.toLocaleString();
  };

  if (loading) {
    return (
      <Card className="w-full">
        <CardHeader>
          <CardTitle className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <MessageSquare className="h-5 w-5" />
              Annotations
            </div>
            <Button variant="ghost" size="sm" onClick={onClose}>
              <X className="h-4 w-4" />
            </Button>
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="text-center py-8 text-gray-500">Loading annotations...</div>
        </CardContent>
      </Card>
    );
  }

  if (error) {
    return (
      <Card className="w-full">
        <CardHeader>
          <CardTitle className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <MessageSquare className="h-5 w-5" />
              Annotations
            </div>
            <Button variant="ghost" size="sm" onClick={onClose}>
              <X className="h-4 w-4" />
            </Button>
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="text-center py-8 text-red-500">Error: {error}</div>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card className="w-full">
      <CardHeader>
        <CardTitle className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <MessageSquare className="h-5 w-5" />
            Annotations
            {selectedNodeId && (
              <Badge variant="secondary" className="ml-2">
                {selectedNodeName || selectedNodeId}
              </Badge>
            )}
          </div>
          <div className="flex gap-2">
            {selectedNodeId && (
              <Button
                variant="outline"
                size="sm"
                onClick={() => setShowAddForm(!showAddForm)}
                className="gap-2"
              >
                <Plus className="h-4 w-4" />
                Add Note
              </Button>
            )}
            <Button variant="ghost" size="sm" onClick={onClose}>
              <X className="h-4 w-4" />
            </Button>
          </div>
        </CardTitle>
      </CardHeader>
      <CardContent>
        <div className="space-y-4">
          {/* Add annotation form */}
          {showAddForm && selectedNodeId && (
            <div className="p-4 border rounded-lg bg-gray-50">
              <textarea
                className="w-full p-2 border rounded-lg resize-none"
                rows={3}
                placeholder="Enter your annotation..."
                value={newAnnotation}
                onChange={(e) => setNewAnnotation(e.target.value)}
              />
              <div className="flex gap-2 mt-2">
                <Button size="sm" onClick={handleAddAnnotation}>
                  Save
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => {
                    setShowAddForm(false);
                    setNewAnnotation('');
                  }}
                >
                  Cancel
                </Button>
              </div>
            </div>
          )}

          {/* Annotations list */}
          <div className="space-y-2 max-h-96 overflow-y-auto">
            {annotations.length === 0 ? (
              <div className="text-center py-8 text-gray-500">
                {selectedNodeId
                  ? 'No annotations for this node. Click "Add Note" to create one.'
                  : 'No annotations yet. Select a node to add one.'}
              </div>
            ) : (
              annotations.map((annotation) => (
                <div key={annotation.id} className="p-4 border rounded-lg hover:bg-gray-50">
                  <div className="flex items-start justify-between">
                    <div className="flex-1">
                      <div className="flex items-center gap-2 mb-2">
                        <Badge className="bg-blue-500">{annotation.node_name}</Badge>
                        <span className="text-xs text-gray-500">by {annotation.created_by}</span>
                        <span className="text-xs text-gray-400">
                          {formatDate(annotation.created_at)}
                        </span>
                      </div>
                      <p className="text-sm text-gray-700 whitespace-pre-wrap">
                        {annotation.annotation_text}
                      </p>
                    </div>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => handleDeleteAnnotation(annotation.id)}
                      className="ml-2"
                    >
                      <Trash2 className="h-4 w-4 text-red-500" />
                    </Button>
                  </div>
                </div>
              ))
            )}
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
