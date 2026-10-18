import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import { Metadata } from '@rapidaai/react';
import { TextToSpeechProvider } from '../text-to-speech';
import {
  GetDefaultTextToSpeechIfInvalid,
  TextToSpeechConfigComponent,
  ValidateTextToSpeechIfInvalid,
} from '../text-to-speech/provider';

const mockGetDefaultsFromConfig = jest.fn();
const mockLoadProviderConfig = jest.fn();
const mockValidateFromConfig = jest.fn();

jest.mock('@/providers', () => ({
  TEXT_TO_SPEECH_PROVIDER: [
    { code: 'openai', featureList: ['tts'], name: 'OpenAI' },
    { code: 'custom-tts', featureList: ['tts'], name: 'Custom TTS' },
  ],
}));

jest.mock('@/providers/config-loader', () => ({
  loadProviderConfig: (...args: unknown[]) => mockLoadProviderConfig(...args),
}));

jest.mock('@/providers/config-defaults', () => ({
  getDefaultsFromConfig: (...args: unknown[]) =>
    mockGetDefaultsFromConfig(...args),
  validateFromConfig: (...args: unknown[]) => mockValidateFromConfig(...args),
}));

jest.mock('@/app/components/domain/providers/config-renderer', () => ({
  ConfigRenderer: ({ category, provider }: any) => (
    <div data-testid={`config-${category}`}>{provider}</div>
  ),
}));

jest.mock('@/app/components/domain/providers/help-label', () => ({
  HelpToggletip: ({ helpText, label }: any) =>
    helpText ? <span aria-label={`${label} help`}>{helpText}</span> : null,
}));

jest.mock('@/app/components/domain/dropdowns/credential-dropdown', () => ({
  CredentialDropdown: ({
    currentCredential,
    onChangeCredential,
    provider,
  }: any) => (
    <button
      data-current={currentCredential}
      data-provider={provider}
      type="button"
      onClick={() => onChangeCredential({ getId: () => 'credential-next' })}
    >
      Pick credential
    </button>
  ),
}));

jest.mock('@/app/components/ui/primitives', () => ({
  Stack: ({ children }: any) => <div>{children}</div>,
}));

jest.mock('@carbon/react', () => ({
  Dropdown: ({
    id,
    itemToString,
    label,
    items = [],
    onChange,
    selectedItem,
    titleText,
  }: any) => (
    <div>
      {titleText ? <span>{titleText}</span> : null}
      <span data-testid={`${id}-empty-name`}>{itemToString?.(null)}</span>
      <select
        aria-label={label}
        value={selectedItem?.code ?? ''}
        onChange={event => {
          const selected = items.find(
            (item: any) => item.code === event.target.value,
          );
          onChange?.({ selectedItem: selected ?? null });
        }}
      >
        <option value="">Select</option>
        {items.map((item: any) => (
          <option key={item.code} value={item.code}>
            {itemToString(item)}
          </option>
        ))}
      </select>
    </div>
  ),
}));

const createMetadata = (key: string, value: string): Metadata => {
  const metadata = new Metadata();
  metadata.setKey(key);
  metadata.setValue(value);
  return metadata;
};

const getMetadataValue = (parameters: Metadata[], key: string) =>
  parameters.find(parameter => parameter.getKey() === key)?.getValue();

beforeEach(() => {
  mockGetDefaultsFromConfig.mockReset();
  mockLoadProviderConfig.mockReset();
  mockValidateFromConfig.mockReset();
  mockGetDefaultsFromConfig.mockImplementation(
    (_config, _category, current: Metadata[]) => current,
  );
});

