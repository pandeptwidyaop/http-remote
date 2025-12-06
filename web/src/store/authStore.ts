import { create } from 'zustand';
import { api } from '@/api/client';
import { API_ENDPOINTS } from '@/lib/config';
import type { User, LoginRequest, LoginResponse } from '@/types';

interface AuthState {
  user: User | null;
  loading: boolean;
  error: string | null;
  isAuthenticated: boolean;

  // Actions
  login: (credentials: LoginRequest) => Promise<boolean | 'requires_totp'>;
  logout: () => Promise<void>;
  checkAuth: () => Promise<void>;
  clearError: () => void;
}

export const useAuthStore = create<AuthState>((set) => ({
  user: null,
  loading: false,
  error: null,
  isAuthenticated: false,

  login: async (credentials) => {
    set({ loading: true, error: null });

    try {
      const response = await api.post<LoginResponse | { requires_totp: boolean; message: string }>(
        API_ENDPOINTS.login,
        credentials
      );

      // Check if 2FA is required
      if ('requires_totp' in response && response.requires_totp) {
        set({ loading: false });
        return 'requires_totp';
      }

      set({
        user: (response as LoginResponse).user,
        isAuthenticated: true,
        loading: false,
        error: null,
      });

      return true;
    } catch (error: any) {
      set({
        user: null,
        isAuthenticated: false,
        loading: false,
        error: error.message || 'Login failed',
      });

      return false;
    }
  },

  logout: async () => {
    try {
      await api.post(API_ENDPOINTS.logout);
    } catch (error) {
      console.error('Logout error:', error);
    } finally {
      set({
        user: null,
        isAuthenticated: false,
        error: null,
      });
    }
  },

  checkAuth: async () => {
    try {
      const user = await api.get<User>(API_ENDPOINTS.me);

      set({
        user,
        isAuthenticated: true,
        loading: false,
      });
    } catch (error) {
      set({
        user: null,
        isAuthenticated: false,
        loading: false,
      });
    }
  },

  clearError: () => set({ error: null }),
}));
