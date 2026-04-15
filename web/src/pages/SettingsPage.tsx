import { useEffect, useState, type ComponentType, type ReactNode } from 'react';
import { Save, Bell, Terminal, DollarSign, RefreshCw, AlertCircle } from 'lucide-react';
import { api, type AppConfig } from '../lib/api';

const DEFAULT_CONFIG: AppConfig = {
  server: {
    host: '0.0.0.0',
    port: 8080,
  },
  notification: {
    webhook_url: '',
    enabled_events: {
      task_complete: true,
      task_failed: true,
      merge_conflict: true,
      cost_alert: true,
    },
  },
  cli: {
    provider: 'claude',
    claude_path: 'claude',
    codex_path: 'codex',
    model: '',
    max_retries: 3,
    default_timeout: '30m',
  },
  cost: {
    alert_threshold: 10.0,
    hard_limit: 50.0,
    input_cost_per_mil: 3.0,
    output_cost_per_mil: 15.0,
  },
};

export function SettingsPage() {
  const [config, setConfig] = useState<AppConfig>(DEFAULT_CONFIG);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    api
      .getConfig()
      .then((data) => {
        if (!cancelled) {
          setConfig(mergeConfigDefaults(data));
          setError(null);
        }
      })
      .catch((err: Error) => {
        if (!cancelled) {
          setError(err.message);
        }
      })
      .finally(() => {
        if (!cancelled) {
          setLoading(false);
        }
      });
    return () => {
      cancelled = true;
    };
  }, []);

  const handleSave = async () => {
    setSaving(true);
    setSaved(false);
    setError(null);
    try {
      const updated = await api.updateConfig(config);
      setConfig(mergeConfigDefaults(updated));
      setSaved(true);
      setTimeout(() => setSaved(false), 2000);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save settings');
    } finally {
      setSaving(false);
    }
  };

  const handleReset = () => {
    setConfig(DEFAULT_CONFIG);
    setSaved(false);
    setError(null);
  };

  const updateCLI = <K extends keyof AppConfig['cli']>(key: K, value: AppConfig['cli'][K]) => {
    setConfig((c) => ({ ...c, cli: { ...c.cli, [key]: value } }));
  };

  const updateNotification = <K extends keyof AppConfig['notification']>(
    key: K,
    value: AppConfig['notification'][K],
  ) => {
    setConfig((c) => ({ ...c, notification: { ...c.notification, [key]: value } }));
  };

  const updateEvent = (key: string, value: boolean) => {
    setConfig((c) => ({
      ...c,
      notification: {
        ...c.notification,
        enabled_events: { ...c.notification.enabled_events, [key]: value },
      },
    }));
  };

  const updateCost = <K extends keyof AppConfig['cost']>(key: K, value: AppConfig['cost'][K]) => {
    setConfig((c) => ({ ...c, cost: { ...c.cost, [key]: value } }));
  };

  if (loading) {
    return <div className="max-w-2xl mx-auto p-6 text-sm text-gray-500">Loading settings...</div>;
  }

  return (
    <div className="max-w-2xl mx-auto p-6 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-800">Settings</h1>
          <p className="text-sm text-gray-500 mt-1">Configure AgentForge behavior</p>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={handleReset}
            className="flex items-center gap-1 px-3 py-2 text-sm text-gray-600 hover:text-gray-800 transition"
          >
            <RefreshCw className="w-4 h-4" /> Reset
          </button>
          <button
            onClick={handleSave}
            disabled={saving}
            className="flex items-center gap-1 px-4 py-2 bg-indigo-600 text-white rounded-lg text-sm hover:bg-indigo-700 disabled:opacity-60 disabled:cursor-not-allowed transition"
          >
            <Save className="w-4 h-4" /> {saving ? 'Saving...' : saved ? 'Saved!' : 'Save'}
          </button>
        </div>
      </div>

      {error && (
        <div className="flex items-start gap-2 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">
          <AlertCircle className="w-4 h-4 mt-0.5" />
          <span>{error}</span>
        </div>
      )}

      <Section icon={Terminal} title="Agent CLI">
        <FormField
          label="Provider"
          description="Applies immediately to newly started sessions; already running sessions keep their current process."
        >
          <select
            value={config.cli.provider}
            onChange={(e) => updateCLI('provider', e.target.value as AppConfig['cli']['provider'])}
            className="w-48 px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
          >
            <option value="claude">Claude Code</option>
            <option value="codex">Codex</option>
          </select>
        </FormField>

        <FormField label="Claude Code CLI path" description="Path to the claude command">
          <input
            value={config.cli.claude_path}
            onChange={(e) => updateCLI('claude_path', e.target.value)}
            placeholder="claude"
            className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm font-mono focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
          />
        </FormField>

        <FormField label="Codex CLI path" description="Path to the codex command">
          <input
            value={config.cli.codex_path}
            onChange={(e) => updateCLI('codex_path', e.target.value)}
            placeholder="codex"
            className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm font-mono focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
          />
        </FormField>

        <FormField label="Model" description="Optional model override for Codex or Claude">
          <input
            value={config.cli.model ?? ''}
            onChange={(e) => updateCLI('model', e.target.value)}
            placeholder="gpt-5.4"
            className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm font-mono focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
          />
        </FormField>

        <FormField label="Max retries" description="Maximum retry attempts for failed sessions">
          <input
            type="number"
            min={1}
            max={10}
            value={config.cli.max_retries}
            onChange={(e) => updateCLI('max_retries', parseInt(e.target.value, 10) || 1)}
            className="w-32 px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
          />
        </FormField>

        <FormField label="Default session timeout" description="Go duration syntax, for example 30m or 1h">
          <input
            value={config.cli.default_timeout}
            onChange={(e) => updateCLI('default_timeout', e.target.value)}
            placeholder="30m"
            className="w-32 px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
          />
        </FormField>
      </Section>

      <Section icon={Bell} title="Notifications">
        <FormField label="Webhook URL" description="Receive task events via webhook">
          <input
            value={config.notification.webhook_url}
            onChange={(e) => updateNotification('webhook_url', e.target.value)}
            placeholder="https://hooks.example.com/agentforge"
            className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
          />
        </FormField>

        <div className="space-y-2">
          <ToggleField
            label="Notify on task completion"
            checked={!!config.notification.enabled_events.task_complete}
            onChange={(v) => updateEvent('task_complete', v)}
          />
          <ToggleField
            label="Notify on task failure"
            checked={!!config.notification.enabled_events.task_failed}
            onChange={(v) => updateEvent('task_failed', v)}
          />
          <ToggleField
            label="Notify on merge conflicts"
            checked={!!config.notification.enabled_events.merge_conflict}
            onChange={(v) => updateEvent('merge_conflict', v)}
          />
          <ToggleField
            label="Notify on cost alerts"
            checked={!!config.notification.enabled_events.cost_alert}
            onChange={(v) => updateEvent('cost_alert', v)}
          />
        </div>
      </Section>

      <Section icon={DollarSign} title="Cost Alerts">
        <FormField label="Cost alert threshold ($)" description="Alert when task cost exceeds this amount">
          <input
            type="number"
            step="0.5"
            min={0}
            value={config.cost.alert_threshold}
            onChange={(e) => updateCost('alert_threshold', parseFloat(e.target.value) || 0)}
            className="w-48 px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
          />
        </FormField>

        <FormField label="Hard limit ($)" description="Pause new work when task cost reaches this amount">
          <input
            type="number"
            step="0.5"
            min={0}
            value={config.cost.hard_limit}
            onChange={(e) => updateCost('hard_limit', parseFloat(e.target.value) || 0)}
            className="w-48 px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
          />
        </FormField>
      </Section>
    </div>
  );
}

