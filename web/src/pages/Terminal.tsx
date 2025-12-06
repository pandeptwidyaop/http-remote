import { useEffect, useRef, useState, useCallback } from 'react';
import { Terminal as XTerm } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import { Maximize2, Minimize2 } from 'lucide-react';
import '@xterm/xterm/css/xterm.css';

const API_BASE = import.meta.env.VITE_API_BASE_URL || '';

function Terminal() {
  const terminalRef = useRef<HTMLDivElement>(null);
  const terminalContainerRef = useRef<HTMLDivElement>(null);
  const xtermRef = useRef<XTerm | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const fitAddonRef = useRef<FitAddon | null>(null);
  const [isFullscreen, setIsFullscreen] = useState(false);
  const [isConnected, setIsConnected] = useState(false);

  useEffect(() => {
    if (!terminalRef.current) return;

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
    xterm.open(terminalRef.current);
    fitAddon.fit();

    xtermRef.current = xterm;
    fitAddonRef.current = fitAddon;

    // Connect to WebSocket
    // Get path prefix from current URL (e.g., /devops)
    const pathPrefix = window.location.pathname.split('/').slice(0, 2).join('/') || '';
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}${pathPrefix}/api/terminal/ws`;

    const ws = new WebSocket(wsUrl);
    wsRef.current = ws;

    ws.onopen = () => {
      xterm.writeln('\x1b[1;32mConnected to remote terminal\x1b[0m\r\n');
      setIsConnected(true);
    };

    ws.onmessage = (event) => {
      if (event.data instanceof Blob) {
        // Binary data
        event.data.arrayBuffer().then((buffer) => {
          xterm.write(new Uint8Array(buffer));
        });
      } else {
        // Text data
        xterm.write(event.data);
      }
    };

    ws.onerror = () => {
      xterm.writeln('\r\n\x1b[1;31mWebSocket error occurred\x1b[0m\r\n');
    };

    ws.onclose = () => {
      xterm.writeln('\r\n\x1b[1;33mConnection closed\x1b[0m\r\n');
      setIsConnected(false);
    };

    // Send terminal input to WebSocket
    xterm.onData((data) => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(data);
      }
    });

    // Handle window resize
    const handleResize = () => {
      fitAddon.fit();
    };

    window.addEventListener('resize', handleResize);

    // Handle fullscreen change
    const handleFullscreenChange = () => {
      const isNowFullscreen = !!document.fullscreenElement;
      setIsFullscreen(isNowFullscreen);
      // Delay fit to allow DOM to update
      setTimeout(() => {
        fitAddon.fit();
      }, 100);
    };

    document.addEventListener('fullscreenchange', handleFullscreenChange);

    // Cleanup
    return () => {
      window.removeEventListener('resize', handleResize);
      document.removeEventListener('fullscreenchange', handleFullscreenChange);
      ws.close();
      xterm.dispose();
    };
  }, []);

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

  return (
    <div className={`${isFullscreen ? '' : 'space-y-6'}`}>
      {!isFullscreen && (
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Remote Terminal</h1>
          <p className="mt-1 text-sm text-gray-600">
            Interactive shell access to the server
          </p>
        </div>
      )}

      <div
        ref={terminalContainerRef}
        className={`bg-white shadow rounded-lg ${isFullscreen ? 'fixed inset-0 z-50 flex flex-col rounded-none' : 'p-6'}`}
      >
        <div className={`${isFullscreen ? 'flex flex-col h-full' : 'space-y-4'}`}>
          <div className={`flex items-center justify-between ${isFullscreen ? 'p-4 bg-gray-800 border-b border-gray-700' : ''}`}>
            <h2 className={`text-lg font-medium ${isFullscreen ? 'text-white' : 'text-gray-900'}`}>
              Shell Session
            </h2>
            <div className="flex items-center gap-3">
              {isConnected ? (
                <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800">
                  <span className="w-2 h-2 mr-1.5 bg-green-400 rounded-full animate-pulse"></span>
                  Connected
                </span>
              ) : (
                <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-red-100 text-red-800">
                  <span className="w-2 h-2 mr-1.5 bg-red-400 rounded-full"></span>
                  Disconnected
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

          <div
            className={`border rounded-md overflow-hidden ${
              isFullscreen
                ? 'flex-1 border-gray-700 bg-[#1e1e1e]'
                : 'border-gray-300'
            }`}
          >
            <div
              ref={terminalRef}
              className={`${isFullscreen ? 'h-full' : ''} p-2`}
              style={isFullscreen ? { height: '100%' } : undefined}
            />
          </div>

          {!isFullscreen && (
            <div className="bg-blue-50 border border-blue-200 rounded-md p-4">
              <div className="flex">
                <div className="flex-shrink-0">
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
                </div>
                <div className="ml-3 flex-1">
                  <p className="text-sm text-blue-700">
                    <strong>Security Notice:</strong> This terminal provides direct shell access to the server.
                    All commands are executed with the server's permissions. Exercise caution when running commands.
                  </p>
                </div>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

export default Terminal;
