import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import { Metadata } from '@rapidaai/react';
import { SpeechToTextProvider } from '../speech-to-text';
import {
  GetDefaultSpeechToTextIfInvalid,
  SpeechToTextConfigComponent,
  ValidateSpeechToTextIfInvalid,
} from '../speech-to-text/provider';

const mockGetDefaultsFromConfig = jest.fn();
const mockLoadProviderConfig = jest.fn();
const mockValidateFromConfig = jest.fn();

jest.mock('@/providers', () => ({
  SPEECH_TO_TEXT_PROVIDER: [
    { code: 'openai', featureList: ['stt'], name: 'OpenAI' },
    { code: 'custom-stt', featureList: ['stt'], name: 'Custom STT' },
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

jest.mock('@/app/components/domain/providers/end-of-speech/provider', () => ({
  GetDefaultEOSConfig: (_provider: string, parameters: Metadata[]) =>
    parameters,
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

describe('SpeechToTextProvider', () => {
  it('renders the provider selector and ignores empty selections', async () => {
    const user = userEvent.setup();
    const onChangeProvider = jest.fn();

    render(
      <SpeechToTextProvider
        provider=""
        parameters={[]}
        onChangeProvider={onChangeProvider}
        onChangeParameter={() => undefined}
      />,
    );

    expect(screen.getByText('Voice input provider')).toBeInTheDocument();
    expect(
      screen.getByLabelText('Voice input provider help'),
    ).toHaveTextContent(
      'Select a speech-to-text provider for assistant microphone transcription.',
    );
    expect(screen.getByTestId('stt-provider-empty-name')).toBeEmptyDOMElement();
    expect(screen.queryByText('Pick credential')).not.toBeInTheDocument();

    const select = screen.getByRole('combobox', {
      name: 'Select voice input provider',
    });
    await user.selectOptions(select, String('openai'));
    await user.selectOptions(select, String(''));

    expect(onChangeProvider).toHaveBeenCalledWith('openai');
    expect(onChangeProvider).toHaveBeenCalledTimes(1);
  });

  it('adds and replaces credential metadata for a selected provider', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onChangeParameter = jest.fn();
    mockLoadProviderConfig.mockReturnValue({ stt: { parameters: [] } });

    const { rerender } = render(
      <SpeechToTextProvider
        provider="openai"
        parameters={[createMetadata('listen.model', 'gpt-4o-mini-transcribe')]}
        onChangeProvider={() => undefined}
        onChangeParameter={onChangeParameter}
      />,
    );

    expect(screen.getByTestId('config-stt')).toHaveTextContent('openai');

    await user.click(screen.getByText('Pick credential'));

    let nextParameters = onChangeParameter.mock.calls[0][0] as Metadata[];
    expect(getMetadataValue(nextParameters, 'listen.model')).toBe(
      'gpt-4o-mini-transcribe',
    );
    expect(getMetadataValue(nextParameters, 'rapida.credential_id')).toBe(
      'credential-next',
    );

    rerender(
      <SpeechToTextProvider
        provider="custom-stt"
        parameters={[
          createMetadata('rapida.credential_id', 'credential-old'),
          createMetadata('listen.model', 'custom'),
        ]}
        onChangeProvider={() => undefined}
        onChangeParameter={onChangeParameter}
      />,
    );

    const credentialButton = screen.getByText('Pick credential');
    expect(credentialButton).toHaveAttribute('data-current', 'credential-old');
    expect(credentialButton).toHaveAttribute('data-provider', 'custom-stt');

    await user.click(credentialButton);

    nextParameters = onChangeParameter.mock.calls[
      onChangeParameter.mock.calls.length - 1
    ][0] as Metadata[];
    expect(getMetadataValue(nextParameters, 'rapida.credential_id')).toBe(
      'credential-next',
    );
    expect(getMetadataValue(nextParameters, 'listen.model')).toBe('custom');
  });

  it('hydrates defaults and validates credentials through provider config', () => {
    const defaults = [createMetadata('listen.model', 'whisper-1')];
    const config = { stt: { parameters: [] } };
    mockLoadProviderConfig.mockReturnValue(config);
    mockGetDefaultsFromConfig.mockReturnValue(defaults);
    mockValidateFromConfig.mockReturnValue(undefined);

    expect(GetDefaultSpeechToTextIfInvalid('openai', [])).toBe(defaults);
    expect(mockGetDefaultsFromConfig).toHaveBeenCalledWith(
      config,
      'stt',
      [],
      'openai',
    );

    expect(
      ValidateSpeechToTextIfInvalid('openai', [
        createMetadata('rapida.credential_id', 'credential-1'),
      ]),
    ).toBeUndefined();
    expect(mockValidateFromConfig).toHaveBeenCalledWith(
      config,
      'stt',
      'openai',
      [expect.any(Metadata)],
    );
  });

  it('returns validation errors for missing or mismatched provider credentials', () => {
    mockLoadProviderConfig.mockReturnValue({ stt: { parameters: [] } });
    mockValidateFromConfig.mockReturnValue(undefined);

    expect(ValidateSpeechToTextIfInvalid('openai', [], [])).toBe(
      'Please provide a valid openai credential.',
    );
    expect(
      ValidateSpeechToTextIfInvalid(
        'openai',
        [createMetadata('rapida.credential_id', 'credential-2')],
        ['credential-1'],
      ),
    ).toBe('Please select a valid openai credential.');

    mockValidateFromConfig.mockReturnValue('Missing model');
    expect(
      ValidateSpeechToTextIfInvalid(
        'openai',
        [createMetadata('rapida.credential_id', 'credential-1')],
        ['credential-1'],
      ),
    ).toBe('Missing model');
  });

  it('validates custom STT HTTP request rules for HTTP compatible credentials', () => {
    const httpCredential = {
      getId: () => 'credential-1',
      getValue: () => ({
        getFieldsMap: () =>
          new Map([['apiCompatibility', { getStringValue: () => 'http_v1' }]]),
      }),
    } as any;
    mockLoadProviderConfig.mockReturnValue({ stt: { parameters: [] } });
    mockValidateFromConfig.mockReturnValue(undefined);

    expect(
      ValidateSpeechToTextIfInvalid(
        'custom-stt',
        [createMetadata('rapida.credential_id', 'credential-1')],
        [httpCredential],
      ),
    ).toBe('Please provide valid custom STT request rules.');
    expect(
      ValidateSpeechToTextIfInvalid(
        'custom-stt',
        [
          createMetadata('rapida.credential_id', 'credential-1'),
          createMetadata('listen.request_rules', 'not-json'),
        ],
        [httpCredential],
      ),
    ).toBe('Please provide valid JSON request rules for custom STT.');

    const customSttContract = require('@/providers/custom-stt/contract');
    jest
      .spyOn(customSttContract, 'parseCustomSttRequestRules')
      .mockImplementationOnce(() => {
        throw 'invalid rules';
      });
    expect(
      ValidateSpeechToTextIfInvalid(
        'custom-stt',
        [
          createMetadata('rapida.credential_id', 'credential-1'),
          createMetadata('listen.request_rules', '[]'),
        ],
        [httpCredential],
      ),
    ).toBe('Please provide valid custom STT request rules.');

    jest
      .spyOn(customSttContract, 'parseCustomSttRequestRules')
      .mockReturnValueOnce([]);

    expect(
      ValidateSpeechToTextIfInvalid(
        'custom-stt',
        [
          createMetadata('rapida.credential_id', 'credential-1'),
          createMetadata(
            'listen.request_rules',
            '[{"when":{"packet":"turn_change"},"send":{"frame":"json","body":{"type":"start"}}}]',
          ),
        ],
        [httpCredential],
      ),
    ).toBe(
      'Custom STT request rules must contain at least one rule with when.packet "audio".',
    );
    expect(
      ValidateSpeechToTextIfInvalid(
        'custom-stt',
        [
          createMetadata('rapida.credential_id', 'credential-1'),
          createMetadata(
            'listen.request_rules',
            '[{"when":{"packet":"audio"},"send":{"frame":"binary","body":{"$path":"packet.audio.bytes"}}}]',
          ),
        ],
        [httpCredential],
      ),
    ).toBe(
      'Custom STT HTTP v1 requires the first audio request rule to use send.frame "json".',
    );
    expect(
      ValidateSpeechToTextIfInvalid(
        'custom-stt',
        [
          createMetadata('rapida.credential_id', 'credential-1'),
          createMetadata(
            'listen.request_rules',
            '[{"when":{"packet":"audio"},"send":{"frame":"json","body":{}}}]',
          ),
        ],
        [httpCredential],
      ),
    ).toBeUndefined();
  });

  it('returns existing parameters and no validation error when config is missing', () => {
    const parameters = [createMetadata('custom.key', 'custom')];
    mockLoadProviderConfig.mockReturnValue({});

    expect(GetDefaultSpeechToTextIfInvalid('unknown', parameters)).toBe(
      parameters,
    );
    expect(ValidateSpeechToTextIfInvalid('unknown', [])).toBeUndefined();
  });

  it('renders speech-to-text config only when provider config exists', () => {
    const props = {
      provider: 'openai',
      parameters: [],
      onChangeProvider: () => undefined,
      onChangeParameter: () => undefined,
    };
    mockLoadProviderConfig.mockReturnValue({});

    const { rerender } = render(<SpeechToTextConfigComponent {...props} />);
    expect(screen.queryByTestId('config-stt')).not.toBeInTheDocument();

    mockLoadProviderConfig.mockReturnValue({ stt: { parameters: [] } });
    rerender(<SpeechToTextConfigComponent {...props} />);

    expect(screen.getByTestId('config-stt')).toHaveTextContent('openai');
  });
});
