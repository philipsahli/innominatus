/**
 * WebSocket connection manager for real-time graph updates
 * Shared across all graph implementations
 */

import type { GraphNode, GraphEdge } from './api';
import { getWebSocketAuthToken } from './api';

// Event from the backend when graph changes occur
export interface GraphEvent {
  type: 'node_added' | 'node_state_changed' | 'node_updated' | 'edge_added' | 'graph_updated';
  timestamp: string;
  node_id?: string;
  node_name?: string;
  node_type?: string;
  old_state?: string;
  new_state?: string;
  edge_id?: string;
  edge_type?: string;
  from_node?: string;
  to_node?: string;
  from_id?: string;
  to_id?: string;
  node_count?: number;
  edge_count?: number;
  metadata?: Record<string, unknown>;
}

// Message received from WebSocket
export interface GraphWebSocketMessage {
  graph: {
    app_name: string;
    nodes: GraphNode[];
    edges: GraphEdge[];
    created_at: string;
    updated_at: string;
  };
  event?: GraphEvent;
}

// Legacy format for backwards compatibility
export interface GraphUpdate {
  type: 'full' | 'node' | 'edge' | 'status';
  nodes?: GraphNode[];
  edges?: GraphEdge[];
  node?: Partial<GraphNode>;
  nodeId?: string;
  status?: string;
  event?: GraphEvent; // New field for activity feed
}

export type GraphUpdateCallback = (update: GraphUpdate) => void;

export class GraphWebSocket {
  private ws: WebSocket | null = null;
  private callbacks: Set<GraphUpdateCallback> = new Set();
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private reconnectDelay = 1000;
  private pingInterval: NodeJS.Timeout | null = null;
  private appName: string;

  constructor(appName: string) {
    this.appName = appName;
  }

  /**
   * Connect to WebSocket
   */
  connect(): void {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      return;
    }

    // Detect Next.js dev server (port 3000) vs Go server (port 8081+)
    // Port 3000 = Next.js dev server → connect to Go server on 8081
    // Port 8081 (or other) = Go server → connect to same host
    const isNextDevServer = typeof window !== 'undefined' && window.location.port === '3000';
    const protocol =
      typeof window !== 'undefined' && window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const host = isNextDevServer
      ? 'localhost:8081'
      : typeof window !== 'undefined'
        ? window.location.host
        : 'localhost:8081';

    // Get authentication token for WebSocket connection
    const token = getWebSocketAuthToken();
    const tokenParam = token ? `?token=${encodeURIComponent(token)}` : '';
    const wsUrl = `${protocol}//${host}/api/graph/${encodeURIComponent(this.appName)}/ws${tokenParam}`;

    try {
      this.ws = new WebSocket(wsUrl);

      this.ws.onopen = () => {
        console.log(`[GraphWebSocket] Connected to ${this.appName}`);
        this.reconnectAttempts = 0;
        this.startPing();
      };

      this.ws.onmessage = (event) => {
        try {
          const message: GraphWebSocketMessage = JSON.parse(event.data);

          // Convert new format to legacy GraphUpdate format for backwards compatibility
          const update: GraphUpdate = {
            type: 'full',
            nodes: message.graph.nodes,
            edges: message.graph.edges,
            event: message.event, // Pass through event for activity feed
          };

          this.notifyCallbacks(update);
        } catch (error) {
          console.error('[GraphWebSocket] Failed to parse message:', error);
        }
      };

      this.ws.onerror = (error) => {
        console.error('[GraphWebSocket] Error:', error);
      };

      this.ws.onclose = () => {
        console.log('[GraphWebSocket] Connection closed');
        this.stopPing();
        this.attemptReconnect();
      };
    } catch (error) {
      console.error('[GraphWebSocket] Failed to create connection:', error);
      this.attemptReconnect();
    }
  }

  /**
   * Disconnect from WebSocket
   */
  disconnect(): void {
    this.maxReconnectAttempts = 0; // Prevent reconnection
    this.stopPing();

    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
  }

  /**
   * Subscribe to graph updates
   */
  subscribe(callback: GraphUpdateCallback): () => void {
    this.callbacks.add(callback);

    // Return unsubscribe function
    return () => {
      this.callbacks.delete(callback);
    };
  }

  /**
   * Check if connected
   */
  isConnected(): boolean {
    return this.ws !== null && this.ws.readyState === WebSocket.OPEN;
  }

  /**
   * Notify all subscribers of an update
   */
  private notifyCallbacks(update: GraphUpdate): void {
    this.callbacks.forEach((callback) => {
      try {
        callback(update);
      } catch (error) {
        console.error('[GraphWebSocket] Callback error:', error);
      }
    });
  }

  /**
   * Attempt to reconnect with exponential backoff
   */
  private attemptReconnect(): void {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.error('[GraphWebSocket] Max reconnect attempts reached');
      return;
    }

    this.reconnectAttempts++;
    const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1);

    console.log(
      `[GraphWebSocket] Reconnecting in ${delay}ms (attempt ${this.reconnectAttempts}/${this.maxReconnectAttempts})`
    );

    setTimeout(() => {
      this.connect();
    }, delay);
  }

  /**
   * Start sending ping messages to keep connection alive
   */
  private startPing(): void {
    this.stopPing();

    this.pingInterval = setInterval(() => {
      if (this.ws && this.ws.readyState === WebSocket.OPEN) {
        try {
          this.ws.send(JSON.stringify({ type: 'ping' }));
        } catch (error) {
          console.error('[GraphWebSocket] Failed to send ping:', error);
        }
      }
    }, 30000); // Ping every 30 seconds
  }

  /**
   * Stop ping interval
   */
  private stopPing(): void {
    if (this.pingInterval) {
      clearInterval(this.pingInterval);
      this.pingInterval = null;
    }
  }
}

/**
 * React hook for using GraphWebSocket
 * Returns connection status and update handler
 */
export function useGraphWebSocket(
  appName: string,
  onUpdate: GraphUpdateCallback,
  enabled: boolean = true
): {
  isConnected: boolean;
  reconnect: () => void;
} {
  const [wsClient] = useState(() => new GraphWebSocket(appName));
  const [isConnected, setIsConnected] = useState(false);

  useEffect(() => {
    if (!enabled) return;

    // Subscribe to updates
    const unsubscribe = wsClient.subscribe((update) => {
      onUpdate(update);
      setIsConnected(wsClient.isConnected());
    });

    // Connect
    wsClient.connect();

    // Check connection status periodically
    const statusCheck = setInterval(() => {
      setIsConnected(wsClient.isConnected());
    }, 1000);

    // Cleanup
    return () => {
      unsubscribe();
      clearInterval(statusCheck);
      wsClient.disconnect();
    };
  }, [appName, onUpdate, enabled, wsClient]);

  const reconnect = useCallback(() => {
    wsClient.disconnect();
    wsClient.connect();
  }, [wsClient]);

  return { isConnected, reconnect };
}

// Import React hooks
import { useState, useEffect, useCallback } from 'react';
