import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import {
  PermissionDefinition,
  RoleDefinition,
  RoleDefinitionType,
} from '@rapidaai/react';
import { DeleteRoleDialog, EditRoleModal, RoleDetailPage } from './detail';

const mockUpdateAccessRole = jest.fn();
const mockReduceAccessRolePermissions = jest.fn();
const mockArchiveAccessRole = jest.fn();
const mockGetAccessRole = jest.fn();
const mockToastError = jest.fn();

jest.mock('@/hooks/use-credential', () => ({
  useCredential: () => ['user-1', 'token-1'],
}));

jest.mock('@/clients', () => ({
  getAccessRole: (...args: unknown[]) => mockGetAccessRole(...args),
  updateAccessRole: (...args: unknown[]) => mockUpdateAccessRole(...args),
  reduceAccessRolePermissions: (...args: unknown[]) =>
    mockReduceAccessRolePermissions(...args),
  archiveAccessRole: (...args: unknown[]) => mockArchiveAccessRole(...args),
}));

jest.mock('react-hot-toast/headless', () => ({
  __esModule: true,
  default: { error: (...args: unknown[]) => mockToastError(...args) },
}));

function role(
  definitionType: Parameters<
    RoleDefinition['setDefinitiontype']
  >[0] = RoleDefinitionType.ROLE_DEFINITION_TYPE_CUSTOM,
) {
  const permission = new PermissionDefinition();
  permission.setId('34');
  permission.setName('Project Read');
  permission.setKey('project.read');

  const value = new RoleDefinition();
  value.setId('101');
  value.setName('Support manager');
  value.setDescription('Manages support');
  value.setDefinitiontype(definitionType);
  value.setPermissionsList([permission]);
  return value;
}

