import { useState } from 'react';
import { ChevronRight, ChevronDown, Terminal, AlertCircle, CheckCircle } from 'lucide-react';
import type { ConversationMessage } from '../stores/conversationStore';

interface Props {
  message: ConversationMessage;
}

export function ToolCallPanel({ message }: Props) {
  const [expanded, setExpanded] = useState(false);

  const isError = message.isError;
  const hasOutput = !!message.toolOutput;

  return (
    <div
      className={`rounded-lg border text-sm ${
        isError
          ? 'border-red-200 bg-red-50'
          : 'border-gray-200 bg-gray-50'
      }`}
    >
      {/* Header - clickable to expand */}
      <button
        onClick={() => setExpanded(!expanded)}
        className="w-full flex items-center gap-2 px-3 py-2 text-left hover:bg-gray-100 rounded-lg transition"
      >
        {expanded ? (
          <ChevronDown className="w-3.5 h-3.5 text-gray-400 shrink-0" />
        ) : (
          <ChevronRight className="w-3.5 h-3.5 text-gray-400 shrink-0" />
        )}
        <Terminal className="w-3.5 h-3.5 text-gray-500 shrink-0" />
        <span className="font-medium text-gray-700 truncate">
          {message.toolName || 'Tool Call'}
        </span>
        <span className="ml-auto flex items-center gap-1 shrink-0">
          {isError ? (
            <AlertCircle className="w-3.5 h-3.5 text-red-500" />
          ) : hasOutput ? (
            <CheckCircle className="w-3.5 h-3.5 text-green-500" />
          ) : null}
          <span className="text-xs text-gray-400">
            {new Date(message.timestamp).toLocaleTimeString()}
          </span>
        </span>
      </button>

      {/* Expanded content */}
      {expanded && (
        <div className="px-3 pb-3 space-y-2">
          {/* Input / Command */}
          {message.toolInput && (
            <div>
              <span className="text-xs font-medium text-gray-500 uppercase">Input</span>
              <pre className="mt-1 p-2 bg-gray-900 text-gray-100 rounded text-xs overflow-x-auto max-h-40 overflow-y-auto whitespace-pre-wrap break-all">
                {message.toolInput}
              </pre>
            </div>
          )}

          {/* Output */}
          {message.toolOutput && (
            <div>
              <span className="text-xs font-medium text-gray-500 uppercase">Output</span>
              <pre
                className={`mt-1 p-2 rounded text-xs overflow-x-auto max-h-60 overflow-y-auto whitespace-pre-wrap break-all ${
                  isError
                    ? 'bg-red-900/90 text-red-100'
                    : 'bg-gray-900 text-gray-100'
                }`}
              >
                {message.toolOutput}
              </pre>
            </div>
          )}

          {/* Content (fallback) */}
          {!message.toolInput && !message.toolOutput && message.content && (
            <pre className="p-2 bg-gray-900 text-gray-100 rounded text-xs overflow-x-auto max-h-40 overflow-y-auto whitespace-pre-wrap break-all">
              {message.content}
            </pre>
          )}
        </div>
      )}
    </div>
  );
}
