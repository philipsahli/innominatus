'use client';

import { WebSocketDebug } from '@/components/dev/websocket-debug';

export default function WebSocketTestPage() {
  return (
    <div className="container mx-auto p-6">
      <div className="mb-6">
        <h1 className="text-3xl font-bold">WebSocket Debug Test</h1>
        <p className="text-muted-foreground mt-2">
          Test WebSocket connectivity in different environments without authentication
        </p>
      </div>

      <div className="space-y-6">
        <WebSocketDebug />

        <div className="rounded-lg border p-4 bg-muted/50">
          <h2 className="font-semibold mb-2">Instructions</h2>
          <ul className="list-disc list-inside space-y-1 text-sm text-muted-foreground">
            <li>
              <strong>From Next.js dev server (port 3000):</strong> Should connect to{' '}
              <code>ws://localhost:8081/api/debug/ws</code>
            </li>
            <li>
              <strong>From Go server (port 8081):</strong> Should connect to{' '}
              <code>ws://localhost:8081/api/debug/ws</code>
            </li>
            <li>Click &quot;Connect&quot; to establish WebSocket connection</li>
            <li>Watch the message log for connection status and received messages</li>
            <li>Click &quot;Send Test Message&quot; to test bidirectional communication</li>
            <li>The debug endpoint does NOT require authentication for testing purposes</li>
          </ul>
        </div>

        <div className="rounded-lg border p-4 bg-yellow-50 dark:bg-yellow-950">
          <h2 className="font-semibold mb-2 text-yellow-900 dark:text-yellow-100">Warning</h2>
          <p className="text-sm text-yellow-800 dark:text-yellow-200">
            The <code>/api/debug/ws</code> endpoint is for development/debugging only. It bypasses
            authentication checks and should be removed or disabled in production environments.
          </p>
        </div>
      </div>
    </div>
  );
}
