import React, { useState, useRef, useEffect } from 'react';
import { Search, Bell, Command, User, Circle, LogOut, CheckCheck } from 'lucide-react';
import { useAuth } from '../contexts/AuthContext';
import { useRealtimeContext } from '../contexts/RealtimeContext';
import './Topbar.css';

export const Topbar: React.FC = () => {
  const { user, logout } = useAuth();
  const { status, unreadCount, notifications, markAllRead, markRead } = useRealtimeContext();
  const [menuOpen, setMenuOpen] = useState(false);
  const [notifOpen, setNotifOpen] = useState(false);
  
  const notifRef = useRef<HTMLDivElement>(null);
  const profileRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (notifRef.current && !notifRef.current.contains(event.target as Node)) {
        setNotifOpen(false);
      }
      if (profileRef.current && !profileRef.current.contains(event.target as Node)) {
        setMenuOpen(false);
      }
    };
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  return (
    <header className="topbar glass-panel flex items-center justify-between">
      <div className="topbar-left">
        <h1 className="page-title">Dashboard</h1>
      </div>
      
      <div className="topbar-center">
        <div className="command-palette-trigger" onClick={() => window.dispatchEvent(new KeyboardEvent('keydown', { key: 'k', ctrlKey: true }))}>
          <Search size={16} className="search-icon" />
          <span className="placeholder">Search or command...</span>
          <div className="shortcut flex items-center">
            <Command size={14} /> <span>K</span>
          </div>
        </div>
      </div>
      
      <div className="topbar-right flex items-center gap-6">
        <div className="system-status flex items-center gap-2">
          <Circle size={10} className={`status-indicator ${status === 'LIVE' ? 'online' : status === 'RECONNECTING' ? 'reconnecting' : 'offline'}`} fill="currentColor" />
          <span className="status-text">{status === 'LIVE' ? 'SYSTEM ONLINE' : status === 'RECONNECTING' ? 'RECONNECTING' : 'OFFLINE'}</span>
        </div>
        
        <div className="topbar-actions flex items-center gap-4 relative">
          <div className="relative" ref={notifRef}>
            <button className="icon-btn relative" onClick={() => setNotifOpen(!notifOpen)}>
              <Bell size={20} />
              {unreadCount > 0 && <span className="notification-dot"></span>}
            </button>
            
            {notifOpen && (
              <div className="notification-dropdown glass-card">
                <div className="dropdown-header flex justify-between items-center" style={{ padding: '1rem', borderBottom: '1px solid var(--border-light)' }}>
                  <h3 style={{ fontSize: '0.9rem', margin: 0 }}>Notifications</h3>
                  {unreadCount > 0 && (
                    <button onClick={markAllRead} style={{ fontSize: '0.75rem', color: 'var(--accent-primary)', display: 'flex', alignItems: 'center', gap: '4px' }}>
                      <CheckCheck size={12} /> Mark all read
                    </button>
                  )}
                </div>
                <div className="notification-list" style={{ maxHeight: '300px', overflowY: 'auto' }}>
                  {notifications.length > 0 ? notifications.slice(0, 10).map((n) => (
                    <div key={n.id} className="notification-item" style={{ padding: '0.75rem 1rem', borderBottom: '1px solid var(--border-light)', background: n.read ? 'transparent' : 'rgba(123, 97, 255, 0.05)', display: 'flex', gap: '0.75rem', alignItems: 'flex-start' }} onClick={() => markRead(n.id)}>
                      <div style={{ width: '8px', height: '8px', borderRadius: '50%', background: n.read ? 'transparent' : 'var(--accent-primary)', marginTop: '6px' }}></div>
                      <div>
                        <p style={{ margin: 0, fontSize: '0.85rem', color: 'var(--text-primary)' }}>{n.type}</p>
                        <p style={{ margin: 0, fontSize: '0.75rem', color: 'var(--text-muted)' }}>{new Date(n.timestamp).toLocaleTimeString()}</p>
                      </div>
                    </div>
                  )) : (
                    <div style={{ padding: '2rem', textAlign: 'center', color: 'var(--text-muted)', fontSize: '0.85rem' }}>No new notifications</div>
                  )}
                </div>
              </div>
            )}
          </div>
          
          <div className="relative" ref={profileRef}>
            <div 
              className="user-profile flex items-center gap-2"
              onClick={() => setMenuOpen(!menuOpen)}
            >
              <div className="avatar">
                <User size={18} />
              </div>
              <span className="username">{user?.username || 'Admin'}</span>
            </div>

            {menuOpen && (
              <div className="profile-dropdown glass-card">
                <div className="dropdown-header">
                  <p className="dropdown-name">{user?.username}</p>
                  <p className="dropdown-role">{user?.role}</p>
                </div>
                <button className="dropdown-item" onClick={logout}>
                  <LogOut size={14} /> Logout
                </button>
              </div>
            )}
          </div>
        </div>
      </div>
    </header>
  );
};
