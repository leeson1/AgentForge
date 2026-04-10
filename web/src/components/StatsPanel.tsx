import { Clock, Cpu, GitCommit, Layers, DollarSign, Zap } from 'lucide-react';
import { useTaskStore } from '../stores/taskStore';
import { useWSStore } from '../stores/wsStore';

export function StatsPanel() {
  const { tasks, activeTaskId } = useTaskStore();
  const { events } = useWSStore();

  const task = tasks.find((t) => t.id === activeTaskId);
  if (!task) return null;

  const p = task.progress;
  const completionPct = p.features_total > 0
    ? Math.round((p.features_completed / p.features_total) * 100)
    : 0;

  const recentEvents = events
    .filter((e) => e.task_id === activeTaskId)
    .slice(-10)
    .reverse();

  return (
    <aside className="w-64 bg-white border-l border-gray-200 flex flex-col shrink-0 overflow-y-auto">
      <div className="p-3 border-b border-gray-200">
        <span className="text-sm font-semibold text-gray-700">Statistics</span>
      </div>

      <div className="p-3 space-y-3">
        {/* Progress */}
        <div>
          <div className="flex items-center justify-between mb-1">
            <span className="text-xs text-gray-500">Progress</span>
            <span className="text-xs font-bold text-indigo-600">{completionPct}%</span>
          </div>
          <div className="w-full bg-gray-200 rounded-full h-2">
            <div
              className="bg-indigo-600 h-2 rounded-full transition-all"
              style={{ width: `${completionPct}%` }}
            />
          </div>
        </div>

        {/* Stats Grid */}
        <div className="grid grid-cols-2 gap-2">
          <StatCard icon={Layers} label="Batch" value={`${p.current_batch}/${p.total_batches}`} />
          <StatCard icon={Cpu} label="Features" value={`${p.features_completed}/${p.features_total}`} />
          <StatCard icon={Clock} label="Sessions" value={String(p.total_sessions)} />
          <StatCard icon={Zap} label="Tokens" value={formatTokens(p.total_tokens)} />
          <StatCard icon={DollarSign} label="Cost" value={`$${p.estimated_cost.toFixed(2)}`} />
          <StatCard icon={GitCommit} label="Template" value={task.template || '-'} />
        </div>

        {/* Recent Events */}
        <div>
          <span className="text-xs font-semibold text-gray-500">Recent Events</span>
          {recentEvents.length === 0 ? (
            <p className="text-xs text-gray-400 mt-1">No events</p>
          ) : (
            <div className="mt-1 space-y-1">
              {recentEvents.map((e) => (
                <div key={e.id} className="text-xs p-1.5 bg-gray-50 rounded text-gray-600">
                  <span className="font-medium">{e.type}</span>
                  <span className="text-gray-400 ml-1">
                    {new Date(e.timestamp).toLocaleTimeString()}
                  </span>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </aside>
  );
}

function StatCard({ icon: Icon, label, value }: { icon: React.ComponentType<{ className?: string }>; label: string; value: string }) {
  return (
    <div className="p-2 bg-gray-50 rounded-lg">
      <div className="flex items-center gap-1 mb-0.5">
        <Icon className="w-3 h-3 text-gray-400" />
        <span className="text-xs text-gray-500">{label}</span>
      </div>
      <span className="text-sm font-semibold text-gray-800">{value}</span>
    </div>
  );
}

function formatTokens(n: number): string {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`;
  if (n >= 1_000) return `${(n / 1_000).toFixed(1)}K`;
  return String(n);
}
