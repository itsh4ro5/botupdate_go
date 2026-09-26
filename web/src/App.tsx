
import { Routes, Route, Navigate } from 'react-router-dom';
import { AppLayout } from './layouts/AppLayout';
import { Dashboard } from './pages/Dashboard';
import { Users } from './pages/Users';
import { Batches } from './pages/Batches';
import { Requests } from './pages/Requests';
import { Support } from './pages/Support';
import { Analytics } from './pages/Analytics';
import { Admins } from './pages/Admins';
import { Audit } from './pages/Audit';
import { Operations } from './pages/Operations';
import { SecuritySettings } from './pages/SecuritySettings';
import { Login } from './pages/Login';
import { ChangePassword } from './pages/ChangePassword';
import { ProtectedRoute } from './components/ProtectedRoute';
import { useAuth } from './contexts/AuthContext';

function App() {
  const { authenticated, user } = useAuth();

  return (
    <Routes>
      <Route 
        path="/login" 
        element={authenticated ? <Navigate to="/" replace /> : <Login />} 
      />
      
      <Route 
        path="/change-password" 
        element={
          <ProtectedRoute requirePasswordChange>
            {user?.must_change_password ? <ChangePassword /> : <Navigate to="/" replace />}
          </ProtectedRoute>
        } 
      />
      
      <Route 
        path="/" 
        element={
          <ProtectedRoute>
            <AppLayout>
              <Dashboard />
            </AppLayout>
          </ProtectedRoute>
        } 
      />

      <Route 
        path="/users" 
        element={
          <ProtectedRoute>
            <AppLayout>
              <Users />
            </AppLayout>
          </ProtectedRoute>
        } 
      />
      
      <Route 
        path="/batches" 
        element={
          <ProtectedRoute>
            <AppLayout>
              <Batches />
            </AppLayout>
          </ProtectedRoute>
        } 
      />

      <Route 
        path="/requests" 
        element={
          <ProtectedRoute>
            <AppLayout>
              <Requests />
            </AppLayout>
          </ProtectedRoute>
        } 
      />

      <Route 
        path="/support" 
        element={
          <ProtectedRoute>
            <AppLayout>
              <Support />
            </AppLayout>
          </ProtectedRoute>
        } 
      />

      <Route 
        path="/analytics" 
        element={
          <ProtectedRoute>
            <AppLayout>
              <Analytics />
            </AppLayout>
          </ProtectedRoute>
        } 
      />

      <Route 
        path="/admins" 
        element={
          <ProtectedRoute>
            <AppLayout>
              <Admins />
            </AppLayout>
          </ProtectedRoute>
        } 
      />

      <Route 
        path="/audit" 
        element={
          <ProtectedRoute>
            <AppLayout>
              <Audit />
            </AppLayout>
          </ProtectedRoute>
        } 
      />

      <Route 
        path="/operations" 
        element={
          <ProtectedRoute>
            <AppLayout>
              <Operations />
            </AppLayout>
          </ProtectedRoute>
        } 
      />

      <Route 
        path="/security" 
        element={
          <ProtectedRoute>
            <AppLayout>
              <SecuritySettings />
            </AppLayout>
          </ProtectedRoute>
        } 
      />
      
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}

export default App;
