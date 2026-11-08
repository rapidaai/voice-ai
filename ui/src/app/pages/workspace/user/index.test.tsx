import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { UserPage } from './index';

const mockGetAllUser = jest.fn();
const mockDeleteOrganizationUser = jest.fn();
const mockUpdateOrganizationUserRole = jest.fn();
const mockShowLoader = jest.fn();
const mockHideLoader = jest.fn();

const user = {
  getId: () => 'user-1',
  getName: () => 'Ada Lovelace',
  getEmail: () => 'ada@example.test',
  getRole: () => 'member',
  getCreateddate: () => undefined,
  getStatus: () => 'ACTIVE',
};

const mockStore = {
  users: [user],
  totalCount: 1,
  page: 1,
  pageSize: 10,
  criteria: [],
  getAllUser: mockGetAllUser,
  setPage: jest.fn(),
  setPageSize: jest.fn(),
};

jest.mock('@/app/components/app-shell/helmet', () => ({
  Helmet: () => null,
}));

jest.mock('@/hooks/use-credential', () => ({
  useCurrentCredential: () => ({
    projectId: 'project-1',
    authId: 'auth-1',
    token: 'token-1',
  }),
}));

jest.mock('@/stores/app', () => ({
  useRapidaStore: () => ({
    loading: false,
    showLoader: mockShowLoader,
    hideLoader: mockHideLoader,
  }),
}));

jest.mock('@/stores/user', () => ({
  useUserPageStore: () => mockStore,
}));

jest.mock('@/clients', () => ({
  deleteOrganizationUser: (...args: unknown[]) =>
    mockDeleteOrganizationUser(...args),
  updateOrganizationUserRole: (...args: unknown[]) =>
    mockUpdateOrganizationUserRole(...args),
}));

jest.mock('@/app/components/dialogs/workspace', () => ({
  InviteOrganizationUserDialog: ({ modalOpen }: { modalOpen: boolean }) =>
    modalOpen ? <div role="dialog">Invite user</div> : null,
  InviteProjectUserDialog: ({
    modalOpen,
    user,
  }: {
    modalOpen: boolean;
    user: { getEmail: () => string } | null;
  }) =>
    modalOpen ? (
      <div role="dialog">Invite {user?.getEmail()} to project</div>
    ) : null,
}));

jest.mock('@/app/components/dialogs/shared', () => ({
  ConfirmDeleteDialog: ({
    showing,
    title,
    content,
    onConfirm,
  }: {
    showing: boolean;
    title: string;
    content: string;
    onConfirm: () => void;
  }) =>
    showing ? (
      <div role="dialog">
        <h2>{title}</h2>
        <p>{content}</p>
        <button onClick={onConfirm}>Confirm</button>
      </div>
    ) : null,
}));

jest.mock('@/app/components/domain/indicators/organization-role', () => ({
  OrganizationRoleIndicator: ({ role }: { role?: string }) => (
    <span>{role}</span>
  ),
}));

jest.mock('@/app/components/ui/feedback', () => ({
  CarbonIconIndicator: ({ state }: { state?: string }) => <span>{state}</span>,
  EmptyState: ({ title }: { title: string }) => <div>{title}</div>,
}));

jest.mock('react-hot-toast/headless', () => ({
  success: jest.fn(),
  error: jest.fn(),
}));

describe('UserPage', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockStore.users = [user];
    mockGetAllUser.mockImplementation(
      (_token, _authId, _projectId, _onError, onSuccess) => onSuccess([user]),
    );
  });

  it('uses Carbon radio selection and batch actions for user actions', async () => {
    const interaction = userEvent.setup();
    const { container } = render(<UserPage />);

    await screen.findByText('Ada Lovelace');
    const selection = screen.getByRole('radio', { name: 'Select row' });
    expect(selection.closest('td')).toHaveClass('cds--table-column-radio');
    expect(container.querySelector('.cds--batch-actions')).toBeNull();

    await interaction.click(selection);

    const batchActions = container.querySelector('.cds--batch-actions');
    expect(batchActions).toHaveAttribute('aria-hidden', 'false');
    expect(batchActions).toHaveTextContent('1 item selected');
    expect(batchActions).toHaveTextContent('Make admin');

    await interaction.click(
      within(batchActions as HTMLElement).getByRole('button', {
        name: 'Invite to project',
      }),
    );
    expect(screen.getByRole('dialog')).toHaveTextContent(
      'Invite ada@example.test to project',
    );
  });

  it('asks for confirmation before deleting the selected user', async () => {
    const interaction = userEvent.setup();
    const { container } = render(<UserPage />);

    await interaction.click(
      await screen.findByRole('radio', { name: 'Select row' }),
    );
    await interaction.click(
      within(
        container.querySelector('.cds--batch-actions') as HTMLElement,
      ).getByRole('button', { name: 'Delete' }),
    );

    expect(mockDeleteOrganizationUser).not.toHaveBeenCalled();
    expect(screen.getByRole('dialog')).toHaveTextContent('Delete user');
    expect(screen.getByRole('dialog')).toHaveTextContent('ada@example.test');
  });

  it('keeps an in-place Carbon table skeleton while users load', async () => {
    mockGetAllUser.mockImplementation(() => undefined);

    const { container } = render(<UserPage />);

    await waitFor(() => expect(mockGetAllUser).toHaveBeenCalled());
    expect(
      screen.getByRole('heading', { level: 1, name: 'Users' }),
    ).toBeInTheDocument();

    const skeleton = container.querySelector(
      '.cds--data-table-container.cds--skeleton',
    );
    expect(skeleton).toBeInTheDocument();
    expect(container.querySelector('[data-page-scroll]')).toHaveClass(
      'overflow-y-auto',
      'overflow-x-hidden',
    );
    expect(skeleton?.parentElement).not.toHaveClass(
      'items-center',
      'justify-center',
    );
  });

  it('uses page scrolling while keeping horizontal table overflow local', async () => {
    const { container } = render(<UserPage />);

    await screen.findByText('Ada Lovelace');
    const pagination = container.querySelector('.cds--pagination');
    expect(
      container
        .querySelector('.cds--table-toolbar')
        ?.closest('[data-page-header-sticky-toolbar]'),
    ).toHaveClass('sticky', 'top-10', 'z-10');
    expect(pagination).toHaveClass(
      'sticky',
      'bottom-0',
      'z-10',
      'mt-auto!',
      'shrink-0',
    );
    expect(pagination?.previousElementSibling).toHaveClass(
      'shrink-0',
      'overflow-x-auto',
    );
    expect(pagination?.previousElementSibling).not.toHaveClass('flex-1');
  });
});
