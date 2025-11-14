'use client';

import { useState, useEffect, useRef } from 'react';
import { ActivityEventItem } from './activity-event-item';
import { GraphWebSocket, type GraphUpdate, type GraphEvent } from '@/lib/graph-websocket';

interface ActivityFeedProps {
  appName?: string; // Optional: filter by application
  maxEvents?: number; // Maximum events to display
  className?: string;
}

/**
 * Live activity feed showing real-time graph events
 * Displays what's happening in the system as it happens
 */
export function ActivityFeed({ appName = 'system', maxEvents = 20, className = '' }: ActivityFeedProps) {
  const [events, setEvents] = useState<GraphEvent[]>([]);
  const [connected, setConnected] = useState(false);
  const wsRef = useRef<GraphWebSocket | null>(null);

  // Subscribe to WebSocket graph updates
  useEffect(() => {
    // Create WebSocket connection
    const ws = new GraphWebSocket(appName);
    wsRef.current = ws;

    // Subscribe to updates
    const unsubscribe = ws.subscribe((update: GraphUpdate) => {
      // Only process updates that have event metadata
      if (update.event) {
        addEvent(update.event);
      }
    });

    // Connect
    ws.connect();
    setConnected(true);

    // Cleanup on unmount
    return () => {
      unsubscribe();
      ws.disconnect();
      setConnected(false);
    };
  }, [appName]);

  const addEvent = (event: GraphEvent) => {
    setEvents((prev) => {
      // Add new event at the beginning (most recent first)
      const updated = [event, ...prev];
      // Keep only the latest maxEvents
      return updated.slice(0, maxEvents);
    });
  };

  // Expose addEvent for parent components to use (for testing)
  useEffect(() => {
    if (typeof window !== 'undefined') {
      window.__addActivityEvent = addEvent;
    }
  }, [addEvent]);

  if (events.length === 0) {
    return (
      <div className={`rounded-lg border border-zinc-200 bg-white p-6 dark:border-zinc-800 dark:bg-zinc-950 ${className}`}>
        <h3 className="mb-4 text-sm font-semibold text-zinc-900 dark:text-white">Live Activity</h3>
        <div className="text-center text-sm text-zinc-500 dark:text-zinc-500">
          <p>No recent activity</p>
          <p className="mt-1 text-xs">Events will appear here as they happen</p>
        </div>
      </div>
    );
  }

  return (
    <div className={`rounded-lg border border-zinc-200 bg-white dark:border-zinc-800 dark:bg-zinc-950 ${className}`}>
      {/* Header */}
      <div className="border-b border-zinc-200 px-4 py-3 dark:border-zinc-800">
        <div className="flex items-center justify-between">
          <h3 className="text-sm font-semibold text-zinc-900 dark:text-white">Live Activity</h3>
          <div className="flex items-center gap-2">
            {connected ? (
              <>
                <div className="h-2 w-2 animate-pulse rounded-full bg-green-500" />
                <span className="text-xs text-zinc-500">Live</span>
              </>
            ) : (
              <>
                <div className="h-2 w-2 rounded-full bg-zinc-400" />
                <span className="text-xs text-zinc-500">Connecting...</span>
              </>
            )}
          </div>
        </div>
        {appName && (
          <p className="mt-1 text-xs text-zinc-500">
            Showing events for <span className="font-mono">{appName}</span>
          </p>
        )}
      </div>

      {/* Event list */}
      <div className="max-h-96 overflow-y-auto p-4">
        <div className="space-y-2">
          {events.map((event, index) => (
            <ActivityEventItem key={`${event.timestamp}-${index}`} event={event} />
          ))}
        </div>
      </div>

      {/* Footer */}
      {events.length >= maxEvents && (
        <div className="border-t border-zinc-200 px-4 py-2 text-center text-xs text-zinc-500 dark:border-zinc-800">
          Showing latest {maxEvents} events
        </div>
      )}
    </div>
  );
}
