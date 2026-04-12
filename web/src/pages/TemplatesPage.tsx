import { useEffect, useState } from 'react';
import {
  Layout, Globe, Database, Terminal,
  ChevronRight, ChevronDown, FileText,
} from 'lucide-react';
import { api, type TemplateInfo } from '../lib/api';

const CATEGORY_ICONS: Record<string, React.ComponentType<{ className?: string }>> = {
  general: Layout,
  web: Globe,
  cli: Terminal,
  data: Database,
};

export function TemplatesPage() {
  const [expandedId, setExpandedId] = useState<string | null>(null);
  const [templates, setTemplates] = useState<TemplateInfo[]>([]);

  useEffect(() => {
    api.listTemplates().then(setTemplates).catch(() => setTemplates([]));
  }, []);

  const toggleExpand = (id: string) => {
    setExpandedId(expandedId === id ? null : id);
  };

  return (
    <div className="max-w-3xl mx-auto p-6 space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-gray-800">Templates</h1>
        <p className="text-sm text-gray-500 mt-1">
          Registered templates from the backend runtime. Custom templates can be added to ~/.agent-forge/templates/
        </p>
      </div>

      <div className="space-y-3">
        {templates.map((template) => {
          const Icon = CATEGORY_ICONS[template.category] || Layout;
          const isExpanded = expandedId === template.id;

          return (
            <div
              key={template.id}
              className="bg-white rounded-xl border border-gray-200 overflow-hidden"
            >
              <button
                onClick={() => toggleExpand(template.id)}
                className="w-full text-left p-4 hover:bg-gray-50 transition"
              >
                <div className="flex items-start gap-3">
                  <div className="p-2 bg-indigo-50 rounded-lg shrink-0">
                    <Icon className="w-5 h-5 text-indigo-600" />
                  </div>
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2">
                      <h3 className="text-base font-semibold text-gray-800">{template.name}</h3>
                      <span className="text-xs bg-gray-100 text-gray-500 px-2 py-0.5 rounded-full">
                        {template.category}
                      </span>
                      <code className="text-xs text-gray-400 font-mono ml-auto">{template.id}</code>
                    </div>
                    <p className="text-sm text-gray-600 mt-1">{template.description}</p>
                  </div>
                  <div className="shrink-0 mt-1">
                    {isExpanded ? (
                      <ChevronDown className="w-4 h-4 text-gray-400" />
                    ) : (
                      <ChevronRight className="w-4 h-4 text-gray-400" />
                    )}
                  </div>
                </div>
              </button>

              {isExpanded && (
                <div className="border-t border-gray-200 p-4 bg-gray-50">
                  <div className="flex items-center gap-2 mb-2">
                    <FileText className="w-4 h-4 text-gray-500" />
                    <span className="text-sm font-medium text-gray-700">Template Details</span>
                  </div>
                  <div className="p-3 bg-gray-900 text-gray-200 rounded-lg text-xs font-mono whitespace-pre-wrap leading-relaxed">
                    {`id: ${template.id}
name: ${template.name}
category: ${template.category}
description: ${template.description}`}
                  </div>
                </div>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}
