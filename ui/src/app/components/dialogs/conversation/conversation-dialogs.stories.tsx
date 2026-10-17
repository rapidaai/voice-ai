import type { Meta, StoryObj } from '@storybook/react-vite';
import type { AssistantConversationTelephonyEvent } from '@rapidaai/react';
import { AssistantConversationTelephonyEventPanel } from './assistant-conversation-telephony-event-modal';
import { LatencyStackChart } from './conversation-telemetry-latency-stack-chart';

const makeTelephonyEvent = ({
  id,
  eventType,
  provider = 'twilio',
  payload = {},
}: {
  id: string;
  eventType: string;
  provider?: string;
  payload?: Record<string, unknown>;
}): AssistantConversationTelephonyEvent =>
  ({
    getId: () => id,
    getProvider: () => provider,
    getEventtype: () => eventType,
    getCreateddate: () => new Date('2026-01-01T12:00:00.000Z'),
    getPayload: () => ({
      toJavaScript: () => payload,
    }),
  }) as unknown as AssistantConversationTelephonyEvent;

const meta = {
  title: 'Dialogs/Conversation',
  tags: ['autodocs'],
  parameters: {
    docs: {
      description: {
        component:
          'Conversation dialog surfaces for telephony events and telemetry latency visualization.',
      },
    },
  },
} satisfies Meta;

export default meta;

type Story = StoryObj<typeof meta>;

export const TelephonyEvents: Story = {
  render: () => (
    <div className="h-[620px] max-w-5xl border border-border-subtle bg-layer text-foreground">
      <AssistantConversationTelephonyEventPanel
        events={[
          makeTelephonyEvent({
            id: 'evt_1001',
            eventType: 'ringing',
            payload: { callSid: 'CA123', direction: 'inbound' },
          }),
          makeTelephonyEvent({
            id: 'evt_1002',
            eventType: 'connected',
            payload: { callSid: 'CA123', status: 'connected' },
          }),
        ]}
      />
    </div>
  ),
};

export const EmptyTelephonyEvents: Story = {
  render: () => (
    <div className="h-[420px] max-w-5xl border border-border-subtle bg-layer text-foreground">
      <AssistantConversationTelephonyEventPanel events={[]} />
    </div>
  ),
};

export const LatencyMetrics: Story = {
  render: () => (
    <div className="h-[520px] max-w-5xl border border-border-subtle bg-layer text-foreground">
      <LatencyStackChart
        isLoading={false}
        latencySeries={[
          {
            key: 'conv-1::ctx-1',
            sequence: 1,
            timestampMs: 1767225601000,
            timeLabel: '2026-01-01 00:00:01.000',
            contextId: 'ctx-1',
            conversationId: 'conv-1',
            'stt.latency_ms': 120,
            'eos.latency_ms': 45,
            'agent.ttft_ms': 380,
            'tts.latency_ms': 160,
          },
          {
            key: 'conv-1::ctx-2',
            sequence: 2,
            timestampMs: 1767225603000,
            timeLabel: '2026-01-01 00:00:03.000',
            contextId: 'ctx-2',
            conversationId: 'conv-1',
            'stt.latency_ms': 90,
            'eos.latency_ms': 35,
            'agent.ttft_ms': 420,
            'tts.latency_ms': 140,
          },
        ]}
      />
    </div>
  ),
};

export const EmptyLatencyMetrics: Story = {
  render: () => (
    <div className="h-[320px] max-w-5xl border border-border-subtle bg-layer text-foreground">
      <LatencyStackChart isLoading={false} latencySeries={[]} />
    </div>
  ),
};
