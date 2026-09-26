import React, { useEffect, useState } from 'react';
import { Shield, AlertCircle, Plus } from 'lucide-react';
import { api } from '../services/api';
import { useAuth } from '../contexts/AuthContext';
import './Admins.css';

interface AdminResponse {
  username: string;
  role: string;
  created_at: string;
  must_change_password: boolean;
  active_sessions: number;
}

export const Admins: React.FC = () => {
  const { user } = useAuth();
  const [admins, setAdmins] = useState<AdminResponse[]>([]);
  const [showModal, setShowModal] = useState(false);
  const [newUsername, setNewUsername] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [newRole, setNewRole] = useState('ADMIN');
  const [error, setError] = useState('');

  const fetchAdmins = async () => {
    try {
      const res = await api.get<AdminResponse[]>('/admins');
      setAdmins(res);
    } catch (e) {
      console.error(e);
    }
  };

  useEffect(() => {
    fetchAdmins();
  }, []);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    try {
      await api.post('/admins', {
        username: newUsername,
        password: newPassword,
        role: newRole
      });
      setShowModal(false);
      setNewUsername('');
      setNewPassword('');
      fetchAdmins();
    } catch (e: any) {
      setError(e.response?.data?.error || 'Failed to create admin');
    }
  };

  const handleChangeRole = async (username: string, currentRole: string) => {
    const role = window.prompt(`Change role for ${username}. Enter new role (OWNER, ADMIN, SUPPORT):`, currentRole);
    if (!role || role === currentRole) return;
    try {
      await api.put(`/admins/${username}/role`, { role: role.toUpperCase() });
      fetchAdmins();
    } catch (e: any) {
      alert(e.response?.data?.error || 'Failed to change role');
    }
  };

  const handleRevokeSessions = async (username: string) => {
    if (!window.confirm(`Are you sure you want to revoke all active sessions for ${username}?`)) return;
    try {
      await api.post(`/admins/${username}/sessions/revoke-all`, {});
      fetchAdmins();
    } catch (e: any) {
      alert(e.response?.data?.error || 'Failed to revoke sessions');
    }
  };

  const handleForcePasswordChange = async (username: string) => {
    if (!window.confirm(`Force password change for ${username}? This will log them out immediately.`)) return;
    try {
      await api.post(`/admins/${username}/force-password-change`, {});
      fetchAdmins();
    } catch (e: any) {
      alert(e.response?.data?.error || 'Failed to force password change');
    }
  };

  if (user?.role !== 'OWNER') {
    return <div className="p-8 text-center text-error">Access Denied. OWNER only.</div>;
  }

  return (
    <div className="admins-page fade-in">
      <div className="page-header">
        <h1>Admin Management</h1>
        <button className="btn btn-primary" onClick={() => setShowModal(true)}>
          <Plus size={16} /> New Admin
        </button>
      </div>

      <div className="admin-grid">
        {admins.map(a => (
          <div key={a.username} className={`admin-card role-${a.role}`}>
            <div className="admin-header">
              <div className="admin-info">
                <div className="admin-avatar">
                  {a.username.charAt(0).toUpperCase()}
                </div>
                <div className="admin-details">
                  <span className="admin-name">{a.username}</span>
                  <span className={`badge badge-outline`}>{a.role}</span>
                </div>
              </div>
            </div>
            
            <div className="admin-meta">
              <span>Status: <span className="text-muted">Unavailable</span></span>
              <span>Created: {a.created_at !== "0001-01-01T00:00:00Z" ? new Date(a.created_at).toLocaleDateString() : "Unavailable"}</span>
              <span>Active Sessions: <strong className="text-white">{a.active_sessions}</strong></span>
              {a.must_change_password && (
                <span className="text-warning flex items-center gap-1">
                  <AlertCircle size={14} /> Pending Password Change
                </span>
              )}
            </div>

            <div className="admin-actions flex flex-col gap-2">
              <button 
                className="btn btn-secondary w-full"
                onClick={() => handleChangeRole(a.username, a.role)}
              >
                <Shield size={14} /> Change Role
              </button>
              <div className="flex gap-2">
                <button 
                  className="btn btn-secondary flex-1"
                  onClick={() => handleRevokeSessions(a.username)}
                  disabled={a.active_sessions === 0}
                >
                  Revoke Sessions
                </button>
                <button 
                  className="btn btn-secondary flex-1"
                  onClick={() => handleForcePasswordChange(a.username)}
                >
                  Reset Password
                </button>
              </div>
            </div>
          </div>
        ))}
      </div>

      {showModal && (
        <div className="modal-overlay" onClick={() => setShowModal(false)}>
          <div className="modal-content" onClick={e => e.stopPropagation()}>
            <h2>Create Administrator</h2>
            <form onSubmit={handleCreate}>
              {error && <div className="error-banner">{error}</div>}
              
              <div className="form-group">
                <label>Username</label>
                <input 
                  type="text" 
                  className="input" 
                  value={newUsername} 
                  onChange={e => setNewUsername(e.target.value)} 
                  required 
                />
              </div>
              
              <div className="form-group">
                <label>Temporary Password</label>
                <input 
                  type="password" 
                  className="input" 
                  value={newPassword} 
                  onChange={e => setNewPassword(e.target.value)} 
                  required 
                  minLength={10}
                />
                <span className="text-xs text-muted">User will be forced to change this on first login. Min 10 chars.</span>
              </div>

              <div className="form-group">
                <label>Role</label>
                <select className="input" value={newRole} onChange={e => setNewRole(e.target.value)}>
                  <option value="ADMIN">ADMIN</option>
                  <option value="SUPPORT">SUPPORT</option>
                </select>
              </div>

              <div className="flex justify-end gap-2 mt-4">
                <button type="button" className="btn btn-secondary" onClick={() => setShowModal(false)}>Cancel</button>
                <button type="submit" className="btn btn-primary">Create Account</button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
};
