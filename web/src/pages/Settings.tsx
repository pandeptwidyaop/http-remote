import { useState, useEffect } from 'react';
import { Shield, Key, AlertCircle, Lock } from 'lucide-react';
import { api } from '@/api/client';
import { API_ENDPOINTS } from '@/lib/config';
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

  useEffect(() => {
    fetchStatus();
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
        <p className="text-gray-600 mt-1">Manage your account security settings</p>
      </div>

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
    </div>
  );
}
