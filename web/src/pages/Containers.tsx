import { useEffect, useState, useRef, useCallback } from 'react';
import {
  Box,
  Play,
  Square,
  RotateCcw,
  Trash2,
  Terminal,
  FileText,
  Info,
  RefreshCw,
  AlertCircle,
  CheckCircle,
  PauseCircle,
  XCircle,
  Download,
  Search,
  X,
  ZoomIn,
  ZoomOut,
  ArrowDown,
} from 'lucide-react';
import { Terminal as XTerm } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import '@xterm/xterm/css/xterm.css';
import { getApiUrl, API_ENDPOINTS } from '@/lib/config';
import ConfirmDialog from '@/components/ui/ConfirmDialog';

interface ContainerInfo {
  id: string;
  name: string;
  image: string;
  state: string;
  status: string;
  created: string;
  ports: PortMapping[];
}

interface PortMapping {
  host_ip: string;
  host_port: string;
  container_port: string;
  protocol: string;
}

interface ContainerDetail extends ContainerInfo {
  config: {
    hostname: string;
    user: string;
    env: string[];
    cmd: string[];
    entrypoint: string[];
    working_dir: string;
    labels: Record<string, string>;
  };
  network: {
    ip_address: string;
    gateway: string;
    mac_address: string;
    networks: Record<string, string>;
    dns_servers: string[];
    network_mode: string;
  };
  mounts: MountInfo[];
  health_check?: {
    status: string;
    failing_streak: number;
    log?: string;
  };
}

interface MountInfo {
  type: string;
  source: string;
  destination: string;
  mode: string;
  rw: boolean;
}

type TabType = 'info' | 'logs' | 'terminal';

