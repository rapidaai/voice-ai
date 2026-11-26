import {
  AddUserToProjects,
  CreateOrganization,
  CreateProject,
  DeleteProject,
  DeleteUserFromOrganization,
  GetAllProject,
  GetOrganization,
  InviteUserToOrganization,
  UpdateOrganization,
  UpdateProject,
  UpdateUserOrganizationRole,
} from '@rapidaai/react';

import {
  addUserToProjectsByRole,
  createWorkspaceOrganization,
  createWorkspaceProject,
  deleteOrganizationUser,
  deleteWorkspaceProject,
  getWorkspaceOrganization,
  inviteOrganizationUser,
  listWorkspaceProjects,
  updateOrganizationUserRole,
  updateWorkspaceOrganization,
  updateWorkspaceProject,
} from '@/clients';

jest.mock('@/configs', () => ({
  connectionConfig: { endpoint: 'test-endpoint' },
}));

jest.mock('@rapidaai/react', () => {
  class ProjectRoleAssignment {
    private projectId = '';
    private roleDefinitionIds: string[] = [];

    setProjectid(projectId: string) {
      this.projectId = projectId;
    }

    setRoledefinitionidsList(roleDefinitionIds: string[]) {
      this.roleDefinitionIds = roleDefinitionIds;
    }

    getProjectid() {
      return this.projectId;
    }

    getRoledefinitionidsList() {
      return this.roleDefinitionIds;
    }
  }

  class InviteUserToOrganizationRequest {
    private email = '';
    private organizationRoleDefinitionIds: string[] = [];
    private projectRoles: ProjectRoleAssignment[] = [];

    setEmail(email: string) {
      this.email = email;
    }

    setOrganizationroledefinitionidsList(roleDefinitionIds: string[]) {
      this.organizationRoleDefinitionIds = roleDefinitionIds;
    }

    setProjectrolesList(projectRoles: ProjectRoleAssignment[]) {
      this.projectRoles = projectRoles;
    }

    getEmail() {
      return this.email;
    }

    getOrganizationroledefinitionidsList() {
      return this.organizationRoleDefinitionIds;
    }

    getProjectrolesList() {
      return this.projectRoles;
    }
  }

  class AddUserToProjectsRequest {
    private userId = '';
    private projectRoles: ProjectRoleAssignment[] = [];

    setUserid(userId: string) {
      this.userId = userId;
    }

    setProjectrolesList(projectRoles: ProjectRoleAssignment[]) {
      this.projectRoles = projectRoles;
    }

    getUserid() {
      return this.userId;
    }

    getProjectrolesList() {
      return this.projectRoles;
    }
  }

  class UpdateUserOrganizationRoleRequest {
    private userId = '';
    private roleDefinitionIds: string[] = [];

    setUserid(userId: string) {
      this.userId = userId;
    }

    setRoledefinitionidsList(roleDefinitionIds: string[]) {
      this.roleDefinitionIds = roleDefinitionIds;
    }

    getUserid() {
      return this.userId;
    }

    getRoledefinitionidsList() {
      return this.roleDefinitionIds;
    }
  }

  return {
    AddUserToProjects: jest.fn(),
    AddUserToProjectsRequest,
    CreateOrganization: jest.fn(),
    CreateProject: jest.fn(),
    DeleteProject: jest.fn(),
    DeleteUserFromOrganization: jest.fn(),
    DeleteUserFromOrganizationRequest: class {
      private userId = '';

      setUserid(userId: string) {
        this.userId = userId;
      }

      getUserid() {
        return this.userId;
      }
    },
    DeleteUserFromProject: jest.fn(),
    GetAllProject: jest.fn(),
    GetOrganization: jest.fn(),
    GetProject: jest.fn(),
    InviteUserToOrganization: jest.fn(),
    InviteUserToOrganizationRequest,
    ProjectRoleAssignment,
    UpdateOrganization: jest.fn(),
    UpdateProject: jest.fn(),
    UpdateUserOrganizationRole: jest.fn(),
    UpdateUserOrganizationRoleRequest,
  };
});

const auth = {
  organizationId: 'org-1',
  token: 'token-1',
  userId: 'auth-1',
};

const metadata = {
  authorization: 'token-1',
  'x-auth-id': 'auth-1',
  'x-organization-id': 'org-1',
};

