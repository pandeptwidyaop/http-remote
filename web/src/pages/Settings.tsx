import { useState, useEffect } from 'react';
import { Shield, Key, AlertCircle, Lock, Server, Download, RotateCcw, RefreshCw, Database, Trash2 } from 'lucide-react';
import { api } from '@/api/client';
import Button from '@/components/ui/Button';
import Card from '@/components/ui/Card';
import Input from '@/components/ui/Input';
import Modal from '@/components/ui/Modal';

interface TwoFAStatus {
  enabled: boolean;
  setup: boolean;
}

interface TwoFASecret {
  secret: string;
  qr_code_url: string;
}

interface SystemStatus {
  platform: string;
  arch: string;
  is_linux: boolean;
  is_systemd: boolean;
  is_service: boolean;
  service_status: string;
  can_upgrade: boolean;
  can_restart: boolean;
  current_version: string;
}

interface BackupInfo {
  path: string;
  version: string;
  timestamp: string;
  size: number;
}

interface VersionCheck {
  current_version: string;
  latest_version: string;
  update_available: boolean;
  release_url: string;
  release_notes: string;
}

interface StorageInfo {
  path: string;
  size_bytes: number;
  size_formatted: string;
  metrics_count: number;
  oldest_timestamp?: string;
  newest_timestamp?: string;
}

