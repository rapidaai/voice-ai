import React from 'react';
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ProjectCredentialPage } from './index';
import { AuthContext } from '@/context/auth-context';
import { listProjectCredentials, revokeProjectCredential } from '@/clients';
import toast from 'react-hot-toast/headless';

jest.mock('@/clients', () => ({
  listProjectCredentials: jest.fn(),
  revokeProjectCredential: jest.fn(),
}));

jest.mock('@/hooks/use-credential', () => ({
  useCredential: () => ['user-1', 'token-1', 'project-1', 'organization-1'],
  useCurrentCredential: jest.fn(),
}));

jest.mock(
  '@/app/components/domain/project-credentials/create-project-token',
  () => ({
    CreateProjectToken: ({ open, projectId, onCreated }: any) =>
      open ? (
        <div>
          <span>Role selector for {projectId}</span>
          <button type="button" onClick={onCreated}>
            Complete selected-role creation
          </button>
        </div>
      ) : null,
  }),
);

jest.mock('@/app/components/app-shell/helmet', () => ({ Helmet: () => null }));
jest.mock(
  '@/app/components/layout/container/message/notice-block/doc-notice-block',
  () => ({ DocNoticeBlock: ({ children }: any) => <div>{children}</div> }),
);
jest.mock('@/utils/date', () => ({
  toHumanReadableDate: () => 'Nov 9, 2026',
  toHumanReadableDateTime: () => 'Nov 9, 2026',
}));
jest.mock('react-hot-toast/headless', () => ({
  error: jest.fn(),
  success: jest.fn(),
}));

const credential = (id = '31', name = 'Web widget') => ({
  getId: () => id,
  getName: () => name,
  getKey: () => 'rpx_secret-value',
  getRolename: () => 'Runtime',
  getRoledefinitionid: () => '117',
  getCreateddate: () => ({ getSeconds: () => 1, getNanos: () => 0 }),
});

const response = (
  credentials = [credential()],
  total = credentials.length,
) => ({
  getSuccess: () => true,
  getDataList: () => credentials,
  getPaginated: () => ({ getTotalitem: () => total }),
});

function renderPage() {
  const currentProject = { id: 'project-1' };
  return render(
    <AuthContext.Provider
      value={
        {
          getCurrentProject: () => currentProject,
        } as any
      }
    >
      <ProjectCredentialPage />
    </AuthContext.Provider>,
  );
}

