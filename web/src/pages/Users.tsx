import React, { useEffect, useState, useCallback } from 'react';
import { Search, Filter, RefreshCw, ChevronLeft, ChevronRight, User as UserIcon, CheckCircle, XCircle, ShieldAlert, Shield } from 'lucide-react';
import { useAuth } from '../contexts/AuthContext';
import { api } from '../services/api';
import './Users.css';

interface UserListItem {
  id: number;
  username: string;
  first_name: string;
  last_name: string;
  joined_at: number;
  is_blocked: boolean;
}

interface ListUsersResponse {
  users: UserListItem[];
  total: number;
  page: number;
  pageSize: number;
  totalPages: number;
}

export const Users: React.FC = () => {
  const [data, setData] = useState<ListUsersResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [page, setPage] = useState(1);
  const [pageSize] = useState(25);
  const [search, setSearch] = useState('');
  const [debouncedSearch, setDebouncedSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [sortField, setSortField] = useState('joined_at_desc');
  const [selectedUser, setSelectedUser] = useState<number | null>(null);

  // Debounce search
  useEffect(() => {
    const handler = setTimeout(() => {
      setDebouncedSearch(search);
      setPage(1); // Reset page on new search
    }, 400);
    return () => clearTimeout(handler);
  }, [search]);

  const fetchUsers = useCallback(async () => {
    setLoading(true);
    try {
      const query = new URLSearchParams({
        page: page.toString(),
        pageSize: pageSize.toString(),
        search: debouncedSearch,
        sort: sortField,
        status: statusFilter
      });
      const res = await api.get<ListUsersResponse>(`/users?${query.toString()}`);
      setData(res);
      setError(null);
    } catch (err: any) {
      setError(err.message || 'Failed to load users');
    } finally {
      setLoading(false);
    }
  }, [page, pageSize, debouncedSearch, sortField, statusFilter]);

  useEffect(() => {
    fetchUsers();
  }, [fetchUsers]);

  const handleRefresh = () => {
    fetchUsers();
  };

  return (
    <div className="users-page fade-in">
      <div className="users-header flex items-center justify-between">
        <div>
          <h2>User Directory</h2>
          <p className="subtitle">Manage and search registered bot users</p>
        </div>
        <div className="users-stats glass-panel">
          <span className="stat-label">Total Users:</span>
          <span className="stat-value">{data?.total || '--'}</span>
        </div>
      </div>

      <div className="users-controls glass-panel">
        <div className="search-bar">
          <Search size={18} className="search-icon" />
          <input 
            type="text" 
            placeholder="Search by ID, username, or name..." 
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        </div>
        
        <div className="filters">
          <div className="filter-group">
            <Filter size={16} />
            <select 
              value={statusFilter} 
              onChange={(e) => { setStatusFilter(e.target.value); setPage(1); }}
            >
              <option value="">All Statuses</option>
              <option value="active">Active</option>
              <option value="blocked">Blocked</option>
            </select>
          </div>
          
          <div className="filter-group">
            <select 
              value={sortField} 
              onChange={(e) => { setSortField(e.target.value); setPage(1); }}
            >
              <option value="joined_at_desc">Newest First</option>
              <option value="joined_at_asc">Oldest First</option>
              <option value="name">Name (A-Z)</option>
            </select>
          </div>

          <button className="icon-btn" onClick={handleRefresh} title="Refresh">
            <RefreshCw size={18} />
          </button>
        </div>
      </div>

      <div className="users-table-container glass-panel">
        {loading && !data ? (
          <div className="users-loading">
            {[...Array(10)].map((_, i) => (
              <div key={i} className="skeleton-row"></div>
            ))}
          </div>
        ) : error ? (
          <div className="users-error">
            <XCircle size={32} />
            <p>{error}</p>
            <button onClick={handleRefresh} className="btn-primary">Try Again</button>
          </div>
        ) : data?.users.length === 0 ? (
          <div className="users-empty">
            <UserIcon size={48} />
            <h3>No users found</h3>
            <p>Try adjusting your search or filters.</p>
          </div>
        ) : (
          <>
            <div className="table-responsive">
              <table className="users-table">
                <thead>
                  <tr>
                    <th>User</th>
                    <th>Telegram ID</th>
                    <th>Joined</th>
                    <th>Status</th>
                  </tr>
                </thead>
                <tbody>
                  {data?.users.map(user => (
                    <tr key={user.id} onClick={() => setSelectedUser(user.id)}>
                      <td>
                        <div className="user-cell">
                          <div className="user-avatar">
                            {user.first_name ? user.first_name.charAt(0).toUpperCase() : <UserIcon size={16} />}
                          </div>
                          <div className="user-info">
                            <span className="user-name">{user.first_name} {user.last_name}</span>
                            {user.username && <span className="user-handle">@{user.username}</span>}
                          </div>
                        </div>
                      </td>
                      <td>
                        <code className="id-badge">{user.id}</code>
                      </td>
                      <td>
                        <span className="date-cell">
                          {user.joined_at > 0 ? new Date(user.joined_at * 1000).toLocaleDateString() : 'Unknown'}
                        </span>
                      </td>
                      <td>
                        {user.is_blocked ? (
                          <span className="badge badge-error">Blocked</span>
                        ) : (
                          <span className="badge badge-success">Active</span>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
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
                <span className="current-page">Page {page} of {data!.totalPages}</span>
                <button 
                  disabled={page >= data!.totalPages}
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

      {selectedUser && (
        <UserProfileModal 
          userId={selectedUser} 
          onClose={() => setSelectedUser(null)} 
        />
      )}
    </div>
  );
};

interface UserProfileModalProps {
  userId: number;
  onClose: () => void;
}

interface UserProfileData {
  id: number;
  username: string;
  first_name: string;
  last_name: string;
  joined_at: number;
  is_blocked: boolean;
  tnc_accepted: boolean;
  referral_count: number;
  total_invited: number;
  tier: string;
  welcome_bonus_claimed: boolean;
  free_unlocked: boolean;
  unlocked_batches: string[];
}

const UserProfileModal: React.FC<UserProfileModalProps> = ({ userId, onClose }) => {
  const [profile, setProfile] = useState<UserProfileData | null>(null);
  const [loading, setLoading] = useState(true);
  const [mutating, setMutating] = useState(false);
  const [confirmAction, setConfirmAction] = useState<{type: 'BLOCK' | 'UNBLOCK' | 'TIER', tier?: string} | null>(null);
  const { user } = useAuth();

  const canManage = user?.role === 'OWNER' || user?.role === 'ADMIN';

  useEffect(() => {
    const fetchProfile = async () => {
      try {
        const res = await api.get<UserProfileData>(`/users/${userId}`);
        setProfile(res);
      } catch (err) {
        console.error("Failed to load profile", err);
      } finally {
        setLoading(false);
      }
    };
    fetchProfile();
  }, [userId]);

  const handleAction = async () => {
    if (!confirmAction || !profile) return;
    
    setMutating(true);
    try {
      if (confirmAction.type === 'BLOCK') {
        const res = await api.post<UserProfileData>(`/users/${profile.id}/block`, {});
        setProfile(res);
      } else if (confirmAction.type === 'UNBLOCK') {
        const res = await api.post<UserProfileData>(`/users/${profile.id}/unblock`, {});
        setProfile(res);
      } else if (confirmAction.type === 'TIER') {
        const res = await api.post<UserProfileData>(`/users/${profile.id}/tier`, { tier: confirmAction.tier });
        setProfile(res);
      }
      setConfirmAction(null);
    } catch (err: any) {
      alert(err.message || 'Action failed');
    } finally {
      setMutating(false);
    }
  };

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal-content glass-panel" onClick={e => e.stopPropagation()}>
        <button className="modal-close" onClick={onClose}><XCircle size={24} /></button>
        
        {loading ? (
          <div className="profile-loading">
            <div className="skeleton-avatar"></div>
            <div className="skeleton-line"></div>
            <div className="skeleton-line short"></div>
          </div>
        ) : profile ? (
          <div className="profile-details">
            <div className="profile-header">
              <div className="profile-avatar large">
                {profile.first_name ? profile.first_name.charAt(0).toUpperCase() : <UserIcon size={32} />}
              </div>
              <div>
                <h3>{profile.first_name} {profile.last_name}</h3>
                {profile.username && <p className="profile-handle">@{profile.username}</p>}
                <code className="id-badge copyable" onClick={() => navigator.clipboard.writeText(profile.id.toString())}>
                  {profile.id}
                </code>
              </div>
            </div>

            <div className="profile-grid">
              <div className="profile-section">
                <h4>Identity & Status</h4>
                <div className="info-row">
                  <span>Status</span>
                  {profile.is_blocked ? <span className="badge badge-error">Blocked</span> : <span className="badge badge-success">Active</span>}
                </div>
                <div className="info-row">
                  <span>Joined</span>
                  <span>{profile.joined_at > 0 ? new Date(profile.joined_at * 1000).toLocaleString() : 'Unknown'}</span>
                </div>
                <div className="info-row">
                  <span>Tier</span>
                  <span>{profile.tier || 'Standard'}</span>
                </div>
                <div className="info-row">
                  <span>TnC Accepted</span>
                  {profile.tnc_accepted ? <CheckCircle size={16} className="text-success" /> : <XCircle size={16} className="text-error" />}
                </div>
              </div>

              <div className="profile-section">
                <h4>Access & Rewards</h4>
                <div className="info-row">
                  <span>Free Batches Unlocked</span>
                  {profile.free_unlocked ? <CheckCircle size={16} className="text-success" /> : <XCircle size={16} className="text-error" />}
                </div>
                <div className="info-row">
                  <span>Welcome Bonus</span>
                  {profile.welcome_bonus_claimed ? <CheckCircle size={16} className="text-success" /> : <XCircle size={16} className="text-error" />}
                </div>
                <div className="info-row">
                  <span>Total Referrals</span>
                  <span>{profile.referral_count}</span>
                </div>
                <div className="info-row">
                  <span>Total Invited</span>
                  <span>{profile.total_invited}</span>
                </div>
              </div>
            </div>
            
            {profile.unlocked_batches && profile.unlocked_batches.length > 0 && (
              <div className="profile-section full-width">
                <h4>Unlocked Batches</h4>
                <div className="batch-tags">
                  {profile.unlocked_batches.map(b => (
                    <span key={b} className="batch-tag">{b}</span>
                  ))}
                </div>
              </div>
            )}

            {canManage && (
              <div className="profile-section full-width manage-section">
                <h4>User Management</h4>
                <div className="manage-actions">
                  {profile.is_blocked ? (
                    <button className="btn btn-secondary action-btn" onClick={() => setConfirmAction({type: 'UNBLOCK'})}>
                      <Shield size={16} /> Unblock User
                    </button>
                  ) : (
                    <button className="btn btn-danger action-btn" onClick={() => setConfirmAction({type: 'BLOCK'})}>
                      <ShieldAlert size={16} /> Block User
                    </button>
                  )}
                  
                  <div className="tier-actions">
                    <span className="text-sm">Change Tier:</span>
                    <button 
                      className={`btn btn-secondary btn-sm ${profile.tier !== 'vip' ? 'active' : ''}`}
                      onClick={() => setConfirmAction({type: 'TIER', tier: ''})}
                      disabled={profile.tier !== 'vip'}
                    >
                      STANDARD
                    </button>
                    <button 
                      className={`btn btn-secondary btn-sm ${profile.tier === 'vip' ? 'active' : ''}`}
                      onClick={() => setConfirmAction({type: 'TIER', tier: 'vip'})}
                      disabled={profile.tier === 'vip'}
                    >
                      VIP
                    </button>
                  </div>
                </div>
              </div>
            )}
          </div>
        ) : (
          <div className="profile-error">User not found</div>
        )}

        {confirmAction && (
          <div className="confirm-modal-overlay">
            <div className="confirm-modal glass-panel">
              <h4>Confirm Action</h4>
              <p>
                {confirmAction.type === 'BLOCK' && `Are you sure you want to block this user? They will lose access to the bot.`}
                {confirmAction.type === 'UNBLOCK' && `Are you sure you want to unblock this user?`}
                {confirmAction.type === 'TIER' && `Are you sure you want to change this user's tier to ${confirmAction.tier === 'vip' ? 'VIP' : 'STANDARD'}?`}
              </p>
              <div className="confirm-actions">
                <button className="btn btn-secondary" onClick={() => setConfirmAction(null)} disabled={mutating}>Cancel</button>
                <button className={`btn ${confirmAction.type === 'BLOCK' ? 'btn-danger' : 'btn-primary'}`} onClick={handleAction} disabled={mutating}>
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
