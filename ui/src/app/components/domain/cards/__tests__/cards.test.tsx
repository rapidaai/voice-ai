import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import { ClickableKnowledgeCard, SelectKnowledgeCard } from '../knowledge-card';
import { ProviderCard } from '../provider-card';
import { RapidaCredentialCard } from '../rapida-credential-card';
import { SelectToolCard } from '../tool-card';

const mockGoTo = jest.fn();
const mockGetAllProjectCredential = jest.fn();
const mockCreateProjectCredential = jest.fn();
const mockShowLoader = jest.fn();
const mockHideLoader = jest.fn();
const mockToastError = jest.fn();
let mockProviderCredentials: any[] = [];
let mockConditionSource = 'api';

jest.mock('@carbon/icons-react', () => ({
  Edit: () => <svg data-testid="edit-icon" />,
  Folders: () => <svg data-testid="folder-icon" />,
  Launch: () => <svg data-testid="launch-icon" />,
  TrashCan: () => <svg data-testid="delete-icon" />,
}));

jest.mock('@carbon/react', () => ({
  Button: ({ children, onClick }: any) => (
    <button type="button" onClick={onClick}>
      {children}
    </button>
  ),
  ButtonSet: ({ children }: any) => <div>{children}</div>,
  Tag: ({ children }: any) => <span>{children}</span>,
}));

jest.mock('@carbon/react/es/components/IconIndicator', () => ({
  __esModule: true,
  default: ({ label }: any) => <span>{label}</span>,
}));

jest.mock('@/app/components/ui/primitives', () => ({
  BaseCard: ({ children, ...props }: any) => (
    <article {...props}>{children}</article>
  ),
  Card: ({ children, ...props }: any) => (
    <article {...props}>{children}</article>
  ),
  CardDescription: ({ children, description }: any) => (
    <p>{description || children}</p>
  ),
  CardTitle: ({ children, title }: any) => <h3>{title || children}</h3>,
  CopyButton: ({ children }: any) => (
    <button
      type="button"
      onClick={() => navigator.clipboard.writeText(children)}
    >
      Copy
    </button>
  ),
  DangerGhostButton: ({ children, onClick, renderIcon: Icon }: any) => (
    <button type="button" onClick={onClick}>
      {Icon ? <Icon /> : null}
      {children}
    </button>
  ),
  GhostButton: ({ iconDescription, onClick, renderIcon: Icon }: any) => (
    <button type="button" aria-label={iconDescription} onClick={onClick}>
      {Icon ? <Icon /> : null}
    </button>
  ),
  LinkCard: ({ children, to }: any) => <a href={to}>{children}</a>,
  OverflowMenu: ({ children, onClose, onOpen }: any) => (
    <div>
      <button
        type="button"
        onClick={() => {
          onOpen?.();
          onClose?.();
        }}
      >
        Actions
      </button>
      {children}
    </div>
  ),
  OverflowMenuItem: ({ itemText, onClick }: any) => (
    <button type="button" onClick={onClick}>
      {itemText}
    </button>
  ),
  PrimaryButton: ({ children, onClick, renderIcon: Icon }: any) => (
    <button type="button" onClick={onClick}>
      {Icon ? <Icon /> : null}
      {children}
    </button>
  ),
  ReloadButton: ({ isLoading, onClick }: any) => (
    <button type="button" aria-busy={isLoading} onClick={onClick}>
      Reload
    </button>
  ),
}));

jest.mock('@/app/components/ui/composites', () => ({
  CardOptionMenu: ({ options }: any) => (
    <div>
      {options.map((option: any, index: number) => (
        <button key={index} type="button" onClick={option.onActionClick}>
          {option.option}
        </button>
      ))}
    </div>
  ),
}));

jest.mock('@/app/components/dialogs/provider', () => ({
  CreateProviderCredentialDialog: ({ modalOpen }: any) =>
    modalOpen ? <div>Create credential dialog</div> : null,
  ViewProviderCredentialDialog: ({ modalOpen, onSetupCredential }: any) =>
    modalOpen ? (
      <button type="button" onClick={onSetupCredential}>
        View credential dialog
      </button>
    ) : null,
}));

jest.mock('@/hooks/use-global-navigator', () => ({
  useGlobalNavigation: () => ({ goTo: mockGoTo }),
}));

jest.mock('@/hooks/use-model', () => ({
  useAllProviderCredentials: () => ({
    providerCredentials: mockProviderCredentials,
  }),
}));

jest.mock('@/hooks/use-credential', () => ({
  useCurrentCredential: () => ({
    authId: 'user-1',
    token: 'token-1',
    projectId: 'project-1',
  }),
}));

jest.mock('@/stores/app', () => ({
  useRapidaStore: () => ({
    hideLoader: mockHideLoader,
    loading: false,
    showLoader: mockShowLoader,
  }),
}));

jest.mock('@/configs', () => ({
  connectionConfig: {},
}));

