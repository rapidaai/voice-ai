import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import { Metadata } from '@rapidaai/react';
import { TelemetryProvider } from '../telemetry';
import {
  GetDefaultTelemetryIfInvalid,
  TelemetryConfigComponent,
  ValidateTelemetry,
} from '../telemetry/provider';

const mockGetDefaultsFromConfig = jest.fn();
const mockLoadProviderConfig = jest.fn();
const mockValidateFromConfig = jest.fn();

jest.mock('@/providers', () => ({
  TELEMETRY_PROVIDER: [
    { code: 'langfuse', featureList: ['telemetry'], name: 'Langfuse' },
    { code: 'otlp_http', featureList: ['telemetry'], name: 'OTLP HTTP' },
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
    label,
    items = [],
    itemToString,
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

describe('telemetry provider configuration', () => {
  it('selects a provider and initializes telemetry defaults', async () => {
    const user = userEvent.setup();
    const onChangeProvider = jest.fn();
    const onChangeParameter = jest.fn();
    const defaults = [createMetadata('public_key', 'pk_test')];
    mockLoadProviderConfig.mockReturnValue({ telemetry: { parameters: [] } });
    mockGetDefaultsFromConfig.mockReturnValue(defaults);

    render(
      <TelemetryProvider
        provider=""
        onChangeProvider={onChangeProvider}
        parameters={[]}
        onChangeParameter={onChangeParameter}
      />,
    );

    expect(screen.queryByText('Pick credential')).not.toBeInTheDocument();
    expect(screen.getByText('Telemetry provider')).toBeInTheDocument();
    expect(screen.getByLabelText('Telemetry provider help')).toHaveTextContent(
      'Select a telemetry provider for assistant observability.',
    );
    expect(
      screen.getByTestId('telemetry-provider-empty-name'),
    ).toBeEmptyDOMElement();

    const select = screen.getByRole('combobox', {
      name: 'Select telemetry provider',
    });
    await user.selectOptions(select, String('langfuse'));
    await user.selectOptions(select, String(''));

    expect(onChangeProvider).toHaveBeenCalledWith('langfuse');
    expect(onChangeProvider).toHaveBeenCalledTimes(1);
    expect(onChangeParameter).toHaveBeenCalledWith(defaults);
    expect(mockGetDefaultsFromConfig).toHaveBeenCalledWith(
      { telemetry: { parameters: [] } },
      'telemetry',
      [],
      'langfuse',
      { includeCredential: true },
    );
  });

  it('updates an existing credential metadata value', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onChangeParameter = jest.fn();
    mockLoadProviderConfig.mockReturnValue({ telemetry: { parameters: [] } });

    render(
      <TelemetryProvider
        provider="langfuse"
        onChangeProvider={() => undefined}
        parameters={[
          createMetadata('rapida.credential_id', 'credential-old'),
          createMetadata('public_key', 'pk_test'),
        ]}
        onChangeParameter={onChangeParameter}
      />,
    );

    const credentialButton = screen.getByText('Pick credential');
    expect(credentialButton).toHaveAttribute('data-current', 'credential-old');
    expect(credentialButton).toHaveAttribute('data-provider', 'langfuse');
    expect(screen.getByTestId('config-telemetry')).toHaveTextContent(
      'langfuse',
    );

    await user.click(credentialButton);

    const nextParameters = onChangeParameter.mock.calls[0][0] as Metadata[];
    expect(getMetadataValue(nextParameters, 'rapida.credential_id')).toBe(
      'credential-next',
    );
    expect(getMetadataValue(nextParameters, 'public_key')).toBe('pk_test');
  });

  it('adds credential metadata when no credential exists yet', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onChangeParameter = jest.fn();
    mockLoadProviderConfig.mockReturnValue({ telemetry: { parameters: [] } });

    render(
      <TelemetryProvider
        provider="otlp_http"
        onChangeProvider={() => undefined}
        parameters={[createMetadata('endpoint', 'https://example.test')]}
        onChangeParameter={onChangeParameter}
      />,
    );

    await user.click(screen.getByText('Pick credential'));

    const nextParameters = onChangeParameter.mock.calls[0][0] as Metadata[];
    expect(getMetadataValue(nextParameters, 'endpoint')).toBe(
      'https://example.test',
    );
    expect(getMetadataValue(nextParameters, 'rapida.credential_id')).toBe(
      'credential-next',
    );
  });

  it('returns existing parameters when telemetry provider config is missing', () => {
    const parameters = [createMetadata('rapida.credential_id', 'credential-1')];
    mockLoadProviderConfig.mockReturnValue({});

    expect(GetDefaultTelemetryIfInvalid('missing', parameters)).toBe(
      parameters,
    );
  });

  it('validates required credentials before provider config validation', () => {
    expect(ValidateTelemetry('langfuse', [])).toBe(
      'Please provide a valid langfuse credential.',
    );
    expect(
      ValidateTelemetry('langfuse', [
        createMetadata('rapida.credential_id', ''),
      ]),
    ).toBe('Please provide a valid langfuse credential.');

    expect(mockLoadProviderConfig).not.toHaveBeenCalled();
  });

  it('validates telemetry config when credentials are present', () => {
    const parameters = [createMetadata('rapida.credential_id', 'credential-1')];
    mockLoadProviderConfig.mockReturnValueOnce({});

    expect(ValidateTelemetry('langfuse', parameters)).toBeUndefined();

    mockLoadProviderConfig.mockReturnValueOnce({
      telemetry: { parameters: [] },
    });
    mockValidateFromConfig.mockReturnValue('Missing public key');

    expect(ValidateTelemetry('langfuse', parameters)).toBe(
      'Missing public key',
    );
    expect(mockValidateFromConfig).toHaveBeenCalledWith(
      { telemetry: { parameters: [] } },
      'telemetry',
      'langfuse',
      parameters,
    );
  });

  it('renders telemetry config only when provider config exists', () => {
    const props = {
      provider: 'langfuse',
      onChangeParameter: () => undefined,
      parameters: [],
    };
    mockLoadProviderConfig.mockReturnValue({});

    const { rerender } = render(<TelemetryConfigComponent {...props} />);
    expect(screen.queryByTestId('config-telemetry')).not.toBeInTheDocument();

    mockLoadProviderConfig.mockReturnValue({ telemetry: { parameters: [] } });
    rerender(<TelemetryConfigComponent {...props} />);

    expect(screen.getByTestId('config-telemetry')).toHaveTextContent(
      'langfuse',
    );
  });
});
