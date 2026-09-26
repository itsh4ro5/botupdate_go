import React, { useEffect, useState } from 'react';
import { Search, RefreshCw } from 'lucide-react';
import { api } from '../services/api';
import './Audit.css';

interface AuditLog {
  id: string;
  timestamp: string;
  actor_username: string;
  actor_role: string;
  action: string;
  target_type: string;
  target_id: string;
  success: boolean;
  metadata: any;
}

interface PaginatedAudit {
  items: AuditLog[];
  page: number;
  pageSize: number;
  total: number;
  totalPages: number;
}

export const Audit: React.FC = () => {
  const [data, setData] = useState<PaginatedAudit | null>(null);
  const [page, setPage] = useState(1);
  const [search, setSearch] = useState('');
  const [actionFilter, setActionFilter] = useState('');
  const [loading, setLoading] = useState(false);

  const fetchLogs = async (p: number) => {
    setLoading(true);
    try {
      const qs = new URLSearchParams({ 
        page: p.toString(), 
        pageSize: '25', 
        search: search, 
        action: actionFilter 
      }).toString();
      const res = await api.get<PaginatedAudit>(`/audit?${qs}`);
      setData(res);
      setPage(p);
    } catch (error) {
      console.error(error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchLogs(1);
  }, [search, actionFilter]);

  const renderMetadata = (meta: any) => {
    if (!meta) return '-';
    return (
      <div className="text-xs font-mono text-muted truncate max-w-xs">
        {JSON.stringify(meta)}
      </div>
    );
  };

  return (
    <div className="audit-page fade-in">
      <div className="page-header">
        <h1>Audit & Security Logs</h1>
        <button className="btn btn-secondary" onClick={() => fetchLogs(page)}>
          <RefreshCw size={16} /> Refresh
        </button>
      </div>

      <div className="audit-controls">
        <div className="search-bar relative flex-1">
          <Search size={18} className="absolute left-3 top-1/2 transform -translate-y-1/2 text-muted" />
          <input 
            type="text" 
            placeholder="Search by actor or target ID..." 
            className="input w-full pl-10"
            value={search}
            onChange={e => setSearch(e.target.value)}
          />
        </div>
        <select 
          className="input w-48"
          value={actionFilter}
          onChange={e => setActionFilter(e.target.value)}
        >
          <option value="">All Actions</option>
          <option value="ADMIN_LOGIN">Login</option>
          <option value="ADMIN_LOGOUT">Logout</option>
          <option value="ADMIN_CREATED">Admin Created</option>
          <option value="SESSION_REVOKED">Session Revoked</option>
        </select>
      </div>

      <div className="audit-table-wrapper">
        <table className="audit-table">
          <thead>
            <tr>
              <th>Timestamp</th>
              <th>Actor</th>
              <th>Action</th>
              <th>Target</th>
              <th>Status</th>
              <th>Metadata</th>
            </tr>
          </thead>
          <tbody>
            {loading && !data ? (
              <tr><td colSpan={6} className="text-center py-4">Loading...</td></tr>
            ) : data?.items.length === 0 ? (
              <tr><td colSpan={6} className="text-center py-8 text-muted">No audit logs found.</td></tr>
            ) : (
              data?.items.map(log => (
                <tr key={log.id}>
                  <td className="whitespace-nowrap">{new Date(log.timestamp).toLocaleString()}</td>
                  <td>
                    <div className="flex flex-col">
                      <span className="font-bold text-white">{log.actor_username}</span>
                      <span className="text-xs text-muted">{log.actor_role}</span>
                    </div>
                  </td>
                  <td>
                    <span className="audit-action">{log.action}</span>
                  </td>
                  <td>
                    {log.target_type}: {log.target_id || '-'}
                  </td>
                  <td>
                    <span className={`badge ${log.success ? 'badge-success' : 'badge-error'}`}>
                      {log.success ? 'SUCCESS' : 'FAILED'}
                    </span>
                  </td>
                  <td>{renderMetadata(log.metadata)}</td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      {data && data.totalPages > 1 && (
        <div className="pagination">
          <button 
            className="btn btn-secondary" 
            disabled={page === 1}
            onClick={() => fetchLogs(page - 1)}
          >
            Previous
          </button>
          <span>Page {page} of {data.totalPages}</span>
          <button 
            className="btn btn-secondary" 
            disabled={page === data.totalPages}
            onClick={() => fetchLogs(page + 1)}
          >
            Next
          </button>
        </div>
      )}
    </div>
  );
};
