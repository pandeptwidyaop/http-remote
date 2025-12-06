import { useState, useEffect, useRef } from 'react';
import { useParams, Link, useNavigate } from 'react-router-dom';
import { ArrowLeft, Play, Trash2 } from 'lucide-react';
import { api } from '@/api/client';
import { API_ENDPOINTS } from '@/lib/config';
import { useSSE } from '@/hooks/useSSE';
import type { App, Command, ExecuteCommandResponse } from '@/types';
import Button from '@/components/ui/Button';
import Card from '@/components/ui/Card';
import Badge from '@/components/ui/Badge';

export default function Execute() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [command, setCommand] = useState<Command | null>(null);
  const [app, setApp] = useState<App | null>(null);
  const [loading, setLoading] = useState(true);
  const [executing, setExecuting] = useState(false);
  const [executionId, setExecutionId] = useState<string | null>(null);
  const [output, setOutput] = useState('');
  const [status, setStatus] = useState<'pending' | 'running' | 'success' | 'failed'>('pending');
  const [exitCode, setExitCode] = useState<number | null>(null);
  const outputRef = useRef<HTMLPreElement>(null);

  useEffect(() => {
    if (id) {
      fetchCommand();
    }
  }, [id]);

  const fetchCommand = async () => {
    if (!id) return;

    try {
      const commandData = await api.get<Command>(API_ENDPOINTS.command(id));
      setCommand(commandData || null);

      if (commandData?.app_id) {
        const appData = await api.get<App>(API_ENDPOINTS.app(commandData.app_id));
        setApp(appData || null);
      }
    } catch (error: any) {
      console.error('Failed to fetch command:', error);
      setCommand(null);
      setApp(null);
      if (error.status === 404) {
        navigate('/apps');
      }
    } finally {
      setLoading(false);
    }
  };

  const { connected } = useSSE(
    executionId ? API_ENDPOINTS.executionStream(executionId) : null,
    {
      onMessage: (data) => {
        setOutput((prev) => prev + data + '\n');
        // Auto scroll to bottom
        if (outputRef.current) {
          outputRef.current.scrollTop = outputRef.current.scrollHeight;
        }
      },
      onComplete: (data) => {
        setStatus(data.status as any);
        setExitCode(data.exit_code);
        setExecuting(false);
      },
      onError: () => {
        setStatus('failed');
        setExecuting(false);
      },
    }
  );

  const handleExecute = async () => {
    if (!id) return;

    setExecuting(true);
    setOutput('');
    setStatus('pending');
    setExitCode(null);

    try {
      const response = await api.post<ExecuteCommandResponse>(
        API_ENDPOINTS.executeCommand(id)
      );
      setExecutionId(response.execution_id);
      setStatus('running');
    } catch (error: any) {
      setOutput(`Error: ${error.message}`);
      setStatus('failed');
      setExecuting(false);
    }
  };

  const handleClear = () => {
    setOutput('');
    setStatus('pending');
    setExitCode(null);
    setExecutionId(null);
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
      </div>
    );
  }

  if (!command || !app) {
    return <div>Command not found</div>;
  }

  return (
    <div className="space-y-6">
      {/* Breadcrumb */}
      <div className="flex items-center space-x-2 text-sm text-gray-600">
        <Link to="/apps" className="hover:text-gray-900">Apps</Link>
        <span>/</span>
        <Link to={`/apps/${app.id}`} className="hover:text-gray-900">{app.name}</Link>
        <span>/</span>
        <span className="text-gray-900 font-medium">{command.name}</span>
      </div>

      {/* Header */}
      <div className="flex items-start justify-between">
        <div>
          <div className="flex items-center space-x-3">
            <Link to={`/apps/${app.id}`}>
              <Button variant="ghost" size="sm">
                <ArrowLeft className="h-4 w-4" />
              </Button>
            </Link>
            <h1 className="text-3xl font-bold text-gray-900">
              Execute: {command.name}
            </h1>
          </div>
          {command.description && (
            <p className="text-gray-600 mt-2">{command.description}</p>
          )}
        </div>
      </div>

      {/* Command Info */}
      <Card className="p-6">
        <div className="space-y-3">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <span className="text-sm font-medium text-gray-600">App:</span>
              <p className="text-gray-900">{app.name}</p>
            </div>
            <div>
              <span className="text-sm font-medium text-gray-600">Working Directory:</span>
              <p className="text-gray-900 font-mono text-sm">{app.working_dir}</p>
            </div>
          </div>
          <div>
            <span className="text-sm font-medium text-gray-600">Command:</span>
            <pre className="bg-gray-900 text-green-400 p-3 rounded-md mt-2 text-sm overflow-x-auto font-mono">
              {command.command}
            </pre>
          </div>
          <div>
            <span className="text-sm font-medium text-gray-600">Timeout:</span>
            <span className="text-gray-900 ml-2">{command.timeout_seconds} seconds</span>
          </div>
        </div>
      </Card>

      {/* Execute Controls */}
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-4">
          <Button
            variant="primary"
            size="lg"
            onClick={handleExecute}
            disabled={executing}
            loading={executing}
          >
            <Play className="h-5 w-5 mr-2" />
            {executing ? 'Executing...' : 'Execute Command'}
          </Button>
          {status !== 'pending' && (
            <Badge status={status}>
              {status}
              {exitCode !== null && ` (exit code: ${exitCode})`}
            </Badge>
          )}
        </div>
        <Button variant="ghost" onClick={handleClear}>
          <Trash2 className="h-4 w-4 mr-2" />
          Clear
        </Button>
      </div>

      {/* Output */}
      <Card>
        <div className="px-6 py-4 border-b border-gray-200 flex items-center justify-between">
          <h2 className="text-lg font-semibold text-gray-900">Output</h2>
          {connected && (
            <span className="flex items-center text-sm text-green-600">
              <span className="animate-pulse h-2 w-2 bg-green-600 rounded-full mr-2"></span>
              Live
            </span>
          )}
        </div>
        <div className="p-0">
          <pre
            ref={outputRef}
            className="bg-gray-900 text-green-400 p-6 font-mono text-sm overflow-auto h-96 scrollbar-hide"
          >
            {output || 'No output yet. Click "Execute Command" to start.'}
          </pre>
        </div>
      </Card>
    </div>
  );
}
