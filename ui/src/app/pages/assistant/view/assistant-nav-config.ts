import {
  Dashboard,
  Chat,
  Settings,
  Deploy,
  ToolKit,
  Webhook,
  Activity,
  Debug,
  Phone,
  ChartLine,
  DigitalIdentity,
  ObjectStorage,
  SourceControl,
  CarbonIconType,
} from '@carbon/icons-react';
import { Assistant } from '@rapidaai/react';

// ─── Types ───────────────────────────────────────────────────────────────────

export interface AssistantNavChild {
  key: string;
  label: string;
  path: string;
  action?: string;
}

export interface AssistantNavItem {
  key: string;
  label: string;
  icon: CarbonIconType;
  path: string;
  exact?: boolean;
  action?: string;
  visible?: (assistant: Assistant) => boolean;
  /** Sub-items — renders as SideNavMenu with children */
  children?: AssistantNavChild[];
}

export interface AssistantNavSection {
  label: string;
  items: AssistantNavItem[];
}

const hasModelProvider = (assistant: Assistant): boolean =>
  assistant.hasAssistantprovidermodel();

// ─── Config ──────────────────────────────────────────────────────────────────

export const assistantNavSections: AssistantNavSection[] = [
  {
    label: '',
    items: [
      {
        key: 'overview',
        label: 'Overview',
        icon: Dashboard,
        path: 'overview',
        exact: true,
      },
      {
        key: 'sessions',
        label: 'Sessions',
        icon: Chat,
        path: 'sessions',
      },
      {
        key: 'versions',
        label: 'Versions',
        icon: SourceControl,
        path: 'version-history',
        children: [
          { key: 'versions-list', label: 'View all', path: 'version-history' },
          {
            key: 'versions-create',
            label: 'Add new version',
            path: 'create-new-version',
          },
          {
            key: 'versions-agentkit',
            label: 'Add AgentKit',
            path: 'create-agentkit-version',
          },
          {
            key: 'versions-agentflow',
            label: 'Add Agentflow',
            path: 'create-agentflow-version',
          },
        ],
      },
    ],
  },
  {
    label: 'Settings',
    items: [
      {
        key: 'general',
        label: 'General',
        icon: Settings,
        path: 'edit-assistant',
      },
      {
        key: 'authentication',
        label: 'Authentication',
        icon: DigitalIdentity,
        path: 'configure-authentication',
        children: [
          {
            key: 'authentication-list',
            label: 'View all',
            path: 'configure-authentication',
          },
          {
            key: 'authentication-create',
            label: 'Add authentication',
            path: 'configure-authentication/create',
          },
        ],
      },
      {
        key: 'deployment',
        label: 'Deployment',
        icon: Deploy,
        path: 'deployment',
        children: [
          { key: 'deployment-list', label: 'View all', path: 'deployment' },
          {
            key: 'deployment-debugger',
            label: 'Add Debugger',
            path: 'deployment/debugger',
          },
          {
            key: 'deployment-web',
            label: 'Add Web Widget',
            path: 'deployment/web',
          },
          {
            key: 'deployment-api',
            label: 'Add SDK / API',
            path: 'deployment/api',
          },
          {
            key: 'deployment-call',
            label: 'Add Phone Call',
            path: 'deployment/call',
          },
        ],
      },
      {
        key: 'tools',
        label: 'Tools & MCP',
        icon: ToolKit,
        path: 'configure-tool',
        visible: hasModelProvider,
        children: [
          { key: 'tools-list', label: 'View all', path: 'configure-tool' },
          {
            key: 'tools-create',
            label: 'Add tool',
            path: 'configure-tool/create',
          },
        ],
      },
      {
        key: 'webhooks',
        label: 'Webhooks',
        icon: Webhook,
        path: 'configure-webhook',
        children: [
          {
            key: 'webhooks-list',
            label: 'View all',
            path: 'configure-webhook',
          },
          {
            key: 'webhooks-create',
            label: 'Add webhook',
            path: 'configure-webhook/create',
          },
        ],
      },
      {
        key: 'telemetry',
        label: 'Telemetry',
        icon: Activity,
        path: 'configure-telemetry',
        children: [
          {
            key: 'telemetry-list',
            label: 'View all',
            path: 'configure-telemetry',
          },
          {
            key: 'telemetry-create',
            label: 'Add provider',
            path: 'configure-telemetry/create',
          },
        ],
      },
      {
        key: 'storage',
        label: 'Storage',
        icon: ObjectStorage,
        path: 'configure-storage',
        children: [
          { key: 'storage-list', label: 'View all', path: 'configure-storage' },
          {
            key: 'storage-create',
            label: 'Add storage',
            path: 'configure-storage/create',
          },
        ],
      },
      {
        key: 'analysis',
        label: 'Analysis',
        icon: ChartLine,
        path: 'configure-analysis',
        children: [
          {
            key: 'analysis-list',
            label: 'View all',
            path: 'configure-analysis',
          },
          {
            key: 'analysis-create',
            label: 'Add analysis',
            path: 'configure-analysis/create',
          },
        ],
      },
    ],
  },
  {
    label: 'Playground',
    items: [
      {
        key: 'debugging',
        label: 'Chat',
        icon: Debug,
        path: 'playground/chat',
      },
      {
        key: 'phone-preview',
        label: 'Phone call',
        icon: Phone,
        path: 'playground/phone',
        visible: (assistant: any) => !!assistant?.getPhonedeployment?.(),
      },
    ],
  },
];
