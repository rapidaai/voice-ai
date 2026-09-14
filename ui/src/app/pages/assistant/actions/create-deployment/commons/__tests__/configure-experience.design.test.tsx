import React from 'react';
import { render } from '@testing-library/react';
import developmentConfig from '@/configs/config.development.json';
import { ThemeProvider } from '@/theme/theme-provider';
import { ThemeManifest } from '@/theme/types';
import {
  ConfigureExperience,
  DEFAULT_IDEAL_TIMEOUT,
  DEFAULT_IDLE_PROMPT_COUNT,
  validateIdleTimeoutConfig,
} from '../configure-experience';

const mockSlider = jest.fn();
const theme = developmentConfig.theme as unknown as ThemeManifest;

jest.mock('@/app/components/carbon/form', () => ({
  Stack: ({ children }: any) => <div>{children}</div>,
  TextArea: ({ labelText, value }: any) => (
    <textarea aria-label={labelText} value={value} readOnly />
  ),
}));

jest.mock('@carbon/icons-react', () => ({
  Information: () => null,
}));

jest.mock('@carbon/react', () => ({
  Button: ({ children }: any) => <button>{children}</button>,
  ComboBox: ({ 'aria-label': ariaLabel, selectedItem }: any) => (
    <input aria-label={ariaLabel} value={selectedItem || ''} readOnly />
  ),
  FormLabel: ({ children, id }: any) => <label id={id}>{children}</label>,
  Slider: (props: any) => {
    mockSlider(props);
    return (
      <input
        aria-label={props.labelText}
        data-max={props.max}
        data-min={props.min}
        data-value={props.value}
        readOnly
      />
    );
  },
  Toggletip: ({ children }: any) => <div>{children}</div>,
  ToggletipActions: ({ children }: any) => <div>{children}</div>,
  ToggletipButton: ({ children, label }: any) => (
    <button aria-label={label}>{children}</button>
  ),
  ToggletipContent: ({ children }: any) => <div>{children}</div>,
  Toggle: ({ toggled }: any) => (
    <input aria-label="toggle" checked={toggled} readOnly type="checkbox" />
  ),
}));

describe('ConfigureExperience', () => {
  beforeEach(() => {
    Object.defineProperty(window, 'matchMedia', {
      configurable: true,
      value: jest.fn().mockReturnValue({
        matches: false,
        addEventListener: jest.fn(),
        removeEventListener: jest.fn(),
      }),
    });
    mockSlider.mockClear();
  });

  it('defaults idle timeout to 10 seconds within backend validation limits', () => {
    render(
      <ThemeProvider theme={theme}>
        <ConfigureExperience
          experienceConfig={{}}
          setExperienceConfig={jest.fn()}
        />
      </ThemeProvider>,
    );

    const idleTimeoutSlider = mockSlider.mock.calls
      .map(([props]) => props)
      .find(props => props.id === 'experience-idle-timeout');

    expect(DEFAULT_IDEAL_TIMEOUT).toBe('10');
    expect(idleTimeoutSlider).toMatchObject({
      min: 5,
      max: 120,
      value: 10,
    });
  });

  it('shows zero as unlimited without replacing the saved count', () => {
    render(
      <ThemeProvider theme={theme}>
        <ConfigureExperience
          experienceConfig={{
            idealTimeout: '30',
            idleTimeoutBackoffTimes: '0',
          }}
          setExperienceConfig={jest.fn()}
        />
      </ThemeProvider>,
    );

    expect(DEFAULT_IDLE_PROMPT_COUNT).toBe('2');
    expect(mockSlider.mock.calls.map(([props]) => props)).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ id: 'experience-idle-timeout', value: 30 }),
        expect.objectContaining({
          id: 'experience-backoff',
          labelText: 'Idle Prompt Count (0 = Unlimited)',
          min: 0,
          max: 5,
          value: 0,
        }),
      ]),
    );
  });

  it.each([
    {},
    { idealTimeout: '', idleTimeoutBackoffTimes: '' },
    { idealTimeout: ' ', idleTimeoutBackoffTimes: ' ' },
    { idealTimeout: ' 10 ', idleTimeoutBackoffTimes: ' 2 ' },
    { idealTimeout: '5', idleTimeoutBackoffTimes: '0' },
    { idealTimeout: '120', idleTimeoutBackoffTimes: '5' },
  ])('accepts valid idle settings and defaults: %j', config => {
    expect(validateIdleTimeoutConfig(config)).toBe('');
  });

  it.each([
    '0',
    '4',
    '121',
    '-10',
    '5.5',
    '10seconds',
    'NaN',
    'Infinity',
    '1e1',
  ])('rejects invalid idle timeout %s', idealTimeout => {
    expect(validateIdleTimeoutConfig({ idealTimeout })).toBe(
      'Idle silence timeout must be a whole number from 5 to 120 seconds.',
    );
  });

  it.each(['-1', '6', '1.5', '2prompts', 'NaN', 'Infinity', '1e0'])(
    'rejects invalid idle count %s',
    idleTimeoutBackoffTimes => {
      expect(validateIdleTimeoutConfig({ idleTimeoutBackoffTimes })).toBe(
        'Idle prompt count must be a whole number from 0 to 5. Use 0 for unlimited prompts.',
      );
    },
  );
});
