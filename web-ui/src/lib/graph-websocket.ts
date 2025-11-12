/**
 * WebSocket connection manager for real-time graph updates
 * Shared across all graph implementations
 */

import type { GraphNode, GraphEdge } from './api';

export interface GraphUpdate {
  type: 'full' | 'node' | 'edge' | 'status';
  nodes?: GraphNode[];
  edges?: GraphEdge[];
  node?: Partial<GraphNode>;
  nodeId?: string;
  status?: string;
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

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const host = window.location.host;
    const wsUrl = `${protocol}//${host}/api/graph/${encodeURIComponent(this.appName)}/ws`;

    try {
      this.ws = new WebSocket(wsUrl);

      this.ws.onopen = () => {
        console.log(`[GraphWebSocket] Connected to ${this.appName}`);
        this.reconnectAttempts = 0;
        this.startPing();
      };

      this.ws.onmessage = (event) => {
        try {
          const update: GraphUpdate = JSON.parse(event.data);
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
