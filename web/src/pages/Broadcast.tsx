import React, { useState, useEffect } from 'react';
import { Send, Users, AlertCircle, Loader2 } from 'lucide-react';
import { api } from '../services/api';

export const Broadcast: React.FC = () => {
  const [text, setText] = useState('');
  const [loading, setLoading] = useState(false);
  const [userCount, setUserCount] = useState(0);

  useEffect(() => {
    // Fetch total users for estimation
    api.get<any>('/dashboard/overview').then(res => {
      if (res && res.users && res.users.total !== undefined) {
        setUserCount(res.users.total);
      }
    }).catch(console.error);
  }, []);

  const handleBroadcast = async () => {
    if (!text.trim()) {
      alert("Please enter a message to broadcast");
      return;
    }
    if (!confirm(`Are you sure you want to broadcast this message to approximately ${userCount} users?`)) {
      return;
    }

    setLoading(true);
    try {
      await api.post('/operations/broadcast', { text: text.trim() });
      alert("Broadcast has started in the background. You can check the logs for progress.");
      setText('');
    } catch (err: any) {
      alert("Error starting broadcast: " + (err.message || "Unknown error"));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fade-in" style={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      <div className="page-header">
        <h1>Broadcast Message</h1>
        <p className="text-muted text-sm mt-1">Send a message to all active users</p>
      </div>

      <div className="glass-panel p-6" style={{ flex: 1, maxWidth: '800px', margin: '0 auto', width: '100%' }}>
        <div className="flex items-center gap-3 mb-6 p-4 rounded-lg" style={{ background: 'rgba(239, 68, 68, 0.1)', border: '1px solid rgba(239, 68, 68, 0.2)' }}>
          <AlertCircle className="text-error" size={24} />
          <div>
            <h4 className="text-error m-0 mb-1">Warning</h4>
            <p className="text-xs text-muted m-0">Broadcasting to thousands of users may take several minutes due to Telegram API limits. The process runs in the background automatically.</p>
          </div>
        </div>

        <div className="form-group mb-6">
          <label className="text-sm font-semibold mb-2 block text-muted">Message Content</label>
          <textarea 
            className="input-field w-full resize-none"
            rows={10}
            placeholder="Type your broadcast message here... Supports Markdown formatting."
            value={text}
            onChange={(e) => setText(e.target.value)}
            disabled={loading}
          ></textarea>
        </div>

        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2 text-sm text-muted">
            <Users size={16} /> Estimated recipients: <strong>{userCount.toLocaleString()}</strong>
          </div>
          <button 
            className="btn btn-primary flex items-center gap-2 px-6 py-3"
            onClick={handleBroadcast}
            disabled={loading || !text.trim()}
          >
            {loading ? <Loader2 className="spin" size={18} /> : <Send size={18} />}
            {loading ? 'Starting...' : 'Start Broadcast'}
          </button>
        </div>
      </div>
    </div>
  );
};
