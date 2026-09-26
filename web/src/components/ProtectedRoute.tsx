import React from 'react';
import { Navigate } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';

export const ProtectedRoute: React.FC<{ children: React.ReactNode, requirePasswordChange?: boolean }> = ({ children, requirePasswordChange = false }) => {
  const { authenticated, loading, user } = useAuth();

  if (loading) {
    return (
      <div className="flex items-center justify-center h-screen w-screen bg-primary">
        <div className="spinner" style={{ width: '32px', height: '32px', borderWidth: '3px' }}></div>
      </div>
    );
  }

  if (!authenticated) {
    return <Navigate to="/login" replace />;
  }

  if (!requirePasswordChange && user?.must_change_password) {
    return <Navigate to="/change-password" replace />;
  }

  return <>{children}</>;
};
