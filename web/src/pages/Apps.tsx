import { useState, useEffect, FormEvent } from 'react';
import { Link } from 'react-router-dom';
import { Plus, Trash2, FolderOpen, Terminal } from 'lucide-react';
import { api } from '@/api/client';
import { API_ENDPOINTS } from '@/lib/config';
import type { App, CreateAppRequest } from '@/types';
import Button from '@/components/ui/Button';
import Card from '@/components/ui/Card';
import Modal from '@/components/ui/Modal';
import Input from '@/components/ui/Input';
import ConfirmDialog from '@/components/ui/ConfirmDialog';
import { toast } from '@/store/toastStore';

export default function Apps() {
  const [apps, setApps] = useState<App[]>([]);
  const [loading, setLoading] = useState(true);
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [formData, setFormData] = useState<CreateAppRequest>({
    name: '',
    description: '',
    working_dir: '',
  });
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<{ id: string; name: string } | null>(null);
  const [deleting, setDeleting] = useState(false);

  useEffect(() => {
    fetchApps();
  }, []);

  const fetchApps = async () => {
    try {
      const data = await api.get<App[]>(API_ENDPOINTS.apps);
      setApps(data || []);
    } catch (error) {
      console.error('Failed to fetch apps:', error);
      setApps([]);
    } finally {
      setLoading(false);
    }
  };

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setSubmitting(true);
    setError(null);

    try {
      await api.post(API_ENDPOINTS.apps, formData);
      setIsCreateModalOpen(false);
      setFormData({ name: '', description: '', working_dir: '' });
      toast.success('App created', `${formData.name} has been created successfully`);
      fetchApps();
    } catch (error: any) {
      setError(error.message || 'Failed to create app');
    } finally {
      setSubmitting(false);
    }
  };

  const handleDelete = async () => {
    if (!deleteTarget) return;

    setDeleting(true);
    try {
      await api.delete(API_ENDPOINTS.app(deleteTarget.id));
      toast.success('App deleted', `${deleteTarget.name} has been deleted`);
      setDeleteTarget(null);
      fetchApps();
    } catch (error: any) {
      toast.error('Delete failed', error.message || 'Failed to delete app');
    } finally {
      setDeleting(false);
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
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-gray-900">Apps</h1>
          <p className="text-gray-600 mt-1">Manage your deployment applications</p>
        </div>
        <Button
          variant="primary"
          onClick={() => setIsCreateModalOpen(true)}
        >
          <Plus className="h-4 w-4 mr-2" />
          New App
        </Button>
      </div>

      {/* Apps Grid */}
      {apps.length === 0 ? (
        <Card className="p-12">
          <div className="text-center text-gray-500">
            <FolderOpen className="h-16 w-16 mx-auto mb-4 text-gray-400" />
            <h3 className="text-lg font-medium text-gray-900 mb-2">No apps yet</h3>
            <p className="mb-4">Create your first app to get started with deployments</p>
            <Button
              variant="primary"
              onClick={() => setIsCreateModalOpen(true)}
            >
              <Plus className="h-4 w-4 mr-2" />
              Create App
            </Button>
          </div>
        </Card>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {apps.map((app) => (
            <Card key={app.id} className="p-6 hover:shadow-lg transition-shadow">
              <div className="space-y-4">
                <div>
                  <Link
                    to={`/apps/${app.id}`}
                    className="text-xl font-semibold text-gray-900 hover:text-blue-600"
                  >
                    {app.name}
                  </Link>
                  {app.description && (
                    <p className="text-gray-600 text-sm mt-1">{app.description}</p>
                  )}
                </div>

                <div className="space-y-2">
                  <div className="flex items-center text-sm text-gray-500">
                    <FolderOpen className="h-4 w-4 mr-2 flex-shrink-0" />
                    <code className="bg-gray-100 px-2 py-1 rounded text-xs truncate">
                      {app.working_dir}
                    </code>
                  </div>
                  <div className="flex items-center text-sm text-gray-500">
                    <Terminal className="h-4 w-4 mr-2 flex-shrink-0" />
                    <span>{app.command_count || 0} command{(app.command_count || 0) !== 1 ? 's' : ''}</span>
                  </div>
                </div>

                <div className="flex items-center space-x-2 pt-4 border-t border-gray-200">
                  <Link to={`/apps/${app.id}`} className="flex-1">
                    <Button variant="secondary" size="sm" className="w-full">
                      View Commands
                    </Button>
                  </Link>
                  <Button
                    variant="danger"
                    size="sm"
                    onClick={() => setDeleteTarget({ id: app.id, name: app.name })}
                  >
                    <Trash2 className="h-4 w-4" />
                  </Button>
                </div>
              </div>
            </Card>
          ))}
        </div>
      )}

      {/* Create Modal */}
      <Modal
        isOpen={isCreateModalOpen}
        onClose={() => {
          setIsCreateModalOpen(false);
          setError(null);
        }}
        title="Create New App"
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
            placeholder="e.g., my-webapp"
          />

          <Input
            label="Description"
            value={formData.description}
            onChange={(e) => setFormData({ ...formData, description: e.target.value })}
            placeholder="What is this app for?"
          />

          <Input
            label="Working Directory"
            required
            value={formData.working_dir}
            onChange={(e) => setFormData({ ...formData, working_dir: e.target.value })}
            placeholder="/path/to/app"
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
              Create App
            </Button>
          </div>
        </form>
      </Modal>

      {/* Delete Confirmation */}
      <ConfirmDialog
        isOpen={!!deleteTarget}
        onClose={() => setDeleteTarget(null)}
        onConfirm={handleDelete}
        title="Delete App"
        message={`Are you sure you want to delete "${deleteTarget?.name}"? This will also delete all associated commands.`}
        confirmText="Delete"
        variant="danger"
        loading={deleting}
      />
    </div>
  );
}
