import { useEffect } from 'react';
import { Outlet, useParams } from 'react-router-dom';
import { Hammer, Wifi, WifiOff } from 'lucide-react';
import { useTaskStore } from '../stores/taskStore';
import { useWSStore } from '../stores/wsStore';
import { TaskSidebar } from './TaskSidebar';
import { TaskDetail } from './TaskDetail';
import { StatsPanel } from './StatsPanel';

export function AppLayout() {
  const { taskId } = useParams();
  const { fetchTasks, activeTaskId, setActiveTask } = useTaskStore();
  const { connected, connect } = useWSStore();

  useEffect(() => {
    fetchTasks();
    connect();
  }, []);

  useEffect(() => {
    if (taskId) setActiveTask(taskId);
  }, [taskId]);

  return (
    <div className="h-screen flex flex-col bg-gray-50">
      {/* Top Nav */}
      <header className="h-12 bg-white border-b border-gray-200 flex items-center px-4 shrink-0">
        <div className="flex items-center gap-2">
          <Hammer className="w-5 h-5 text-indigo-600" />
          <span className="font-bold text-lg text-gray-800">AgentForge</span>
        </div>
        <div className="ml-auto flex items-center gap-3">
          <span className="text-xs text-gray-500">
            {connected ? (
              <span className="flex items-center gap-1 text-green-600">
                <Wifi className="w-3 h-3" /> Connected
              </span>
            ) : (
              <span className="flex items-center gap-1 text-red-500">
                <WifiOff className="w-3 h-3" /> Disconnected
              </span>
            )}
          </span>
        </div>
      </header>

      {/* Three-column layout */}
      <div className="flex flex-1 overflow-hidden">
        {/* Left: Task List */}
        <TaskSidebar />

        {/* Center: Task Detail */}
        <main className="flex-1 overflow-auto">
          {activeTaskId ? <TaskDetail /> : (
            <div className="flex items-center justify-center h-full text-gray-400">
              <div className="text-center">
                <Hammer className="w-12 h-12 mx-auto mb-3 opacity-30" />
                <p>Select a task or create a new one</p>
              </div>
            </div>
          )}
          <Outlet />
        </main>

        {/* Right: Stats Panel */}
        {activeTaskId && <StatsPanel />}
      </div>
    </div>
  );
}
