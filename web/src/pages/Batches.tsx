import React, { useEffect, useState, useCallback } from 'react';
import { Search, Filter, RefreshCw, ChevronLeft, ChevronRight, XCircle, Users, BookOpen } from 'lucide-react';
import { useAuth } from '../contexts/AuthContext';
import { api } from '../services/api';
import './Batches.css';

interface BatchDTO {
  id: number;
  name: string;
  category: string;
  type: string;
  welcome_text: string;
  user_count: number;
}

interface BatchUserDTO {
  id: number;
  username: string;
  first_name: string;
  last_name: string;
  tier: string;
}

interface BatchDetailDTO {
  batch: BatchDTO;
  users: BatchUserDTO[];
}

interface ListBatchesResponse {
  batches: BatchDTO[];
  total: number;
}

export const Batches: React.FC = () => {
  const [data, setData] = useState<ListBatchesResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [search, setSearch] = useState('');
  const [debouncedSearch, setDebouncedSearch] = useState('');
  const [typeFilter, setTypeFilter] = useState('');
  const [categoryFilter] = useState('');
  const [selectedBatchId, setSelectedBatchId] = useState<number | null>(null);

  // Pagination for client side
  const [page, setPage] = useState(1);
  const pageSize = 10;

  // Debounce search
  useEffect(() => {
    const handler = setTimeout(() => {
      setDebouncedSearch(search);
      setPage(1);
    }, 400);
    return () => clearTimeout(handler);
  }, [search]);

  const fetchBatches = useCallback(async () => {
    setLoading(true);
    try {
      const query = new URLSearchParams({
        search: debouncedSearch,
        type: typeFilter,
        category: categoryFilter
      });
      const res = await api.get<ListBatchesResponse>(`/batches?${query.toString()}`);
      setData(res);
      setError(null);
    } catch (err: any) {
      setError(err.message || 'Failed to load batches');
    } finally {
      setLoading(false);
    }
  }, [debouncedSearch, typeFilter, categoryFilter]);

  useEffect(() => {
    fetchBatches();
  }, [fetchBatches]);

  const batchesToDisplay = data?.batches.slice((page - 1) * pageSize, page * pageSize) || [];
  const totalPages = data ? Math.ceil(data.total / pageSize) : 1;

  return (
    <div className="batches-page fade-in">
      <div className="batches-header flex items-center justify-between">
        <div>
          <h2>Batch Directory</h2>
          <p className="subtitle">Manage batch configurations and access controls</p>
        </div>
        <div className="batches-stats glass-panel">
          <span className="stat-label">Total Batches:</span>
          <span className="stat-value">{data?.total || '--'}</span>
        </div>
      </div>

      <div className="batches-controls glass-panel">
        <div className="search-bar">
          <Search size={18} className="search-icon" />
          <input 
            type="text" 
            placeholder="Search by ID or name..." 
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        </div>
        
        <div className="filters">
          <div className="filter-group">
            <Filter size={16} />
            <select 
              value={typeFilter} 
              onChange={(e) => setTypeFilter(e.target.value)}
            >
              <option value="">All Types</option>
              <option value="free">Free</option>
              <option value="paid">Paid</option>
              <option value="special">Special</option>
            </select>
          </div>
          
          <button className="btn btn-secondary btn-icon" onClick={() => fetchBatches()}>
            <RefreshCw size={16} className={loading ? 'spin' : ''} />
          </button>
        </div>
      </div>

      <div className="batches-content glass-panel">
        {loading && !data ? (
          <div className="batches-loading">
            {[1,2,3,4,5].map(i => <div key={i} className="skeleton-row"></div>)}
          </div>
        ) : error ? (
          <div className="batches-error">
            <XCircle size={48} className="text-error mb-4" />
            <h3>Error loading batches</h3>
            <p>{error}</p>
            <button className="btn btn-primary mt-4" onClick={fetchBatches}>Try Again</button>
          </div>
        ) : data?.batches.length === 0 ? (
          <div className="batches-empty">
            <BookOpen size={48} className="text-muted mb-4" />
            <h3>No batches found</h3>
            <p>Try adjusting your search or filters.</p>
          </div>
        ) : (
          <>
            <div className="batches-grid">
              {batchesToDisplay.map(batch => (
                <div key={batch.id} className="batch-card" onClick={() => setSelectedBatchId(batch.id)}>
                  <div className="batch-card-header">
                    <h4>{batch.name}</h4>
                    <span className={`badge badge-${batch.type === 'paid' ? 'success' : batch.type === 'special' ? 'accent' : 'info'}`}>
                      {batch.type.toUpperCase()}
                    </span>
                  </div>
                  <div className="batch-card-body">
                    <div className="info-row">
                      <span className="label">ID:</span>
                      <code className="value">{batch.id}</code>
                    </div>
                    <div className="info-row">
                      <span className="label">Category:</span>
                      <span className="value">{batch.category || 'N/A'}</span>
                    </div>
                  </div>
                  <div className="batch-card-footer">
                    <div className="user-count">
                      <Users size={14} />
                      <span>{batch.user_count} Users</span>
                    </div>
                    <span className="view-link">View Details &rarr;</span>
                  </div>
                </div>
              ))}
            </div>

            <div className="pagination">
              <span className="page-info">
                Showing {(page - 1) * pageSize + 1} to {Math.min(page * pageSize, data!.total)} of {data!.total}
              </span>
              <div className="page-controls">
                <button 
                  disabled={page === 1}
                  onClick={() => setPage(p => p - 1)}
                  className="page-btn"
                >
                  <ChevronLeft size={16} />
                </button>
                <span className="current-page">Page {page} of {totalPages}</span>
                <button 
                  disabled={page >= totalPages}
                  onClick={() => setPage(p => p + 1)}
                  className="page-btn"
                >
                  <ChevronRight size={16} />
                </button>
              </div>
            </div>
          </>
        )}
      </div>

      {selectedBatchId && (
        <BatchProfileModal 
          batchId={selectedBatchId} 
          onClose={() => {
            setSelectedBatchId(null);
            fetchBatches(); // Refresh list to get updated user count
          }} 
        />
      )}
    </div>
  );
};

