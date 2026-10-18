import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { setRangeValue } from '@/test-utils/set-range-value';
import { Metadata } from '@rapidaai/react';
import { ConfigureAudioOutputProvider } from '../configure-audio-output';

const mockGetDefaultSpeakerConfig = jest.fn();
const mockGetDefaultTextToSpeechIfInvalid = jest.fn();

jest.mock('@/utils', () => ({
  cn: (...inputs: any[]) => inputs.filter(Boolean).join(' '),
}));

jest.mock('@/app/components/layout/blocks/section-divider', () => ({
  SectionDivider: ({ label }: { label: string }) => <div>{label}</div>,
}));

jest.mock('@/app/components/domain/providers/text-to-speech', () => ({
  TextToSpeechProvider: ({
    onChangeProvider,
  }: {
    onChangeProvider: (provider: string) => void;
  }) => (
    <button onClick={() => onChangeProvider('openai')} type="button">
      change tts
    </button>
  ),
}));

jest.mock('@/providers', () => ({
  PRONUNCIATION_DICTIONARIES: ['medical', 'retail'],
  CONJUNCTION_BOUNDARIES: ['and', 'or'],
}));

jest.mock('@/app/components/ui/primitives/form', () => {
  const React = require('react');
  return {
    TextInput: ({ id, labelText, value, onChange }: any) =>
      React.createElement(
        'div',
        null,
        labelText
          ? React.createElement('label', { htmlFor: id }, labelText)
          : null,
        React.createElement('input', {
          id,
          value: value ?? '',
          onChange,
          'data-testid': id,
        }),
      ),
  };
});

jest.mock('@carbon/react', () => {
  const React = require('react');
  return {
    MultiSelect: ({ id, titleText, selectedItems = [], onChange }: any) =>
      React.createElement(
        'div',
        null,
        titleText ? React.createElement('div', null, titleText) : null,
        React.createElement('input', {
          'data-testid': id,
          value: selectedItems.map((i: any) => i.id).join('<|||>'),
          onChange: (e: any) =>
            onChange({
              selectedItems: e.target.value
                .split('<|||>')
                .filter(Boolean)
                .map((id: string) => ({ id, label: id })),
            }),
        }),
      ),
    Dropdown: ({ id, titleText, items = [], selectedItem, onChange }: any) =>
      React.createElement(
        'div',
        null,
        titleText ? React.createElement('div', null, titleText) : null,
        React.createElement(
          'select',
          {
            'data-testid': id,
            value: selectedItem?.id ?? '',
            onChange: (e: any) =>
              onChange({
                selectedItem:
                  items.find((item: any) => item.id === e.target.value) || null,
              }),
          },
          items.map((item: any) =>
            React.createElement(
              'option',
              { key: item.id, value: item.id },
              item.label,
            ),
          ),
        ),
      ),
    Slider: ({ id, value = 0, onChange, min, max, step }: any) =>
      React.createElement('input', {
        id,
        type: 'range',
        min,
        max,
        step,
        value,
        'data-testid': id,
        onChange: (e: any) => onChange?.({ value: Number(e.target.value) }),
      }),
  };
});

jest.mock('@/app/components/domain/providers/text-to-speech/provider', () => ({
  GetDefaultSpeakerConfig: (...args: any[]) =>
    mockGetDefaultSpeakerConfig(...args),
  GetDefaultTextToSpeechIfInvalid: (...args: any[]) =>
    mockGetDefaultTextToSpeechIfInvalid(...args),
}));

const createMetadata = (key: string, value: string): Metadata => {
  const metadata = new Metadata();
  metadata.setKey(key);
  metadata.setValue(value);
  return metadata;
};

