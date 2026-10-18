import React from 'react';
import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import toast from 'react-hot-toast/headless';
import {
  CredentialDropdown,
  CredentialDropdownView,
} from '../credential-dropdown';
import { EndpointDropdown, EndpointDropdownView } from '../endpoint-dropdown';
import {
  KnowledgeDropdown,
  KnowledgeDropdownView,
} from '../knowledge-dropdown';

type MockCredential = {
  getId: () => string;
  getName: () => string;
  getProvider: () => string;
};

type MockEndpoint = {
  getId: () => string;
  getName: () => string;
};

type MockKnowledge = {
  getId: () => string;
  getName: () => string;
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

const makeEndpoint = (id: string, name: string): MockEndpoint => ({
  getId: () => id,
  getName: () => name,
});

const makeKnowledge = (id: string, name: string): MockKnowledge => ({
  getId: () => id,
  getName: () => name,
});

const mockReloadProviderCredentials = jest.fn();
const mockAllProvider = jest.fn();
const mockCreateProviderCredentialDialog = jest.fn();
const mockAddEndpointCriteria = jest.fn();
const mockGetAllEndpoint = jest.fn();
const mockAddKnowledgeCriteria = jest.fn();
const mockGetAllKnowledge = jest.fn();

let mockProviderCredentials: any[] = [];
let mockEndpoints: MockEndpoint[] = [];
let mockEndpointPage = 1;
let mockEndpointPageSize = 10;
let mockEndpointCriteria: any[] = [];
let mockKnowledgeBases: MockKnowledge[] = [];
let mockKnowledgePage = 1;
let mockKnowledgePageSize = 10;
let mockKnowledgeCriteria: any[] = [];

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

jest.mock('@/stores/endpoint', () => ({
  useEndpointPageStore: () => ({
    endpoints: mockEndpoints,
    page: mockEndpointPage,
    pageSize: mockEndpointPageSize,
    criteria: mockEndpointCriteria,
    addCriteria: (...args: any[]) => mockAddEndpointCriteria(...args),
    onGetAllEndpoint: (...args: any[]) => mockGetAllEndpoint(...args),
  }),
}));

jest.mock('@/stores/knowledge/knowledge.store', () => ({
  useKnowledgePageStore: () => ({
    knowledgeBases: mockKnowledgeBases,
    page: mockKnowledgePage,
    pageSize: mockKnowledgePageSize,
    criteria: mockKnowledgeCriteria,
    addCriteria: (...args: any[]) => mockAddKnowledgeCriteria(...args),
    getAllKnowledge: (...args: any[]) => mockGetAllKnowledge(...args),
  }),
}));

jest.mock('@/hooks/use-credential', () => ({
  useCredential: () => ['auth-1', 'token-1', 'project-1'],
}));

jest.mock('react-hot-toast/headless', () => ({
  __esModule: true,
  default: {
    error: jest.fn(),
  },
}));

jest.mock('@/app/components/dialogs/provider', () => ({
  CreateProviderCredentialDialog: (props: any) => {
    mockCreateProviderCredentialDialog(props);
    return props.modalOpen ? (
      <div data-testid="create-provider-credential-modal" />
    ) : null;
  },
}));

jest.mock('@carbon/icons-react', () => ({
  Add: () => <svg data-testid="add-icon" />,
  Information: () => <svg data-testid="information-icon" />,
  Launch: () => <svg data-testid="launch-icon" />,
  Renew: () => <svg data-testid="renew-icon" />,
}));

jest.mock('@carbon/react', () => ({
  Button: ({
    children,
    disabled,
    iconDescription,
    hasIconOnly: _hasIconOnly,
    renderIcon: _renderIcon,
    ...props
  }: any) => (
    <button aria-label={iconDescription} disabled={disabled} {...props}>
      {children}
    </button>
  ),
  Dropdown: ({
    disabled,
    hideLabel,
    id,
    items,
    itemToString,
    label,
    onChange,
    selectedItem,
    titleText,
  }: any) => {
    const selectedIndex = items.findIndex((item: any) => item === selectedItem);

    return (
      <label>
        {!hideLabel && <span>{titleText}</span>}
        <select
          aria-label={label}
          data-testid={id}
          disabled={disabled}
          value={selectedIndex >= 0 ? String(selectedIndex) : ''}
          onChange={event => {
            const value = event.target.value;
            onChange({
              selectedItem: value === '' ? null : items[Number(value)] ?? null,
            });
          }}
        >
          <option value="">{itemToString(null)}</option>
          {items.map((item: any, index: number) => (
            <option key={item.getId?.() || index} value={String(index)}>
              {itemToString(item)}
            </option>
          ))}
        </select>
      </label>
    );
  },
  Toggletip: ({ children }: any) => <span>{children}</span>,
  ToggletipButton: ({ children, label }: any) => (
    <button type="button" aria-label={label}>
      {children || label}
    </button>
  ),
  ToggletipContent: ({ children }: any) => <span>{children}</span>,
}));

describe('domain dropdowns', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockGetAllEndpoint.mockImplementation(
      (_projectId, _token, _userId, _onError, onSuccess) => onSuccess([]),
    );
    mockGetAllKnowledge.mockImplementation(
      (_projectId, _token, _userId, _onError, onSuccess) => onSuccess([]),
    );
    mockProviderCredentials = [];
    mockAllProvider.mockReturnValue([
      { code: 'openai', name: 'OpenAI' },
      { code: 'azure', name: 'Azure' },
    ]);
    mockEndpoints = [
      makeEndpoint('e1', 'Endpoint One'),
      makeEndpoint('e2', 'Endpoint Two'),
    ];
    mockEndpointPage = 1;
    mockEndpointPageSize = 10;
    mockEndpointCriteria = [];
    mockKnowledgeBases = [
      makeKnowledge('k1', 'Knowledge One'),
      makeKnowledge('k2', 'Knowledge Two'),
    ];
    mockKnowledgePage = 1;
    mockKnowledgePageSize = 10;
    mockKnowledgeCriteria = [];
  });

  it('filters credentials by provider and delegates selection changes', async () => {
    const user = userEvent.setup();
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
    expect(
      screen.getByRole('option', { name: 'OpenAI / Primary' }),
    ).toBeInTheDocument();
    expect(
      screen.queryByRole('option', { name: 'Azure / Secondary' }),
    ).not.toBeInTheDocument();

    await user.selectOptions(
      screen.getByTestId('credential-dropdown'),
      String('0'),
    );

    expect(onChangeCredential).toHaveBeenCalledWith(openAiCred);
  });

  it('renders action dropdowns as a single connected Carbon field', () => {
    const credential = makeCredential('c1', 'Primary', 'openai');

    render(
      <>
        <CredentialDropdownView
          credentials={[credential] as any}
          currentCredential="c1"
          getProviderName={() => 'OpenAI'}
          onChangeCredential={jest.fn() as any}
          onReloadCredentials={jest.fn()}
          onCreateCredential={jest.fn()}
        />
        <EndpointDropdownView
          endpoints={mockEndpoints as any}
          currentEndpoint="e1"
          onChangeEndpoint={jest.fn() as any}
          onRefresh={jest.fn()}
          onCreateEndpoint={jest.fn()}
        />
        <KnowledgeDropdownView
          knowledgeBases={mockKnowledgeBases as any}
          currentKnowledge="k1"
          onChangeKnowledge={jest.fn()}
          onRefresh={jest.fn()}
          onCreateKnowledge={jest.fn()}
        />
      </>,
    );

    const credentialRow = screen
      .getByTestId('credential-dropdown')
      .closest('.domain-connected-dropdown-row');
    const endpointRow = screen
      .getByTestId('endpoint-dropdown')
      .closest('.domain-connected-dropdown-row');
    const knowledgeRow = screen
      .getByTestId('knowledge-dropdown')
      .closest('.domain-connected-dropdown-row');

    expect(credentialRow).toHaveClass('bg-[var(--cds-field)]', 'border-b');
    expect(endpointRow).toHaveClass('bg-[var(--cds-field)]', 'border-b');
    expect(knowledgeRow).toHaveClass('bg-[var(--cds-field)]', 'border-b');
    expect(screen.getByText('Credential')).toHaveClass(
      'cds--label',
      'domain-connected-dropdown-label',
    );
    expect(screen.getByText('Endpoint')).toHaveClass(
      'cds--label',
      'domain-connected-dropdown-label',
    );
    expect(screen.getByText('Knowledge')).toHaveClass(
      'cds--label',
      'domain-connected-dropdown-label',
    );
    expect(
      within(credentialRow as HTMLElement).getByRole('button', {
        name: 'Refresh credentials',
      }),
    ).toHaveClass('domain-connected-dropdown-action');
    expect(
      within(endpointRow as HTMLElement).getByRole('button', {
        name: 'Create endpoint',
      }),
    ).toHaveClass('domain-connected-dropdown-action');
    expect(
      within(knowledgeRow as HTMLElement).getByRole('button', {
        name: 'Create knowledge',
      }),
    ).toHaveClass('domain-connected-dropdown-action');
  });

  it('supports credential fallback labels, empty changes, refresh, and create', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onChangeCredential = jest.fn();
    const onReloadCredentials = jest.fn();
    const onCreateCredential = jest.fn();
    const unknownCred = makeCredential('c3', 'Lonely', 'unknown-provider');

    render(
      <CredentialDropdownView
        credentials={[unknownCred] as any}
        currentCredential="missing"
        getProviderName={() => undefined}
        onChangeCredential={onChangeCredential as any}
        onReloadCredentials={onReloadCredentials}
        onCreateCredential={onCreateCredential}
      />,
    );

    expect(screen.getByRole('option', { name: 'Lonely' })).toBeInTheDocument();

    await user.selectOptions(
      screen.getByTestId('credential-dropdown'),
      String(''),
    );
    expect(onChangeCredential).not.toHaveBeenCalled();

    await user.click(
      screen.getByRole('button', { name: 'Refresh credentials' }),
    );
    expect(onReloadCredentials).toHaveBeenCalledTimes(1);

    await user.click(screen.getByRole('button', { name: 'Create credential' }));
    expect(onCreateCredential).toHaveBeenCalledTimes(1);
  });

  it('opens the credential creation modal from the data wrapper', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    mockProviderCredentials = [makeCredential('c1', 'Primary', 'openai')];

    render(
      <CredentialDropdown
        provider="openai"
        onChangeCredential={jest.fn() as any}
      />,
    );

    await user.click(screen.getByRole('button', { name: 'Create credential' }));

    expect(
      screen.getByTestId('create-provider-credential-modal'),
    ).toBeInTheDocument();
    expect(mockCreateProviderCredentialDialog).toHaveBeenLastCalledWith(
      expect.objectContaining({
        currentProvider: 'openai',
        modalOpen: true,
      }),
    );
  });

  it('loads endpoints and delegates endpoint view actions', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onChangeEndpoint = jest.fn();
    const openSpy = jest.spyOn(window, 'open').mockImplementation(() => null);

    render(
      <EndpointDropdown
        currentEndpoint="e1"
        onChangeEndpoint={onChangeEndpoint as any}
      />,
    );

    expect(mockAddEndpointCriteria).toHaveBeenCalledWith('id', 'e1', 'or');
    expect(mockGetAllEndpoint).toHaveBeenCalled();

    await user.selectOptions(
      screen.getByTestId('endpoint-dropdown'),
      String('1'),
    );
    expect(onChangeEndpoint).toHaveBeenCalledWith(mockEndpoints[1]);

    await user.click(screen.getByRole('button', { name: 'Refresh endpoints' }));
    expect(mockGetAllEndpoint).toHaveBeenCalledTimes(2);

    await user.click(screen.getByRole('button', { name: 'Create endpoint' }));
    expect(openSpy).toHaveBeenCalledWith(
      '/deployment/endpoint/create-endpoint',
      '_blank',
    );

    openSpy.mockRestore();
  });

  it('handles endpoint loading, empty selection, and fetch errors', async () => {
    const user = userEvent.setup();
    const onChangeEndpoint = jest.fn();
    const onRefresh = jest.fn();

    const { rerender } = render(
      <EndpointDropdownView
        endpoints={mockEndpoints as any}
        isLoading
        onChangeEndpoint={onChangeEndpoint as any}
        onRefresh={onRefresh}
        onCreateEndpoint={jest.fn()}
      />,
    );

    expect(screen.getByTestId('endpoint-dropdown')).toBeDisabled();
    expect(
      screen.getByRole('button', { name: 'Refresh endpoints' }),
    ).toBeDisabled();

    rerender(
      <EndpointDropdownView
        endpoints={mockEndpoints as any}
        onChangeEndpoint={onChangeEndpoint as any}
        onRefresh={onRefresh}
        onCreateEndpoint={jest.fn()}
      />,
    );

    await user.selectOptions(
      screen.getByTestId('endpoint-dropdown'),
      String(''),
    );
    expect(onChangeEndpoint).not.toHaveBeenCalled();

    mockGetAllEndpoint.mockImplementation(
      (_projectId, _token, _userId, onError) => onError('endpoint failed'),
    );

    render(<EndpointDropdown onChangeEndpoint={jest.fn() as any} />);

    expect(toast.error).toHaveBeenCalledWith('endpoint failed');
  });

  it('loads knowledge bases and delegates knowledge view actions', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onChangeKnowledge = jest.fn();
    const openSpy = jest.spyOn(window, 'open').mockImplementation(() => null);

    render(
      <KnowledgeDropdown
        currentKnowledge="k1"
        onChangeKnowledge={onChangeKnowledge as any}
      />,
    );

    expect(mockAddKnowledgeCriteria).toHaveBeenCalledWith('id', 'k1', 'or');
    expect(mockGetAllKnowledge).toHaveBeenCalled();

    await user.selectOptions(
      screen.getByTestId('knowledge-dropdown'),
      String('1'),
    );
    expect(onChangeKnowledge).toHaveBeenCalledWith(mockKnowledgeBases[1]);

    await user.click(screen.getByRole('button', { name: 'Refresh knowledge' }));
    expect(mockGetAllKnowledge).toHaveBeenCalledTimes(2);

    await user.click(screen.getByRole('button', { name: 'Create knowledge' }));
    expect(openSpy).toHaveBeenCalledWith(
      '/knowledge/create-knowledge',
      '_blank',
    );

    openSpy.mockRestore();
  });

  it('handles knowledge loading, optional selection callback, and fetch errors', async () => {
    const user = userEvent.setup();
    const { rerender } = render(
      <KnowledgeDropdownView
        knowledgeBases={mockKnowledgeBases as any}
        isLoading
        onRefresh={jest.fn()}
        onCreateKnowledge={jest.fn()}
      />,
    );

    expect(screen.getByTestId('knowledge-dropdown')).toBeDisabled();
    expect(
      screen.getByRole('button', { name: 'Refresh knowledge' }),
    ).toBeDisabled();

    rerender(
      <KnowledgeDropdownView
        knowledgeBases={mockKnowledgeBases as any}
        onRefresh={jest.fn()}
        onCreateKnowledge={jest.fn()}
      />,
    );

    await user.selectOptions(
      screen.getByTestId('knowledge-dropdown'),
      String('1'),
    );
    await user.selectOptions(
      screen.getByTestId('knowledge-dropdown'),
      String(''),
    );

    mockGetAllKnowledge.mockImplementation(
      (_projectId, _token, _userId, onError) => onError('knowledge failed'),
    );

    render(<KnowledgeDropdown />);

    expect(toast.error).toHaveBeenCalledWith('knowledge failed');
  });
});