jest.mock('react-hot-toast/headless', () => ({
  error: (...args: any[]) => mockToastError(...args),
  success: jest.fn(),
}));

jest.mock('@rapidaai/react', () => ({
  CreateProjectCredential: (...args: any[]) =>
    mockCreateProjectCredential(...args),
  GetAllProjectCredential: (...args: any[]) =>
    mockGetAllProjectCredential(...args),
}));

jest.mock('@/llm-tools', () => ({
  BUILDIN_TOOLS: [
    {
      code: 'mcp',
      icon: '/mcp.svg',
      name: 'MCP',
    },
    {
      code: 'endpoint',
      icon: '/endpoint.svg',
      name: 'Endpoint',
    },
  ],
}));

jest.mock('@/app/components/domain/tools/common', () => ({
  getToolConditionSource: () => mockConditionSource,
  getToolConditionSourceLabel: () => 'API',
}));

const makeKnowledge = () =>
  ({
    getDescription: () => 'Support policies and product documentation.',
    getDocumentcount: () => 12,
    getId: () => 'knowledge-1',
    getName: () => 'Support knowledge',
    getTokencount: () => 34567,
    getWordcount: () => 12345,
  }) as any;

const makeProvider = () => ({
  code: 'openai',
  description: 'OpenAI model provider.',
  featureList: ['external'],
  image: '/openai.svg',
  name: 'OpenAI',
  url: '/providers/openai',
});

const makeCredential = (provider = 'openai') => ({
  getProvider: () => provider,
});

const makeTool = (executionMethod = 'mcp') =>
  ({
    getDescription: () => 'Runs a remote tool.',
    getExecutionmethod: () => executionMethod,
    getExecutionoptionsList: () => [],
    getName: () => 'Customer lookup',
  }) as any;

const makeProjectCredential = () =>
  ({
    getCreateddate: () => null,
    getKey: () => 'pk_test_123',
  }) as any;

