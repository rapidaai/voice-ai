import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { createRef } from 'react';
import { RoleDefinitionType, RoleScope, RoleSubject } from '@rapidaai/react';
import { RoleDetailsSidePanel } from './role-details-side-panel';

const mockGetAccessRole = jest.fn();

jest.mock('@/hooks/use-credential', () => ({
  useCredential: () => ['user-1', 'token-1'],
}));

jest.mock('@/clients', () => ({
  getAccessRole: (...args: unknown[]) => mockGetAccessRole(...args),
}));

jest.mock('@/app/components/dialogs/shared', () => ({
  OverviewRow: ({
    children,
    label,
  }: {
    children: React.ReactNode;
    label: string;
  }) => <div data-testid={`overview-${label}`}>{children}</div>,
  RightSideModal: ({
    children,
    className,
    label,
    modalOpen,
    setModalOpen,
    title,
  }: {
    children: React.ReactNode;
    className?: string;
    label: string;
    modalOpen: boolean;
    setModalOpen: (open: boolean) => void;
    title?: string;
  }) =>
    modalOpen ? (
      <aside
        aria-label={label}
        className={className}
        data-testid="right-side-modal"
      >
        <h2>{title}</h2>
        <button type="button" onClick={() => setModalOpen(false)}>
          Close
        </button>
        {children}
      </aside>
    ) : null,
}));

jest.mock('@/app/components/ui/primitives', () => ({
  PrimaryButton: ({ children, ...props }: any) => (
    <button {...props}>{children}</button>
  ),
  Tabs: ({
    children,
    onChange,
    selectedIndex,
    tabs,
    'aria-label': ariaLabel,
  }: {
    children: React.ReactNode;
    onChange: (index: number) => void;
    selectedIndex: number;
    tabs: string[];
    'aria-label': string;
  }) => (
    <section aria-label={ariaLabel}>
      <div role="tablist">
        {tabs.map((tab, index) => (
          <button
            key={tab}
            type="button"
            role="tab"
            aria-selected={selectedIndex === index}
            onClick={() => onChange(index)}
          >
            {tab}
          </button>
        ))}
      </div>
      {Array.isArray(children) ? children[selectedIndex] : children}
    </section>
  ),
  Text: ({ as: Tag = 'span', children, variant: _variant, ...props }: any) => (
    <Tag {...props}>{children}</Tag>
  ),
}));

const permission = (description = 'Read project records') => ({
  getId: () => '34',
  getName: () => 'Project Read',
  getKey: () => 'project.read',
  getDescription: () => description,
});

const role = (permissionDescription = 'Read project records') => ({
  getId: () => '101',
  getName: () => 'Support manager',
  getDescription: () => 'Manages support workflows',
  getKey: () => 'support_manager',
  getScope: () => RoleScope.ROLE_SCOPE_PROJECT,
  getSubject: () => RoleSubject.ROLE_SUBJECT_USER,
  getDefinitiontype: () => RoleDefinitionType.ROLE_DEFINITION_TYPE_CUSTOM,
  getCreateddate: () => ({ getSeconds: () => 1, getNanos: () => 0 }),
  getUpdateddate: () => ({ getSeconds: () => 2, getNanos: () => 0 }),
  getCreatedactor: () => ({ getDisplayname: () => 'Ada Lovelace' }),
  getUpdatedactor: () => ({ getDisplayname: () => 'Grace Hopper' }),
  getPermissionsList: () => [permission(permissionDescription)],
});

const successResponse = (currentRole = role()) => ({
  getSuccess: () => true,
  getRole: () => currentRole,
  getCode: () => 200,
});

