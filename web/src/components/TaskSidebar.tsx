import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Plus, Trash2 } from 'lucide-react';
import { useTaskStore } from '../stores/taskStore';
import { CreateTaskModal } from './CreateTaskModal';
import { StatusBadge } from './StatusBadge';

export function TaskSidebar() {
  const { tasks, activeTaskId, setActiveTask, deleteTask } = useTaskStore();
  const [showCreate, setShowCreate] = useState(false);
  const navigate = useNavigate();

  const handleSelect = (id: string) => {
    setActiveTask(id);
    navigate(`/tasks/${id}`);
  };

  return (
    <aside className="w-72 bg-white border-r border-gray-200 flex flex-col shrink-0">
      {/* Header */}
      <div className="p-3 border-b border-gray-200 flex items-center justify-between">
        <span className="text-sm font-semibold text-gray-700">Tasks</span>
        <button
          onClick={() => setShowCreate(true)}
          className="p-1.5 rounded-md bg-indigo-600 text-white hover:bg-indigo-700 transition"
        >
          <Plus className="w-4 h-4" />
        </button>
      </div>

      {/* Task List */}
      <div className="flex-1 overflow-y-auto">
        {tasks.length === 0 ? (
          <div className="p-4 text-center text-sm text-gray-400">
            No tasks yet
          </div>
        ) : (
          tasks.map((task) => (
            <div
              key={task.id}
              onClick={() => handleSelect(task.id)}
              className={`p-3 border-b border-gray-100 cursor-pointer hover:bg-gray-50 transition ${
                activeTaskId === task.id ? 'bg-indigo-50 border-l-2 border-l-indigo-600' : ''
              }`}
            >
              <div className="flex items-start justify-between">
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-medium text-gray-800 truncate">{task.name}</p>
                  <div className="mt-1 flex items-center gap-2">
                    <StatusBadge status={task.status} />
                    {task.progress.features_total > 0 && (
                      <span className="text-xs text-gray-500">
                        {task.progress.features_completed}/{task.progress.features_total}
                      </span>
                    )}
                  </div>
                  {/* Progress bar */}
                  {task.progress.features_total > 0 && (
                    <div className="mt-1.5 w-full bg-gray-200 rounded-full h-1">
                      <div
                        className="bg-indigo-600 h-1 rounded-full transition-all"
                        style={{
                          width: `${(task.progress.features_completed / task.progress.features_total) * 100}%`,
                        }}
                      />
                    </div>
                  )}
                </div>
                <button
                  onClick={(e) => {
                    e.stopPropagation();
                    if (confirm('Delete this task?')) deleteTask(task.id);
                  }}
                  className="p-1 text-gray-400 hover:text-red-500 transition ml-2"
                >
                  <Trash2 className="w-3.5 h-3.5" />
                </button>
              </div>
            </div>
          ))
        )}
      </div>

      {showCreate && <CreateTaskModal onClose={() => setShowCreate(false)} />}
    </aside>
  );
}