interface BatchProfileModalProps {
  batchId: number;
  onClose: () => void;
}

const BatchProfileModal: React.FC<BatchProfileModalProps> = ({ batchId, onClose }) => {
  const [detail, setDetail] = useState<BatchDetailDTO | null>(null);
  const [loading, setLoading] = useState(true);
  const [mutating, setMutating] = useState(false);
  const [search, setSearch] = useState('');
  
  // Access granting
  const [grantUserId, setGrantUserId] = useState('');
  
  const [confirmAction, setConfirmAction] = useState<{type: 'GRANT' | 'REVOKE', userId: number} | null>(null);
  
  const { user } = useAuth();
  const canManage = user?.role === 'OWNER' || user?.role === 'ADMIN';

  const fetchDetail = useCallback(async () => {
    try {
      const res = await api.get<BatchDetailDTO>(`/batches/${batchId}`);
      setDetail(res);
    } catch (err) {
      console.error("Failed to load batch details", err);
    } finally {
      setLoading(false);
    }
  }, [batchId]);

  useEffect(() => {
    fetchDetail();
  }, [fetchDetail]);

  const handleAction = async () => {
    if (!confirmAction) return;
    
    setMutating(true);
    try {
      if (confirmAction.type === 'GRANT') {
        await api.post(`/batches/${batchId}/access/grant`, { user_id: confirmAction.userId });
        setGrantUserId(''); // Clear input
      } else if (confirmAction.type === 'REVOKE') {
        await api.post(`/batches/${batchId}/access/revoke`, { user_id: confirmAction.userId });
      }
      setConfirmAction(null);
      await fetchDetail(); // Refresh details
    } catch (err: any) {
      alert(err.message || 'Action failed');
    } finally {
      setMutating(false);
    }
  };

  const handleInitiateGrant = () => {
    const id = parseInt(grantUserId, 10);
    if (isNaN(id) || id <= 0) {
      alert('Please enter a valid numeric User ID');
      return;
    }
    setConfirmAction({ type: 'GRANT', userId: id });
  };

  const filteredUsers = detail?.users.filter(u => 
    u.id.toString().includes(search) || 
    u.first_name.toLowerCase().includes(search.toLowerCase()) || 
    (u.username && u.username.toLowerCase().includes(search.toLowerCase()))
  ) || [];

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal-content glass-panel batch-modal" onClick={e => e.stopPropagation()}>
        <button className="modal-close" onClick={onClose}><XCircle size={24} /></button>
        
        {loading ? (
          <div className="profile-loading">
            <div className="skeleton-line" style={{height: '30px', width: '50%'}}></div>
            <div className="skeleton-line"></div>
            <div className="skeleton-line short"></div>
          </div>
        ) : detail ? (
          <div className="batch-details">
            <div className="batch-header-large">
              <div className="batch-icon">
                <BookOpen size={32} />
              </div>
              <div className="batch-title">
                <h3>{detail.batch.name}</h3>
                <div className="batch-meta">
                  <code className="id-badge copyable" onClick={() => navigator.clipboard.writeText(detail.batch.id.toString())}>
                    {detail.batch.id}
                  </code>
                  <span className={`badge badge-${detail.batch.type === 'paid' ? 'success' : detail.batch.type === 'special' ? 'accent' : 'info'}`}>
                    {detail.batch.type.toUpperCase()}
                  </span>
                  {detail.batch.category && <span className="badge badge-secondary">{detail.batch.category}</span>}
                </div>
              </div>
            </div>

            <div className="batch-section">
              <h4>Access Management</h4>
              <div className="access-panel">
                <div className="access-header">
                  <div className="search-bar small">
                    <Search size={14} className="search-icon" />
                    <input 
                      type="text" 
                      placeholder="Search users..." 
                      value={search}
                      onChange={(e) => setSearch(e.target.value)}
                    />
                  </div>
                  
                  {canManage && (
                    <div className="grant-input-group">
                      <input 
                        type="text" 
                        placeholder="User ID to grant..."
                        value={grantUserId}
                        onChange={(e) => setGrantUserId(e.target.value)}
                      />
                      <button className="btn btn-primary btn-sm" onClick={handleInitiateGrant} disabled={!grantUserId}>
                        Grant Access
                      </button>
                    </div>
                  )}
                </div>

                <div className="access-list-container">
                  {filteredUsers.length === 0 ? (
                    <div className="empty-state">No users match your search.</div>
                  ) : (
                    <table className="access-table">
                      <thead>
                        <tr>
                          <th>User</th>
                          <th>ID</th>
                          <th>Tier</th>
                          {canManage && <th className="text-right">Action</th>}
                        </tr>
                      </thead>
                      <tbody>
                        {filteredUsers.map(u => (
                          <tr key={u.id}>
                            <td>
                              <div className="user-name">
                                {u.first_name} {u.last_name}
                                {u.username && <span className="user-handle">@{u.username}</span>}
                              </div>
                            </td>
                            <td><code>{u.id}</code></td>
                            <td><span className="badge badge-secondary">{u.tier || 'standard'}</span></td>
                            {canManage && (
                              <td className="text-right">
                                <button 
                                  className="btn btn-danger btn-sm"
                                  onClick={() => setConfirmAction({type: 'REVOKE', userId: u.id})}
                                >
                                  Revoke
                                </button>
                              </td>
                            )}
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  )}
                </div>
              </div>
            </div>
          </div>
        ) : (
          <div className="profile-error">Batch not found</div>
        )}

        {confirmAction && (
          <div className="confirm-modal-overlay">
            <div className="confirm-modal glass-panel">
              <h4>Confirm Action</h4>
              <p>
                {confirmAction.type === 'GRANT' && `Are you sure you want to GRANT access to User ID ${confirmAction.userId} for this batch?`}
                {confirmAction.type === 'REVOKE' && `Are you sure you want to REVOKE access for User ID ${confirmAction.userId}?`}
              </p>
              <div className="confirm-actions">
                <button className="btn btn-secondary" onClick={() => setConfirmAction(null)} disabled={mutating}>Cancel</button>
                <button className={`btn ${confirmAction.type === 'REVOKE' ? 'btn-danger' : 'btn-primary'}`} onClick={handleAction} disabled={mutating}>
                  {mutating ? 'Processing...' : 'Confirm'}
                </button>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};
