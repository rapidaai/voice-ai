import { act, render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import {
  MemoryRouter,
  Route,
  Routes,
  useLocation,
  useNavigate,
} from 'react-router-dom';
import { CreateRoleModal } from './create';

const mockListAccessPermissions = jest.fn();
const mockCreateAccessRole = jest.fn();

jest.mock('@/hooks/use-credential', () => ({
  useCredential: () => ['user-1', 'token-1'],
}));

jest.mock('@/clients', () => ({
  listAccessPermissions: (...args: unknown[]) =>
    mockListAccessPermissions(...args),
  createAccessRole: (...args: unknown[]) => mockCreateAccessRole(...args),
}));

jest.mock('@carbon/ibm-products', () => {
  const React = jest.requireActual<typeof import('react')>('react');
  const TearsheetContext = React.createContext<{ onClose?: () => void }>({});
  const AddSelectContext = React.createContext<{
    onItemSelect?: (itemId: string, selected: boolean, value: string) => void;
    selectedItems?: Set<string>;
  }>({});

  function Tearsheet({
    children,
    className,
    onClose,
    open,
  }: {
    children: React.ReactNode;
    className?: string;
    onClose?: () => void;
    open?: boolean;
  }) {
    if (!open) return null;
    return (
      <TearsheetContext.Provider value={{ onClose }}>
        <div role="dialog" aria-label="Create Role" className={className}>
          {children}
        </div>
      </TearsheetContext.Provider>
    );
  }

  function TearsheetHeader({
    children,
    closeIconDescription,
  }: {
    children: React.ReactNode;
    closeIconDescription?: string;
  }) {
    const { onClose } = React.useContext(TearsheetContext);
    return (
      <header>
        {children}
        <button type="button" onClick={onClose}>
          {closeIconDescription || 'Close'}
        </button>
      </header>
    );
  }
  Tearsheet.Header = TearsheetHeader;
  Tearsheet.HeaderContent = ({
    description,
    label,
    title,
  }: {
    description?: React.ReactNode;
    label?: React.ReactNode;
    title: React.ReactNode;
  }) => (
    <div>
      {label}
      <h1>{title}</h1>
      {description && <p>{description}</p>}
    </div>
  );
  Tearsheet.Influencer = ({ children }: { children: React.ReactNode }) => (
    <aside>{children}</aside>
  );
  Tearsheet.Body = ({ children }: { children: React.ReactNode }) => (
    <div>{children}</div>
  );
  Tearsheet.MainContent = ({
    children,
    isFlush,
  }: {
    children: React.ReactNode;
    isFlush?: boolean;
  }) => <main data-is-flush={isFlush}>{children}</main>;
  Tearsheet.Footer = ({
    actions,
    buttonSize,
    children,
  }: {
    actions: Array<{
      disabled?: boolean;
      isExpressive?: boolean;
      key: string;
      label: string;
      onClick: () => void;
    }>;
    buttonSize?: string;
    children?: React.ReactNode;
  }) => (
    <footer data-button-size={buttonSize}>
      {children}
      {actions.map(({ isExpressive = true, ...action }) => (
        <button
          data-is-expressive={isExpressive}
          type="button"
          disabled={action.disabled}
          key={action.key}
          onClick={action.onClick}
        >
          {action.label}
        </button>
      ))}
    </footer>
  );

  function AddSelect({
    children,
    onItemSelect,
    selectedItems,
  }: {
    children: React.ReactNode;
    onItemSelect?: (itemId: string, selected: boolean, value: string) => void;
    selectedItems?: Set<string>;
  }) {
    return (
      <AddSelectContext.Provider value={{ onItemSelect, selectedItems }}>
        <div data-testid="inline-permission-selector">{children}</div>
      </AddSelectContext.Provider>
    );
  }

  AddSelect.Body = ({
    children,
    globalSearchLabel,
    globalSearchPlaceholder,
    itemCount,
    itemsLabel,
    onSearch,
  }: {
    children: React.ReactNode;
    globalSearchLabel?: string;
    globalSearchPlaceholder?: string;
    itemCount?: number;
    itemsLabel?: string;
    onSearch?: (value: string) => void;
  }) => (
    <section>
      <label>
        {globalSearchLabel}
        <input
          placeholder={globalSearchPlaceholder}
          onChange={event => onSearch?.(event.target.value)}
        />
      </label>
      <p>
        {itemsLabel} {itemCount}
      </p>
      {children}
    </section>
  );

  function AddSelectRow({
    itemId,
    subtitle,
    title,
    value,
  }: {
    itemId: string;
    subtitle?: string;
    title: string;
    value: string;
  }) {
    const { onItemSelect, selectedItems } = React.useContext(AddSelectContext);
    const selected = selectedItems?.has(itemId) || false;
    return (
      <div role="row" aria-selected={selected}>
        <input
          aria-label={title}
          checked={selected}
          type="checkbox"
          onChange={() => onItemSelect?.(itemId, !selected, value)}
        />
        {title}
        {subtitle}
      </div>
    );
  }

  AddSelect.Row = AddSelectRow;

  AddSelect.Column = ({ children }: { children: React.ReactNode }) => (
    <div>{children}</div>
  );

  AddSelect.SelectionSummary = ({
    children,
    emptyState,
    selectedItemCount = 0,
    title,
  }: {
    children: React.ReactNode;
    emptyState?: React.ReactNode;
    selectedItemCount?: number;
    title?: string;
  }) => (
    <aside aria-label={title}>
      <h3>{title}</h3>
      <span>{selectedItemCount}</span>
      {selectedItemCount > 0 ? children : emptyState}
    </aside>
  );

  AddSelect.SelectionSummaryItem = ({
    item,
    onRemove,
    removeButtonLabel,
  }: {
    item: { id: string; subtitle?: string; title: string };
    onRemove?: (itemId: string) => void;
    removeButtonLabel?: string;
  }) => (
    <div>
      {item.title}
      {item.subtitle}
      <button type="button" onClick={() => onRemove?.(item.id)}>
        {removeButtonLabel}
      </button>
    </div>
  );

  const EmptyState = ({
    subtitle,
    title,
  }: {
    subtitle?: string;
    title: string;
  }) => (
    <div>
      <h4>{title}</h4>
      {subtitle}
    </div>
  );

  return {
    NoDataEmptyState: EmptyState,
    NotFoundEmptyState: EmptyState,
    preview__AddSelect: AddSelect,
    preview__Tearsheet: Tearsheet,
  };
});

jest.mock('@/app/components/app-shell/helmet', () => ({ Helmet: () => null }));

const permission = (id: string, name: string) => ({
  getId: () => id,
  getName: () => name,
  getKey: () => name.toLowerCase().replaceAll(' ', '.'),
  getDescription: () => `${name} description`,
});

const permissionResponse = (
  permissions: ReturnType<typeof permission>[],
  total: number,
) => ({
  getSuccess: () => true,
  getPermissionsList: () => permissions,
  getPaginated: () => ({ getTotalitem: () => total }),
});

function CreateRoute() {
  const navigate = useNavigate();
  return (
    <>
      <button type="button" onClick={() => navigate('/outside')}>
        Leave role creation
      </button>
      <CreateRoleModal
        open
        onClose={() => navigate('/organization/roles', { replace: true })}
      />
    </>
  );
}

function RoleCatalog() {
  const location = useLocation();
  const createdRoleDefinitionId = (
    location.state as { createdRoleDefinitionId?: string } | null
  )?.createdRoleDefinitionId;
  return (
    <div>
      Role catalog
      {createdRoleDefinitionId && <span>{createdRoleDefinitionId}</span>}
    </div>
  );
}

function renderPage() {
  return render(
    <MemoryRouter initialEntries={['/test-create-role']}>
      <Routes>
        <Route path="/test-create-role" element={<CreateRoute />} />
        <Route path="/organization/roles" element={<RoleCatalog />} />
        <Route path="/outside" element={<div>Outside role management</div>} />
      </Routes>
    </MemoryRouter>,
  );
}

describe('CreateRoleModal', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockListAccessPermissions
      .mockResolvedValueOnce(
        permissionResponse([permission('1', 'Organization Read')], 2),
      )
      .mockResolvedValueOnce(
        permissionResponse([permission('2', 'Organization Update')], 2),
      );
    mockCreateAccessRole.mockResolvedValue({
      getSuccess: () => true,
      getRole: () => ({ getId: () => '101' }),
    });
  });

  it('loads every permission page and creates a role with selected permissions', async () => {
    const user = userEvent.setup();
    renderPage();

    expect(screen.getByRole('dialog', { name: 'Create Role' })).toHaveClass(
      'role-create-tearsheet',
    );
    expect(
      screen.getByRole('button', { name: 'Close create role' }),
    ).toBeVisible();
    const footer = screen
      .getByRole('button', { name: 'Cancel' })
      .closest('footer');
    expect(footer).not.toHaveAttribute('data-button-size');
    within(footer!)
      .getAllByRole('button')
      .forEach(button => {
        expect(button).toHaveAttribute('data-is-expressive', 'true');
      });
    const steps = screen.getByRole('list', { name: 'Create role steps' });
    expect(within(steps).getByText('Role Details')).toBeInTheDocument();
    expect(
      within(steps).getByText('Define access type and scope'),
    ).toBeInTheDocument();
    expect(within(steps).getByText('Permissions')).toBeInTheDocument();
    expect(
      screen.getByText(
        'Create a custom role for your organization and choose the permissions it grants.',
      ),
    ).toBeVisible();
    await waitFor(() =>
      expect(mockListAccessPermissions).toHaveBeenCalledTimes(2),
    );
    expect(screen.getByLabelText('Role name')).toHaveAttribute(
      'placeholder',
      'Enter role name',
    );
    expect(screen.getByLabelText('Role key')).toHaveAttribute(
      'placeholder',
      'Enter role key',
    );
    expect(screen.getByLabelText('Description')).toHaveAttribute(
      'placeholder',
      'Enter role description',
    );
    await user.click(
      screen.getByRole('button', { name: 'Role key information' }),
    );
    expect(
      screen.getByText(/Lowercase letters, numbers, and underscores/),
    ).toBeVisible();
    await user.click(
      screen.getByRole('button', { name: 'Access type information' }),
    );
    expect(
      screen.getByText(/For signed-in users.*organization or project scope/),
    ).toBeVisible();
    expect(
      screen.getByText(/For API integrations.*Project scope only/),
    ).toBeVisible();
    expect(screen.getByLabelText('Access type')).toHaveDisplayValue(
      'Console access (user)',
    );
    expect(mockListAccessPermissions).toHaveBeenNthCalledWith(
      2,
      expect.objectContaining({ page: 2, pageSize: 100 }),
    );
    await user.type(screen.getByLabelText('Role name'), 'Support manager');
    await user.type(screen.getByLabelText('Role key'), 'support_manager');
    await user.click(screen.getByRole('button', { name: 'Next' }));
    expect(screen.getByRole('main')).toHaveAttribute('data-is-flush', 'true');
    expect(
      screen.queryByRole('dialog', { name: 'Select Permissions' }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByText(
        'Select the permissions this role grants. Permissions can be removed later, but cannot yet be added after creation.',
      ),
    ).not.toBeInTheDocument();
    await screen.findByRole('checkbox', { name: 'Organization Update' });
    await user.type(screen.getByPlaceholderText('Search permissions'), 'read');
    expect(
      screen.queryByRole('checkbox', { name: 'Organization Update' }),
    ).not.toBeInTheDocument();
    await user.clear(screen.getByPlaceholderText('Search permissions'));
    await user.click(
      screen.getByRole('checkbox', { name: 'Organization Read' }),
    );
    expect(
      screen.getByRole('checkbox', { name: 'Organization Read' }),
    ).toBeChecked();
    const selectedPermissions = screen.getByRole('complementary', {
      name: 'Selected Permissions',
    });
    expect(within(selectedPermissions).getByText('1')).toBeVisible();
    expect(
      within(selectedPermissions).getByRole('button', {
        name: 'Remove Organization Read',
      }),
    ).toBeVisible();
    await user.click(screen.getByRole('button', { name: 'Create role' }));

    await waitFor(() =>
      expect(mockCreateAccessRole).toHaveBeenCalledWith(
        expect.objectContaining({
          key: 'support_manager',
          permissionDefinitionIds: ['1'],
        }),
      ),
    );
    expect(await screen.findByText('Role catalog')).toBeInTheDocument();
    expect(screen.getByText('101')).toBeInTheDocument();
  });

  it('rejects an invalid role key without sending a request', async () => {
    const user = userEvent.setup();
    renderPage();
    await waitFor(() =>
      expect(mockListAccessPermissions).toHaveBeenCalledTimes(2),
    );
    await user.type(screen.getByLabelText('Role name'), 'Support manager');
    await user.type(screen.getByLabelText('Role key'), 'Invalid Key');
    await user.click(screen.getByRole('button', { name: 'Next' }));

    const notification = (await screen.findByText('Error')).closest(
      '.cds--inline-notification',
    );
    expect(notification?.closest('footer')).toBeInTheDocument();
    expect(notification).toHaveTextContent(
      'Key must start with a lowercase letter and contain 3 to 64',
    );
    expect(mockCreateAccessRole).not.toHaveBeenCalled();
  });

  it('blocks creation after a catalog failure and retries the complete load', async () => {
    const user = userEvent.setup();
    mockListAccessPermissions.mockReset();
    mockListAccessPermissions
      .mockResolvedValueOnce({
        getSuccess: () => false,
        getError: () => ({ getHumanmessage: () => 'Catalog unavailable' }),
      })
      .mockResolvedValueOnce(
        permissionResponse([permission('1', 'Organization Read')], 1),
      );
    renderPage();

    await user.type(screen.getByLabelText('Role name'), 'Support manager');
    await user.type(screen.getByLabelText('Role key'), 'support_manager');
    await user.click(screen.getByRole('button', { name: 'Next' }));
    const notification = (await screen.findByText('Error')).closest(
      '.cds--actionable-notification',
    );
    expect(notification?.closest('footer')).toBeInTheDocument();
    expect(notification).toHaveTextContent('Catalog unavailable');
    expect(screen.getByRole('button', { name: 'Create role' })).toBeDisabled();
    await user.click(screen.getByRole('button', { name: 'Retry permissions' }));

    expect(
      await screen.findByPlaceholderText('Search permissions'),
    ).toBeEnabled();
    expect(screen.getByRole('button', { name: 'Create role' })).toBeEnabled();
  });

  it('forces project scope for project-token roles', async () => {
    const user = userEvent.setup();
    mockListAccessPermissions.mockReset();
    mockListAccessPermissions
      .mockResolvedValueOnce(
        permissionResponse([permission('1', 'Organization Read')], 1),
      )
      .mockResolvedValueOnce(
        permissionResponse([permission('34', 'Project Read')], 1),
      );
    renderPage();
    await waitFor(() =>
      expect(mockListAccessPermissions).toHaveBeenCalledTimes(1),
    );

    await user.selectOptions(
      screen.getByLabelText('Access type'),
      'ROLE_SUBJECT_PROJECT_TOKEN',
    );

    expect(screen.getByLabelText('Access type')).toHaveDisplayValue(
      'API access (project token)',
    );

    expect(screen.getByLabelText('Scope')).toHaveValue('ROLE_SCOPE_PROJECT');
    expect(screen.getByLabelText('Scope')).toBeDisabled();
    await user.type(screen.getByLabelText('Role name'), 'Project viewer');
    await user.type(screen.getByLabelText('Role key'), 'project_viewer');
    await user.click(screen.getByRole('button', { name: 'Next' }));
    expect(
      await screen.findByRole('checkbox', { name: 'Project Read' }),
    ).toBeInTheDocument();
  });

  it('keeps form values when the role key already exists', async () => {
    const user = userEvent.setup();
    mockCreateAccessRole.mockResolvedValueOnce({
      getSuccess: () => false,
      getRole: () => undefined,
      getError: () => ({ getHumanmessage: () => 'Role key already exists' }),
    });
    renderPage();
    await waitFor(() =>
      expect(mockListAccessPermissions).toHaveBeenCalledTimes(2),
    );
    await user.type(screen.getByLabelText('Role name'), 'Support manager');
    await user.type(screen.getByLabelText('Role key'), 'support_manager');
    await user.click(screen.getByRole('button', { name: 'Next' }));
    await screen.findByPlaceholderText('Search permissions');
    await user.click(screen.getByRole('button', { name: 'Create role' }));

    const notification = (await screen.findByText('Error')).closest(
      '.cds--inline-notification',
    );
    expect(notification?.closest('footer')).toBeInTheDocument();
    expect(notification).toHaveTextContent('Role key already exists');
    await user.click(screen.getByRole('button', { name: 'Back' }));
    expect(screen.getByLabelText('Role name')).toHaveValue('Support manager');
    expect(screen.getByLabelText('Role key')).toHaveValue('support_manager');
  });

  it('does not redirect after submission finishes on an exited page', async () => {
    const user = userEvent.setup();
    let resolveCreate: (value: unknown) => void = () => undefined;
    mockCreateAccessRole.mockImplementationOnce(
      () => new Promise(resolve => (resolveCreate = resolve)),
    );
    renderPage();
    await waitFor(() =>
      expect(mockListAccessPermissions).toHaveBeenCalledTimes(2),
    );
    await user.type(screen.getByLabelText('Role name'), 'Support manager');
    await user.type(screen.getByLabelText('Role key'), 'support_manager');
    await user.click(screen.getByRole('button', { name: 'Next' }));
    await screen.findByPlaceholderText('Search permissions');
    await user.click(screen.getByRole('button', { name: 'Create role' }));
    await user.click(
      screen.getByRole('button', { name: 'Leave role creation', hidden: true }),
    );
    expect(
      await screen.findByText('Outside role management'),
    ).toBeInTheDocument();

    await act(async () => {
      resolveCreate({
        getSuccess: () => true,
        getRole: () => ({ getId: () => '101' }),
      });
    });
    expect(screen.getByText('Outside role management')).toBeInTheDocument();
    expect(screen.queryByText('Role catalog')).not.toBeInTheDocument();
  });

  it('clears selection and ignores an old scope response that finishes last', async () => {
    const user = userEvent.setup();
    let resolveOrganization: (value: unknown) => void = () => undefined;
    let resolveProject: (value: unknown) => void = () => undefined;
    mockListAccessPermissions.mockReset();
    mockListAccessPermissions
      .mockImplementationOnce(
        () => new Promise(resolve => (resolveOrganization = resolve)),
      )
      .mockImplementationOnce(
        () => new Promise(resolve => (resolveProject = resolve)),
      );
    renderPage();

    await user.selectOptions(
      screen.getByLabelText('Scope'),
      'ROLE_SCOPE_PROJECT',
    );
    await waitFor(() =>
      expect(mockListAccessPermissions).toHaveBeenCalledTimes(2),
    );
    await act(async () => {
      resolveProject(permissionResponse([permission('34', 'Project Read')], 1));
    });
    await act(async () => {
      resolveOrganization(
        permissionResponse([permission('1', 'Organization Read')], 1),
      );
    });

    await user.type(screen.getByLabelText('Role name'), 'Project viewer');
    await user.type(screen.getByLabelText('Role key'), 'project_viewer');
    await user.click(screen.getByRole('button', { name: 'Next' }));
    expect(
      await screen.findByRole('checkbox', { name: 'Project Read' }),
    ).toBeInTheDocument();
    expect(
      screen.queryByRole('checkbox', { name: 'Organization Read' }),
    ).not.toBeInTheDocument();
    expect(
      screen.getByRole('checkbox', { name: 'Project Read' }),
    ).not.toBeChecked();
  });
});
