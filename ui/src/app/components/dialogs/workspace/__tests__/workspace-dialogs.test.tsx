import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import { AuthContext } from '@/context/auth-context';
import {
  AddUserToProjects,
  CreateProject,
  InviteUserToOrganization,
  UpdateProject,
} from '@rapidaai/react';
import {
  CreateProjectDialog,
  InviteOrganizationUserDialog,
  InviteProjectUserDialog,
  UpdateProjectDialog,
} from '..';

const mockToast = {
  error: jest.fn(),
  success: jest.fn(),
};

const mockRapidaStore = {
  loading: false,
  showLoader: jest.fn(),
  hideLoader: jest.fn(),
};

const mockGetAllUser = jest.fn();
const mockAvailableUser = {
  getId: () => 'user-2',
  getName: () => 'Priyanka Iyer',
  getEmail: () => 'p_iyer@outlook.com',
};
let mockUsers = [mockAvailableUser];

jest.mock('@rapidaai/react', () => ({
  ...(() => {
    class ProjectRoleAssignment {
      projectId = '';
      projectRole = '';

      setProjectid(projectId: string) {
        this.projectId = projectId;
      }

      setProjectrole(projectRole: string) {
        this.projectRole = projectRole;
      }
    }

    class InviteUserToOrganizationRequest {
      email = '';
      organizationRole = '';
      projectRoles: ProjectRoleAssignment[] = [];

      setEmail(email: string) {
        this.email = email;
      }

      setOrganizationrole(organizationRole: string) {
        this.organizationRole = organizationRole;
      }

      setProjectrolesList(projectRoles: ProjectRoleAssignment[]) {
        this.projectRoles = projectRoles;
      }
    }

    class AddUserToProjectsRequest {
      userId = '';
      projectRoles: ProjectRoleAssignment[] = [];

      setUserid(userId: string) {
        this.userId = userId;
      }

      setProjectrolesList(projectRoles: ProjectRoleAssignment[]) {
        this.projectRoles = projectRoles;
      }
    }

    return {
      AddUserToProjects: jest.fn(),
      AddUserToProjectsRequest,
      CreateProject: jest.fn(),
      InviteUserToOrganization: jest.fn(),
      InviteUserToOrganizationRequest,
      ProjectRoleAssignment,
      UpdateProject: jest.fn(),
    };
  })(),
}));

jest.mock('react-hot-toast/headless', () => ({
  error: (...args: unknown[]) => mockToast.error(...args),
  success: (...args: unknown[]) => mockToast.success(...args),
}));

jest.mock('@/configs', () => ({
  connectionConfig: { endpoint: 'test-endpoint' },
}));

jest.mock('@/hooks/use-credential', () => ({
  useCredential: () => ['auth-1', 'token-1', 'project-1', 'org-1'],
  useCurrentCredential: () => ({
    authId: 'auth-1',
    projectId: 'project-1',
    token: 'token-1',
  }),
}));

jest.mock('@/stores/app', () => ({
  useRapidaStore: () => mockRapidaStore,
}));

jest.mock('@/stores/user', () => ({
  useUserPageStore: () => ({
    users: mockUsers,
    getAllUser: mockGetAllUser,
  }),
}));

jest.mock('@/app/components/ui/feedback', () => ({
  ErrorMessage: ({ message }: { message?: string }) =>
    message ? <div role="alert">{message}</div> : null,
}));

