import { useState, FormEvent, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { User, Lock, Shield, Terminal, Server, ArrowRight } from 'lucide-react';
import { useAuthStore } from '@/store/authStore';
import Button from '@/components/ui/Button';

export default function Login() {
  const navigate = useNavigate();
  const { login, error, loading, isAuthenticated, clearError } = useAuthStore();
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [totpCode, setTotpCode] = useState('');
  const [requiresTOTP, setRequiresTOTP] = useState(false);
  const [focusedField, setFocusedField] = useState<string | null>(null);

  useEffect(() => {
    if (isAuthenticated) {
      navigate('/dashboard');
    }
  }, [isAuthenticated, navigate]);

  useEffect(() => {
    return () => {
      clearError();
    };
  }, [clearError]);

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();

    const success = await login({ username, password, totp_code: totpCode });

    if (success === 'requires_totp') {
      setRequiresTOTP(true);
      return;
    }

    if (success) {
      navigate('/dashboard');
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center relative overflow-hidden">
      {/* Animated gradient background */}
      <div className="absolute inset-0 bg-gradient-to-br from-slate-900 via-blue-900 to-slate-900">
        <div className="absolute inset-0 bg-[radial-gradient(ellipse_at_top_right,_var(--tw-gradient-stops))] from-blue-600/20 via-transparent to-transparent" />
        <div className="absolute inset-0 bg-[radial-gradient(ellipse_at_bottom_left,_var(--tw-gradient-stops))] from-indigo-600/20 via-transparent to-transparent" />
      </div>

      {/* Floating decorative elements */}
      <div className="absolute top-20 left-20 opacity-10">
        <Terminal className="w-32 h-32 text-white" />
      </div>
      <div className="absolute bottom-20 right-20 opacity-10">
        <Server className="w-24 h-24 text-white" />
      </div>

      {/* Grid pattern overlay */}
      <div className="absolute inset-0 bg-[url('data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iNDAiIGhlaWdodD0iNDAiIHhtbG5zPSJodHRwOi8vd3d3LnczLm9yZy8yMDAwL3N2ZyI+PGRlZnM+PHBhdHRlcm4gaWQ9ImdyaWQiIHdpZHRoPSI0MCIgaGVpZ2h0PSI0MCIgcGF0dGVyblVuaXRzPSJ1c2VyU3BhY2VPblVzZSI+PHBhdGggZD0iTSAwIDEwIEwgNDAgMTAgTSAxMCAwIEwgMTAgNDAgTSAwIDIwIEwgNDAgMjAgTSAyMCAwIEwgMjAgNDAgTSAwIDMwIEwgNDAgMzAgTSAzMCAwIEwgMzAgNDAiIGZpbGw9Im5vbmUiIHN0cm9rZT0iIzM0MzQ1MCIgc3Ryb2tlLXdpZHRoPSIwLjUiIG9wYWNpdHk9IjAuMyIvPjwvcGF0dGVybj48L2RlZnM+PHJlY3QgZmlsbD0idXJsKCNncmlkKSIgd2lkdGg9IjEwMCUiIGhlaWdodD0iMTAwJSIvPjwvc3ZnPg==')] opacity-30" />

      {/* Login card */}
      <div className="relative z-10 w-full max-w-md mx-4">
        {/* Glassmorphism card */}
        <div className="backdrop-blur-xl bg-white/10 rounded-3xl border border-white/20 shadow-2xl overflow-hidden">
          {/* Header with gradient */}
          <div className="bg-gradient-to-r from-blue-600/80 to-indigo-600/80 px-8 py-8">
            <div className="flex items-center justify-center mb-4">
              <div className="p-3 bg-white/20 rounded-2xl backdrop-blur-sm">
                <Terminal className="w-10 h-10 text-white" />
              </div>
            </div>
            <h1 className="text-3xl font-bold text-white text-center tracking-tight">
              HTTP Remote
            </h1>
            <p className="text-blue-100 text-center mt-2 text-sm">
              Secure deployment management
            </p>
          </div>

          {/* Form section */}
          <div className="px-8 py-8">
            <form onSubmit={handleSubmit} className="space-y-5">
              {error && (
                <div className="bg-red-500/20 border border-red-500/30 text-red-200 px-4 py-3 rounded-xl text-sm backdrop-blur-sm flex items-center gap-2">
                  <div className="w-2 h-2 bg-red-400 rounded-full animate-pulse" />
                  {error}
                </div>
              )}

              {/* Show either login fields OR 2FA field, not both */}
              {!requiresTOTP ? (
                <>
                  {/* Username field */}
                  <div className="space-y-2">
                    <label className="text-sm font-medium text-gray-300 block">
                      Username
                    </label>
                    <div className={`relative group transition-all duration-300 ${focusedField === 'username' ? 'scale-[1.02]' : ''}`}>
                      <div className="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
                        <User className={`w-5 h-5 transition-colors duration-300 ${focusedField === 'username' ? 'text-blue-400' : 'text-gray-500'}`} />
                      </div>
                      <input
                        type="text"
                        value={username}
                        onChange={(e) => setUsername(e.target.value)}
                        onFocus={() => setFocusedField('username')}
                        onBlur={() => setFocusedField(null)}
                        required
                        autoComplete="username"
                        placeholder="Enter your username"
                        className="w-full pl-12 pr-4 py-3.5 bg-white/5 border border-white/10 rounded-xl text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500/50 focus:border-blue-500/50 transition-all duration-300"
                      />
                    </div>
                  </div>

                  {/* Password field */}
                  <div className="space-y-2">
                    <label className="text-sm font-medium text-gray-300 block">
                      Password
                    </label>
                    <div className={`relative group transition-all duration-300 ${focusedField === 'password' ? 'scale-[1.02]' : ''}`}>
                      <div className="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
                        <Lock className={`w-5 h-5 transition-colors duration-300 ${focusedField === 'password' ? 'text-blue-400' : 'text-gray-500'}`} />
                      </div>
                      <input
                        type="password"
                        value={password}
                        onChange={(e) => setPassword(e.target.value)}
                        onFocus={() => setFocusedField('password')}
                        onBlur={() => setFocusedField(null)}
                        required
                        autoComplete="current-password"
                        placeholder="Enter your password"
                        className="w-full pl-12 pr-4 py-3.5 bg-white/5 border border-white/10 rounded-xl text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500/50 focus:border-blue-500/50 transition-all duration-300"
                      />
                    </div>
                  </div>
                </>
              ) : (
                <>
                  {/* 2FA step - clean simple view */}
                  <div className="text-center mb-2">
                    <p className="text-gray-300 text-sm">
                      Enter the 6-digit code from your authenticator app
                    </p>
                  </div>

                  {/* 2FA Code field */}
                  <div className="space-y-2">
                    <label className="text-sm font-medium text-gray-300 block">
                      Verification Code
                    </label>
                    <div className={`relative group transition-all duration-300 ${focusedField === 'totp' ? 'scale-[1.02]' : ''}`}>
                      <div className="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
                        <Shield className={`w-5 h-5 transition-colors duration-300 ${focusedField === 'totp' ? 'text-blue-400' : 'text-gray-500'}`} />
                      </div>
                      <input
                        type="text"
                        value={totpCode}
                        onChange={(e) => setTotpCode(e.target.value.replace(/\D/g, '').slice(0, 6))}
                        onFocus={() => setFocusedField('totp')}
                        onBlur={() => setFocusedField(null)}
                        required
                        maxLength={6}
                        autoComplete="off"
                        placeholder="000000"
                        autoFocus
                        className="w-full pl-12 pr-4 py-3.5 bg-white/5 border border-white/10 rounded-xl text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500/50 focus:border-blue-500/50 transition-all duration-300 tracking-[0.5em] text-center font-mono text-lg"
                      />
                    </div>
                  </div>
                </>
              )}

              {/* Submit button */}
              <Button
                type="submit"
                variant="primary"
                size="lg"
                loading={loading}
                className="w-full bg-gradient-to-r from-blue-600 to-indigo-600 hover:from-blue-500 hover:to-indigo-500 text-white font-semibold py-4 rounded-xl shadow-lg shadow-blue-500/25 hover:shadow-blue-500/40 transition-all duration-300 flex items-center justify-center gap-2 group"
              >
                <span>{requiresTOTP ? 'Verify' : 'Sign In'}</span>
                {!loading && <ArrowRight className="w-5 h-5 group-hover:translate-x-1 transition-transform" />}
              </Button>

              {requiresTOTP && (
                <button
                  type="button"
                  className="w-full text-gray-400 hover:text-white text-sm py-2 transition-colors duration-300"
                  onClick={() => {
                    setRequiresTOTP(false);
                    setTotpCode('');
                    clearError();
                  }}
                >
                  ← Back to login
                </button>
              )}
            </form>
          </div>
        </div>

        {/* Footer text */}
        <p className="text-center mt-6 text-gray-500 text-sm">
          Secure access to your deployment infrastructure
        </p>
      </div>
    </div>
  );
}
