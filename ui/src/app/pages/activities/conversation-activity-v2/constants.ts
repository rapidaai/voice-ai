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
  { id: 'stt_init_ms', text: 'stt_init_ms' },
  { id: 'stt_latency_ms', text: 'stt_latency_ms' },
  { id: 'tts_init_ms', text: 'tts_init_ms' },
  { id: 'tts_latency_ms', text: 'tts_latency_ms' },
  { id: 'vad.init_ms', text: 'vad.init_ms' },
  { id: 'eos.init_ms', text: 'eos.init_ms' },
  { id: 'eos_latency_ms', text: 'eos_latency_ms' },
  { id: 'eos_text_to_trigger_ms', text: 'eos_text_to_trigger_ms' },
  { id: 'eos_word_count', text: 'eos_word_count' },
  { id: 'eos_char_count', text: 'eos_char_count' },
  { id: 'eos_confidence', text: 'eos_confidence' },
  { id: 'denoise.init_ms', text: 'denoise.init_ms' },
  { id: 'llm_init_ms', text: 'llm_init_ms' },
  { id: 'storage.init_ms', text: 'storage.init_ms' },
  { id: 'analysis.init_ms', text: 'analysis.init_ms' },
  { id: 'authentication.init_ms', text: 'authentication.init_ms' },
  { id: 'authentication.latency_ms', text: 'authentication.latency_ms' },
  { id: 'recording.init_ms', text: 'recording.init_ms' },
  { id: 'knowledge_latency_ms', text: 'knowledge_latency_ms' },
  { id: 'llm_latency_ms', text: 'llm_latency_ms' },
  { id: 'llm_input_char_count', text: 'llm_input_char_count' },
  { id: 'llm_history_count', text: 'llm_history_count' },
  { id: 'llm_response_char_count', text: 'llm_response_char_count' },
  { id: 'llm_error', text: 'llm_error' },
  { id: 'stt_error', text: 'stt_error' },
  { id: 'tts_error', text: 'tts_error' },
  { id: 'discarded_tts_chunk', text: 'discarded_tts_chunk' },
  { id: 'discarded_tts', text: 'discarded_tts' },
  { id: 'time_taken', text: 'time_taken' },
  { id: 'agent_time_taken', text: 'agent_time_taken' },
  { id: 'agent_status', text: 'agent_status' },
  { id: 'agent_llm_request_id', text: 'agent_llm_request_id' },
  { id: 'agent_input_token', text: 'agent_input_token' },
  { id: 'agent_output_token', text: 'agent_output_token' },
  { id: 'agent_total_token', text: 'agent_total_token' },
  { id: 'agent_cached_content_token', text: 'agent_cached_content_token' },
  { id: 'agent_cost', text: 'agent_cost' },
  { id: 'agent_input_cost', text: 'agent_input_cost' },
  { id: 'agent_output_cost', text: 'agent_output_cost' },
  { id: 'agent_token_pre_second', text: 'agent_token_pre_second' },
  { id: 'agent.ttft_ms', text: 'agent.ttft_ms' },
  { id: 'agent_provider_total_time', text: 'agent_provider_total_time' },
  { id: 'agent_provider_generate_time', text: 'agent_provider_generate_time' },
  { id: 'agent_token_count', text: 'agent_token_count' },
  { id: 'input_token', text: 'input_token' },
  { id: 'output_token', text: 'output_token' },
  { id: 'total_token', text: 'total_token' },
  { id: 'cached_content_token', text: 'cached_content_token' },
  { id: 'cost', text: 'cost' },
  { id: 'input_cost', text: 'input_cost' },
  { id: 'output_cost', text: 'output_cost' },
  { id: 'llm_request_id', text: 'llm_request_id' },
  { id: 'token_pre_second', text: 'token_pre_second' },
  { id: 'time_to_first_token', text: 'time_to_first_token' },
  { id: 'provider_total_time', text: 'provider_total_time' },
  { id: 'provider_generate_time', text: 'provider_generate_time' },
];

export const COMPONENT_OPTIONS: FilterOption[] = [
  { id: 'all', text: 'All components' },
  { id: 'call', text: 'call' },
  { id: 'conversation', text: 'conversation' },
  { id: 'turn', text: 'turn' },
  { id: 'stt', text: 'stt' },
  { id: 'tts', text: 'tts' },
  { id: 'llm', text: 'llm' },
  { id: 'agentflow', text: 'agentflow' },
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
    'tts.discarded',
    'tts.interrupted',
    'tts.closed',
    'tts.error',
  ],
  llm: ['llm.started', 'llm.completed', 'llm.discarded', 'llm.error'],
  agentflow: [
    'agentflow.transition.triggered',
    'agentflow.transition.matched',
    'agentflow.transition.missing_edge',
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
