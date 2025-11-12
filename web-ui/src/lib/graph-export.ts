/**
 * Graph export utilities for saving graph visualizations
 * Supports PNG, SVG, and JSON formats
 */

import type { GraphNode, GraphEdge } from './api';

/**
 * Export graph data to JSON file
 */
export function exportGraphJSON(
  nodes: GraphNode[],
  edges: GraphEdge[],
  filename: string = 'graph.json'
): void {
  const data = {
    nodes,
    edges,
    exportedAt: new Date().toISOString(),
  };

  const blob = new Blob([JSON.stringify(data, null, 2)], {
    type: 'application/json',
  });
  downloadBlob(blob, filename);
}

/**
 * Export SVG element to SVG file
 */
export function exportSVG(svgElement: SVGSVGElement, filename: string = 'graph.svg'): void {
  const serializer = new XMLSerializer();
  const svgString = serializer.serializeToString(svgElement);

  const blob = new Blob([svgString], {
    type: 'image/svg+xml;charset=utf-8',
  });
  downloadBlob(blob, filename);
}

/**
 * Export SVG element to PNG file
 */
export function exportPNG(
  svgElement: SVGSVGElement,
  filename: string = 'graph.png',
  scale: number = 2
): Promise<void> {
  return new Promise((resolve, reject) => {
    try {
      const canvas = document.createElement('canvas');
      const ctx = canvas.getContext('2d');
      if (!ctx) {
        throw new Error('Failed to get canvas context');
      }

      const serializer = new XMLSerializer();
      const svgString = serializer.serializeToString(svgElement);

      const img = new Image();
      const svgBlob = new Blob([svgString], {
        type: 'image/svg+xml;charset=utf-8',
      });
      const url = URL.createObjectURL(svgBlob);

      img.onload = () => {
        canvas.width = img.width * scale;
        canvas.height = img.height * scale;
        ctx.scale(scale, scale);
        ctx.drawImage(img, 0, 0);

        canvas.toBlob((blob) => {
          if (blob) {
            downloadBlob(blob, filename);
            URL.revokeObjectURL(url);
            resolve();
          } else {
            reject(new Error('Failed to create blob'));
          }
        }, 'image/png');
      };

      img.onerror = () => {
        URL.revokeObjectURL(url);
        reject(new Error('Failed to load SVG image'));
      };

      img.src = url;
    } catch (error) {
      reject(error);
    }
  });
}

/**
 * Export canvas element to PNG file
 * Used by D3 canvas-based implementations
 */
export function exportCanvasPNG(canvas: HTMLCanvasElement, filename: string = 'graph.png'): void {
  canvas.toBlob((blob) => {
    if (blob) {
      downloadBlob(blob, filename);
    }
  }, 'image/png');
}

/**
 * Helper function to download a blob
 */
function downloadBlob(blob: Blob, filename: string): void {
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  URL.revokeObjectURL(url);
}

/**
 * Copy graph data to clipboard as JSON
 */
export async function copyGraphToClipboard(nodes: GraphNode[], edges: GraphEdge[]): Promise<void> {
  const data = JSON.stringify({ nodes, edges }, null, 2);

  if (navigator.clipboard && navigator.clipboard.writeText) {
    await navigator.clipboard.writeText(data);
  } else {
    // Fallback for older browsers
    const textArea = document.createElement('textarea');
    textArea.value = data;
    textArea.style.position = 'fixed';
    textArea.style.left = '-999999px';
    document.body.appendChild(textArea);
    textArea.select();
    document.execCommand('copy');
    document.body.removeChild(textArea);
  }
}

/**
 * Export graph statistics
 */
export function exportGraphStats(
  nodes: GraphNode[],
  edges: GraphEdge[],
  filename: string = 'graph-stats.json'
): void {
  // Count by type
  const nodesByType = nodes.reduce(
    (acc, node) => {
      acc[node.type] = (acc[node.type] || 0) + 1;
      return acc;
    },
    {} as Record<string, number>
  );

  // Count by status
  const nodesByStatus = nodes.reduce(
    (acc, node) => {
      const status = node.status || 'unknown';
      acc[status] = (acc[status] || 0) + 1;
      return acc;
    },
    {} as Record<string, number>
  );

  // Count edges by relationship
  const edgesByRelationship = edges.reduce(
    (acc, edge) => {
      acc[edge.relationship] = (acc[edge.relationship] || 0) + 1;
      return acc;
    },
    {} as Record<string, number>
  );

  const stats = {
    totalNodes: nodes.length,
    totalEdges: edges.length,
    nodesByType,
    nodesByStatus,
    edgesByRelationship,
    exportedAt: new Date().toISOString(),
  };

  const blob = new Blob([JSON.stringify(stats, null, 2)], {
    type: 'application/json',
  });
  downloadBlob(blob, filename);
}
