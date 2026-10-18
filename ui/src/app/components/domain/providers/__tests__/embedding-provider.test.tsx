import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import { Metadata } from '@rapidaai/react';
import { EmbeddingConfigComponent, EmbeddingProvider } from '../embedding';
import {
  GetDefaultEmbeddingConfigIfInvalid,
  ValidateEmbeddingDefaultOptions,
} from '@/providers/embedding-defaults';
import { COHERE_EMBEDDING_MODEL } from '../embedding/cohere/constants';
import { OPENAI_EMBEDDING_MODEL } from '../embedding/openai/constants';
import { VOYAGE_EMBEDDING_MODEL } from '../embedding/voyageai/constants';

jest.mock('@/providers', () => ({
  EMBEDDING_PROVIDERS: [
    { code: 'openai', featureList: ['embedding'], name: 'OpenAI' },
    { code: 'cohere', featureList: ['embedding'], name: 'Cohere' },
    { code: 'gemini', featureList: ['embedding'], name: 'Gemini' },
    { code: 'voyageai', featureList: ['embedding'], name: 'Voyage AI' },
  ],
  GEMINI_EMBEDDING_MODEL: () => [
    { id: 'gemini-embedding-1', name: 'gemini-embedding-001' },
    { id: 'gemini-embedding-2', name: 'gemini-embedding-002' },
  ],
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
        value={selectedItem?.id ?? selectedItem?.code ?? ''}
        onChange={event => {
          const selected = items.find((item: any) =>
            item.id
              ? item.id === event.target.value
              : item.code === event.target.value,
          );
          onChange?.({ selectedItem: selected ?? null });
        }}
      >
        <option value="">Select</option>
        {items.map((item: any) => (
          <option key={item.id ?? item.code} value={item.id ?? item.code}>
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

describe('EmbeddingProvider', () => {
  it('renders the provider selector and ignores empty selections', async () => {
    const user = userEvent.setup();
    const onChangeProvider = jest.fn();

    render(
      <EmbeddingProvider
        provider=""
        parameters={[]}
        onChangeProvider={onChangeProvider}
        onChangeParameter={() => undefined}
      />,
    );

    expect(screen.getByText('Embedding provider')).toBeInTheDocument();
    expect(screen.getByLabelText('Embedding provider help')).toHaveTextContent(
      'Select an embedding provider and model for knowledge retrieval.',
    );
    expect(
      screen.getByTestId('embedding-provider-empty-name'),
    ).toBeEmptyDOMElement();
    expect(screen.queryByText('Pick credential')).not.toBeInTheDocument();

    const select = screen.getByRole('combobox', {
      name: 'Select embedding provider',
    });
    await user.selectOptions(select, String('openai'));
    await user.selectOptions(select, String(''));

    expect(onChangeProvider).toHaveBeenCalledWith('openai');
    expect(onChangeProvider).toHaveBeenCalledTimes(1);
  });

  it('adds and replaces credential metadata for a selected provider', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onChangeParameter = jest.fn();

    const { rerender } = render(
      <EmbeddingProvider
        provider="openai"
        parameters={[createMetadata('model.id', OPENAI_EMBEDDING_MODEL[0].id)]}
        onChangeProvider={() => undefined}
        onChangeParameter={onChangeParameter}
      />,
    );

    await user.click(screen.getByText('Pick credential'));

    let nextParameters = onChangeParameter.mock.calls[0][0] as Metadata[];
    expect(getMetadataValue(nextParameters, 'model.id')).toBe(
      OPENAI_EMBEDDING_MODEL[0].id,
    );
    expect(getMetadataValue(nextParameters, 'rapida.credential_id')).toBe(
      'credential-next',
    );

    rerender(
      <EmbeddingProvider
        provider="cohere"
        parameters={[
          createMetadata('rapida.credential_id', 'credential-old'),
          createMetadata('model.id', COHERE_EMBEDDING_MODEL[0].id),
        ]}
        onChangeProvider={() => undefined}
        onChangeParameter={onChangeParameter}
      />,
    );

    const credentialButton = screen.getByText('Pick credential');
    expect(credentialButton).toHaveAttribute('data-current', 'credential-old');
    expect(credentialButton).toHaveAttribute('data-provider', 'cohere');

    await user.click(credentialButton);
    nextParameters = onChangeParameter.mock.calls[
      onChangeParameter.mock.calls.length - 1
    ][0] as Metadata[];
    expect(getMetadataValue(nextParameters, 'rapida.credential_id')).toBe(
      'credential-next',
    );
  });

  it.each([
    ['openai', OPENAI_EMBEDDING_MODEL[1].id, OPENAI_EMBEDDING_MODEL[1].name],
    ['cohere', COHERE_EMBEDDING_MODEL[1].id, COHERE_EMBEDDING_MODEL[1].name],
    ['voyageai', VOYAGE_EMBEDDING_MODEL[1].id, VOYAGE_EMBEDDING_MODEL[1].name],
    ['gemini', 'gemini-embedding-2', 'gemini-embedding-002'],
  ])(
    'updates %s embedding model metadata',
    async (provider, modelId, modelName) => {
      const user = userEvent.setup();
      const onChangeParameter = jest.fn();

      render(
        <EmbeddingConfigComponent
          provider={provider}
          parameters={[
            createMetadata('model.id', 'old-id'),
            createMetadata('model.name', 'old-name'),
            createMetadata('custom.key', 'custom'),
          ]}
          onChangeProvider={() => undefined}
          onChangeParameter={onChangeParameter}
        />,
      );

      const modelIdPrefix = provider === 'voyageai' ? 'voyage' : provider;
      expect(
        screen.getByTestId(`${modelIdPrefix}-embedding-model-empty-name`),
      ).toBeEmptyDOMElement();

      const select = screen.getByRole('combobox', {
        name: 'Select embedding model',
      });
      await user.selectOptions(select, String(modelId));
      await user.selectOptions(select, String(''));

      const nextParameters = onChangeParameter.mock.calls[0][0] as Metadata[];
      expect(getMetadataValue(nextParameters, 'model.id')).toBe(modelId);
      expect(getMetadataValue(nextParameters, 'model.name')).toBe(modelName);
      expect(getMetadataValue(nextParameters, 'custom.key')).toBe('custom');
      expect(onChangeParameter).toHaveBeenCalledTimes(1);
    },
  );

  it('returns null config for unknown provider', () => {
    render(
      <EmbeddingConfigComponent
        provider="unknown"
        parameters={[]}
        onChangeProvider={() => undefined}
        onChangeParameter={() => undefined}
      />,
    );

    expect(
      screen.queryByRole('combobox', { name: 'Select embedding model' }),
    ).not.toBeInTheDocument();
  });

  it('routes default and validation helpers by provider', () => {
    const existingCredential = [
      createMetadata('rapida.credential_id', 'cred-1'),
    ];

    expect(
      GetDefaultEmbeddingConfigIfInvalid('unknown', [
        createMetadata('custom.key', 'custom'),
      ]).map(parameter => parameter.getKey()),
    ).toEqual(['custom.key']);
    expect(ValidateEmbeddingDefaultOptions('unknown', [])).toBe(
      'Please select a valid provider and model for embedding',
    );
    expect(
      GetDefaultEmbeddingConfigIfInvalid('openai', []).map(parameter =>
        parameter.getKey(),
      ),
    ).toEqual(['model.id', 'model.name']);
    expect(
      GetDefaultEmbeddingConfigIfInvalid('openai', existingCredential).map(
        parameter => parameter.getKey(),
      ),
    ).toEqual(['rapida.credential_id', 'model.id', 'model.name']);
    expect(
      GetDefaultEmbeddingConfigIfInvalid('cohere', []).map(parameter =>
        parameter.getKey(),
      ),
    ).toEqual(['model.id', 'model.name']);
    expect(
      GetDefaultEmbeddingConfigIfInvalid('gemini', []).map(parameter =>
        parameter.getKey(),
      ),
    ).toEqual(['model.id', 'model.name']);
    expect(
      GetDefaultEmbeddingConfigIfInvalid('voyageai', []).map(parameter =>
        parameter.getKey(),
      ),
    ).toEqual(['model.id', 'model.name']);
    expect(ValidateEmbeddingDefaultOptions('openai', existingCredential)).toBe(
      'Please select a valid embedding model.',
    );
    expect(ValidateEmbeddingDefaultOptions('cohere', existingCredential)).toBe(
      'Please select valid embedding model.',
    );
    expect(ValidateEmbeddingDefaultOptions('gemini', existingCredential)).toBe(
      'Please select a valid embedding model.',
    );
    expect(
      ValidateEmbeddingDefaultOptions('voyageai', existingCredential),
    ).toBe('Please select a valid embedding model.');
  });
});