function mergeConfigDefaults(config: AppConfig): AppConfig {
  return {
    ...DEFAULT_CONFIG,
    ...config,
    server: { ...DEFAULT_CONFIG.server, ...config.server },
    notification: {
      ...DEFAULT_CONFIG.notification,
      ...config.notification,
      enabled_events: {
        ...DEFAULT_CONFIG.notification.enabled_events,
        ...(config.notification?.enabled_events ?? {}),
      },
    },
    cli: { ...DEFAULT_CONFIG.cli, ...config.cli },
    cost: { ...DEFAULT_CONFIG.cost, ...config.cost },
  };
}

function Section({
  icon: Icon,
  title,
  children,
}: {
  icon: ComponentType<{ className?: string }>;
  title: string;
  children: ReactNode;
}) {
  return (
    <div className="bg-white rounded-xl border border-gray-200 overflow-hidden">
      <div className="flex items-center gap-2 px-4 py-3 border-b border-gray-200 bg-gray-50">
        <Icon className="w-4 h-4 text-gray-500" />
        <h2 className="text-sm font-semibold text-gray-700">{title}</h2>
      </div>
      <div className="p-4 space-y-4">{children}</div>
    </div>
  );
}

function FormField({
  label,
  description,
  children,
}: {
  label: string;
  description?: string;
  children: ReactNode;
}) {
  return (
    <div>
      <label className="block text-sm font-medium text-gray-700">{label}</label>
      {description && <p className="text-xs text-gray-500 mb-1">{description}</p>}
      <div className="mt-1">{children}</div>
    </div>
  );
}

function ToggleField({
  label,
  checked,
  onChange,
}: {
  label: string;
  checked: boolean;
  onChange: (v: boolean) => void;
}) {
  return (
    <label className="flex items-center gap-3 cursor-pointer">
      <button
        type="button"
        onClick={() => onChange(!checked)}
        className={`relative w-9 h-5 rounded-full transition-colors ${
          checked ? 'bg-indigo-600' : 'bg-gray-300'
        }`}
      >
        <span
          className={`absolute top-0.5 left-0.5 w-4 h-4 rounded-full bg-white shadow transition-transform ${
            checked ? 'translate-x-4' : ''
          }`}
        />
      </button>
      <span className="text-sm text-gray-700">{label}</span>
    </label>
  );
}
