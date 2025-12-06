import { useEffect, useRef, useState, useCallback } from 'react';
import { Terminal as XTerm } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import { Maximize2, Minimize2, Plus, X, Circle, Settings, RefreshCw, Trash2 } from 'lucide-react';
import '@xterm/xterm/css/xterm.css';
import { api } from '@/api/client';
import { getPathPrefix } from '@/lib/config';

interface ServerSession {
  id: string;
  name: string;
  created_at: string;
  last_activity: string;
  client_count: number;
  is_active: boolean;
}

interface TerminalSession {
  id: string;
  name: string;
  serverId: string | null; // null for ephemeral sessions
  xterm: XTerm | null;
  ws: WebSocket | null;
  fitAddon: FitAddon | null;
  isConnected: boolean;
  isPersistent: boolean;
}

const PERSISTENT_MODE_KEY = 'terminal_persistent_mode';

function generateSessionId(): string {
  return `local-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
}

function Terminal() {
  const terminalContainerRef = useRef<HTMLDivElement>(null);
  const terminalRefs = useRef<Map<string, HTMLDivElement>>(new Map());
  const [sessions, setSessions] = useState<TerminalSession[]>([]);
  const [activeSessionId, setActiveSessionId] = useState<string | null>(null);
  const [isFullscreen, setIsFullscreen] = useState(false);
  const [sessionCounter, setSessionCounter] = useState(1);
  const [persistentMode, setPersistentMode] = useState(() => {
    return localStorage.getItem(PERSISTENT_MODE_KEY) === 'true';
  });
  const [showSettings, setShowSettings] = useState(false);
  const [serverSessions, setServerSessions] = useState<ServerSession[]>([]);
  const [loadingServerSessions, setLoadingServerSessions] = useState(false);

  // Fetch server-side sessions
  const fetchServerSessions = useCallback(async () => {
    if (!persistentMode) return;

    setLoadingServerSessions(true);
    try {
      const response = await api.get<{ sessions: ServerSession[] }>('/api/terminal/sessions');
      setServerSessions(response.sessions || []);
    } catch (error) {
      console.error('Failed to fetch server sessions:', error);
    } finally {
      setLoadingServerSessions(false);
    }
  }, [persistentMode]);

  // Create a new terminal session
  const createSession = useCallback((serverSessionId?: string) => {
    const sessionId = generateSessionId();
    const sessionName = serverSessionId
      ? `Persistent ${sessionCounter}`
      : `Terminal ${sessionCounter}`;
    setSessionCounter((prev) => prev + 1);

    const newSession: TerminalSession = {
      id: sessionId,
      name: sessionName,
      serverId: serverSessionId || null,
      xterm: null,
      ws: null,
      fitAddon: null,
      isConnected: false,
      isPersistent: persistentMode,
    };

    setSessions((prev) => [...prev, newSession]);
    setActiveSessionId(sessionId);

    return sessionId;
  }, [sessionCounter, persistentMode]);

  // Initialize terminal for a session
  const initializeTerminal = useCallback((sessionId: string, container: HTMLDivElement) => {
    const session = sessions.find((s) => s.id === sessionId);
    if (!session || session.xterm) return;

    // Create terminal instance
    const xterm = new XTerm({
      cursorBlink: true,
      fontSize: 14,
      fontFamily: 'Menlo, Monaco, "Courier New", monospace',
      theme: {
        background: '#1e1e1e',
        foreground: '#d4d4d4',
      },
      rows: 30,
      cols: 100,
    });

    // Create fit addon
    const fitAddon = new FitAddon();
    xterm.loadAddon(fitAddon);

    // Open terminal in DOM
    xterm.open(container);
    fitAddon.fit();

    // Build WebSocket URL with persistent mode parameters
    const pathPrefix = getPathPrefix();
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    let wsUrl = `${protocol}//${window.location.host}${pathPrefix}/api/terminal/ws`;

    const params = new URLSearchParams();
    if (session.isPersistent) {
      params.set('persistent', 'true');
      if (session.serverId) {
        params.set('session_id', session.serverId);
      }
    }

    if (params.toString()) {
      wsUrl += `?${params.toString()}`;
    }

    const ws = new WebSocket(wsUrl);

    ws.onopen = () => {
      const modeText = session.isPersistent ? 'persistent' : 'ephemeral';
      xterm.writeln(`\x1b[1;32mConnected to remote terminal (${modeText} mode)\x1b[0m\r\n`);
      setSessions((prev) =>
        prev.map((s) => (s.id === sessionId ? { ...s, isConnected: true } : s))
      );
    };

    ws.onmessage = (event) => {
      const handleData = (data: string) => {
        // Check if it's a session info message
        if (data.startsWith('{"type":"session_info"')) {
          try {
            const info = JSON.parse(data);
            if (info.session && info.session.id) {
              setSessions((prev) =>
                prev.map((s) =>
                  s.id === sessionId
                    ? { ...s, serverId: info.session.id, name: info.session.name || s.name }
                    : s
                )
              );
            }
          } catch {
            // Not JSON, treat as terminal output
            xterm.write(data);
          }
        } else {
          xterm.write(data);
        }
      };

      if (event.data instanceof Blob) {
        event.data.arrayBuffer().then((buffer) => {
          const decoder = new TextDecoder();
          handleData(decoder.decode(buffer));
        });
      } else {
        handleData(event.data);
      }
    };

    ws.onerror = () => {
      xterm.writeln('\r\n\x1b[1;31mWebSocket error occurred\x1b[0m\r\n');
    };

    ws.onclose = () => {
      const session = sessions.find((s) => s.id === sessionId);
      if (session?.isPersistent) {
        xterm.writeln('\r\n\x1b[1;33mConnection closed. Session is still running on server.\x1b[0m');
        xterm.writeln('\x1b[1;33mReconnect to resume.\x1b[0m\r\n');
      } else {
        xterm.writeln('\r\n\x1b[1;33mConnection closed\x1b[0m\r\n');
      }
      setSessions((prev) =>
        prev.map((s) => (s.id === sessionId ? { ...s, isConnected: false } : s))
      );
    };

    // Send terminal input to WebSocket
    xterm.onData((data) => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(data);
      }
    });

    // Update session with terminal instances
    setSessions((prev) =>
      prev.map((s) =>
        s.id === sessionId ? { ...s, xterm, ws, fitAddon } : s
      )
    );
  }, [sessions]);

  // Reconnect to a server session
  const reconnectSession = useCallback((sessionId: string) => {
    const session = sessions.find((s) => s.id === sessionId);
    if (!session || !session.serverId) return;

    // Close existing WebSocket if any
    if (session.ws) {
      session.ws.close();
    }

    // Clear terminal
    if (session.xterm) {
      session.xterm.clear();
      session.xterm.writeln('\x1b[1;36mReconnecting to session...\x1b[0m\r\n');
    }

    // Reinitialize with same session
    const container = terminalRefs.current.get(sessionId);
    if (container && session.xterm) {
      const pathPrefix = getPathPrefix();
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
      const wsUrl = `${protocol}//${window.location.host}${pathPrefix}/api/terminal/ws?persistent=true&session_id=${session.serverId}`;

      const ws = new WebSocket(wsUrl);

      ws.onopen = () => {
        session.xterm?.writeln('\x1b[1;32mReconnected!\x1b[0m\r\n');
        setSessions((prev) =>
          prev.map((s) => (s.id === sessionId ? { ...s, ws, isConnected: true } : s))
        );
      };

      ws.onmessage = (event) => {
        if (event.data instanceof Blob) {
          event.data.arrayBuffer().then((buffer) => {
            const decoder = new TextDecoder();
            session.xterm?.write(decoder.decode(buffer));
          });
        } else {
          if (!event.data.startsWith('{"type":"session_info"')) {
            session.xterm?.write(event.data);
          }
        }
      };

      ws.onerror = () => {
        session.xterm?.writeln('\r\n\x1b[1;31mReconnection failed\x1b[0m\r\n');
      };

      ws.onclose = () => {
        session.xterm?.writeln('\r\n\x1b[1;33mConnection closed\x1b[0m\r\n');
        setSessions((prev) =>
          prev.map((s) => (s.id === sessionId ? { ...s, isConnected: false } : s))
        );
      };

      session.xterm.onData((data) => {
        if (ws.readyState === WebSocket.OPEN) {
          ws.send(data);
        }
      });

      setSessions((prev) =>
        prev.map((s) => (s.id === sessionId ? { ...s, ws } : s))
      );
    }
  }, [sessions]);

  // Close a terminal session
  const closeSession = useCallback(async (sessionId: string, deleteOnServer = false) => {
    const session = sessions.find((s) => s.id === sessionId);
    if (session) {
      session.ws?.close();
      session.xterm?.dispose();

      // Delete persistent session on server if requested
      if (deleteOnServer && session.serverId && session.isPersistent) {
        try {
          await api.delete(`/api/terminal/sessions/${session.serverId}`);
        } catch (error) {
          console.error('Failed to delete server session:', error);
        }
      }
    }

    setSessions((prev) => {
      const newSessions = prev.filter((s) => s.id !== sessionId);

      if (activeSessionId === sessionId && newSessions.length > 0) {
        setActiveSessionId(newSessions[newSessions.length - 1].id);
      } else if (newSessions.length === 0) {
        setActiveSessionId(null);
      }

      return newSessions;
    });

    terminalRefs.current.delete(sessionId);
  }, [sessions, activeSessionId]);

  // Attach to existing server session
  const attachToServerSession = useCallback((serverSession: ServerSession) => {
    // Check if already attached
    const existing = sessions.find((s) => s.serverId === serverSession.id);
    if (existing) {
      setActiveSessionId(existing.id);
      return;
    }

    const sessionId = generateSessionId();
    const newSession: TerminalSession = {
      id: sessionId,
      name: serverSession.name,
      serverId: serverSession.id,
      xterm: null,
      ws: null,
      fitAddon: null,
      isConnected: false,
      isPersistent: true,
    };

    setSessions((prev) => [...prev, newSession]);
    setActiveSessionId(sessionId);
  }, [sessions]);

  // Toggle persistent mode
  const togglePersistentMode = useCallback((enabled: boolean) => {
    setPersistentMode(enabled);
    localStorage.setItem(PERSISTENT_MODE_KEY, enabled ? 'true' : 'false');
    if (enabled) {
      fetchServerSessions();
    }
  }, [fetchServerSessions]);

  // Create first session on mount
  useEffect(() => {
    if (sessions.length === 0) {
      createSession();
    }
  }, []);

  // Fetch server sessions when persistent mode is enabled
  useEffect(() => {
    if (persistentMode) {
      fetchServerSessions();
    }
  }, [persistentMode, fetchServerSessions]);

  // Initialize terminal when container is available
  useEffect(() => {
    if (activeSessionId) {
      const container = terminalRefs.current.get(activeSessionId);
      const session = sessions.find((s) => s.id === activeSessionId);
      if (container && session && !session.xterm) {
        initializeTerminal(activeSessionId, container);
      }
    }
  }, [activeSessionId, sessions, initializeTerminal]);

  // Handle resize for active session
  useEffect(() => {
    const handleResize = () => {
      const session = sessions.find((s) => s.id === activeSessionId);
      if (session?.fitAddon) {
        session.fitAddon.fit();
      }
    };

    window.addEventListener('resize', handleResize);
    return () => window.removeEventListener('resize', handleResize);
  }, [sessions, activeSessionId]);

  // Handle fullscreen change
  useEffect(() => {
    const handleFullscreenChange = () => {
      const isNowFullscreen = !!document.fullscreenElement;
      setIsFullscreen(isNowFullscreen);

      setTimeout(() => {
        const session = sessions.find((s) => s.id === activeSessionId);
        if (session?.fitAddon) {
          session.fitAddon.fit();
        }
      }, 100);
    };

    document.addEventListener('fullscreenchange', handleFullscreenChange);
    return () => document.removeEventListener('fullscreenchange', handleFullscreenChange);
  }, [sessions, activeSessionId]);

  // Fit terminal when switching tabs
  useEffect(() => {
    const session = sessions.find((s) => s.id === activeSessionId);
    if (session?.fitAddon) {
      setTimeout(() => session.fitAddon?.fit(), 50);
    }
  }, [activeSessionId, sessions]);

  // Toggle fullscreen
  const toggleFullscreen = useCallback(() => {
    if (!terminalContainerRef.current) return;

    if (!document.fullscreenElement) {
      terminalContainerRef.current.requestFullscreen().catch((err) => {
        console.error('Error attempting to enable fullscreen:', err);
      });
    } else {
      document.exitFullscreen();
    }
  }, []);

  // Set terminal ref
  const setTerminalRef = useCallback((sessionId: string) => (el: HTMLDivElement | null) => {
    if (el) {
      terminalRefs.current.set(sessionId, el);
    }
  }, []);

  const activeSession = sessions.find((s) => s.id === activeSessionId);

  return (
    <div className={`${isFullscreen ? '' : 'space-y-6'}`}>
      {!isFullscreen && (
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold text-gray-900">Remote Terminal</h1>
            <p className="mt-1 text-sm text-gray-600">
              Interactive shell access to the server.
              {persistentMode && ' Sessions persist even when you navigate away.'}
            </p>
          </div>
          <button
            onClick={() => setShowSettings(!showSettings)}
            className={`p-2 rounded-md transition-colors ${
              showSettings ? 'bg-blue-100 text-blue-700' : 'text-gray-500 hover:bg-gray-100'
            }`}
            title="Terminal Settings"
          >
            <Settings className="h-5 w-5" />
          </button>
        </div>
      )}

      {/* Settings Panel */}
      {showSettings && !isFullscreen && (
        <div className="bg-white shadow rounded-lg p-4 space-y-4">
          <h3 className="text-lg font-medium text-gray-900">Terminal Settings</h3>

          <div className="flex items-center justify-between">
            <div>
              <label className="text-sm font-medium text-gray-700">Persistent Sessions</label>
              <p className="text-xs text-gray-500">
                Keep terminal sessions running when you navigate away or logout
              </p>
            </div>
            <button
              onClick={() => togglePersistentMode(!persistentMode)}
              className={`relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none ${
                persistentMode ? 'bg-blue-600' : 'bg-gray-200'
              }`}
            >
              <span
                className={`pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out ${
                  persistentMode ? 'translate-x-5' : 'translate-x-0'
                }`}
              />
            </button>
          </div>

          {/* Server Sessions List */}
          {persistentMode && (
            <div className="border-t pt-4">
              <div className="flex items-center justify-between mb-2">
                <h4 className="text-sm font-medium text-gray-700">Active Server Sessions</h4>
                <button
                  onClick={fetchServerSessions}
                  disabled={loadingServerSessions}
                  className="p-1 text-gray-500 hover:text-gray-700 disabled:opacity-50"
                  title="Refresh"
                >
                  <RefreshCw className={`h-4 w-4 ${loadingServerSessions ? 'animate-spin' : ''}`} />
                </button>
              </div>

              {serverSessions.length === 0 ? (
                <p className="text-sm text-gray-500">No active server sessions</p>
              ) : (
                <div className="space-y-2">
                  {serverSessions.map((serverSession) => {
                    const isAttached = sessions.some((s) => s.serverId === serverSession.id);
                    return (
                      <div
                        key={serverSession.id}
                        className="flex items-center justify-between p-2 bg-gray-50 rounded-md"
                      >
                        <div className="flex items-center gap-2">
                          <Circle
                            className={`h-2 w-2 ${
                              serverSession.is_active ? 'text-green-500 fill-green-500' : 'text-gray-400 fill-gray-400'
                            }`}
                          />
                          <span className="text-sm font-medium">{serverSession.name}</span>
                          <span className="text-xs text-gray-500">
                            ({serverSession.client_count} clients)
                          </span>
                        </div>
                        <div className="flex items-center gap-1">
                          {!isAttached && (
                            <button
                              onClick={() => attachToServerSession(serverSession)}
                              className="px-2 py-1 text-xs bg-blue-600 text-white rounded hover:bg-blue-700"
                            >
                              Attach
                            </button>
                          )}
                          <button
                            onClick={async () => {
                              try {
                                await api.delete(`/api/terminal/sessions/${serverSession.id}`);
                                fetchServerSessions();
                                // Close local session if attached
                                const localSession = sessions.find((s) => s.serverId === serverSession.id);
                                if (localSession) {
                                  closeSession(localSession.id, false);
                                }
                              } catch (error) {
                                console.error('Failed to delete session:', error);
                              }
                            }}
                            className="p-1 text-red-500 hover:text-red-700"
                            title="Delete session"
                          >
                            <Trash2 className="h-4 w-4" />
                          </button>
                        </div>
                      </div>
                    );
                  })}
                </div>
              )}
            </div>
          )}
        </div>
      )}

      <div
        ref={terminalContainerRef}
        className={`bg-white shadow rounded-lg ${isFullscreen ? 'fixed inset-0 z-50 flex flex-col rounded-none' : ''}`}
      >
        {/* Tab Bar */}
        <div className={`flex items-center border-b ${isFullscreen ? 'bg-gray-800 border-gray-700' : 'bg-gray-50 border-gray-200 rounded-t-lg'}`}>
          <div className="flex-1 flex items-center overflow-x-auto">
            {sessions.map((session) => (
              <div
                key={session.id}
                className={`group flex items-center gap-2 px-4 py-2 cursor-pointer border-r transition-colors ${
                  isFullscreen ? 'border-gray-700' : 'border-gray-200'
                } ${
                  session.id === activeSessionId
                    ? isFullscreen
                      ? 'bg-gray-900 text-white'
                      : 'bg-white text-gray-900'
                    : isFullscreen
                    ? 'bg-gray-800 text-gray-400 hover:bg-gray-700 hover:text-gray-200'
                    : 'bg-gray-100 text-gray-600 hover:bg-gray-200 hover:text-gray-900'
                }`}
                onClick={() => setActiveSessionId(session.id)}
              >
                <Circle
                  className={`h-2 w-2 ${
                    session.isConnected ? 'text-green-500 fill-green-500' : 'text-red-500 fill-red-500'
                  }`}
                />
                <span className="text-sm font-medium whitespace-nowrap">
                  {session.name}
                  {session.isPersistent && (
                    <span className="ml-1 text-xs opacity-50">(P)</span>
                  )}
                </span>

                {/* Reconnect button for disconnected persistent sessions */}
                {!session.isConnected && session.isPersistent && session.serverId && (
                  <button
                    onClick={(e) => {
                      e.stopPropagation();
                      reconnectSession(session.id);
                    }}
                    className={`p-0.5 rounded transition-opacity ${
                      isFullscreen ? 'hover:bg-gray-600' : 'hover:bg-gray-300'
                    }`}
                    title="Reconnect"
                  >
                    <RefreshCw className="h-3 w-3" />
                  </button>
                )}

                {sessions.length > 1 && (
                  <button
                    onClick={(e) => {
                      e.stopPropagation();
                      closeSession(session.id, true);
                    }}
                    className={`p-0.5 rounded opacity-0 group-hover:opacity-100 transition-opacity ${
                      isFullscreen ? 'hover:bg-gray-600' : 'hover:bg-gray-300'
                    }`}
                    title={session.isPersistent ? 'Close and delete session' : 'Close session'}
                  >
                    <X className="h-3 w-3" />
                  </button>
                )}
              </div>
            ))}

            {/* New Tab Button */}
            <button
              onClick={() => createSession()}
              className={`flex items-center gap-1 px-3 py-2 transition-colors ${
                isFullscreen
                  ? 'text-gray-400 hover:text-white hover:bg-gray-700'
                  : 'text-gray-500 hover:text-gray-700 hover:bg-gray-200'
              }`}
              title={persistentMode ? 'New persistent session' : 'New terminal session'}
            >
              <Plus className="h-4 w-4" />
            </button>
          </div>

          {/* Right side controls */}
          <div className={`flex items-center gap-2 px-3 ${isFullscreen ? 'border-l border-gray-700' : 'border-l border-gray-200'}`}>
            {activeSession && (
              <>
                {activeSession.isPersistent && (
                  <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ${
                    isFullscreen ? 'bg-blue-900 text-blue-200' : 'bg-blue-100 text-blue-800'
                  }`}>
                    Persistent
                  </span>
                )}
                <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ${
                  activeSession.isConnected
                    ? 'bg-green-100 text-green-800'
                    : 'bg-red-100 text-red-800'
                }`}>
                  {activeSession.isConnected ? 'Connected' : 'Disconnected'}
                </span>
              </>
            )}
            <button
              onClick={toggleFullscreen}
              className={`p-2 rounded-md transition-colors ${
                isFullscreen
                  ? 'text-gray-300 hover:text-white hover:bg-gray-700'
                  : 'text-gray-500 hover:text-gray-700 hover:bg-gray-100'
              }`}
              title={isFullscreen ? 'Exit fullscreen (Esc)' : 'Enter fullscreen'}
            >
              {isFullscreen ? (
                <Minimize2 className="h-5 w-5" />
              ) : (
                <Maximize2 className="h-5 w-5" />
              )}
            </button>
          </div>
        </div>

        {/* Terminal Container */}
        <div className={`${isFullscreen ? 'flex-1' : 'p-4'}`}>
          {sessions.map((session) => (
            <div
              key={session.id}
              className={`${session.id === activeSessionId ? 'block' : 'hidden'} ${
                isFullscreen ? 'h-full' : ''
              }`}
            >
              <div
                ref={setTerminalRef(session.id)}
                className={`border rounded-md overflow-hidden ${
                  isFullscreen
                    ? 'h-full border-gray-700 bg-[#1e1e1e]'
                    : 'border-gray-300'
                }`}
                style={isFullscreen ? { height: '100%' } : { minHeight: '400px' }}
              />
            </div>
          ))}

          {sessions.length === 0 && (
            <div className="flex items-center justify-center h-64 text-gray-500">
              <button
                onClick={() => createSession()}
                className="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 transition-colors"
              >
                <Plus className="h-5 w-5" />
                Create Terminal Session
              </button>
            </div>
          )}
        </div>

        {/* Info Notice - only in non-fullscreen */}
        {!isFullscreen && (
          <div className="p-4 pt-0">
            <div className={`border rounded-md p-4 ${
              persistentMode
                ? 'bg-blue-50 border-blue-200'
                : 'bg-yellow-50 border-yellow-200'
            }`}>
              <div className="flex">
                <div className="flex-shrink-0">
                  {persistentMode ? (
                    <svg
                      className="h-5 w-5 text-blue-400"
                      xmlns="http://www.w3.org/2000/svg"
                      viewBox="0 0 20 20"
                      fill="currentColor"
                    >
                      <path
                        fillRule="evenodd"
                        d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z"
                        clipRule="evenodd"
                      />
                    </svg>
                  ) : (
                    <svg
                      className="h-5 w-5 text-yellow-400"
                      xmlns="http://www.w3.org/2000/svg"
                      viewBox="0 0 20 20"
                      fill="currentColor"
                    >
                      <path
                        fillRule="evenodd"
                        d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z"
                        clipRule="evenodd"
                      />
                    </svg>
                  )}
                </div>
                <div className="ml-3">
                  {persistentMode ? (
                    <p className="text-sm text-blue-700">
                      <strong>Persistent Mode Enabled:</strong> Terminal sessions will keep running on the server even when you close the browser or navigate away. Use the settings panel to manage your sessions.
                    </p>
                  ) : (
                    <p className="text-sm text-yellow-700">
                      <strong>Security Notice:</strong> This terminal provides direct shell access to the server.
                      All commands are executed with the server process permissions. Enable persistent mode in settings to keep sessions running when you navigate away.
                    </p>
                  )}
                </div>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

export default Terminal;
