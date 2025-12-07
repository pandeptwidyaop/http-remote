import { useState, useEffect, FormEvent } from 'react';
import { useParams, Link, useNavigate } from 'react-router-dom';
import { ArrowLeft, Plus, Play, Trash2, Copy, RefreshCw, GripVertical, PlayCircle } from 'lucide-react';
import { api } from '@/api/client';
import { API_ENDPOINTS, getBaseUrl } from '@/lib/config';
import type { App, Command, CreateCommandRequest } from '@/types';
import Button from '@/components/ui/Button';
import Card from '@/components/ui/Card';
import Modal from '@/components/ui/Modal';
import Input from '@/components/ui/Input';
import Textarea from '@/components/ui/Textarea';
import ConfirmDialog from '@/components/ui/ConfirmDialog';
import { toast } from '@/store/toastStore';

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

  // Drag and drop state
  const [draggedIndex, setDraggedIndex] = useState<number | null>(null);
  const [dragOverIndex, setDragOverIndex] = useState<number | null>(null);
  const [isReordering, setIsReordering] = useState(false);

  // Confirm dialogs state
  const [deleteCommandTarget, setDeleteCommandTarget] = useState<{ id: string; name: string } | null>(null);
  const [deletingCommand, setDeletingCommand] = useState(false);
  const [showRegenerateConfirm, setShowRegenerateConfirm] = useState(false);
  const [regenerating, setRegenerating] = useState(false);

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
      toast.success('Command created', `${formData.name} has been created`);
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

  const handleDeleteCommand = async () => {
    if (!deleteCommandTarget) return;

    setDeletingCommand(true);
    try {
      await api.delete(API_ENDPOINTS.command(deleteCommandTarget.id));
      toast.success('Command deleted', `${deleteCommandTarget.name} has been deleted`);
      setDeleteCommandTarget(null);
      fetchData();
    } catch (error: any) {
      toast.error('Delete failed', error.message || 'Failed to delete command');
    } finally {
      setDeletingCommand(false);
    }
  };

  const handleCopyToken = () => {
    if (app?.token) {
      navigator.clipboard.writeText(app.token);
      toast.success('Copied!', 'Token copied to clipboard');
    }
  };

  const handleRegenerateToken = async () => {
    if (!id) return;

    setRegenerating(true);
    try {
      await api.post(API_ENDPOINTS.regenerateToken(id));
      fetchData();
      setShowRegenerateConfirm(false);
      toast.success('Token regenerated', 'The new token is now active');
    } catch (error: any) {
      toast.error('Failed', error.message || 'Failed to regenerate token');
    } finally {
      setRegenerating(false);
    }
  };

  // Drag and drop handlers
  const handleDragStart = (index: number) => {
    setDraggedIndex(index);
  };

  const handleDragOver = (e: React.DragEvent, index: number) => {
    e.preventDefault();
    if (draggedIndex !== null && draggedIndex !== index) {
      setDragOverIndex(index);
    }
  };

  const handleDragLeave = () => {
    setDragOverIndex(null);
  };

  const handleDrop = async (e: React.DragEvent, dropIndex: number) => {
    e.preventDefault();
    if (draggedIndex === null || draggedIndex === dropIndex || !id) {
      setDraggedIndex(null);
      setDragOverIndex(null);
      return;
    }

    // Reorder locally first for immediate feedback
    const newCommands = [...commands];
    const [draggedItem] = newCommands.splice(draggedIndex, 1);
    newCommands.splice(dropIndex, 0, draggedItem);
    setCommands(newCommands);

    // Reset drag state
    setDraggedIndex(null);
    setDragOverIndex(null);

    // Send reorder request to server
    setIsReordering(true);
    try {
      const commandIds = newCommands.map(cmd => cmd.id);
      await api.post(API_ENDPOINTS.reorderCommands(id), { command_ids: commandIds });
      toast.success('Reordered', 'Commands have been reordered');
    } catch (error: any) {
      console.error('Failed to reorder commands:', error);
      // Revert on error
      fetchData();
      toast.error('Reorder failed', error.message || 'Failed to reorder commands');
    } finally {
      setIsReordering(false);
    }
  };

  const handleDragEnd = () => {
    setDraggedIndex(null);
    setDragOverIndex(null);
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
          <Button variant="danger" size="sm" onClick={() => setShowRegenerateConfirm(true)}>
            <RefreshCw className="h-4 w-4" />
          </Button>
        </div>
      </Card>

      {/* Commands */}
      <div>
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center gap-4">
            <h2 className="text-2xl font-bold text-gray-900">Commands</h2>
            {commands.length > 1 && (
              <Link to={`/apps/${app.id}/execute-all`}>
                <Button variant="secondary" size="sm">
                  <PlayCircle className="h-4 w-4 mr-2" />
                  Execute All
                </Button>
              </Link>
            )}
          </div>
          {commands.length > 1 && (
            <p className="text-sm text-gray-500">
              Drag to reorder • First command is default for API deploy
            </p>
          )}
        </div>
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
          <div className="space-y-2">
            {commands.map((command, index) => (
              <Card
                key={command.id}
                className={`p-4 transition-all duration-200 ${
                  draggedIndex === index ? 'opacity-50 scale-[0.98]' : ''
                } ${
                  dragOverIndex === index ? 'border-blue-500 border-2 bg-blue-50' : ''
                } ${isReordering ? 'pointer-events-none' : ''}`}
                draggable
                onDragStart={() => handleDragStart(index)}
                onDragOver={(e) => handleDragOver(e, index)}
                onDragLeave={handleDragLeave}
                onDrop={(e) => handleDrop(e, index)}
                onDragEnd={handleDragEnd}
              >
                <div className="flex items-start">
                  {/* Drag Handle */}
                  <div className="flex items-center mr-3 cursor-grab active:cursor-grabbing text-gray-400 hover:text-gray-600">
                    <GripVertical className="h-5 w-5" />
                    <span className="text-xs font-medium ml-1 w-4">{index + 1}</span>
                  </div>

                  {/* Command Content */}
                  <div className="flex-1 min-w-0">
                    <div className="flex items-start justify-between">
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2">
                          <h3 className="text-lg font-semibold text-gray-900">{command.name}</h3>
                          {index === 0 && (
                            <span className="px-2 py-0.5 text-xs font-medium bg-blue-100 text-blue-800 rounded">
                              Default
                            </span>
                          )}
                        </div>
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
                      <div className="flex items-center space-x-2 ml-4 flex-shrink-0">
                        <Link to={`/execute/${command.id}`}>
                          <Button variant="primary" size="sm">
                            <Play className="h-4 w-4 mr-1" />
                            Execute
                          </Button>
                        </Link>
                        <Button
                          variant="danger"
                          size="sm"
                          onClick={() => setDeleteCommandTarget({ id: command.id, name: command.name })}
                        >
                          <Trash2 className="h-4 w-4" />
                        </Button>
                      </div>
                    </div>
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

      {/* Delete Command Confirmation */}
      <ConfirmDialog
        isOpen={!!deleteCommandTarget}
        onClose={() => setDeleteCommandTarget(null)}
        onConfirm={handleDeleteCommand}
        title="Delete Command"
        message={`Are you sure you want to delete "${deleteCommandTarget?.name}"?`}
        confirmText="Delete"
        variant="danger"
        loading={deletingCommand}
      />

      {/* Regenerate Token Confirmation */}
      <ConfirmDialog
        isOpen={showRegenerateConfirm}
        onClose={() => setShowRegenerateConfirm(false)}
        onConfirm={handleRegenerateToken}
        title="Regenerate Token"
        message="Are you sure you want to regenerate the token? The old token will stop working immediately."
        confirmText="Regenerate"
        variant="warning"
        loading={regenerating}
      />
    </div>
  );
}
