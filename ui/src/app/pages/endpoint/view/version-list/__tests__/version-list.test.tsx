import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Version } from '../index';

const mockGetEndpointProviderModels = jest.fn();
const mockReleaseVersion = jest.fn();
const mockSetPage = jest.fn();
const mockSetPageSize = jest.fn();
const mockShowLoader = jest.fn();
const mockHideLoader = jest.fn();

const makeVersion = (id: string, description: string) => ({
  getId: () => id,
  getDescription: () => description,
  getCreateddate: () => undefined,
});

const currentVersion = makeVersion(
  'current-version',
  'Current endpoint version',
);
const availableVersion = makeVersion(
  'available-version',
  'Available endpoint version',
);

const mockStore = {
  page: 1,
  pageSize: 10,
  criteria: [],
  totalCount: 2,
  currentEndpoint: {
    getEndpointprovidermodelid: () => 'current-version',
  },
  endpointProviderModels: [currentVersion, availableVersion],
  onChangeCurrentEndpoint: jest.fn(),
  getEndpointProviderModels: mockGetEndpointProviderModels,
  onReleaseVersion: mockReleaseVersion,
  setPage: mockSetPage,
  setPageSize: mockSetPageSize,
};

jest.mock('@/stores/endpoint', () => ({
  useEndpointProviderModelPageStore: () => mockStore,
}));

jest.mock('@/stores/app', () => ({
  useRapidaStore: () => ({
    loading: false,
    showLoader: mockShowLoader,
    hideLoader: mockHideLoader,
  }),
}));

jest.mock('@/hooks/use-credential', () => ({
  useCurrentCredential: () => ({
    authId: 'user-1',
    token: 'token-1',
    projectId: 'project-1',
  }),
}));

jest.mock('@/utils/audit-actor', () => ({
  createdAuditActor: jest.fn(),
  auditActorLabel: () => 'Test user',
}));

jest.mock('@carbon/react/es/components/IconIndicator', () => ({
  __esModule: true,
  default: ({ label }: { label: string }) => <span>{label}</span>,
}));

jest.mock('react-hot-toast/headless', () => ({
  success: jest.fn(),
  error: jest.fn(),
}));

const endpoint = {
  getId: () => 'endpoint-1',
} as any;

describe('Endpoint version list', () => {
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
    mockGetEndpointProviderModels.mockImplementation((...args: unknown[]) => {
      const onSuccess = args.at(-1);
      if (typeof onSuccess === 'function') onSuccess([]);
    });
  });

  it('confirms deployment from Carbon radio selection and batch actions', async () => {
    const user = userEvent.setup();
    const { container } = render(
      <Version currentEndpoint={endpoint} onReload={jest.fn()} />,
    );

    expect(
      container.querySelector('.cds--batch-actions'),
    ).not.toBeInTheDocument();

    const selections = screen.getAllByRole('radio');
    const currentSelection = selections.find(selection =>
      selection.hasAttribute('disabled'),
    );
    const availableSelection = selections.find(
      selection => !selection.hasAttribute('disabled'),
    );
    expect(currentSelection).toBeDisabled();
    expect(availableSelection?.closest('td')).toHaveClass(
      'cds--table-column-radio',
    );

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
      'available-version',
      'project-1',
      'token-1',
      'user-1',
      expect.any(Function),
      expect.any(Function),
    );
  });

  it('filters endpoint versions through the Carbon table search', async () => {
    const user = userEvent.setup();
    render(<Version currentEndpoint={endpoint} onReload={jest.fn()} />);

    await user.type(
      screen.getByRole('searchbox', { name: 'Filter table' }),
      'Available endpoint',
    );

    expect(screen.getByText('Available endpoint version')).toBeInTheDocument();
    expect(
      screen.queryByText('Current endpoint version'),
    ).not.toBeInTheDocument();
  });

  it('keeps the Carbon table skeleton at the table origin while loading', () => {
    mockGetEndpointProviderModels.mockImplementation(() => undefined);

    const { container } = render(
      <Version currentEndpoint={endpoint} onReload={jest.fn()} />,
    );

    const skeleton = container.querySelector(
      '.cds--data-table-container.cds--skeleton',
    );
    expect(skeleton).toBeInTheDocument();
    expect(skeleton?.parentElement).toHaveClass(
      'flex-1',
      'flex-col',
      'overflow-hidden',
    );
    expect(skeleton?.parentElement).not.toHaveClass(
      'items-center',
      'justify-center',
    );
    expect(mockShowLoader).not.toHaveBeenCalled();
  });

  it('keeps pagination below the scrollable table region', async () => {
    const { container } = render(
      <Version currentEndpoint={endpoint} onReload={jest.fn()} />,
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