describe('ProjectCredentialPage', () => {
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
    (listProjectCredentials as jest.Mock).mockImplementation(({ callback }) =>
      callback(null, response()),
    );
    (revokeProjectCredential as jest.Mock).mockImplementation(({ callback }) =>
      callback(null, { getSuccess: () => true, getId: () => '31' }),
    );
  });

  it('renders a paginated enterprise inventory with role metadata', async () => {
    renderPage();

    expect(await screen.findByText('Web widget')).toBeInTheDocument();
    expect(
      screen.getByRole('heading', { level: 1, name: 'Project API keys' }),
    ).toBeInTheDocument();
    expect(
      screen.getAllByRole('columnheader').map(header => header.textContent),
    ).toEqual(['Name', 'API key', 'Access role', 'Created', 'Actions']);
    expect(screen.getByText('Runtime')).toBeInTheDocument();
    expect(screen.getByText('1 of 1')).toBeInTheDocument();
    expect(
      screen.queryByText(/Use a separate API key for each application/),
    ).not.toBeInTheDocument();
    expect(listProjectCredentials).toHaveBeenLastCalledWith(
      expect.objectContaining({
        auth: {
          organizationId: 'organization-1',
          token: 'token-1',
          userId: 'user-1',
        },
        projectId: 'project-1',
        page: 1,
        pageSize: 25,
        criteria: [],
      }),
    );
  });

  it('opens the shared role selector and refreshes after creation', async () => {
    const user = userEvent.setup();
    renderPage();

    await screen.findByText('Web widget');
    await user.click(screen.getByRole('button', { name: 'Create API key' }));
    expect(screen.getByText('Role selector for project-1')).toBeInTheDocument();
    await user.click(
      screen.getByRole('button', { name: 'Complete selected-role creation' }),
    );
    await waitFor(() =>
      expect(listProjectCredentials).toHaveBeenCalledTimes(2),
    );
    expect(screen.getByText('Role selector for project-1')).toBeInTheDocument();
  });

  it('separates loading failure from an empty project', async () => {
    (listProjectCredentials as jest.Mock).mockImplementation(({ callback }) =>
      callback({ message: 'unavailable' }, null),
    );
    renderPage();

    expect(
      await screen.findByText('API keys could not be loaded'),
    ).toBeInTheDocument();
    expect(
      screen.getByText('Try again. No API keys were changed.'),
    ).toBeInTheDocument();
    expect(toast.error).toHaveBeenCalledWith(
      'API keys could not be refreshed. Try again.',
    );
    expect(screen.queryByText('No project API keys')).not.toBeInTheDocument();
  });

  it('reports a refresh failure globally while preserving the inventory', async () => {
    (listProjectCredentials as jest.Mock)
      .mockImplementationOnce(({ callback }) => callback(null, response()))
      .mockImplementationOnce(({ callback }) =>
        callback({ message: 'unavailable' }, null),
      );
    const user = userEvent.setup();
    renderPage();

    expect(await screen.findByText('Web widget')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Refresh API keys' }));

    await waitFor(() =>
      expect(toast.error).toHaveBeenCalledWith(
        'API keys could not be refreshed. Try again.',
      ),
    );
    expect(screen.getByText('Web widget')).toBeInTheDocument();
    expect(
      screen.queryByText('API keys could not be refreshed'),
    ).not.toBeInTheDocument();
  });

  it('renders the initial empty and search empty states separately', async () => {
    (listProjectCredentials as jest.Mock).mockImplementation(({ callback }) =>
      callback(null, response([], 0)),
    );
    const { unmount } = renderPage();

    expect(await screen.findByText('No project API keys')).toBeInTheDocument();
    expect(
      screen.getByText(
        'Create an API key for an application or tool that needs access to this project.',
      ),
    ).toBeInTheDocument();
    unmount();

    (listProjectCredentials as jest.Mock).mockImplementation(
      ({ criteria, callback }) =>
        callback(null, criteria.length > 0 ? response([], 0) : response()),
    );
    const user = userEvent.setup();
    renderPage();
    const search = await screen.findByPlaceholderText(
      'Search for name or role',
    );
    await user.click(search);
    await user.click(screen.getByRole('button', { name: 'name contains' }));
    await user.type(screen.getByRole('textbox'), 'missing{Enter}');

    expect(
      await screen.findByText('No API keys match your search.'),
    ).toBeInTheDocument();
    expect(listProjectCredentials).toHaveBeenLastCalledWith(
      expect.objectContaining({
        page: 1,
        criteria: [{ key: 'name', logic: 'contains', value: 'missing' }],
      }),
    );
  });

  it('reveals and copies only from the fixed-width API key column', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const writeText = jest
      .spyOn(navigator.clipboard, 'writeText')
      .mockResolvedValue(undefined);
    renderPage();

    const name = await screen.findByText('Web widget');
    const row = name.closest('tr');
    expect(row).not.toBeNull();
    expect(
      screen.queryByRole('button', { name: 'Web widget' }),
    ).not.toBeInTheDocument();
    expect(screen.queryByText('rpx_secret-value')).not.toBeInTheDocument();
    const showKey = screen.getByRole('button', {
      name: 'Show Web widget API key',
    });
    expect(showKey).toHaveAttribute('aria-expanded', 'false');
    await user.click(showKey);
    expect(showKey).toHaveAttribute('aria-expanded', 'true');
    expect(screen.getByText('rpx_secret-value')).toBeInTheDocument();
    await user.click(
      screen.getByRole('button', { name: 'Copy Web widget API key' }),
    );

    expect(writeText).toHaveBeenCalledWith('rpx_secret-value');
    expect(toast.success).toHaveBeenCalledWith('API key copied.');
    const hideKey = screen.getByRole('button', {
      name: 'Hide Web widget API key',
    });
    await user.click(hideKey);
    expect(hideKey).toHaveAttribute('aria-expanded', 'false');
    expect(screen.queryByText('rpx_secret-value')).not.toBeInTheDocument();

    const cells = within(row!).getAllByRole('cell');
    expect(
      within(cells[4]).queryByRole('button', {
        name: /^(show|hide|copy) web widget api key$/i,
      }),
    ).not.toBeInTheDocument();
    expect(
      within(cells[1]).getByRole('button', {
        name: 'Show Web widget API key',
      }),
    ).toBeInTheDocument();
    expect(
      within(cells[4]).getByRole('button', { name: 'Revoke Web widget' }),
    ).toBeInTheDocument();
  });

  it('requests the selected page size', async () => {
    const user = userEvent.setup();
    renderPage();

    await screen.findByText('Web widget');
    await user.selectOptions(
      screen.getByRole('combobox', { name: 'Items per page:' }),
      '50',
    );

    await waitFor(() =>
      expect(listProjectCredentials).toHaveBeenLastCalledWith(
        expect.objectContaining({ page: 1, pageSize: 50 }),
      ),
    );
  });

  it('shows revoke without a console permission projection and confirms by name', async () => {
    const user = userEvent.setup();
    renderPage();
    await user.click(
      await screen.findByRole('button', { name: 'Revoke Web widget' }),
    );
    expect(
      screen.getByText(
        'Applications using "Web widget" will lose access immediately.',
      ),
    ).toBeInTheDocument();
    await user.type(
      screen.getByLabelText('Type "Web widget" to confirm'),
      'Web widget',
    );
    await user.click(screen.getByRole('button', { name: 'Revoke API key' }));

    expect(revokeProjectCredential).toHaveBeenCalledWith(
      expect.objectContaining({
        auth: {
          organizationId: 'organization-1',
          token: 'token-1',
          userId: 'user-1',
        },
        projectId: 'project-1',
        credentialId: '31',
      }),
    );
    await waitFor(() =>
      expect(listProjectCredentials).toHaveBeenCalledTimes(2),
    );
  });

  it('retains the row when a revoke transport failure cannot be confirmed', async () => {
    (revokeProjectCredential as jest.Mock).mockImplementation(({ callback }) =>
      callback({ message: 'connection lost' }, null),
    );
    const user = userEvent.setup();
    renderPage();

    await user.click(
      await screen.findByRole('button', { name: 'Revoke Web widget' }),
    );
    await user.type(
      screen.getByLabelText('Type "Web widget" to confirm'),
      'Web widget',
    );
    await user.click(screen.getByRole('button', { name: 'Revoke API key' }));

    await waitFor(() =>
      expect(toast.error).toHaveBeenCalledWith(
        'API key revocation could not be confirmed. Try again.',
      ),
    );
    expect(screen.getByText('Web widget')).toBeInTheDocument();
  });
});