describe('RoleDetailsSidePanel', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockGetAccessRole.mockResolvedValue(successResponse());
  });

  it('loads the canonical role and renders all role and permission details', async () => {
    const user = userEvent.setup();
    const launcherButtonRef = createRef<HTMLButtonElement>();
    render(
      <RoleDetailsSidePanel
        open
        roleDefinitionId="101"
        launcherButtonRef={launcherButtonRef}
        onClose={jest.fn()}
      />,
    );

    expect(
      await screen.findByRole('heading', { name: 'Support manager' }),
    ).toBeInTheDocument();
    expect(mockGetAccessRole).toHaveBeenCalledWith({
      roleDefinitionId: '101',
      auth: { token: 'token-1', userId: 'user-1' },
    });
    expect(screen.getByTestId('right-side-modal')).toHaveClass('w-[580px]');
    expect(screen.getByRole('tab', { name: 'Overview' })).toHaveAttribute(
      'aria-selected',
      'true',
    );
    expect(screen.getByTestId('overview-Key')).toHaveTextContent(
      'support_manager',
    );
    expect(screen.getByText('Manages support workflows')).toBeInTheDocument();
    expect(screen.getByText('support_manager')).toBeInTheDocument();
    expect(screen.getByText('Custom')).toBeInTheDocument();
    expect(screen.getByText('Project')).toBeInTheDocument();
    expect(screen.getByText('User')).toBeInTheDocument();
    expect(screen.getByText('Ada Lovelace')).toBeInTheDocument();
    expect(screen.getByText('Grace Hopper')).toBeInTheDocument();
    expect(screen.queryByText('Project Read')).not.toBeInTheDocument();

    await user.click(screen.getByRole('tab', { name: 'Permissions' }));
    expect(screen.getByText('Project Read')).toBeInTheDocument();
    expect(screen.getByText('project.read')).toBeInTheDocument();
    expect(screen.getByText('Read project records')).toBeInTheDocument();
  });

  it('shows explicit fallbacks for missing descriptions and permissions', async () => {
    const user = userEvent.setup();
    mockGetAccessRole.mockResolvedValueOnce(
      successResponse({
        ...role(''),
        getDescription: () => '',
        getPermissionsList: () => [],
      }),
    );

    render(
      <RoleDetailsSidePanel
        open
        roleDefinitionId="101"
        launcherButtonRef={createRef<HTMLButtonElement>()}
        onClose={jest.fn()}
      />,
    );

    expect(await screen.findByText('No role description.')).toBeInTheDocument();
    await user.click(screen.getByRole('tab', { name: 'Permissions' }));
    expect(screen.getByText('No permissions assigned.')).toBeInTheDocument();
  });

  it('retries transient failures but never offers Retry for not-found', async () => {
    const user = userEvent.setup();
    mockGetAccessRole
      .mockRejectedValueOnce({ code: 13, message: 'Request unavailable' })
      .mockResolvedValueOnce(successResponse())
      .mockRejectedValueOnce({ code: 5, message: 'role not found' });
    const launcherButtonRef = createRef<HTMLButtonElement>();
    const { rerender } = render(
      <RoleDetailsSidePanel
        open
        roleDefinitionId="101"
        launcherButtonRef={launcherButtonRef}
        onClose={jest.fn()}
      />,
    );

    expect(await screen.findByText('Request unavailable')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Retry' }));
    expect(await screen.findByText('Support manager')).toBeInTheDocument();

    rerender(
      <RoleDetailsSidePanel
        open
        roleDefinitionId="stale-role"
        launcherButtonRef={launcherButtonRef}
        onClose={jest.fn()}
      />,
    );
    expect(
      await screen.findByText(/This role is no longer available\./),
    ).toBeInTheDocument();
    expect(
      screen.queryByRole('button', { name: 'Retry' }),
    ).not.toBeInTheDocument();
  });

  it('shows denial distinctly and ignores an older role response', async () => {
    let resolveFirst: (value: unknown) => void = () => undefined;
    mockGetAccessRole
      .mockImplementationOnce(
        () => new Promise(resolve => (resolveFirst = resolve)),
      )
      .mockRejectedValueOnce({ code: 7, message: 'denied' });
    const launcherButtonRef = createRef<HTMLButtonElement>();
    const { rerender } = render(
      <RoleDetailsSidePanel
        open
        roleDefinitionId="101"
        launcherButtonRef={launcherButtonRef}
        onClose={jest.fn()}
      />,
    );

    rerender(
      <RoleDetailsSidePanel
        open
        roleDefinitionId="102"
        launcherButtonRef={launcherButtonRef}
        onClose={jest.fn()}
      />,
    );
    expect(
      await screen.findByText('You do not have permission to view this role.'),
    ).toBeInTheDocument();

    resolveFirst(successResponse());
    await waitFor(() =>
      expect(screen.queryByText('Support manager')).not.toBeInTheDocument(),
    );
  });

  it('closes without navigation and returns focus to the launcher', async () => {
    const user = userEvent.setup();
    const onClose = jest.fn();
    const launcherButtonRef = createRef<HTMLButtonElement>();
    render(
      <>
        <button ref={launcherButtonRef} type="button">
          View Support manager
        </button>
        <RoleDetailsSidePanel
          open
          roleDefinitionId="101"
          launcherButtonRef={launcherButtonRef}
          onClose={onClose}
        />
      </>,
    );
    await screen.findByText('Support manager');

    await user.click(screen.getByRole('button', { name: 'Close' }));
    expect(onClose).toHaveBeenCalledTimes(1);
    expect(launcherButtonRef.current).toHaveFocus();
  });
});