describe('role action dialogs', () => {
  beforeAll(() => {
    HTMLElement.prototype.scrollIntoView = jest.fn();
    Object.defineProperty(window, 'matchMedia', {
      configurable: true,
      value: jest.fn().mockReturnValue({
        matches: false,
        addEventListener: jest.fn(),
        removeEventListener: jest.fn(),
      }),
    });
  });

  beforeEach(() => {
    jest.clearAllMocks();
    mockUpdateAccessRole.mockResolvedValue({ getSuccess: () => true });
    mockReduceAccessRolePermissions.mockResolvedValue({
      getSuccess: () => true,
    });
    mockArchiveAccessRole.mockResolvedValue({ getSuccess: () => true });
    mockGetAccessRole.mockResolvedValue({
      getSuccess: () => true,
      getRole: () => role(),
    });
  });

  it('loads the complete custom role without exposing permission editing', async () => {
    const summaryRole = role();
    summaryRole.clearPermissionsList();

    render(
      <EditRoleModal
        open
        role={summaryRole}
        onClose={jest.fn()}
        onUpdated={jest.fn()}
      />,
    );

    expect(mockGetAccessRole).toHaveBeenCalledWith({
      roleDefinitionId: '101',
      auth: { token: 'token-1', userId: 'user-1' },
    });
    const dialog = screen.getByRole('dialog', { name: 'Edit role' });
    const name = await within(dialog).findByLabelText('Name');
    expect(name).toHaveValue('Support manager');
    expect(name).toHaveFocus();
    expect(
      within(dialog).queryByRole('tab', { name: 'Permissions' }),
    ).not.toBeInTheDocument();
    expect(within(dialog).queryByRole('checkbox')).not.toBeInTheDocument();
    expect(
      within(dialog).queryByRole('button', { name: /Remove permissions/i }),
    ).not.toBeInTheDocument();
  });

  it('updates custom-role metadata in the edit modal', async () => {
    const user = userEvent.setup();
    const onClose = jest.fn();
    const onUpdated = jest.fn();
    render(
      <EditRoleModal
        open
        role={role()}
        onClose={onClose}
        onUpdated={onUpdated}
      />,
    );

    const dialog = screen.getByRole('dialog', { name: 'Edit role' });
    const name = await within(dialog).findByLabelText('Name');
    await user.clear(name);
    await user.type(name, 'Support lead');
    await user.click(
      within(dialog).getByRole('button', { name: 'Save changes' }),
    );

    expect(mockUpdateAccessRole).toHaveBeenCalledWith({
      roleDefinitionId: '101',
      name: 'Support lead',
      description: 'Manages support',
      auth: { token: 'token-1', userId: 'user-1' },
    });
    await waitFor(() => expect(onUpdated).toHaveBeenCalledTimes(1));
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it('keeps permission controls unavailable after a metadata failure', async () => {
    const user = userEvent.setup();
    const onClose = jest.fn();
    const onUpdated = jest.fn();
    mockUpdateAccessRole.mockResolvedValueOnce({
      getSuccess: () => false,
      getError: () => ({ getHumanmessage: () => 'Metadata rejected' }),
    });
    render(
      <EditRoleModal
        open
        role={role()}
        onClose={onClose}
        onUpdated={onUpdated}
      />,
    );

    const dialog = screen.getByRole('dialog', { name: 'Edit role' });
    await within(dialog).findByLabelText('Name');
    await user.click(
      within(dialog).getByRole('button', { name: 'Save changes' }),
    );
    expect(
      await within(dialog).findByText('Metadata rejected'),
    ).toBeInTheDocument();
    expect(within(dialog).queryByRole('checkbox')).not.toBeInTheDocument();
    expect(mockReduceAccessRolePermissions).not.toHaveBeenCalled();
    expect(onUpdated).not.toHaveBeenCalled();
    expect(onClose).not.toHaveBeenCalled();
  });

  it('uses the shared name confirmation before archiving from Delete', async () => {
    const user = userEvent.setup();
    const onClose = jest.fn();
    const onDeleted = jest.fn();
    render(
      <DeleteRoleDialog
        open
        role={role()}
        onClose={onClose}
        onDeleted={onDeleted}
      />,
    );

    const dialog = screen.getByRole('dialog', {
      name: 'Delete role',
    });
    const deleteButton = within(dialog).getByRole('button', {
      name: 'Delete role',
    });
    expect(deleteButton).toBeDisabled();
    expect(
      within(dialog).getByPlaceholderText('Support manager'),
    ).toBeInTheDocument();
    await user.type(
      within(dialog).getByPlaceholderText('Support manager'),
      'Support manager',
    );
    await user.click(deleteButton);

    expect(mockArchiveAccessRole).toHaveBeenCalledWith({
      roleDefinitionId: '101',
      reason: 'Deleted from Roles & Permissions.',
      auth: { token: 'token-1', userId: 'user-1' },
    });
    await waitFor(() => expect(onDeleted).toHaveBeenCalledTimes(1));
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it('keeps the Delete dialog open when archive is rejected', async () => {
    const user = userEvent.setup();
    mockArchiveAccessRole.mockResolvedValueOnce({
      getSuccess: () => false,
      getError: () => ({
        getHumanmessage: () => 'Move active assignments before deleting.',
      }),
    });
    render(
      <DeleteRoleDialog
        open
        role={role()}
        onClose={jest.fn()}
        onDeleted={jest.fn()}
      />,
    );

    const dialog = screen.getByRole('dialog', {
      name: 'Delete role',
    });
    await user.type(
      within(dialog).getByPlaceholderText('Support manager'),
      'Support manager',
    );
    await user.click(
      within(dialog).getByRole('button', { name: 'Delete role' }),
    );

    await waitFor(() =>
      expect(mockToastError).toHaveBeenCalledWith(
        'Move active assignments before deleting.',
      ),
    );
    expect(dialog).toBeInTheDocument();
  });

  it('fails closed when a built-in role reaches a mutation dialog', async () => {
    const user = userEvent.setup();
    mockGetAccessRole.mockResolvedValueOnce({
      getSuccess: () => true,
      getRole: () => role(RoleDefinitionType.ROLE_DEFINITION_TYPE_BUILT_IN),
    });
    render(
      <EditRoleModal
        open
        role={role(RoleDefinitionType.ROLE_DEFINITION_TYPE_BUILT_IN)}
        onClose={jest.fn()}
        onUpdated={jest.fn()}
      />,
    );

    expect(
      await screen.findByText('Built-in roles cannot be edited.'),
    ).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Save changes' }));
    expect(mockUpdateAccessRole).not.toHaveBeenCalled();
  });

  it('shows a load error and retries the complete role request', async () => {
    const user = userEvent.setup();
    mockGetAccessRole
      .mockRejectedValueOnce({ message: 'Role service unavailable' })
      .mockResolvedValueOnce({
        getSuccess: () => true,
        getRole: () => role(),
      });

    render(
      <EditRoleModal
        open
        role={role()}
        onClose={jest.fn()}
        onUpdated={jest.fn()}
      />,
    );

    expect(
      await screen.findByText('Role service unavailable'),
    ).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Retry' }));

    expect(await screen.findByLabelText('Name')).toHaveValue('Support manager');
    expect(mockGetAccessRole).toHaveBeenCalledTimes(2);
  });
});

describe('RoleDetailPage', () => {
  it('redirects legacy detail URLs to the role catalog', async () => {
    render(
      <MemoryRouter initialEntries={['/organization/roles/101']}>
        <Routes>
          <Route
            path="/organization/roles/:roleDefinitionId"
            element={<RoleDetailPage />}
          />
          <Route path="/organization/roles" element={<div>Role catalog</div>} />
        </Routes>
      </MemoryRouter>,
    );

    expect(await screen.findByText('Role catalog')).toBeInTheDocument();
  });
});
