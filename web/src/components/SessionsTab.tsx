import { useEffect, useState } from 'react';
import { Clock } from 'lucide-react';
import { useTaskStore } from '../stores/taskStore';
import { api, type Session } from '../lib/api';
import { StatusBadge } from './StatusBadge';

export function SessionsTab() {
  const { activeTaskId } = useTaskStore();
  const [sessions, setSessions] = useState<Session[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (!activeTaskId) return;
    setLoading(true);
    api
      .listSessions(activeTaskId)
      .then(setSessions)
      .catch(() => setSessions([]))
      .finally(() => setLoading(false));
  }, [activeTaskId]);

  if (loading) {
    return (
      <div className="flex items-center justify-center h-full text-gray-400">
        <Clock className="w-8 h-8 animate-pulse opacity-30" />
      </div>
    );
  }

  if (sessions.length === 0) {
    return (
      <div className="flex items-center justify-center h-full text-gray-400">
        <div className="text-center">
          <Clock className="w-8 h-8 mx-auto mb-2 opacity-30" />
          <p className="text-sm">No sessions yet</p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-2">
      {sessions.map((s) => (
        <div key={s.id} className="p-3 bg-white rounded-lg border border-gray-200">
          <div className="flex items-center justify-between">
            <div className="min-w-0">
              <span className="text-sm font-medium text-gray-800">{s.id}</span>
              <span className="ml-2 text-xs text-gray-500">{s.type}</span>
              {s.worker_name && (
                <span className="ml-2 text-xs text-indigo-500 bg-indigo-50 px-1.5 py-0.5 rounded">
                  {s.worker_name}
                </span>
              )}
            </div>
            <StatusBadge status={s.status} />
          </div>
          {s.feature_id && (
            <p className="text-xs text-gray-500 mt-1">Feature: {s.feature_id}</p>
          )}
          <div className="mt-1 flex items-center gap-3 text-xs text-gray-400">
            <span>Tokens: {s.result.tokens_input + s.result.tokens_output}</span>
            {s.started_at && (
              <span>Started: {new Date(s.started_at).toLocaleTimeString()}</span>
            )}
            {s.retry_count > 0 && (
              <span className="text-orange-500">Retries: {s.retry_count}</span>
            )}
          </div>
          {s.result.error_message && (
            <p className="text-xs text-red-500 mt-1 bg-red-50 px-2 py-1 rounded">
              {s.result.error_message}
            </p>
          )}
        </div>
      ))}
    </div>
  );
}
