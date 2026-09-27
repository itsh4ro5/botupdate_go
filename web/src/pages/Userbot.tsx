import React, { useState } from 'react';
import { Bot, Phone, Key, Activity, Info, Loader2 } from 'lucide-react';
import { api } from '../services/api';

export const Userbot: React.FC = () => {
  const [phone, setPhone] = useState('');
  const [code, setCode] = useState('');
  const [loadingCode, setLoadingCode] = useState(false);
  const [loadingLogin, setLoadingLogin] = useState(false);
  const [step, setStep] = useState(1);

  const requestLoginCode = async () => {
    if (!phone) {
      alert("Please enter a phone number");
      return;
    }
    setLoadingCode(true);
    try {
      await api.post('/operations/userbot/request-code', { phone });
      setStep(2);
      alert("Login code requested. Please check your Telegram app.");
    } catch (err: any) {
      alert("Error: " + (err.message || "Failed to request code"));
    } finally {
      setLoadingCode(false);
    }
  };

  const startUserbot = async () => {
    if (!code) {
      alert("Please enter the authentication code");
      return;
    }
    setLoadingLogin(true);
    try {
      await api.post('/operations/userbot/submit-code', { code });
      alert("Successfully logged in! Userbot is now online.");
      setStep(3);
    } catch (err: any) {
      alert("Error: " + (err.message || "Failed to submit code"));
    } finally {
      setLoadingLogin(false);
    }
  };

  return (
    <div className="fade-in" style={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      <div className="page-header">
        <h1>Userbot Settings</h1>
        <p className="text-muted text-sm mt-1">Configure automated user account behaviors</p>
      </div>

      <div className="glass-panel p-6" style={{ flex: 1, maxWidth: '800px', margin: '0 auto', width: '100%' }}>
        <div className={`flex items-center gap-3 mb-6 p-4 rounded-lg`} style={{ background: step === 3 ? 'rgba(16, 185, 129, 0.1)' : 'rgba(59, 130, 246, 0.1)', border: step === 3 ? '1px solid rgba(16, 185, 129, 0.2)' : '1px solid rgba(59, 130, 246, 0.2)' }}>
          <Info className={step === 3 ? "text-success" : "text-accent"} size={24} />
          <div>
            <h4 className={`m-0 mb-1 ${step === 3 ? 'text-success' : 'text-accent'}`}>Userbot Status</h4>
            <p className="text-xs text-muted m-0">
              {step === 3 
                ? "The userbot is currently ONLINE. Session is active." 
                : "The userbot is currently OFFLINE. You need to provide a phone number and authenticate to start the session."}
            </p>
          </div>
        </div>

        <div className="form-group mb-6">
          <label className="text-sm font-semibold mb-2 flex items-center gap-2 text-muted">
            <Phone size={16} /> Phone Number
          </label>
          <input 
            type="text" 
            className="w-full bg-dark/50 border border-white/10 rounded-lg p-3 text-white focus:outline-none focus:border-accent transition-colors"
            placeholder="+1234567890"
            value={phone}
            onChange={e => setPhone(e.target.value)}
            disabled={step > 1}
          />
        </div>

        <div className="form-group mb-8">
          <label className="text-sm font-semibold mb-2 flex items-center gap-2 text-muted">
            <Key size={16} /> Authentication Code
          </label>
          <input 
            type="text" 
            className="w-full bg-dark/50 border border-white/10 rounded-lg p-3 text-white focus:outline-none focus:border-accent transition-colors"
            placeholder="Enter code received on Telegram"
            disabled={step !== 2}
            value={code}
            onChange={e => setCode(e.target.value)}
          />
          <p className="text-xs text-muted mt-2">The authentication code field will be enabled after you request a code.</p>
        </div>

        <div className="flex items-center gap-4 border-t border-white/10 pt-6">
          <button 
            className="btn btn-primary flex items-center gap-2 px-6 py-2"
            onClick={requestLoginCode}
            disabled={step > 1 || loadingCode}
          >
            {loadingCode ? <Loader2 size={18} className="spin" /> : <Bot size={18} />} Request Login Code
          </button>
          <button 
            className="btn btn-secondary flex items-center gap-2 px-6 py-2" 
            disabled={step !== 2 || loadingLogin}
            onClick={startUserbot}
          >
            {loadingLogin ? <Loader2 size={18} className="spin" /> : <Activity size={18} />} Start Userbot
          </button>
        </div>
      </div>
    </div>
  );
};
