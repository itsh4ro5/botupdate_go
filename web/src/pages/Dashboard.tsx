import React, { useEffect, useState, useRef } from 'react';
import { api } from '../services/api';
import { useRealtimeContext } from '../contexts/RealtimeContext';
import { type ConnectionStatus, type AppEvent } from '../hooks/useRealtime';
import './Dashboard.css';
import { Users, Layers, Activity, Radio, RefreshCw, AlertCircle, MessageCircle, ShieldCheck, UserPlus, PlayCircle, Clock } from 'lucide-react';

interface DashboardOverview {
  users: {
    total: number;
    active: number;
  };
  batches: {
    total: number;
  };
  requests: {
    pending: number;
  };
  system: {
    telegram: string;
    database: string;
    scheduler: string;
    userbot: string;
  };
  generated_at: string;
}

export const Dashboard: React.FC = () => {
  const [data, setData] = useState<DashboardOverview | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [refreshing, setRefreshing] = useState(false);
  const { events, status: wsStatus } = useRealtimeContext();
  const lastEventCount = useRef(events.length);

  const fetchOverview = async (isManual = false) => {
    if (isManual) setRefreshing(true);
    try {
      const res = await api.get<DashboardOverview>('/dashboard/overview');
      setData(res);
      setError(null);
    } catch (err: any) {
      setError(err.message || 'Database unavailable');
    } finally {
      setLoading(false);
      if (isManual) setTimeout(() => setRefreshing(false), 500); // Visual feedback
    }
  };

  useEffect(() => {
    fetchOverview();
    const interval = setInterval(() => fetchOverview(), 30000); // Auto-refresh every 30s
    return () => clearInterval(interval);
  }, []);

  // Smart debounce refresh when events arrive
  useEffect(() => {
    if (events.length > lastEventCount.current) {
      const newEvent = events[0];
      if (['USER_JOINED', 'ACCESS_GRANTED', 'ACCESS_REVOKED'].includes(newEvent.type)) {
        fetchOverview();
      }
    }
    lastEventCount.current = events.length;
  }, [events]);

  const formatRelativeTime = (isoString: string) => {
    const date = new Date(isoString);
    const seconds = Math.floor((new Date().getTime() - date.getTime()) / 1000);
    if (seconds < 60) return `Just now`;
    if (seconds < 3600) return `${Math.floor(seconds / 60)} minutes ago`;
    return date.toLocaleTimeString();
  };

  const getStatusColor = (status: ConnectionStatus) => {
    switch (status) {
      case 'LIVE': return 'var(--success)';
      case 'RECONNECTING': return 'var(--warning)';
      default: return 'var(--error)';
    }
  };

  const renderEventIcon = (type: string) => {
    switch (type) {
      case 'USER_JOINED': return <UserPlus size={16} color="var(--primary-light)" />;
      case 'SUPPORT_MESSAGE': return <MessageCircle size={16} color="var(--secondary)" />;
      case 'ACCESS_GRANTED': return <ShieldCheck size={16} color="var(--success)" />;
      default: return <PlayCircle size={16} color="var(--muted)" />;
    }
  };

  const renderEventText = (event: AppEvent) => {
    switch (event.type) {
      case 'USER_JOINED':
        return <span>User <b>{event.data.name || event.data.username || event.data.user_id}</b> joined the bot</span>;
      case 'SUPPORT_MESSAGE':
        return <span>Support message {event.data.direction} for user {event.data.username || event.data.user_id}</span>;
      case 'ACCESS_GRANTED':
        return <span>Granted batch {event.data.batch_id} to user {event.data.user_id} via {event.data.method}</span>;
      default:
        return <span>{event.type} event</span>;
    }
  };

  return (
    <div className="dashboard-page animate-in">
      <div className="header-section flex justify-between items-center mb-6">
        <div>
          <div className="flex items-center gap-3">
            <h2>System Overview</h2>
            <div 
              style={{
                display: 'inline-flex', alignItems: 'center', gap: '6px',
                padding: '4px 10px', borderRadius: '12px',
                background: 'rgba(255,255,255,0.05)',
                border: '1px solid var(--border-light)',
                fontSize: '0.75rem', fontWeight: 600,
                color: getStatusColor(wsStatus)
              }}
            >
              <div style={{
                width: '8px', height: '8px', borderRadius: '50%',
                background: getStatusColor(wsStatus),
                boxShadow: `0 0 6px ${getStatusColor(wsStatus)}`,
                animation: wsStatus === 'LIVE' ? 'pulse 2s infinite' : 'none'
              }}></div>
              {wsStatus}
            </div>
          </div>
          <p className="subtitle">Real-time metrics and system health</p>
        </div>
        <div className="flex items-center gap-4" style={{ display: 'flex', alignItems: 'center', gap: '1rem' }}>
          {data && !error && (
            <span className="text-muted" style={{ fontSize: '0.8rem', color: 'var(--text-secondary)' }}>
              Last updated {formatRelativeTime(data.generated_at)}
            </span>
          )}
          <a href="/analytics" style={{
            background: 'var(--accent-primary)',
            color: 'white',
            padding: '0.5rem 1rem',
            borderRadius: '8px',
            textDecoration: 'none',
            fontSize: '0.85rem',
            fontWeight: '600',
            display: 'flex',
            alignItems: 'center',
            gap: '0.5rem'
          }}>
            <Activity size={16} /> View Analytics
          </a>
          <button 
            className="icon-btn refresh-btn" 
            onClick={() => fetchOverview(true)}
            disabled={loading || refreshing}
            title="Refresh Dashboard"
            style={{ 
              background: 'rgba(255,255,255,0.05)', 
              border: '1px solid var(--border-light)', 
              borderRadius: '8px', 
              padding: '0.5rem',
              cursor: 'pointer',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              color: 'var(--text-primary)'
            }}
          >
            <RefreshCw size={16} style={{ animation: refreshing ? 'spin 1s linear infinite' : 'none' }} />
          </button>
        </div>
      </div>

      {error && (
        <div className="error-banner mb-6 p-4 rounded-lg bg-red-900/20 border border-red-500/30 text-red-400" style={{ background: 'rgba(239, 68, 68, 0.1)', border: '1px solid rgba(239, 68, 68, 0.3)', color: 'var(--error)', padding: '1rem', borderRadius: '8px', marginBottom: '1.5rem', display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
          <AlertCircle size={18} /> {error}
        </div>
      )}

      <div className="stats-grid">
        <div className="glass-card stat-card">
          <div className="stat-icon users-icon"><Users size={24} /></div>
          <div className="stat-content">
            <p className="stat-label">Total Users</p>
            <h3 className="stat-value">
              {loading ? <span className="skeleton-text"></span> : data ? data.users.total.toLocaleString() : '--'}
            </h3>
          </div>
        </div>
        
        <div className="glass-card stat-card">
          <div className="stat-icon batches-icon"><Layers size={24} /></div>
          <div className="stat-content">
            <p className="stat-label">Total Batches</p>
            <h3 className="stat-value">
              {loading ? <span className="skeleton-text"></span> : data ? data.batches.total.toLocaleString() : '--'}
            </h3>
          </div>
        </div>
        
        <div className="glass-card stat-card">
          <div className="stat-icon requests-icon"><Radio size={24} /></div>
          <div className="stat-content">
            <p className="stat-label">Pending Requests</p>
            <h3 className="stat-value">
              {loading ? <span className="skeleton-text"></span> : data ? data.requests.pending.toLocaleString() : '--'}
            </h3>
          </div>
        </div>
        
        <div className="glass-card stat-card">
          <div className="stat-icon activity-icon"><Activity size={24} /></div>
          <div className="stat-content">
            <p className="stat-label">Active Users (24h)</p>
            <h3 className="stat-value">
              {loading ? <span className="skeleton-text"></span> : (data?.users.active === -1 ? <span style={{fontSize: '1rem', color: 'var(--text-muted)'}}>Unavailable</span> : data ? data.users.active.toLocaleString() : '--')}
            </h3>
          </div>
        </div>
      </div>

      <div className="dashboard-grid mt-8" style={{ marginTop: '2rem', display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(300px, 1fr))', gap: '1.5rem' }}>
        
        {/* Live Activity Feed */}
        <div className="glass-card flex-1 p-6" style={{ minHeight: '350px', display: 'flex', flexDirection: 'column' }}>
          <div className="flex justify-between items-center mb-4" style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '1rem' }}>
            <h3 className="card-title m-0" style={{ margin: 0, fontSize: '1.1rem', color: 'var(--text-primary)' }}>Live Activity</h3>
            {wsStatus === 'LIVE' && <span style={{ fontSize: '0.75rem', color: 'var(--success)', display: 'flex', alignItems: 'center', gap: '4px' }}><Activity size={12}/> Monitoring</span>}
          </div>
          
          <div className="activity-feed" style={{ flex: 1, overflowY: 'auto', paddingRight: '0.5rem', display: 'flex', flexDirection: 'column', gap: '0.75rem' }}>
            {events.length === 0 ? (
              <div className="flex items-center justify-center h-full text-muted" style={{ height: '100%', display: 'flex', alignItems: 'center', justifyContent: 'center', color: 'var(--text-muted)' }}>
                {wsStatus === 'LIVE' ? 'Waiting for events...' : 'Connect to see live activity'}
              </div>
            ) : (
              events.map(event => (
                <div key={event.id} className="activity-item animate-in" style={{ 
                  display: 'flex', gap: '1rem', padding: '0.75rem', 
                  background: 'rgba(255,255,255,0.02)', borderRadius: '8px',
                  borderLeft: `3px solid ${event.severity === 'error' ? 'var(--error)' : event.severity === 'warning' ? 'var(--warning)' : 'var(--primary-light)'}`
                }}>
                  <div className="activity-icon" style={{ marginTop: '2px' }}>
                    {renderEventIcon(event.type)}
                  </div>
                  <div className="activity-content" style={{ flex: 1 }}>
                    <div className="activity-text" style={{ fontSize: '0.9rem', color: 'var(--text-primary)' }}>
                      {renderEventText(event)}
                    </div>
                    <div className="activity-time" style={{ fontSize: '0.75rem', color: 'var(--text-muted)', marginTop: '4px', display: 'flex', alignItems: 'center', gap: '4px' }}>
                      <Clock size={10} /> {new Date(event.timestamp).toLocaleTimeString()}
                    </div>
                  </div>
                </div>
              ))
            )}
          </div>
        </div>

        {/* System Health */}
        <div className="glass-card flex-1 p-6" style={{ minHeight: '350px' }}>
          <h3 className="card-title mb-4" style={{ margin: 0, marginBottom: '1.5rem', fontSize: '1.1rem', color: 'var(--text-primary)' }}>Service Status</h3>
          <div className="health-grid" style={{ display: 'flex', flexDirection: 'column', gap: '1rem' }}>
            {['Telegram', 'Database', 'Scheduler', 'Userbot'].map(service => {
              const key = service.toLowerCase() as keyof DashboardOverview['system'];
              const isOnline = data && !error ? data.system[key] === 'online' : false;
              
              return (
                <div key={service} className="glass-card health-card" style={{ display: 'flex', alignItems: 'center', padding: '1rem', gap: '1rem', background: 'rgba(255,255,255,0.03)' }}>
                  <div className={`status-indicator ${isOnline ? 'online' : 'offline'}`} style={{ 
                    width: '12px', height: '12px', borderRadius: '50%', 
                    background: isOnline ? 'var(--success)' : 'var(--error)',
                    boxShadow: `0 0 8px ${isOnline ? 'var(--success)' : 'var(--error)'}`
                  }}></div>
                  <div>
                    <h4 style={{ margin: 0, fontSize: '0.95rem', color: 'var(--text-primary)' }}>{service}</h4>
                    <p style={{ margin: 0, fontSize: '0.8rem', color: 'var(--text-muted)', marginTop: '2px' }}>
                      {loading ? 'Checking...' : (isOnline ? 'Operational' : 'Unavailable')}
                    </p>
                  </div>
                </div>
              )
            })}
          </div>
        </div>

      </div>
    </div>
  );
};
