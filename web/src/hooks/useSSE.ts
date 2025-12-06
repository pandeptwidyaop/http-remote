import { useEffect, useRef, useState, useCallback } from 'react';
import { api } from '@/api/client';

interface UseSSEOptions {
  onMessage?: (data: string) => void;
  onComplete?: (data: { status: string; exit_code: number }) => void;
  onError?: (error: Event) => void;
}

export function useSSE(endpoint: string | null, options: UseSSEOptions = {}) {
  const [connected, setConnected] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const eventSourceRef = useRef<EventSource | null>(null);
  const optionsRef = useRef(options);

  // Update options ref when options change
  useEffect(() => {
    optionsRef.current = options;
  }, [options]);

  const connect = useCallback(() => {
    if (!endpoint) return;

    // Close existing connection if any
    if (eventSourceRef.current) {
      eventSourceRef.current.close();
    }

    try {
      const eventSource = api.createEventSource(endpoint);
      eventSourceRef.current = eventSource;

      eventSource.onopen = () => {
        setConnected(true);
        setError(null);
      };

      eventSource.addEventListener('output', (event) => {
        optionsRef.current.onMessage?.(event.data);
      });

      eventSource.addEventListener('complete', (event) => {
        try {
          const data = JSON.parse(event.data);
          optionsRef.current.onComplete?.(data);
        } catch (err) {
          console.error('Failed to parse complete event data:', err);
        }
        eventSource.close();
        setConnected(false);
      });

      eventSource.onerror = (event) => {
        setError('Connection error');
        setConnected(false);
        optionsRef.current.onError?.(event);
        eventSource.close();
      };
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to connect');
      setConnected(false);
    }
  }, [endpoint]);

  const disconnect = useCallback(() => {
    if (eventSourceRef.current) {
      eventSourceRef.current.close();
      eventSourceRef.current = null;
      setConnected(false);
    }
  }, []);

  useEffect(() => {
    if (endpoint) {
      connect();
    }

    return () => {
      if (eventSourceRef.current) {
        eventSourceRef.current.close();
        eventSourceRef.current = null;
        setConnected(false);
      }
    };
  }, [endpoint, connect]); // Inline cleanup to avoid dependency issues

  return {
    connected,
    error,
    reconnect: connect,
    disconnect,
  };
}
