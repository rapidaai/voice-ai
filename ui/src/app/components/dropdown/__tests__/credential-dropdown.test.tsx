import React from 'react';
import { fireEvent, render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';

import { CredentialDropdown } from '@/app/components/dropdown/credential-dropdown';

const mockReloadProviderCredentials = jest.fn();
const mockAllProvider = jest.fn();
const mockCreateProviderCredentialDialog = jest.fn();

type MockCredential = {
  getId: () => string;
  getName: () => string;
  getProvider: () => string;
};

const makeCredential = (
  id: string,
  name: string,
  provider: string,
): MockCredential => ({
  getId: () => id,
  getName: () => name,
  getProvider: () => provider,
});

let mockProviderCredentials: MockCredential[] = [];

jest.mock('@/hooks/use-model', () => ({
  useAllProviderCredentials: () => ({
    providerCredentials: mockProviderCredentials,
  }),
}));

jest.mock('@/context/provider-context', () => ({
  useProviderContext: () => ({
    reloadProviderCredentials: () => mockReloadProviderCredentials(),
  }),
}));

jest.mock('@/providers', () => ({
  allProvider: () => mockAllProvider(),
}));

jest.mock(
  '@/app/components/base/modal/create-provider-credential-modal',
  () => ({
    CreateProviderCredentialDialog: (props: any) => {
      mockCreateProviderCredentialDialog(props);
      return props.modalOpen ? (
        <div data-testid="create-provider-credential-modal" />
      ) : null;
    },
  }),
);

jest.mock('@/app/components/carbon/dropdown', () => ({
  Dropdown: ({
    id,
    titleText,
    label,
    items,
    selectedItem,
    itemToString,
    onChange,
  }: any) => {
    const selectedIndex = Math.max(
      items.findIndex((x: any) => x === selectedItem),
      0,
    );
    return (
      <div>
        <label htmlFor={id}>{titleText}</label>
        <select
          id={id}
          aria-label={label}
          data-testid={id}
          value={String(selectedIndex)}
          onChange={e => {
            const idx = Number(e.target.value);
            onChange({ selectedItem: items[idx] ?? null });
          }}
        >
          {items.map((item: any, idx: number) => (
            <option key={item.getId?.() || idx} value={String(idx)}>
              {itemToString(item)}
            </option>
          ))}
        </select>
      </div>
    );
  },
}));

jest.mock('@carbon/react', () => ({
  Button: ({
    children,
    iconDescription,
    hasIconOnly: _,
    renderIcon: _r,
    ...props
  }: any) => (
    <button aria-label={iconDescription} {...props}>
      {children}
    </button>
  ),
  Toggletip: ({ children }: any) => <span>{children}</span>,
  ToggletipButton: ({ children, label }: any) => (
    <button type="button" aria-label={label}>
      {children || label}
    </button>
  ),
  ToggletipContent: ({ children }: any) => <span>{children}</span>,
}));

describe('CredentialDropdown', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockProviderCredentials = [];
    mockAllProvider.mockReturnValue([
      { code: 'openai', name: 'OpenAI' },
      { code: 'azure', name: 'Azure' },
    ]);
  });

  it('filters credentials by provider and calls onChangeCredential on selection', () => {
    const onChangeCredential = jest.fn();
    const openAiCred = makeCredential('c1', 'Primary', 'openai');
    const azureCred = makeCredential('c2', 'Secondary', 'azure');
    mockProviderCredentials = [openAiCred, azureCred];

    render(
      <CredentialDropdown
        provider="openai"
        currentCredential="c1"
        onChangeCredential={onChangeCredential as any}
      />,
    );

    expect(screen.getByText('Credential')).toBeInTheDocument();
    const select = screen.getByTestId('credential-dropdown');
    expect(
      screen.getByRole('option', { name: 'OpenAI / Primary' }),
    ).toBeInTheDocument();
    expect(
      screen.queryByRole('option', { name: 'Azure / Secondary' }),
    ).not.toBeInTheDocument();

    fireEvent.change(select, { target: { value: '0' } });
    expect(onChangeCredential).toHaveBeenCalledWith(openAiCred);
  });

  it('falls back to credential name when provider metadata is missing', () => {
    const onChangeCredential = jest.fn();
    const unknownCred = makeCredential('c3', 'Lonely', 'unknown-provider');
    mockProviderCredentials = [unknownCred];
    mockAllProvider.mockReturnValue([{ code: 'openai', name: 'OpenAI' }]);

    render(
      <CredentialDropdown
        provider="unknown-provider"
        onChangeCredential={onChangeCredential as any}
      />,
    );

    expect(screen.getByRole('option', { name: 'Lonely' })).toBeInTheDocument();
  });

  it('reload and create actions work (refresh callback + modal trigger)', () => {
    const onChangeCredential = jest.fn();
    mockProviderCredentials = [makeCredential('c1', 'Primary', 'openai')];

    render(
      <CredentialDropdown
        provider="openai"
        onChangeCredential={onChangeCredential as any}
      />,
    );

    fireEvent.click(
      screen.getByRole('button', { name: 'Refresh credentials' }),
    );
    expect(mockReloadProviderCredentials).toHaveBeenCalledTimes(1);

    fireEvent.click(screen.getByRole('button', { name: 'Create credential' }));
    expect(
      screen.getByTestId('create-provider-credential-modal'),
    ).toBeInTheDocument();
    expect(mockCreateProviderCredentialDialog).toHaveBeenLastCalledWith(
      expect.objectContaining({
        modalOpen: true,
        currentProvider: 'openai',
      }),
    );
  });
});
