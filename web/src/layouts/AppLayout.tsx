import React from 'react';
import { Sidebar } from '../components/Sidebar';
import { Topbar } from '../components/Topbar';
import { CommandPalette } from '../components/CommandPalette';
import { RealtimeProvider } from '../contexts/RealtimeContext';
import './AppLayout.css';

interface AppLayoutProps {
  children: React.ReactNode;
}

export const AppLayout: React.FC<AppLayoutProps> = ({ children }) => {
  return (
    <RealtimeProvider>
      <div className="app-container">
        <Sidebar />
        <div className="main-wrapper">
          <Topbar />
          <main className="main-content">
            {children}
          </main>
        </div>
        <CommandPalette />
      </div>
    </RealtimeProvider>
  );
};