jest.mock('@/app/components/ui/primitives', () => {
  const React = jest.requireActual('react');
  const { typographyClassNames } = jest.requireActual(
    '@/app/components/ui/primitives/text',
  );

  return {
    Form: ({ children, onSubmit }: any) => (
      <form onSubmit={onSubmit}>{children}</form>
    ),
    Modal: ({ open, children, onClose }: any) =>
      open ? (
        <section>
          <button type="button" onClick={onClose}>
            Modal close
          </button>
          {children}
        </section>
      ) : null,
    ModalBody: ({ children }: any) => <main>{children}</main>,
    ModalFooter: ({ children }: any) => <footer>{children}</footer>,
    ModalHeader: ({
      label,
      labelClassName,
      title,
      titleClassName,
      onClose,
    }: any) => (
      <header>
        {label ? <p className={labelClassName}>{label}</p> : null}
        <h2 className={titleClassName}>{title}</h2>
        <button type="button" onClick={onClose}>
          Header close
        </button>
      </header>
    ),
    PrimaryButton: ({ children, isLoading, ...props }: any) => (
      <button data-loading={String(Boolean(isLoading))} {...props}>
        {children}
      </button>
    ),
    SecondaryButton: ({ children, ...props }: any) => (
      <button {...props}>{children}</button>
    ),
    Stack: ({ children }: any) => <div>{children}</div>,
    typographyClassNames,
    TextArea: React.forwardRef(
      (
        { id, labelText, ...props }: any,
        ref: React.ForwardedRef<HTMLTextAreaElement>,
      ) => (
        <label htmlFor={id}>
          {labelText}
          <textarea id={id} ref={ref} {...props} />
        </label>
      ),
    ),
    TextInput: React.forwardRef(
      (
        { id, labelText, ...props }: any,
        ref: React.ForwardedRef<HTMLInputElement>,
      ) => (
        <label htmlFor={id}>
          {labelText}
          <input id={id} ref={ref} {...props} />
        </label>
      ),
    ),
  };
});

jest.mock('@carbon/react', () => ({
  ComboBox: ({
    itemToString,
    items,
    onChange,
    placeholder,
    selectedItem,
    titleText,
  }: any) => {
    itemToString?.(null);
    return (
      <button
        type="button"
        data-label={titleText}
        onClick={() => onChange({ selectedItem: selectedItem || items[0] })}
      >
        {placeholder}
      </button>
    );
  },
  Dropdown: ({ itemToString, items, label, onChange, titleText }: any) => {
    itemToString?.(null);
    return (
      <button
        type="button"
        data-label={titleText}
        onClick={() => onChange({ selectedItem: items[0] })}
      >
        {label}
      </button>
    );
  },
  DefinitionTooltip: ({ children, definition }: any) => (
    <span data-definition={definition}>{children}</span>
  ),
}));

jest.mock('@/app/components/domain/project-role-table', () => ({
  ProjectRoleTable: ({
    addButtonLabel,
    defaultProjectId,
    onChange,
    projectOptions,
    rows,
    roleOptions,
    title,
  }: any) => (
    <section aria-label={title}>
      <p>
        {title} ({rows.length})
      </p>
      <button
        type="button"
        onClick={() =>
          onChange([
            ...rows,
            {
              projectId: defaultProjectId || projectOptions[0]?.value || '',
              projectRole: '',
            },
          ])
        }
      >
        {addButtonLabel}
      </button>
      <button
        type="button"
        onClick={() =>
          onChange([
            {
              projectId: defaultProjectId || projectOptions[0]?.value || '',
              projectRole: roleOptions[0]?.value || '',
            },
          ])
        }
      >
        Set valid project role
      </button>
      <button
        type="button"
        onClick={() =>
          onChange([
            { projectId: 'project-1', projectRole: 'admin' },
            { projectId: 'project-1', projectRole: 'reader' },
          ])
        }
      >
        Set duplicate project roles
      </button>
      <button
        type="button"
        onClick={() => onChange([{ projectId: '', projectRole: '' }])}
      >
        Set incomplete project role
      </button>
    </section>
  ),
}));

const mockCreateProject = CreateProject as jest.Mock;
const mockUpdateProject = UpdateProject as jest.Mock;
const mockInviteUserToOrganization = InviteUserToOrganization as jest.Mock;
const mockAddUserToProjects = AddUserToProjects as jest.Mock;

const successResponse = () => ({
  getError: () => null,
  getSuccess: () => true,
});

const errorResponse = (message: string) => ({
  getError: () => ({
    getHumanmessage: () => message,
  }),
  getSuccess: () => false,
});

const mockAuthorize = jest.fn<
  void,
  [onSuccess: () => void, onFailure?: () => void]
>((onSuccess: () => void) => onSuccess());

const authContextValue = {
  authorize: mockAuthorize,
  projectRoles: [
    { projectid: 'project-1', projectname: 'Support operations' },
    { projectid: 'project-2', projectname: 'Sales enablement' },
  ],
};

function renderWithAuth(ui: React.ReactElement) {
  return render(
    <AuthContext.Provider value={authContextValue as never}>
      {ui}
    </AuthContext.Provider>,
  );
}

