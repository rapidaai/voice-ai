import {
  COMPONENT_OPTIONS,
  EVENTS_BY_COMPONENT,
  LEVEL_OPTIONS,
  METRIC_NAME_OPTIONS,
  ROLE_OPTIONS,
  SCOPE_OPTIONS,
  getEventOptionsForComponent,
} from './constants';
import {
  createTraceFilter,
  dedupeTraceFilters,
  getDocumentComponent,
  getTraceFilterValues,
  matchesTraceFilters,
  parseTraceFilterQuery,
  telemetryRecordToTimelineDocument,
} from './utils';
import type { TimelineDocument } from './types';

const mapFromEntries = (entries: Array<[string, string]> = []) => ({
  toArray: () => entries,
});

const timestamp = (value = '2026-06-04T03:10:00.000Z') => ({
  toDate: () => new Date(value),
});

const metricRecord = ({
  component = 'stt',
  name,
  value,
}: {
  component?: string;
  name: string;
  value: string;
}) =>
  ({
    getAttributesMap: () =>
      mapFromEntries(component ? [['component', component]] : []),
    getContextMap: () => mapFromEntries([['traceId', 'trace-1']]),
    getDescription: () => '',
    getId: () => 'metric-1',
    getName: () => name,
    getOccurredat: () => timestamp(),
    getOrganizationid: () => '1',
    getProjectid: () => '2',
    getScope: () => 'message',
    getScopeattributesMap: () =>
      mapFromEntries([
        ['assistantId', '10'],
        ['assistantConversationId', '20'],
        ['messageId', 'message-1'],
        ['messageRole', 'user'],
      ]),
    getValue: () => value,
  }) as any;

const observabilityRecord = (metric: any) =>
  ({
    getEvent: () => undefined,
    getLog: () => undefined,
    getMetric: () => metric,
  }) as any;

