import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Link, MemoryRouter, Route, Routes } from 'react-router-dom';
import { RoleDefinitionType, RoleScope, RoleSubject } from '@rapidaai/react';
import { RoleDetailPage } from './detail';

const mockGetAccessRole = jest.fn();
const mockUpdateAccessRole = jest.fn();
const mockReduceAccessRolePermissions = jest.fn();
const mockArchiveAccessRole = jest.fn();

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

jest.mock('@/app/components/app-shell/helmet', () => ({ Helmet: () => null }));

const permission = {
  getId: () => '34',
  getName: () => 'Project Read',
  getKey: () => 'project.read',
};

const role = (definitionType: number) => ({
  getId: () => '101',
  getName: () => 'Support manager',
  getDescription: () => 'Manages support',
  getKey: () => 'support_manager',
  getScope: () => RoleScope.ROLE_SCOPE_PROJECT,
  getSubject: () => RoleSubject.ROLE_SUBJECT_USER,
  getDefinitiontype: () => definitionType,
  getCreateddate: () => undefined,
  getUpdateddate: () => undefined,
  getCreatedactor: () => undefined,
  getUpdatedactor: () => undefined,
  getPermissionsList: () => [permission],
});

const response = (definitionType: number) => ({
  getSuccess: () => true,
  getRole: () => role(definitionType),
});

function renderPage() {
  return render(
    <MemoryRouter initialEntries={['/organization/roles/101']}>
      <Link to="/outside">Leave page</Link>
      <Routes>
        <Route
          path="/organization/roles/:roleDefinitionId"
          element={<RoleDetailPage />}
        />
        <Route path="/organization/roles" element={<div>Role catalog</div>} />
        <Route path="/outside" element={<div>Outside role management</div>} />
      </Routes>
    </MemoryRouter>,
  );
}