async function submitFormByButton(name: string) {
  const user = userEvent.setup();
  await user.click(screen.getByRole('button', { name }));
}

function renderOrganizationInviteDialog() {
  renderWithAuth(
    <InviteOrganizationUserDialog
      modalOpen
      setModalOpen={jest.fn()}
      onSuccess={jest.fn()}
    />,
  );
}

async function enterOrganizationInviteEmail(
  user: ReturnType<typeof userEvent.setup>,
) {
  await user.click(screen.getByLabelText('Email address'));
  await user.paste(String('member@example.com'));
}

describe('workspace dialogs', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockAuthorize.mockImplementation((onSuccess: () => void) => onSuccess());
    mockUsers = [mockAvailableUser];
    mockRapidaStore.loading = false;
  });

  it('creates a project and closes after the auth context refresh succeeds', async () => {
    const user = userEvent.setup();
    const setModalOpen = jest.fn();
    const afterCreateProject = jest.fn();
    mockCreateProject.mockImplementation(
      (_config, _name, _description, _metadata, callback) =>
        callback(null, successResponse()),
    );

    renderWithAuth(
      <CreateProjectDialog
        modalOpen
        setModalOpen={setModalOpen}
        afterCreateProject={afterCreateProject}
      />,
    );

    await user.click(screen.getByLabelText('Project Name'));
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('Support operations'));
    await user.click(screen.getByLabelText('Project Description'));
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('Support workspace'));
    await submitFormByButton('Create Project');

    await waitFor(() => {
      expect(mockCreateProject).toHaveBeenCalledWith(
        { endpoint: 'test-endpoint' },
        'Support operations',
        'Support workspace',
        {
          authorization: 'token-1',
          'x-auth-id': 'auth-1',
        },
        expect.any(Function),
      );
    });
    expect(mockRapidaStore.showLoader).toHaveBeenCalledTimes(1);
    expect(authContextValue.authorize).toHaveBeenCalledTimes(1);
    await waitFor(() => expect(setModalOpen).toHaveBeenCalledWith(false));
    expect(mockRapidaStore.hideLoader).toHaveBeenCalledTimes(1);
    expect(mockToast.success).toHaveBeenCalledWith(
      'The project has been created successfully.',
    );
    expect(afterCreateProject).toHaveBeenCalledTimes(1);
  });

  it('shows create project service errors without closing', async () => {
    const user = userEvent.setup();
    const setModalOpen = jest.fn();
    mockCreateProject.mockImplementation(
      (_config, _name, _description, _metadata, callback) =>
        callback(null, errorResponse('Project already exists')),
    );

    renderWithAuth(
      <CreateProjectDialog
        modalOpen
        setModalOpen={setModalOpen}
        afterCreateProject={jest.fn()}
      />,
    );

    await user.click(screen.getByLabelText('Project Name'));
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('Support operations'));
    await user.click(screen.getByLabelText('Project Description'));
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('Support workspace'));
    await submitFormByButton('Create Project');

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Project already exists',
    );
    expect(mockToast.error).toHaveBeenCalledWith('Project already exists');
    expect(setModalOpen).not.toHaveBeenCalledWith(false);
  });

  it('shows create project transport and generic response errors', async () => {
    const user = userEvent.setup();
    mockCreateProject.mockImplementationOnce(
      (_config, _name, _description, _metadata, callback) =>
        callback({ message: 'network failed' }, null),
    );

    const { rerender } = renderWithAuth(
      <CreateProjectDialog
        modalOpen
        setModalOpen={jest.fn()}
        afterCreateProject={jest.fn()}
      />,
    );

    await user.click(screen.getByLabelText('Project Name'));
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('Support operations'));
    await user.click(screen.getByLabelText('Project Description'));
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('Support workspace'));
    await submitFormByButton('Create Project');

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Unable to process your request. please try again later.',
    );

    mockCreateProject.mockImplementationOnce(
      (_config, _name, _description, _metadata, callback) =>
        callback(null, { getSuccess: () => false, getError: () => null }),
    );

    rerender(
      <AuthContext.Provider value={authContextValue as never}>
        <CreateProjectDialog
          modalOpen
          setModalOpen={jest.fn()}
          afterCreateProject={jest.fn()}
        />
      </AuthContext.Provider>,
    );
    await user.click(screen.getByLabelText('Project Name'));
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('Support operations'));
    await user.click(screen.getByLabelText('Project Description'));
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('Support workspace'));
    await submitFormByButton('Create Project');

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Unable to process your request. please try again later.',
    );
  });

  it('shows create project authorization refresh failures', async () => {
    const user = userEvent.setup();
    authContextValue.authorize.mockImplementationOnce(
      (_onSuccess: () => void, onFailure?: () => void) => onFailure?.(),
    );
    mockCreateProject.mockImplementation(
      (_config, _name, _description, _metadata, callback) =>
        callback(null, successResponse()),
    );

    renderWithAuth(
      <CreateProjectDialog
        modalOpen
        setModalOpen={jest.fn()}
        afterCreateProject={jest.fn()}
      />,
    );

    await user.click(screen.getByLabelText('Project Name'));
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('Support operations'));
    await user.click(screen.getByLabelText('Project Description'));
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('Support workspace'));
    await submitFormByButton('Create Project');

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Unable to process your request. please try again later.',
    );
    expect(mockRapidaStore.hideLoader).toHaveBeenCalledTimes(1);
  });

  it('hides the create project loader when auth refresh is unavailable', async () => {
    const user = userEvent.setup();
    mockCreateProject.mockImplementation(
      (_config, _name, _description, _metadata, callback) =>
        callback(null, successResponse()),
    );

    render(
      <AuthContext.Provider
        value={{ projectRoles: authContextValue.projectRoles } as never}
      >
        <CreateProjectDialog
          modalOpen
          setModalOpen={jest.fn()}
          afterCreateProject={jest.fn()}
        />
      </AuthContext.Provider>,
    );

    await user.click(screen.getByLabelText('Project Name'));
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('Support operations'));
    await user.click(screen.getByLabelText('Project Description'));
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('Support workspace'));
    await submitFormByButton('Create Project');

    await waitFor(() =>
      expect(mockRapidaStore.hideLoader).toHaveBeenCalledTimes(1),
    );
  });

  it('updates a project and hides the loader after success', async () => {
    const user = userEvent.setup();
    const setModalOpen = jest.fn();
    const afterUpdateProject = jest.fn();
    mockUpdateProject.mockImplementation((_config, _projectId, callback) =>
      callback(null, successResponse()),
    );

    renderWithAuth(
      <UpdateProjectDialog
        modalOpen
        setModalOpen={setModalOpen}
        existingProject={
          {
            id: 'project-1',
            name: 'Existing project',
            description: 'Existing description',
          } as any
        }
        afterUpdateProject={afterUpdateProject}
      />,
    );

    await user.click(screen.getByLabelText('Project Name'));
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('Updated project'));
    await user.click(screen.getByLabelText('Project Description'));
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('Updated description'));
    await submitFormByButton('Update');

    await waitFor(() => {
      expect(mockUpdateProject).toHaveBeenCalledWith(
        { endpoint: 'test-endpoint' },
        'project-1',
        expect.any(Function),
        {
          authorization: 'token-1',
          'x-auth-id': 'auth-1',
        },
        'Updated project',
        'Updated description',
      );
    });
    await waitFor(() => expect(setModalOpen).toHaveBeenCalledWith(false));
    expect(mockRapidaStore.hideLoader).toHaveBeenCalledTimes(1);
    expect(mockToast.success).toHaveBeenCalledWith(
      'Your project has been updated successfully.',
    );
    expect(afterUpdateProject).toHaveBeenCalledTimes(1);
  });

  it('shows update project service errors', async () => {
    mockUpdateProject.mockImplementation((_config, _projectId, callback) =>
      callback(null, errorResponse('Project update failed')),
    );

    renderWithAuth(
      <UpdateProjectDialog
        modalOpen
        setModalOpen={jest.fn()}
        existingProject={
          {
            id: 'project-1',
            name: 'Existing project',
            description: 'Existing description',
          } as any
        }
        afterUpdateProject={jest.fn()}
      />,
    );

    await submitFormByButton('Update');

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Project update failed',
    );
    expect(mockToast.error).toHaveBeenCalledWith('Project update failed');
  });

  it('shows update project transport, generic, and auth refresh errors', async () => {
    mockUpdateProject.mockImplementationOnce((_config, _projectId, callback) =>
      callback({ message: 'network failed' }, null),
    );

    const renderUpdate = () => (
      <AuthContext.Provider value={authContextValue as never}>
        <UpdateProjectDialog
          modalOpen
          setModalOpen={jest.fn()}
          existingProject={
            {
              id: 'project-1',
              name: 'Existing project',
              description: 'Existing description',
            } as any
          }
          afterUpdateProject={jest.fn()}
        />
      </AuthContext.Provider>
    );

    const { rerender } = render(renderUpdate());

    await submitFormByButton('Update');
    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Unable to process your request. please try again later.',
    );

    mockUpdateProject.mockImplementationOnce((_config, _projectId, callback) =>
      callback(null, { getSuccess: () => false, getError: () => null }),
    );
    rerender(renderUpdate());

    await submitFormByButton('Update');
    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Unable to process your request. please try again later.',
    );

    authContextValue.authorize.mockImplementationOnce(
      (_onSuccess: () => void, onFailure?: () => void) => onFailure?.(),
    );
    mockUpdateProject.mockImplementationOnce((_config, _projectId, callback) =>
      callback(null, successResponse()),
    );
    rerender(renderUpdate());

    await submitFormByButton('Update');
    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Unable to process your request. please try again later.',
    );
  });

  it('hides the update project loader when auth refresh is unavailable', async () => {
    mockUpdateProject.mockImplementation((_config, _projectId, callback) =>
      callback(null, successResponse()),
    );

    render(
      <AuthContext.Provider
        value={{ projectRoles: authContextValue.projectRoles } as never}
      >
        <UpdateProjectDialog
          modalOpen
          setModalOpen={jest.fn()}
          existingProject={
            {
              id: 'project-1',
              name: 'Existing project',
              description: 'Existing description',
            } as any
          }
          afterUpdateProject={jest.fn()}
        />
      </AuthContext.Provider>,
    );

    await submitFormByButton('Update');

    await waitFor(() =>
      expect(mockRapidaStore.hideLoader).toHaveBeenCalledTimes(1),
    );
  });

  it('describes optional project access and requires a valid invitation email', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    renderOrganizationInviteDialog();

    expect(
      screen.queryByText(
        'Send an invitation and choose the access this user should receive.',
      ),
    ).not.toBeInTheDocument();
    expect(
      screen.getByRole('heading', { name: 'Project access' }),
    ).toContainHTML(
      'data-definition="Optional. Assign project roles now, or invite the user with organization access only."',
    );

    await user.click(screen.getByRole('button', { name: 'Invite user' }));
    expect(screen.getByRole('alert')).toHaveTextContent(
      'Please provide a valid email to invite user.',
    );
    expect(mockInviteUserToOrganization).not.toHaveBeenCalled();
  });

  it('requires an organization role before sending an invitation', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    renderOrganizationInviteDialog();

    await enterOrganizationInviteEmail(user);
    await user.click(screen.getByRole('button', { name: 'Invite user' }));
    expect(screen.getByRole('alert')).toHaveTextContent(
      'Please select an organization role.',
    );
    expect(mockInviteUserToOrganization).not.toHaveBeenCalled();
  });

  it('rejects incomplete project roles on organization invitations', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    renderOrganizationInviteDialog();

    await enterOrganizationInviteEmail(user);
    await user.click(
      screen.getByRole('button', { name: 'Select organization role' }),
    );
    await user.click(
      screen.getByRole('button', { name: 'Set incomplete project role' }),
    );
    await user.click(screen.getByRole('button', { name: 'Invite user' }));
    expect(screen.getByRole('alert')).toHaveTextContent(
      'Select both project and role for each project role row.',
    );
    expect(mockInviteUserToOrganization).not.toHaveBeenCalled();
  });

  it('rejects duplicate project roles on organization invitations', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    renderOrganizationInviteDialog();

    await enterOrganizationInviteEmail(user);
    await user.click(
      screen.getByRole('button', { name: 'Select organization role' }),
    );
    await user.click(
      screen.getByRole('button', { name: 'Set duplicate project roles' }),
    );
    await user.click(screen.getByRole('button', { name: 'Invite user' }));
    expect(screen.getByRole('alert')).toHaveTextContent(
      'A project can only be assigned once.',
    );
    expect(mockInviteUserToOrganization).not.toHaveBeenCalled();
  });

  it('sends organization invitations with optional project roles', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const setModalOpen = jest.fn();
    const onSuccess = jest.fn();
    mockInviteUserToOrganization.mockResolvedValue(successResponse());

    renderWithAuth(
      <InviteOrganizationUserDialog
        modalOpen
        setModalOpen={setModalOpen}
        onSuccess={onSuccess}
      />,
    );

    await user.click(screen.getByLabelText('Email address'));
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('member@example.com'));
    await user.click(
      screen.getByRole('button', { name: 'Select organization role' }),
    );
    await user.click(
      screen.getByRole('button', { name: 'Set valid project role' }),
    );
    await user.click(screen.getByRole('button', { name: 'Invite user' }));

    await waitFor(() => expect(setModalOpen).toHaveBeenCalledWith(false));
    const request = mockInviteUserToOrganization.mock.calls[0][1];
    expect(request.email).toBe('member@example.com');
    expect(request.organizationRole).toBe('admin');
    expect(request.projectRoles).toEqual([
      { projectId: 'project-1', projectRole: 'super admin' },
    ]);
    expect(mockInviteUserToOrganization).toHaveBeenCalledWith(
      { endpoint: 'test-endpoint' },
      request,
      {
        authorization: 'token-1',
        'x-auth-id': 'auth-1',
      },
    );
    expect(mockRapidaStore.hideLoader).toHaveBeenCalledTimes(1);
    expect(mockToast.success).toHaveBeenCalledWith(
      'The organization invitation was sent successfully.',
    );
    expect(onSuccess).toHaveBeenCalledTimes(1);
  });

  it('shows organization invitation API errors', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    mockInviteUserToOrganization.mockResolvedValue(
      errorResponse('Invitation failed'),
    );

    renderWithAuth(
      <InviteOrganizationUserDialog
        modalOpen
        setModalOpen={jest.fn()}
        onSuccess={jest.fn()}
      />,
    );

    await user.click(screen.getByLabelText('Email address'));
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('member@example.com'));
    await user.click(
      screen.getByRole('button', { name: 'Select organization role' }),
    );
    await user.click(screen.getByRole('button', { name: 'Invite user' }));

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Invitation failed',
    );
    expect(mockToast.error).toHaveBeenCalledWith('Invitation failed');
  });

  it('shows organization invitation thrown errors', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    mockInviteUserToOrganization.mockRejectedValue({
      getHumanmessage: () => 'Transport failed',
    });

    renderWithAuth(
      <InviteOrganizationUserDialog
        modalOpen
        setModalOpen={jest.fn()}
        onSuccess={jest.fn()}
      />,
    );

    await user.click(screen.getByLabelText('Email address'));
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('member@example.com'));
    await user.click(
      screen.getByRole('button', { name: 'Select organization role' }),
    );
    await user.click(screen.getByRole('button', { name: 'Invite user' }));

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Transport failed',
    );
    expect(mockToast.error).toHaveBeenCalledWith('Transport failed');
  });

  it('loads users when a project invite opens without a fixed user', () => {
    mockUsers = [];

    renderWithAuth(
      <InviteProjectUserDialog
        modalOpen
        setModalOpen={jest.fn()}
        onSuccess={jest.fn()}
      />,
    );

    expect(mockGetAllUser).toHaveBeenCalledWith(
      'token-1',
      'auth-1',
      'project-1',
      expect.any(Function),
      expect.any(Function),
    );
  });

  it('shows user loading errors when project invite opens', () => {
    mockUsers = [];
    mockGetAllUser.mockImplementationOnce(
      (_token, _authId, _projectId, onError) => onError('Unable to load users'),
    );

    renderWithAuth(
      <InviteProjectUserDialog
        modalOpen
        setModalOpen={jest.fn()}
        onSuccess={jest.fn()}
      />,
    );

    expect(screen.getByRole('alert')).toHaveTextContent('Unable to load users');
  });

  it('validates project invitations before calling the API', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    renderWithAuth(
      <InviteProjectUserDialog
        modalOpen
        setModalOpen={jest.fn()}
        onSuccess={jest.fn()}
      />,
    );

    await user.click(screen.getByRole('button', { name: 'Invite user' }));
    expect(screen.getByRole('alert')).toHaveTextContent(
      'Please select a user.',
    );

    await user.click(screen.getByRole('button', { name: 'Select user' }));
    await user.click(screen.getByRole('button', { name: 'Invite user' }));
    expect(screen.getByRole('alert')).toHaveTextContent(
      'Please add at least one project role.',
    );

    await user.click(
      screen.getByRole('button', { name: 'Set incomplete project role' }),
    );
    await user.click(screen.getByRole('button', { name: 'Invite user' }));
    expect(screen.getByRole('alert')).toHaveTextContent(
      'Please select project and role.',
    );

    await user.click(
      screen.getByRole('button', { name: 'Set duplicate project roles' }),
    );
    await user.click(screen.getByRole('button', { name: 'Invite user' }));
    expect(screen.getByRole('alert')).toHaveTextContent(
      'A project can only be assigned once.',
    );
    expect(mockAddUserToProjects).not.toHaveBeenCalled();
  });

  it('adds a selected user to projects', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const setModalOpen = jest.fn();
    const onSuccess = jest.fn();
    mockAddUserToProjects.mockResolvedValue(successResponse());

    renderWithAuth(
      <InviteProjectUserDialog
        modalOpen
        setModalOpen={setModalOpen}
        onSuccess={onSuccess}
      />,
    );

    await user.click(screen.getByRole('button', { name: 'Select user' }));
    await user.click(
      screen.getByRole('button', { name: 'Set valid project role' }),
    );
    await user.click(screen.getByRole('button', { name: 'Invite user' }));

    await waitFor(() => expect(setModalOpen).toHaveBeenCalledWith(false));
    const request = mockAddUserToProjects.mock.calls[0][1];
    expect(request.userId).toBe('user-2');
    expect(request.projectRoles).toEqual([
      { projectId: 'project-1', projectRole: 'super admin' },
    ]);
    expect(mockToast.success).toHaveBeenCalledWith(
      'The user was added to the project successfully.',
    );
    expect(onSuccess).toHaveBeenCalledTimes(1);
  });

  it('adds a fixed user to a fixed project', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const setModalOpen = jest.fn();
    mockAddUserToProjects.mockResolvedValue(successResponse());

    renderWithAuth(
      <InviteProjectUserDialog
        modalOpen
        setModalOpen={setModalOpen}
        user={mockAvailableUser as any}
        projectId="project-2"
        onSuccess={jest.fn()}
      />,
    );

    expect(screen.getByLabelText('User')).toHaveValue(
      'Priyanka Iyer (p_iyer@outlook.com)',
    );

    await user.click(
      screen.getByRole('button', { name: 'Set valid project role' }),
    );
    await user.click(screen.getByRole('button', { name: 'Invite user' }));

    await waitFor(() => expect(setModalOpen).toHaveBeenCalledWith(false));
    const request = mockAddUserToProjects.mock.calls[0][1];
    expect(request.userId).toBe('user-2');
    expect(request.projectRoles).toEqual([
      { projectId: 'project-2', projectRole: 'super admin' },
    ]);
  });

  it('shows project invitation API errors', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    mockAddUserToProjects.mockResolvedValue(errorResponse('Access failed'));

    renderWithAuth(
      <InviteProjectUserDialog
        modalOpen
        setModalOpen={jest.fn()}
        user={mockAvailableUser as any}
        projectId="project-1"
        onSuccess={jest.fn()}
      />,
    );

    await user.click(
      screen.getByRole('button', { name: 'Set valid project role' }),
    );
    await user.click(screen.getByRole('button', { name: 'Invite user' }));

    expect(await screen.findByRole('alert')).toHaveTextContent('Access failed');
    expect(mockToast.error).toHaveBeenCalledWith('Access failed');
  });

  it('shows project invitation thrown errors', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    mockAddUserToProjects.mockRejectedValue(new Error('Project access failed'));

    renderWithAuth(
      <InviteProjectUserDialog
        modalOpen
        setModalOpen={jest.fn()}
        user={mockAvailableUser as any}
        projectId="project-1"
        onSuccess={jest.fn()}
      />,
    );

    await user.click(
      screen.getByRole('button', { name: 'Set valid project role' }),
    );
    await user.click(screen.getByRole('button', { name: 'Invite user' }));

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Project access failed',
    );
    expect(mockToast.error).toHaveBeenCalledWith('Project access failed');
  });
});
