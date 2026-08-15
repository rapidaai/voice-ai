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

  it('builds latency series with stt/tts/llm/eos metrics merged by context', () => {
    const series = buildLatencySeries([
      {
        timestampMs: 2000,
        contextId: 'ctx-1',
        conversationId: 'conv-1',
        metrics: [
          { name: 'stt.latency_ms', value: '10' },
          { name: 'tts_latency_ms', value: '20' },
        ],
      },
      {
        timestampMs: 1900,
        contextId: 'ctx-1',
        conversationId: 'conv-1',
        metrics: [
          { name: 'llm_latency_ms', value: '30' },
          { name: 'eos_latency_ms', value: '40' },
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
        metrics: [{ name: 'agent_total_token', value: '100' }],
      },
    ]);

    expect(series).toHaveLength(2);

    expect(series[0]).toMatchObject({
      contextId: 'ctx-1',
      conversationId: 'conv-1',
      'stt.latency_ms': 10,
      tts_latency_ms: 20,
      llm_latency_ms: 30,
      eos_latency_ms: 40,
      sequence: 1,
      timestampMs: 1900,
    });

    expect(series[1]).toMatchObject({
      contextId: 'ctx-2',
      conversationId: 'conv-1',
      'stt.latency_ms': 5,
      sequence: 2,
    });
  });
});
