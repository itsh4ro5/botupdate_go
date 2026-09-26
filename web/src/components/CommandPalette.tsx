import React, { useEffect, useState, useRef } from 'react';
import { Search, Home, Activity, LogOut, Server, User, BookOpen, MessageSquare, Settings } from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';
import './CommandPalette.css';

export const CommandPalette: React.FC = () => {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState('');
  const [activeIndex, setActiveIndex] = useState(0);
  const inputRef = useRef<HTMLInputElement>(null);
  const navigate = useNavigate();
  const { logout } = useAuth();

  const commands = [
    { id: 'dashboard', label: 'Dashboard', icon: <Home size={16} />, action: () => navigate('/') },
    { id: 'analytics', label: 'Analytics', icon: <Activity size={16} />, action: () => navigate('/analytics') },
    { id: 'users', label: 'Users Directory', icon: <User size={16} />, action: () => navigate('/users') },
    { id: 'batches', label: 'Batch Management', icon: <BookOpen size={16} />, action: () => navigate('/batches') },
    { id: 'requests', label: 'Request Center', icon: <Activity size={16} />, action: () => navigate('/requests') },
    { id: 'support', label: 'Support Center', icon: <MessageSquare size={16} />, action: () => navigate('/support') },
    { id: 'audit', label: 'Audit Logs', icon: <Activity size={16} />, action: () => navigate('/audit') },
    { id: 'operations', label: 'System Operations', icon: <Server size={16} />, action: () => navigate('/operations') },
    { id: 'security', label: 'Security Settings', icon: <Settings size={16} />, action: () => navigate('/security') },
    { id: 'refresh', label: 'Refresh Dashboard', icon: <Activity size={16} />, action: () => { window.dispatchEvent(new CustomEvent('REFRESH_DASHBOARD')); } },
    { id: 'logout', label: 'Logout', icon: <LogOut size={16} />, action: logout }
  ];

  const filteredCommands = commands.filter(cmd => 
    cmd.label.toLowerCase().includes(query.toLowerCase())
  );

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault();
        setOpen(prev => !prev);
      }
      
      if (e.key === 'Escape' && open) {
        setOpen(false);
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [open]);

  useEffect(() => {
    if (open) {
      setQuery('');
      setActiveIndex(0);
      setTimeout(() => inputRef.current?.focus(), 50);
    }
  }, [open]);

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (!open) return;
      
      if (e.key === 'ArrowDown') {
        e.preventDefault();
        setActiveIndex(prev => (prev + 1) % filteredCommands.length);
      } else if (e.key === 'ArrowUp') {
        e.preventDefault();
        setActiveIndex(prev => (prev - 1 + filteredCommands.length) % filteredCommands.length);
      } else if (e.key === 'Enter') {
        e.preventDefault();
        if (filteredCommands[activeIndex]) {
          filteredCommands[activeIndex].action();
          setOpen(false);
        }
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [open, activeIndex, filteredCommands]);

  if (!open) return null;

  return (
    <div className="command-palette-overlay" onClick={() => setOpen(false)}>
      <div className="command-palette glass-panel" onClick={e => e.stopPropagation()}>
        <div className="command-input-wrapper">
          <Search size={20} className="search-icon" />
          <input
            ref={inputRef}
            type="text"
            placeholder="Type a command or search..."
            value={query}
            onChange={e => { setQuery(e.target.value); setActiveIndex(0); }}
            className="command-input"
          />
          <div className="esc-hint">ESC</div>
        </div>
        
        <div className="command-results">
          {filteredCommands.length > 0 ? (
            filteredCommands.map((cmd, idx) => (
              <div 
                key={cmd.id} 
                className={`command-item ${idx === activeIndex ? 'active' : ''}`}
                onMouseEnter={() => setActiveIndex(idx)}
                onClick={() => { cmd.action(); setOpen(false); }}
              >
                <div className="command-item-icon">{cmd.icon}</div>
                <div className="command-item-label">{cmd.label}</div>
              </div>
            ))
          ) : (
            <div className="command-empty">No results found.</div>
          )}
        </div>
      </div>
    </div>
  );
};
