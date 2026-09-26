import React, { useState } from 'react';
import { 
  LayoutDashboard, Users, Layers, MessageSquare, 
  Radio, Clock, Bot, Server, Settings, Activity 
} from 'lucide-react';
import { NavLink } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';
import './Sidebar.css';

interface NavItemProps {
  icon: React.ReactNode;
  label: string;
  to: string;
}

const NavItem: React.FC<NavItemProps> = ({ icon, label, to }) => {
  return (
    <NavLink to={to} className={({ isActive }) => `nav-item ${isActive ? 'active' : ''}`}>
      <span className="nav-icon">{icon}</span>
      <span className="nav-label">{label}</span>
    </NavLink>
  );
};

export const Sidebar: React.FC = () => {
  const [collapsed] = useState(false);
  const { user } = useAuth();

  return (
    <aside className={`sidebar glass-panel ${collapsed ? 'collapsed' : ''}`}>
      <div className="sidebar-header flex items-center justify-between">
        <div className="logo flex items-center gap-2">
          <div className="logo-icon"></div>
          {!collapsed && <h2>COMMAND</h2>}
        </div>
      </div>
      
      <div className="sidebar-content">
        <div className="nav-section">
          <p className="nav-section-title">{!collapsed && "MAIN"}</p>
          <NavItem icon={<LayoutDashboard size={20} />} label="Dashboard" to="/" />
          <NavItem icon={<Activity size={20} />} label="Analytics" to="/analytics" />
          <NavItem icon={<Users size={20} />} label="Users" to="/users" />
          <NavItem icon={<Layers size={20} />} label="Batches" to="/batches" />
          <NavItem icon={<MessageSquare size={20} />} label="Requests" to="/requests" />
          <NavItem icon={<MessageSquare size={20} />} label="Support" to="/support" />
        </div>
        
        <div className="nav-section">
          <p className="nav-section-title">{!collapsed && "OPERATIONS"}</p>
          <NavItem icon={<Radio size={20} />} label="Broadcast" to="/broadcast" />
          <NavItem icon={<Bot size={20} />} label="Userbot" to="/userbot" />
          <NavItem icon={<Clock size={20} />} label="Scheduler" to="/scheduler" />
        </div>
        
        <div className="nav-section">
          <p className="nav-section-title">{!collapsed && "SYSTEM"}</p>
          <NavItem icon={<Activity size={20} />} label="Audit Logs" to="/audit" />
          <NavItem icon={<Server size={20} />} label="Operations" to="/operations" />
          <NavItem icon={<Settings size={20} />} label="Security" to="/security" />
          {user?.role === 'OWNER' && (
            <NavItem icon={<Users size={20} />} label="Admins" to="/admins" />
          )}
        </div>
      </div>
    </aside>
  );
};
