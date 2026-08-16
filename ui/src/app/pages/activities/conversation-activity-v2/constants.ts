export type FilterOption = {
  id: string;
  text: string;
};

export const SCOPE_OPTIONS: FilterOption[] = [
  { id: 'all', text: 'All scopes' },
  { id: 'project', text: 'Project' },
  { id: 'assistant', text: 'Assistant' },
  { id: 'conversation', text: 'Conversation' },
  { id: 'message', text: 'Message' },
];

export const KIND_OPTIONS: FilterOption[] = [
  { id: 'all', text: 'All records' },
  { id: 'log', text: 'Logs' },
  { id: 'event', text: 'Events' },
  { id: 'metric', text: 'Metrics' },
];

export const LEVEL_OPTIONS: FilterOption[] = [
  { id: 'all', text: 'All levels' },
  { id: 'debug', text: 'Debug' },
  { id: 'info', text: 'Info' },
  { id: 'error', text: 'Error' },
  { id: 'critical', text: 'Critical' },
];

export const ROLE_OPTIONS: FilterOption[] = [
  { id: 'all', text: 'All roles' },
  { id: 'user', text: 'User' },
  { id: 'assistant', text: 'Assistant' },
];

export const METRIC_NAME_OPTIONS: FilterOption[] = [
  { id: 'all', text: 'All metrics' },
  { id: 'status', text: 'status' },
  { id: 'duration', text: 'duration' },
  { id: 'stt_duration', text: 'stt_duration' },
  { id: 'tts_duration', text: 'tts_duration' },
  { id: 'call.duration_ms', text: 'call.duration_ms' },
  { id: 'call.status', text: 'call.status' },
  { id: 'call.price', text: 'call.price' },
  { id: 'sip.registration.status', text: 'sip.registration.status' },
  {
    id: 'call.transfer.bridge_duration_ms',
    text: 'call.transfer.bridge_duration_ms',
  },
  { id: 'webrtc.ice_latency_ms', text: 'webrtc.ice_latency_ms' },
  {
    id: 'webrtc.output_queue_dropped_frames',
    text: 'webrtc.output_queue_dropped_frames',
  },
  { id: 'user_turn', text: 'user_turn' },
  { id: 'assistant_turn', text: 'assistant_turn' },
  { id: 'stt.init_ms', text: 'stt.init_ms' },
  { id: 'stt.latency_ms', text: 'stt.latency_ms' },
  { id: 'stt.ttft_ms', text: 'stt.ttft_ms' },
  { id: 'stt.ttlt_ms', text: 'stt.ttlt_ms' },
  { id: 'tts_init_ms', text: 'tts_init_ms' },
  { id: 'tts.latency_ms', text: 'tts.latency_ms' },
  { id: 'vad.init_ms', text: 'vad.init_ms' },
  { id: 'eos.init_ms', text: 'eos.init_ms' },
  { id: 'eos.latency_ms', text: 'eos.latency_ms' },
  { id: 'eos.trigger_ms', text: 'eos.trigger_ms' },
  { id: 'eos.word_count', text: 'eos.word_count' },
  { id: 'eos.confidence', text: 'eos.confidence' },
  { id: 'denoise.init_ms', text: 'denoise.init_ms' },
  { id: 'agent.init_ms', text: 'agent.init_ms' },
  { id: 'storage.init_ms', text: 'storage.init_ms' },
  { id: 'analysis.init_ms', text: 'analysis.init_ms' },
  { id: 'authentication.init_ms', text: 'authentication.init_ms' },
  { id: 'authentication.latency_ms', text: 'authentication.latency_ms' },
  { id: 'recording.init_ms', text: 'recording.init_ms' },
  { id: 'knowledge_latency_ms', text: 'knowledge_latency_ms' },
  { id: 'agent.latency_ms', text: 'agent.latency_ms' },
  { id: 'agent.message_char_count', text: 'agent.message_char_count' },
  { id: 'agent.message_count', text: 'agent.message_count' },
  { id: 'agent.response_char_count', text: 'agent.response_char_count' },
  { id: 'agent.error', text: 'agent.error' },
  { id: 'stt.error', text: 'stt.error' },
  { id: 'tts.error', text: 'tts.error' },
  { id: 'tts.discard_chunk_count', text: 'tts.discard_chunk_count' },
  { id: 'agent.ttft_ms', text: 'agent.ttft_ms' },
  { id: 'agent.trt_ms', text: 'agent.trt_ms' },
];

