import { useState, useEffect, FormEvent } from 'react';
import { useParams, Link, useNavigate } from 'react-router-dom';
import { ArrowLeft, Plus, Play, Edit2, Trash2, Copy, RefreshCw } from 'lucide-react';
import { api } from '@/api/client';
import { API_ENDPOINTS, getBaseUrl } from '@/lib/config';
import type { App, Command, CreateCommandRequest } from '@/types';
import Button from '@/components/ui/Button';
import Card from '@/components/ui/Card';
import Modal from '@/components/ui/Modal';
import Input from '@/components/ui/Input';
import Textarea from '@/components/ui/Textarea';

export default function AppDetail() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [app, setApp] = useState<App | null>(null);
  const [commands, setCommands] = useState<Command[]>([]);
  const [loading, setLoading] = useState(true);
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [formData, setFormData] = useState<CreateCommandRequest>({
    name: '',
    description: '',
    command: '',
    timeout_seconds: 300,
  });
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

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
    } catch (error: any) {
      console.error('Failed to fetch app details:', error);
      setApp(null);
      setCommands([]);
      if (error.status === 404) {
        navigate('/apps');
      }
    } finally {
      setLoading(false);
    }
  };

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    if (!id) return;

    setSubmitting(true);
    setError(null);

    try {
      await api.post(API_ENDPOINTS.appCommands(id), formData);
      setIsCreateModalOpen(false);
      setFormData({
        name: '',
        description: '',
        command: '',
        timeout_seconds: 300,
      });
      fetchData();
    } catch (error: any) {
      setError(error.message || 'Failed to create command');
    } finally {
      setSubmitting(false);
    }
  };

  const handleDeleteCommand = async (commandId: string, commandName: string) => {
    if (!window.confirm(`Are you sure you want to delete command "${commandName}"?`)) {
      return;
    }

    try {
      await api.delete(API_ENDPOINTS.command(commandId));
      fetchData();
    } catch (error: any) {
      alert(error.message || 'Failed to delete command');
    }
  };

  const handleCopyToken = () => {
    if (app?.token) {
      navigator.clipboard.writeText(app.token);
      alert('Token copied to clipboard!');
    }
  };

  const handleRegenerateToken = async () => {
    if (!id) return;
    if (!window.confirm('Are you sure you want to regenerate the token? The old token will stop working.')) {
      return;
    }

    try {
      await api.post(API_ENDPOINTS.regenerateToken(id));
      fetchData();
      alert('Token regenerated successfully!');
    } catch (error: any) {
      alert(error.message || 'Failed to regenerate token');
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

  const deployUrl = `${window.location.origin}${getBaseUrl()}/deploy/${app.id}`;

  return (
    <div className="space-y-6">
      {/* Breadcrumb */}
      <div className="flex items-center space-x-2 text-sm text-gray-600">
        <Link to="/apps" className="hover:text-gray-900">Apps</Link>
        <span>/</span>
        <span className="text-gray-900 font-medium">{app.name}</span>
      </div>

      {/* Header */}
      <div className="flex items-start justify-between">
        <div>
          <div className="flex items-center space-x-3">
            <Link to="/apps">
              <Button variant="ghost" size="sm">
                <ArrowLeft className="h-4 w-4" />
              </Button>
            </Link>
            <h1 className="text-3xl font-bold text-gray-900">{app.name}</h1>
          </div>
          {app.description && (
            <p className="text-gray-600 mt-2">{app.description}</p>
          )}
          <p className="text-sm text-gray-500 mt-1">
            Working Directory: <code className="bg-gray-100 px-2 py-1 rounded">{app.working_dir}</code>
          </p>
        </div>
        <Button variant="primary" onClick={() => setIsCreateModalOpen(true)}>
          <Plus className="h-4 w-4 mr-2" />
          New Command
        </Button>
      </div>

      {/* API Deploy Info */}
      <Card className="p-6 bg-blue-50 border-blue-200">
        <h3 className="text-lg font-semibold text-gray-900 mb-3">API Deploy</h3>
        <p className="text-sm text-gray-600 mb-3">
          Use this endpoint to trigger deployment via API:
        </p>
        <div className="bg-gray-900 text-green-400 p-4 rounded-md font-mono text-sm overflow-x-auto mb-3">
          curl -X POST {deployUrl} \<br />
          &nbsp;&nbsp;-H "X-Deploy-Token: {app.token}"
        </div>
        <div className="flex items-center space-x-2">
          <span className="text-sm font-medium text-gray-700">Token:</span>
          <code className="bg-white px-3 py-1 rounded border text-sm flex-1">
            {app.token}
          </code>
          <Button variant="secondary" size="sm" onClick={handleCopyToken}>
            <Copy className="h-4 w-4" />
          </Button>
          <Button variant="danger" size="sm" onClick={handleRegenerateToken}>
            <RefreshCw className="h-4 w-4" />
          </Button>
        </div>
      </Card>

      {/* Commands */}
      <div>
        <h2 className="text-2xl font-bold text-gray-900 mb-4">Commands</h2>
        {commands.length === 0 ? (
          <Card className="p-12">
            <div className="text-center text-gray-500">
              <Play className="h-16 w-16 mx-auto mb-4 text-gray-400" />
              <h3 className="text-lg font-medium text-gray-900 mb-2">No commands yet</h3>
              <p className="mb-4">Create your first command to start deploying</p>
              <Button variant="primary" onClick={() => setIsCreateModalOpen(true)}>
                <Plus className="h-4 w-4 mr-2" />
                Create Command
              </Button>
            </div>
          </Card>
        ) : (
          <div className="space-y-4">
            {commands.map((command) => (
              <Card key={command.id} className="p-6">
                <div className="flex items-start justify-between">
                  <div className="flex-1">
                    <h3 className="text-lg font-semibold text-gray-900">{command.name}</h3>
                    {command.description && (
                      <p className="text-gray-600 text-sm mt-1">{command.description}</p>
                    )}
                    <pre className="bg-gray-900 text-green-400 p-3 rounded-md mt-3 text-sm overflow-x-auto font-mono">
                      {command.command}
                    </pre>
                    <p className="text-sm text-gray-500 mt-2">
                      Timeout: {command.timeout_seconds}s
                    </p>
                  </div>
                  <div className="flex items-center space-x-2 ml-4">
                    <Link to={`/execute/${command.id}`}>
                      <Button variant="primary" size="sm">
                        <Play className="h-4 w-4 mr-1" />
                        Execute
                      </Button>
                    </Link>
                    <Button
                      variant="danger"
                      size="sm"
                      onClick={() => handleDeleteCommand(command.id, command.name)}
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </div>
                </div>
              </Card>
            ))}
          </div>
        )}
      </div>

      {/* Create Command Modal */}
      <Modal
        isOpen={isCreateModalOpen}
        onClose={() => {
          setIsCreateModalOpen(false);
          setError(null);
        }}
        title="Create New Command"
        size="lg"
      >
        <form onSubmit={handleSubmit} className="space-y-4">
          {error && (
            <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-md text-sm">
              {error}
            </div>
          )}

          <Input
            label="Name"
            required
            value={formData.name}
            onChange={(e) => setFormData({ ...formData, name: e.target.value })}
            placeholder="e.g., deploy"
          />

          <Input
            label="Description"
            value={formData.description}
            onChange={(e) => setFormData({ ...formData, description: e.target.value })}
            placeholder="What does this command do?"
          />

          <Textarea
            label="Command"
            required
            rows={4}
            value={formData.command}
            onChange={(e) => setFormData({ ...formData, command: e.target.value })}
            placeholder="git pull && docker-compose up -d --build"
          />

          <Input
            label="Timeout (seconds)"
            type="number"
            required
            min={1}
            max={3600}
            value={formData.timeout_seconds}
            onChange={(e) => setFormData({ ...formData, timeout_seconds: parseInt(e.target.value) })}
          />

          <div className="flex items-center justify-end space-x-3 pt-4">
            <Button
              type="button"
              variant="secondary"
              onClick={() => setIsCreateModalOpen(false)}
            >
              Cancel
            </Button>
            <Button type="submit" variant="primary" loading={submitting}>
              Create Command
            </Button>
          </div>
        </form>
      </Modal>
    </div>
  );
}
