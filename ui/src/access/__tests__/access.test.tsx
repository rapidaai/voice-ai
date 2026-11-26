import React from 'react';
import { renderHook } from '@testing-library/react';

import { AuthContext } from '@/context/auth-context';
import { ConsoleAccess, Permission, useAccess } from '@/access';

type AuthValue = React.ContextType<typeof AuthContext>;

function authWrapper(value: AuthValue) {
  return function AccessTestProvider({
    children,
  }: {
    children: React.ReactNode;
  }) {
    return (
      <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
    );
  };
}

describe('useAccess', () => {
  it('publishes every distinct permission from the access catalog', () => {
    const permissions = Object.values(Permission);

    expect(permissions).toHaveLength(64);
    expect(new Set(permissions).size).toBe(64);
  });

  it('allows an organization requirement when every permission is granted', () => {
    const wrapper = authWrapper({
      getCurrentOrganization: () =>
        ({
          rolesList: [
            {
              permissionsList: [
                { key: Permission.OrganizationMembersInvite },
                { key: Permission.OrganizationAccessManage },
              ],
            },
          ],
        }) as never,
      getCurrentProject: () => ({ rolesList: [] }) as never,
    });

    const { result } = renderHook(
      () => useAccess(ConsoleAccess.organization.members.invite, 'ready'),
      { wrapper },
    );

    expect(result.current).toBe('allowed');
  });

  it('denies an all-of requirement when one permission is missing', () => {
    const wrapper = authWrapper({
      getCurrentOrganization: () =>
        ({
          rolesList: [
            {
              permissionsList: [{ key: Permission.OrganizationMembersInvite }],
            },
          ],
        }) as never,
      getCurrentProject: () => ({ rolesList: [] }) as never,
    });

    const { result } = renderHook(
      () => useAccess(ConsoleAccess.organization.members.invite, 'ready'),
      { wrapper },
    );

    expect(result.current).toBe('denied');
  });

  it('evaluates project access against the current project permissions', () => {
    const wrapper = authWrapper({
      getCurrentOrganization: () => ({ rolesList: [] }) as never,
      getCurrentProject: () =>
        ({
          rolesList: [
            {
              permissionsList: [{ key: Permission.ProjectResourcesUpdate }],
            },
          ],
        }) as never,
    });

    const { result } = renderHook(
      () => useAccess(ConsoleAccess.project.resources.update, 'ready'),
      { wrapper },
    );

    expect(result.current).toBe('allowed');
  });

  it('allows an any-of requirement through either scope', () => {
    const wrapper = authWrapper({
      getCurrentOrganization: () =>
        ({
          rolesList: [
            {
              permissionsList: [
                { key: Permission.OrganizationProjectAccessTokensRevoke },
              ],
            },
          ],
        }) as never,
      getCurrentProject: () => ({ rolesList: [] }) as never,
    });

    const { result } = renderHook(
      () => useAccess(ConsoleAccess.projectAccessTokens.revoke, 'ready'),
      { wrapper },
    );

    expect(result.current).toBe('allowed');
  });

  it('allows an any-of requirement through its project alternative', () => {
    const wrapper = authWrapper({
      getCurrentOrganization: () => ({ rolesList: [] }) as never,
      getCurrentProject: () =>
        ({
          rolesList: [
            {
              permissionsList: [{ key: Permission.ProjectAccessTokensRevoke }],
            },
          ],
        }) as never,
    });

    const { result } = renderHook(
      () => useAccess(ConsoleAccess.projectAccessTokens.revoke, 'ready'),
      { wrapper },
    );

    expect(result.current).toBe('allowed');
  });

  it('reports loading when permissions for a required scope are unavailable', () => {
    const wrapper = authWrapper({
      getCurrentOrganization: () => ({}) as never,
      getCurrentProject: () => ({ rolesList: [] }) as never,
    });

    const { result } = renderHook(
      () => useAccess(ConsoleAccess.organization.roles.read, 'loading'),
      { wrapper },
    );

    expect(result.current).toBe('loading');
  });

  it('denies access when the required scope is ready without the permission', () => {
    const wrapper = authWrapper({
      getCurrentOrganization: () => ({ rolesList: [] }) as never,
      getCurrentProject: () => ({ rolesList: [] }) as never,
    });

    const { result } = renderHook(
      () => useAccess(ConsoleAccess.organization.roles.create, 'ready'),
      { wrapper },
    );

    expect(result.current).toBe('denied');
  });

  it('fails closed when a ready projection omits its required scope', () => {
    const wrapper = authWrapper({
      getCurrentOrganization: () => ({}) as never,
      getCurrentProject: () => ({ rolesList: [] }) as never,
    });

    const { result } = renderHook(
      () => useAccess(ConsoleAccess.organization.roles.read, 'ready'),
      { wrapper },
    );

    expect(result.current).toBe('denied');
  });
});