describe('RoleDetailPage', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockGetAccessRole.mockResolvedValue(
      response(RoleDefinitionType.ROLE_DEFINITION_TYPE_CUSTOM),
    );
    mockUpdateAccessRole.mockResolvedValue({ getSuccess: () => true });
    mockReduceAccessRolePermissions.mockResolvedValue({
      getSuccess: () => true,
    });
    mockArchiveAccessRole.mockResolvedValue({ getSuccess: () => true });
  });

  it('renders built-in roles as read-only with audit fallbacks', async () => {
    mockGetAccessRole.mockResolvedValue(
      response(RoleDefinitionType.ROLE_DEFINITION_TYPE_BUILT_IN),
    );
    renderPage();

    expect(await screen.findByText('Built-in role')).toBeInTheDocument();
    expect(screen.getByLabelText('Name')).toBeDisabled();
    expect(screen.getByText('System')).toBeInTheDocument();
    expect(screen.getByText('Unknown')).toBeInTheDocument();
    expect(
      screen.queryByRole('button', { name: 'Archive role' }),
    ).not.toBeInTheDocument();
  });

  it('updates metadata and refetches the role after success', async () => {
    const user = userEvent.setup();
    renderPage();
    const name = await screen.findByLabelText('Name');
    await user.clear(name);
    await user.type(name, 'Support lead');
    await user.click(screen.getByRole('button', { name: 'Save details' }));

    await waitFor(() =>
      expect(mockUpdateAccessRole).toHaveBeenCalledWith(
        expect.objectContaining({
          name: 'Support lead',
          roleDefinitionId: '101',
        }),
      ),
    );
    await waitFor(() => expect(mockGetAccessRole).toHaveBeenCalledTimes(2));
  });

  it('renders stored audit actors and dates', async () => {
    mockGetAccessRole.mockResolvedValueOnce({
      getSuccess: () => true,
      getRole: () => ({
        ...role(RoleDefinitionType.ROLE_DEFINITION_TYPE_CUSTOM),
        getCreateddate: () => ({ getSeconds: () => 1, getNanos: () => 0 }),
        getUpdateddate: () => ({ getSeconds: () => 2, getNanos: () => 0 }),
        getCreatedactor: () => ({
          getDisplayname: () => 'Ada Lovelace',
          getType: () => 'user',
          getId: () => '1',
        }),
        getUpdatedactor: () => ({
          getDisplayname: () => 'Grace Hopper',
          getType: () => 'user',
          getId: () => '2',
        }),
      }),
    });
    renderPage();

    expect(await screen.findByText('Ada Lovelace')).toBeInTheDocument();
    expect(screen.getByText(/by Grace Hopper/)).toBeInTheDocument();
    expect(screen.getAllByText(/1970/)).toHaveLength(2);
  });

  it('keeps permission removal usable after a metadata failure', async () => {
    const user = userEvent.setup();
    mockUpdateAccessRole.mockResolvedValueOnce({
      getSuccess: () => false,
      getError: () => ({ getHumanmessage: () => 'Metadata rejected' }),
    });
    renderPage();
    await screen.findByRole('heading', { level: 1, name: 'Support manager' });
    await user.click(screen.getByRole('button', { name: 'Save details' }));
    expect(await screen.findByText('Metadata rejected')).toBeInTheDocument();

    await user.click(
      screen.getByRole('checkbox', { name: 'Remove Project Read' }),
    );
    await user.type(
      screen.getByLabelText('Reason for removal'),
      'No longer needed',
    );
    await user.click(screen.getByRole('button', { name: 'Remove selected' }));
    await waitFor(() =>
      expect(mockReduceAccessRolePermissions).toHaveBeenCalledTimes(1),
    );
  });

  it('requires reasons, removes selected permissions, and archives independently', async () => {
    const user = userEvent.setup();
    renderPage();
    await screen.findByRole('heading', { level: 1, name: 'Support manager' });

    await user.click(screen.getByRole('button', { name: 'Remove selected' }));
    expect(
      await screen.findByText('Select at least one permission to remove.'),
    ).toBeInTheDocument();
    await user.click(
      screen.getByRole('checkbox', { name: 'Remove Project Read' }),
    );
    await user.type(
      screen.getByLabelText('Reason for removal'),
      'No longer needed',
    );
    await user.click(screen.getByRole('button', { name: 'Remove selected' }));
    await waitFor(() =>
      expect(mockReduceAccessRolePermissions).toHaveBeenCalledWith(
        expect.objectContaining({
          permissionDefinitionIds: ['34'],
          reason: 'No longer needed',
        }),
      ),
    );

    await user.click(screen.getByRole('button', { name: 'Archive role' }));
    expect(
      await screen.findByText('A reason is required.'),
    ).toBeInTheDocument();
    await user.type(screen.getByLabelText('Reason for archive'), 'Replaced');
    await user.click(screen.getByRole('button', { name: 'Archive role' }));
    expect(await screen.findByText('Role catalog')).toBeInTheDocument();
  });

  it('does not redirect when archive finishes after leaving the page', async () => {
    const user = userEvent.setup();
    let resolveArchive: (value: unknown) => void = () => undefined;
    mockArchiveAccessRole.mockImplementationOnce(
      () => new Promise(resolve => (resolveArchive = resolve)),
    );
    renderPage();
    await screen.findByRole('heading', { level: 1, name: 'Support manager' });
    await user.type(screen.getByLabelText('Reason for archive'), 'Replaced');
    await user.click(screen.getByRole('button', { name: 'Archive role' }));
    await user.click(screen.getByRole('link', { name: 'Leave page' }));
    expect(
      await screen.findByText('Outside role management'),
    ).toBeInTheDocument();

    resolveArchive({ getSuccess: () => true });
    await waitFor(() =>
      expect(screen.getByText('Outside role management')).toBeInTheDocument(),
    );
    expect(screen.queryByText('Role catalog')).not.toBeInTheDocument();
  });
});
