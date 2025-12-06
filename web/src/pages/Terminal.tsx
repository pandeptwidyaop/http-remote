import { useEffect, useRef, useState, useCallback } from 'react';
import { Terminal as XTerm } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import { Maximize2, Minimize2, Plus, X, Circle } from 'lucide-react';
import '@xterm/xterm/css/xterm.css';

interface TerminalSession {
  id: string;
  name: string;
  xterm: XTerm | null;
  ws: WebSocket | null;
  fitAddon: FitAddon | null;
  isConnected: boolean;
}

function generateSessionId(): string {
  return `session-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
}

function Terminal() {
  const terminalContainerRef = useRef<HTMLDivElement>(null);
  const terminalRefs = useRef<Map<string, HTMLDivElement>>(new Map());
  const [sessions, setSessions] = useState<TerminalSession[]>([]);
  const [activeSessionId, setActiveSessionId] = useState<string | null>(null);
  const [isFullscreen, setIsFullscreen] = useState(false);
  const [sessionCounter, setSessionCounter] = useState(1);

  // Create a new terminal session
  const createSession = useCallback(() => {
    const sessionId = generateSessionId();
    const sessionName = `Terminal ${sessionCounter}`;
    setSessionCounter((prev) => prev + 1);

    const newSession: TerminalSession = {
      id: sessionId,
      name: sessionName,
      xterm: null,
      ws: null,
      fitAddon: null,
      isConnected: false,
    };

    setSessions((prev) => [...prev, newSession]);
    setActiveSessionId(sessionId);

    return sessionId;
  }, [sessionCounter]);

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

    // Connect to WebSocket
    const pathPrefix = window.location.pathname.split('/').slice(0, 2).join('/') || '';
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}${pathPrefix}/api/terminal/ws`;

    const ws = new WebSocket(wsUrl);

    ws.onopen = () => {
      xterm.writeln('\x1b[1;32mConnected to remote terminal\x1b[0m\r\n');
      setSessions((prev) =>
        prev.map((s) => (s.id === sessionId ? { ...s, isConnected: true } : s))
      );
    };

    ws.onmessage = (event) => {
      const handleData = (data: string) => {
        xterm.write(data);
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
      xterm.writeln('\r\n\x1b[1;33mConnection closed\x1b[0m\r\n');
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

  // Close a terminal session
  const closeSession = useCallback((sessionId: string) => {
    const session = sessions.find((s) => s.id === sessionId);
    if (session) {
      session.ws?.close();
      session.xterm?.dispose();
    }

    setSessions((prev) => {
      const newSessions = prev.filter((s) => s.id !== sessionId);

      // If closing active session, switch to another
      if (activeSessionId === sessionId && newSessions.length > 0) {
        setActiveSessionId(newSessions[newSessions.length - 1].id);
      } else if (newSessions.length === 0) {
        setActiveSessionId(null);
      }

      return newSessions;
    });

    terminalRefs.current.delete(sessionId);
  }, [sessions, activeSessionId]);

  // Create first session on mount
  useEffect(() => {
    if (sessions.length === 0) {
      createSession();
    }
  }, []);

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

      // Delay fit to allow DOM to update
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
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Remote Terminal</h1>
          <p className="mt-1 text-sm text-gray-600">
            Interactive shell access to the server. Open multiple sessions with tabs.
          </p>
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
                <span className="text-sm font-medium whitespace-nowrap">{session.name}</span>
                {sessions.length > 1 && (
                  <button
                    onClick={(e) => {
                      e.stopPropagation();
                      closeSession(session.id);
                    }}
                    className={`p-0.5 rounded opacity-0 group-hover:opacity-100 transition-opacity ${
                      isFullscreen
                        ? 'hover:bg-gray-600'
                        : 'hover:bg-gray-300'
                    }`}
                    title="Close session"
                  >
                    <X className="h-3 w-3" />
                  </button>
                )}
              </div>
            ))}

            {/* New Tab Button */}
            <button
              onClick={createSession}
              className={`flex items-center gap-1 px-3 py-2 transition-colors ${
                isFullscreen
                  ? 'text-gray-400 hover:text-white hover:bg-gray-700'
                  : 'text-gray-500 hover:text-gray-700 hover:bg-gray-200'
              }`}
              title="New terminal session"
            >
              <Plus className="h-4 w-4" />
            </button>
          </div>

          {/* Right side controls */}
          <div className={`flex items-center gap-2 px-3 ${isFullscreen ? 'border-l border-gray-700' : 'border-l border-gray-200'}`}>
            {activeSession && (
              <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ${
                activeSession.isConnected
                  ? 'bg-green-100 text-green-800'
                  : 'bg-red-100 text-red-800'
              }`}>
                {activeSession.isConnected ? 'Connected' : 'Disconnected'}
              </span>
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
                onClick={createSession}
                className="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 transition-colors"
              >
                <Plus className="h-5 w-5" />
                Create Terminal Session
              </button>
            </div>
          )}
        </div>

        {/* Security Notice - only in non-fullscreen */}
        {!isFullscreen && (
          <div className="p-4 pt-0">
            <div className="bg-yellow-50 border border-yellow-200 rounded-md p-4">
              <div className="flex">
                <div className="flex-shrink-0">
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
                </div>
                <div className="ml-3">
                  <p className="text-sm text-yellow-700">
                    <strong>Security Notice:</strong> This terminal provides direct shell access to the server.
                    All commands are executed with the server process permissions. Use with caution.
                  </p>
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
