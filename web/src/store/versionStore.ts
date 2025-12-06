import { create } from 'zustand';
import { api } from '@/api/client';
import { API_ENDPOINTS } from '@/lib/config';

interface VersionInfo {
  version: string;
  build_time: string;
  git_commit: string;
}

interface UpdateInfo {
  update_available: boolean;
  current_version: string;
  latest_version: string;
  release_url: string;
  release_notes: string;
}

interface VersionState {
  version: VersionInfo | null;
  updateInfo: UpdateInfo | null;
  loading: boolean;
  error: string | null;
  dismissed: boolean;
  fetchVersion: () => Promise<void>;
  checkForUpdates: () => Promise<void>;
  dismissUpdate: () => void;
}

export const useVersionStore = create<VersionState>((set, get) => ({
  version: null,
  updateInfo: null,
  loading: false,
  error: null,
  dismissed: false,

  fetchVersion: async () => {
    try {
      set({ loading: true, error: null });
      const version = await api.get<VersionInfo>(API_ENDPOINTS.version);
      set({ version, loading: false });
    } catch (error) {
      set({
        error: error instanceof Error ? error.message : 'Failed to fetch version',
        loading: false
      });
    }
  },

  checkForUpdates: async () => {
    try {
      const updateInfo = await api.get<UpdateInfo>(API_ENDPOINTS.versionCheck);
      set({ updateInfo });
    } catch (error) {
      // Silently fail - update check is optional
      console.warn('Failed to check for updates:', error);
    }
  },

  dismissUpdate: () => {
    set({ dismissed: true });
  },
}));
