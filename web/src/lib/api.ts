const BASE_URL = '/api';

export interface Task {
  id: string;
  name: string;
  description: string;
  template: string;
  status: string;
  config: TaskConfig;
  progress: TaskProgress;
  created_at: string;
  updated_at: string;
}

export interface TaskConfig {
  max_parallel_workers: number;
  session_timeout: string;
  workspace_dir: string;
}

export interface TaskProgress {
  current_batch: number;
  total_batches: number;
  features_completed: number;
  features_total: number;
  total_sessions: number;
  total_tokens: number;
  estimated_cost: number;
}

export interface Session {
  id: string;
  task_id: string;
  type: string;
  status: string;
  feature_id?: string;
  batch_num?: number;
  worker_name?: string;
  work_dir: string;
  pid?: number;
  retry_count: number;
  result: SessionResult;
  started_at: string;
  ended_at?: string;
}

export interface SessionResult {
  features_completed?: string[];
  tokens_input: number;
  tokens_output: number;
  git_commits?: string[];
  error_message?: string;
}

export interface Feature {
  id: string;
  category: string;
  description: string;
  steps: string[];
  depends_on: string[];
  batch: number | null;
  passes: boolean;
}

export interface FeatureList {
  features: Feature[];
}

export interface CreateTaskRequest {
  name: string;
  description: string;
  template: string;
  config: {
    max_parallel_workers: number;
    session_timeout: string;
    workspace_dir: string;
  };
}

export interface InterventionRequest {
  content: string;
  target_worker?: string;
}

/** WebSocket 事件 (来自后端 EventBus) */
export interface WSEvent {
  id: string;
  type: string;
  task_id: string;
  session_id?: string;
  data: Record<string, unknown>;
  timestamp: string;
}

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE_URL}${path}`, {
    headers: { 'Content-Type': 'application/json' },
    ...options,
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({ message: res.statusText }));
    throw new Error(err.message || err.error || res.statusText);
  }
  return res.json();
}

export const api = {
  // Tasks
  listTasks: (status?: string) =>
    request<Task[]>(`/tasks${status ? `?status=${status}` : ''}`),

  getTask: (id: string) =>
    request<Task>(`/tasks/${id}`),

  createTask: (data: CreateTaskRequest) =>
    request<Task>('/tasks', { method: 'POST', body: JSON.stringify(data) }),

  updateTask: (id: string, data: Partial<Task>) =>
    request<Task>(`/tasks/${id}`, { method: 'PUT', body: JSON.stringify(data) }),

  deleteTask: (id: string) =>
    request<void>(`/tasks/${id}`, { method: 'DELETE' }),

  startTask: (id: string) =>
    request<{ message: string }>(`/tasks/${id}/start`, { method: 'POST' }),

  stopTask: (id: string) =>
    request<{ message: string }>(`/tasks/${id}/stop`, { method: 'POST' }),

  // Sessions
  listSessions: (taskId: string) =>
    request<Session[]>(`/tasks/${taskId}/sessions`),

  getSession: (taskId: string, sessionId: string) =>
    request<Session>(`/tasks/${taskId}/sessions/${sessionId}`),

  // Features
  getFeatures: (taskId: string) =>
    request<FeatureList>(`/tasks/${taskId}/features`),

  // Logs
  getLogs: (taskId: string, sessionId: string, tail?: number) =>
    request<string[] | { content: string }>(`/tasks/${taskId}/logs/${sessionId}${tail ? `?tail=${tail}` : ''}`),

  // Events
  getEvents: (taskId: string) =>
    request<string[]>(`/tasks/${taskId}/events`),

  // Intervention
  sendIntervention: (taskId: string, data: InterventionRequest) =>
    request<{ message: string }>(`/tasks/${taskId}/intervene`, {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  // Health
  health: () => request<{ status: string }>('/health'),
};

// WebSocket
export function createWebSocket(taskId?: string): WebSocket {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  const host = window.location.host;
  const url = `${protocol}//${host}/api/ws${taskId ? `?task_id=${taskId}` : ''}`;
  return new WebSocket(url);
}
