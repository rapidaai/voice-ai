import { useCallback, useContext, useEffect, useMemo, useState } from 'react';
import {
  RoleScope,
  RoleSubject,
  type RoleDefinition,
  type ServiceError,
  type User,
} from '@rapidaai/react';
import toast from 'react-hot-toast/headless';
import {
  ComboBox,
  Modal,
  ModalBody,
  ModalFooter,
  ModalHeader,
  type ModalProps,
  PrimaryButton,
  SecondaryButton,
  Stack,
  TextInput,
} from '@/app/components/ui/primitives';
import { ErrorMessage } from '@/app/components/ui/feedback';
import { AuthContext } from '@/context/auth-context';
import { useCurrentCredential } from '@/hooks/use-credential';
import { useRapidaStore } from '@/stores/app';
import { useUserPageStore } from '@/stores/user';
import {
  ProjectRoleRow,
  ProjectRoleTable,
} from '@/app/components/domain/project-role-table';
import { addUserToProjectsByRole, listAccessRoles } from '@/clients';

const PROJECT_ACTION_ERROR =
  'Unable to process your request. please try again later.';

interface InviteProjectUserDialogProps extends ModalProps {
  user?: User | null;
  projectId?: string;
  onSuccess?: () => void;
}

export function InviteProjectUserDialog({
  modalOpen,
  setModalOpen,
  user,
  projectId,
  onSuccess,
}: InviteProjectUserDialogProps) {
  const {
    authId,
    token,
    organizationId,
    projectId: currentProjectId,
  } = useCurrentCredential();
  const { getProjects } = useContext(AuthContext);
  const availableProjects = getProjects?.();
  const { loading, showLoader, hideLoader } = useRapidaStore();
  const { users, getAllUser } = useUserPageStore();
  const [projectRoles, setProjectRoles] = useState<RoleDefinition[]>([]);
  const [selectedUser, setSelectedUser] = useState<User | null>(user || null);
  const [projectRoleRows, setProjectRoleRows] = useState<ProjectRoleRow[]>(
    projectId ? [{ projectId, roleDefinitionIds: [] }] : [],
  );
  const [isRoleCatalogLoading, setIsRoleCatalogLoading] = useState(false);
  const [isRoleCatalogReady, setIsRoleCatalogReady] = useState(false);
  const [error, setError] = useState('');

  const closeDialog = useCallback(() => {
    setModalOpen(false);
  }, [setModalOpen]);

  useEffect(() => {
    if (modalOpen) {
      setSelectedUser(user || null);
      setProjectRoleRows(
        projectId ? [{ projectId, roleDefinitionIds: [] }] : [],
      );
      setError('');

      if (!user && users.length === 0) {
        getAllUser(
          token,
          authId,
          organizationId,
          currentProjectId,
          err => toast.error(err),
          () => {},
        );
      }
    }
  }, [
    authId,
    currentProjectId,
    modalOpen,
    organizationId,
    projectId,
    token,
    user,
    users.length,
    getAllUser,
  ]);

  useEffect(() => {
    if (!modalOpen) return;

    let isCurrent = true;
    setProjectRoles([]);
    setProjectRoleRows(rows =>
      rows.map(row => ({ ...row, roleDefinitionIds: [] })),
    );
    setIsRoleCatalogReady(false);
    setIsRoleCatalogLoading(true);
    (async () => {
      const roles: RoleDefinition[] = [];
      let page = 1;
      let received = 0;
      let total = 0;

      do {
        const response = await listAccessRoles({
          page,
          pageSize: 100,
          criteria: [
            { key: 'scope', logic: '=', value: 'PROJECT' },
            { key: 'subject', logic: '=', value: 'USER' },
          ],
          auth: { organizationId, token, userId: authId },
        });
        if (!response.getSuccess()) {
          throw new Error(
            response.getError()?.getHumanmessage() ||
              'Project roles could not be loaded.',
          );
        }
        const pageRoles = response.getRolesList();
        received += pageRoles.length;
        roles.push(
          ...pageRoles.filter(
            role =>
              role.getScope() === RoleScope.ROLE_SCOPE_PROJECT &&
              role.getSubject() === RoleSubject.ROLE_SUBJECT_USER,
          ),
        );
        total = response.getPaginated()?.getTotalitem() || received;
        if (pageRoles.length === 0 && received < total) {
          throw new Error('The role catalog returned an incomplete result.');
        }
        page += 1;
      } while (received < total);

      return roles;
    })()
      .then(roles => {
        if (!isCurrent) return;
        setProjectRoles(roles);
        setIsRoleCatalogReady(true);
        setIsRoleCatalogLoading(false);
      })
      .catch((requestError: ServiceError) => {
        if (isCurrent) {
          setIsRoleCatalogLoading(false);
          toast.error(
            requestError.message || 'Project roles could not be loaded.',
          );
        }
      });

    return () => {
      isCurrent = false;
    };
  }, [authId, modalOpen, token]);

  const submitInvite = async () => {
    if (!selectedUser) {
      setError('Please select a user.');
      return;
    }
    if (projectRoleRows.length === 0) {
      setError('Please add at least one project role.');
      return;
    }
    if (
      projectRoleRows.some(
        row => !row.projectId || row.roleDefinitionIds.length === 0,
      )
    ) {
      setError('Select a project and at least one role.');
      return;
    }
    const projectIds = projectRoleRows.map(row => row.projectId);
    if (new Set(projectIds).size !== projectIds.length) {
      setError('A project can only be assigned once.');
      return;
    }

    setError('');
    showLoader('overlay');

    try {
      const response = await addUserToProjectsByRole({
        userId: selectedUser.getId(),
        projectRoles: projectRoleRows,
        auth: { organizationId, token, userId: authId },
      });
      hideLoader();

      const responseError = response.getError();
      const message = responseError?.getHumanmessage() || PROJECT_ACTION_ERROR;

      if (response.getSuccess()) {
        setSelectedUser(null);
        setProjectRoleRows(
          projectId
            ? [
                {
                  projectId,
                  roleDefinitionIds: [],
                },
              ]
            : [],
        );
        closeDialog();
        toast.success('The user was added to the project successfully.');
        onSuccess?.();
        return;
      }

      toast.error(message);
    } catch (err: unknown) {
      hideLoader();
      const message = err instanceof Error ? err.message : PROJECT_ACTION_ERROR;
      toast.error(message);
    }
  };

  const projectOptions = useMemo(
    () =>
      projectId
        ? [
            {
              name:
                (availableProjects || []).find(
                  project => project.id === projectId,
                )?.name || projectId,
              value: projectId,
            },
          ]
        : (availableProjects || []).map(project => ({
            name: project.name,
            value: project.id,
          })),
    [availableProjects, projectId],
  );

  return (
    <Modal
      open={modalOpen}
      onClose={closeDialog}
      size="md"
      preventCloseOnClickOutside
    >
      <ModalHeader
        label="Project Access"
        title="Invite User to Project"
        onClose={closeDialog}
      />
      <ModalBody hasForm hasScrollingContent>
        <Stack gap={6}>
          {user ? (
            <TextInput
              id="project-invite-selected-user"
              labelText="User"
              value={`${user.getName()} (${user.getEmail()})`}
              readOnly
            />
          ) : (
            <ComboBox
              id="project-invite-user"
              titleText="User"
              placeholder="Select user"
              items={users}
              selectedItem={selectedUser}
              itemToString={(item: User | null) =>
                item ? `${item.getName()} (${item.getEmail()})` : ''
              }
              onChange={({ selectedItem }) => {
                setError('');
                setSelectedUser(selectedItem || null);
              }}
            />
          )}
          <ProjectRoleTable
            rows={projectRoleRows}
            onChange={next => {
              setError('');
              setProjectRoleRows(next);
            }}
            projectOptions={projectOptions}
            roleOptions={projectRoles.map(role => ({
              name: role.getName(),
              value: role.getId(),
            }))}
            defaultProjectId={projectId || projectOptions[0]?.value || ''}
            title="Project roles"
            addButtonLabel="Add project role"
            showAddButton={!projectId}
            showRemoveColumn={!projectId}
          />
          <ErrorMessage message={error} />
        </Stack>
      </ModalBody>
      <ModalFooter>
        <SecondaryButton size="lg" onClick={closeDialog}>
          Cancel
        </SecondaryButton>
        <PrimaryButton
          size="lg"
          onClick={submitInvite}
          disabled={!isRoleCatalogReady}
          isLoading={loading || isRoleCatalogLoading}
        >
          Invite user
        </PrimaryButton>
      </ModalFooter>
    </Modal>
  );
}
