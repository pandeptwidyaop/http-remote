import { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';
import { Package, Activity, Clock } from 'lucide-react';
import { api } from '@/api/client';
import { API_ENDPOINTS } from '@/lib/config';
import { formatDate } from '@/lib/utils';
import type { App, ExecutionWithDetails } from '@/types';
import Card from '@/components/ui/Card';
import Badge from '@/components/ui/Badge';

export default function Dashboard() {
  const [apps, setApps] = useState<App[]>([]);
  const [executions, setExecutions] = useState<ExecutionWithDetails[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchData();
  }, []);

  const fetchData = async () => {
    try {
      const [appsData, executionsData] = await Promise.all([
        api.get<App[]>(API_ENDPOINTS.apps),
        api.get<ExecutionWithDetails[]>(API_ENDPOINTS.executions),
      ]);

      setApps(appsData || []);
      setExecutions(executionsData ? executionsData.slice(0, 10) : []); // Show last 10
    } catch (error) {
      console.error('Failed to fetch dashboard data:', error);
      setApps([]);
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
      <div>
        <h1 className="text-3xl font-bold text-gray-900">Dashboard</h1>
        <p className="text-gray-600 mt-1">Overview of your deployment infrastructure</p>
      </div>

      {/* Stats Cards */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <Card className="p-6">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium text-gray-600">Total Apps</p>
              <p className="text-3xl font-bold text-gray-900 mt-2">{apps.length}</p>
            </div>
            <div className="bg-blue-100 p-3 rounded-lg">
              <Package className="h-8 w-8 text-blue-600" />
            </div>
          </div>
          <Link
            to="/apps"
            className="text-sm text-blue-600 hover:text-blue-700 font-medium mt-4 inline-block"
          >
            Manage Apps →
          </Link>
        </Card>

        <Card className="p-6">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium text-gray-600">Recent Executions</p>
              <p className="text-3xl font-bold text-gray-900 mt-2">{executions.length}</p>
            </div>
            <div className="bg-green-100 p-3 rounded-lg">
              <Activity className="h-8 w-8 text-green-600" />
            </div>
          </div>
          <Link
            to="/executions"
            className="text-sm text-green-600 hover:text-green-700 font-medium mt-4 inline-block"
          >
            View History →
          </Link>
        </Card>

        <Card className="p-6">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium text-gray-600">Active Now</p>
              <p className="text-3xl font-bold text-gray-900 mt-2">
                {executions.filter((e) => e.status === 'running').length}
              </p>
            </div>
            <div className="bg-yellow-100 p-3 rounded-lg">
              <Clock className="h-8 w-8 text-yellow-600" />
            </div>
          </div>
        </Card>
      </div>

      {/* Recent Executions Table */}
      <Card>
        <div className="px-6 py-4 border-b border-gray-200">
          <h2 className="text-lg font-semibold text-gray-900">Recent Executions</h2>
        </div>
        <div className="overflow-x-auto">
          {executions.length === 0 ? (
            <div className="text-center py-12 text-gray-500">
              <Activity className="h-12 w-12 mx-auto mb-3 text-gray-400" />
              <p>No executions yet</p>
              <Link to="/apps" className="text-blue-600 hover:text-blue-700 text-sm mt-2 inline-block">
                Create your first app to get started
              </Link>
            </div>
          ) : (
            <table className="min-w-full divide-y divide-gray-200">
              <thead className="bg-gray-50">
                <tr>
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
                    Created
                  </th>
                </tr>
              </thead>
              <tbody className="bg-white divide-y divide-gray-200">
                {executions.map((execution) => (
                  <tr key={execution.id} className="hover:bg-gray-50">
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
                      <Badge status={execution.status} />
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-600">
                      {formatDate(execution.created_at)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      </Card>
    </div>
  );
}
