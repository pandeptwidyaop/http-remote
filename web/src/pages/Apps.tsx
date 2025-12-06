import { useState, useEffect, FormEvent } from 'react';
import { Link } from 'react-router-dom';
import { Plus, Trash2, FolderOpen } from 'lucide-react';
import { api } from '@/api/client';
import { API_ENDPOINTS } from '@/lib/config';
import type { App, CreateAppRequest } from '@/types';
import Button from '@/components/ui/Button';
import Card from '@/components/ui/Card';
import Modal from '@/components/ui/Modal';
import Input from '@/components/ui/Input';

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
      fetchApps();
    } catch (error: any) {
      setError(error.message || 'Failed to create app');
    } finally {
      setSubmitting(false);
    }
  };

  const handleDelete = async (id: string, name: string) => {
    if (!window.confirm(`Are you sure you want to delete "${name}"? This will also delete all associated commands.`)) {
      return;
    }

    try {
      await api.delete(API_ENDPOINTS.app(id));
      fetchApps();
    } catch (error: any) {
      alert(error.message || 'Failed to delete app');
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

                <div className="flex items-center text-sm text-gray-500">
                  <FolderOpen className="h-4 w-4 mr-2" />
                  <code className="bg-gray-100 px-2 py-1 rounded text-xs">
                    {app.working_dir}
                  </code>
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
                    onClick={() => handleDelete(app.id, app.name)}
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
    </div>
  );
}
