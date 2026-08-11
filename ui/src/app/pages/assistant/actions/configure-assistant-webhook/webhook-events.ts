export type WebhookEventGroup = 'Call' | 'Conversation' | 'WebRTC';

export type WebhookEventOption = {
  id: string;
  name: string;
  description: string;
  group: WebhookEventGroup;
};

export const webhookEvents: WebhookEventOption[] = [
  {
    id: 'call.received',
    name: 'call.received',
    description: 'Triggered when an inbound call reaches the assistant.',
    group: 'Call',
  },
  {
    id: 'call.ringing',
    name: 'call.ringing',
    description: 'Triggered when an outbound call is ringing.',
    group: 'Call',
  },
  {
    id: 'call.provider_answered',
    name: 'call.provider_answered',
    description: 'Triggered when the telephony provider answers the call.',
    group: 'Call',
  },
  {
    id: 'call.outbound_requested',
    name: 'call.outbound_requested',
    description: 'Triggered when an outbound call is requested.',
    group: 'Call',
  },
  {
    id: 'call.outbound_dispatched',
    name: 'call.outbound_dispatched',
    description:
      'Triggered when an outbound call request is sent to the provider.',
    group: 'Call',
  },
  {
    id: 'call.started',
    name: 'call.started',
    description: 'Triggered when the call session starts.',
    group: 'Call',
  },
  {
    id: 'call.ended',
    name: 'call.ended',
    description: 'Triggered when the call session finishes.',
    group: 'Call',
  },
  {
    id: 'call.failed',
    name: 'call.failed',
    description: 'Triggered when the call fails.',
    group: 'Call',
  },
  {
    id: 'conversation.begin',
    name: 'conversation.begin',
    description: 'Triggered when a new conversation begins.',
    group: 'Conversation',
  },
  {
    id: 'conversation.resume',
    name: 'conversation.resume',
    description: 'Triggered when an existing conversation resumes.',
    group: 'Conversation',
  },
  {
    id: 'conversation.completed',
    name: 'conversation.completed',
    description: 'Triggered when a conversation completes.',
    group: 'Conversation',
  },
  {
    id: 'conversation.error',
    name: 'conversation.error',
    description: 'Triggered when a conversation fails with an error.',
    group: 'Conversation',
  },
  {
    id: 'webrtc.connected',
    name: 'webrtc.connected',
    description: 'Triggered when the WebRTC media connection is established.',
    group: 'WebRTC',
  },
  {
    id: 'webrtc.audio_track_received',
    name: 'webrtc.audio_track_received',
    description: 'Triggered when the remote WebRTC audio track is received.',
    group: 'WebRTC',
  },
  {
    id: 'webrtc.reconnecting',
    name: 'webrtc.reconnecting',
    description: 'Triggered when the WebRTC media connection is reconnecting.',
    group: 'WebRTC',
  },
  {
    id: 'webrtc.failed',
    name: 'webrtc.failed',
    description: 'Triggered when the WebRTC media connection fails.',
    group: 'WebRTC',
  },
  {
    id: 'webrtc.disconnected',
    name: 'webrtc.disconnected',
    description: 'Triggered when the WebRTC media connection disconnects.',
    group: 'WebRTC',
  },
];
