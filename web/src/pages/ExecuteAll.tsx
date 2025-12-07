import { useState, useEffect, useRef } from 'react';
import { useParams, Link, useNavigate } from 'react-router-dom';
import { ArrowLeft, Play, Trash2, CheckCircle, XCircle, Clock, Loader2 } from 'lucide-react';
import { api } from '@/api/client';
import { API_ENDPOINTS } from '@/lib/config';
import { useSSE } from '@/hooks/useSSE';
import type { App, Command, ExecuteCommandResponse } from '@/types';
import Button from '@/components/ui/Button';
import Card from '@/components/ui/Card';
import Badge from '@/components/ui/Badge';

type CommandStatus = 'pending' | 'running' | 'success' | 'failed' | 'skipped';

interface CommandExecution {
  command: Command;
  status: CommandStatus;
  output: string;
  exitCode: number | null;
  executionId: string | null;
}

export default function ExecuteAll() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [app, setApp] = useState<App | null>(null);
  const [commands, setCommands] = useState<Command[]>([]);
  const [loading, setLoading] = useState(true);
  const [executing, setExecuting] = useState(false);
  const [currentIndex, setCurrentIndex] = useState<number>(-1);
  const [executions, setExecutions] = useState<CommandExecution[]>([]);
  const [currentExecutionId, setCurrentExecutionId] = useState<string | null>(null);
  const [stopOnError, setStopOnError] = useState(true);
  const [overallStatus, setOverallStatus] = useState<'idle' | 'running' | 'success' | 'failed'>('idle');
  const outputRef = useRef<HTMLPreElement>(null);

  useEffect(() => {
    if (id) {
      fetchData();
    }
  }, [id]);

  const fetchData = async () => {
    if (!id) return;

    try {
      const [appData, commandsData] = await Promise.all([
        api.get<App>(API_ENDPOINTS.app(id)),
        api.get<Command[]>(API_ENDPOINTS.appCommands(id)),
      ]);
      setApp(appData || null);
      setCommands(commandsData || []);

      // Initialize executions state
      if (commandsData) {
        setExecutions(commandsData.map(cmd => ({
          command: cmd,
          status: 'pending',
          output: '',
          exitCode: null,
          executionId: null,
        })));
      }
    } catch (error: any) {
      console.error('Failed to fetch app:', error);
      setApp(null);
      setCommands([]);
      if (error.status === 404) {
        navigate('/apps');
      }
    } finally {
      setLoading(false);
    }
  };

  // SSE for current execution
  const { connected } = useSSE(
    currentExecutionId ? API_ENDPOINTS.executionStream(currentExecutionId) : null,
    {
      onMessage: (data) => {
        setExecutions(prev => prev.map((exec, idx) =>
          idx === currentIndex
            ? { ...exec, output: exec.output + data + '\n' }
            : exec
        ));
        // Auto scroll to bottom
        if (outputRef.current) {
          outputRef.current.scrollTop = outputRef.current.scrollHeight;
        }
      },
      onComplete: (data) => {
        const success = data.status === 'success';
        setExecutions(prev => prev.map((exec, idx) =>
          idx === currentIndex
            ? { ...exec, status: data.status as CommandStatus, exitCode: data.exit_code }
            : exec
        ));

        // Move to next command or finish
        if (success || !stopOnError) {
          executeNextCommand(currentIndex + 1);
        } else {
          // Mark remaining as skipped
          setExecutions(prev => prev.map((exec, idx) =>
            idx > currentIndex
              ? { ...exec, status: 'skipped' }
              : exec
          ));
          setOverallStatus('failed');
          setExecuting(false);
          setCurrentExecutionId(null);
        }
      },
      onError: () => {
        setExecutions(prev => prev.map((exec, idx) =>
          idx === currentIndex
            ? { ...exec, status: 'failed' }
            : exec
        ));

        if (stopOnError) {
          setExecutions(prev => prev.map((exec, idx) =>
            idx > currentIndex
              ? { ...exec, status: 'skipped' }
              : exec
          ));
          setOverallStatus('failed');
          setExecuting(false);
          setCurrentExecutionId(null);
        } else {
          executeNextCommand(currentIndex + 1);
        }
      },
    }
  );

  const executeNextCommand = async (index: number) => {
    if (index >= commands.length) {
      // All commands completed
      const anyFailed = executions.some(e => e.status === 'failed');
      setOverallStatus(anyFailed ? 'failed' : 'success');
      setExecuting(false);
      setCurrentExecutionId(null);
      return;
    }

    setCurrentIndex(index);
    const command = commands[index];

    // Update status to running
    setExecutions(prev => prev.map((exec, idx) =>
      idx === index
        ? { ...exec, status: 'running', output: '' }
        : exec
    ));

    try {
      const response = await api.post<ExecuteCommandResponse>(
        API_ENDPOINTS.executeCommand(command.id)
      );
      setCurrentExecutionId(response.execution_id);
      setExecutions(prev => prev.map((exec, idx) =>
        idx === index
          ? { ...exec, executionId: response.execution_id }
          : exec
      ));
    } catch (error: any) {
      setExecutions(prev => prev.map((exec, idx) =>
        idx === index
          ? { ...exec, status: 'failed', output: `Error: ${error.message}` }
          : exec
      ));

      if (stopOnError) {
        setExecutions(prev => prev.map((exec, idx) =>
          idx > index
            ? { ...exec, status: 'skipped' }
            : exec
        ));
        setOverallStatus('failed');
        setExecuting(false);
      } else {
        executeNextCommand(index + 1);
      }
    }
  };

  const handleExecuteAll = async () => {
    if (commands.length === 0) return;

    // Reset all executions
    setExecutions(commands.map(cmd => ({
      command: cmd,
      status: 'pending',
      output: '',
      exitCode: null,
      executionId: null,
    })));

    setExecuting(true);
    setOverallStatus('running');
    executeNextCommand(0);
  };

  const handleClear = () => {
    setExecutions(commands.map(cmd => ({
      command: cmd,
      status: 'pending',
      output: '',
      exitCode: null,
      executionId: null,
    })));
    setCurrentIndex(-1);
    setCurrentExecutionId(null);
    setOverallStatus('idle');
  };

  const getStatusIcon = (status: CommandStatus) => {
    switch (status) {
      case 'pending':
        return <Clock className="h-5 w-5 text-gray-400" />;
      case 'running':
        return <Loader2 className="h-5 w-5 text-blue-500 animate-spin" />;
      case 'success':
        return <CheckCircle className="h-5 w-5 text-green-500" />;
      case 'failed':
        return <XCircle className="h-5 w-5 text-red-500" />;
      case 'skipped':
        return <Clock className="h-5 w-5 text-yellow-500" />;
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
      </div>
    );
  }

  if (!app) {
    return <div>App not found</div>;
  }

  const successCount = executions.filter(e => e.status === 'success').length;
  const failedCount = executions.filter(e => e.status === 'failed').length;
  const skippedCount = executions.filter(e => e.status === 'skipped').length;

  return (
    <div className="space-y-6">
      {/* Breadcrumb */}
      <div className="flex items-center space-x-2 text-sm text-gray-600">
        <Link to="/apps" className="hover:text-gray-900">Apps</Link>
        <span>/</span>
        <Link to={`/apps/${app.id}`} className="hover:text-gray-900">{app.name}</Link>
        <span>/</span>
        <span className="text-gray-900 font-medium">Execute All</span>
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
              Execute All Commands
            </h1>
          </div>
          <p className="text-gray-600 mt-2">
            Run all {commands.length} commands sequentially for {app.name}
          </p>
        </div>
      </div>

      {/* Controls */}
      <Card className="p-4 sm:p-6">
        <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
          <div className="flex flex-col sm:flex-row sm:items-center gap-3 sm:gap-4">
            <Button
              variant="primary"
              size="lg"
              onClick={handleExecuteAll}
              disabled={executing || commands.length === 0}
              loading={executing}
              className="w-full sm:w-auto"
            >
              <Play className="h-5 w-5 mr-2" />
              {executing ? 'Executing...' : 'Execute All'}
            </Button>

            <div className="flex items-center justify-between sm:justify-start gap-3">
              <label className="flex items-center space-x-2 text-sm text-gray-600">
                <input
                  type="checkbox"
                  checked={stopOnError}
                  onChange={(e) => setStopOnError(e.target.checked)}
                  disabled={executing}
                  className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                />
                <span>Stop on error</span>
              </label>

              {overallStatus !== 'idle' && (
                <Badge status={overallStatus === 'running' ? 'running' : overallStatus}>
                  {overallStatus === 'running' ? `${currentIndex + 1}/${commands.length}` : overallStatus}
                </Badge>
              )}
            </div>
          </div>

          <div className="flex items-center justify-between sm:justify-end gap-3 sm:gap-4">
            {overallStatus !== 'idle' && (
              <div className="flex items-center gap-2 sm:gap-3 text-sm">
                <span className="text-green-600">{successCount} passed</span>
                <span className="text-red-600">{failedCount} failed</span>
                {skippedCount > 0 && <span className="text-yellow-600">{skippedCount} skipped</span>}
              </div>
            )}
            <Button variant="ghost" onClick={handleClear} disabled={executing}>
              <Trash2 className="h-4 w-4 sm:mr-2" />
              <span className="hidden sm:inline">Clear</span>
            </Button>
          </div>
        </div>

        {/* Progress Bar */}
        {overallStatus !== 'idle' && commands.length > 0 && (
          <div className="mt-4">
            <div className="flex items-center justify-between text-sm text-gray-600 mb-2">
              <span>Progress</span>
              <span>{successCount + failedCount + skippedCount} / {commands.length} completed</span>
            </div>
            <div className="w-full h-3 bg-gray-200 rounded-full overflow-hidden">
              <div className="h-full flex">
                {/* Success portion */}
                <div
                  className="bg-green-500 transition-all duration-300"
                  style={{ width: `${(successCount / commands.length) * 100}%` }}
                />
                {/* Failed portion */}
                <div
                  className="bg-red-500 transition-all duration-300"
                  style={{ width: `${(failedCount / commands.length) * 100}%` }}
                />
                {/* Skipped portion */}
                <div
                  className="bg-yellow-400 transition-all duration-300"
                  style={{ width: `${(skippedCount / commands.length) * 100}%` }}
                />
                {/* Running indicator */}
                {executing && (
                  <div
                    className="bg-blue-500 animate-pulse transition-all duration-300"
                    style={{ width: `${(1 / commands.length) * 100}%` }}
                  />
                )}
              </div>
            </div>
          </div>
        )}
      </Card>

      {/* Command List */}
      {commands.length === 0 ? (
        <Card className="p-12 text-center text-gray-500">
          No commands to execute. Add commands to this app first.
        </Card>
      ) : (
        <div className="space-y-3">
          {executions.map((exec, index) => (
            <Card
              key={exec.command.id}
              className={`overflow-hidden ${
                index === currentIndex && executing ? 'ring-2 ring-blue-500' : ''
              }`}
            >
              {/* Command Header */}
              <div
                className={`px-6 py-4 flex items-center justify-between cursor-pointer hover:bg-gray-50 ${
                  exec.status === 'running' ? 'bg-blue-50' : ''
                }`}
              >
                <div className="flex items-center space-x-4">
                  {getStatusIcon(exec.status)}
                  <div>
                    <div className="flex items-center gap-2">
                      <span className="text-sm font-medium text-gray-500">#{index + 1}</span>
                      <h3 className="text-lg font-semibold text-gray-900">{exec.command.name}</h3>
                    </div>
                    {exec.command.description && (
                      <p className="text-sm text-gray-500">{exec.command.description}</p>
                    )}
                  </div>
                </div>
                <div className="flex items-center space-x-3">
                  {exec.exitCode !== null && (
                    <span className="text-sm text-gray-500">
                      Exit code: {exec.exitCode}
                    </span>
                  )}
                  <Badge status={exec.status === 'skipped' ? 'pending' : exec.status}>
                    {exec.status}
                  </Badge>
                </div>
              </div>

              {/* Command details and output */}
              {(exec.status !== 'pending' || index === currentIndex) && (
                <div className="border-t border-gray-200">
                  <div className="px-6 py-3 bg-gray-50">
                    <pre className="text-sm font-mono text-gray-700 overflow-x-auto">
                      {exec.command.command}
                    </pre>
                  </div>
                  {exec.output && (
                    <pre
                      ref={index === currentIndex ? outputRef : undefined}
                      className="bg-gray-900 text-green-400 p-4 font-mono text-sm overflow-auto max-h-64"
                    >
                      {exec.output}
                    </pre>
                  )}
                </div>
              )}
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}
