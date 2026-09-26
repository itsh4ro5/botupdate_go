import React, { useEffect, useState } from 'react';
import { Shield, Key, LogOut, Smartphone, AlertCircle } from 'lucide-react';
import { api } from '../services/api';
import { useAuth } from '../contexts/AuthContext';
import './SecuritySettings.css';

interface Session {
  id: string;
  created_at: string;
  expires_at: string;
  is_current: boolean;
}

export const SecuritySettings: React.FC = () => {
  const { user, logout } = useAuth();
  const [sessions, setSessions] = useState<Session[]>([]);
  const [currentPassword, setCurrentPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');

  const loadSessions = async () => {
    if (!user) return;
    try {
      const res = await api.get<Session[]>(`/admins/${user.username}/sessions`);
      setSessions(res);
    } catch (e) {
      console.error(e);
    }
  };

  useEffect(() => {
    loadSessions();
  }, [user]);

  const handleChangePassword = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setSuccess('');
    
    if (newPassword.length < 10) {
      setError('Password must be at least 10 characters.');
      return;
    }

    try {
      await api.post('/auth/change-password', {
        current_password: currentPassword,
        new_password: newPassword
      });
      setSuccess('Password changed successfully.');
      setCurrentPassword('');
      setNewPassword('');
    } catch (e: any) {
      setError(e.response?.data?.error || 'Failed to change password.');
    }
  };

  const handleRevoke = async (id: string) => {
    try {
      await api.post(`/admins/${user?.username}/sessions/${id}/revoke`, {});
      loadSessions();
    } catch (e) {
      alert('Failed to revoke session');
    }
  };

  const handleRevokeAll = async () => {
    if (!window.confirm('This will log you out immediately. Proceed?')) return;
    try {
      await api.post(`/admins/${user?.username}/sessions/revoke-all`, {});
      logout();
    } catch (e) {
      alert('Failed to revoke sessions');
    }
  };

  return (
    <div className="security-page fade-in">
      <div className="page-header">
        <h1>Security Settings</h1>
      </div>

      <div className="security-card">
        <h2 className="flex items-center gap-2 text-xl font-bold">
          <Shield size={24} className="text-accent" /> Account Security
        </h2>
        
        <form onSubmit={handleChangePassword} className="password-form">
          {error && <div className="text-error flex gap-2 text-sm"><AlertCircle size={16}/> {error}</div>}
          {success && <div className="text-success text-sm">{success}</div>}
          
          <div className="form-group">
            <label>Current Password</label>
            <input 
              type="password" 
              className="input" 
              value={currentPassword}
              onChange={e => setCurrentPassword(e.target.value)}
              required 
            />
          </div>
          <div className="form-group">
            <label>New Password</label>
            <input 
              type="password" 
              className="input" 
              value={newPassword}
              onChange={e => setNewPassword(e.target.value)}
              required 
            />
          </div>
          <button type="submit" className="btn btn-primary" style={{ alignSelf: 'flex-start' }}>
            <Key size={16} /> Update Password
          </button>
        </form>
      </div>

      <div className="security-card">
        <div className="flex items-center justify-between">
          <h2 className="flex items-center gap-2 text-xl font-bold">
            <Smartphone size={24} className="text-secondary" /> Active Sessions
          </h2>
          <button className="btn btn-danger" onClick={handleRevokeAll}>
            Revoke All
          </button>
        </div>

        <div className="session-list">
          {sessions.map(s => (
            <div key={s.id} className="session-item">
              <div className="session-info">
                <span className="font-bold flex items-center gap-2">
                  Session {s.id.substring(0,8)}...
                  {s.is_current && <span className="badge badge-primary text-xs">Current</span>}
                </span>
                <span className="session-meta">Expires: {new Date(s.expires_at).toLocaleString()}</span>
              </div>
              {!s.is_current && (
                <button className="btn btn-danger text-sm" onClick={() => handleRevoke(s.id)}>
                  <LogOut size={14} /> Revoke
                </button>
              )}
            </div>
          ))}
        </div>
      </div>
    </div>
  );
};
