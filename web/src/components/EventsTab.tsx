import { useEffect, useState } from 'react';
import { List, Filter } from 'lucide-react';
import { useTaskStore } from '../stores/taskStore';
import { useWSStore } from '../stores/wsStore';
import { api, type WSEvent } from '../lib/api';

const EVENT_TYPE_COLORS: Record<string, string> = {
  task_status: 'text-blue-700 bg-blue-50',
  session_start: 'text-green-700 bg-green-50',
  session_end: 'text-gray-700 bg-gray-50',
  agent_message: 'text-purple-700 bg-purple-50',
  tool_call: 'text-amber-700 bg-amber-50',
  feature_update: 'text-cyan-700 bg-cyan-50',
  merge_conflict: 'text-red-700 bg-red-50',
  batch_update: 'text-indigo-700 bg-indigo-50',
  alert: 'text-orange-700 bg-orange-50',
  log: 'text-gray-600 bg-gray-50',
};

export function EventsTab() {
  const { activeTaskId } = useTaskStore();
  const { events: wsEvents } = useWSStore();
  const [historicEvents, setHistoricEvents] = useState<string[]>([]);
  const [typeFilter, setTypeFilter] = useState<string>('all');
  const [loading, setLoading] = useState(false);

  // 加载历史事件
  useEffect(() => {
    if (!activeTaskId) return;
    setLoading(true);
    api
      .getEvents(activeTaskId)
      .then(setHistoricEvents)
      .catch(() => setHistoricEvents([]))
      .finally(() => setLoading(false));
  }, [activeTaskId]);

  // 实时事件（来自 WebSocket）
  const realtimeEvents = wsEvents.filter(
    (e) => e.task_id === activeTaskId
  );

  // 可用的事件类型
  const availableTypes = Array.from(
    new Set(realtimeEvents.map((e) => e.type))
  ).sort();

  // 过滤
  const filteredEvents =
    typeFilter === 'all'
      ? realtimeEvents
      : realtimeEvents.filter((e) => e.type === typeFilter);

  if (loading) {
    return (
      <div className="flex items-center justify-center h-full text-gray-400">
        <List className="w-8 h-8 animate-pulse opacity-30" />
      </div>
    );
  }

  return (
    <div className="space-y-3">
      {/* 过滤器 */}
      {availableTypes.length > 0 && (
        <div className="flex items-center gap-2 flex-wrap">
          <Filter className="w-3.5 h-3.5 text-gray-400" />
          <button
            onClick={() => setTypeFilter('all')}
            className={`px-2 py-0.5 rounded-full text-xs ${
              typeFilter === 'all'
                ? 'bg-indigo-600 text-white'
                : 'bg-gray-100 text-gray-600 hover:bg-gray-200'
            }`}
          >
            All
          </button>
          {availableTypes.map((type) => (
            <button
              key={type}
              onClick={() => setTypeFilter(type)}
              className={`px-2 py-0.5 rounded-full text-xs ${
                typeFilter === type
                  ? 'bg-indigo-600 text-white'
                  : 'bg-gray-100 text-gray-600 hover:bg-gray-200'
              }`}
            >
              {type}
            </button>
          ))}
        </div>
      )}

      {/* 实时事件 */}
      {filteredEvents.length > 0 && (
        <div>
          <h3 className="text-xs font-semibold text-gray-500 mb-1.5">Real-time Events</h3>
          <div className="space-y-1">
            {[...filteredEvents].reverse().map((e) => (
              <EventCard key={e.id} event={e} />
            ))}
          </div>
        </div>
      )}

      {/* 历史事件 */}
      {historicEvents.length > 0 && (
        <div>
          <h3 className="text-xs font-semibold text-gray-500 mb-1.5">Historic Events</h3>
          <div className="space-y-1 font-mono text-xs">
            {historicEvents.map((e, i) => (
              <div
                key={i}
                className="p-2 bg-white rounded border border-gray-100 break-all text-gray-600"
              >
                {e}
              </div>
            ))}
          </div>
        </div>
      )}

      {filteredEvents.length === 0 && historicEvents.length === 0 && (
        <div className="flex items-center justify-center py-12 text-gray-400">
          <div className="text-center">
            <List className="w-8 h-8 mx-auto mb-2 opacity-30" />
            <p className="text-sm">No events yet</p>
          </div>
        </div>
      )}
    </div>
  );
}

function EventCard({ event }: { event: WSEvent }) {
  const colorClass = EVENT_TYPE_COLORS[event.type] || 'text-gray-600 bg-gray-50';

  return (
    <div className="p-2 bg-white rounded-lg border border-gray-100 text-xs">
      <div className="flex items-center gap-2">
        <span className={`px-1.5 py-0.5 rounded font-medium ${colorClass}`}>
          {event.type}
        </span>
        {event.session_id && (
          <span className="text-gray-400 truncate">
            session: {event.session_id.slice(0, 8)}...
          </span>
        )}
        <span className="text-gray-400 ml-auto shrink-0">
          {new Date(event.timestamp).toLocaleTimeString()}
        </span>
      </div>
      {event.data && Object.keys(event.data).length > 0 && (
        <pre className="mt-1 text-gray-500 overflow-x-auto whitespace-pre-wrap break-all">
          {JSON.stringify(event.data, null, 2).slice(0, 500)}
        </pre>
      )}
    </div>
  );
}
