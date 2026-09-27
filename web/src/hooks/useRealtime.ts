import { useEffect, useRef, useState, useCallback } from 'react';

type EventType = 'USER_JOINED' | 'USER_UPDATED' | 'USER_ACTIVITY' | 'SUPPORT_MESSAGE' | 'ACCESS_REQUEST' | 'ACCESS_GRANTED' | 'ACCESS_REVOKED' | 'BATCH_SCAN_STARTED' | 'BATCH_SCAN_COMPLETED' | 'BROADCAST_STARTED' | 'BROADCAST_COMPLETED' | 'USERBOT_STATUS' | 'SCHEDULER_JOB' | 'SYSTEM_ERROR' | 'SYSTEM_INFO';

export interface AppEvent {
  id: string;
  type: EventType;
  timestamp: string;
  severity: 'info' | 'warning' | 'error';
  data: Record<string, any>;
}

export type ConnectionStatus = 'LIVE' | 'RECONNECTING' | 'OFFLINE';

export function useRealtime() {
  const [events, setEvents] = useState<AppEvent[]>([]);
  const [status, setStatus] = useState<ConnectionStatus>('OFFLINE');
  const ws = useRef<WebSocket | null>(null);
  const reconnectAttempts = useRef(0);
  const maxReconnects = 10;
  
  // Track IDs to prevent duplicates
  const eventIds = useRef<Set<string>>(new Set());

  const connect = useCallback(() => {
    if (ws.current?.readyState === WebSocket.OPEN || ws.current?.readyState === WebSocket.CONNECTING) return;
    
    // Construct ws url based on current origin
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    // For dev: fallback to localhost:3000 if running in vite port 5173
    const host = window.location.port === '5173' ? 'localhost:3000' : window.location.host;
    const wsUrl = `${protocol}//${host}/api/v1/ws`;

    console.log(`[WebSocket] Connecting to ${wsUrl}`);
    const socket = new WebSocket(wsUrl);

    socket.onopen = () => {
      setStatus('LIVE');
      reconnectAttempts.current = 0;
    };

    socket.onmessage = (msg) => {
      try {
        const payload = JSON.parse(msg.data);
        if (payload.type === 'event' && payload.event) {
          const newEvent = payload.event as AppEvent;
          
          if (!eventIds.current.has(newEvent.id)) {
            eventIds.current.add(newEvent.id);
            
            // Dispatch event for components listening directly
            window.dispatchEvent(new CustomEvent('REALTIME_EVENT', {
              detail: newEvent
            }));

            setEvents(prev => {
              const updated = [newEvent, ...prev];
              if (updated.length > 50) {
                // Keep only last 50 events, remove oldest from tracking set
                const removed = updated.pop();
                if (removed) eventIds.current.delete(removed.id);
              }
              return updated;
            });
          }
        }
      } catch (err) {
        console.error('[WebSocket] Failed to parse message', err);
      }
    };

    socket.onclose = () => {
      setStatus('OFFLINE');
      ws.current = null;
      
      // Exponential backoff reconnect
      if (reconnectAttempts.current < maxReconnects) {
        setStatus('RECONNECTING');
        const timeout = Math.min(1000 * Math.pow(2, reconnectAttempts.current), 30000);
        reconnectAttempts.current++;
        setTimeout(connect, timeout);
      }
    };

    socket.onerror = (error) => {
      console.error('[WebSocket] Error', error);
      socket.close();
    };

    ws.current = socket;
  }, []);

  const disconnect = useCallback(() => {
    reconnectAttempts.current = maxReconnects; // Prevent auto-reconnect
    if (ws.current) {
      ws.current.close();
      ws.current = null;
    }
  }, []);

  useEffect(() => {
    connect();
    return () => {
      disconnect();
    };
  }, [connect, disconnect]);

  return { events, status };
}
