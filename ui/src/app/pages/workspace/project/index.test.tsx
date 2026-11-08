import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { AuthContext } from '@/context/auth-context';
import { ProjectPage } from './index';

const mockListWorkspaceProjects = jest.fn();
const mockDeleteWorkspaceProject = jest.fn();

const project = {
  getId: () => 'project-1',
  getName: () => 'Customer support',
  getCreateddate: () => undefined,
  getMembersList: () => [{ getName: () => 'Ada Lovelace' }],
  getStatus: () => 'ACTIVE',
  toObject: () => ({ id: 'project-1', name: 'Customer support' }),
};

const projectResponse = {
  getSuccess: () => true,
  getDataList: () => [project],
  getPaginated: () => ({ getTotalitem: () => 1 }),
};

jest.mock('@/hooks/use-credential', () => ({
  useCredential: () => ['user-1', 'token-1'],
}));

jest.mock('@/app/components/app-shell/helmet', () => ({
  Helmet: () => null,
}));

jest.mock('@/clients', () => ({
  listWorkspaceProjects: (...args: unknown[]) =>
    mockListWorkspaceProjects(...args),
  deleteWorkspaceProject: (...args: unknown[]) =>
    mockDeleteWorkspaceProject(...args),
}));

jest.mock('@/app/components/dialogs/workspace', () => ({
  CreateProjectDialog: () => null,
  UpdateProjectDialog: ({ modalOpen }: { modalOpen: boolean }) =>
    modalOpen ? <div role="dialog">Edit project</div> : null,
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

jest.mock('@/app/components/domain/avatar/project-user-group-avatar', () => ({
  ProjectUserGroupAvatar: () => <span>Ada Lovelace</span>,
}));

jest.mock('@/app/components/domain/indicators/role', () => ({
  RoleIndicator: ({ role }: { role?: string }) => <span>{role}</span>,
}));

jest.mock('@/app/components/ui/feedback', () => ({
  CarbonIconIndicator: ({ state }: { state?: string }) => <span>{state}</span>,
  EmptyState: ({ title }: { title: string }) => <div>{title}</div>,
}));

function renderPage() {
  return render(
    <AuthContext.Provider
      value={{
        projectRoles: [
          {
            id: 'project-role-1',
            projectid: 'project-1',
            projectname: 'Customer support',
            role: 'admin',
          },
        ],
      }}
    >
      <ProjectPage />
    </AuthContext.Provider>,
  );
}

describe('ProjectPage', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockListWorkspaceProjects.mockImplementation(({ callback }) =>
      callback(null, projectResponse),
    );
  });

  it('uses Carbon radio selection and batch actions for project actions', async () => {
    const user = userEvent.setup();
    const { container } = renderPage();

    await screen.findByText('Customer support');
    const selection = screen.getByRole('radio', { name: 'Select row' });
    expect(selection.closest('td')).toHaveClass('cds--table-column-radio');
    expect(container.querySelector('.cds--batch-actions')).toBeNull();

    await user.click(selection);

    const batchActions = container.querySelector('.cds--batch-actions');
    expect(batchActions).toHaveAttribute('aria-hidden', 'false');
    expect(batchActions).toHaveTextContent('1 item selected');

    await user.click(
      within(batchActions as HTMLElement).getByRole('button', { name: 'Edit' }),
    );
    expect(screen.getByRole('dialog')).toHaveTextContent('Edit project');
  });

  it('asks for confirmation before deleting the selected project', async () => {
    const user = userEvent.setup();
    const { container } = renderPage();

    await user.click(await screen.findByRole('radio', { name: 'Select row' }));
    await user.click(
      within(
        container.querySelector('.cds--batch-actions') as HTMLElement,
      ).getByRole('button', { name: 'Delete' }),
    );

    expect(mockDeleteWorkspaceProject).not.toHaveBeenCalled();
    expect(screen.getByRole('dialog')).toHaveTextContent('Delete project');
    expect(screen.getByRole('dialog')).toHaveTextContent('Customer support');
  });

  it('keeps an in-place Carbon table skeleton while projects load', async () => {
    mockListWorkspaceProjects.mockImplementation(() => undefined);

    const { container } = renderPage();

    await waitFor(() => expect(mockListWorkspaceProjects).toHaveBeenCalled());
    expect(
      screen.getByRole('heading', { level: 1, name: 'Projects' }),
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
    const { container } = renderPage();

    await screen.findByText('Customer support');
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
