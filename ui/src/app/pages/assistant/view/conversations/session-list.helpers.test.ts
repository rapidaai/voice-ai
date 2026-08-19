import { AssistantConversation, Metadata, Metric } from '@rapidaai/react';
import {
  getChannelValue,
  getDisconnectReasonValue,
  getDurationBreakdownRows,
} from './session-list.helpers';

const metadata = (key: string, value: string): Metadata => {
  const m = new Metadata();
  m.setKey(key);
  m.setValue(value);
  return m;
};

const metric = (name: string, value: string): Metric => {
  const m = new Metric();
  m.setName(name);
  m.setValue(value);
  return m;
};

const conversation = (): AssistantConversation => new AssistantConversation();

describe('session list helpers', () => {
  it('reads channel metadata with WebRTC fallback', () => {
    const conv = conversation();
    conv.addMetadata(metadata('client.channel', 'phone'));

    expect(getChannelValue(conv)).toBe('phone');
    expect(getChannelValue(conversation())).toBe('webrtc');
  });

  it('reads disconnect reason metadata with unknown fallback for blank values', () => {
    const conv = conversation();
    conv.addMetadata(metadata('disconnect_reason', '  '));

    expect(getDisconnectReasonValue(conv)).toBe('unknown');
  });

  it('converts duration breakdown metrics to seconds', () => {
    const conv = conversation();
    conv.addMetrics(metric('conversation.duration_ms', '6575'));
    conv.addMetrics(metric('call.duration_ms', '7000'));
    conv.addMetrics(metric('tts.duration_ms', '5456'));
    conv.addMetrics(metric('stt.duration_ms', '5481'));

    expect(getDurationBreakdownRows(conv)).toEqual([
      { key: 'conversation.duration_ms', value: '6.58' },
      { key: 'call.duration_ms', value: '7.00' },
      { key: 'tts.duration_ms', value: '5.46' },
      { key: 'stt.duration_ms', value: '5.48' },
    ]);
  });
});
