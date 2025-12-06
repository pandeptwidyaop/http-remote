import { useEffect, useState } from 'react';
import { Link, useLocation } from 'react-router-dom';
import { LogOut, LayoutDashboard, Package, History, FileText, Settings, Terminal, X, ArrowUpCircle, FolderOpen, Users, Menu } from 'lucide-react';
import { useAuthStore } from '@/store/authStore';
import { useVersionStore } from '@/store/versionStore';
import Button from '@/components/ui/Button';

export default function Navbar() {
  const location = useLocation();
  const { user, logout } = useAuthStore();
  const { version, updateInfo, dismissed, fetchVersion, checkForUpdates, dismissUpdate } = useVersionStore();
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);

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
      {/* Update Available Banner */}
      {showUpdateBanner && (
        <div className="bg-gradient-to-r from-blue-600 to-indigo-600 text-white px-4 py-2">
          <div className="max-w-7xl mx-auto flex items-center justify-between">
            <div className="flex items-center space-x-3">
              <ArrowUpCircle className="h-5 w-5 animate-bounce" />
              <span className="text-sm font-medium">
                New version available: <strong>{updateInfo.latest_version}</strong>
                {updateInfo.current_version && (
                  <span className="text-blue-200 ml-1">
                    (current: {updateInfo.current_version})
                  </span>
                )}
              </span>
            </div>
            <div className="flex items-center space-x-2">
              {updateInfo.release_url && (
                <a
                  href={updateInfo.release_url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-xs bg-white/20 hover:bg-white/30 px-3 py-1 rounded transition-colors"
                >
                  View Release
                </a>
              )}
              <button
                onClick={dismissUpdate}
                className="p-1 hover:bg-white/20 rounded transition-colors"
                title="Dismiss"
              >
                <X className="h-4 w-4" />
              </button>
            </div>
          </div>
        </div>
      )}

      <nav className="bg-gray-900 text-white shadow-lg">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex items-center justify-between h-16">
          {/* Mobile menu button */}
          <button
            onClick={() => setMobileMenuOpen(!mobileMenuOpen)}
            className="md:hidden p-2 rounded-md text-gray-400 hover:text-white hover:bg-gray-800 focus:outline-none"
            aria-label="Toggle menu"
          >
            {mobileMenuOpen ? (
              <X className="h-6 w-6" />
            ) : (
              <Menu className="h-6 w-6" />
            )}
          </button>

          {/* Brand */}
          <div className="flex items-center space-x-2">
            <Link to="/" className="text-xl font-bold hover:text-blue-400 transition-colors">
              HTTP Remote
            </Link>
            <span className="px-2 py-0.5 text-xs bg-gray-800 text-gray-400 rounded hidden sm:inline">
              {version?.version || 'dev'}
            </span>
          </div>

          {/* Nav Links - Desktop */}
          <div className="hidden md:flex items-center space-x-1">
            {navLinks.map((link) => {
              const Icon = link.icon;
              const active = isActive(link.path);

              return (
                <Link
                  key={link.path}
                  to={link.path}
                  className={`flex items-center space-x-2 px-3 py-2 rounded-md text-sm font-medium transition-colors ${
                    active
                      ? 'bg-gray-800 text-white'
                      : 'text-gray-300 hover:bg-gray-800 hover:text-white'
                  }`}
                >
                  <Icon className="h-4 w-4" />
                  <span>{link.label}</span>
                </Link>
              );
            })}
          </div>

          {/* User & Logout */}
          <div className="flex items-center space-x-4">
            <span className="text-sm text-gray-300 hidden sm:block">
              {user?.username}
            </span>
            <Button
              variant="ghost"
              size="sm"
              onClick={handleLogout}
              className="text-gray-300 hover:text-white hover:bg-gray-800"
            >
              <LogOut className="h-4 w-4 sm:mr-2" />
              <span className="hidden sm:inline">Logout</span>
            </Button>
          </div>
        </div>
      </div>

      {/* Mobile Nav - Collapsible */}
      <div
        className={`md:hidden border-t border-gray-800 overflow-hidden transition-all duration-300 ease-in-out ${
          mobileMenuOpen ? 'max-h-screen opacity-100' : 'max-h-0 opacity-0'
        }`}
      >
        <div className="px-2 pt-2 pb-3 space-y-1">
          {navLinks.map((link) => {
            const Icon = link.icon;
            const active = isActive(link.path);

            return (
              <Link
                key={link.path}
                to={link.path}
                className={`flex items-center space-x-3 px-3 py-3 rounded-md text-base font-medium ${
                  active
                    ? 'bg-gray-800 text-white'
                    : 'text-gray-300 hover:bg-gray-800 hover:text-white'
                }`}
              >
                <Icon className="h-5 w-5" />
                <span>{link.label}</span>
              </Link>
            );
          })}
          {/* Mobile user info */}
          <div className="border-t border-gray-800 mt-2 pt-2 px-3 py-2">
            <span className="text-sm text-gray-400">Logged in as: </span>
            <span className="text-sm text-white font-medium">{user?.username}</span>
          </div>
        </div>
      </div>
      </nav>
    </>
  );
}
