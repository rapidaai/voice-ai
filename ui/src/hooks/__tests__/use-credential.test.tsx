import React from 'react';
import { renderHook } from '@testing-library/react';

import { AuthContext } from '@/context/auth-context';
import {
  useAuth,
  useCredential,
  useCurrentCredential,
  useResourceRole,
} from '@/hooks/use-credential';
import { ResourceRole } from '@/models/common';

type AuthValue = React.ContextType<typeof AuthContext>;

const makeWrapper = (value: AuthValue) => {
  const Wrapper: React.FC<{ children: React.ReactNode }> = ({ children }) => (
    <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
  );
  return Wrapper;
};

describe('use-credential hooks', () => {
  it('throws when useAuth is called with null context value', () => {
    const wrapper = makeWrapper(null as unknown as AuthValue);
    const consoleErrorSpy = jest
      .spyOn(console, 'error')
      .mockImplementation(() => undefined);

    expect(() => renderHook(() => useAuth(), { wrapper })).toThrow(
      'useAuth must be used within an AuthProvider',
    );

    consoleErrorSpy.mockRestore();
  });

  it('returns credential tuple with empty-string fallbacks', () => {
    const wrapper = makeWrapper({
      authentication: { user: { id: 'user-1' } } as any,
      token: {} as any,
      getCurrentProject: () => undefined,
      getCurrentOrganization: () => undefined,
    });

    const { result } = renderHook(() => useCredential(), { wrapper });
    expect(result.current).toEqual(['user-1', '', '', '']);
  });

  it('returns current credential object', () => {
    const user = { id: 'user-1', email: 'a@b.c' } as any;
    const wrapper = makeWrapper({
      authentication: { user } as any,
      token: { token: 'token-1' } as any,
      getCurrentProject: () => ({ id: 'project-1' }) as any,
      getCurrentOrganization: () => ({ id: 'org-1' }) as any,
    });

    const { result } = renderHook(() => useCurrentCredential(), { wrapper });
    expect(result.current).toEqual({
      user,
      authId: 'user-1',
      token: 'token-1',
      projectId: 'project-1',
      organizationId: 'org-1',
    });
  });

  it('resolves resource role in owner/project/org/anyone order', () => {
    const wrapper = makeWrapper({
      authentication: { user: { id: 'owner-user' } } as any,
      getProjects: () => [{ id: 'project-1' }, { id: 'project-2' }] as any,
      getCurrentOrganization: () => ({ id: 'org-1' }) as any,
    });

    const ownerResource = {
      getCreatedby: () => 'owner-user',
      getProjectid: () => 'other-project',
      getOrganizationid: () => 'other-org',
    };
    const projectResource = {
      getCreatedby: () => 'someone-else',
      getProjectid: () => 'project-2',
      getOrganizationid: () => 'other-org',
    };
    const orgResource = {
      getCreatedby: () => 'someone-else',
      getProjectid: () => 'other-project',
      getOrganizationid: () => 'org-1',
    };
    const unrelatedResource = {
      getCreatedby: () => 'someone-else',
      getProjectid: () => 'other-project',
      getOrganizationid: () => 'other-org',
    };

    expect(
      renderHook(() => useResourceRole(ownerResource), { wrapper }).result
        .current,
    ).toBe(ResourceRole.owner);
    expect(
      renderHook(() => useResourceRole(projectResource), { wrapper }).result
        .current,
    ).toBe(ResourceRole.projectMember);
    expect(
      renderHook(() => useResourceRole(orgResource), { wrapper }).result
        .current,
    ).toBe(ResourceRole.organizationMember);
    expect(
      renderHook(() => useResourceRole(unrelatedResource), { wrapper }).result
        .current,
    ).toBe(ResourceRole.anyone);
  });

  it('falls back to anyone when auth context is incomplete', () => {
    const wrapper = makeWrapper({
      authentication: { user: { id: 'owner-user' } } as any,
      getProjects: undefined,
      getCurrentOrganization: () => ({ id: 'org-1' }) as any,
    });

    const resource = {
      getCreatedby: () => 'owner-user',
      getProjectid: () => 'project-1',
      getOrganizationid: () => 'org-1',
    };

    const { result } = renderHook(() => useResourceRole(resource), { wrapper });
    expect(result.current).toBe(ResourceRole.anyone);
  });
});
