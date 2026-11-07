import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Version } from '../index';

const mockGetAssistantProviders = jest.fn();
const mockReleaseVersion = jest.fn();
const mockSetPage = jest.fn();
const mockSetPageSize = jest.fn();
const mockGoToCreateAssistantVersion = jest.fn();

const makeVersion = (id: string, description: string) => ({
  getId: () => id,
  getDescription: () => description,
  getCreateddate: () => undefined,
});

const currentVersion = makeVersion('current-version', 'Current version');
const availableVersion = makeVersion('available-version', 'Available version');

const mockStore = {
  page: 1,
  pageSize: 10,
  criteria: [],
  totalCount: 2,
  assistant: {
    getAssistantproviderid: () => 'current-version',
  },
  assistantProviders: [
    {
      getAssistantproviderCase: () => 1,
      getAssistantprovidermodel: () => currentVersion,
    },
    {
      getAssistantproviderCase: () => 1,
      getAssistantprovidermodel: () => availableVersion,
    },
  ],
  onChangeAssistant: jest.fn(),
  getAssistantProviders: mockGetAssistantProviders,
  onReleaseVersion: mockReleaseVersion,
  setPage: mockSetPage,
  setPageSize: mockSetPageSize,
};

jest.mock('@/stores/assistant', () => ({
  useAssistantProviderPageStore: () => mockStore,
}));

jest.mock('@/hooks/use-credential', () => ({
  useCredential: () => ['user-1', 'token-1', 'project-1'],
}));

jest.mock('@/hooks/use-global-navigator', () => ({
  useGlobalNavigation: () => ({
    goToCreateAssistantVersion: mockGoToCreateAssistantVersion,
    goToCreateAssistantAgentKitVersion: jest.fn(),
    goToCreateAssistantAgentflowVersion: jest.fn(),
  }),
}));

jest.mock('@/utils/audit-actor', () => ({
  createdAuditActor: jest.fn(),
  auditActorLabel: () => 'Test user',
}));

jest.mock('@rapidaai/react', () => ({
  GetAllAssistantProviderResponse: {
    AssistantProvider: {
      AssistantproviderCase: {
        ASSISTANTPROVIDERMODEL: 1,
        ASSISTANTPROVIDERAGENTKIT: 2,
        ASSISTANTPROVIDERWEBSOCKET: 3,
        ASSISTANTPROVIDERAGENTFLOW: 4,
      },
    },
  },
}));

jest.mock('@carbon/react/es/components/IconIndicator', () => ({
  __esModule: true,
  default: ({ label }: { label: string }) => <span>{label}</span>,
}));

jest.mock('react-hot-toast/headless', () => ({
  success: jest.fn(),
  error: jest.fn(),
}));

const assistant = {
  getId: () => 'assistant-1',
  getAssistantproviderid: () => 'current-version',
  hasAssistantprovideragentkit: () => false,
  hasAssistantprovideragentflow: () => false,
} as any;

describe('Version list', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    Object.defineProperty(window, 'matchMedia', {
      configurable: true,
      value: jest.fn().mockReturnValue({
        matches: false,
        addEventListener: jest.fn(),
        removeEventListener: jest.fn(),
      }),
    });
    mockGetAssistantProviders.mockImplementation(
      (...args: Array<(...values: unknown[]) => void>) => {
        args.at(-1)?.();
      },
    );
  });

  it('uses Carbon radio selection and batch-action state to deploy one version', async () => {
    const user = userEvent.setup();
    const { container } = render(
      <Version assistant={assistant} onReload={jest.fn()} />,
    );

    await screen.findByText('Available version');

    expect(
      container.querySelector('.cds--batch-actions'),
    ).not.toBeInTheDocument();

    const availableSelection = screen
      .getAllByRole('radio', { name: 'Select row' })
      .find(selection => !selection.hasAttribute('disabled'));
    expect(availableSelection).toBeDefined();
    expect(availableSelection!.closest('td')).toHaveClass(
      'cds--table-column-radio',
    );
    expect(
      screen
        .getAllByRole('radio', { name: 'Select row' })
        .find(selection => selection.hasAttribute('disabled')),
    ).toBeDisabled();

    await user.click(availableSelection!);
    const batchActions = container.querySelector('.cds--batch-actions');
    expect(batchActions).toHaveAttribute('aria-hidden', 'false');
    expect(batchActions).toHaveTextContent('1 item selected');

    await user.click(
      within(batchActions as HTMLElement).getByRole('button', {
        name: 'Deploy Version',
      }),
    );

    expect(mockReleaseVersion).not.toHaveBeenCalled();
    expect(
      screen.getByRole('heading', { name: 'Deploy version?' }),
    ).toBeInTheDocument();
    expect(
      screen.getByText(
        'Deploying vrsn_available-version will replace the version currently in use.',
      ),
    ).toBeInTheDocument();

    await user.click(
      within(screen.getByRole('dialog')).getByRole('button', {
        name: 'Cancel',
      }),
    );
    expect(mockReleaseVersion).not.toHaveBeenCalled();

    await user.click(
      within(batchActions as HTMLElement).getByRole('button', {
        name: 'Deploy Version',
      }),
    );
    await user.click(
      within(screen.getByRole('dialog')).getByRole('button', {
        name: 'Deploy',
      }),
    );

    expect(mockReleaseVersion).toHaveBeenCalledWith(
      'MODEL',
      'available-version',
      'project-1',
      'token-1',
      'user-1',
      expect.any(Function),
      expect.any(Function),
    );
  });

  it('centers a Carbon table skeleton within the page while loading', () => {
    mockGetAssistantProviders.mockImplementation(() => undefined);

    const { container } = render(
      <Version assistant={assistant} onReload={jest.fn()} />,
    );

    const skeleton = container.querySelector(
      '.cds--data-table-container.cds--skeleton',
    );
    expect(skeleton).toBeInTheDocument();
    expect(skeleton?.parentElement).toHaveClass(
      'flex-1',
      'items-center',
      'justify-center',
    );
  });

  it('keeps pagination at the bottom of the page container', async () => {
    const { container } = render(
      <Version assistant={assistant} onReload={jest.fn()} />,
    );

    await waitFor(() => {
      expect(container.querySelector('.cds--pagination')).toBeInTheDocument();
    });

    const pagination = container.querySelector('.cds--pagination');
    expect(pagination).toHaveClass('shrink-0');
    expect(pagination?.previousElementSibling).toHaveClass(
      'min-h-0',
      'flex-1',
      'overflow-auto',
    );
  });
});
