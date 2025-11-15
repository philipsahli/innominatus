'use client';

import { useState, useEffect, useRef } from 'react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';

interface WebSocketMessage {
  timestamp: string;
  type: 'sent' | 'received' | 'error' | 'status';
  content: string;
}

export function WebSocketDebug() {
  const [connectionStatus, setConnectionStatus] = useState<
    'disconnected' | 'connecting' | 'connected' | 'error'
  >('disconnected');
  const [messages, setMessages] = useState<WebSocketMessage[]>([]);
  const [wsUrl, setWsUrl] = useState<string>('');
  const [envInfo, setEnvInfo] = useState<{
    protocol: string;
    host: string;
    port: string;
    detection: string;
  }>({ protocol: '', host: '', port: '', detection: '' });
  const wsRef = useRef<WebSocket | null>(null);

  // Calculate WebSocket URL based on environment (in useEffect to avoid SSR/client mismatch)
  useEffect(() => {
    const isNextDevServer = window.location.port === '3000';
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const host = isNextDevServer ? 'localhost:8081' : window.location.host;
    const url = `${protocol}//${host}/api/debug/ws`;

    setWsUrl(url);
    setEnvInfo({
      protocol: window.location.protocol,
      host: window.location.host,
      port: window.location.port,
      detection: isNextDevServer
        ? 'Next.js dev server → Connect to Go server (8081)'
        : 'Go server → Connect to same host',
    });
  }, []);

  const addMessage = (type: WebSocketMessage['type'], content: string) => {
    setMessages((prev) => [
      ...prev,
      {
        timestamp: new Date().toISOString(),
        type,
        content,
      },
    ]);
  };

  const connect = () => {
    if (!wsUrl) return;

    setConnectionStatus('connecting');
    addMessage('status', `Attempting to connect to: ${wsUrl}`);

    try {
      const ws = new WebSocket(wsUrl);
      wsRef.current = ws;

      ws.onopen = () => {
        setConnectionStatus('connected');
        addMessage('status', 'WebSocket connected successfully');

        // Send a test message
        const testMsg = JSON.stringify({ type: 'test', message: 'Hello from client' });
        ws.send(testMsg);
        addMessage('sent', testMsg);
      };

      ws.onmessage = (event) => {
        addMessage('received', event.data);
      };

      ws.onerror = (error) => {
        setConnectionStatus('error');
        addMessage('error', `WebSocket error: ${error}`);
      };

      ws.onclose = () => {
        setConnectionStatus('disconnected');
        addMessage('status', 'WebSocket connection closed');
      };
    } catch (error) {
      setConnectionStatus('error');
      addMessage('error', `Failed to create WebSocket: ${error}`);
    }
  };

  const disconnect = () => {
    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }
  };

  const sendTestMessage = () => {
    if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
      const testMsg = JSON.stringify({
        type: 'ping',
        timestamp: new Date().toISOString(),
        message: 'Test message from client',
      });
      wsRef.current.send(testMsg);
      addMessage('sent', testMsg);
    }
  };

  const clearMessages = () => {
    setMessages([]);
  };

  const getStatusBadge = () => {
    switch (connectionStatus) {
      case 'connected':
        return <Badge className="bg-green-500">Connected</Badge>;
      case 'connecting':
        return <Badge className="bg-yellow-500">Connecting...</Badge>;
      case 'error':
        return <Badge className="bg-red-500">Error</Badge>;
      default:
        return <Badge variant="outline">Disconnected</Badge>;
    }
  };

  return (
    <div className="space-y-4">
      <Card>
        <CardHeader>
          <CardTitle>WebSocket Debug Console</CardTitle>
          <CardDescription>Test WebSocket connectivity without authentication</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {/* Connection Info */}
          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <span className="text-sm font-medium">Status:</span>
              {getStatusBadge()}
            </div>
            <div className="space-y-1">
              <span className="text-sm font-medium">URL:</span>
              <code className="block rounded bg-muted px-2 py-1 text-xs">
                {wsUrl || 'Calculating...'}
              </code>
            </div>
            <div className="space-y-1">
              <span className="text-sm font-medium">Environment:</span>
              <ul className="text-xs text-muted-foreground space-y-1">
                <li>Protocol: {envInfo.protocol || 'Loading...'}</li>
                <li>Host: {envInfo.host || 'Loading...'}</li>
                <li>Port: {envInfo.port || 'Loading...'}</li>
                <li>Detection: {envInfo.detection || 'Loading...'}</li>
              </ul>
            </div>
          </div>

          {/* Controls */}
          <div className="flex gap-2">
            <Button onClick={connect} disabled={connectionStatus === 'connected'} size="sm">
              Connect
            </Button>
            <Button
              onClick={disconnect}
              disabled={connectionStatus !== 'connected'}
              size="sm"
              variant="outline"
            >
              Disconnect
            </Button>
            <Button
              onClick={sendTestMessage}
              disabled={connectionStatus !== 'connected'}
              size="sm"
              variant="outline"
            >
              Send Test Message
            </Button>
            <Button onClick={clearMessages} size="sm" variant="ghost">
              Clear Messages
            </Button>
          </div>

          {/* Message Log */}
          <div className="space-y-2">
            <span className="text-sm font-medium">Message Log ({messages.length}):</span>
            <div className="max-h-96 overflow-y-auto rounded border">
              {messages.length === 0 ? (
                <div className="p-4 text-center text-sm text-muted-foreground">No messages yet</div>
              ) : (
                <div className="space-y-2 p-2">
                  {messages.map((msg, idx) => (
                    <div
                      key={idx}
                      className={`rounded p-2 text-xs font-mono ${
                        msg.type === 'sent'
                          ? 'bg-blue-50 dark:bg-blue-950'
                          : msg.type === 'received'
                            ? 'bg-green-50 dark:bg-green-950'
                            : msg.type === 'error'
                              ? 'bg-red-50 dark:bg-red-950'
                              : 'bg-gray-50 dark:bg-gray-950'
                      }`}
                    >
                      <div className="flex items-center justify-between">
                        <span className="font-semibold">
                          {msg.type === 'sent'
                            ? '→ SENT'
                            : msg.type === 'received'
                              ? '← RECEIVED'
                              : msg.type === 'error'
                                ? '✕ ERROR'
                                : '• STATUS'}
                        </span>
                        <span className="text-muted-foreground">
                          {new Date(msg.timestamp).toLocaleTimeString()}
                        </span>
                      </div>
                      <pre className="mt-1 whitespace-pre-wrap break-all">{msg.content}</pre>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
