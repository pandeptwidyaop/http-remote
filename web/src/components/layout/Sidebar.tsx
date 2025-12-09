import { useEffect, useState } from 'react';
import { Link, useLocation } from 'react-router-dom';
import {
  LogOut,
  LayoutDashboard,
  Package,
  History,
  FileText,
  Settings,
  Terminal,
  X,
  ArrowUpCircle,
  FolderOpen,
  Users,
  Menu,
  Activity,
  ChevronLeft,
  ChevronRight,
  Box,
} from 'lucide-react';
import { useAuthStore } from '@/store/authStore';
import { useVersionStore } from '@/store/versionStore';

export default function Sidebar() {
  const location = useLocation();
  const { user, logout } = useAuthStore();
  const { version, updateInfo, dismissed, fetchVersion, checkForUpdates, dismissUpdate } =
    useVersionStore();
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);
  const [collapsed, setCollapsed] = useState(false);

  useEffect(() => {
    fetchVersion();
    checkForUpdates();
  }, [fetchVersion, checkForUpdates]);

  // Close mobile menu on route change
  useEffect(() => {
    setMobileMenuOpen(false);
  }, [location.pathname]);

  const handleLogout = async () => {
    await logout();
  };

  const isAdmin = user?.role === 'admin' || user?.is_admin;

  const navLinks = [
    { path: '/dashboard', label: 'Dashboard', icon: LayoutDashboard },
    { path: '/apps', label: 'Apps', icon: Package },
    { path: '/executions', label: 'History', icon: History },
    { path: '/monitoring', label: 'Monitoring', icon: Activity },
    { path: '/containers', label: 'Containers', icon: Box },
    { path: '/terminal', label: 'Terminal', icon: Terminal },
    { path: '/files', label: 'Files', icon: FolderOpen },
    { path: '/audit-logs', label: 'Audit Logs', icon: FileText },
    ...(isAdmin ? [{ path: '/users', label: 'Users', icon: Users }] : []),
    { path: '/settings', label: 'Settings', icon: Settings },
  ];

  const isActive = (path: string) => {
    return location.pathname === path || location.pathname.startsWith(path + '/');
  };

  const showUpdateBanner = updateInfo?.update_available && !dismissed;

  return (
    <>
      {/* Mobile Header */}
      <div className="md:hidden fixed top-0 left-0 right-0 z-50 bg-gray-900 text-white h-14 flex items-center justify-between px-4 shadow-lg">
        <button
          onClick={() => setMobileMenuOpen(!mobileMenuOpen)}
          className="p-2 rounded-md text-gray-400 hover:text-white hover:bg-gray-800"
        >
          {mobileMenuOpen ? <X className="h-6 w-6" /> : <Menu className="h-6 w-6" />}
        </button>
        <span className="text-lg font-bold">HTTP Remote</span>
        <button
          onClick={handleLogout}
          className="p-2 rounded-md text-gray-400 hover:text-white hover:bg-gray-800"
        >
          <LogOut className="h-5 w-5" />
        </button>
      </div>

      {/* Mobile Menu Overlay */}
      {mobileMenuOpen && (
        <div
          className="md:hidden fixed inset-0 z-40 bg-black/50"
          onClick={() => setMobileMenuOpen(false)}
        />
      )}

      {/* Sidebar */}
      <aside
        className={`
          fixed top-0 left-0 z-40 h-screen bg-gray-900 text-white transition-all duration-300 ease-in-out flex flex-col
          ${mobileMenuOpen ? 'translate-x-0' : '-translate-x-full'}
          md:translate-x-0
          ${collapsed ? 'md:w-16' : 'md:w-64'}
        `}
      >
        {/* Logo / Brand */}
        <div
          className={`h-14 flex-shrink-0 flex items-center border-b border-gray-800 ${collapsed ? 'justify-center px-2' : 'justify-between px-4'}`}
        >
          {!collapsed && (
            <div className="flex items-center gap-2">
              <span className="text-xl font-bold">HTTP Remote</span>
              <span className="px-2 py-0.5 text-xs bg-gray-800 text-gray-400 rounded">
                {version?.version || 'dev'}
              </span>
            </div>
          )}
          {collapsed && (
            <span className="text-xl font-bold">HR</span>
          )}
          <button
            onClick={() => setCollapsed(!collapsed)}
            className="hidden md:flex p-1.5 rounded-md text-gray-400 hover:text-white hover:bg-gray-800"
          >
            {collapsed ? <ChevronRight className="h-4 w-4" /> : <ChevronLeft className="h-4 w-4" />}
          </button>
        </div>

        {/* Update Banner */}
        {showUpdateBanner && !collapsed && (
          <div className="flex-shrink-0 mx-3 mt-3 p-3 bg-gradient-to-r from-blue-600 to-indigo-600 rounded-lg">
            <div className="flex items-start gap-2">
              <ArrowUpCircle className="h-5 w-5 flex-shrink-0 animate-bounce" />
              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium">Update available</p>
                <p className="text-xs text-blue-200 truncate">{updateInfo.latest_version}</p>
              </div>
              <button onClick={dismissUpdate} className="p-1 hover:bg-white/20 rounded">
                <X className="h-3 w-3" />
              </button>
            </div>
            {updateInfo.release_url && (
              <a
                href={updateInfo.release_url}
                target="_blank"
                rel="noopener noreferrer"
                className="mt-2 block text-center text-xs bg-white/20 hover:bg-white/30 px-3 py-1.5 rounded transition-colors"
              >
                View Release
              </a>
            )}
          </div>
        )}

        {/* Navigation - scrollable area */}
        <nav className="flex-1 px-3 py-4 space-y-1 overflow-y-auto min-h-0">
          {navLinks.map((link) => {
            const Icon = link.icon;
            const active = isActive(link.path);

            return (
              <Link
                key={link.path}
                to={link.path}
                title={collapsed ? link.label : undefined}
                className={`
                  group flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-colors
                  ${active ? 'bg-blue-600 text-white' : 'text-gray-300 hover:bg-gray-800 hover:text-white'}
                  ${collapsed ? 'justify-center' : ''}
                `}
              >
                <Icon className="h-5 w-5 flex-shrink-0" />
                {!collapsed && <span>{link.label}</span>}
                {collapsed && (
                  <span className="absolute left-full ml-2 px-2 py-1 bg-gray-800 text-white text-xs rounded opacity-0 group-hover:opacity-100 transition-opacity whitespace-nowrap pointer-events-none z-50 shadow-lg">
                    {link.label}
                  </span>
                )}
              </Link>
            );
          })}
        </nav>

        {/* User Section - fixed at bottom */}
        <div className="flex-shrink-0 border-t border-gray-800 p-3 mt-auto">
          <div
            className={`flex items-center gap-3 ${collapsed ? 'justify-center' : ''}`}
          >
            <div className="w-8 h-8 rounded-full bg-gray-700 flex items-center justify-center text-sm font-medium">
              {user?.username?.charAt(0).toUpperCase() || 'U'}
            </div>
            {!collapsed && (
              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium text-white truncate">{user?.username}</p>
                <p className="text-xs text-gray-400">{isAdmin ? 'Admin' : 'User'}</p>
              </div>
            )}
            {!collapsed && (
              <button
                onClick={handleLogout}
                title="Logout"
                className="p-2 rounded-md text-gray-400 hover:text-white hover:bg-gray-800"
              >
                <LogOut className="h-4 w-4" />
              </button>
            )}
          </div>
          {collapsed && (
            <button
              onClick={handleLogout}
              title="Logout"
              className="mt-2 w-full p-2 rounded-md text-gray-400 hover:text-white hover:bg-gray-800 flex justify-center"
            >
              <LogOut className="h-4 w-4" />
            </button>
          )}
        </div>
      </aside>
    </>
  );
}
