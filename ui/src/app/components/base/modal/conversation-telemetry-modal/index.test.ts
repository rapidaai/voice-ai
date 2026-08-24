import {
  buildLatencySeries,
  buildTelemetryCriteriaInputs,
  matchesTelemetryFilters,
  splitStructuredTelemetryCriteria,
} from '@/app/components/base/modal/conversation-telemetry-modal';

describe('conversation telemetry structured criteria helpers', () => {
  it('extracts conversation and message/context ids from criteria list', () => {
    const parsed = splitStructuredTelemetryCriteria([
      { key: 'conversationId', value: '123' },
      { key: 'contextId', value: 'ctx-1' },
      { key: 'scope', value: 'telephony' },
    ]);

    expect(parsed.conversationId).toBe('123');
    expect(parsed.messageId).toBe('ctx-1');
    expect(parsed.remaining).toEqual([{ key: 'scope', value: 'telephony' }]);
  });

  it('builds server criteria with backend keys conversationId and messageId', () => {
    const criteria = buildTelemetryCriteriaInputs(
      [{ key: 'scope', value: 'telephony' }],
      '321',
      'msg-9',
    );

    expect(criteria).toEqual([
      { key: 'scope', value: 'telephony' },
      { key: 'conversationId', value: '321' },
      { key: 'messageId', value: 'msg-9' },
    ]);
  });

  it('matches telemetry documents against free-text and dropdown filters', () => {
    expect(
      matchesTelemetryFilters(
        {
          kind: 'event',
          componentType: 'telephony',
          typeLabel: 'sip.call.connected',
          name: 'sip.call.connected',
          scope: '',
          conversationId: '100',
          messageId: 'call-9',
          contextId: '',
          eventDataType: 'connected',
          rawText: '{"status":"connected"}',
        },
        {
          searchText: 'connected',
          names: ['sip.call'],
          messageOrContextId: 'call-9',
          eventDataType: 'connected',
          metricScope: '',
        },
      ),
    ).toBe(true);

    expect(
      matchesTelemetryFilters(
        {
          kind: 'event',
          componentType: 'telephony',
          typeLabel: 'sip.call.lifecycle',
          name: 'sip.call.lifecycle',
          scope: '',
          conversationId: '100',
          messageId: 'call-9',
          contextId: '',
          eventDataType: 'initialized',
          rawText: '{\n  "data": {\n    "type": "initialized"\n  }\n}',
        },
        {
          searchText: '"type": "initialized"',
          names: [],
          messageOrContextId: '',
          eventDataType: '',
          metricScope: '',
        },
      ),
    ).toBe(true);

    expect(
      matchesTelemetryFilters(
        {
          kind: 'metric',
          componentType: 'metric',
          typeLabel: 'metric.llm',
          name: '',
          scope: 'llm',
          conversationId: '100',
          messageId: '',
          contextId: 'ctx-1',
          eventDataType: '',
          rawText: '{"status":"connected"}',
        },
        {
          searchText: 'connected',
          names: [],
          messageOrContextId: '',
          eventDataType: '',
          metricScope: 'conversation',
        },
      ),
    ).toBe(false);

    expect(
      matchesTelemetryFilters(
        {
          kind: 'metric',
          componentType: 'metric',
          typeLabel: 'metric.llm',
          name: '',
          scope: 'llm',
          conversationId: '100',
          messageId: '',
          contextId: 'ctx-1',
          eventDataType: '',
          rawText: '{"status":"connected"}',
        },
        {
          searchText: '',
          names: [],
          messageOrContextId: '',
          eventDataType: '',
          metricScope: 'llm',
        },
      ),
    ).toBe(true);
  });

  it('builds latency series with stt/tts/agent TTFT/eos metrics merged by context', () => {
    const series = buildLatencySeries([
      {
        timestampMs: 2000,
        contextId: 'ctx-1',
        conversationId: 'conv-1',
        metrics: [
          { name: 'stt.latency_ms', value: '10' },
          { name: 'tts.latency_ms', value: '20' },
        ],
      },
      {
        timestampMs: 1900,
        contextId: 'ctx-1',
        conversationId: 'conv-1',
        metrics: [
          { name: 'agent.latency_ms', value: '300' },
          { name: 'agent.ttft_ms', value: '30' },
          { name: 'eos.latency_ms', value: '40' },
        ],
      },
      {
        timestampMs: 2100,
        contextId: 'ctx-2',
        conversationId: 'conv-1',
        metrics: [{ name: 'stt.latency_ms', value: '5' }],
      },
      {
        timestampMs: 2200,
        contextId: 'ctx-3',
        conversationId: 'conv-1',
        metrics: [{ name: 'agent.ttft_ms', value: '15' }],
      },
      {
        timestampMs: 2300,
        contextId: 'ctx-4',
        conversationId: 'conv-1',
        metrics: [{ name: 'agent.total_token', value: '100' }],
      },
    ]);

    expect(series).toHaveLength(3);

    expect(series[0]).toMatchObject({
      contextId: 'ctx-1',
      conversationId: 'conv-1',
      'stt.latency_ms': 10,
      'tts.latency_ms': 20,
      'agent.ttft_ms': 30,
      'eos.latency_ms': 40,
      sequence: 1,
      timestampMs: 1900,
    });
    expect(series[0]).not.toHaveProperty('agent.latency_ms');

    expect(series[1]).toMatchObject({
      contextId: 'ctx-2',
      conversationId: 'conv-1',
      'stt.latency_ms': 5,
      sequence: 2,
    });
    expect(series[2]).toMatchObject({
      contextId: 'ctx-3',
      conversationId: 'conv-1',
      'agent.ttft_ms': 15,
      sequence: 3,
    });
  });
});
