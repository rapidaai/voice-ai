import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import { AuthenticationType } from '../types/types.authentication';
import {
  Authentication,
  Token,
  ProjectRole,
  OrganizationRole,
  AuthenticateResponse,
} from '@rapidaai/react';
import { User } from '@rapidaai/react';
import { AuthorizeUser } from '@rapidaai/react';
import { ServiceError } from '@rapidaai/react';
import { connectionConfig } from '@/configs';

export const useAuthenticationStore = create<AuthenticationType>()(
  persist(
    (set, get) => ({
      currentUser: {} as User.AsObject,
      setCurrentUser: (u: User.AsObject) => set({ currentUser: u }),
      token: {} as Token.AsObject,
      setToken: (t: Token.AsObject) => set({ token: t }),
      setAuthentication: (
        a: Authentication | undefined,
        callback: () => void,
      ) => {
        if (a) {
          const projectRoles = a.getProjectrolesList().map(pr => pr.toObject());
          const currentProjectRole = get().currentProjectRole;

          let newCurrentProjectRole = currentProjectRole;
          if (
            Object.keys(currentProjectRole).length === 0 &&
            projectRoles.length > 0
          ) {
            newCurrentProjectRole = projectRoles[0];
          } else if (projectRoles.length > 0) {
            const matchedRole = projectRoles.find(
              role => role.id === currentProjectRole.id,
            );
            newCurrentProjectRole = matchedRole || projectRoles[0];
          }

          set({
            currentUser: a.getUser()?.toObject(),
            token: a.getToken()?.toObject(),
            organizationRole: a.getOrganizationrole()?.toObject(),
            projectRoles: a.getProjectrolesList().map(pr => pr.toObject()),
            currentProjectRole: newCurrentProjectRole,
            featurePermissions: a
              .getFeaturepermissionsList()
              .map(p => p.toObject()),
          });
        } else {
          set({});
        }
        callback();
      },
      isAuthenticated: () => !!get().token.token,
      authorize: (
        onSuccess: () => void,
        onFailure: (error: string) => void,
      ) => {
        const { token, currentUser } = get();
        if (!token.token) {
          onFailure('Missing token');
          return;
        }
        AuthorizeUser(
          connectionConfig,
          (err: ServiceError | null, auth: AuthenticateResponse | null) => {
            if (err) {
              onFailure(err.message);
              return;
            }
            if (auth?.getSuccess()) {
              get().setAuthentication(auth.getData(), onSuccess);
              onSuccess();
            } else {
              onFailure('Authorization failed');
            }
          },
          {
            authorization: token.token,
            'x-auth-id': currentUser.id,
          },
        );
      },
      projectRoles: [],
      organizationRole: {} as OrganizationRole.AsObject,
      currentProjectRole: {} as ProjectRole.AsObject,
      setCurrentProjectRole: (p: ProjectRole.AsObject) =>
        set({ currentProjectRole: p }),
      isThereOrganization: () => {
        const state = get();
        return (
          state.organizationRole &&
          Object.keys(state.organizationRole).length > 0
        );
      },
      isThereProject: () => {
        return get().projectRoles && Object.keys(get().projectRoles).length > 0;
      },
      featurePermissions: [],
      isFeatureEnable: (feature: string) => {
        for (const permission of get().featurePermissions) {
          const pattern = new RegExp(permission.feature);
          if (pattern.test(feature)) {
            return permission.isenable;
          }
        }
        return false;
      },

      unauthenticate: () => {
        localStorage.clear();
        set({
          currentUser: {} as User.AsObject,
          token: {} as Token.AsObject,
          organizationRole: {} as OrganizationRole.AsObject,
          projectRoles: [],
          currentProjectRole: {} as ProjectRole.AsObject,
          featurePermissions: [],
        });
      },
    }),
    {
      version: 1,
      onRehydrateStorage: () => state => {},
      name: 'rpd::__user',
    },
  ),
);