describe('domain cards', () => {
  const originalClipboard = navigator.clipboard;

  beforeEach(() => {
    jest.clearAllMocks();
    mockProviderCredentials = [];
    mockConditionSource = 'api';
    Object.assign(navigator, {
      clipboard: {
        writeText: jest.fn(),
      },
    });
    mockGetAllProjectCredential.mockImplementation(
      (_config, _projectId, cb) => {
        cb(null, {
          getDataList: () => [makeProjectCredential()],
          getSuccess: () => true,
        });
      },
    );
  });

  afterAll(() => {
    Object.assign(navigator, { clipboard: originalClipboard });
  });

  it('renders selectable and clickable knowledge cards', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onActionClick = jest.fn();

    render(
      <>
        <SelectKnowledgeCard
          knowledge={makeKnowledge()}
          knowledgeOptions={[
            {
              option: 'Archive',
              onActionClick,
            },
          ]}
        />
        <ClickableKnowledgeCard knowledge={makeKnowledge()} />
      </>,
    );

    expect(screen.getAllByText('Support knowledge')).toHaveLength(2);
    expect(screen.getAllByTestId('folder-icon')).toHaveLength(2);
    expect(screen.getByText('12 docs')).toBeInTheDocument();
    expect(screen.getByText('12.3K words')).toBeInTheDocument();
    expect(screen.getByText('34.6K tokens')).toBeInTheDocument();
    expect(screen.getByRole('link')).toHaveAttribute(
      'href',
      '/knowledge/knowledge-1',
    );

    await user.click(screen.getByRole('button', { name: 'Archive' }));
    expect(onActionClick).toHaveBeenCalled();
  });

  it('renders provider connection state and actions', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    mockProviderCredentials = [makeCredential()];

    render(<ProviderCard provider={makeProvider()} />);

    expect(screen.getByText('OpenAI')).toBeInTheDocument();
    expect(screen.getByText('Connected')).toBeInTheDocument();
    expect(
      screen.queryByRole('button', { name: 'Setup Credential' }),
    ).not.toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Open provider' }));
    expect(mockGoTo).toHaveBeenCalledWith('/providers/openai');
  });

  it('opens provider credential dialogs from card actions', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    render(<ProviderCard provider={makeProvider()} />);

    await user.click(screen.getByRole('button', { name: 'Actions' }));
    await user.click(
      screen.getByRole('button', { name: 'Create a credential' }),
    );
    expect(screen.getByText('Create credential dialog')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'View credential' }));
    expect(screen.getByText('View credential dialog')).toBeInTheDocument();
    await user.click(screen.getByText('View credential dialog'));
    expect(screen.getByText('Create credential dialog')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Setup Credential' }));
    expect(screen.getByText('Create credential dialog')).toBeInTheDocument();
  });

  it('renders tool metadata and dispatches edit/delete actions', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onEdit = jest.fn();
    const onDelete = jest.fn();

    render(
      <SelectToolCard tool={makeTool()} onEdit={onEdit} onDelete={onDelete} />,
    );

    expect(screen.getByText('Customer lookup')).toBeInTheDocument();
    expect(screen.getAllByText('MCP')).toHaveLength(2);
    expect(screen.getByText('Source: API')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: /Edit/ }));
    await user.click(screen.getByRole('button', { name: /Delete/ }));

    expect(onEdit).toHaveBeenCalled();
    expect(onDelete).toHaveBeenCalled();
  });

  it('renders fallback tool metadata for unknown and plain-object tools', () => {
    mockConditionSource = 'all';

    const { rerender } = render(
      <SelectToolCard tool={makeTool('custom_tool')} />,
    );

    expect(screen.getByText('custom tool')).toBeInTheDocument();
    expect(screen.queryByText('Source: API')).not.toBeInTheDocument();

    rerender(
      <SelectToolCard
        tool={
          {
            buildinToolConfig: { parameters: [] },
            description: 'Plain object tool.',
            name: 'Plain tool',
          } as any
        }
      />,
    );

    expect(screen.getByText('Plain tool')).toBeInTheDocument();
    expect(screen.getByText('Plain object tool.')).toBeInTheDocument();
  });

  it('renders SDK credentials and reloads the list', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const writeText = jest.spyOn(navigator.clipboard, 'writeText');
    render(<RapidaCredentialCard />);

    await waitFor(() => {
      expect(screen.getByText('pk_test_123')).toBeInTheDocument();
    });
    expect(mockShowLoader).toHaveBeenCalled();
    expect(mockHideLoader).toHaveBeenCalled();

    await user.click(screen.getByRole('button', { name: 'Reload' }));
    expect(mockGetAllProjectCredential).toHaveBeenCalledTimes(2);

    await user.click(screen.getByRole('button', { name: 'Copy' }));
    expect(writeText).toHaveBeenCalledWith('pk_test_123');
  });

  it('creates a credential from the empty credential state', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    mockGetAllProjectCredential.mockImplementation(
      (_config, _projectId, cb) => {
        cb(null, {
          getDataList: () => [],
          getSuccess: () => true,
        });
      },
    );
    mockCreateProjectCredential.mockImplementation(
      (_config, _projectId, _name, cb) => {
        cb(null, { getSuccess: () => true });
      },
    );

    render(<RapidaCredentialCard />);

    await waitFor(() => {
      expect(screen.getByText('No credentials')).toBeInTheDocument();
    });

    await user.click(
      screen.getByRole('button', { name: 'Create new credential' }),
    );

    expect(mockCreateProjectCredential).toHaveBeenCalled();
  });

  it('shows SDK credential errors from create and list requests', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    mockGetAllProjectCredential.mockImplementationOnce(
      (_config, _projectId, cb) => {
        cb(null, {
          getError: () => ({ getHumanmessage: () => 'Unable to list keys.' }),
          getSuccess: () => false,
        });
      },
    );

    const { unmount } = render(<RapidaCredentialCard />);

    await waitFor(() => {
      expect(mockToastError).toHaveBeenCalledWith('Unable to list keys.');
    });
    unmount();

    mockGetAllProjectCredential.mockImplementation(
      (_config, _projectId, cb) => {
        cb(null, {
          getDataList: () => [],
          getSuccess: () => true,
        });
      },
    );
    mockCreateProjectCredential.mockImplementation(
      (_config, _projectId, _name, cb) => {
        cb(null, {
          getError: () => ({ getHumanmessage: () => 'Unable to create key.' }),
          getSuccess: () => false,
        });
      },
    );

    render(<RapidaCredentialCard />);

    await waitFor(() => {
      expect(screen.getByText('No credentials')).toBeInTheDocument();
    });
    await user.click(
      screen.getByRole('button', { name: 'Create new credential' }),
    );

    expect(mockToastError).toHaveBeenCalledWith('Unable to create key.');
  });

  it('uses fallback SDK credential errors when responses omit details', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    mockGetAllProjectCredential.mockImplementationOnce(
      (_config, _projectId, cb) => {
        cb(null, {
          getError: () => null,
          getSuccess: () => false,
        });
      },
    );

    const { unmount } = render(<RapidaCredentialCard />);

    await waitFor(() => {
      expect(mockToastError).toHaveBeenCalledWith(
        'Unable to process your request. please try again later.',
      );
    });
    unmount();

    mockToastError.mockClear();
    mockGetAllProjectCredential.mockImplementation(
      (_config, _projectId, cb) => {
        cb(null, {
          getDataList: () => [],
          getSuccess: () => true,
        });
      },
    );
    mockCreateProjectCredential.mockImplementation(
      (_config, _projectId, _name, cb) => {
        cb(null, {
          getError: () => null,
          getSuccess: () => false,
        });
      },
    );

    render(<RapidaCredentialCard />);

    await waitFor(() => {
      expect(screen.getByText('No credentials')).toBeInTheDocument();
    });
    await user.click(
      screen.getByRole('button', { name: 'Create new credential' }),
    );

    expect(mockToastError).toHaveBeenCalledWith(
      'Unable to process your request. please try again later.',
    );
  });
});
