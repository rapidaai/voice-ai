import type { Meta, StoryObj } from '@storybook/react-vite';
import { useState } from 'react';
import type {
  AssistantConversationMessage,
  AssistantHTTPLog,
  AssistantToolLog,
  AuditLog as AuditLogMessage,
  KnowledgeLog as KnowledgeLogMessage,
  Metadata,
} from '@rapidaai/react';
import type { ReactNode } from 'react';
import { ConversationLogContent } from './conversation-log-modal';
import { KnowledgeLogContent } from './knowledge-log-modal';
import { LLMLogContent } from './llm-log-modal';
import { ToolLogContent } from './tool-log-modal';
import { RequestLogContent } from './webhook-log-modal';

const meta = {
  title: 'Dialogs/Activity/LogModals',
  tags: ['autodocs'],
  parameters: {
    docs: {
      description: {
        component:
          'Activity log detail panels used inside drawer modals for conversation, knowledge, LLM, tool, and webhook logs.',
      },
    },
  },
} satisfies Meta;

export default meta;

type Story = StoryObj;

const jsonValue = (value: unknown) => ({
  toJavaScript: () => value,
});

const metric = (value: Record<string, unknown>) => ({
  toObject: () => value,
});

const metadata = (key: string, value: string) =>
  ({
    getKey: () => key,
    getValue: () => value,
  }) as Metadata;

const storyFrame = (children: ReactNode) => (
  <div className="h-[32rem] w-[36rem] border border-border-subtle bg-layer text-foreground">
    {children}
  </div>
);

export const ConversationLog: Story = {
  render: function Render() {
    const [selectedTab, setSelectedTab] = useState(0);
    const message = {
      getAssistantconversationid: () => 'conv_01HZY9',
      getBody: () => 'The assistant confirmed the deployment was complete.',
      getMetricsList: () => [
        metric({ name: 'first_token_latency_ms', value: 341 }),
        metric({ name: 'total_tokens', value: 892 }),
      ],
      getMetadataList: () => [
        metadata('channel', 'debugger'),
        metadata('region', 'us-east'),
      ],
    } as unknown as AssistantConversationMessage;

    return storyFrame(
      <ConversationLogContent
        currentAssistantMessage={message}
        selectedTab={selectedTab}
        onTabChange={setSelectedTab}
      />,
    );
  },
};

export const KnowledgeLog: Story = {
  render: function Render() {
    const [selectedTab, setSelectedTab] = useState(0);
    const activity = {
      getStatus: () => 'SUCCESS',
      getTimetaken: () => '2400000',
      getCreateddate: () => '2026-09-19T15:45:00Z',
      getRequest: () => jsonValue({ query: 'return policy', topK: 4 }),
      getResponse: () =>
        jsonValue({
          documents: ['policy-handbook.pdf', 'returns.md'],
          confidence: 0.91,
        }),
    } as unknown as KnowledgeLogMessage;

    return storyFrame(
      <KnowledgeLogContent
        activity={activity}
        selectedTab={selectedTab}
        onTabChange={setSelectedTab}
      />,
    );
  },
};

export const LLMLog: Story = {
  render: function Render() {
    const [selectedTab, setSelectedTab] = useState(0);
    const activity = {
      getStatus: () => 'SUCCESS',
      getTimetaken: () => 8900000,
      getCreateddate: () => '2026-09-19T16:10:00Z',
      getResponsestatus: () => 200,
      getRequest: () =>
        jsonValue({
          model: 'gpt-4.1',
          messages: [{ role: 'user', content: 'Summarize the trace' }],
        }),
      getResponse: () =>
        jsonValue({ role: 'assistant', content: 'Trace completed normally.' }),
      getMetricsList: () => [metric({ inputTokens: 362, outputTokens: 78 })],
    } as unknown as AuditLogMessage;

    return storyFrame(
      <LLMLogContent
        activity={activity}
        additionalData={[metadata('model_name', 'gpt-4.1')]}
        selectedTab={selectedTab}
        onTabChange={setSelectedTab}
      />,
    );
  },
};

export const ToolLog: Story = {
  render: function Render() {
    const [selectedTab, setSelectedTab] = useState(0);
    const activity = {
      getRequest: () => jsonValue({ tool: 'lookupCustomer', id: 'cus_928' }),
      getResponse: () =>
        jsonValue({ customerTier: 'enterprise', renewalRisk: 'low' }),
    } as unknown as AssistantToolLog;

    return storyFrame(
      <ToolLogContent
        activity={activity}
        selectedTab={selectedTab}
        onTabChange={setSelectedTab}
      />,
    );
  },
};

export const RequestLog: Story = {
  render: function Render() {
    const [selectedTab, setSelectedTab] = useState(0);
    const activity = {
      getRequest: () =>
        jsonValue({
          method: 'POST',
          path: '/webhooks/order-created',
          body: { orderId: 'ord_394' },
        }),
      getResponse: () => jsonValue({ status: 202, accepted: true }),
    } as unknown as AssistantHTTPLog;

    return storyFrame(
      <RequestLogContent
        activity={activity}
        selectedTab={selectedTab}
        onTabChange={setSelectedTab}
      />,
    );
  },
};
