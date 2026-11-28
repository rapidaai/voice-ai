import { authorizeUser } from '@/clients/authentication.client';
import { useAuthenticationStore } from '@/stores/auth/authentication.store';

jest.mock('@/clients/authentication.client', () => ({
  authorizeUser: jest.fn(),
}));

const emptyAuthenticationState = {
  authentication: {},
  token: {},
  selectedOrganizationID: '',
  selectedProjectID: '',
};

function authenticationFields({
  organizationID = 'organization-2',
  projectID = 'project-2',
  firstOrganizationHasProject = true,
}: {
  organizationID?: string;
  projectID?: string;
  firstOrganizationHasProject?: boolean;
} = {}) {
  const organizations = [
    {
      id: 'organization-1',
      name: 'First organization',
      rolesList: [],
      projectsList: firstOrganizationHasProject
        ? [{ id: 'project-0', name: 'Default project', rolesList: [] }]
        : [],
    },
    {
      id: 'organization-2',
      name: 'Selected organization',
      rolesList: [],
      projectsList: [
        { id: 'project-1', name: 'First project', rolesList: [] },
        { id: 'project-2', name: 'Selected project', rolesList: [] },
      ],
    },
  ];

  return {
    authentication: {
      toObject: () => ({
        principal: {
          selectedtenant: {
            organizationid: organizationID,
            projectid: projectID,
          },
        },
        user: { id: 'user-1', email: 'user@example.com' },
        organizationsList: organizations,
      }),
      getOrganizationsList: () =>
        organizations.map(organization => ({
          toObject: () => organization,
        })),
      getPrincipal: () => ({
        getSelectedtenant: () => ({
          getOrganizationid: () => organizationID,
          getProjectid: () => projectID,
        }),
      }),
      getUser: () => ({
        toObject: () => ({ id: 'user-1', email: 'user@example.com' }),
      }),
    },
    token: {
      toObject: () => ({ token: 'session-token' }),
    },
  };
}

