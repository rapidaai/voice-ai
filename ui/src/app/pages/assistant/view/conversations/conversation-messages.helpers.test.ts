import { Metric } from '@rapidaai/react';
import {
  formatLatency,
  getRoleVisual,
} from '@/app/pages/assistant/view/conversations/conversation-messages.helpers';

const metric = (name: string, value: string): Metric => {
  const m = new Metric();
  m.setName(name);
  m.setValue(value);
  return m;
};

describe('conversation message helpers', () => {
  it('returns expected visual metadata for known and unknown roles', () => {
    expect(getRoleVisual('assistant').label).toBe('Assistant');
    expect(getRoleVisual('rapida').label).toBe('Assistant');
    expect(getRoleVisual('user').shortLabel).toBe('U');
    expect(getRoleVisual('custom').shortLabel).toBe('C');
  });

  it('formats latency from TIME_TAKEN metric and handles missing metrics', () => {
    expect(formatLatency([])).toBe('0 ms');
    expect(formatLatency([metric('time_taken', '2500000')])).toBe('2.50 ms');
    expect(formatLatency([metric('time_taken', '1500000000')])).toBe('1.50 s');
  });
});
