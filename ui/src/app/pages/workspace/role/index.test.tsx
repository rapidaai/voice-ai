import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes, useLocation } from 'react-router-dom';
import { RoleDefinitionType, RoleScope, RoleSubject } from '@rapidaai/react';
import { RoleCatalogPage } from './index';

const mockListAccessRoles = jest.fn();

jest.mock('@/hooks/use-credential', () => ({
  useCredential: () => ['user-1', 'token-1'],
}));

jest.mock('@/clients', () => ({
  listAccessRoles: (...args: unknown[]) => mockListAccessRoles(...args),
}));

jest.mock('@/app/components/app-shell/helmet', () => ({ Helmet: () => null }));

jest.mock('@/utils/date', () => ({
  toHumanReadableDateTime: (timestamp: { getSeconds: () => number }) =>
    `date:${timestamp.getSeconds()}`,
}));

jest.mock('./role-details-side-panel', () => ({
  RoleDetailsSidePanel: ({
    onClose,
    open,
    roleDefinitionId,
  }: {
    onClose: () => void;
    open: boolean;
    roleDefinitionId: string | null;
  }) =>
    open ? (
      <section aria-label="Role details panel">
        <span>Viewing role {roleDefinitionId}</span>
        <button type="button" onClick={onClose}>
          Close panel
        </button>
      </section>
    ) : null,
}));

jest.mock('./create', () => ({
  CreateRoleModal: ({
    onClose,
    open,
  }: {
    onClose: () => void;
    open: boolean;
  }) =>
    open ? (
      <section aria-label="Create role modal">
        <button type="button" onClick={onClose}>
          Cancel creation
        </button>
      </section>
    ) : null,
}));

const role = (
  id: string,
  name: string,
  definitionType: number = RoleDefinitionType.ROLE_DEFINITION_TYPE_CUSTOM,
  subject: number = RoleSubject.ROLE_SUBJECT_USER,
) => ({
  getId: () => id,
  getName: () => name,
  getKey: () => name.toLowerCase().replace(' ', '_'),
  getScope: () => RoleScope.ROLE_SCOPE_PROJECT,
  getSubject: () => subject,
  getDefinitiontype: () => definitionType,
  getCreateddate: () => ({ getSeconds: () => 1, getNanos: () => 0 }),
  getUpdateddate: () => undefined,
});

const response = (roles: ReturnType<typeof role>[]) => ({
  getSuccess: () => true,
  getRolesList: () => roles,
  getPaginated: () => ({ getTotalitem: () => roles.length }),
});

function LocationPath() {
  const location = useLocation();
  return <div data-testid="location-path">{location.pathname}</div>;
}

function renderPage(
  initialEntry:
    | string
    | {
        pathname: string;
        state?: { createdRoleDefinitionId: string };
      } = '/organization/roles',
) {
  return render(
    <MemoryRouter initialEntries={[initialEntry]}>
      <LocationPath />
      <Routes>
        <Route path="/organization/roles" element={<RoleCatalogPage />} />
        <Route
          path="/organization/roles/:roleDefinitionId"
          element={<div>Role detail route</div>}
        />
      </Routes>
    </MemoryRouter>,
  );
}

