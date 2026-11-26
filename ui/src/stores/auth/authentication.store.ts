import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import { AuthenticationType } from '@/types/types.authentication';
import type {
  AuthenticateResponse,
  Authentication,
  ServiceError,
  Token,
} from '@rapidaai/react';
import { authorizeUser } from '@/clients/authentication.client';

type AuthenticationState = Pick<
  AuthenticationType,
  'authentication' | 'token' | 'selectedOrganizationID' | 'selectedProjectID'
>;

const emptyAuthenticationState: AuthenticationState = {
  authentication: {} as Authentication.AsObject,
  token: {} as Token.AsObject,
  selectedOrganizationID: '',
  selectedProjectID: '',
};

export const useAuthenticationStore = create<AuthenticationType>()(
  persist<AuthenticationType, [], [], AuthenticationState>(
    (set, get) => ({
      ...emptyAuthenticationState,

      setAuthentication: (
        authentication: Authentication | undefined,
        token: Token | undefined,
        callback: () => void,
      ) => {
        if (!authentication || !token) {
          set(emptyAuthenticationState);
          return false;
        }

        const authenticationValue = authentication.toObject();
        const principalOrganizationID =
          authenticationValue.principal?.selectedtenant?.organizationid;
        const previousOrganizationID = get().selectedOrganizationID;
        const organizations = authenticationValue.organizationsList || [];
        const currentOrganization =
          organizations.find(
            organization => organization.id === principalOrganizationID,
          ) ||
          organizations.find(
            organization => organization.id === previousOrganizationID,
          ) ||
          (organizations.length === 1 ? organizations[0] : undefined);
        const selectedOrganizationID = currentOrganization?.id || '';
        const projects = currentOrganization?.projectsList || [];
        const principalProjectID =
          authenticationValue.principal?.selectedtenant?.projectid;
        const previousProjectID = get().selectedProjectID;
        const selectedProjectID =
          projects.find(project => project.id === principalProjectID)?.id ||
          projects.find(project => project.id === previousProjectID)?.id ||
          projects[0]?.id ||
          '';

        set({
          authentication: authenticationValue,
          token: token.toObject(),
          selectedOrganizationID,
          selectedProjectID,
        });
        callback();
        return true;
      },

      setCurrentContext: (organization, project) => {
        const selected = get()
          .getOrganizations()
          .find(candidate => candidate.id === organization.id);
        if (!selected) return false;
        if (
          project &&
          !selected.projectsList.some(candidate => candidate.id === project.id)
        )
          return false;
        const selectedProjectID =
          project?.id ||
          (selected.projectsList.some(
            candidate => candidate.id === get().selectedProjectID,
          )
            ? get().selectedProjectID
            : selected.projectsList[0]?.id || '');
        set({ selectedOrganizationID: selected.id, selectedProjectID });
        return true;
      },
      getOrganizations: () => get().authentication.organizationsList || [],
      getCurrentOrganization: () => {
        const { selectedOrganizationID } = get();
        return get()
          .getOrganizations()
          .find(organization => organization.id === selectedOrganizationID);
      },
      getProjects: () => get().getCurrentOrganization()?.projectsList || [],
      getCurrentProject: () => {
        const { selectedProjectID } = get();
        return get()
          .getProjects()
          .find(project => project.id === selectedProjectID);
      },

      isAuthenticated: () => !!get().token.token,
      authorize: (onSuccess, onFailure) => {
        const { authentication, selectedOrganizationID, token } = get();
        if (!token.token) {
          onFailure('Missing token');
          return;
        }
        authorizeUser(
          (
            error: ServiceError | null,
            response: AuthenticateResponse | null,
          ) => {
            if (error) {
              onFailure(error.message);
              return;
            }
            if (!response?.getSuccess()) {
              onFailure('Authorization failed');
              return;
            }
            if (
              !get().setAuthentication(
                response.getAuthentication(),
                response.getToken(),
                onSuccess,
              )
            ) {
              onFailure('Authorization failed');
            }
          },
          {
            authorization: token.token,
            'x-auth-id': authentication.user?.id || '',
            ...(selectedOrganizationID
              ? { 'x-organization-id': selectedOrganizationID }
              : {}),
          },
        );
      },

      hasOrganizations: () => get().getOrganizations().length > 0,
      isThereOrganization: () => !!get().getCurrentOrganization()?.id,
      isThereProject: () => !!get().getCurrentProject()?.id,
      unauthenticate: () => {
        localStorage.clear();
        set(emptyAuthenticationState);
      },
    }),
    {
      version: 3,
      migrate: (_persistedState, version) =>
        version < 3
          ? (emptyAuthenticationState as unknown as AuthenticationType)
          : (_persistedState as AuthenticationType),
      partialize: state => ({
        authentication: state.authentication,
        token: state.token,
        selectedOrganizationID: state.selectedOrganizationID,
        selectedProjectID: state.selectedProjectID,
      }),
      name: 'rpd::__user',
    },
  ),
);