describe('conversation activity v2 telemetry utilities', () => {
  it('uses component latency metrics as waterfall durations', () => {
    const document = telemetryRecordToTimelineDocument(
      observabilityRecord(
        metricRecord({ name: 'stt.latency_ms', value: '47.4' }),
      ),
      0,
    );

    expect(document?.durationMs).toBe(47);
  });

  it('does not treat non-latency counters as durations', () => {
    const document = telemetryRecordToTimelineDocument(
      observabilityRecord(metricRecord({ name: 'input_token', value: '200' })),
      0,
    );

    expect(document?.durationMs).toBeUndefined();
  });

  it('maps backend metric names to canonical timeline components', () => {
    const document = telemetryRecordToTimelineDocument(
      observabilityRecord(
        metricRecord({
          component: '',
          name: 'webrtc.ice_latency_ms',
          value: '22',
        }),
      ),
      0,
    );

    expect(document?.category).toBe('webrtc');
    expect(document ? getDocumentComponent(document) : '').toBe('webrtc');
  });

  it('prefers the record component over provider attributes', () => {
    const document: TimelineDocument = {
      id: 'evt-eos-completed',
      kind: 'event',
      name: 'eos.completed',
      component: 'eos',
      category: 'eos',
      level: 'info',
      outcome: 'success',
      title: 'eos.completed',
      projectId: 2,
      organizationId: 1,
      scope: 'message',
      assistantId: '2337454103765975040',
      assistantConversationId: '2340105440068632576',
      messageId: 'a49f2845-68ec-4a59-a30d-5e2b30df87bf',
      messageRole: 'user',
      traceId: 'trace-1',
      contextId: 'a49f2845-68ec-4a59-a30d-5e2b30df87bf',
      occurredAt: '2026-06-04T03:10:00.000Z',
      receivedAt: '2026-06-04T03:10:00.000Z',
      attributes: { provider: 'pipecatEndOfSpeech' },
    };

    const filters = parseTraceFilterQuery('component:eos').filters;

    expect(getDocumentComponent(document)).toBe('eos');
    expect(matchesTraceFilters(document, filters)).toBe(true);
  });

  it('uses current backend log levels, scopes, and message roles', () => {
    expect(LEVEL_OPTIONS.map(option => option.id)).toEqual([
      'all',
      'debug',
      'info',
      'error',
      'critical',
    ]);
    expect(SCOPE_OPTIONS.map(option => option.id)).toEqual([
      'all',
      'project',
      'assistant',
      'conversation',
      'message',
    ]);
    expect(ROLE_OPTIONS.map(option => option.id)).toEqual([
      'all',
      'user',
      'assistant',
    ]);
  });

  it('uses current backend model and AgentKit metric names', () => {
    const metricIds = METRIC_NAME_OPTIONS.map(option => option.id);

    expect(metricIds).toEqual(
      expect.arrayContaining([
        'agent.latency_ms',
        'agent.message_char_count',
        'agent.message_count',
        'agent.response_char_count',
        'authentication.init_ms',
        'recording.init_ms',
        'agent.ttft_ms',
        'agent.trt_ms',
        'agent.error',
      ]),
    );
    expect(metricIds).not.toEqual(
      expect.arrayContaining([
        'time_taken',
        'agent_time_taken',
        'time_to_first_token',
        'provider_total_time',
        'provider_generate_time',
        'agent_provider_total_time',
        'agent_provider_generate_time',
        'agent_status',
        'agent.llm_request_id',
        'agent_input_token',
        'agent_output_token',
        'agent.total_token',
        'agent.cached_content_token',
        'agent.cost',
        'agent.input_cost',
        'agent.output_cost',
        'agent.token_pre_second',
        'agent_token_count',
        'input_token',
        'output_token',
      ]),
    );
  });

  it('uses current backend observability components and events', () => {
    const componentIds = COMPONENT_OPTIONS.map(option => option.id);
    expect(componentIds).toContain('agent');
    expect(componentIds).toContain('sip');
    expect(componentIds).toContain('webrtc');
    expect(componentIds).not.toContain('agentflow');
    expect(componentIds).not.toContain('llm');
    expect(componentIds).not.toContain('session');
    expect(componentIds).not.toContain('audio');

    expect(EVENTS_BY_COMPONENT.call).toContain('call.provider_answered');
    expect(EVENTS_BY_COMPONENT.call).not.toContain('call.answered');
    expect(EVENTS_BY_COMPONENT.stt).toContain('stt.interim');
    expect(EVENTS_BY_COMPONENT.stt).not.toContain('stt.final');
    expect(EVENTS_BY_COMPONENT.turn).toEqual([
      'turn.interrupted',
      'turn.change',
      'turn.started',
    ]);
    expect(EVENTS_BY_COMPONENT.tts).not.toContain('tts.discard_chunk');
    expect(EVENTS_BY_COMPONENT.tts).not.toContain('tts.discarded');
    expect(EVENTS_BY_COMPONENT.tts).not.toContain('tts.first_audio');
    expect(EVENTS_BY_COMPONENT.agent).toEqual([
      'agent.started',
      'agent.completed',
      'agent.discarded',
      'agent.error',
      'agent.transition.triggered',
      'agent.transition.matched',
      'agent.transition.missing_edge',
    ]);
    expect(EVENTS_BY_COMPONENT.conversation).toContain(
      'conversation.authentication_started',
    );
    expect(EVENTS_BY_COMPONENT.conversation).toContain(
      'conversation.mode_switch_failed',
    );

    expect(
      getEventOptionsForComponent('webrtc').map(option => option.id),
    ).toEqual([
      'all',
      'webrtc.connecting',
      'webrtc.connected',
      'webrtc.reconnecting',
      'webrtc.disconnected',
      'webrtc.failed',
      'webrtc.ice_connection_state',
      'webrtc.ice_connected',
      'webrtc.ice_failed',
      'webrtc.audio_track_received',
      'webrtc.peer_quality',
      'webrtc.selected_ice_candidate_pair',
      'webrtc.negotiation_offer_sent',
      'webrtc.negotiation_answer_received',
      'webrtc.negotiation_retry_queued',
      'webrtc.negotiation_retry_sent',
      'webrtc.ice_restart_deferred',
    ]);
  });

  it('parses Sentry-style query filters and leaves unmatched text as search', () => {
    const parsed = parseTraceFilterQuery(
      'refund conversation:2340105440068632576 component:tts event:tts.speaking role:assistant',
    );

    expect(parsed.freeText).toBe('refund');
    expect(
      parsed.filters.map(filter => [
        filter.fieldKey,
        filter.criteriaKey,
        filter.value,
      ]),
    ).toEqual([
      ['conversation', 'assistantConversationId', '2340105440068632576'],
      ['component', 'component', 'tts'],
      ['event', 'event', 'tts.speaking'],
      ['role', 'messageRole', 'assistant'],
    ]);
  });

  it('parses timestamp range filters with one criteria key and different logic', () => {
    const parsed = parseTraceFilterQuery(
      'timestamp:2026-06-04T03:10:00.000Z timestamp~<=:2026-06-04T03:20:00.000Z',
    );

    expect(
      parsed.filters.map(filter => [
        filter.fieldKey,
        filter.criteriaKey,
        filter.logic,
        filter.value,
      ]),
    ).toEqual([
      ['timestamp', 'timestamp', '>=', '2026-06-04T03:10:00.000Z'],
      ['timestamp', 'timestamp', '<=', '2026-06-04T03:20:00.000Z'],
    ]);
  });

  it('filters conversation attributes across message-scope records', () => {
    const document: TimelineDocument = {
      id: 'evt-tts-speaking',
      kind: 'event',
      name: 'tts.speaking',
      category: 'tts',
      level: 'info',
      outcome: 'success',
      title: 'TTS speaking',
      projectId: 2,
      organizationId: 1,
      scope: 'message',
      assistantId: '2337454103765975040',
      assistantConversationId: '2340105440068632576',
      messageId: 'a49f2845-68ec-4a59-a30d-5e2b30df87bf',
      messageRole: 'user',
      traceId: 'trace-1',
      contextId: 'a49f2845-68ec-4a59-a30d-5e2b30df87bf',
      occurredAt: '2026-06-04T03:10:00.000Z',
      receivedAt: '2026-06-04T03:10:00.000Z',
      attributes: { component: 'tts' },
      data: {
        scopeAttributes: {
          assistantConversationId: '2340105440068632576',
        },
      },
    };

    const conversationFilters = parseTraceFilterQuery(
      'conversation:2340105440068632576 component:tts',
    ).filters;
    const attributeFilters = parseTraceFilterQuery(
      'scopeAttributes.assistantConversationId:2340105440068632576 component:tts',
    ).filters;
    const conversationScopeFilters = parseTraceFilterQuery(
      'conversation:2340105440068632576 scope:conversation',
    ).filters;

    expect(matchesTraceFilters(document, conversationFilters)).toBe(true);
    expect(matchesTraceFilters(document, attributeFilters)).toBe(true);
    expect(matchesTraceFilters(document, conversationScopeFilters)).toBe(false);
  });

  it('deduplicates query and facet filters that map to the same criteria', () => {
    const parsed = parseTraceFilterQuery('conversation:234');
    const filters = dedupeTraceFilters([
      ...parsed.filters,
      createTraceFilter('conversation', '234', 'facet'),
    ]);

    expect(filters).toHaveLength(1);
    expect(filters[0]?.criteriaKey).toBe('assistantConversationId');
  });

  it('matches comma-separated component filters as OR values', () => {
    const document: TimelineDocument = {
      id: 'evt-eos-completed',
      kind: 'event',
      name: 'eos.completed',
      category: 'eos',
      level: 'info',
      outcome: 'success',
      title: 'EOS completed',
      projectId: 2,
      organizationId: 1,
      scope: 'message',
      assistantId: '2337454103765975040',
      assistantConversationId: '2340105440068632576',
      messageId: 'message-1',
      messageRole: 'user',
      traceId: 'trace-1',
      contextId: 'message-1',
      occurredAt: '2026-06-04T03:10:00.000Z',
      receivedAt: '2026-06-04T03:10:00.000Z',
      attributes: { component: 'eos' },
      data: {},
    };
    const filters = parseTraceFilterQuery('component:eos,stt,agent').filters;

    expect(getTraceFilterValues(filters[0])).toEqual(['eos', 'stt', 'agent']);
    expect(matchesTraceFilters(document, filters)).toBe(true);
    expect(
      matchesTraceFilters(
        document,
        parseTraceFilterQuery('component:stt,agent').filters,
      ),
    ).toBe(false);
  });
});
