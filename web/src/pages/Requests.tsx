import React, { useEffect, useState, useCallback } from 'react';
import { Search, CheckCircle, XCircle, Clock, ChevronLeft, ChevronRight, RefreshCw, User, Inbox } from 'lucide-react';
import { useAuth } from '../contexts/AuthContext';
import { api } from '../services/api';
import './Requests.css';

interface RequestDTO {
  id: string;
  user_id: number;
  username: string;
  first_name: string;
  last_name: string;
  batch_id: number;
  batch_name: string;
  requested_at: string;
  status: string;
}

interface ListRequestsResponse {
  items: RequestDTO[];
  page: number;
  pageSize: number;
  total: number;
  totalPages: number;
}

export const Requests: React.FC = () => {
  const [data, setData] = useState<ListRequestsResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [search, setSearch] = useState('');
  const [debouncedSearch, setDebouncedSearch] = useState('');
  
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(25);

  const [selectedRequest, setSelectedRequest] = useState<RequestDTO | null>(null);

  const { user } = useAuth();
  const canManage = user?.role === 'OWNER' || user?.role === 'ADMIN';

  useEffect(() => {
    const handler = setTimeout(() => {
      setDebouncedSearch(search);
      setPage(1);
    }, 400);
    return () => clearTimeout(handler);
  }, [search]);

  const fetchRequests = useCallback(async () => {
    setLoading(true);
    try {
      const query = new URLSearchParams({
        page: page.toString(),
        pageSize: pageSize.toString(),
        search: debouncedSearch
      });
      const res = await api.get<ListRequestsResponse>(`/requests?${query.toString()}`);
      setData(res);
      setError(null);
    } catch (err: any) {
      setError(err.message || 'Failed to load requests');
    } finally {
      setLoading(false);
    }
  }, [page, pageSize, debouncedSearch]);

  useEffect(() => {
    fetchRequests();
    
    // Listen for realtime updates
    const handleRealtime = (e: CustomEvent) => {
      if (e.detail?.type === 'REQUEST_APPROVED' || e.detail?.type === 'REQUEST_REJECTED') {
        fetchRequests(); // Refresh list on change
      }
    };
    
    window.addEventListener('REALTIME_EVENT' as any, handleRealtime);
    return () => window.removeEventListener('REALTIME_EVENT' as any, handleRealtime);
  }, [fetchRequests]);

  const handleRowClick = (req: RequestDTO) => {
    setSelectedRequest(req);
  };

  return (
    <div className="requests-page fade-in">
      <div className="requests-header flex items-center justify-between">
        <div>
          <h2>Request Center</h2>
          <p className="subtitle">Manage pending batch join requests</p>
        </div>
        <div className="requests-stats glass-panel">
          <span className="stat-label">Pending:</span>
          <span className="stat-value text-warning">{data?.total || 0}</span>
        </div>
      </div>

      <div className="requests-controls glass-panel">
        <div className="search-bar">
          <Search size={18} className="search-icon" />
          <input 
            type="text" 
            placeholder="Search by ID, name, or username..." 
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        </div>
        
        <div className="filters flex items-center gap-4">
          <div className="page-size-selector">
            <span className="text-muted text-sm mr-2">Show:</span>
            <select 
              value={pageSize} 
              onChange={(e) => {
                setPageSize(Number(e.target.value));
                setPage(1);
              }}
            >
              <option value="10">10</option>
              <option value="25">25</option>
              <option value="50">50</option>
              <option value="100">100</option>
            </select>
          </div>
          
          <button className="btn btn-secondary btn-icon" onClick={fetchRequests} title="Refresh">
            <RefreshCw size={16} className={loading ? 'spin' : ''} />
          </button>
        </div>
      </div>

      <div className="requests-content glass-panel">
        {loading && !data ? (
          <div className="requests-loading">
            {[1,2,3,4,5].map(i => <div key={i} className="skeleton-row"></div>)}
          </div>
        ) : error ? (
          <div className="requests-error">
            <XCircle size={48} className="text-error mb-4" />
            <h3>Error loading requests</h3>
            <p>{error}</p>
            <button className="btn btn-primary mt-4" onClick={fetchRequests}>Try Again</button>
          </div>
        ) : data?.items.length === 0 ? (
          <div className="requests-empty">
            <Inbox size={48} className="text-muted mb-4" />
            <h3>No Pending Requests</h3>
            <p>You're all caught up! No active join requests found.</p>
          </div>
        ) : (
          <>
            <div className="table-responsive">
              <table className="requests-table">
                <thead>
                  <tr>
                    <th>User</th>
                    <th>Telegram ID</th>
                    <th>Batch</th>
                    <th>Requested At</th>
                    <th>Status</th>
                  </tr>
                </thead>
                <tbody>
                  {data?.items.map(req => (
                    <tr key={req.id} onClick={() => handleRowClick(req)} className="clickable-row">
                      <td>
                        <div className="user-name">
                          {req.first_name} {req.last_name}
                          {req.username && <span className="user-handle">@{req.username}</span>}
                        </div>
                      </td>
                      <td><code>{req.user_id}</code></td>
                      <td><span className="badge badge-secondary">{req.batch_name}</span></td>
                      <td>
                        <div className="flex items-center gap-2 text-sm text-muted">
                          <Clock size={14} />
                          {new Date(req.requested_at).toLocaleString()}
                        </div>
                      </td>
                      <td>
                        <span className="badge badge-warning">Pending</span>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            {data && data.totalPages > 1 && (
              <div className="pagination">
                <span className="page-info">
                  Showing {(page - 1) * pageSize + 1} to {Math.min(page * pageSize, data.total)} of {data.total}
                </span>
                <div className="page-controls">
                  <button 
                    disabled={page === 1}
                    onClick={() => setPage(p => p - 1)}
                    className="page-btn"
                  >
                    <ChevronLeft size={16} />
                  </button>
                  <span className="current-page">Page {page} of {data.totalPages}</span>
                  <button 
                    disabled={page >= data.totalPages}
                    onClick={() => setPage(p => p + 1)}
                    className="page-btn"
                  >
                    <ChevronRight size={16} />
                  </button>
                </div>
              </div>
            )}
          </>
        )}
      </div>

      {selectedRequest && (
        <RequestDetailModal 
          request={selectedRequest}
          canManage={canManage}
          onClose={() => setSelectedRequest(null)}
          onProcessed={() => {
            setSelectedRequest(null);
            fetchRequests();
          }}
        />
      )}
    </div>
  );
};

interface RequestDetailModalProps {
  request: RequestDTO;
  canManage: boolean;
  onClose: () => void;
  onProcessed: () => void;
}

const RequestDetailModal: React.FC<RequestDetailModalProps> = ({ request, canManage, onClose, onProcessed }) => {
  const [mutating, setMutating] = useState(false);
  const [confirmAction, setConfirmAction] = useState<'APPROVE' | 'REJECT' | null>(null);

  const handleAction = async () => {
    if (!confirmAction) return;
    
    setMutating(true);
    try {
      if (confirmAction === 'APPROVE') {
        await api.post(`/requests/${request.id}/approve`, {});
      } else if (confirmAction === 'REJECT') {
        await api.post(`/requests/${request.id}/reject`, {});
      }
      onProcessed();
    } catch (err: any) {
      alert(err.message || 'Action failed');
    } finally {
      setMutating(false);
      setConfirmAction(null);
    }
  };

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal-content glass-panel request-modal" onClick={e => e.stopPropagation()}>
        <button className="modal-close" onClick={onClose}><XCircle size={24} /></button>
        
        <div className="request-details">
          <div className="request-header-large">
            <div className="request-icon">
              <User size={32} />
            </div>
            <div className="request-title">
              <h3>{request.first_name} {request.last_name}</h3>
              <div className="request-meta">
                <code className="id-badge copyable" onClick={() => navigator.clipboard.writeText(request.user_id.toString())}>
                  {request.user_id}
                </code>
                {request.username && <span className="badge badge-secondary">@{request.username}</span>}
              </div>
            </div>
          </div>

          <div className="request-section">
            <h4>Request Information</h4>
            <div className="info-grid">
              <div className="info-item">
                <span className="info-label">Batch</span>
                <span className="info-value">{request.batch_name}</span>
              </div>
              <div className="info-item">
                <span className="info-label">Batch ID</span>
                <span className="info-value"><code>{request.batch_id}</code></span>
              </div>
              <div className="info-item">
                <span className="info-label">Requested At</span>
                <span className="info-value">{new Date(request.requested_at).toLocaleString()}</span>
              </div>
              <div className="info-item">
                <span className="info-label">Status</span>
                <span className="badge badge-warning">Pending</span>
              </div>
            </div>
          </div>

          {canManage && (
            <div className="request-actions-panel">
              <button 
                className="btn btn-danger btn-lg flex-1 flex items-center justify-center gap-2"
                onClick={() => setConfirmAction('REJECT')}
              >
                <XCircle size={18} />
                Reject Request
              </button>
              <button 
                className="btn btn-primary btn-lg flex-1 flex items-center justify-center gap-2"
                onClick={() => setConfirmAction('APPROVE')}
              >
                <CheckCircle size={18} />
                Approve Access
              </button>
            </div>
          )}
        </div>

        {confirmAction && (
          <div className="confirm-modal-overlay">
            <div className="confirm-modal glass-panel">
              <h4>Confirm Action</h4>
              <p>
                {confirmAction === 'APPROVE' && `Are you sure you want to approve this request? This will grant ${request.first_name} permanent access to ${request.batch_name}.`}
                {confirmAction === 'REJECT' && `Are you sure you want to reject this request from ${request.first_name}?`}
              </p>
              <div className="confirm-actions">
                <button className="btn btn-secondary" onClick={() => setConfirmAction(null)} disabled={mutating}>Cancel</button>
                <button 
                  className={`btn ${confirmAction === 'REJECT' ? 'btn-danger' : 'btn-primary'}`} 
                  onClick={handleAction} 
                  disabled={mutating}
                >
                  {mutating ? 'Processing...' : (confirmAction === 'APPROVE' ? 'Approve Access' : 'Reject Request')}
                </button>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};