describe('RoleCatalogPage', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockListAccessRoles.mockResolvedValue(
      response([
        role(
          '1',
          'Support manager',
          RoleDefinitionType.ROLE_DEFINITION_TYPE_CUSTOM,
          RoleSubject.ROLE_SUBJECT_PROJECT_TOKEN,
        ),
        role(
          '2',
          'Organization owner',
          RoleDefinitionType.ROLE_DEFINITION_TYPE_BUILT_IN,
        ),
      ]),
    );
  });

  it('uses the shared organization catalog header treatment', async () => {
    const { container } = renderPage();

    expect(await screen.findByText('Support manager')).toBeInTheDocument();
    expect(
      screen.getByRole('heading', { level: 1, name: 'Roles & Permissions' }),
    ).toBeInTheDocument();
    expect(container.querySelector('[data-page-scroll]')).toBeInTheDocument();
    expect(screen.getByText('2 of 2')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Create role' })).toHaveClass(
      'w-full',
      'md:w-auto',
      'cds--btn--md',
    );
    expect(
      screen
        .getAllByRole('columnheader')
        .map(columnHeader => columnHeader.textContent),
    ).toEqual([
      'Name',
      'Scope',
      'Subject',
      'Type',
      'Actions',
      'Created',
      'Updated',
    ]);
    expect(
      screen.queryByRole('columnheader', { name: 'Key' }),
    ).not.toBeInTheDocument();
    screen
      .getAllByText('Project')
      .forEach(scope => expect(scope.closest('.cds--tag')).toBeNull());
    expect(screen.getByText('Custom').closest('.cds--tag')).toHaveClass(
      'cds--tag--purple',
    );
    expect(screen.getByText('Built-in').closest('.cds--tag')).toHaveClass(
      'cds--tag--cool-gray',
    );
    const projectTokenTag = screen
      .getByText('Project token')
      .closest('.cds--tag');
    const userTag = screen.getByText('User').closest('.cds--tag');
    expect(projectTokenTag).toHaveClass('cds--tag--gray');
    expect(projectTokenTag?.querySelector('svg')).toBeInTheDocument();
    expect(userTag).toHaveClass('cds--tag--gray');
    expect(userTag?.querySelector('svg')).toBeInTheDocument();
    expect(screen.getByRole('columnheader', { name: 'Created' })).toHaveClass(
      'min-w-36',
      'whitespace-nowrap',
    );
    expect(screen.getByRole('columnheader', { name: 'Updated' })).toHaveClass(
      'min-w-36',
      'whitespace-nowrap',
    );
    expect(screen.getAllByText('date:1')[0].closest('td')).toHaveClass(
      'min-w-36',
      'whitespace-nowrap',
      'text-[13px]',
    );
  });

  it('renders role-qualified inline actions and routes only custom Edit', async () => {
    const user = userEvent.setup();
    renderPage();

    expect(await screen.findByText('Support manager')).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: 'View Support manager' }),
    ).toBeEnabled();
    expect(
      screen.getByRole('button', { name: 'Edit Support manager' }),
    ).toBeEnabled();
    expect(
      screen.getByRole('button', { name: 'View Organization owner' }),
    ).toBeEnabled();
    expect(
      screen.getByRole('button', { name: 'Edit Organization owner' }),
    ).toBeDisabled();
    expect(
      screen.getByRole('button', { name: 'View Support manager' }),
    ).toHaveClass('cds--btn--md', 'cds--btn--ghost');
    expect(screen.getAllByText('-')[0].closest('td')).toHaveClass(
      'min-w-36',
      'whitespace-nowrap',
      'text-[13px]',
    );

    await user.click(
      screen.getByRole('button', { name: 'View Support manager' }),
    );
    expect(screen.getByText('Viewing role 1')).toBeInTheDocument();
    expect(screen.getByTestId('location-path')).toHaveTextContent(
      '/organization/roles',
    );

    await user.click(
      screen.getByRole('button', { name: 'Edit Support manager' }),
    );
    expect(await screen.findByText('Role detail route')).toBeInTheDocument();
  });

  it('keeps the catalog URL while locally controlling role creation', async () => {
    const user = userEvent.setup();
    renderPage();
    await screen.findByText('Support manager');

    await user.click(screen.getByRole('button', { name: 'Create role' }));
    expect(
      screen.getByRole('region', { name: 'Create role modal' }),
    ).toBeInTheDocument();
    expect(screen.getByTestId('location-path')).toHaveTextContent(
      '/organization/roles',
    );

    await user.click(screen.getByRole('button', { name: 'Cancel creation' }));
    expect(
      screen.queryByRole('region', { name: 'Create role modal' }),
    ).not.toBeInTheDocument();
    expect(screen.getByTestId('location-path')).toHaveTextContent(
      '/organization/roles',
    );
  });

  it('consumes created-role state, refreshes, and opens the panel independently', async () => {
    mockListAccessRoles
      .mockRejectedValueOnce({ message: 'Catalog unavailable' })
      .mockRejectedValueOnce({ message: 'Catalog unavailable' });
    renderPage({
      pathname: '/organization/roles',
      state: { createdRoleDefinitionId: 'new-role' },
    });

    expect(
      await screen.findByText('Viewing role new-role'),
    ).toBeInTheDocument();
    expect(await screen.findByText('Catalog unavailable')).toBeInTheDocument();
    await waitFor(() => expect(mockListAccessRoles).toHaveBeenCalledTimes(2));
  });

  it('keeps the latest search result when an older request finishes last', async () => {
    const user = userEvent.setup();
    let resolveSearch: (value: unknown) => void = () => undefined;
    let resolveFilter: (value: unknown) => void = () => undefined;
    renderPage();
    await screen.findByText('Support manager');
    mockListAccessRoles
      .mockImplementationOnce(
        () => new Promise(resolve => (resolveSearch = resolve)),
      )
      .mockImplementationOnce(
        () => new Promise(resolve => (resolveFilter = resolve)),
      );

    const search = screen.getByPlaceholderText(
      'Search for name, scope, subject, type',
    );
    await user.click(search);
    await user.click(screen.getByRole('button', { name: 'name contains' }));
    await user.type(screen.getByRole('textbox'), 'n{Enter}');
    await user.click(screen.getByRole('textbox'));
    await user.click(screen.getByRole('button', { name: 'scope string' }));
    await user.click(screen.getByRole('button', { name: 'project string' }));
    expect(mockListAccessRoles).toHaveBeenLastCalledWith(
      expect.objectContaining({
        page: 1,
        query: 'n',
        scope: 'ROLE_SCOPE_PROJECT',
      }),
    );
    resolveFilter(response([role('3', 'New role')]));
    expect(await screen.findByText('New role')).toBeInTheDocument();
    resolveSearch(response([role('1', 'Old role')]));
    await waitFor(() =>
      expect(screen.queryByText('Old role')).not.toBeInTheDocument(),
    );
  });

  it('renders a retry action after a permission failure', async () => {
    const user = userEvent.setup();
    mockListAccessRoles.mockRejectedValue({ code: 7, message: 'denied' });
    renderPage();

    expect(
      await screen.findByText('You do not have permission to manage roles.'),
    ).toBeInTheDocument();
    mockListAccessRoles.mockResolvedValue(response([]));
    await user.click(screen.getByRole('button', { name: 'Try again' }));
    expect(await screen.findByText('No roles found')).toBeInTheDocument();
  });
});
