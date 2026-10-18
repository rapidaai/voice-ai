import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import { Metadata } from '@rapidaai/react';
import {
  GetDefaultTextProviderConfigIfInvalid,
  GetDefaultTextProviderConfigOnProviderSwitch,
  TextProvider,
  TextProviderConfigComponent,
  ValidateTextProviderDefaultOptions,
} from '../text';

const mockGetDefaultsFromConfig = jest.fn();
const mockLoadProviderConfig = jest.fn();
const mockValidateFromConfig = jest.fn();

jest.mock('@/providers', () => ({
  TEXT_PROVIDERS: [
    { code: 'openai', featureList: ['text'], name: 'OpenAI' },
    { code: 'missing', featureList: ['text'], name: 'Missing' },
  ],
}));

jest.mock('@/providers/config-loader', () => ({
  loadProviderConfig: (...args: unknown[]) => mockLoadProviderConfig(...args),
  loadProviderData: () => [
    {
      id: 'openai/gpt-4o-mini',
      name: 'gpt-4o-mini',
      config: { parameters: [] },
    },
  ],
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

jest.mock('@carbon/react', () => ({
  Dropdown: ({
    id,
    itemToString,
    label,
    items = [],
    onChange,
    selectedItem,
    titleText,
    disabled,
    hideLabel,
  }: any) => (
    <div>
      {titleText && !hideLabel ? <span>{titleText}</span> : null}
      <span data-testid={`${id}-empty-name`}>{itemToString?.(null)}</span>
      <select
        aria-label={label}
        value={selectedItem?.code ?? ''}
        disabled={disabled}
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
  Stack: ({ children }: any) => <div>{children}</div>,
}));

const createMetadata = (key: string, value: string): Metadata => {
  const metadata = new Metadata();
  metadata.setKey(key);
  metadata.setValue(value);
  return metadata;
};

const getMetadataValue = (parameters: Metadata[], key: string) =>
  parameters.find(parameter => parameter.getKey() === key)?.getValue();

const textConfig = { text: { parameters: [] } };

beforeEach(() => {
  mockGetDefaultsFromConfig.mockReset();
  mockLoadProviderConfig.mockReset();
  mockValidateFromConfig.mockReset();
  mockLoadProviderConfig.mockImplementation((provider: string) =>
    provider === 'missing' ? {} : textConfig,
  );
  mockGetDefaultsFromConfig.mockImplementation(
    (_config, _category, current: Metadata[]) => current,
  );
});

describe('TextProvider', () => {
  it('renders configured providers and ignores empty selections', async () => {
    const user = userEvent.setup();
    const onChangeProvider = jest.fn();

    render(
      <TextProvider
        provider=""
        parameters={[]}
        onChangeProvider={onChangeProvider}
        onChangeParameter={() => undefined}
      />,
    );

    expect(screen.getByText('Model provider')).toBeInTheDocument();
    expect(screen.getByLabelText('Model provider help')).toHaveTextContent(
      'Select the provider and model configuration used by this agent.',
    );
    expect(
      screen.getByTestId('text-provider-empty-name'),
    ).toBeEmptyDOMElement();
    expect(screen.getByRole('option', { name: 'OpenAI' })).toBeInTheDocument();
    expect(
      screen.queryByRole('option', { name: 'Missing' }),
    ).not.toBeInTheDocument();
    expect(screen.getByText('Model')).toBeInTheDocument();

    const labelRow = screen
      .getByText('Model provider')
      .closest('.text-provider-label-row');
    expect(labelRow).toHaveClass(
      'cds--label',
      'text-provider-label-row',
      '!flex',
    );

    const selects = screen.getAllByRole('combobox');
    expect(selects).toHaveLength(2);
    expect(selects[1]).toBeDisabled();

    const select = screen.getByRole('combobox', { name: 'Select provider' });
    const selectorGroup = select.closest('.flex');
    expect(selectorGroup).toHaveClass(
      'text-provider-combo-row',
      'w-full',
      'items-stretch',
      'border-b',
    );
    expect(selectorGroup).not.toHaveClass('w-fit');
    expect(selectorGroup).not.toHaveClass('gap-3');

    await user.selectOptions(select, String('openai'));
    await user.selectOptions(select, String(''));

    expect(onChangeProvider).toHaveBeenCalledWith('openai');
    expect(onChangeProvider).toHaveBeenCalledTimes(1);
  });

  it('adds and replaces credential metadata for a selected provider', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onChangeParameter = jest.fn();

    const { rerender } = render(
      <TextProvider
        provider="openai"
        parameters={[createMetadata('model.id', 'openai/gpt-4o-mini')]}
        onChangeProvider={() => undefined}
        onChangeParameter={onChangeParameter}
      />,
    );

    expect(screen.getByTestId('config-text')).toHaveTextContent('openai');
    expect(
      screen
        .getByRole('combobox', { name: 'Select provider' })
        .closest('.flex'),
    ).toHaveClass('w-full');

    await user.click(screen.getByText('Pick credential'));

    let nextParameters = onChangeParameter.mock.calls[0][0] as Metadata[];
    expect(getMetadataValue(nextParameters, 'model.id')).toBe(
      'openai/gpt-4o-mini',
    );
    expect(getMetadataValue(nextParameters, 'rapida.credential_id')).toBe(
      'credential-next',
    );

    rerender(
      <TextProvider
        provider="openai"
        parameters={[
          createMetadata('rapida.credential_id', 'credential-old'),
          createMetadata('model.id', 'openai/gpt-4o-mini'),
        ]}
        onChangeProvider={() => undefined}
        onChangeParameter={onChangeParameter}
      />,
    );

    const credentialButton = screen.getByText('Pick credential');
    expect(credentialButton).toHaveAttribute('data-current', 'credential-old');
    expect(credentialButton).toHaveAttribute('data-provider', 'openai');

    await user.click(credentialButton);
    nextParameters = onChangeParameter.mock.calls[
      onChangeParameter.mock.calls.length - 1
    ][0] as Metadata[];
    expect(getMetadataValue(nextParameters, 'rapida.credential_id')).toBe(
      'credential-next',
    );
  });

  it('hydrates and validates text defaults through provider config', () => {
    const defaults = [createMetadata('model.id', 'openai/gpt-4o-mini')];
    mockGetDefaultsFromConfig.mockReturnValue(defaults);
    mockValidateFromConfig.mockReturnValue(undefined);

    expect(GetDefaultTextProviderConfigIfInvalid('openai', [])).toBe(defaults);
    expect(mockGetDefaultsFromConfig).toHaveBeenCalledWith(
      textConfig,
      'text',
      [],
      'openai',
    );
    expect(
      ValidateTextProviderDefaultOptions('openai', [
        createMetadata('rapida.credential_id', 'credential-1'),
      ]),
    ).toBeUndefined();
    expect(mockValidateFromConfig).toHaveBeenCalledWith(
      textConfig,
      'text',
      'openai',
      [expect.any(Metadata)],
    );
  });

  it('clears credential and stale model metadata on provider switch', () => {
    mockGetDefaultsFromConfig.mockReturnValue([
      createMetadata('model.id', 'openai/gpt-4o-mini'),
    ]);

    const result = GetDefaultTextProviderConfigOnProviderSwitch('openai', [
      createMetadata('rapida.credential_id', 'credential-1'),
      createMetadata('model.id', 'old-model'),
      createMetadata('custom.key', 'custom'),
    ]);

    expect(getMetadataValue(result, 'rapida.credential_id')).toBeUndefined();
    expect(getMetadataValue(result, 'old-model')).toBeUndefined();
    expect(mockGetDefaultsFromConfig.mock.calls[0][2]).toEqual([
      expect.objectContaining({}),
    ]);
    expect(
      getMetadataValue(
        mockGetDefaultsFromConfig.mock.calls[0][2],
        'custom.key',
      ),
    ).toBe('custom');
  });

  it('returns validation errors for missing or mismatched provider credentials', () => {
    mockValidateFromConfig.mockReturnValue(undefined);

    expect(
      ValidateTextProviderDefaultOptions('openai', [], ['credential-1']),
    ).toBe('Please provide a valid openai credential.');
    expect(
      ValidateTextProviderDefaultOptions(
        'openai',
        [createMetadata('rapida.credential_id', 'credential-2')],
        ['credential-1'],
      ),
    ).toBe('Please select a valid openai credential.');

    mockValidateFromConfig.mockReturnValue('Missing model');
    expect(
      ValidateTextProviderDefaultOptions(
        'openai',
        [createMetadata('rapida.credential_id', 'credential-1')],
        ['credential-1'],
      ),
    ).toBe('Missing model');
  });

  it('handles missing provider config without rendering config controls', () => {
    const parameters = [createMetadata('custom.key', 'custom')];
    mockLoadProviderConfig.mockReturnValue({});

    expect(GetDefaultTextProviderConfigIfInvalid('missing', parameters)).toBe(
      parameters,
    );
    expect(ValidateTextProviderDefaultOptions('missing', [])).toBe(
      'Please select a valid model and provider.',
    );

    const props = {
      provider: 'missing',
      parameters: [],
      onChangeProvider: () => undefined,
      onChangeParameter: () => undefined,
    };
    const { rerender } = render(<TextProviderConfigComponent {...props} />);
    expect(screen.queryByTestId('config-text')).not.toBeInTheDocument();

    mockLoadProviderConfig.mockReturnValue(textConfig);
    rerender(<TextProviderConfigComponent {...props} provider="openai" />);

    expect(screen.getByTestId('config-text')).toHaveTextContent('openai');
  });
});