export default function Containers() {
  const [containers, setContainers] = useState<ContainerInfo[]>([]);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [selectedDetail, setSelectedDetail] = useState<ContainerDetail | null>(null);
  const [activeTab, setActiveTab] = useState<TabType>('info');
  const [showAll, setShowAll] = useState(false);
  const [loading, setLoading] = useState(true);
  const [dockerAvailable, setDockerAvailable] = useState(true);
  const [actionLoading, setActionLoading] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  // Confirm dialog state
  const [confirmDialog, setConfirmDialog] = useState<{
    isOpen: boolean;
    action: 'stop' | 'restart' | 'remove' | null;
    containerId: string;
    containerName: string;
  }>({ isOpen: false, action: null, containerId: '', containerName: '' });

  // Get CSRF token from cookie
  const getCSRFToken = (): string => {
    const cookies = document.cookie.split(';');
    for (const cookie of cookies) {
      const [name, value] = cookie.trim().split('=');
      if (name === 'csrf_token') {
        return value;
      }
    }
    return '';
  };

  // Logs state
  const [logs, setLogs] = useState<string[]>([]);
  const [logsFollowing, setLogsFollowing] = useState(true);
  const [logsFilter, setLogsFilter] = useState('');
  const [logsFontSize, setLogsFontSize] = useState(12); // Font size in pixels
  const logsEndRef = useRef<HTMLDivElement>(null);
  const logsContainerRef = useRef<HTMLDivElement>(null);
  const eventSourceRef = useRef<EventSource | null>(null);

  // Terminal state
  const [terminalConnected, setTerminalConnected] = useState(false);
  const [terminalUser, setTerminalUser] = useState(''); // User to exec as (empty = default)
  const [terminalShell, setTerminalShell] = useState('/bin/sh');
  const [terminalFontSize, setTerminalFontSize] = useState(14); // Font size in pixels
  const wsRef = useRef<WebSocket | null>(null);
  const terminalRef = useRef<HTMLDivElement>(null);
  const xtermRef = useRef<XTerm | null>(null);
  const fitAddonRef = useRef<FitAddon | null>(null);

  const fetchContainers = useCallback(async () => {
    try {
      const response = await fetch(getApiUrl(`${API_ENDPOINTS.containers}?all=${showAll}`), {
        credentials: 'include',
      });
      if (response.ok) {
        const data = await response.json();
        setContainers(data || []);
      }
    } catch {
      console.error('Failed to fetch containers');
    } finally {
      setLoading(false);
    }
  }, [showAll]);

  const checkDocker = useCallback(async () => {
    try {
      const response = await fetch(getApiUrl(API_ENDPOINTS.containersStatus), {
        credentials: 'include',
      });
      if (response.ok) {
        const data = await response.json();
        setDockerAvailable(data.available);
      }
    } catch {
      setDockerAvailable(false);
    }
  }, []);

  useEffect(() => {
    checkDocker();
    fetchContainers();
    const interval = setInterval(fetchContainers, 5000);
    return () => clearInterval(interval);
  }, [checkDocker, fetchContainers]);

  useEffect(() => {
    if (selectedId) {
      fetchContainerDetail(selectedId);
    }
  }, [selectedId]);

  // Clear selection if selected container is no longer in the list
  useEffect(() => {
    if (selectedId && containers.length > 0 && !containers.find(c => c.id === selectedId)) {
      setSelectedId(null);
      setSelectedDetail(null);
    }
  }, [containers, selectedId]);

  // Auto-scroll logs
  useEffect(() => {
    if (logsFollowing && logsEndRef.current) {
      logsEndRef.current.scrollIntoView({ behavior: 'smooth' });
    }
  }, [logs, logsFollowing]);

  // Fit xterm on tab change
  useEffect(() => {
    if (activeTab === 'terminal' && fitAddonRef.current) {
      setTimeout(() => fitAddonRef.current?.fit(), 100);
    }
  }, [activeTab]);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      if (eventSourceRef.current) {
        eventSourceRef.current.close();
      }
      if (wsRef.current) {
        wsRef.current.close();
      }
    };
  }, []);

  // Start log streaming when logs tab is active
  useEffect(() => {
    if (activeTab === 'logs' && selectedId) {
      startLogStream(selectedId);
    } else {
      stopLogStream();
    }
    return () => stopLogStream();
  }, [activeTab, selectedId]);

  const fetchContainerDetail = async (id: string) => {
    try {
      const response = await fetch(getApiUrl(API_ENDPOINTS.container(id)), {
        credentials: 'include',
      });
      if (response.ok) {
        const data = await response.json();
        setSelectedDetail(data);
      }
    } catch {
      console.error('Failed to fetch container detail');
    }
  };

  const startLogStream = (containerId: string) => {
    stopLogStream();
    setLogs([]);

    const url = getApiUrl(`${API_ENDPOINTS.containerLogs(containerId)}?follow=true&tail=500`);
    const eventSource = new EventSource(url, { withCredentials: true });
    eventSourceRef.current = eventSource;

    eventSource.addEventListener('log', (event) => {
      try {
        const data = JSON.parse(event.data);
        setLogs((prev) => [...prev.slice(-1000), data.line]);
      } catch {
        setLogs((prev) => [...prev.slice(-1000), event.data]);
      }
    });

    eventSource.addEventListener('error', () => {
      setLogs((prev) => [...prev, '[Connection lost]']);
    });

    eventSource.addEventListener('end', () => {
      setLogs((prev) => [...prev, '[Stream ended]']);
    });
  };

  const stopLogStream = () => {
    if (eventSourceRef.current) {
      eventSourceRef.current.close();
      eventSourceRef.current = null;
    }
  };

  const initXterm = useCallback(() => {
    if (xtermRef.current || !terminalRef.current) return;

    const term = new XTerm({
      cursorBlink: true,
      fontSize: terminalFontSize,
      fontFamily: 'Menlo, Monaco, "Courier New", monospace',
      theme: {
        background: '#1a1a1a',
        foreground: '#4ade80',
        cursor: '#4ade80',
        cursorAccent: '#1a1a1a',
      },
      convertEol: true,
    });

    const fitAddon = new FitAddon();
    term.loadAddon(fitAddon);

    term.open(terminalRef.current);
    fitAddon.fit();

    xtermRef.current = term;
    fitAddonRef.current = fitAddon;

    // Handle resize
    const resizeObserver = new ResizeObserver(() => {
      fitAddon.fit();
    });
    resizeObserver.observe(terminalRef.current);

    return () => {
      resizeObserver.disconnect();
    };
  }, [terminalFontSize]);

  // Update terminal font size when changed
  const updateTerminalFontSize = useCallback((newSize: number) => {
    setTerminalFontSize(newSize);
    if (xtermRef.current) {
      xtermRef.current.options.fontSize = newSize;
      fitAddonRef.current?.fit();
    }
  }, []);

  const connectTerminal = (containerId: string) => {
    if (wsRef.current) {
      wsRef.current.close();
    }

    // Initialize xterm if not already done
    initXterm();

    // Clear terminal
    if (xtermRef.current) {
      xtermRef.current.clear();
      xtermRef.current.reset();
    }

    setTerminalConnected(false);

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    let wsUrl = `${protocol}//${window.location.host}${getApiUrl(API_ENDPOINTS.containerTerminal(containerId))}?shell=${encodeURIComponent(terminalShell)}`;
    if (terminalUser) {
      wsUrl += `&user=${encodeURIComponent(terminalUser)}`;
    }

    const ws = new WebSocket(wsUrl);
    wsRef.current = ws;

    ws.onopen = () => {
      setTerminalConnected(true);
      // Focus terminal after connection
      setTimeout(() => xtermRef.current?.focus(), 100);
    };

    ws.onmessage = async (event) => {
      // Handle both text and binary data
      let text: string;
      if (event.data instanceof Blob) {
        text = await event.data.text();
      } else if (event.data instanceof ArrayBuffer) {
        text = new TextDecoder().decode(event.data);
      } else {
        text = event.data;
      }
      // Write to xterm
      xtermRef.current?.write(text);
    };

    ws.onclose = () => {
      setTerminalConnected(false);
      xtermRef.current?.writeln('\r\n[Connection closed]');
    };

    ws.onerror = () => {
      xtermRef.current?.writeln('\r\n[Connection error]');
    };

    // Send input from xterm to websocket
    xtermRef.current?.onData((data) => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(data);
      }
    });
  };

  const disconnectTerminal = () => {
    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }
    setTerminalConnected(false);
  };

  // Cleanup xterm on unmount or when switching containers
  useEffect(() => {
    return () => {
      if (xtermRef.current) {
        xtermRef.current.dispose();
        xtermRef.current = null;
      }
    };
  }, [selectedId]);

  const containerAction = async (action: 'start' | 'stop' | 'restart' | 'remove', containerId: string) => {
    setActionLoading(`${action}-${containerId}`);
    setError(null);

    try {
      const endpoint =
        action === 'start'
          ? API_ENDPOINTS.containerStart(containerId)
          : action === 'stop'
          ? API_ENDPOINTS.containerStop(containerId)
          : action === 'restart'
          ? API_ENDPOINTS.containerRestart(containerId)
          : API_ENDPOINTS.container(containerId);

      const method = action === 'remove' ? 'DELETE' : 'POST';

      const response = await fetch(getApiUrl(endpoint), {
        method,
        credentials: 'include',
        headers: {
          'Content-Type': 'application/json',
          'X-CSRF-Token': getCSRFToken(),
        },
      });

      if (!response.ok) {
        const data = await response.json();
        throw new Error(data.error || `Failed to ${action} container`);
      }

      // Refresh containers list
      await fetchContainers();

      // If removed, clear selection
      if (action === 'remove' && selectedId === containerId) {
        setSelectedId(null);
        setSelectedDetail(null);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'An error occurred');
    } finally {
      setActionLoading(null);
    }
  };

  const getStateIcon = (state: string) => {
    switch (state) {
      case 'running':
        return <CheckCircle className="h-4 w-4 text-green-500" />;
      case 'paused':
        return <PauseCircle className="h-4 w-4 text-yellow-500" />;
      case 'exited':
        return <XCircle className="h-4 w-4 text-gray-400" />;
      default:
        return <AlertCircle className="h-4 w-4 text-gray-400" />;
    }
  };

  const filteredLogs = logsFilter
    ? logs.filter((line) => line.toLowerCase().includes(logsFilter.toLowerCase()))
    : logs;

  // Confirm dialog helpers
  const openConfirmDialog = (action: 'stop' | 'restart' | 'remove', containerId: string, containerName: string) => {
    setConfirmDialog({ isOpen: true, action, containerId, containerName });
  };

  const closeConfirmDialog = () => {
    setConfirmDialog({ isOpen: false, action: null, containerId: '', containerName: '' });
  };

  const handleConfirmAction = () => {
    if (confirmDialog.action && confirmDialog.containerId) {
      containerAction(confirmDialog.action, confirmDialog.containerId);
    }
    closeConfirmDialog();
  };

  const getConfirmDialogConfig = () => {
    const { action, containerName } = confirmDialog;
    switch (action) {
      case 'stop':
        return {
          title: 'Stop Container',
          message: `Are you sure you want to stop "${containerName}"?`,
          confirmText: 'Stop',
          variant: 'warning' as const,
        };
      case 'restart':
        return {
          title: 'Restart Container',
          message: `Are you sure you want to restart "${containerName}"?`,
          confirmText: 'Restart',
          variant: 'warning' as const,
        };
      case 'remove':
        return {
          title: 'Remove Container',
          message: `Are you sure you want to remove "${containerName}"? This action cannot be undone.`,
          confirmText: 'Remove',
          variant: 'danger' as const,
        };
      default:
        return {
          title: '',
          message: '',
          confirmText: 'Confirm',
          variant: 'danger' as const,
        };
    }
  };

  if (!dockerAvailable) {
    return (
      <div className="p-6">
        <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-6 text-center">
          <AlertCircle className="h-12 w-12 text-yellow-500 mx-auto mb-4" />
          <h2 className="text-xl font-semibold text-yellow-800 mb-2">Docker Not Available</h2>
          <p className="text-yellow-600">
            Docker daemon is not running or not accessible. Please ensure Docker is installed and running.
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="h-[calc(100vh-200px)] flex flex-col bg-white rounded-lg shadow-sm overflow-hidden">
      {/* Header */}
      <div className="flex-shrink-0 bg-white border-b px-6 py-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <Box className="h-6 w-6 text-blue-600" />
            <h1 className="text-xl font-semibold">Containers</h1>
            <span className="px-2 py-0.5 text-xs bg-gray-100 text-gray-600 rounded-full">
              {containers.length}
            </span>
          </div>
          <div className="flex items-center gap-4">
            <label className="flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                checked={showAll}
                onChange={(e) => setShowAll(e.target.checked)}
                className="rounded border-gray-300"
              />
              Show all
            </label>
            <button
              onClick={() => fetchContainers()}
              className="p-2 text-gray-500 hover:text-gray-700 hover:bg-gray-100 rounded-lg"
              title="Refresh"
            >
              <RefreshCw className="h-4 w-4" />
            </button>
          </div>
        </div>
      </div>

      {/* Error message */}
      {error && (
        <div className="flex-shrink-0 mx-6 mt-4 p-3 bg-red-50 border border-red-200 rounded-lg flex items-center gap-2 text-red-700">
          <AlertCircle className="h-4 w-4" />
          <span className="text-sm">{error}</span>
          <button onClick={() => setError(null)} className="ml-auto">
            <X className="h-4 w-4" />
          </button>
        </div>
      )}

      {/* Main content */}
      <div className="flex-1 flex min-h-0">
        {/* Container List */}
        <div className="w-80 border-r bg-gray-50 flex flex-col min-h-0">
          <div className="flex-1 overflow-y-auto">
            {loading ? (
              <div className="p-4 text-center text-gray-500">Loading...</div>
            ) : containers.length === 0 ? (
              <div className="p-8 text-center text-gray-500">
                <Box className="h-10 w-10 mx-auto mb-3 text-gray-300" />
                <p className="font-medium">{showAll ? 'No containers found' : 'No running containers'}</p>
                {!showAll && (
                  <button
                    onClick={() => setShowAll(true)}
                    className="mt-3 text-sm text-blue-600 hover:text-blue-700 hover:underline"
                  >
                    Show all containers
                  </button>
                )}
              </div>
            ) : (
              containers.map((container) => (
                <div
                  key={container.id}
                  onClick={() => setSelectedId(container.id)}
                  className={`p-3 border-b cursor-pointer hover:bg-white transition-colors ${
                    selectedId === container.id ? 'bg-white border-l-4 border-l-blue-500' : ''
                  }`}
                >
                  <div className="flex items-center gap-2">
                    {getStateIcon(container.state)}
                    <div className="flex-1 min-w-0">
                      <p className="font-medium truncate text-sm">{container.name}</p>
                      <p className="text-xs text-gray-500 truncate">{container.image}</p>
                    </div>
                    {/* Quick actions - only show start for stopped containers */}
                    {container.state !== 'running' && (
                      <button
                        onClick={(e) => {
                          e.stopPropagation();
                          containerAction('start', container.id);
                        }}
                        disabled={actionLoading === `start-${container.id}`}
                        className="p-1 text-gray-400 hover:text-green-500 hover:bg-green-50 rounded"
                        title="Start"
                      >
                        <Play className="h-3 w-3" />
                      </button>
                    )}
                  </div>
                  <p className="text-xs text-gray-400 mt-1 truncate">{container.status}</p>
                </div>
              ))
            )}
          </div>
        </div>

        {/* Container Detail */}
        <div className="flex-1 flex flex-col min-w-0 min-h-0">
          {selectedId && selectedDetail ? (
            <>
              {/* Tabs */}
              <div className="flex-shrink-0 bg-white border-b">
                <div className="flex">
                  <button
                    onClick={() => setActiveTab('info')}
                    className={`px-4 py-3 text-sm font-medium border-b-2 transition-colors flex items-center gap-2 ${
                      activeTab === 'info'
                        ? 'border-blue-500 text-blue-600'
                        : 'border-transparent text-gray-500 hover:text-gray-700'
                    }`}
                  >
                    <Info className="h-4 w-4" />
                    Info
                  </button>
                  <button
                    onClick={() => setActiveTab('logs')}
                    className={`px-4 py-3 text-sm font-medium border-b-2 transition-colors flex items-center gap-2 ${
                      activeTab === 'logs'
                        ? 'border-blue-500 text-blue-600'
                        : 'border-transparent text-gray-500 hover:text-gray-700'
                    }`}
                  >
                    <FileText className="h-4 w-4" />
                    Logs
                  </button>
                  <button
                    onClick={() => setActiveTab('terminal')}
                    className={`px-4 py-3 text-sm font-medium border-b-2 transition-colors flex items-center gap-2 ${
                      activeTab === 'terminal'
                        ? 'border-blue-500 text-blue-600'
                        : 'border-transparent text-gray-500 hover:text-gray-700'
                    }`}
                    disabled={selectedDetail.state !== 'running'}
                  >
                    <Terminal className="h-4 w-4" />
                    Terminal
                  </button>
                </div>
              </div>

              {/* Tab content */}
              <div className="flex-1 overflow-auto">
                {activeTab === 'info' && (
                  <div className="p-6 space-y-6">
                    {/* Header with actions */}
                    <div className="flex items-start justify-between">
                      <div>
                        <h2 className="text-xl font-bold flex items-center gap-2">
                          {getStateIcon(selectedDetail.state)}
                          {selectedDetail.name}
                        </h2>
                        <p className="text-gray-500 text-sm">{selectedDetail.image}</p>
                      </div>
                      <div className="flex gap-2">
                        {selectedDetail.state === 'running' ? (
                          <>
                            <button
                              onClick={() => openConfirmDialog('stop', selectedDetail.id, selectedDetail.name)}
                              disabled={actionLoading?.startsWith('stop')}
                              className="px-3 py-1.5 text-sm bg-red-50 text-red-600 hover:bg-red-100 rounded-lg flex items-center gap-1"
                            >
                              <Square className="h-4 w-4" />
                              Stop
                            </button>
                            <button
                              onClick={() => openConfirmDialog('restart', selectedDetail.id, selectedDetail.name)}
                              disabled={actionLoading?.startsWith('restart')}
                              className="px-3 py-1.5 text-sm bg-yellow-50 text-yellow-600 hover:bg-yellow-100 rounded-lg flex items-center gap-1"
                            >
                              <RotateCcw className="h-4 w-4" />
                              Restart
                            </button>
                          </>
                        ) : (
                          <button
                            onClick={() => containerAction('start', selectedDetail.id)}
                            disabled={actionLoading?.startsWith('start')}
                            className="px-3 py-1.5 text-sm bg-green-50 text-green-600 hover:bg-green-100 rounded-lg flex items-center gap-1"
                          >
                            <Play className="h-4 w-4" />
                            Start
                          </button>
                        )}
                        <button
                          onClick={() => openConfirmDialog('remove', selectedDetail.id, selectedDetail.name)}
                          disabled={actionLoading?.startsWith('remove')}
                          className="px-3 py-1.5 text-sm bg-gray-50 text-gray-600 hover:bg-gray-100 rounded-lg flex items-center gap-1"
                        >
                          <Trash2 className="h-4 w-4" />
                          Remove
                        </button>
                      </div>
                    </div>

                    {/* Status */}
                    <div className="bg-white border rounded-lg p-4">
                      <h3 className="font-medium mb-3">Status</h3>
                      <div className="grid grid-cols-2 gap-4 text-sm">
                        <div>
                          <span className="text-gray-500">State:</span>
                          <span className="ml-2 font-medium capitalize">{selectedDetail.state}</span>
                        </div>
                        <div>
                          <span className="text-gray-500">Container ID:</span>
                          <span className="ml-2 font-mono">{selectedDetail.id}</span>
                        </div>
                        <div>
                          <span className="text-gray-500">Created:</span>
                          <span className="ml-2">{new Date(selectedDetail.created).toLocaleString()}</span>
                        </div>
                        {selectedDetail.health_check && (
                          <div>
                            <span className="text-gray-500">Health:</span>
                            <span className={`ml-2 capitalize ${
                              selectedDetail.health_check.status === 'healthy' ? 'text-green-600' : 'text-yellow-600'
                            }`}>
                              {selectedDetail.health_check.status}
                            </span>
                          </div>
                        )}
                      </div>
                    </div>

                    {/* Ports */}
                    {selectedDetail.ports.length > 0 && (
                      <div className="bg-white border rounded-lg p-4">
                        <h3 className="font-medium mb-3">Ports</h3>
                        <div className="space-y-2 text-sm">
                          {selectedDetail.ports.map((port, i) => (
                            <div key={i} className="flex items-center gap-2">
                              <span className="font-mono bg-gray-100 px-2 py-0.5 rounded">
                                {port.container_port}/{port.protocol}
                              </span>
                              <span className="text-gray-400">→</span>
                              <span className="font-mono">
                                {port.host_ip || '0.0.0.0'}:{port.host_port}
                              </span>
                            </div>
                          ))}
                        </div>
                      </div>
                    )}

                    {/* Volumes */}
                    {selectedDetail.mounts.length > 0 && (
                      <div className="bg-white border rounded-lg p-4">
                        <h3 className="font-medium mb-3">Volumes</h3>
                        <div className="space-y-2 text-sm">
                          {selectedDetail.mounts.map((mount, i) => (
                            <div key={i} className="font-mono text-xs bg-gray-50 p-2 rounded">
                              <span className="text-gray-500">{mount.source}</span>
                              <span className="mx-2">→</span>
                              <span>{mount.destination}</span>
                              <span className="ml-2 text-gray-400">({mount.mode})</span>
                            </div>
                          ))}
                        </div>
                      </div>
                    )}

                    {/* Network */}
                    <div className="bg-white border rounded-lg p-4">
                      <h3 className="font-medium mb-3">Network</h3>
                      <div className="grid grid-cols-2 gap-4 text-sm">
                        <div>
                          <span className="text-gray-500">IP Address:</span>
                          <span className="ml-2 font-mono">{selectedDetail.network.ip_address || 'N/A'}</span>
                        </div>
                        <div>
                          <span className="text-gray-500">Gateway:</span>
                          <span className="ml-2 font-mono">{selectedDetail.network.gateway || 'N/A'}</span>
                        </div>
                        <div>
                          <span className="text-gray-500">Network Mode:</span>
                          <span className="ml-2">{selectedDetail.network.network_mode}</span>
                        </div>
                        <div>
                          <span className="text-gray-500">MAC:</span>
                          <span className="ml-2 font-mono">{selectedDetail.network.mac_address || 'N/A'}</span>
                        </div>
                      </div>
                    </div>

                    {/* Environment */}
                    {selectedDetail.config.env.length > 0 && (
                      <div className="bg-white border rounded-lg p-4">
                        <h3 className="font-medium mb-3">Environment Variables</h3>
                        <div className="max-h-48 overflow-y-auto">
                          <div className="space-y-1 text-xs font-mono">
                            {selectedDetail.config.env.map((env, i) => (
                              <div key={i} className="bg-gray-50 p-1.5 rounded truncate">
                                {env}
                              </div>
                            ))}
                          </div>
                        </div>
                      </div>
                    )}
                  </div>
                )}

                {activeTab === 'logs' && (
                  <div className="h-full flex flex-col">
                    {/* Logs toolbar */}
                    <div className="flex-shrink-0 p-2 border-b bg-gray-50 flex items-center gap-2">
                      <div className="flex-1 flex items-center gap-2">
                        <Search className="h-4 w-4 text-gray-400" />
                        <input
                          type="text"
                          placeholder="Filter logs..."
                          value={logsFilter}
                          onChange={(e) => setLogsFilter(e.target.value)}
                          className="flex-1 bg-transparent border-none outline-none text-sm"
                        />
                        {logsFilter && (
                          <button onClick={() => setLogsFilter('')}>
                            <X className="h-4 w-4 text-gray-400" />
                          </button>
                        )}
                      </div>
                      <label className="flex items-center gap-1 text-sm text-gray-600">
                        <input
                          type="checkbox"
                          checked={logsFollowing}
                          onChange={(e) => setLogsFollowing(e.target.checked)}
                          className="rounded"
                        />
                        Follow
                      </label>
                      <button
                        onClick={() => setLogs([])}
                        className="px-2 py-1 text-xs text-gray-600 hover:bg-gray-200 rounded"
                      >
                        Clear
                      </button>
                      <button
                        onClick={() => {
                          const blob = new Blob([filteredLogs.join('\n')], { type: 'text/plain' });
                          const url = URL.createObjectURL(blob);
                          const a = document.createElement('a');
                          a.href = url;
                          a.download = `${selectedDetail?.name || 'container'}-logs.txt`;
                          a.click();
                        }}
                        className="p-1 text-gray-600 hover:bg-gray-200 rounded"
                        title="Download logs"
                      >
                        <Download className="h-4 w-4" />
                      </button>
                      <div className="border-l pl-2 ml-1 flex items-center gap-1">
                        <button
                          onClick={() => setLogsFontSize((prev) => Math.max(8, prev - 2))}
                          className="p-1 text-gray-600 hover:bg-gray-200 rounded disabled:opacity-50"
                          title="Zoom out"
                          disabled={logsFontSize <= 8}
                        >
                          <ZoomOut className="h-4 w-4" />
                        </button>
                        <span className="text-xs text-gray-500 min-w-[32px] text-center">
                          {logsFontSize}px
                        </span>
                        <button
                          onClick={() => setLogsFontSize((prev) => Math.min(24, prev + 2))}
                          className="p-1 text-gray-600 hover:bg-gray-200 rounded disabled:opacity-50"
                          title="Zoom in"
                          disabled={logsFontSize >= 24}
                        >
                          <ZoomIn className="h-4 w-4" />
                        </button>
                      </div>
                      <button
                        onClick={() => {
                          logsEndRef.current?.scrollIntoView({ behavior: 'smooth' });
                        }}
                        className="p-1 text-gray-600 hover:bg-gray-200 rounded"
                        title="Scroll to bottom"
                      >
                        <ArrowDown className="h-4 w-4" />
                      </button>
                    </div>

                    {/* Logs content */}
                    <div
                      ref={logsContainerRef}
                      className="flex-1 overflow-y-auto bg-gray-900 p-4 font-mono text-gray-100"
                      style={{ fontSize: `${logsFontSize}px`, lineHeight: '1.5', maxHeight: 'calc(100vh - 280px)' }}
                    >
                      {filteredLogs.length === 0 ? (
                        <div className="text-gray-500">No logs available</div>
                      ) : (
                        filteredLogs.map((line, i) => (
                          <div key={i} className="whitespace-pre-wrap hover:bg-gray-800">
                            {line}
                          </div>
                        ))
                      )}
                      <div ref={logsEndRef} />
                    </div>
                  </div>
                )}

                {activeTab === 'terminal' && (
                  <div className="h-full flex flex-col">
                    {/* Terminal toolbar */}
                    <div className="flex-shrink-0 p-2 border-b bg-gray-50 flex items-center gap-2 flex-wrap">
                      {terminalConnected ? (
                        <>
                          <span className="flex items-center gap-1 text-sm text-green-600">
                            <CheckCircle className="h-4 w-4" />
                            Connected
                          </span>
                          {/* Zoom controls */}
                          <div className="border-l pl-2 ml-2 flex items-center gap-1">
                            <button
                              onClick={() => updateTerminalFontSize(Math.max(8, terminalFontSize - 2))}
                              className="p-1 text-gray-600 hover:bg-gray-200 rounded disabled:opacity-50"
                              title="Zoom out"
                              disabled={terminalFontSize <= 8}
                            >
                              <ZoomOut className="h-4 w-4" />
                            </button>
                            <span className="text-xs text-gray-500 min-w-[32px] text-center">
                              {terminalFontSize}px
                            </span>
                            <button
                              onClick={() => updateTerminalFontSize(Math.min(24, terminalFontSize + 2))}
                              className="p-1 text-gray-600 hover:bg-gray-200 rounded disabled:opacity-50"
                              title="Zoom in"
                              disabled={terminalFontSize >= 24}
                            >
                              <ZoomIn className="h-4 w-4" />
                            </button>
                          </div>
                          <button
                            onClick={disconnectTerminal}
                            className="ml-auto px-2 py-1 text-xs text-red-600 hover:bg-red-50 rounded"
                          >
                            Disconnect
                          </button>
                        </>
                      ) : (
                        <>
                          <div className="flex items-center gap-2">
                            <label className="text-xs text-gray-500">Shell:</label>
                            <select
                              value={terminalShell}
                              onChange={(e) => setTerminalShell(e.target.value)}
                              className="text-xs border rounded px-2 py-1 bg-white"
                            >
                              <option value="/bin/sh">/bin/sh</option>
                              <option value="/bin/bash">/bin/bash</option>
                              <option value="/bin/zsh">/bin/zsh</option>
                              <option value="/bin/ash">/bin/ash</option>
                            </select>
                          </div>
                          <div className="flex items-center gap-2">
                            <label className="text-xs text-gray-500">User:</label>
                            <input
                              type="text"
                              value={terminalUser}
                              onChange={(e) => setTerminalUser(e.target.value)}
                              placeholder="default"
                              className="text-xs border rounded px-2 py-1 w-20"
                            />
                          </div>
                          {/* Zoom controls */}
                          <div className="border-l pl-2 ml-2 flex items-center gap-1">
                            <button
                              onClick={() => updateTerminalFontSize(Math.max(8, terminalFontSize - 2))}
                              className="p-1 text-gray-600 hover:bg-gray-200 rounded disabled:opacity-50"
                              title="Zoom out"
                              disabled={terminalFontSize <= 8}
                            >
                              <ZoomOut className="h-4 w-4" />
                            </button>
                            <span className="text-xs text-gray-500 min-w-[32px] text-center">
                              {terminalFontSize}px
                            </span>
                            <button
                              onClick={() => updateTerminalFontSize(Math.min(24, terminalFontSize + 2))}
                              className="p-1 text-gray-600 hover:bg-gray-200 rounded disabled:opacity-50"
                              title="Zoom in"
                              disabled={terminalFontSize >= 24}
                            >
                              <ZoomIn className="h-4 w-4" />
                            </button>
                          </div>
                          <button
                            onClick={() => connectTerminal(selectedId!)}
                            disabled={selectedDetail?.state !== 'running'}
                            className="ml-auto px-3 py-1 text-sm bg-blue-600 text-white rounded hover:bg-blue-700 disabled:opacity-50"
                          >
                            Connect
                          </button>
                        </>
                      )}
                    </div>

                    {/* Terminal content - xterm.js container */}
                    <div className="flex-1 relative bg-[#1a1a1a]" style={{ minHeight: '300px' }}>
                      {!terminalConnected && (
                        <div className="absolute inset-0 flex items-center justify-center z-10 bg-[#1a1a1a]">
                          <div className="text-gray-500 text-center">
                            Click "Connect" to start terminal session.
                            {selectedDetail?.state !== 'running' && (
                              <span className="text-yellow-500 block mt-2">
                                Container must be running to connect.
                              </span>
                            )}
                          </div>
                        </div>
                      )}
                      <div
                        ref={terminalRef}
                        className={`absolute inset-0 p-1 ${!terminalConnected ? 'invisible' : ''}`}
                        onClick={() => xtermRef.current?.focus()}
                      />
                    </div>

                    {/* Help text */}
                    {terminalConnected && (
                      <div className="flex-shrink-0 px-2 py-1 bg-gray-800 text-xs text-gray-400">
                        Type directly in terminal • Ctrl+C: interrupt • Ctrl+D: exit • Ctrl+L: clear
                      </div>
                    )}
                  </div>
                )}
              </div>
            </>
          ) : (
            <div className="flex-1 flex items-center justify-center text-gray-400 bg-gray-50">
              <div className="text-center">
                <Box className="h-16 w-16 mx-auto mb-4 text-gray-300" />
                <p className="font-medium">Select a container to view details</p>
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Confirm Dialog */}
      <ConfirmDialog
        isOpen={confirmDialog.isOpen}
        onClose={closeConfirmDialog}
        onConfirm={handleConfirmAction}
        {...getConfirmDialogConfig()}
      />
    </div>
  );
}
