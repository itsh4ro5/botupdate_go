import React, { useState } from 'react';
import { api } from '../services/api';
import { useAuth, type User } from '../contexts/AuthContext';
import { Eye, EyeOff, Lock, User as UserIcon } from 'lucide-react';
import './Login.css';

export const Login: React.FC = () => {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  
  const { login } = useAuth();

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!username || !password) {
      setError('Please enter both username and password.');
      return;
    }

    setLoading(true);
    setError('');

    try {
      const data = await api.post<{ authenticated: boolean; user: User }>('/auth/login', {
        username,
        password
      });
      login(data.user);
    } catch (err: any) {
      setError(err.message || 'Invalid credentials.');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="login-container flex items-center justify-center">
      <div className="glass-panel login-card">
        <div className="login-header text-center">
          <div className="logo-icon mx-auto mb-4"></div>
          <h2>COMMAND CENTER</h2>
          <p className="subtitle">Authenticate to access secure systems</p>
        </div>

        {error && <div className="login-error">{error}</div>}

        <form onSubmit={handleLogin} className="login-form">
          <div className="input-group">
            <div className="input-icon"><UserIcon size={18} /></div>
            <input 
              type="text" 
              className="input-field with-icon" 
              placeholder="Username" 
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              disabled={loading}
            />
          </div>

          <div className="input-group">
            <div className="input-icon"><Lock size={18} /></div>
            <input 
              type={showPassword ? 'text' : 'password'} 
              className="input-field with-icon with-action" 
              placeholder="Password" 
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              disabled={loading}
            />
            <button 
              type="button" 
              className="input-action"
              onClick={() => setShowPassword(!showPassword)}
            >
              {showPassword ? <EyeOff size={16} /> : <Eye size={16} />}
            </button>
          </div>

          <button 
            type="submit" 
            className="btn-primary w-full login-btn"
            disabled={loading}
          >
            {loading ? <span className="spinner"></span> : 'AUTHENTICATE'}
          </button>
        </form>
      </div>
    </div>
  );
};
