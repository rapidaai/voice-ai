import { TELEPHONY_PROVIDER } from '../index';
import { loadProviderConfig } from '../config-loader';

const EXPECTED_TELEPHONY_CODES = [
  'vonage',
  'exotel',
  'asterisk',
  'sip',
  'telnyx',
  'twilio',
  'vobiz',
  'plivo',
];

describe('Telephony providers registry', () => {
  it('includes only supported telephony providers', () => {
    const codes = TELEPHONY_PROVIDER.map(p => p.code).sort();
    expect(codes).toEqual([...EXPECTED_TELEPHONY_CODES].sort());
  });

  it('all telephony providers carry telephony feature', () => {
    TELEPHONY_PROVIDER.forEach(provider => {
      expect(provider.featureList).toContain('telephony');
    });
  });

  it('all config-backed telephony providers have telephony config with phone field', () => {
    TELEPHONY_PROVIDER.filter(provider => provider.code !== 'telnyx').forEach(
      provider => {
        const config = loadProviderConfig(provider.code);
        expect(config).not.toBeNull();
        expect(config?.telephony).toBeDefined();
        expect(config?.telephony?.parameters).toBeDefined();
        const hasPhone = config?.telephony?.parameters.some(
          p => p.key === 'phone',
        );
        expect(hasPhone).toBe(true);
      },
    );
  });

  it('telnyx currently has no split telephony config', () => {
    expect(loadProviderConfig('telnyx')).toBeNull();
  });
});
