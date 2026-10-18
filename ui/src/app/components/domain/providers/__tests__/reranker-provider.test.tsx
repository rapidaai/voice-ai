import React from 'react';
import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';
import { Metadata } from '@rapidaai/react';
import {
  GetDefaultRerankerConfigIfInvalid,
  RerankerConfigComponent,
  RerankerProvider,
} from '../reranker';
import {
  COHERE_RERANKER_MODEL,
  ValidateCohereRerankerDefaultOptions,
} from '../reranker/cohere/constants';
import { ConfigureCohereRerankerModel } from '../reranker/cohere';
import userEvent from '@testing-library/user-event';

jest.mock('@/providers', () => ({
  RERANKER_PROVIDER: [
    { code: 'cohere', featureList: ['reranker'], name: 'Cohere' },
  ],
}));

jest.mock('@/app/components/domain/providers/help-label', () => ({
  HelpToggletip: ({ helpText, label }: any) =>
    helpText ? <span aria-label={`${label} help`}>{helpText}</span> : null,
}));

jest.mock('@/app/components/ui/primitives', () => ({
  Stack: ({ children }: any) => <div>{children}</div>,
}));

jest.mock('@carbon/react', () => ({
  Dropdown: ({
    disabled,
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
        disabled={disabled}
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

describe('RerankerProvider', () => {
  it('renders the provider selector and ignores empty selections', async () => {
    const user = userEvent.setup();
    const onChangeProvider = jest.fn();

    render(
      <RerankerProvider
        provider=""
        parameters={[]}
        onChangeProvider={onChangeProvider}
        onChangeParameter={() => undefined}
      />,
    );

    expect(screen.getByText('Reranker provider')).toBeInTheDocument();
    expect(screen.getByLabelText('Reranker provider help')).toHaveTextContent(
      'Select a reranker provider and model for knowledge retrieval ranking.',
    );
    expect(
      screen.getByTestId('reranker-provider-empty-name'),
    ).toBeEmptyDOMElement();
    expect(
      screen.queryByRole('combobox', { name: 'Select reranker model' }),
    ).not.toBeInTheDocument();

    const select = screen.getByRole('combobox', {
      name: 'Select reranker provider',
    });
    await user.selectOptions(select, String('cohere'));
    await user.selectOptions(select, String(''));

    expect(onChangeProvider).toHaveBeenCalledWith('cohere');
    expect(onChangeProvider).toHaveBeenCalledTimes(1);
  });

  it('updates Cohere model metadata while preserving custom parameters', async () => {
    const user = userEvent.setup();
    const onChangeParameter = jest.fn();

    render(
      <RerankerProvider
        provider="cohere"
        parameters={[
          createMetadata('model.id', COHERE_RERANKER_MODEL[0].id),
          createMetadata('model.name', COHERE_RERANKER_MODEL[0].name),
          createMetadata('custom.key', 'custom'),
        ]}
        onChangeProvider={() => undefined}
        onChangeParameter={onChangeParameter}
      />,
    );

    expect(
      screen.getByTestId('cohere-reranker-model-empty-name'),
    ).toBeEmptyDOMElement();

    const select = screen.getByRole('combobox', {
      name: 'Select reranker model',
    });
    await user.selectOptions(select, String(COHERE_RERANKER_MODEL[1].id));
    await user.selectOptions(select, String(''));

    const nextParameters = onChangeParameter.mock.calls[0][0] as Metadata[];
    expect(getMetadataValue(nextParameters, 'model.id')).toBe(
      COHERE_RERANKER_MODEL[1].id,
    );
    expect(getMetadataValue(nextParameters, 'model.name')).toBe(
      COHERE_RERANKER_MODEL[1].name,
    );
    expect(getMetadataValue(nextParameters, 'custom.key')).toBe('custom');
    expect(onChangeParameter).toHaveBeenCalledTimes(1);
  });

  it('returns null config for unknown provider', () => {
    render(
      <RerankerConfigComponent
        provider="unknown"
        parameters={[]}
        onChangeProvider={() => undefined}
        onChangeParameter={() => undefined}
      />,
    );

    expect(
      screen.queryByRole('combobox', { name: 'Select reranker model' }),
    ).not.toBeInTheDocument();
  });

  it('handles empty model parameters and disabled state', () => {
    render(
      <ConfigureCohereRerankerModel
        disabled
        parameters={null}
        onParameterChange={() => undefined}
      />,
    );

    const select = screen.getByRole('combobox', {
      name: 'Select reranker model',
    });
    expect(select).toBeDisabled();
    expect(select).toHaveValue('');
  });

  it('routes default and validation helpers', () => {
    expect(
      GetDefaultRerankerConfigIfInvalid('unknown', [
        createMetadata('custom.key', 'custom'),
      ]).map(parameter => parameter.getKey()),
    ).toEqual(['custom.key']);
    expect(
      GetDefaultRerankerConfigIfInvalid('cohere', []).map(parameter =>
        parameter.getKey(),
      ),
    ).toEqual(['model.id', 'model.name']);
    expect(ValidateCohereRerankerDefaultOptions([])).toBe(false);
    expect(
      ValidateCohereRerankerDefaultOptions([
        createMetadata('model.id', COHERE_RERANKER_MODEL[0].id),
      ]),
    ).toBe(true);
  });
});
