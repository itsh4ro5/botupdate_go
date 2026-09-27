import React, { useEffect, useState, useRef } from 'react';
import { Search, Send, User, MessageSquare, AlertCircle, RefreshCw, XCircle, Wifi, WifiOff, Loader2 } from 'lucide-react';
import { api } from '../services/api';
import { useRealtimeContext } from '../contexts/RealtimeContext';
import './Support.css';

interface ConversationDTO {
  id: string;
  user_id: number;
  username: string;
  first_name: string;
  last_name: string;
  blocked: boolean;
  topic_id: number;
  last_message_at?: string;
}

interface MessageDTO {
  id: number;
  text: string;
  direction: 'incoming' | 'outgoing';
  timestamp?: string;
}

export const Support: React.FC = () => {
  const { status } = useRealtimeContext();
  const [conversations, setConversations] = useState<ConversationDTO[]>([]);
  const [selectedConv, setSelectedConv] = useState<ConversationDTO | null>(null);
  const [messages, setMessages] = useState<MessageDTO[]>([]);
  
  const [unreadCounts, setUnreadCounts] = useState<Record<string, number>>({});
  const [loadingConvs, setLoadingConvs] = useState(true);
  const [loadingMessages, setLoadingMessages] = useState(false);
  const [search, setSearch] = useState('');
  
  const [replyText, setReplyText] = useState('');
  const [sending, setSending] = useState(false);
  
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  const fetchConversations = async () => {
    try {
      const res = await api.get<ConversationDTO[]>(`/support/conversations?search=${search}`);
      setConversations(res || []);
    } catch (err) {
      console.error('Failed to load conversations', err);
    } finally {
      setLoadingConvs(false);
    }
  };

  useEffect(() => {
    const handler = setTimeout(() => {
      fetchConversations();
    }, 400);
    return () => clearTimeout(handler);
  }, [search]);

  useEffect(() => {
    const handleRealtime = (e: CustomEvent) => {
      if (e.detail?.type === 'SUPPORT_MESSAGE') {
        const msg = e.detail.data.message;
        if (!msg) return;
        
        const userIdStr = String(msg.conversation_id);
        
        if (selectedConv && String(selectedConv.user_id) === userIdStr) {
          // It's the active conversation
          if (msg.sender_type === 'incoming' || msg.sender_type === 'outgoing') {
            setMessages(prev => {
              // Ensure we don't duplicate by ID just in case (websocket might deliver twice if reconnected)
              const msgId = msg.telegram_msg_id || msg.id;
              if (prev.some(p => p.id === msgId)) return prev;
              
              return [...prev, {
                id: msgId,
                text: msg.text || '[Message]',
                direction: msg.sender_type as 'incoming' | 'outgoing',
                timestamp: msg.timestamp || new Date().toISOString()
              }];
            });
            scrollToBottom();
          }
        } else {
          // It's a different conversation, increase unread count
          if (msg.sender_type === 'incoming') {
            setUnreadCounts(prev => ({
              ...prev,
              [userIdStr]: (prev[userIdStr] || 0) + 1
            }));
          }
        }
        
        // Refresh conversations to possibly update list ordering or details
        fetchConversations();
      }
    };
    
    window.addEventListener('REALTIME_EVENT' as any, handleRealtime);
    return () => window.removeEventListener('REALTIME_EVENT' as any, handleRealtime);
  }, [selectedConv]);

  const handleSelectConv = async (conv: ConversationDTO) => {
    setSelectedConv(conv);
    
    // Clear unread count for this user
    setUnreadCounts(prev => ({ ...prev, [String(conv.user_id)]: 0 }));
    
    setLoadingMessages(true);
    setMessages([]); // clear immediately for smooth transition
    try {
      const res = await api.get<MessageDTO[]>(`/support/conversations/${conv.user_id}/messages`);
      setMessages(res || []);
      scrollToBottom();
    } catch (err) {
      console.error('Failed to load messages', err);
    } finally {
      setLoadingMessages(false);
      // Auto-focus composer
      setTimeout(() => {
        textareaRef.current?.focus();
      }, 100);
    }
  };

  const [replyToMsg, setReplyToMsg] = useState<{id: number, text: string} | null>(null);

  const scrollToBottom = () => {
    setTimeout(() => {
      messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
    }, 100);
  };

  const handleSend = async () => {
    if (!replyText.trim() || !selectedConv) return;
    if (replyText.length > 4000) {
      alert("Message is too long. Telegram limits messages to 4096 characters.");
      return;
    }
    
    setSending(true);
    try {
      await api.post<{success: boolean, message_id: number}>(`/support/conversations/${selectedConv.user_id}/reply`, {
        text: replyText.trim(),
        reply_to_msg_id: replyToMsg ? replyToMsg.id : undefined
      });
      
      setReplyText('');
      setReplyToMsg(null);
      scrollToBottom();
      textareaRef.current?.focus();
    } catch (err: any) {
      alert(err.message || 'Failed to send reply');
    } finally {
      setSending(false);
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };
  
  const charCount = replyText.length;
  const isOverLimit = charCount > 4000;

  return (
    <div className="support-page fade-in">
      <div className="support-container glass-panel">
        
        {/* LEFT: Conversation List */}
        <div className={`support-sidebar ${selectedConv ? 'hidden-mobile' : ''}`}>
          <div className="support-sidebar-header">
            <div className="flex items-center justify-between mb-4">
              <h3 className="m-0">Conversations</h3>
              <div className={`connection-status ${status.toLowerCase()}`}>
                {status === 'LIVE' && <Wifi size={14} className="text-success" />}
                {status === 'RECONNECTING' && <RefreshCw size={14} className="text-warning spin" />}
                {status === 'OFFLINE' && <WifiOff size={14} className="text-error" />}
                <span className="text-xs uppercase font-bold tracking-wider opacity-80">
                  {status}
                </span>
              </div>
            </div>
            
            <div className="search-bar">
              <Search size={16} className="search-icon" />
              <input 
                type="text" 
                placeholder="Search users..." 
                value={search}
                onChange={(e) => setSearch(e.target.value)}
              />
            </div>
          </div>
          
          <div className="conversation-list">
            {loadingConvs ? (
              <div className="p-4 flex-center text-muted flex-col gap-2">
                <Loader2 className="spin" size={24}/>
                <span className="text-sm">Loading...</span>
              </div>
            ) : conversations.length === 0 ? (
              <div className="p-6 text-center text-muted text-sm">
                <MessageSquare size={32} className="mx-auto mb-3 opacity-30" />
                No conversations found.
              </div>
            ) : (
              conversations.map(conv => {
                const unread = unreadCounts[String(conv.user_id)] || 0;
                return (
                  <div 
                    key={conv.id} 
                    className={`conversation-item ${selectedConv?.id === conv.id ? 'active' : ''} ${unread > 0 ? 'has-unread' : ''}`}
                    onClick={() => handleSelectConv(conv)}
                  >
                    <div className="avatar">
                      <User size={20} />
                      {unread > 0 && <span className="unread-badge">{unread}</span>}
                    </div>
                    <div className="conv-info">
                      <div className="conv-name flex justify-between w-full">
                        <span>{conv.first_name} {conv.last_name}</span>
                        {conv.last_message_at && (
                          <span className="text-[10px] text-muted whitespace-nowrap ml-2 font-normal mt-[2px]">
                            {new Date(conv.last_message_at).toLocaleTimeString([], {hour: '2-digit', minute:'2-digit'})}
                          </span>
                        )}
                      </div>
                      <div className="conv-meta">
                        {conv.username ? `@${conv.username}` : `ID: ${conv.user_id}`}
                        {conv.blocked && <span className="badge badge-danger ml-auto">Blocked</span>}
                      </div>
                    </div>
                  </div>
                );
              })
            )}
          </div>
        </div>

        {/* CENTER: Message View */}
        <div className={`support-main ${!selectedConv ? 'hidden-mobile' : ''}`}>
          {selectedConv ? (
            <>
              <div className="support-main-header flex items-center justify-between">
                <div className="flex items-center gap-3">
                  <button className="back-btn mobile-only btn-icon" onClick={() => setSelectedConv(null)}>
                    <XCircle size={20} />
                  </button>
                  <div className="avatar">
                    <User size={20} />
                  </div>
                  <div>
                    <h3 className="m-0 text-white">{selectedConv.first_name} {selectedConv.last_name}</h3>
                    <div className="text-xs text-muted">ID: {selectedConv.user_id}</div>
                  </div>
                </div>
              </div>

              <div className="messages-container">
                {loadingMessages ? (
                  <div className="flex-center h-full text-muted flex-col gap-2">
                    <Loader2 className="spin text-accent" size={32} />
                  </div>
                ) : messages.length === 0 ? (
                  <div className="empty-messages fade-in">
                    <MessageSquare size={56} className="text-muted mb-4 opacity-30" />
                    <h3 className="mb-2">Support Session Started</h3>
                    <p className="text-muted text-center max-w-sm">
                      History is not persisted in memory. New live messages sent during this session will appear here instantly.
                    </p>
                  </div>
                ) : (
                  messages.map(msg => (
                    <div key={msg.id} className={`message-bubble ${msg.direction} fade-in-up`}>
                      {msg.direction === 'incoming' && (
                         <div className="msg-sender text-xs opacity-60 mb-1 flex items-center gap-2">
                           <User size={12}/> User
                         </div>
                      )}
                      {msg.direction === 'outgoing' && (
                         <div className="msg-sender text-xs opacity-60 mb-1 text-right flex items-center gap-2 justify-end">
                           Admin <AlertCircle size={12} className="opacity-0"/> 
                         </div>
                      )}
                      <div className="message-content">
                        {msg.text}
                      </div>
                      <div className="msg-meta flex justify-between items-center mt-1">
                        <button className="btn-icon small text-muted opacity-50 hover:opacity-100" onClick={() => setReplyToMsg({id: msg.id, text: msg.text})} title="Reply">
                          <MessageSquare size={10} />
                        </button>
                        {msg.timestamp && (
                          <div className="msg-timestamp text-[10px] opacity-40 text-right ml-auto">
                             {new Date(msg.timestamp).toLocaleTimeString([], {hour: '2-digit', minute:'2-digit'})}
                          </div>
                        )}
                      </div>
                    </div>
                  ))
                )}
                <div ref={messagesEndRef} />
              </div>

              <div className="reply-container">
                {replyToMsg && (
                  <div className="reply-preview text-xs bg-[var(--background-secondary)] p-2 rounded mb-2 flex justify-between items-center opacity-80 border-l-2 border-accent">
                    <span className="truncate mr-4 flex-1">
                      <span className="font-bold mr-2 text-accent">Replying to:</span> 
                      {replyToMsg.text}
                    </span>
                    <button className="btn-icon small hover:text-error" onClick={() => setReplyToMsg(null)}><XCircle size={14}/></button>
                  </div>
                )}
                <div className="reply-composer-wrapper">
                  <textarea 
                    ref={textareaRef}
                    placeholder="Type a reply... (Enter to send, Shift+Enter for newline)"
                    value={replyText}
                    onChange={e => setReplyText(e.target.value)}
                    onKeyDown={handleKeyDown}
                    disabled={sending}
                  />
                  <div className={`char-counter ${isOverLimit ? 'text-error font-bold' : 'text-muted'} text-xs`}>
                    {charCount}/4000
                  </div>
                </div>
                <button 
                  className={`btn btn-primary btn-icon send-btn ${sending ? 'loading' : ''}`} 
                  onClick={handleSend}
                  disabled={sending || !replyText.trim() || isOverLimit}
                  title="Send message (Enter)"
                >
                  {sending ? <Loader2 className="spin" size={20} /> : <Send size={20} />}
                </button>
              </div>
            </>
          ) : (
            <div className="flex-center h-full flex-col text-muted fade-in">
              <div className="empty-state-icon glass-icon-lg mb-6">
                 <MessageSquare size={48} className="text-accent opacity-80" />
              </div>
              <h2 className="mb-2 text-white">Live Support Console</h2>
              <p className="max-w-md text-center opacity-80">
                Select a conversation from the sidebar to start providing support. Live messages will appear instantly via Event Bus.
              </p>
            </div>
          )}
        </div>

        {/* RIGHT: User Info Panel (Desktop Only) */}
        {selectedConv && (
          <div className="support-info-panel hidden-mobile">
            <div className="info-header">
              <h3 className="m-0">User Details</h3>
            </div>
            <div className="info-content">
              <div className="info-group">
                <label>Telegram ID</label>
                <div className="info-value copyable"><code>{selectedConv.user_id}</code></div>
              </div>
              <div className="info-group">
                <label>Name</label>
                <div className="info-value">{selectedConv.first_name} {selectedConv.last_name}</div>
              </div>
              {selectedConv.username && (
                <div className="info-group">
                  <label>Username</label>
                  <div className="info-value">@{selectedConv.username}</div>
                </div>
              )}
              <div className="info-group">
                <label>Topic ID</label>
                <div className="info-value"><code>{selectedConv.topic_id}</code></div>
              </div>
              <div className="info-group mt-4 p-4 glass-panel rounded-lg text-center">
                {selectedConv.blocked ? (
                  <div className="flex items-center justify-center gap-2 text-error font-bold">
                    <AlertCircle size={18} />
                    <span>User Blocked</span>
                  </div>
                ) : (
                  <div className="flex flex-col items-center gap-1">
                     <span className="badge badge-success px-3 py-1">Active Account</span>
                     <span className="text-xs text-muted mt-2">Verified Telegram Status</span>
                  </div>
                )}
              </div>
            </div>
          </div>
        )}

      </div>
    </div>
  );
};
