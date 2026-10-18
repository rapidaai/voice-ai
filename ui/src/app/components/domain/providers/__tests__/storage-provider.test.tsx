import React from 'react';
import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import { Metadata } from '@rapidaai/react';
import {
  CloudStorageProvider,
  ConfigureStorageComponent,
  defaultStorageFiles,
  GetDefaultStorageConfigIfInvalid,
  parseSelectedStorageFiles,
  preserveStorageConfigurationOptions,
  STORAGE_FILES_OPTION_KEY,
  StorageFileSelector,
  storageFiles,
  upsertStorageFilesOption,
  ValidateStorageOptions,
} from '../storage';

const mockGetDefaultsFromConfig = jest.fn();
const mockLoadProviderConfig = jest.fn();
const mockValidateFromConfig = jest.fn();

jest.mock('@/providers', () => ({
  STORAGE_PROVIDER: [
    { code: 'aws', featureList: ['storage'], name: 'AWS S3' },
    { code: 'azure', featureList: ['storage'], name: 'Azure Blob' },
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
  Checkbox: ({ checked, id, labelText, onChange }: any) => (
    <label htmlFor={id}>
      <input
        checked={checked}
        id={id}
        onChange={event => onChange(event, { checked: event.target.checked })}
        type="checkbox"
      />
      {labelText}
    </label>
  ),
  Dropdown: ({
    id,
    label,
    items = [],
    selectedItem,
    itemToString,
    onChange,
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
  StructuredListBody: ({ children }: any) => <tbody>{children}</tbody>,
  StructuredListCell: ({ children }: any) => <td>{children}</td>,
  StructuredListHead: ({ children }: any) => <thead>{children}</thead>,
  StructuredListRow: ({ children }: any) => <tr>{children}</tr>,
  StructuredListWrapper: ({
    children,
    isCondensed: _isCondensed,
    isFlush: _isFlush,
    ...props
  }: any) => <table {...props}>{children}</table>,
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

describe('storage file helpers', () => {
  it('parses selected files and falls back to defaults for invalid input', () => {
    expect(parseSelectedStorageFiles([])).toEqual(defaultStorageFiles);
    expect(
      parseSelectedStorageFiles([
        createMetadata(STORAGE_FILES_OPTION_KEY, 'not-json'),
      ]),
    ).toEqual(defaultStorageFiles);
    expect(
      parseSelectedStorageFiles([
        createMetadata(STORAGE_FILES_OPTION_KEY, '{}'),
      ]),
    ).toEqual(defaultStorageFiles);
    expect(
      parseSelectedStorageFiles([
        createMetadata(STORAGE_FILES_OPTION_KEY, '["missing"]'),
      ]),
    ).toEqual(defaultStorageFiles);
    expect(
      parseSelectedStorageFiles([
        createMetadata(
          STORAGE_FILES_OPTION_KEY,
          JSON.stringify(['recording.user', 4, 'missing']),
        ),
      ]),
    ).toEqual(['recording.user']);
  });

  it('upserts and preserves storage metadata options', () => {
    const parameters = [
      createMetadata('rapida.credential_id', 'credential-1'),
      createMetadata(STORAGE_FILES_OPTION_KEY, '[]'),
      createMetadata('bucket', 'support'),
    ];

    const upserted = upsertStorageFilesOption(parameters, [
      'recording.assistant',
    ]);

    expect(getMetadataValue(upserted, STORAGE_FILES_OPTION_KEY)).toBe(
      '["recording.assistant"]',
    );
    expect(getMetadataValue(upserted, 'bucket')).toBe('support');
    expect(
      preserveStorageConfigurationOptions(parameters).map(p => p.getKey()),
    ).toEqual(['rapida.credential_id', STORAGE_FILES_OPTION_KEY]);
  });
});

describe('StorageFileSelector', () => {
  it('renders recording files and updates checked file state', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onChange = jest.fn();

    render(
      <StorageFileSelector
        group="Recording"
        selectedFiles={['recording.user']}
        onChange={onChange}
      />,
    );

    expect(
      screen.getByRole('table', { name: 'Recording storage files' }),
    ).toBeInTheDocument();
    expect(screen.getAllByRole('checkbox')).toHaveLength(storageFiles.length);

    const userFile = screen.getByLabelText('recording.user');
    expect(userFile).toBeChecked();

    await user.click(screen.getByLabelText('recording.conversation'));
    expect(onChange).toHaveBeenCalledWith([
      'recording.user',
      'recording.conversation',
    ]);

    await user.click(userFile);
    expect(onChange).toHaveBeenCalledWith([]);
  });
});

describe('storage provider configuration', () => {
  it('builds default config with preserved credential and file options', () => {
    const current = [
      createMetadata('rapida.credential_id', 'credential-1'),
      createMetadata(STORAGE_FILES_OPTION_KEY, '["recording.user"]'),
      createMetadata('old', 'remove'),
    ];
    const defaults = [createMetadata('container', 'support')];
    mockLoadProviderConfig.mockReturnValue({ storage: { parameters: [] } });
    mockGetDefaultsFromConfig.mockReturnValue(defaults);

    const result = GetDefaultStorageConfigIfInvalid('azure', current);

    expect(result.map(param => param.getKey())).toEqual([
      'rapida.credential_id',
      STORAGE_FILES_OPTION_KEY,
      'container',
    ]);
    expect(getMetadataValue(result, 'rapida.credential_id')).toBe(
      'credential-1',
    );
    expect(getMetadataValue(result, STORAGE_FILES_OPTION_KEY)).toBe(
      '["recording.user"]',
    );
  });

  it('adds missing credential and file options while building storage defaults', () => {
    const defaults = [createMetadata('s3_bucket_name', 'support')];
    mockLoadProviderConfig.mockReturnValue({ storage: { parameters: [] } });
    mockGetDefaultsFromConfig.mockReturnValue(defaults);

    const result = GetDefaultStorageConfigIfInvalid('aws', []);

    expect(getMetadataValue(result, 'rapida.credential_id')).toBe('');
    expect(getMetadataValue(result, STORAGE_FILES_OPTION_KEY)).toBe('');
    expect(getMetadataValue(result, 's3_bucket_name')).toBe('support');
  });

  it('returns empty defaults and invalid validation when storage config is missing', () => {
    mockLoadProviderConfig.mockReturnValue({});

    expect(GetDefaultStorageConfigIfInvalid('missing', [])).toEqual([]);
    expect(ValidateStorageOptions('missing', [])).toBe(false);
  });

  it('validates storage options using provider config validation', () => {
    mockLoadProviderConfig.mockReturnValue({ storage: { parameters: [] } });
    mockValidateFromConfig.mockReturnValue(undefined);

    expect(ValidateStorageOptions('aws', [])).toBe(true);
    expect(mockValidateFromConfig).toHaveBeenCalledWith(
      { storage: { parameters: [] } },
      'storage',
      'aws',
      [],
    );

    mockValidateFromConfig.mockReturnValue('Missing bucket');

    expect(ValidateStorageOptions('aws', [])).toBe(false);
  });

  it('renders storage config only when provider config exists', () => {
    const props = {
      provider: 'aws',
      onChangeProvider: () => undefined,
      parameters: [],
      onChangeParameter: () => undefined,
    };
    mockLoadProviderConfig.mockReturnValue({});

    const { rerender } = render(<ConfigureStorageComponent {...props} />);
    expect(screen.queryByTestId('config-storage')).not.toBeInTheDocument();

    mockLoadProviderConfig.mockReturnValue({ storage: { parameters: [] } });
    rerender(<ConfigureStorageComponent {...props} />);

    expect(screen.getByTestId('config-storage')).toHaveTextContent('aws');
  });
});

describe('CloudStorageProvider', () => {
  it('selects a provider and initializes provider defaults', async () => {
    const user = userEvent.setup();
    const onChangeProvider = jest.fn();
    const onChangeParameter = jest.fn();
    const defaults = [createMetadata('container', 'support')];
    mockLoadProviderConfig.mockReturnValue({ storage: { parameters: [] } });
    mockGetDefaultsFromConfig.mockReturnValue(defaults);

    render(
      <CloudStorageProvider
        provider=""
        onChangeProvider={onChangeProvider}
        parameters={[]}
        onChangeParameter={onChangeParameter}
      />,
    );

    expect(screen.queryByText('Pick credential')).not.toBeInTheDocument();
    expect(
      screen.getByTestId('storage-provider-empty-name'),
    ).toBeEmptyDOMElement();

    const select = screen.getByRole('combobox', {
      name: 'Select storage provider',
    });
    await user.selectOptions(select, String('aws'));
    await user.selectOptions(select, String(''));

    expect(onChangeProvider).toHaveBeenCalledWith('aws');
    expect(onChangeProvider).toHaveBeenCalledTimes(1);
    expect(onChangeParameter).toHaveBeenCalledTimes(1);
    expect(
      getMetadataValue(onChangeParameter.mock.calls[0][0], 'container'),
    ).toBe('support');
  });

  it('renders credential and config controls for a selected provider', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onChangeParameter = jest.fn();
    mockLoadProviderConfig.mockReturnValue({ storage: { parameters: [] } });

    render(
      <CloudStorageProvider
        provider="aws"
        onChangeProvider={() => undefined}
        parameters={[createMetadata('rapida.credential_id', 'credential-old')]}
        onChangeParameter={onChangeParameter}
      />,
    );

    const credentialButton = screen.getByText('Pick credential');
    expect(credentialButton).toHaveAttribute('data-current', 'credential-old');
    expect(credentialButton).toHaveAttribute('data-provider', 'aws');
    expect(screen.getByTestId('config-storage')).toHaveTextContent('aws');

    await user.click(credentialButton);

    expect(
      getMetadataValue(
        onChangeParameter.mock.calls[0][0],
        'rapida.credential_id',
      ),
    ).toBe('credential-next');
  });

  it('adds a credential metadata value when one does not exist yet', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onChangeParameter = jest.fn();
    mockLoadProviderConfig.mockReturnValue({ storage: { parameters: [] } });

    render(
      <CloudStorageProvider
        provider="azure"
        onChangeProvider={() => undefined}
        parameters={[createMetadata('container', 'support')]}
        onChangeParameter={onChangeParameter}
      />,
    );

    await user.click(screen.getByText('Pick credential'));

    const nextParameters = onChangeParameter.mock.calls[0][0] as Metadata[];
    expect(getMetadataValue(nextParameters, 'container')).toBe('support');
    expect(getMetadataValue(nextParameters, 'rapida.credential_id')).toBe(
      'credential-next',
    );
  });
});
