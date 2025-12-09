import { useEffect, useRef, useState, useCallback } from 'react';
import { api } from '@/api/client';
import { API_ENDPOINTS } from '@/lib/config';

// Types for metrics data (matching backend types)
interface CPUMetrics {
  usage_percent: number;
  cores: number;
  model: string;
}

interface MemoryMetrics {
  total: number;
  used: number;
  available: number;
  used_percent: number;
  swap_total: number;
  swap_used: number;
  swap_percent: number;
}

interface DiskMetrics {
  device: string;
  mountpoint: string;
  fstype: string;
  total: number;
  used: number;
  free: number;
  used_percent: number;
}

interface NetworkMetrics {
  interface: string;
  bytes_sent: number;
  bytes_recv: number;
  packets_sent: number;
  packets_recv: number;
  err_in: number;
  err_out: number;
  is_up: boolean;
}

export interface SystemMetrics {
  cpu: CPUMetrics;
  memory: MemoryMetrics;
  disks: DiskMetrics[];
  network: NetworkMetrics[];
  uptime: number;
  load_avg: number[];
}

interface ContainerCPU {
  usage_percent: number;
}

interface ContainerMemory {
  usage: number;
  limit: number;
  used_percent: number;
  cache: number;
}

interface ContainerNetwork {
  rx_bytes: number;
  tx_bytes: number;
  rx_packets: number;
  tx_packets: number;
}

interface ContainerBlockIO {
  read_bytes: number;
  write_bytes: number;
}

interface ContainerMetrics {
  id: string;
  name: string;
  image: string;
  status: string;
  state: string;
  created: string;
  cpu: ContainerCPU;
  memory: ContainerMemory;
  network: ContainerNetwork;
  block_io: ContainerBlockIO;
}

export interface DockerMetrics {
  available: boolean;
  version?: string;
  containers: ContainerMetrics[];
  summary: {
    total: number;
    running: number;
    paused: number;
    stopped: number;
  };
}

export interface MetricsStreamData {
  system?: SystemMetrics;
  docker?: DockerMetrics;
  timestamp: string;
}

interface UseMetricsSSEOptions {
  interval?: number; // refresh interval in seconds (1-60)
  enabled?: boolean; // whether to enable SSE streaming
  onData?: (data: MetricsStreamData) => void;
  onError?: (error: Event) => void;
}

export function useMetricsSSE(options: UseMetricsSSEOptions = {}) {
  const { interval = 5, enabled = true, onData, onError } = options;

  const [connected, setConnected] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [systemMetrics, setSystemMetrics] = useState<SystemMetrics | null>(null);
  const [dockerMetrics, setDockerMetrics] = useState<DockerMetrics | null>(null);
  const [lastUpdate, setLastUpdate] = useState<Date | null>(null);

  const eventSourceRef = useRef<EventSource | null>(null);
  const optionsRef = useRef({ onData, onError });

  // Update options ref when options change
  useEffect(() => {
    optionsRef.current = { onData, onError };
  }, [onData, onError]);

  const connect = useCallback(() => {
    // Close existing connection if any
    if (eventSourceRef.current) {
      eventSourceRef.current.close();
    }

    try {
      const endpoint = `${API_ENDPOINTS.metricsStream}?interval=${interval}`;
      const eventSource = api.createEventSource(endpoint);
      eventSourceRef.current = eventSource;

      eventSource.onopen = () => {
        setConnected(true);
        setError(null);
      };

      eventSource.addEventListener('metrics', (event) => {
        try {
          const data: MetricsStreamData = JSON.parse(event.data);

          if (data.system) {
            setSystemMetrics(data.system);
          }
          if (data.docker) {
            setDockerMetrics(data.docker);
          }
          setLastUpdate(new Date(data.timestamp));

          optionsRef.current.onData?.(data);
        } catch (err) {
          console.error('Failed to parse metrics event data:', err);
        }
      });

      eventSource.onerror = (event) => {
        setError('Connection error');
        setConnected(false);
        optionsRef.current.onError?.(event);

        // Auto-reconnect after 5 seconds
        setTimeout(() => {
          if (eventSourceRef.current === eventSource) {
            connect();
          }
        }, 5000);
      };
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to connect');
      setConnected(false);
    }
  }, [interval]);

  const disconnect = useCallback(() => {
    if (eventSourceRef.current) {
      eventSourceRef.current.close();
      eventSourceRef.current = null;
      setConnected(false);
    }
  }, []);

  useEffect(() => {
    if (enabled) {
      connect();
    } else {
      disconnect();
    }

    return () => {
      if (eventSourceRef.current) {
        eventSourceRef.current.close();
        eventSourceRef.current = null;
      }
    };
  }, [enabled, connect, disconnect]);

  return {
    connected,
    error,
    systemMetrics,
    dockerMetrics,
    lastUpdate,
    reconnect: connect,
    disconnect,
  };
}