export const COMPONENT_OPTIONS: FilterOption[] = [
  { id: 'all', text: 'All components' },
  { id: 'call', text: 'call' },
  { id: 'conversation', text: 'conversation' },
  { id: 'turn', text: 'turn' },
  { id: 'stt', text: 'stt' },
  { id: 'tts', text: 'tts' },
  { id: 'agent', text: 'agent' },
  { id: 'vad', text: 'vad' },
  { id: 'eos', text: 'eos' },
  { id: 'denoise', text: 'denoise' },
  { id: 'tool', text: 'tool' },
  { id: 'webhook', text: 'webhook' },
  { id: 'analysis', text: 'analysis' },
  { id: 'authentication', text: 'authentication' },
  { id: 'recording', text: 'recording' },
  { id: 'storage', text: 'storage' },
  { id: 'sip', text: 'sip' },
  { id: 'webrtc', text: 'webrtc' },
  { id: 'usage', text: 'usage' },
  { id: 'log', text: 'log' },
  { id: 'metric', text: 'metric' },
  { id: 'metadata', text: 'metadata' },
];

export const ALL_EVENT_OPTION: FilterOption = {
  id: 'all',
  text: 'All events',
};

export const EVENTS_BY_COMPONENT: Record<string, string[]> = {
  call: [
    'call.status',
    'call.received',
    'call.ringing',
    'call.started',
    'call.media_started',
    'call.hangup',
    'call.ended',
    'call.failed',
    'call.cancelled',
    'call.outbound_requested',
    'call.outbound_dispatched',
    'call.outbound_dispatch_failed',
    'call.provider_answered',
    'call.session_connected',
    'call.assistant_loaded',
    'call.conversation_created',
    'call.context_saved',
  ],
  conversation: [
    'conversation.begin',
    'conversation.resume',
    'conversation.initializing',
    'conversation.initialized',
    'conversation.authentication_started',
    'conversation.completed',
    'conversation.cleanup',
    'conversation.error',
    'conversation.agent_state_changed',
    'conversation.mode_switch_failed',
  ],
  turn: ['turn.change'],
  stt: [
    'stt.interim',
    'stt.completed',
    'stt.low_confidence',
    'stt.closed',
    'stt.error',
  ],
  tts: [
    'tts.speaking',
    'tts.completed',
    'tts.discard_chunk',
    'tts.interrupted',
    'tts.closed',
    'tts.error',
  ],
  agent: [
    'agent.started',
    'agent.completed',
    'agent.discarded',
    'agent.error',
    'agent.transition.triggered',
    'agent.transition.matched',
    'agent.transition.missing_edge',
  ],
  vad: ['vad.speech_started', 'vad.speech_ended', 'vad.closed', 'vad.error'],
  eos: ['eos.started', 'eos.completed', 'eos.closed'],
  denoise: ['denoise.closed', 'denoise.error'],
  tool: ['tool.call_started', 'tool.call_completed', 'tool.call_failed'],
  recording: ['recording.started', 'recording.completed'],
  sip: [
    'sip.transfer_requested',
    'sip.transferring',
    'sip.register_started',
    'sip.register_active',
    'sip.register_failed',
    'sip.register_renewed',
    'sip.register_renewal_failed',
    'sip.register_expired',
    'sip.unregister_failed',
  ],
  webrtc: [
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
  ],
};

export const ALL_EVENT_OPTIONS: FilterOption[] = Array.from(
  new Set(Object.values(EVENTS_BY_COMPONENT).flat()),
)
  .sort()
  .map(eventName => ({ id: eventName, text: eventName }));

export const itemToString = (item: FilterOption | null) => item?.text || '';

export const getOptionById = (
  options: FilterOption[],
  id: string,
): FilterOption => options.find(option => option.id === id) || options[0];

export const getEventOptionsForComponent = (
  componentId: string,
): FilterOption[] =>
  componentId === 'all'
    ? [ALL_EVENT_OPTION, ...ALL_EVENT_OPTIONS]
    : [
        ALL_EVENT_OPTION,
        ...(EVENTS_BY_COMPONENT[componentId] || []).map(eventName => ({
          id: eventName,
          text: eventName,
        })),
      ];
