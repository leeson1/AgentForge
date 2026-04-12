import { useEffect, useState } from 'react';
import { X } from 'lucide-react';
import { useTaskStore } from '../stores/taskStore';
import { api, type TemplateInfo } from '../lib/api';

interface Props {
  onClose: () => void;
}

export function CreateTaskModal({ onClose }: Props) {
  const { createTask, setActiveTask } = useTaskStore();
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [template, setTemplate] = useState('default');
  const [templates, setTemplates] = useState<TemplateInfo[]>([]);
  const [workspaceDir, setWorkspaceDir] = useState('');
  const [maxWorkers, setMaxWorkers] = useState(2);
  const [timeout, setTimeout] = useState('30m');
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    api.listTemplates().then(setTemplates).catch(() => setTemplates([]));
  }, []);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim() || !workspaceDir.trim()) {
      setError('Name and Workspace Dir are required');
      return;
    }

    setSubmitting(true);
    setError('');
    try {
      const task = await createTask({
        name: name.trim(),
        description: description.trim(),
        template,
        config: {
          max_parallel_workers: maxWorkers,
          session_timeout: timeout,
          workspace_dir: workspaceDir.trim(),
        },
      });
      setActiveTask(task.id);
      onClose();
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-white rounded-xl shadow-xl w-full max-w-md mx-4">
        <div className="flex items-center justify-between p-4 border-b border-gray-200">
          <h3 className="text-lg font-semibold text-gray-800">New Task</h3>
          <button onClick={onClose} className="text-gray-400 hover:text-gray-600">
            <X className="w-5 h-5" />
          </button>
        </div>

        <form onSubmit={handleSubmit} className="p-4 space-y-4">
          {error && (
            <div className="p-2 bg-red-50 text-red-700 text-sm rounded">{error}</div>
          )}

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Name *</label>
            <input
              value={name}
              onChange={(e) => setName(e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded-md text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
              placeholder="My Project"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Description</label>
            <textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              rows={3}
              className="w-full px-3 py-2 border border-gray-300 rounded-md text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
              placeholder="Describe what you want to build..."
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Workspace Dir *</label>
            <input
              value={workspaceDir}
              onChange={(e) => setWorkspaceDir(e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded-md text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
              placeholder="/path/to/project"
            />
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Template</label>
              <select
                value={template}
                onChange={(e) => setTemplate(e.target.value)}
                className="w-full px-3 py-2 border border-gray-300 rounded-md text-sm"
              >
                {(templates.length > 0 ? templates : [{ id: 'default', name: 'Default', description: '', category: 'general' }]).map((tmpl) => (
                  <option key={tmpl.id} value={tmpl.id}>
                    {tmpl.name}
                  </option>
                ))}
              </select>
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Max Workers</label>
              <input
                type="number"
                min={1}
                max={10}
                value={maxWorkers}
                onChange={(e) => setMaxWorkers(Number(e.target.value))}
                className="w-full px-3 py-2 border border-gray-300 rounded-md text-sm"
              />
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Session Timeout</label>
            <input
              value={timeout}
              onChange={(e) => setTimeout(e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded-md text-sm"
              placeholder="30m"
            />
          </div>

          <div className="flex justify-end gap-2 pt-2">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 text-sm text-gray-600 hover:text-gray-800 transition"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={submitting}
              className="px-4 py-2 bg-indigo-600 text-white rounded-md text-sm hover:bg-indigo-700 transition disabled:opacity-50"
            >
              {submitting ? 'Creating...' : 'Create Task'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
