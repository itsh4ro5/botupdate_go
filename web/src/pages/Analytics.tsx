import React, { useEffect, useState } from 'react';
import { 
  Users, Layers, Inbox, ShieldAlert, Star, MessageSquare, 
  Activity, Database, Clock, RefreshCw, BarChart2
} from 'lucide-react';
import { api } from '../services/api';
import { useRealtimeContext } from '../contexts/RealtimeContext';
import './Analytics.css';

interface SystemStatus {
  telegram: string;
  database: string;
  userbot: string;
  scheduler: string;
  websocket: string;
  event_bus: string;
  goroutines: number;
  memory_mb: number;
  uptime: string;
}

interface OverviewDTO {
  available: boolean;
  total_users: number;
  total_batches: number;
  pending_requests: number;
  blocked_users: number;
  vip_users: number;
  unlocked_access_count: number;
  pending_support_conversations: number;
  system_status: SystemStatus;
}

interface BatchStats {
  id: string;
  name: string;
  type: string;
  category: string;
  user_count: number;
  percentage: number;
}

interface BatchAnalyticsDTO {
  available: boolean;
  total_batches: number;
  free_batches: number;
  paid_batches: number;
  special_batches: number;
  batches: BatchStats[];
}

export const Analytics: React.FC = () => {
  const { status, events } = useRealtimeContext();
  
  const [overview, setOverview] = useState<OverviewDTO | null>(null);
  const [batchData, setBatchData] = useState<BatchAnalyticsDTO | null>(null);
  const [loading, setLoading] = useState(true);

  const fetchAnalytics = async () => {
    try {
      const [overviewRes, batchRes] = await Promise.all([
        api.get<OverviewDTO>('/analytics/overview'),
        api.get<BatchAnalyticsDTO>('/analytics/batches')
      ]);
      setOverview(overviewRes);
      setBatchData(batchRes);
    } catch (err) {
      console.error('Failed to load analytics', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchAnalytics();
    
    // Refresh interval for realtime feel (30s)
    const interval = setInterval(() => {
      fetchAnalytics();
    }, 30000);
    
    return () => clearInterval(interval);
  }, []);

  // Invalidate on relevant realtime events
  useEffect(() => {
    if (events.length > 0) {
      const latest = events[events.length - 1];
      if (['USER_JOINED', 'ACCESS_GRANTED', 'REQUEST_APPROVED', 'SUPPORT_MESSAGE'].includes(latest.type)) {
        // Debounce refresh
        const timer = setTimeout(() => {
          fetchAnalytics();
        }, 1500);
        return () => clearTimeout(timer);
      }
    }
  }, [events]);

  if (loading) {
    return (
      <div className="flex-center h-full w-full">
        <RefreshCw className="spin text-accent" size={48} />
      </div>
    );
  }

  if (!overview) return null;

  return (
    <div className="analytics-page fade-in">
      <div className="analytics-header">
        <h1>Analytics & Intelligence</h1>
        <div className="system-status-indicator">
          <div className={`status-dot ${status === 'LIVE' ? 'online' : 'offline'}`}></div>
          WebSocket: {status}
        </div>
      </div>

      <div className="analytics-grid">
        <div className="analytics-card glass-panel">
          <div className="card-header">
            <span className="card-title">Total Users</span>
            <div className="card-icon"><Users size={24} /></div>
          </div>
          <div className="card-value">{overview.total_users}</div>
        </div>

        <div className="analytics-card glass-panel">
          <div className="card-header">
            <span className="card-title">Total Batches</span>
            <div className="card-icon"><Layers size={24} /></div>
          </div>
          <div className="card-value">{overview.total_batches}</div>
        </div>

        <div className="analytics-card glass-panel">
          <div className="card-header">
            <span className="card-title">Pending Requests</span>
            <div className="card-icon"><Inbox size={24} /></div>
          </div>
          <div className="card-value">{overview.pending_requests}</div>
        </div>

        <div className="analytics-card glass-panel">
          <div className="card-header">
            <span className="card-title">Active Support</span>
            <div className="card-icon"><MessageSquare size={24} /></div>
          </div>
          <div className="card-value">{overview.pending_support_conversations}</div>
        </div>
      </div>

      {/* System Health */}
      <div className="analytics-section">
        <h2 className="section-title"><Activity size={24} /> System Health</h2>
        <div className="system-health-grid">
          <div className="health-item">
            <span className="health-label">Telegram API</span>
            <div className={`health-status ${overview.system_status.telegram === 'ONLINE' ? 'online' : 'offline'}`}>
              <div className={`status-dot ${overview.system_status.telegram === 'ONLINE' ? 'online' : 'offline'}`}></div>
              {overview.system_status.telegram}
            </div>
          </div>
          <div className="health-item">
            <span className="health-label">Database</span>
            <div className={`health-status ${overview.system_status.database === 'ONLINE' ? 'online' : 'offline'}`}>
              <div className={`status-dot ${overview.system_status.database === 'ONLINE' ? 'online' : 'offline'}`}></div>
              {overview.system_status.database}
            </div>
          </div>
          <div className="health-item">
            <span className="health-label">Memory Allocation</span>
            <div className="health-status text-white">{overview.system_status.memory_mb} MB</div>
          </div>
          <div className="health-item">
            <span className="health-label">Goroutines</span>
            <div className="health-status text-white">{overview.system_status.goroutines}</div>
          </div>
          <div className="health-item">
            <span className="health-label">Uptime</span>
            <div className="health-status text-white">{overview.system_status.uptime}</div>
          </div>
        </div>
      </div>

      {/* User Intelligence */}
      <div className="analytics-section">
        <h2 className="section-title"><Users size={24} /> User Intelligence</h2>
        <div className="analytics-grid">
          <div className="analytics-card glass-panel">
            <div className="card-header">
              <span className="card-title">VIP Users</span>
              <div className="card-icon text-warning"><Star size={24} /></div>
            </div>
            <div className="card-value">{overview.vip_users}</div>
          </div>
          <div className="analytics-card glass-panel">
            <div className="card-header">
              <span className="card-title">Blocked Users</span>
              <div className="card-icon text-error"><ShieldAlert size={24} /></div>
            </div>
            <div className="card-value">{overview.blocked_users}</div>
          </div>
          <div className="analytics-card glass-panel">
            <div className="card-header">
              <span className="card-title">Unlocked Access</span>
              <div className="card-icon text-success"><Database size={24} /></div>
            </div>
            <div className="card-value">{overview.unlocked_access_count}</div>
          </div>
        </div>
      </div>

      {/* Historical Note */}
      <div className="analytics-section">
        <h2 className="section-title"><Clock size={24} /> Historical Analytics</h2>
        <div className="empty-state-card">
          <div className="empty-state-icon">
            <BarChart2 size={32} />
          </div>
          <h3>Insufficient Historical Data</h3>
          <p className="text-muted max-w-lg">
            Historical analytics (such as DAU, MAU, or historical approval charts) are unavailable because the current persistence model does not retain long-term behavioral history. Showing real-time current snapshot data instead.
          </p>
        </div>
      </div>

      {/* Batch Intelligence */}
      {batchData && batchData.batches && batchData.batches.length > 0 && (
        <div className="analytics-section">
          <h2 className="section-title"><Layers size={24} /> Batch Intelligence</h2>
          <div className="batch-list glass-panel p-4">
            {batchData.batches.sort((a, b) => b.user_count - a.user_count).map(b => (
              <div key={b.id} className="batch-item">
                <div className="batch-info">
                  <span className="batch-name">{b.name}</span>
                  <div className="batch-meta">
                    <span className="badge badge-outline uppercase">{b.type}</span>
                    <span>{b.category}</span>
                  </div>
                </div>
                <div className="batch-stats">
                  <div className="flex flex-col items-end">
                    <span className="batch-count">{b.user_count} Users</span>
                    <span className="text-xs text-muted">{b.percentage.toFixed(1)}% of total</span>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

    </div>
  );
};
