import { useEffect, useState } from 'react';
import { Play, Square, Clock, Layers } from 'lucide-react';
import { useTaskStore } from '../stores/taskStore';
import { api, type Session, type FeatureList } from '../lib/api';
import { StatusBadge } from './StatusBadge';

type Tab = 'sessions' | 'features' | 'events';

export function TaskDetail() {
  const { tasks, activeTaskId, startTask, stopTask } = useTaskStore();
  const [tab, setTab] = useState<Tab>('sessions');
  const [sessions, setSessions] = useState<Session[]>([]);
  const [features, setFeatures] = useState<FeatureList | null>(null);
  const [events, setEvents] = useState<string[]>([]);

  const task = tasks.find((t) => t.id === activeTaskId);

  useEffect(() => {
    if (!activeTaskId) return;
    if (tab === 'sessions') {
      api.listSessions(activeTaskId).then(setSessions).catch(() => setSessions([]));
    } else if (tab === 'features') {
      api.getFeatures(activeTaskId).then(setFeatures).catch(() => setFeatures(null));
    } else if (tab === 'events') {
      api.getEvents(activeTaskId).then(setEvents).catch(() => setEvents([]));
    }
  }, [activeTaskId, tab]);

  if (!task) return null;

  const isActive = ['initializing', 'planning', 'running', 'merging', 'auto_resolving', 'agent_resolving', 'validating'].includes(task.status);
  const canStart = task.status === 'pending' || task.status === 'failed';

  return (
    <div className="h-full flex flex-col">
      {/* Task Header */}
      <div className="p-4 border-b border-gray-200 bg-white">
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-lg font-semibold text-gray-800">{task.name}</h2>
            <p className="text-sm text-gray-500 mt-0.5">{task.description || 'No description'}</p>
          </div>
          <div className="flex items-center gap-2">
            <StatusBadge status={task.status} />
            {canStart && (
              <button
                onClick={() => startTask(task.id)}
                className="flex items-center gap-1 px-3 py-1.5 bg-green-600 text-white rounded-md text-sm hover:bg-green-700 transition"
              >
                <Play className="w-3.5 h-3.5" /> Start
              </button>
            )}
            {isActive && (
              <button
                onClick={() => stopTask(task.id)}
                className="flex items-center gap-1 px-3 py-1.5 bg-red-600 text-white rounded-md text-sm hover:bg-red-700 transition"
              >
                <Square className="w-3.5 h-3.5" /> Stop
              </button>
            )}
          </div>
        </div>
      </div>

      {/* Tabs */}
      <div className="flex border-b border-gray-200 bg-white px-4">
        {([
          { key: 'sessions' as Tab, label: 'Sessions', icon: Clock },
          { key: 'features' as Tab, label: 'Features', icon: Layers },
          { key: 'events' as Tab, label: 'Events', icon: Layers },
        ]).map(({ key, label, icon: Icon }) => (
          <button
            key={key}
            onClick={() => setTab(key)}
            className={`flex items-center gap-1.5 px-3 py-2 text-sm border-b-2 transition ${
              tab === key
                ? 'border-indigo-600 text-indigo-600 font-medium'
                : 'border-transparent text-gray-500 hover:text-gray-700'
            }`}
          >
            <Icon className="w-3.5 h-3.5" />
            {label}
          </button>
        ))}
      </div>

      {/* Tab Content */}
      <div className="flex-1 overflow-auto p-4">
        {tab === 'sessions' && <SessionsList sessions={sessions} />}
        {tab === 'features' && <FeaturesList features={features} />}
        {tab === 'events' && <EventsList events={events} />}
      </div>
    </div>
  );
}

function SessionsList({ sessions }: { sessions: Session[] }) {
  if (sessions.length === 0) {
    return <p className="text-sm text-gray-400 text-center py-8">No sessions yet</p>;
  }
  return (
    <div className="space-y-2">
      {sessions.map((s) => (
        <div key={s.id} className="p-3 bg-white rounded-lg border border-gray-200">
          <div className="flex items-center justify-between">
            <div>
              <span className="text-sm font-medium text-gray-800">{s.id}</span>
              <span className="ml-2 text-xs text-gray-500">{s.type}</span>
            </div>
            <StatusBadge status={s.status} />
          </div>
          {s.feature_id && (
            <p className="text-xs text-gray-500 mt-1">Feature: {s.feature_id}</p>
          )}
          <div className="mt-1 text-xs text-gray-400">
            Tokens: {s.result.tokens_input + s.result.tokens_output}
          </div>
        </div>
      ))}
    </div>
  );
}

function FeaturesList({ features }: { features: FeatureList | null }) {
  if (!features || features.features.length === 0) {
    return <p className="text-sm text-gray-400 text-center py-8">No features yet</p>;
  }

  // Group by batch
  const batches = new Map<number | 'unassigned', typeof features.features>();
  for (const f of features.features) {
    const key = f.batch ?? 'unassigned';
    if (!batches.has(key)) batches.set(key, []);
    batches.get(key)!.push(f);
  }

  return (
    <div className="space-y-4">
      {Array.from(batches.entries()).map(([batch, items]) => (
        <div key={String(batch)}>
          <h3 className="text-xs font-semibold text-gray-500 uppercase mb-2">
            {batch === 'unassigned' ? 'Unassigned' : `Batch ${batch}`}
          </h3>
          <div className="space-y-1.5">
            {items.map((f) => (
              <div
                key={f.id}
                className={`p-2.5 rounded-lg border ${
                  f.passes
                    ? 'bg-green-50 border-green-200'
                    : 'bg-white border-gray-200'
                }`}
              >
                <div className="flex items-center gap-2">
                  <span className="text-sm">
                    {f.passes ? '✅' : '⬜'}
                  </span>
                  <span className="text-sm font-medium text-gray-800">{f.id}</span>
                  <span className="text-xs text-gray-500">{f.category}</span>
                </div>
                <p className="text-xs text-gray-600 mt-1 ml-6">{f.description}</p>
                {f.depends_on.length > 0 && (
                  <p className="text-xs text-gray-400 mt-0.5 ml-6">
                    Depends on: {f.depends_on.join(', ')}
                  </p>
                )}
              </div>
            ))}
          </div>
        </div>
      ))}
    </div>
  );
}

function EventsList({ events }: { events: string[] }) {
  if (events.length === 0) {
    return <p className="text-sm text-gray-400 text-center py-8">No events yet</p>;
  }
  return (
    <div className="space-y-1 font-mono text-xs">
      {events.map((e, i) => (
        <div key={i} className="p-2 bg-white rounded border border-gray-100 break-all text-gray-600">
          {e}
        </div>
      ))}
    </div>
  );
}
