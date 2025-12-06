import { useState, useEffect } from 'react';
import { History } from 'lucide-react';
import { api } from '@/api/client';
import { API_ENDPOINTS } from '@/lib/config';
import { formatDate, formatDuration } from '@/lib/utils';
import type { ExecutionWithDetails } from '@/types';
import Card from '@/components/ui/Card';
import Badge from '@/components/ui/Badge';

export default function Executions() {
  const [executions, setExecutions] = useState<ExecutionWithDetails[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchExecutions();
  }, []);

  const fetchExecutions = async () => {
    try {
      const data = await api.get<ExecutionWithDetails[]>(API_ENDPOINTS.executions);
      setExecutions(data || []);
    } catch (error) {
      console.error('Failed to fetch executions:', error);
      setExecutions([]);
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-3xl font-bold text-gray-900">Execution History</h1>
        <p className="text-gray-600 mt-1">View all command execution history</p>
      </div>

      {/* Executions Table */}
      <Card>
        {executions.length === 0 ? (
          <div className="text-center py-12 text-gray-500">
            <History className="h-16 w-16 mx-auto mb-4 text-gray-400" />
            <h3 className="text-lg font-medium text-gray-900 mb-2">No executions yet</h3>
            <p>Execution history will appear here once you run commands</p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-gray-200">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    ID
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    App
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Command
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    User
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Status
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Duration
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Created
                  </th>
                </tr>
              </thead>
              <tbody className="bg-white divide-y divide-gray-200">
                {executions.map((execution) => (
                  <tr key={execution.id} className="hover:bg-gray-50">
                    <td className="px-6 py-4 whitespace-nowrap text-sm font-mono text-gray-600">
                      {execution.id.substring(0, 8)}...
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">
                      {execution.app_name}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-600">
                      {execution.command_name}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-600">
                      {execution.username}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <Badge status={execution.status}>
                        {execution.status}
                        {execution.exit_code !== undefined && execution.exit_code !== null &&
                          ` (${execution.exit_code})`
                        }
                      </Badge>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-600">
                      {execution.started_at && execution.finished_at
                        ? formatDuration(execution.started_at, execution.finished_at)
                        : execution.started_at
                        ? formatDuration(execution.started_at)
                        : '-'}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-600">
                      {formatDate(execution.created_at)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </Card>
    </div>
  );
}