describe('TextToSpeechProvider', () => {
  it('renders the provider selector and ignores empty selections', async () => {
    const user = userEvent.setup();
    const onChangeProvider = jest.fn();

    render(
      <TextToSpeechProvider
        provider=""
        parameters={[]}
        onChangeProvider={onChangeProvider}
        onChangeParameter={() => undefined}
      />,
    );

    expect(screen.getByText('Voice output provider')).toBeInTheDocument();
    expect(
      screen.getByLabelText('Voice output provider help'),
    ).toHaveTextContent(
      'Select a text-to-speech provider for generated assistant audio.',
    );
    expect(screen.getByTestId('tts-provider-empty-name')).toBeEmptyDOMElement();
    expect(screen.queryByText('Pick credential')).not.toBeInTheDocument();

    const select = screen.getByRole('combobox', {
      name: 'Select voice output provider',
    });
    await user.selectOptions(select, String('openai'));
    await user.selectOptions(select, String(''));

    expect(onChangeProvider).toHaveBeenCalledWith('openai');
    expect(onChangeProvider).toHaveBeenCalledTimes(1);
  });

  it('adds and replaces credential metadata for a selected provider', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onChangeParameter = jest.fn();
    mockLoadProviderConfig.mockReturnValue({ tts: { parameters: [] } });

    const { rerender } = render(
      <TextToSpeechProvider
        provider="openai"
        parameters={[createMetadata('speak.voice.id', 'voice-1')]}
        onChangeProvider={() => undefined}
        onChangeParameter={onChangeParameter}
      />,
    );

    expect(screen.getByTestId('config-tts')).toHaveTextContent('openai');

    await user.click(screen.getByText('Pick credential'));

    let nextParameters = onChangeParameter.mock.calls[0][0] as Metadata[];
    expect(getMetadataValue(nextParameters, 'speak.voice.id')).toBe('voice-1');
    expect(getMetadataValue(nextParameters, 'rapida.credential_id')).toBe(
      'credential-next',
    );

    rerender(
      <TextToSpeechProvider
        provider="custom-tts"
        parameters={[
          createMetadata('rapida.credential_id', 'credential-old'),
          createMetadata('speak.voice.id', 'voice-2'),
        ]}
        onChangeProvider={() => undefined}
        onChangeParameter={onChangeParameter}
      />,
    );

    const credentialButton = screen.getByText('Pick credential');
    expect(credentialButton).toHaveAttribute('data-current', 'credential-old');
    expect(credentialButton).toHaveAttribute('data-provider', 'custom-tts');

    await user.click(credentialButton);

    nextParameters = onChangeParameter.mock.calls[
      onChangeParameter.mock.calls.length - 1
    ][0] as Metadata[];
    expect(getMetadataValue(nextParameters, 'rapida.credential_id')).toBe(
      'credential-next',
    );
    expect(getMetadataValue(nextParameters, 'speak.voice.id')).toBe('voice-2');
  });

  it('hydrates defaults and validates credentials through provider config', () => {
    const defaults = [createMetadata('speak.voice.id', 'alloy')];
    const config = { tts: { parameters: [] } };
    mockLoadProviderConfig.mockReturnValue(config);
    mockGetDefaultsFromConfig.mockReturnValue(defaults);
    mockValidateFromConfig.mockReturnValue(undefined);

    expect(GetDefaultTextToSpeechIfInvalid('openai', [])).toBe(defaults);
    expect(mockGetDefaultsFromConfig).toHaveBeenCalledWith(
      config,
      'tts',
      [],
      'openai',
    );

    expect(
      ValidateTextToSpeechIfInvalid('openai', [
        createMetadata('rapida.credential_id', 'credential-1'),
      ]),
    ).toBeUndefined();
    expect(mockValidateFromConfig).toHaveBeenCalledWith(
      config,
      'tts',
      'openai',
      [expect.any(Metadata)],
    );
  });

  it('returns validation errors for missing or mismatched provider credentials', () => {
    mockLoadProviderConfig.mockReturnValue({ tts: { parameters: [] } });
    mockValidateFromConfig.mockReturnValue(undefined);

    expect(ValidateTextToSpeechIfInvalid('openai', [], [])).toBe(
      'Please provide a valid openai credential.',
    );
    expect(
      ValidateTextToSpeechIfInvalid(
        'openai',
        [createMetadata('rapida.credential_id', 'credential-2')],
        ['credential-1'],
      ),
    ).toBe('Please select a valid openai credential.');

    mockValidateFromConfig.mockReturnValue('Missing voice');
    expect(
      ValidateTextToSpeechIfInvalid(
        'openai',
        [createMetadata('rapida.credential_id', 'credential-1')],
        ['credential-1'],
      ),
    ).toBe('Missing voice');
  });

  it('accepts VaultCredential objects as selected provider credentials', () => {
    mockLoadProviderConfig.mockReturnValue({ tts: { parameters: [] } });
    mockValidateFromConfig.mockReturnValue(undefined);

    expect(
      ValidateTextToSpeechIfInvalid(
        'openai',
        [createMetadata('rapida.credential_id', 'credential-1')],
        [{ getId: () => 'credential-1' } as any],
      ),
    ).toBeUndefined();
  });

  it('returns existing parameters and no validation error when config is missing', () => {
    const parameters = [createMetadata('custom.key', 'custom')];
    mockLoadProviderConfig.mockReturnValue({});

    expect(GetDefaultTextToSpeechIfInvalid('unknown', parameters)).toBe(
      parameters,
    );
    expect(ValidateTextToSpeechIfInvalid('unknown', [])).toBeUndefined();
  });

  it('renders text-to-speech config only when provider config exists', () => {
    const props = {
      provider: 'openai',
      parameters: [],
      onChangeProvider: () => undefined,
      onChangeParameter: () => undefined,
    };
    mockLoadProviderConfig.mockReturnValue({});

    const { rerender } = render(<TextToSpeechConfigComponent {...props} />);
    expect(screen.queryByTestId('config-tts')).not.toBeInTheDocument();

    mockLoadProviderConfig.mockReturnValue({ tts: { parameters: [] } });
    rerender(<TextToSpeechConfigComponent {...props} />);

    expect(screen.getByTestId('config-tts')).toHaveTextContent('openai');
  });
});
