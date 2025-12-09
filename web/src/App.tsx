import { HashRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import { useEffect } from 'react';
import { useAuthStore } from './store/authStore';

// Layout
import Layout from './components/layout/Layout';
import ToastContainer from './components/ui/Toast';

// Pages
import Login from './pages/Login';
import Dashboard from './pages/Dashboard';
import Apps from './pages/Apps';
import AppDetail from './pages/AppDetail';
import Execute from './pages/Execute';
import ExecuteAll from './pages/ExecuteAll';
import Executions from './pages/Executions';
import AuditLogs from './pages/AuditLogs';
import Settings from './pages/Settings';
import Terminal from './pages/Terminal';
import Files from './pages/Files';
import Users from './pages/Users';
import Monitoring from './pages/Monitoring';
import Containers from './pages/Containers';

// Protected Route Component
function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { isAuthenticated, loading } = useAuthStore();

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-100">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
      </div>
    );
  }

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />;
  }

  return <Layout>{children}</Layout>;
}

function App() {
  const checkAuth = useAuthStore((state) => state.checkAuth);

  useEffect(() => {
    // Check authentication status on mount
    checkAuth();
  }, [checkAuth]);

  return (
    <Router>
      <ToastContainer />
      <Routes>
        <Route path="/login" element={<Login />} />

        <Route
          path="/"
          element={
            <ProtectedRoute>
              <Dashboard />
            </ProtectedRoute>
          }
        />

        <Route
          path="/dashboard"
          element={
            <ProtectedRoute>
              <Dashboard />
            </ProtectedRoute>
          }
        />

        <Route
          path="/apps"
          element={
            <ProtectedRoute>
              <Apps />
            </ProtectedRoute>
          }
        />

        <Route
          path="/apps/:id"
          element={
            <ProtectedRoute>
              <AppDetail />
            </ProtectedRoute>
          }
        />

        <Route
          path="/execute/:id"
          element={
            <ProtectedRoute>
              <Execute />
            </ProtectedRoute>
          }
        />

        <Route
          path="/apps/:id/execute-all"
          element={
            <ProtectedRoute>
              <ExecuteAll />
            </ProtectedRoute>
          }
        />

        <Route
          path="/executions"
          element={
            <ProtectedRoute>
              <Executions />
            </ProtectedRoute>
          }
        />

        <Route
          path="/audit-logs"
          element={
            <ProtectedRoute>
              <AuditLogs />
            </ProtectedRoute>
          }
        />

        <Route
          path="/settings"
          element={
            <ProtectedRoute>
              <Settings />
            </ProtectedRoute>
          }
        />

        <Route
          path="/terminal"
          element={
            <ProtectedRoute>
              <Terminal />
            </ProtectedRoute>
          }
        />

        <Route
          path="/files"
          element={
            <ProtectedRoute>
              <Files />
            </ProtectedRoute>
          }
        />

        <Route
          path="/users"
          element={
            <ProtectedRoute>
              <Users />
            </ProtectedRoute>
          }
        />

        <Route
          path="/monitoring"
          element={
            <ProtectedRoute>
              <Monitoring />
            </ProtectedRoute>
          }
        />

        <Route
          path="/containers"
          element={
            <ProtectedRoute>
              <Containers />
            </ProtectedRoute>
          }
        />

        {/* Catch all - redirect to dashboard */}
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </Router>
  );
}

export default App;
