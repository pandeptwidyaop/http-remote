import { useState, useCallback } from 'react';
import { api, ApiError } from '@/api/client';

interface UseApiState<T> {
  data: T | null;
  loading: boolean;
  error: string | null;
}

export function useApi<T = any>() {
  const [state, setState] = useState<UseApiState<T>>({
    data: null,
    loading: false,
    error: null,
  });

  const execute = useCallback(
    async <R = T>(apiCall: () => Promise<R>): Promise<R | null> => {
      setState({ data: null, loading: true, error: null });

      try {
        const result = await apiCall();
        setState({ data: result as any, loading: false, error: null });
        return result;
      } catch (err) {
        const errorMessage =
          err instanceof ApiError ? err.message : 'An unexpected error occurred';
        setState({ data: null, loading: false, error: errorMessage });
        return null;
      }
    },
    []
  );

  const reset = useCallback(() => {
    setState({ data: null, loading: false, error: null });
  }, []);

  return {
    ...state,
    execute,
    reset,
  };
}

export default useApi;
