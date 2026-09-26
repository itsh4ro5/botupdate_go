import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { api } from '../services/api';
import { useAuth } from '../contexts/AuthContext';
import { Eye, EyeOff, Lock } from 'lucide-react';
import './Login.css'; // Reuse login card styles

export const ChangePassword: React.FC = () => {
  const [currentPassword, setCurrentPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [showCurrent, setShowCurrent] = useState(false);
  const [showNew, setShowNew] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');
  
  const { refreshSession, user } = useAuth();
  const navigate = useNavigate();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!currentPassword || !newPassword || !confirmPassword) {
      setError('Please fill all fields.');
      return;
    }
    if (newPassword !== confirmPassword) {
      setError('New passwords do not match.');
      return;
    }
    if (newPassword.length < 10) {
      setError('Password must be at least 10 characters long.');
      return;
    }

    setLoading(true);
    setError('');
    setSuccess('');

    try {
      await api.post('/auth/change-password', {
        current_password: currentPassword,
        new_password: newPassword
      });
      
      setSuccess('Password updated securely. Redirecting...');
      await refreshSession();
      setTimeout(() => navigate('/'), 1500);
    } catch (err: any) {
      setError(err.message || 'Failed to update password.');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="login-container flex items-center justify-center">
      <div className="glass-panel login-card" style={{ maxWidth: '480px' }}>
        <div className="login-header text-center">
          <div className="logo-icon mx-auto mb-4"></div>
          <h2>SECURITY MANDATE</h2>
          {user?.must_change_password ? (
            <p className="subtitle" style={{ color: 'var(--warning)' }}>Initial bootstrap password must be changed</p>
          ) : (
            <p className="subtitle">Update your credentials</p>
          )}
        </div>

        {error && <div className="login-error">{error}</div>}
        {success && <div className="login-error" style={{ background: 'rgba(16, 185, 129, 0.1)', borderColor: 'rgba(16, 185, 129, 0.3)', color: 'var(--success)' }}>{success}</div>}

        <form onSubmit={handleSubmit} className="login-form">
          <div className="input-group">
            <div className="input-icon"><Lock size={18} /></div>
            <input 
              type={showCurrent ? 'text' : 'password'} 
              className="input-field with-icon with-action" 
              placeholder="Current Password" 
              value={currentPassword}
              onChange={(e) => setCurrentPassword(e.target.value)}
              disabled={loading || !!success}
            />
            <button 
              type="button" 
              className="input-action"
              onClick={() => setShowCurrent(!showCurrent)}
            >
              {showCurrent ? <EyeOff size={16} /> : <Eye size={16} />}
            </button>
          </div>

          <div className="input-group mt-2">
            <div className="input-icon"><Lock size={18} /></div>
            <input 
              type={showNew ? 'text' : 'password'} 
              className="input-field with-icon with-action" 
              placeholder="New Password (min 10 chars)" 
              value={newPassword}
              onChange={(e) => setNewPassword(e.target.value)}
              disabled={loading || !!success}
            />
            <button 
              type="button" 
              className="input-action"
              onClick={() => setShowNew(!showNew)}
            >
              {showNew ? <EyeOff size={16} /> : <Eye size={16} />}
            </button>
          </div>

          <div className="input-group">
            <div className="input-icon"><Lock size={18} /></div>
            <input 
              type="password"
              className="input-field with-icon" 
              placeholder="Confirm New Password" 
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
              disabled={loading || !!success}
            />
          </div>

          <button 
            type="submit" 
            className="btn-primary w-full login-btn"
            disabled={loading || !!success}
          >
            {loading ? <span className="spinner"></span> : 'ENCRYPT & SAVE'}
          </button>
        </form>
      </div>
    </div>
  );
};
