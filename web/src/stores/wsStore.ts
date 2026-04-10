import { create } from 'zustand';
import { createWebSocket } from '../lib/api';

interface WSEvent {
  id: string;
  type: string;
  task_id: string;
  session_id?: string;
  data: unknown;
  timestamp: string;
}

interface WSStore {
  connected: boolean;
  events: WSEvent[];
  ws: WebSocket | null;

  connect: (taskId?: string) => void;
  disconnect: () => void;
  clearEvents: () => void;
}

export const useWSStore = create<WSStore>((set, get) => ({
  connected: false,
  events: [],
  ws: null,

  connect: (taskId?: string) => {
    const existing = get().ws;
    if (existing) {
      existing.close();
    }

    const ws = createWebSocket(taskId);

    ws.onopen = () => set({ connected: true });

    ws.onclose = () => {
      set({ connected: false, ws: null });
      // 自动重连
      setTimeout(() => {
        if (!get().ws) {
          get().connect(taskId);
        }
      }, 3000);
    };

    ws.onmessage = (e) => {
      try {
        const event: WSEvent = JSON.parse(e.data);
        set((s) => ({
          events: [...s.events.slice(-500), event], // 保留最近 500 条
        }));
      } catch {
        // 忽略解析错误
      }
    };

    ws.onerror = () => {
      // onclose 会处理重连
    };

    set({ ws });
  },

  disconnect: () => {
    const ws = get().ws;
    if (ws) {
      ws.close();
      set({ ws: null, connected: false });
    }
  },

  clearEvents: () => set({ events: [] }),
}));
