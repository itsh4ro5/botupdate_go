import React, { createContext, useContext } from 'react';
import { useRealtime as useRealtimeHook, type ConnectionStatus, type AppEvent } from '../hooks/useRealtime';

interface RealtimeContextType {
  events: AppEvent[];
  status: ConnectionStatus;
  unreadCount: number;
  markAllRead: () => void;
  markRead: (id: string) => void;
  notifications: (AppEvent & { read?: boolean })[];
}

const RealtimeContext = createContext<RealtimeContextType | null>(null);

export const RealtimeProvider: React.FC<{children: React.ReactNode}> = ({ children }) => {
  const { events, status } = useRealtimeHook();
  const [readIds, setReadIds] = React.useState<Set<string>>(new Set());

  const notifications = events.map(e => ({
    ...e,
    read: readIds.has(e.id)
  }));

  const unreadCount = notifications.filter(n => !n.read).length;

  const markAllRead = () => {
    setReadIds(new Set(events.map(e => e.id)));
  };

  const markRead = (id: string) => {
    setReadIds(prev => {
      const newSet = new Set(prev);
      newSet.add(id);
      return newSet;
    });
  };

  return (
    <RealtimeContext.Provider value={{ events, status, unreadCount, markAllRead, markRead, notifications }}>
      {children}
    </RealtimeContext.Provider>
  );
};

export const useRealtimeContext = () => {
  const ctx = useContext(RealtimeContext);
  if (!ctx) throw new Error('useRealtimeContext must be used within RealtimeProvider');
  return ctx;
};
