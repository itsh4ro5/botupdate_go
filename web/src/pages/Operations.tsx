import React, { useEffect, useState } from 'react';
import { Activity, Server, Cpu, Database, Network, Clock, Settings, RefreshCw, Zap } from 'lucide-react';
import { api } from '../services/api';
import './Operations.css';

interface RuntimeStatus {
  go_version: string;
  os: string;
  arch: string;
  uptime: string;
  goroutines: number;
  memory_alloc: string;
  start_time: string;
}

interface ServiceStatus {
  telegram: string;
  mongodb: string;
  web_api: string;
  websocket: string;
  scheduler: string;
  userbot: string;
}

interface OpsData {
  runtime: RuntimeStatus;
  services: ServiceStatus;
}

export const Operations: React.FC = () => {
  const [data, setData] = useState<OpsData | null>(null);
  const [loading, setLoading] = useState(true);

  const fetchOps = async () => {
    try {
      const res = await api.get<OpsData>('/operations/system');
      setData(res);
    } catch (error) {
      console.error('Failed to fetch operations data', error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchOps();
    const interval = setInterval(fetchOps, 30000);
    return () => clearInterval(interval);
  }, []);

  const refreshCache = async () => {
    try {
      await api.post('/operations/maintenance/refresh-cache', {});
      alert('Cache refreshed successfully');
    } catch (e) {
      alert('Failed to refresh cache');
    }
  };

  const StatusDot = ({ status }: { status: string }) => {
    const cls = status.toLowerCase() === 'online' ? 'online' : status.toLowerCase() === 'degraded' ? 'degraded' : status.toLowerCase() === 'maintenance' ? 'maintenance' : 'offline';
    return (
      <div className={`ops-status ${cls}`}>
        <div className={`status-dot ${cls}`}></div>
        {status}
      </div>
    );
  };

  if (loading || !data) {
    return <div className="flex-center h-full"><RefreshCw className="spin text-accent" size={32} /></div>;
  }

  return (
    <div className="operations-page fade-in">
      <div className="page-header">
        <h1>System Operations</h1>
        <button className="btn btn-primary" onClick={fetchOps}>
          <RefreshCw size={16} /> Refresh
        </button>
      </div>

      <div className="ops-grid">
        <div className="ops-card">
          <div className="ops-card-header">
            <Cpu size={20} className="text-accent" /> Runtime Environment
          </div>
          <div className="ops-row">
            <span className="ops-label">Go Version</span>
            <span className="ops-value">{data.runtime.go_version}</span>
          </div>
          <div className="ops-row">
            <span className="ops-label">OS / Arch</span>
            <span className="ops-value">{data.runtime.os} / {data.runtime.arch}</span>
          </div>
          <div className="ops-row">
            <span className="ops-label">Memory Allocated</span>
            <span className="ops-value">{data.runtime.memory_alloc}</span>
          </div>
          <div className="ops-row">
            <span className="ops-label">Active Goroutines</span>
            <span className="ops-value">{data.runtime.goroutines}</span>
          </div>
          <div className="ops-row">
            <span className="ops-label">Uptime</span>
            <span className="ops-value">{data.runtime.uptime}</span>
          </div>
        </div>

        <div className="ops-card">
          <div className="ops-card-header">
            <Server size={20} className="text-secondary" /> Service Health
          </div>
          <div className="ops-row">
            <span className="ops-label flex items-center gap-2"><Network size={16}/> Telegram API</span>
            <StatusDot status={data.services.telegram} />
          </div>
          <div className="ops-row">
            <span className="ops-label flex items-center gap-2"><Database size={16}/> MongoDB</span>
            <StatusDot status={data.services.mongodb} />
          </div>
          <div className="ops-row">
            <span className="ops-label flex items-center gap-2"><Activity size={16}/> Web API</span>
            <StatusDot status={data.services.web_api} />
          </div>
          <div className="ops-row">
            <span className="ops-label flex items-center gap-2"><Activity size={16}/> WebSocket</span>
            <StatusDot status={data.services.websocket} />
          </div>
          <div className="ops-row">
            <span className="ops-label flex items-center gap-2"><Clock size={16}/> Scheduler</span>
            <StatusDot status={data.services.scheduler} />
          </div>
        </div>
      </div>

      <div className="ops-card">
        <div className="ops-card-header">
          <Settings size={20} className="text-warning" /> Safe Maintenance Actions
        </div>
        <div className="maintenance-actions">
          <button className="btn btn-secondary" onClick={refreshCache}>
            <Zap size={16} /> Refresh Analytics Cache
          </button>
          <button className="btn btn-secondary" onClick={() => window.dispatchEvent(new CustomEvent('REFRESH_DASHBOARD'))}>
            <RefreshCw size={16} /> Refresh Dashboard Sync
          </button>
        </div>
      </div>
    </div>
  );
};