describe('useAuthenticationStore', () => {
  beforeEach(() => {
    localStorage.clear();
    jest.clearAllMocks();
    useAuthenticationStore.setState(emptyAuthenticationState as never);
  });

  it('stores the authentication graph and token and selects its tenant', () => {
    const onStored = jest.fn();
    const { authentication, token } = authenticationFields();

    const stored = useAuthenticationStore
      .getState()
      .setAuthentication(authentication as never, token as never, onStored);

    const state = useAuthenticationStore.getState();
    expect(state.authentication.user).toEqual({
      id: 'user-1',
      email: 'user@example.com',
    });
    expect(state.token).toEqual({ token: 'session-token' });
    expect(state.authentication.organizationsList).toHaveLength(2);
    expect(state.getCurrentOrganization()?.id).toBe('organization-2');
    expect(state.getProjects()).toHaveLength(2);
    expect(state.getCurrentProject()?.id).toBe('project-2');
    expect(stored).toBe(true);
    expect(onStored).toHaveBeenCalledTimes(1);
  });

  it('defaults a tenantless multi-organization authentication to its first context', () => {
    const { authentication, token } = authenticationFields({
      organizationID: '',
      projectID: '',
    });
    const stored = useAuthenticationStore
      .getState()
      .setAuthentication(authentication as never, token as never, jest.fn());

    const state = useAuthenticationStore.getState();
    expect(stored).toBe(true);
    expect(state.hasOrganizations()).toBe(true);
    expect(state.getCurrentOrganization()?.id).toBe('organization-1');
    expect(state.getCurrentProject()?.id).toBe('project-0');
  });

  it('keeps a tenantless projectless first organization at organization scope', () => {
    const { authentication, token } = authenticationFields({
      organizationID: '',
      projectID: '',
      firstOrganizationHasProject: false,
    });
    useAuthenticationStore
      .getState()
      .setAuthentication(authentication as never, token as never, jest.fn());

    const state = useAuthenticationStore.getState();
    expect(state.getCurrentOrganization()?.id).toBe('organization-1');
    expect(state.getCurrentProject()).toBeUndefined();

    const secondOrganization = state.getOrganizations()[1];
    state.setCurrentContext(secondOrganization);
    expect(useAuthenticationStore.getState().getCurrentOrganization()?.id).toBe(
      'organization-2',
    );
    expect(useAuthenticationStore.getState().getCurrentProject()?.id).toBe(
      'project-1',
    );
  });

  it('keeps a valid prior context ahead of the tenantless fallback', () => {
    useAuthenticationStore.setState({
      selectedOrganizationID: 'organization-2',
      selectedProjectID: 'project-2',
    });
    const { authentication, token } = authenticationFields({
      organizationID: '',
      projectID: '',
    });

    useAuthenticationStore
      .getState()
      .setAuthentication(authentication as never, token as never, jest.fn());

    expect(useAuthenticationStore.getState()).toMatchObject({
      selectedOrganizationID: 'organization-2',
      selectedProjectID: 'project-2',
    });
  });

  it('ignores stale prior context and uses the first verified context', () => {
    useAuthenticationStore.setState({
      selectedOrganizationID: 'stale-organization',
      selectedProjectID: 'stale-project',
    });
    const { authentication, token } = authenticationFields({
      organizationID: '',
      projectID: '',
    });

    useAuthenticationStore
      .getState()
      .setAuthentication(authentication as never, token as never, jest.fn());

    expect(useAuthenticationStore.getState()).toMatchObject({
      selectedOrganizationID: 'organization-1',
      selectedProjectID: 'project-0',
    });
  });

  it('changes organization and project together only for a valid pair', () => {
    const { authentication, token } = authenticationFields();
    useAuthenticationStore
      .getState()
      .setAuthentication(authentication as never, token as never, jest.fn());
    const state = useAuthenticationStore.getState();
    const firstOrganization = state.getOrganizations()[0];
    const secondOrganization = state.getOrganizations()[1];
    const secondProject = secondOrganization.projectsList[1];

    expect(state.setCurrentContext(secondOrganization, secondProject)).toBe(
      true,
    );
    expect(useAuthenticationStore.getState()).toMatchObject({
      selectedOrganizationID: 'organization-2',
      selectedProjectID: 'project-2',
    });

    expect(state.setCurrentContext(firstOrganization, secondProject)).toBe(
      false,
    );
    expect(useAuthenticationStore.getState()).toMatchObject({
      selectedOrganizationID: 'organization-2',
      selectedProjectID: 'project-2',
    });
  });

  it('fails closed when the response omits its authentication aggregate', () => {
    useAuthenticationStore.setState({
      authentication: {
        user: { id: 'stale-user' },
        organizationsList: [],
      } as never,
      token: { token: 'stale-token' } as never,
      selectedOrganizationID: 'stale-organization',
      selectedProjectID: 'stale-project',
    });
    const onCleared = jest.fn();

    const stored = useAuthenticationStore
      .getState()
      .setAuthentication(
        undefined,
        { toObject: () => ({ token: 'new-token' }) } as never,
        onCleared,
      );

    expect(useAuthenticationStore.getState()).toMatchObject(
      emptyAuthenticationState,
    );
    expect(stored).toBe(false);
    expect(onCleared).not.toHaveBeenCalled();
  });

  it('fails closed when the response omits its token', () => {
    const { authentication } = authenticationFields();
    const onCleared = jest.fn();

    const stored = useAuthenticationStore
      .getState()
      .setAuthentication(authentication as never, undefined, onCleared);

    expect(useAuthenticationStore.getState()).toMatchObject(
      emptyAuthenticationState,
    );
    expect(stored).toBe(false);
    expect(onCleared).not.toHaveBeenCalled();
  });

  it('refreshes the complete authentication through Authorize', () => {
    const { authentication, token } = authenticationFields();
    (authorizeUser as jest.Mock).mockImplementation(callback =>
      callback(null, {
        getSuccess: () => true,
        getAuthentication: () => authentication,
        getToken: () => token,
      }),
    );
    useAuthenticationStore.setState({
      authentication: {
        user: { id: 'user-1' },
        organizationsList: [],
      } as never,
      token: { token: 'old-token' } as never,
      selectedOrganizationID: 'organization-2',
    });
    const onSuccess = jest.fn();
    const onFailure = jest.fn();

    useAuthenticationStore.getState().authorize(onSuccess, onFailure);

    expect(authorizeUser).toHaveBeenCalledWith(expect.any(Function), {
      authorization: 'old-token',
      'x-auth-id': 'user-1',
      'x-organization-id': 'organization-2',
    });
    expect(useAuthenticationStore.getState().token).toEqual({
      token: 'session-token',
    });
    expect(onSuccess).toHaveBeenCalledTimes(1);
    expect(onFailure).not.toHaveBeenCalled();
  });

  it('rejects an Authorize response without a complete authentication graph', () => {
    const { token } = authenticationFields();
    (authorizeUser as jest.Mock).mockImplementation(callback =>
      callback(null, {
        getSuccess: () => true,
        getAuthentication: () => undefined,
        getToken: () => token,
      }),
    );
    useAuthenticationStore.setState({
      authentication: {
        user: { id: 'user-1' },
        organizationsList: [],
      } as never,
      token: { token: 'old-token' } as never,
      selectedOrganizationID: '',
    });
    const onSuccess = jest.fn();
    const onFailure = jest.fn();

    useAuthenticationStore.getState().authorize(onSuccess, onFailure);

    expect(onSuccess).not.toHaveBeenCalled();
    expect(onFailure).toHaveBeenCalledWith('Authorization failed');
    expect(useAuthenticationStore.getState()).toMatchObject(
      emptyAuthenticationState,
    );
  });

  it('discards the persisted legacy authentication shape', async () => {
    localStorage.setItem(
      'rpd::__user',
      JSON.stringify({
        version: 1,
        state: {
          currentUser: { id: 'legacy-user' },
          [['organization', 'Role'].join('')]: {
            organizationid: 'legacy-organization',
          },
          [['project', 'Roles'].join('')]: [{ projectid: 'legacy-project' }],
          [['feature', 'Permissions'].join('')]: [{ key: 'legacy.permission' }],
        },
      }),
    );

    await useAuthenticationStore.persist.rehydrate();

    const state = useAuthenticationStore.getState();
    expect(state).toMatchObject(emptyAuthenticationState);
    expect(state).not.toHaveProperty(['organization', 'Role'].join(''));
    expect(state).not.toHaveProperty(['project', 'Roles'].join(''));
    expect(state).not.toHaveProperty(['feature', 'Permissions'].join(''));
  });
});
