import { useState, useEffect, useCallback } from 'react';
import {
  Folder,
  File,
  ChevronRight,
  Home,
  Upload,
  Download,
  Trash2,
  Plus,
  Edit3,
  Copy,
  RefreshCw,
  ArrowLeft,
  FileText,
  Image,
  Code,
  Archive,
  Film,
  Music,
  X,
  Save,
  FolderPlus,
} from 'lucide-react';
import Editor, { loader } from '@monaco-editor/react';
import * as monaco from 'monaco-editor';
import { api } from '@/api/client';

// Configure Monaco to use local bundle instead of CDN
loader.config({ monaco });
import { getPathPrefix } from '@/lib/config';
import Button from '@/components/ui/Button';
import Card from '@/components/ui/Card';
import Modal from '@/components/ui/Modal';
import Input from '@/components/ui/Input';

// Get Monaco language from file extension
function getLanguageFromFilename(filename: string): string {
  const ext = filename.split('.').pop()?.toLowerCase() || '';
  const languageMap: Record<string, string> = {
    // JavaScript/TypeScript
    js: 'javascript',
    jsx: 'javascript',
    ts: 'typescript',
    tsx: 'typescript',
    mjs: 'javascript',
    cjs: 'javascript',
    // Web
    html: 'html',
    htm: 'html',
    css: 'css',
    scss: 'scss',
    less: 'less',
    // Data formats
    json: 'json',
    xml: 'xml',
    yaml: 'yaml',
    yml: 'yaml',
    toml: 'ini',
    // Programming languages
    py: 'python',
    go: 'go',
    rs: 'rust',
    java: 'java',
    kt: 'kotlin',
    scala: 'scala',
    c: 'c',
    cpp: 'cpp',
    cc: 'cpp',
    h: 'c',
    hpp: 'cpp',
    cs: 'csharp',
    rb: 'ruby',
    php: 'php',
    swift: 'swift',
    r: 'r',
    lua: 'lua',
    perl: 'perl',
    pl: 'perl',
    // Shell
    sh: 'shell',
    bash: 'shell',
    zsh: 'shell',
    fish: 'shell',
    ps1: 'powershell',
    bat: 'bat',
    cmd: 'bat',
    // Config files
    dockerfile: 'dockerfile',
    makefile: 'makefile',
    gitignore: 'plaintext',
    env: 'plaintext',
    // Database
    sql: 'sql',
    // Markdown
    md: 'markdown',
    markdown: 'markdown',
    // Other
    graphql: 'graphql',
    gql: 'graphql',
    vue: 'html',
    svelte: 'html',
  };
  return languageMap[ext] || 'plaintext';
}

interface FileInfo {
  name: string;
  path: string;
  is_dir: boolean;
  size: number;
  mod_time: string;
  permissions: string;
}

interface FilesResponse {
  path: string;
  parent_path: string;
  files: FileInfo[];
}

interface FileContent {
  path: string;
  name: string;
  size: number;
  is_binary: boolean;
  content: string;
}