describe('workspace client', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('gets and updates organization details with workspace metadata', () => {
    const getCallback = jest.fn();
    const updateCallback = jest.fn();

    getWorkspaceOrganization({ auth, callback: getCallback });
    updateWorkspaceOrganization({
      organizationId: 'org-1',
      name: 'Rapida',
      industry: 'Support',
      contact: 'ops@example.com',
      auth,
      callback: updateCallback,
    });

    expect(GetOrganization).toHaveBeenCalledWith(
      { endpoint: 'test-endpoint' },
      metadata,
      getCallback,
    );
    expect(UpdateOrganization).toHaveBeenCalledWith(
      { endpoint: 'test-endpoint' },
      'org-1',
      metadata,
      updateCallback,
      'Rapida',
      'Support',
      'ops@example.com',
    );
  });

  it('creates, lists, updates, and deletes projects with workspace metadata', () => {
    const callback = jest.fn();
    const criteria = [{ key: 'name', value: 'support' }];

    createWorkspaceProject({
      name: 'Support',
      description: 'Customer support',
      auth,
      callback,
    });
    listWorkspaceProjects({
      page: 2,
      pageSize: 25,
      criteria,
      auth,
      callback,
    });
    updateWorkspaceProject({
      projectId: 'project-1',
      name: 'Sales',
      description: 'Revenue',
      auth,
      callback,
    });
    deleteWorkspaceProject({ projectId: 'project-1', auth, callback });

    expect(CreateProject).toHaveBeenCalledWith(
      { endpoint: 'test-endpoint' },
      'Support',
      'Customer support',
      metadata,
      callback,
    );
    expect(GetAllProject).toHaveBeenCalledWith(
      { endpoint: 'test-endpoint' },
      2,
      25,
      criteria,
      callback,
      metadata,
    );
    expect(UpdateProject).toHaveBeenCalledWith(
      { endpoint: 'test-endpoint' },
      'project-1',
      callback,
      metadata,
      'Sales',
      'Revenue',
    );
    expect(DeleteProject).toHaveBeenCalledWith(
      { endpoint: 'test-endpoint' },
      'project-1',
      callback,
      metadata,
    );
  });

  it('creates organization users and project role assignment requests', () => {
    createWorkspaceOrganization({
      name: 'Rapida',
      size: '50',
      industry: 'Support',
      auth: { token: auth.token, userId: auth.userId },
      callback: jest.fn(),
    });
    inviteOrganizationUser({
      email: 'p_iyer@outlook.com',
      organizationRoleDefinitionIds: ['role-org-admin'],
      projectRoles: [
        { projectId: 'project-1', roleDefinitionIds: ['role-editor'] },
      ],
      auth,
    });
    addUserToProjectsByRole({
      userId: 'user-1',
      projectRoles: [
        { projectId: 'project-2', roleDefinitionIds: ['role-reader'] },
      ],
      auth,
    });

    expect(CreateOrganization).toHaveBeenCalledWith(
      { endpoint: 'test-endpoint' },
      'Rapida',
      '50',
      'Support',
      {
        authorization: 'token-1',
        'x-auth-id': 'auth-1',
      },
      expect.any(Function),
    );

    const inviteRequest = (InviteUserToOrganization as jest.Mock).mock
      .calls[0][1];
    expect(inviteRequest.getEmail()).toBe('p_iyer@outlook.com');
    expect(inviteRequest.getOrganizationroledefinitionidsList()).toEqual([
      'role-org-admin',
    ]);
    expect(inviteRequest.getProjectrolesList()[0].getProjectid()).toBe(
      'project-1',
    );
    expect(
      inviteRequest.getProjectrolesList()[0].getRoledefinitionidsList(),
    ).toEqual(['role-editor']);

    const projectRequest = (AddUserToProjects as jest.Mock).mock.calls[0][1];
    expect(projectRequest.getUserid()).toBe('user-1');
    expect(projectRequest.getProjectrolesList()[0].getProjectid()).toBe(
      'project-2',
    );
    expect(
      projectRequest.getProjectrolesList()[0].getRoledefinitionidsList(),
    ).toEqual(['role-reader']);
  });

  it('creates organization user delete and role update requests', () => {
    deleteOrganizationUser({ userId: 'user-1', auth });
    updateOrganizationUserRole({
      userId: 'user-2',
      roleDefinitionIds: ['role-member', 'role-auditor'],
      auth,
    });

    const deleteRequest = (DeleteUserFromOrganization as jest.Mock).mock
      .calls[0][1];
    expect(deleteRequest.getUserid()).toBe('user-1');
    expect(DeleteUserFromOrganization).toHaveBeenCalledWith(
      { endpoint: 'test-endpoint' },
      deleteRequest,
      metadata,
    );

    const updateRequest = (UpdateUserOrganizationRole as jest.Mock).mock
      .calls[0][1];
    expect(updateRequest.getUserid()).toBe('user-2');
    expect(updateRequest.getRoledefinitionidsList()).toEqual([
      'role-member',
      'role-auditor',
    ]);
    expect(UpdateUserOrganizationRole).toHaveBeenCalledWith(
      { endpoint: 'test-endpoint' },
      updateRequest,
      metadata,
    );
  });
});
