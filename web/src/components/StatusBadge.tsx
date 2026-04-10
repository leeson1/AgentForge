const statusConfig: Record<string, { color: string; label: string }> = {
  pending: { color: 'bg-gray-100 text-gray-600', label: 'Pending' },
  initializing: { color: 'bg-blue-100 text-blue-700', label: 'Initializing' },
  planning: { color: 'bg-purple-100 text-purple-700', label: 'Planning' },
  running: { color: 'bg-green-100 text-green-700', label: 'Running' },
  merging: { color: 'bg-yellow-100 text-yellow-700', label: 'Merging' },
  auto_resolving: { color: 'bg-orange-100 text-orange-700', label: 'Auto Resolving' },
  agent_resolving: { color: 'bg-orange-100 text-orange-700', label: 'Agent Resolving' },
  validating: { color: 'bg-cyan-100 text-cyan-700', label: 'Validating' },
  conflict_wait: { color: 'bg-red-100 text-red-700', label: 'Conflict Wait' },
  paused: { color: 'bg-gray-100 text-gray-600', label: 'Paused' },
  completed: { color: 'bg-emerald-100 text-emerald-700', label: 'Completed' },
  failed: { color: 'bg-red-100 text-red-700', label: 'Failed' },
  cancelled: { color: 'bg-gray-100 text-gray-500', label: 'Cancelled' },
};

export function StatusBadge({ status }: { status: string }) {
  const config = statusConfig[status] || { color: 'bg-gray-100 text-gray-600', label: status };
  return (
    <span className={`inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium ${config.color}`}>
      {config.label}
    </span>
  );
}