// Helper to format file size
function formatSize(bytes: number): string {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

// Helper to format date
function formatDate(dateStr: string): string {
  const date = new Date(dateStr);
  return date.toLocaleDateString() + ' ' + date.toLocaleTimeString();
}

// Get file icon based on extension
function getFileIcon(name: string, isDir: boolean): React.ReactNode {
  if (isDir) return <Folder className="h-5 w-5 text-yellow-500" />;

  const ext = name.split('.').pop()?.toLowerCase() || '';

  const iconMap: Record<string, React.ReactNode> = {
    // Code files
    js: <Code className="h-5 w-5 text-yellow-400" />,
    ts: <Code className="h-5 w-5 text-blue-400" />,
    jsx: <Code className="h-5 w-5 text-cyan-400" />,
    tsx: <Code className="h-5 w-5 text-blue-500" />,
    py: <Code className="h-5 w-5 text-green-400" />,
    go: <Code className="h-5 w-5 text-cyan-500" />,
    rs: <Code className="h-5 w-5 text-orange-400" />,
    java: <Code className="h-5 w-5 text-red-400" />,
    cpp: <Code className="h-5 w-5 text-blue-600" />,
    c: <Code className="h-5 w-5 text-blue-500" />,
    h: <Code className="h-5 w-5 text-purple-400" />,
    css: <Code className="h-5 w-5 text-pink-400" />,
    html: <Code className="h-5 w-5 text-orange-500" />,
    json: <Code className="h-5 w-5 text-yellow-500" />,
    xml: <Code className="h-5 w-5 text-orange-400" />,
    yaml: <Code className="h-5 w-5 text-red-300" />,
    yml: <Code className="h-5 w-5 text-red-300" />,
    sh: <Code className="h-5 w-5 text-green-500" />,
    bash: <Code className="h-5 w-5 text-green-500" />,
    sql: <Code className="h-5 w-5 text-blue-300" />,
    // Images
    png: <Image className="h-5 w-5 text-purple-500" />,
    jpg: <Image className="h-5 w-5 text-purple-500" />,
    jpeg: <Image className="h-5 w-5 text-purple-500" />,
    gif: <Image className="h-5 w-5 text-purple-500" />,
    svg: <Image className="h-5 w-5 text-orange-400" />,
    webp: <Image className="h-5 w-5 text-purple-500" />,
    // Archives
    zip: <Archive className="h-5 w-5 text-yellow-600" />,
    tar: <Archive className="h-5 w-5 text-yellow-600" />,
    gz: <Archive className="h-5 w-5 text-yellow-600" />,
    rar: <Archive className="h-5 w-5 text-yellow-600" />,
    '7z': <Archive className="h-5 w-5 text-yellow-600" />,
    // Video
    mp4: <Film className="h-5 w-5 text-pink-500" />,
    mkv: <Film className="h-5 w-5 text-pink-500" />,
    avi: <Film className="h-5 w-5 text-pink-500" />,
    mov: <Film className="h-5 w-5 text-pink-500" />,
    // Audio
    mp3: <Music className="h-5 w-5 text-green-500" />,
    wav: <Music className="h-5 w-5 text-green-500" />,
    flac: <Music className="h-5 w-5 text-green-500" />,
    // Text
    txt: <FileText className="h-5 w-5 text-gray-500" />,
    md: <FileText className="h-5 w-5 text-gray-600" />,
    log: <FileText className="h-5 w-5 text-gray-400" />,
    env: <FileText className="h-5 w-5 text-green-400" />,
  };

  return iconMap[ext] || <File className="h-5 w-5 text-gray-400" />;
}

export default function Files() {
  const [currentPath, setCurrentPath] = useState('');
  const [pathInput, setPathInput] = useState('');
  const [files, setFiles] = useState<FileInfo[]>([]);
  const [parentPath, setParentPath] = useState('');
  const [loading, setLoading] = useState(true);
  const [loadingDefault, setLoadingDefault] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Modal states
  const [isUploadModalOpen, setIsUploadModalOpen] = useState(false);
  const [isNewFolderModalOpen, setIsNewFolderModalOpen] = useState(false);
  const [isDeleteModalOpen, setIsDeleteModalOpen] = useState(false);
  const [isRenameModalOpen, setIsRenameModalOpen] = useState(false);
  const [isViewModalOpen, setIsViewModalOpen] = useState(false);
  const [isEditModalOpen, setIsEditModalOpen] = useState(false);

  // Form states
  const [selectedFile, setSelectedFile] = useState<FileInfo | null>(null);
  const [newFolderName, setNewFolderName] = useState('');
  const [newName, setNewName] = useState('');
  const [fileContent, setFileContent] = useState<FileContent | null>(null);
  const [editContent, setEditContent] = useState('');
  const [uploadFile, setUploadFile] = useState<File | null>(null);
  const [submitting, setSubmitting] = useState(false);

  // Fetch default path on mount
  useEffect(() => {
    const fetchDefaultPath = async () => {
      try {
        const data = await api.get<{ default_path: string }>('/api/files/default-path');
        setCurrentPath(data.default_path);
        setPathInput(data.default_path);
        fetchFiles(data.default_path);
      } catch {
        // Fallback to root
        setCurrentPath('/');
        setPathInput('/');
        fetchFiles('/');
      } finally {
        setLoadingDefault(false);
      }
    };
    fetchDefaultPath();
  }, []);

  // Fetch files
  const fetchFiles = useCallback(async (path: string) => {
    setLoading(true);
    setError(null);
    try {
      const data = await api.get<FilesResponse>(`/api/files?path=${encodeURIComponent(path)}`);
      setFiles(data.files || []);
      setParentPath(data.parent_path || '');
      setCurrentPath(data.path);
      setPathInput(data.path);
    } catch (err: any) {
      setError(err.message || 'Failed to load files');
    } finally {
      setLoading(false);
    }
  }, []);

  // Navigate to folder
  const navigateTo = (path: string) => {
    fetchFiles(path);
  };

  // Go back
  const goBack = () => {
    if (parentPath) {
      navigateTo(parentPath);
    }
  };

  // Go home
  const goHome = () => {
    navigateTo('/');
  };

  // Refresh
  const refresh = () => {
    fetchFiles(currentPath);
  };

  // Handle file click
  const handleFileClick = (file: FileInfo) => {
    if (file.is_dir) {
      navigateTo(file.path);
    } else {
      viewFile(file);
    }
  };

  // View file
  const viewFile = async (file: FileInfo) => {
    setSelectedFile(file);
    try {
      const data = await api.get<FileContent>(`/api/files/read?path=${encodeURIComponent(file.path)}`);
      setFileContent(data);
      setIsViewModalOpen(true);
    } catch (err: any) {
      setError(err.message || 'Failed to read file');
    }
  };

  // Edit file
  const editFile = () => {
    if (fileContent) {
      setEditContent(fileContent.content);
      setIsViewModalOpen(false);
      setIsEditModalOpen(true);
    }
  };

  // Save file
  const saveFile = async () => {
    if (!selectedFile) return;
    setSubmitting(true);
    try {
      await api.post('/api/files/save', {
        path: selectedFile.path,
        content: editContent,
      });
      setIsEditModalOpen(false);
      refresh();
    } catch (err: any) {
      setError(err.message || 'Failed to save file');
    } finally {
      setSubmitting(false);
    }
  };

  // Download file
  const downloadFile = (file: FileInfo) => {
    const pathPrefix = getPathPrefix();
    window.open(`${pathPrefix}/api/files/download?path=${encodeURIComponent(file.path)}`, '_blank');
  };

  // Delete file
  const confirmDelete = (file: FileInfo) => {
    setSelectedFile(file);
    setIsDeleteModalOpen(true);
  };

  const deleteFile = async () => {
    if (!selectedFile) return;
    setSubmitting(true);
    try {
      await api.delete(`/api/files?path=${encodeURIComponent(selectedFile.path)}&recursive=true`);
      setIsDeleteModalOpen(false);
      setSelectedFile(null);
      refresh();
    } catch (err: any) {
      setError(err.message || 'Failed to delete');
    } finally {
      setSubmitting(false);
    }
  };

  // Create folder
  const createFolder = async () => {
    if (!newFolderName.trim()) return;
    setSubmitting(true);
    try {
      const newPath = currentPath === '/' ? `/${newFolderName}` : `${currentPath}/${newFolderName}`;
      await api.post('/api/files/mkdir', { path: newPath });
      setIsNewFolderModalOpen(false);
      setNewFolderName('');
      refresh();
    } catch (err: any) {
      setError(err.message || 'Failed to create folder');
    } finally {
      setSubmitting(false);
    }
  };

  // Rename file
  const openRenameModal = (file: FileInfo) => {
    setSelectedFile(file);
    setNewName(file.name);
    setIsRenameModalOpen(true);
  };

  const renameFile = async () => {
    if (!selectedFile || !newName.trim()) return;
    setSubmitting(true);
    try {
      const newPath = currentPath === '/' ? `/${newName}` : `${currentPath}/${newName}`;
      await api.post('/api/files/rename', {
        old_path: selectedFile.path,
        new_path: newPath,
      });
      setIsRenameModalOpen(false);
      setSelectedFile(null);
      setNewName('');
      refresh();
    } catch (err: any) {
      setError(err.message || 'Failed to rename');
    } finally {
      setSubmitting(false);
    }
  };

  // Upload file
  const handleUpload = async () => {
    if (!uploadFile) return;
    setSubmitting(true);
    try {
      const formData = new FormData();
      formData.append('file', uploadFile);
      formData.append('path', currentPath);
      formData.append('overwrite', 'true');

      const pathPrefix = getPathPrefix();
      const response = await fetch(`${pathPrefix}/api/files/upload`, {
        method: 'POST',
        body: formData,
        credentials: 'include',
      });

      if (!response.ok) {
        const data = await response.json();
        throw new Error(data.error || 'Upload failed');
      }

      setIsUploadModalOpen(false);
      setUploadFile(null);
      refresh();
    } catch (err: any) {
      setError(err.message || 'Failed to upload file');
    } finally {
      setSubmitting(false);
    }
  };

  // Build breadcrumb
  const breadcrumbs = currentPath.split('/').filter(Boolean);

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">File Browser</h1>
          <p className="mt-1 text-sm text-gray-600">Browse and manage files on the server</p>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="secondary" size="sm" onClick={() => setIsUploadModalOpen(true)}>
            <Upload className="h-4 w-4 mr-2" />
            Upload
          </Button>
          <Button variant="secondary" size="sm" onClick={() => setIsNewFolderModalOpen(true)}>
            <FolderPlus className="h-4 w-4 mr-2" />
            New Folder
          </Button>
        </div>
      </div>

      {/* Error message */}
      {error && (
        <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-md text-sm flex items-center justify-between">
          <span>{error}</span>
          <button onClick={() => setError(null)}>
            <X className="h-4 w-4" />
          </button>
        </div>
      )}

      {/* Navigation */}
      <Card className="p-4">
        <div className="flex flex-col gap-3">
          {/* Path input row */}
          <div className="flex items-center gap-2">
            <button
              onClick={goHome}
              className="p-2 rounded-md hover:bg-gray-100 transition-colors flex-shrink-0"
              title="Home (root)"
            >
              <Home className="h-4 w-4 text-gray-600" />
            </button>
            <button
              onClick={goBack}
              disabled={!parentPath}
              className="p-2 rounded-md hover:bg-gray-100 transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex-shrink-0"
              title="Back"
            >
              <ArrowLeft className="h-4 w-4 text-gray-600" />
            </button>
            <button
              onClick={refresh}
              className="p-2 rounded-md hover:bg-gray-100 transition-colors flex-shrink-0"
              title="Refresh"
            >
              <RefreshCw className={`h-4 w-4 text-gray-600 ${loading ? 'animate-spin' : ''}`} />
            </button>

            <div className="h-6 w-px bg-gray-300 mx-1 flex-shrink-0" />

            {/* Path input */}
            <form
              onSubmit={(e) => {
                e.preventDefault();
                if (pathInput.trim()) {
                  navigateTo(pathInput.trim());
                }
              }}
              className="flex-1 flex items-center gap-2"
            >
              <input
                type="text"
                value={pathInput}
                onChange={(e) => setPathInput(e.target.value)}
                placeholder="Enter path..."
                className="flex-1 px-3 py-1.5 text-sm border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent font-mono bg-gray-50"
              />
              <Button
                type="submit"
                variant="secondary"
                size="sm"
                disabled={!pathInput.trim() || pathInput === currentPath}
              >
                Go
              </Button>
            </form>
          </div>

          {/* Breadcrumb row */}
          <div className="flex items-center gap-1 text-sm overflow-x-auto pb-1">
            <button
              onClick={goHome}
              className="text-blue-600 hover:text-blue-800 hover:underline flex-shrink-0"
            >
              /
            </button>
            {breadcrumbs.map((crumb, index) => {
              const path = '/' + breadcrumbs.slice(0, index + 1).join('/');
              const isLast = index === breadcrumbs.length - 1;
              return (
                <span key={path} className="flex items-center gap-1 flex-shrink-0">
                  <ChevronRight className="h-4 w-4 text-gray-400" />
                  {isLast ? (
                    <span className="text-gray-900 font-medium">{crumb}</span>
                  ) : (
                    <button
                      onClick={() => navigateTo(path)}
                      className="text-blue-600 hover:text-blue-800 hover:underline"
                    >
                      {crumb}
                    </button>
                  )}
                </span>
              );
            })}
          </div>
        </div>
      </Card>

      {/* File list */}
      <Card>
        {loading ? (
          <div className="flex items-center justify-center h-64">
            <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
          </div>
        ) : files.length === 0 ? (
          <div className="flex flex-col items-center justify-center h-64 text-gray-500">
            <Folder className="h-16 w-16 mb-4 text-gray-300" />
            <p>This folder is empty</p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="bg-gray-50 border-b border-gray-200">
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Name
                  </th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Size
                  </th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Modified
                  </th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Permissions
                  </th>
                  <th className="px-4 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Actions
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-200">
                {files.map((file) => (
                  <tr
                    key={file.path}
                    className="hover:bg-gray-50 transition-colors cursor-pointer"
                    onClick={() => handleFileClick(file)}
                  >
                    <td className="px-4 py-3">
                      <div className="flex items-center gap-3">
                        {getFileIcon(file.name, file.is_dir)}
                        <span className="font-medium text-gray-900">{file.name}</span>
                      </div>
                    </td>
                    <td className="px-4 py-3 text-sm text-gray-600">
                      {file.is_dir ? '-' : formatSize(file.size)}
                    </td>
                    <td className="px-4 py-3 text-sm text-gray-600">
                      {formatDate(file.mod_time)}
                    </td>
                    <td className="px-4 py-3 text-sm text-gray-600 font-mono">
                      {file.permissions}
                    </td>
                    <td className="px-4 py-3 text-right">
                      <div className="flex items-center justify-end gap-1" onClick={(e) => e.stopPropagation()}>
                        {!file.is_dir && (
                          <button
                            onClick={() => downloadFile(file)}
                            className="p-1.5 rounded-md hover:bg-gray-200 transition-colors"
                            title="Download"
                          >
                            <Download className="h-4 w-4 text-gray-600" />
                          </button>
                        )}
                        <button
                          onClick={() => openRenameModal(file)}
                          className="p-1.5 rounded-md hover:bg-gray-200 transition-colors"
                          title="Rename"
                        >
                          <Edit3 className="h-4 w-4 text-gray-600" />
                        </button>
                        <button
                          onClick={() => confirmDelete(file)}
                          className="p-1.5 rounded-md hover:bg-red-100 transition-colors"
                          title="Delete"
                        >
                          <Trash2 className="h-4 w-4 text-red-600" />
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </Card>

      {/* Upload Modal */}
      <Modal
        isOpen={isUploadModalOpen}
        onClose={() => {
          setIsUploadModalOpen(false);
          setUploadFile(null);
        }}
        title="Upload File"
      >
        <div className="space-y-4">
          <div className="border-2 border-dashed border-gray-300 rounded-lg p-8 text-center">
            <input
              type="file"
              onChange={(e) => setUploadFile(e.target.files?.[0] || null)}
              className="hidden"
              id="file-upload"
            />
            <label
              htmlFor="file-upload"
              className="cursor-pointer flex flex-col items-center gap-2"
            >
              <Upload className="h-10 w-10 text-gray-400" />
              <span className="text-sm text-gray-600">
                {uploadFile ? uploadFile.name : 'Click to select a file'}
              </span>
              {uploadFile && (
                <span className="text-xs text-gray-500">
                  {formatSize(uploadFile.size)}
                </span>
              )}
            </label>
          </div>
          <p className="text-sm text-gray-500">
            Upload to: <code className="bg-gray-100 px-2 py-1 rounded">{currentPath}</code>
          </p>
          <div className="flex justify-end gap-3">
            <Button
              variant="secondary"
              onClick={() => {
                setIsUploadModalOpen(false);
                setUploadFile(null);
              }}
            >
              Cancel
            </Button>
            <Button
              variant="primary"
              onClick={handleUpload}
              loading={submitting}
              disabled={!uploadFile}
            >
              Upload
            </Button>
          </div>
        </div>
      </Modal>

      {/* New Folder Modal */}
      <Modal
        isOpen={isNewFolderModalOpen}
        onClose={() => {
          setIsNewFolderModalOpen(false);
          setNewFolderName('');
        }}
        title="Create New Folder"
      >
        <div className="space-y-4">
          <Input
            label="Folder Name"
            value={newFolderName}
            onChange={(e) => setNewFolderName(e.target.value)}
            placeholder="Enter folder name"
            autoFocus
          />
          <div className="flex justify-end gap-3">
            <Button
              variant="secondary"
              onClick={() => {
                setIsNewFolderModalOpen(false);
                setNewFolderName('');
              }}
            >
              Cancel
            </Button>
            <Button
              variant="primary"
              onClick={createFolder}
              loading={submitting}
              disabled={!newFolderName.trim()}
            >
              Create
            </Button>
          </div>
        </div>
      </Modal>

      {/* Delete Confirmation Modal */}
      <Modal
        isOpen={isDeleteModalOpen}
        onClose={() => {
          setIsDeleteModalOpen(false);
          setSelectedFile(null);
        }}
        title="Confirm Delete"
      >
        <div className="space-y-4">
          <p className="text-gray-600">
            Are you sure you want to delete{' '}
            <strong className="text-gray-900">{selectedFile?.name}</strong>?
            {selectedFile?.is_dir && (
              <span className="block mt-2 text-red-600 text-sm">
                This will delete all contents inside this folder.
              </span>
            )}
          </p>
          <div className="flex justify-end gap-3">
            <Button
              variant="secondary"
              onClick={() => {
                setIsDeleteModalOpen(false);
                setSelectedFile(null);
              }}
            >
              Cancel
            </Button>
            <Button variant="danger" onClick={deleteFile} loading={submitting}>
              Delete
            </Button>
          </div>
        </div>
      </Modal>

      {/* Rename Modal */}
      <Modal
        isOpen={isRenameModalOpen}
        onClose={() => {
          setIsRenameModalOpen(false);
          setSelectedFile(null);
          setNewName('');
        }}
        title="Rename"
      >
        <div className="space-y-4">
          <Input
            label="New Name"
            value={newName}
            onChange={(e) => setNewName(e.target.value)}
            placeholder="Enter new name"
            autoFocus
          />
          <div className="flex justify-end gap-3">
            <Button
              variant="secondary"
              onClick={() => {
                setIsRenameModalOpen(false);
                setSelectedFile(null);
                setNewName('');
              }}
            >
              Cancel
            </Button>
            <Button
              variant="primary"
              onClick={renameFile}
              loading={submitting}
              disabled={!newName.trim() || newName === selectedFile?.name}
            >
              Rename
            </Button>
          </div>
        </div>
      </Modal>

      {/* View File Modal */}
      <Modal
        isOpen={isViewModalOpen}
        onClose={() => {
          setIsViewModalOpen(false);
          setFileContent(null);
        }}
        title={fileContent?.name || 'View File'}
        size="xl"
      >
        {fileContent && (
          <div className="space-y-4">
            <div className="flex items-center justify-between text-sm text-gray-500">
              <span>Size: {formatSize(fileContent.size)}</span>
              {!fileContent.is_binary && (
                <Button variant="secondary" size="sm" onClick={editFile}>
                  <Edit3 className="h-4 w-4 mr-2" />
                  Edit
                </Button>
              )}
            </div>
            {fileContent.is_binary ? (
              <div className="bg-gray-100 rounded-lg p-8 text-center text-gray-500">
                <File className="h-16 w-16 mx-auto mb-4 text-gray-400" />
                <p>Binary file cannot be displayed</p>
                <Button
                  variant="primary"
                  size="sm"
                  className="mt-4"
                  onClick={() => selectedFile && downloadFile(selectedFile)}
                >
                  <Download className="h-4 w-4 mr-2" />
                  Download
                </Button>
              </div>
            ) : (
              <div className="border border-gray-200 rounded-lg overflow-hidden">
                <Editor
                  height="500px"
                  language={getLanguageFromFilename(fileContent.name)}
                  value={fileContent.content}
                  theme="vs-dark"
                  loading={<div className="flex items-center justify-center h-[500px] bg-gray-900 text-gray-400">Loading editor...</div>}
                  options={{
                    readOnly: true,
                    minimap: { enabled: false },
                    scrollBeyondLastLine: false,
                    fontSize: 13,
                    lineNumbers: 'on',
                    wordWrap: 'on',
                    automaticLayout: true,
                  }}
                />
              </div>
            )}
          </div>
        )}
      </Modal>

      {/* Edit File Modal - Full screen editor */}
      <Modal
        isOpen={isEditModalOpen}
        onClose={() => {
          setIsEditModalOpen(false);
          setEditContent('');
        }}
        title={`Edit: ${selectedFile?.name}`}
        size="full"
      >
        <div className="flex flex-col h-[calc(100vh-200px)]">
          <div className="flex-1 border border-gray-200 rounded-lg overflow-hidden">
            <Editor
              height="100%"
              language={selectedFile ? getLanguageFromFilename(selectedFile.name) : 'plaintext'}
              value={editContent}
              onChange={(value) => setEditContent(value || '')}
              theme="vs-dark"
              loading={<div className="flex items-center justify-center h-full bg-gray-900 text-gray-400">Loading editor...</div>}
              options={{
                minimap: { enabled: true },
                scrollBeyondLastLine: false,
                fontSize: 14,
                lineNumbers: 'on',
                wordWrap: 'on',
                automaticLayout: true,
                tabSize: 2,
                insertSpaces: true,
                formatOnPaste: true,
                bracketPairColorization: { enabled: true },
              }}
            />
          </div>
          <div className="flex justify-end gap-3 pt-4">
            <Button
              variant="secondary"
              onClick={() => {
                setIsEditModalOpen(false);
                setEditContent('');
              }}
            >
              Cancel
            </Button>
            <Button variant="primary" onClick={saveFile} loading={submitting}>
              <Save className="h-4 w-4 mr-2" />
              Save
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  );
}