export default function Settings() {
  const [status, setStatus] = useState<TwoFAStatus>({ enabled: false, setup: false });
  const [loading, setLoading] = useState(true);
  const [qrCode, setQRCode] = useState<string | null>(null);
  const [secret, setSecret] = useState<string | null>(null);
  const [isSetupModalOpen, setIsSetupModalOpen] = useState(false);
  const [isDisableModalOpen, setIsDisableModalOpen] = useState(false);
  const [verificationCode, setVerificationCode] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  // Change password state
  const [isChangePasswordModalOpen, setIsChangePasswordModalOpen] = useState(false);
  const [oldPassword, setOldPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [passwordError, setPasswordError] = useState<string | null>(null);
  const [passwordSuccess, setPasswordSuccess] = useState(false);

  // System management state
  const [systemStatus, setSystemStatus] = useState<SystemStatus | null>(null);
  const [versionInfo, setVersionInfo] = useState<VersionCheck | null>(null);
  const [backups, setBackups] = useState<BackupInfo[]>([]);
  const [upgrading, setUpgrading] = useState(false);
  const [upgradeMessage, setUpgradeMessage] = useState('');
  const [restarting, setRestarting] = useState(false);
  const [systemError, setSystemError] = useState<string | null>(null);
  const [isRestartModalOpen, setIsRestartModalOpen] = useState(false);
  const [isRollbackModalOpen, setIsRollbackModalOpen] = useState(false);
  const [selectedBackup, setSelectedBackup] = useState<string>('');

  // Storage management state
  const [storageInfo, setStorageInfo] = useState<StorageInfo | null>(null);
  const [isPruneModalOpen, setIsPruneModalOpen] = useState(false);
  const [pruneDate, setPruneDate] = useState<string>('');
  const [pruning, setPruning] = useState(false);
  const [vacuuming, setVacuuming] = useState(false);
  const [storageMessage, setStorageMessage] = useState<string | null>(null);

  useEffect(() => {
    fetchStatus();
    fetchSystemStatus();
    fetchStorageInfo();
  }, []);

  const fetchStatus = async () => {
    try {
      const data = await api.get<TwoFAStatus>('/api/2fa/status');
      setStatus(data || { enabled: false, setup: false });
    } catch (error) {
      console.error('Failed to fetch 2FA status:', error);
      setStatus({ enabled: false, setup: false });
    } finally {
      setLoading(false);
    }
  };

  const fetchSystemStatus = async () => {
    try {
      const [sysStatus, verCheck, backupData] = await Promise.all([
        api.get<SystemStatus>('/api/system/status'),
        api.get<VersionCheck>('/api/version/check'),
        api.get<{ backups: BackupInfo[]; current_version: string }>('/api/system/rollback-versions'),
      ]);
      setSystemStatus(sysStatus);
      setVersionInfo(verCheck);
      setBackups(backupData?.backups || []);
    } catch (error) {
      console.error('Failed to fetch system status:', error);
    }
  };

  const fetchStorageInfo = async () => {
    try {
      const data = await api.get<StorageInfo>('/api/metrics/storage');
      setStorageInfo(data);
    } catch (error) {
      console.error('Failed to fetch storage info:', error);
    }
  };

  const handlePruneMetrics = async () => {
    if (!pruneDate) return;

    setPruning(true);
    setStorageMessage(null);

    try {
      const beforeDate = new Date(pruneDate).toISOString();
      const result = await api.post<{ success: boolean; deleted_records: number }>('/api/metrics/prune', {
        before: beforeDate,
      });
      setStorageMessage(`Pruned ${result.deleted_records} metrics records.`);
      setIsPruneModalOpen(false);
      setPruneDate('');
      await fetchStorageInfo();
    } catch (error: any) {
      setStorageMessage(`Error: ${error.message || 'Failed to prune metrics'}`);
    } finally {
      setPruning(false);
    }
  };

  const handleVacuumDatabase = async () => {
    setVacuuming(true);
    setStorageMessage(null);

    try {
      const result = await api.post<{ success: boolean; size_before: number; size_after: number; space_reclaimed: number }>('/api/metrics/vacuum');
      const reclaimed = result.space_reclaimed;
      const reclaimedStr = reclaimed > 0 ? formatBytes(reclaimed) : '0 B';
      setStorageMessage(`Database optimized. Reclaimed ${reclaimedStr} of space.`);
      await fetchStorageInfo();
    } catch (error: any) {
      setStorageMessage(`Error: ${error.message || 'Failed to vacuum database'}`);
    } finally {
      setVacuuming(false);
    }
  };

  const handleGenerateSecret = async () => {
    setError(null);
    setSubmitting(true);
    try {
      const secretData = await api.post<TwoFASecret>('/api/2fa/generate-secret');
      setSecret(secretData.secret);

      // Fetch QR code
      const qrData = await api.get<{ qr_code: string }>('/api/2fa/qrcode');
      setQRCode(qrData.qr_code);

      setIsSetupModalOpen(true);
    } catch (error: any) {
      setError(error.message || 'Failed to generate secret');
    } finally {
      setSubmitting(false);
    }
  };

  const handleEnable2FA = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setSubmitting(true);

    try {
      await api.post('/api/2fa/enable', { code: verificationCode });
      setIsSetupModalOpen(false);
      setVerificationCode('');
      setQRCode(null);
      setSecret(null);
      await fetchStatus();
    } catch (error: any) {
      setError(error.message || 'Invalid verification code');
    } finally {
      setSubmitting(false);
    }
  };

  const handleDisable2FA = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setSubmitting(true);

    try {
      await api.post('/api/2fa/disable', { code: verificationCode });
      setIsDisableModalOpen(false);
      setVerificationCode('');
      await fetchStatus();
    } catch (error: any) {
      setError(error.message || 'Invalid verification code');
    } finally {
      setSubmitting(false);
    }
  };

  const handleChangePassword = async (e: React.FormEvent) => {
    e.preventDefault();
    setPasswordError(null);
    setPasswordSuccess(false);

    // Validate passwords
    if (newPassword.length < 8) {
      setPasswordError('New password must be at least 8 characters long');
      return;
    }

    if (newPassword !== confirmPassword) {
      setPasswordError('New passwords do not match');
      return;
    }

    setSubmitting(true);

    try {
      await api.post('/api/auth/change-password', {
        old_password: oldPassword,
        new_password: newPassword,
      });

      setPasswordSuccess(true);
      setOldPassword('');
      setNewPassword('');
      setConfirmPassword('');

      // Close modal after 2 seconds
      setTimeout(() => {
        setIsChangePasswordModalOpen(false);
        setPasswordSuccess(false);
      }, 2000);
    } catch (error: any) {
      setPasswordError(error.message || 'Failed to change password');
    } finally {
      setSubmitting(false);
    }
  };

  const handleUpgrade = async () => {
    setUpgrading(true);
    setSystemError(null);
    setUpgradeMessage('Starting upgrade...');

    try {
      const result = await api.post<{ success: boolean; new_version: string; message: string; need_restart: boolean }>('/api/system/upgrade');
      setUpgradeMessage(result.message);
      if (result.need_restart) {
        setUpgradeMessage(`${result.message} Click "Restart Service" to apply changes.`);
      }
      await fetchSystemStatus();
    } catch (error: any) {
      setSystemError(error.message || 'Upgrade failed');
      setUpgradeMessage('');
    } finally {
      setUpgrading(false);
    }
  };

  const handleRestart = async () => {
    setRestarting(true);
    setSystemError(null);
    setIsRestartModalOpen(false);

    try {
      await api.post('/api/system/restart');
      // Show reconnecting message and reload after delay
      setUpgradeMessage('Service is restarting... Reconnecting in 5 seconds.');
      setTimeout(() => {
        window.location.reload();
      }, 5000);
    } catch (error: any) {
      setSystemError(error.message || 'Restart failed');
      setRestarting(false);
    }
  };

  const handleRollback = async () => {
    if (!selectedBackup) return;

    setUpgrading(true);
    setSystemError(null);
    setIsRollbackModalOpen(false);

    try {
      const result = await api.post<{ success: boolean; version: string; message: string }>('/api/system/rollback', {
        backup_path: selectedBackup,
      });
      setUpgradeMessage(result.message);
      await fetchSystemStatus();
    } catch (error: any) {
      setSystemError(error.message || 'Rollback failed');
    } finally {
      setUpgrading(false);
    }
  };

  const formatBytes = (bytes: number) => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
  };

  const formatDate = (dateStr: string) => {
    return new Date(dateStr).toLocaleString();
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-3xl font-bold text-gray-900">Settings</h1>
        <p className="text-gray-600 mt-1">Manage your account and system settings</p>
      </div>

      {/* System Management */}
      <Card className="p-6">
        <div className="flex items-start space-x-4">
          <div className="flex-shrink-0">
            <div className="w-12 h-12 bg-purple-100 rounded-lg flex items-center justify-center">
              <Server className="h-6 w-6 text-purple-600" />
            </div>
          </div>
          <div className="flex-1">
            <h3 className="text-lg font-semibold text-gray-900">System Management</h3>
            <p className="text-sm text-gray-600 mt-1">
              Manage system upgrades, service control, and version rollback.
            </p>

            {/* Version Info */}
            <div className="mt-4 p-4 bg-gray-50 rounded-lg">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm text-gray-600">Current Version</p>
                  <p className="text-lg font-semibold text-gray-900">{systemStatus?.current_version || 'Unknown'}</p>
                </div>
                {versionInfo?.update_available && (
                  <div className="text-right">
                    <p className="text-sm text-gray-600">Latest Version</p>
                    <p className="text-lg font-semibold text-green-600">{versionInfo.latest_version}</p>
                  </div>
                )}
              </div>

              {/* Platform Info */}
              <div className="mt-3 flex items-center space-x-4 text-sm text-gray-500">
                <span>Platform: {systemStatus?.platform}/{systemStatus?.arch}</span>
                <span>•</span>
                <span>Service: {systemStatus?.service_status || 'Unknown'}</span>
              </div>
            </div>

            {/* Warning for non-Linux */}
            {systemStatus && !systemStatus.is_linux && (
              <div className="mt-4 bg-yellow-50 border border-yellow-200 rounded-md p-4">
                <div className="flex items-start space-x-3">
                  <AlertCircle className="h-5 w-5 text-yellow-600 flex-shrink-0 mt-0.5" />
                  <div className="text-sm text-yellow-800">
                    <p className="font-medium">Limited functionality on {systemStatus.platform}</p>
                    <p className="mt-1">
                      Upgrade and restart features are only available on Linux with systemd.
                      Please manage the service manually on this platform.
                    </p>
                  </div>
                </div>
              </div>
            )}

            {/* Success/Error Messages */}
            {upgradeMessage && (
              <div className="mt-4 bg-green-50 border border-green-200 rounded-md p-4">
                <p className="text-sm text-green-800">{upgradeMessage}</p>
              </div>
            )}

            {systemError && (
              <div className="mt-4 bg-red-50 border border-red-200 rounded-md p-4">
                <p className="text-sm text-red-800">{systemError}</p>
              </div>
            )}

            {/* Action Buttons */}
            <div className="mt-4 flex flex-wrap gap-3">
              {/* Upgrade Button */}
              {versionInfo?.update_available && systemStatus?.can_upgrade && (
                <Button
                  variant="primary"
                  size="sm"
                  onClick={handleUpgrade}
                  loading={upgrading}
                  disabled={restarting}
                >
                  <Download className="h-4 w-4 mr-2" />
                  Upgrade to {versionInfo.latest_version}
                </Button>
              )}

              {/* Restart Button */}
              {systemStatus?.can_restart && (
                <Button
                  variant="secondary"
                  size="sm"
                  onClick={() => setIsRestartModalOpen(true)}
                  disabled={upgrading || restarting}
                  loading={restarting}
                >
                  <RefreshCw className="h-4 w-4 mr-2" />
                  Restart Service
                </Button>
              )}

              {/* Rollback Button */}
              {backups.length > 0 && systemStatus?.is_linux && (
                <Button
                  variant="secondary"
                  size="sm"
                  onClick={() => {
                    setSelectedBackup(backups[0]?.path || '');
                    setIsRollbackModalOpen(true);
                  }}
                  disabled={upgrading || restarting}
                >
                  <RotateCcw className="h-4 w-4 mr-2" />
                  Rollback
                </Button>
              )}
            </div>

            {/* No update available */}
            {versionInfo && !versionInfo.update_available && systemStatus?.is_linux && (
              <p className="mt-4 text-sm text-gray-500">
                You are running the latest version.
              </p>
            )}
          </div>
        </div>
      </Card>

      {/* Storage Management */}
      <Card className="p-6">
        <div className="flex items-start space-x-4">
          <div className="flex-shrink-0">
            <div className="w-12 h-12 bg-teal-100 rounded-lg flex items-center justify-center">
              <Database className="h-6 w-6 text-teal-600" />
            </div>
          </div>
          <div className="flex-1">
            <h3 className="text-lg font-semibold text-gray-900">Storage Management</h3>
            <p className="text-sm text-gray-600 mt-1">
              Manage metrics database storage and cleanup old data.
            </p>

            {/* Storage Info */}
            {storageInfo && (
              <div className="mt-4 p-4 bg-gray-50 rounded-lg">
                <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                  <div>
                    <p className="text-sm text-gray-600">Database Size</p>
                    <p className="text-lg font-semibold text-gray-900">{storageInfo.size_formatted}</p>
                  </div>
                  <div>
                    <p className="text-sm text-gray-600">Total Records</p>
                    <p className="text-lg font-semibold text-gray-900">{storageInfo.metrics_count.toLocaleString()}</p>
                  </div>
                  {storageInfo.oldest_timestamp && (
                    <div>
                      <p className="text-sm text-gray-600">Oldest Record</p>
                      <p className="text-sm font-medium text-gray-900">
                        {formatDate(storageInfo.oldest_timestamp)}
                      </p>
                    </div>
                  )}
                  {storageInfo.newest_timestamp && (
                    <div>
                      <p className="text-sm text-gray-600">Newest Record</p>
                      <p className="text-sm font-medium text-gray-900">
                        {formatDate(storageInfo.newest_timestamp)}
                      </p>
                    </div>
                  )}
                </div>
              </div>
            )}

            {/* Storage Messages */}
            {storageMessage && (
              <div className={`mt-4 p-4 rounded-md ${storageMessage.startsWith('Error') ? 'bg-red-50 border border-red-200 text-red-800' : 'bg-green-50 border border-green-200 text-green-800'}`}>
                <p className="text-sm">{storageMessage}</p>
              </div>
            )}

            {/* Action Buttons */}
            <div className="mt-4 flex flex-wrap gap-3">
              <Button
                variant="secondary"
                size="sm"
                onClick={() => {
                  const thirtyDaysAgo = new Date();
                  thirtyDaysAgo.setDate(thirtyDaysAgo.getDate() - 30);
                  setPruneDate(thirtyDaysAgo.toISOString().split('T')[0]);
                  setIsPruneModalOpen(true);
                }}
                disabled={pruning || vacuuming}
              >
                <Trash2 className="h-4 w-4 mr-2" />
                Prune Old Data
              </Button>
              <Button
                variant="secondary"
                size="sm"
                onClick={handleVacuumDatabase}
                loading={vacuuming}
                disabled={pruning}
              >
                <Database className="h-4 w-4 mr-2" />
                Optimize Database
              </Button>
            </div>
          </div>
        </div>
      </Card>

      {/* Change Password */}
      <Card className="p-6">
        <div className="flex items-start space-x-4">
          <div className="flex-shrink-0">
            <div className="w-12 h-12 bg-green-100 rounded-lg flex items-center justify-center">
              <Lock className="h-6 w-6 text-green-600" />
            </div>
          </div>
          <div className="flex-1">
            <h3 className="text-lg font-semibold text-gray-900">Change Password</h3>
            <p className="text-sm text-gray-600 mt-1">
              Update your password to keep your account secure. Use a strong password with at least 8 characters.
            </p>

            <div className="mt-4">
              <Button
                variant="primary"
                size="sm"
                onClick={() => {
                  setPasswordError(null);
                  setPasswordSuccess(false);
                  setOldPassword('');
                  setNewPassword('');
                  setConfirmPassword('');
                  setIsChangePasswordModalOpen(true);
                }}
              >
                <Key className="h-4 w-4 mr-2" />
                Change Password
              </Button>
            </div>
          </div>
        </div>
      </Card>

      {/* Two-Factor Authentication */}
      <Card className="p-6">
        <div className="flex items-start space-x-4">
          <div className="flex-shrink-0">
            <div className="w-12 h-12 bg-blue-100 rounded-lg flex items-center justify-center">
              <Shield className="h-6 w-6 text-blue-600" />
            </div>
          </div>
          <div className="flex-1">
            <h3 className="text-lg font-semibold text-gray-900">Two-Factor Authentication (2FA)</h3>
            <p className="text-sm text-gray-600 mt-1">
              Add an extra layer of security to your account by requiring a code from your authenticator app.
            </p>

            <div className="mt-4">
              {status.enabled ? (
                <div className="flex items-center space-x-4">
                  <div className="flex items-center space-x-2">
                    <div className="w-2 h-2 bg-green-500 rounded-full"></div>
                    <span className="text-sm font-medium text-green-700">2FA is enabled</span>
                  </div>
                  <Button
                    variant="danger"
                    size="sm"
                    onClick={() => {
                      setError(null);
                      setVerificationCode('');
                      setIsDisableModalOpen(true);
                    }}
                  >
                    Disable 2FA
                  </Button>
                </div>
              ) : (
                <div className="flex items-center space-x-4">
                  <div className="flex items-center space-x-2">
                    <div className="w-2 h-2 bg-gray-400 rounded-full"></div>
                    <span className="text-sm font-medium text-gray-700">2FA is disabled</span>
                  </div>
                  <Button
                    variant="primary"
                    size="sm"
                    onClick={handleGenerateSecret}
                    loading={submitting}
                  >
                    <Key className="h-4 w-4 mr-2" />
                    Enable 2FA
                  </Button>
                </div>
              )}
            </div>

            {!status.enabled && (
              <div className="mt-4 bg-blue-50 border border-blue-200 rounded-md p-4">
                <div className="flex items-start space-x-3">
                  <AlertCircle className="h-5 w-5 text-blue-600 flex-shrink-0 mt-0.5" />
                  <div className="text-sm text-blue-800">
                    <p className="font-medium">Recommended for better security</p>
                    <p className="mt-1">
                      Use an authenticator app like Google Authenticator, Authy, or 1Password to generate verification codes.
                    </p>
                  </div>
                </div>
              </div>
            )}
          </div>
        </div>
      </Card>

      {/* Restart Confirmation Modal */}
      <Modal
        isOpen={isRestartModalOpen}
        onClose={() => setIsRestartModalOpen(false)}
        title="Restart Service"
      >
        <div className="space-y-4">
          <div className="bg-yellow-50 border border-yellow-200 rounded-md p-4">
            <div className="flex items-start space-x-3">
              <AlertCircle className="h-5 w-5 text-yellow-600 flex-shrink-0 mt-0.5" />
              <div className="text-sm text-yellow-800">
                <p className="font-medium">Warning</p>
                <p className="mt-1">
                  This will restart the HTTP Remote service. The connection will be temporarily lost and the page will reload automatically.
                </p>
              </div>
            </div>
          </div>

          <div className="flex items-center justify-end space-x-3 pt-4">
            <Button
              type="button"
              variant="secondary"
              onClick={() => setIsRestartModalOpen(false)}
            >
              Cancel
            </Button>
            <Button
              variant="danger"
              onClick={handleRestart}
            >
              <RefreshCw className="h-4 w-4 mr-2" />
              Restart Now
            </Button>
          </div>
        </div>
      </Modal>

      {/* Rollback Modal */}
      <Modal
        isOpen={isRollbackModalOpen}
        onClose={() => setIsRollbackModalOpen(false)}
        title="Rollback to Previous Version"
      >
        <div className="space-y-4">
          <p className="text-sm text-gray-600">
            Select a backup version to rollback to. This will replace the current binary with the selected backup.
          </p>

          <div className="space-y-2">
            {backups.map((backup) => (
              <label
                key={backup.path}
                className={`flex items-center p-3 border rounded-lg cursor-pointer transition-colors ${
                  selectedBackup === backup.path
                    ? 'border-blue-500 bg-blue-50'
                    : 'border-gray-200 hover:border-gray-300'
                }`}
              >
                <input
                  type="radio"
                  name="backup"
                  value={backup.path}
                  checked={selectedBackup === backup.path}
                  onChange={(e) => setSelectedBackup(e.target.value)}
                  className="sr-only"
                />
                <div className="flex-1">
                  <p className="font-medium text-gray-900">{backup.version}</p>
                  <p className="text-sm text-gray-500">
                    {formatDate(backup.timestamp)} • {formatBytes(backup.size)}
                  </p>
                </div>
                {selectedBackup === backup.path && (
                  <div className="w-5 h-5 bg-blue-500 rounded-full flex items-center justify-center">
                    <svg className="w-3 h-3 text-white" fill="currentColor" viewBox="0 0 20 20">
                      <path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd" />
                    </svg>
                  </div>
                )}
              </label>
            ))}
          </div>

          {backups.length === 0 && (
            <p className="text-sm text-gray-500 text-center py-4">
              No backup versions available.
            </p>
          )}

          <div className="bg-yellow-50 border border-yellow-200 rounded-md p-4">
            <div className="flex items-start space-x-3">
              <AlertCircle className="h-5 w-5 text-yellow-600 flex-shrink-0 mt-0.5" />
              <div className="text-sm text-yellow-800">
                <p className="font-medium">Note</p>
                <p className="mt-1">
                  After rollback, you will need to restart the service for changes to take effect.
                </p>
              </div>
            </div>
          </div>

          <div className="flex items-center justify-end space-x-3 pt-4">
            <Button
              type="button"
              variant="secondary"
              onClick={() => setIsRollbackModalOpen(false)}
            >
              Cancel
            </Button>
            <Button
              variant="danger"
              onClick={handleRollback}
              disabled={!selectedBackup}
            >
              <RotateCcw className="h-4 w-4 mr-2" />
              Rollback
            </Button>
          </div>
        </div>
      </Modal>

      {/* Setup 2FA Modal */}
      <Modal
        isOpen={isSetupModalOpen}
        onClose={() => {
          setIsSetupModalOpen(false);
          setQRCode(null);
          setSecret(null);
          setVerificationCode('');
          setError(null);
        }}
        title="Enable Two-Factor Authentication"
        size="lg"
      >
        <form onSubmit={handleEnable2FA} className="space-y-6">
          {error && (
            <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-md text-sm">
              {error}
            </div>
          )}

          <div className="space-y-4">
            <div>
              <h4 className="font-medium text-gray-900 mb-2">1. Scan QR Code</h4>
              <p className="text-sm text-gray-600 mb-4">
                Open your authenticator app and scan this QR code:
              </p>
              {qrCode && (
                <div className="flex justify-center bg-white p-4 border border-gray-200 rounded-lg">
                  <img src={qrCode} alt="2FA QR Code" className="w-48 h-48" />
                </div>
              )}
            </div>

            <div>
              <h4 className="font-medium text-gray-900 mb-2">2. Or Enter Secret Manually</h4>
              <p className="text-sm text-gray-600 mb-2">
                If you can't scan the QR code, enter this secret in your app:
              </p>
              <div className="bg-gray-100 px-4 py-3 rounded-md font-mono text-sm break-all">
                {secret}
              </div>
            </div>

            <div>
              <h4 className="font-medium text-gray-900 mb-2">3. Enter Verification Code</h4>
              <p className="text-sm text-gray-600 mb-3">
                Enter the 6-digit code from your authenticator app to verify:
              </p>
              <Input
                type="text"
                placeholder="000000"
                value={verificationCode}
                onChange={(e) => setVerificationCode(e.target.value.replace(/\D/g, '').slice(0, 6))}
                required
                maxLength={6}
                pattern="[0-9]{6}"
                autoComplete="off"
              />
            </div>
          </div>

          <div className="flex items-center justify-end space-x-3 pt-4 border-t border-gray-200">
            <Button
              type="button"
              variant="secondary"
              onClick={() => {
                setIsSetupModalOpen(false);
                setQRCode(null);
                setSecret(null);
                setVerificationCode('');
                setError(null);
              }}
            >
              Cancel
            </Button>
            <Button type="submit" variant="primary" loading={submitting}>
              Verify and Enable
            </Button>
          </div>
        </form>
      </Modal>

      {/* Disable 2FA Modal */}
      <Modal
        isOpen={isDisableModalOpen}
        onClose={() => {
          setIsDisableModalOpen(false);
          setVerificationCode('');
          setError(null);
        }}
        title="Disable Two-Factor Authentication"
      >
        <form onSubmit={handleDisable2FA} className="space-y-4">
          {error && (
            <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-md text-sm">
              {error}
            </div>
          )}

          <div className="bg-yellow-50 border border-yellow-200 rounded-md p-4">
            <div className="flex items-start space-x-3">
              <AlertCircle className="h-5 w-5 text-yellow-600 flex-shrink-0 mt-0.5" />
              <div className="text-sm text-yellow-800">
                <p className="font-medium">Warning</p>
                <p className="mt-1">
                  Disabling 2FA will make your account less secure. Enter your verification code to confirm.
                </p>
              </div>
            </div>
          </div>

          <Input
            label="Verification Code"
            type="text"
            placeholder="000000"
            value={verificationCode}
            onChange={(e) => setVerificationCode(e.target.value.replace(/\D/g, '').slice(0, 6))}
            required
            maxLength={6}
            pattern="[0-9]{6}"
            autoComplete="off"
          />

          <div className="flex items-center justify-end space-x-3 pt-4">
            <Button
              type="button"
              variant="secondary"
              onClick={() => {
                setIsDisableModalOpen(false);
                setVerificationCode('');
                setError(null);
              }}
            >
              Cancel
            </Button>
            <Button type="submit" variant="danger" loading={submitting}>
              Disable 2FA
            </Button>
          </div>
        </form>
      </Modal>

      {/* Change Password Modal */}
      <Modal
        isOpen={isChangePasswordModalOpen}
        onClose={() => {
          setIsChangePasswordModalOpen(false);
          setOldPassword('');
          setNewPassword('');
          setConfirmPassword('');
          setPasswordError(null);
          setPasswordSuccess(false);
        }}
        title="Change Password"
      >
        <form onSubmit={handleChangePassword} className="space-y-4">
          {passwordError && (
            <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-md text-sm">
              {passwordError}
            </div>
          )}

          {passwordSuccess && (
            <div className="bg-green-50 border border-green-200 text-green-700 px-4 py-3 rounded-md text-sm">
              Password changed successfully!
            </div>
          )}

          <Input
            label="Current Password"
            type="password"
            value={oldPassword}
            onChange={(e) => setOldPassword(e.target.value)}
            required
            autoComplete="current-password"
            disabled={submitting || passwordSuccess}
          />

          <div>
            <Input
              label="New Password"
              type="password"
              value={newPassword}
              onChange={(e) => setNewPassword(e.target.value)}
              required
              minLength={8}
              autoComplete="new-password"
              disabled={submitting || passwordSuccess}
            />
            <p className="mt-1 text-sm text-gray-500">Must be at least 8 characters long</p>
          </div>

          <Input
            label="Confirm New Password"
            type="password"
            value={confirmPassword}
            onChange={(e) => setConfirmPassword(e.target.value)}
            required
            minLength={8}
            autoComplete="new-password"
            disabled={submitting || passwordSuccess}
          />

          <div className="flex items-center justify-end space-x-3 pt-4 border-t border-gray-200">
            <Button
              type="button"
              variant="secondary"
              onClick={() => {
                setIsChangePasswordModalOpen(false);
                setOldPassword('');
                setNewPassword('');
                setConfirmPassword('');
                setPasswordError(null);
                setPasswordSuccess(false);
              }}
              disabled={submitting}
            >
              Cancel
            </Button>
            <Button
              type="submit"
              variant="primary"
              loading={submitting}
              disabled={passwordSuccess}
            >
              Change Password
            </Button>
          </div>
        </form>
      </Modal>

      {/* Prune Metrics Modal */}
      <Modal
        isOpen={isPruneModalOpen}
        onClose={() => {
          setIsPruneModalOpen(false);
          setPruneDate('');
        }}
        title="Prune Old Metrics Data"
      >
        <div className="space-y-4">
          <p className="text-sm text-gray-600">
            This will permanently delete all metrics data older than the selected date.
            This action cannot be undone.
          </p>

          <Input
            label="Delete data older than"
            type="date"
            value={pruneDate}
            onChange={(e) => setPruneDate(e.target.value)}
            max={new Date().toISOString().split('T')[0]}
          />

          {pruneDate && (
            <div className="bg-yellow-50 border border-yellow-200 p-4 rounded-md">
              <div className="flex items-start">
                <AlertCircle className="h-5 w-5 text-yellow-600 mt-0.5 mr-2 flex-shrink-0" />
                <p className="text-sm text-yellow-800">
                  All metrics recorded before {new Date(pruneDate).toLocaleDateString()} will be permanently deleted.
                </p>
              </div>
            </div>
          )}

          <div className="flex items-center justify-end space-x-3 pt-4 border-t border-gray-200">
            <Button
              type="button"
              variant="secondary"
              onClick={() => {
                setIsPruneModalOpen(false);
                setPruneDate('');
              }}
              disabled={pruning}
            >
              Cancel
            </Button>
            <Button
              type="button"
              variant="danger"
              onClick={handlePruneMetrics}
              loading={pruning}
              disabled={!pruneDate}
            >
              <Trash2 className="h-4 w-4 mr-2" />
              Delete Old Data
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  );
}