describe('ConfigureAudioOutputProvider design integration', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('keeps only shared speaker advanced keys when switching TTS provider', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const inputParameters = [
      createMetadata('rapida.credential_id', 'cred-1'),
      createMetadata('speaker.model', 'sonic-2'),
      createMetadata('speaker.voice', 'f6f3f5f8'),
      createMetadata('speaker.language', 'en'),
      createMetadata('speaker.ambient', 'office'),
      createMetadata('speaker.ambient_volume', '24'),
      createMetadata('speaker.conjunction.boundaries', 'and<|||>or'),
      createMetadata('speaker.conjunction.break', '240'),
      createMetadata('speaker.pronunciation.dictionaries', 'medical'),
    ];

    const speakerDefaults = [
      createMetadata('speaker.model', 'gpt-4o-mini-tts'),
    ];
    const ttsDefaults = [createMetadata('speaker.voice', 'alloy')];
    mockGetDefaultSpeakerConfig.mockReturnValue(speakerDefaults);
    mockGetDefaultTextToSpeechIfInvalid.mockReturnValue(ttsDefaults);

    const setAudioOutputConfig = jest.fn();
    render(
      <ConfigureAudioOutputProvider
        audioOutputConfig={{
          provider: 'cartesia',
          parameters: inputParameters,
        }}
        setAudioOutputConfig={setAudioOutputConfig}
      />,
    );

    await user.click(screen.getByRole('button', { name: 'change tts' }));

    const keptParams = mockGetDefaultSpeakerConfig.mock
      .calls[0][0] as Metadata[];
    expect(keptParams.map(p => `${p.getKey()}=${p.getValue()}`).sort()).toEqual(
      [
        'speaker.ambient=office',
        'speaker.ambient_volume=24',
        'speaker.conjunction.boundaries=and<|||>or',
        'speaker.conjunction.break=240',
        'speaker.pronunciation.dictionaries=medical',
      ].sort(),
    );
    expect(mockGetDefaultTextToSpeechIfInvalid).toHaveBeenCalledWith(
      'openai',
      speakerDefaults,
    );
    expect(setAudioOutputConfig).toHaveBeenCalledWith({
      provider: 'openai',
      parameters: ttsDefaults,
    });
  });

  it('updates pronunciation, conjunction boundaries and pause from advanced settings', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const setAudioOutputConfig = jest.fn();
    render(
      <ConfigureAudioOutputProvider
        audioOutputConfig={{
          provider: 'cartesia',
          parameters: [createMetadata('speaker.model', 'sonic-2')],
        }}
        setAudioOutputConfig={setAudioOutputConfig}
      />,
    );

    await user.click(
      screen.getByRole('button', { name: /show advanced settings/i }),
    );

    await user.click(screen.getByTestId('pronunciation-dictionaries'));
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('medical<|||>retail'));
    await user.selectOptions(
      screen.getByTestId('ambient-sound'),
      String('cafe'),
    );
    const ambientVolume = screen.getByTestId('ambient-volume');
    await user.click(ambientVolume);
    setRangeValue(ambientVolume as HTMLInputElement, 34);
    await user.click(screen.getByTestId('conjunction-boundaries'));
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('and<|||>or'));
    const conjunctionBreak = screen.getByTestId('conjunction-break');
    await user.click(conjunctionBreak);
    setRangeValue(conjunctionBreak as HTMLInputElement, 280);

    const allMaps = setAudioOutputConfig.mock.calls.map(call =>
      Object.fromEntries(
        ((call[0]?.parameters || []) as Metadata[]).map(p => [
          p.getKey(),
          p.getValue(),
        ]),
      ),
    );

    expect(allMaps.some(values => values['speaker.ambient'] === 'cafe')).toBe(
      true,
    );
    expect(
      allMaps.some(values => values['speaker.ambient_volume'] === '34'),
    ).toBe(true);
    expect(
      allMaps.some(
        values =>
          values['speaker.pronunciation.dictionaries'] === 'medical<|||>retail',
      ),
    ).toBe(true);
    expect(
      allMaps.some(
        values => values['speaker.conjunction.boundaries'] === 'and<|||>or',
      ),
    ).toBe(true);
    expect(
      allMaps.some(values => values['speaker.conjunction.break'] === '280'),
    ).toBe(true);
  });
});
