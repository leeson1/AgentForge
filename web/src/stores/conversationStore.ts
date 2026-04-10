import { create } from 'zustand';

/** 对话消息类型 */
export type MessageRole = 'assistant' | 'tool_call' | 'tool_result' | 'user' | 'system';

export interface ConversationMessage {
  id: string;
  timestamp: string;
  role: MessageRole;
  sessionId?: string;
  workerId?: string;
  featureId?: string;
  content: string;
  /** tool_call 专用 */
  toolName?: string;
  toolInput?: string;
  toolOutput?: string;
  isError?: boolean;
}

interface ConversationStore {
  /** taskId -> messages */
  messagesByTask: Record<string, ConversationMessage[]>;
  /** 当前选中的 Worker 过滤 (null = 全部) */
  activeWorkerFilter: string | null;

  /** 追加消息 */
  appendMessage: (taskId: string, msg: ConversationMessage) => void;
  /** 批量追加 */
  appendMessages: (taskId: string, msgs: ConversationMessage[]) => void;
  /** 清除指定 task 的消息 */
  clearMessages: (taskId: string) => void;
  /** 设置 Worker 过滤 */
  setWorkerFilter: (workerId: string | null) => void;
  /** 获取指定 task 的消息（带可选 Worker 过滤） */
  getMessages: (taskId: string) => ConversationMessage[];
  /** 获取所有活跃 Worker ID 列表 */
  getWorkerIds: (taskId: string) => string[];
}

const MAX_MESSAGES_PER_TASK = 2000;

export const useConversationStore = create<ConversationStore>((set, get) => ({
  messagesByTask: {},
  activeWorkerFilter: null,

  appendMessage: (taskId, msg) => {
    set((s) => {
      const existing = s.messagesByTask[taskId] || [];
      const updated = [...existing, msg].slice(-MAX_MESSAGES_PER_TASK);
      return {
        messagesByTask: { ...s.messagesByTask, [taskId]: updated },
      };
    });
  },

  appendMessages: (taskId, msgs) => {
    set((s) => {
      const existing = s.messagesByTask[taskId] || [];
      const updated = [...existing, ...msgs].slice(-MAX_MESSAGES_PER_TASK);
      return {
        messagesByTask: { ...s.messagesByTask, [taskId]: updated },
      };
    });
  },

  clearMessages: (taskId) => {
    set((s) => {
      const copy = { ...s.messagesByTask };
      delete copy[taskId];
      return { messagesByTask: copy };
    });
  },

  setWorkerFilter: (workerId) => set({ activeWorkerFilter: workerId }),

  getMessages: (taskId) => {
    const state = get();
    const msgs = state.messagesByTask[taskId] || [];
    if (!state.activeWorkerFilter) return msgs;
    return msgs.filter(
      (m) => !m.workerId || m.workerId === state.activeWorkerFilter
    );
  },

  getWorkerIds: (taskId) => {
    const msgs = get().messagesByTask[taskId] || [];
    const ids = new Set<string>();
    for (const m of msgs) {
      if (m.workerId) ids.add(m.workerId);
    }
    return Array.from(ids).sort();
  },
}));
