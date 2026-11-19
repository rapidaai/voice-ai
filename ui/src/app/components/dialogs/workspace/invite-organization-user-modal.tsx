import { useCallback, useContext, useEffect, useMemo, useState } from 'react';
import {
  RoleScope,
  RoleSubject,
  type RoleDefinition,
  type ServiceError,
} from '@rapidaai/react';
import toast from 'react-hot-toast/headless';
import {
  Modal,
  ModalBody,
  ModalFooter,
  ModalHeader,
  MultiSelect,
  type ModalProps,
  PrimaryButton,
  SecondaryButton,
  Stack,
  Text,
  TextInput,
  typographyClassNames,
} from '@/app/components/ui/primitives';
import { HelpToggletip } from '@/app/components/ui/composites';
import { ErrorMessage } from '@/app/components/ui/feedback';
import { AuthContext } from '@/context/auth-context';
import { useCurrentCredential } from '@/hooks/use-credential';
import { useRapidaStore } from '@/stores/app';
import {
  ProjectRoleRow,
  ProjectRoleTable,
} from '@/app/components/domain/project-role-table';
import { inviteOrganizationUser, listAccessRoles } from '@/clients';

const PROJECT_ACTION_ERROR =
  'Unable to process your request. please try again later.';

type OrganizationInviteFailure = {
  getHumanmessage?: () => string;
  message?: string;
};

interface InviteOrganizationUserDialogProps extends ModalProps {
  onSuccess?: () => void;
}

export function InviteOrganizationUserDialog({
  modalOpen,
  setModalOpen,
  onSuccess,
}: InviteOrganizationUserDialogProps) {
  const { authId, organizationId, token } = useCurrentCredential();
  const { getProjects } = useContext(AuthContext);
  const availableProjects = getProjects?.();
  const { loading, showLoader, hideLoader } = useRapidaStore();
  const [email, setEmail] = useState('');
  const [organizationRoles, setOrganizationRoles] = useState<RoleDefinition[]>(
    [],
  );
  const [projectRoles, setProjectRoles] = useState<RoleDefinition[]>([]);
  const [organizationRoleDefinitionIds, setOrganizationRoleDefinitionIds] =
    useState<string[]>([]);
  const [projectRoleRows, setProjectRoleRows] = useState<ProjectRoleRow[]>([]);
  const [isRoleCatalogLoading, setIsRoleCatalogLoading] = useState(false);
  const [isRoleCatalogReady, setIsRoleCatalogReady] = useState(false);
  const [error, setError] = useState('');

  const resetForm = useCallback(() => {
    setEmail('');
    setOrganizationRoleDefinitionIds([]);
    setProjectRoleRows([]);
    setError('');
  }, []);

  const closeDialog = useCallback(() => {
    resetForm();
    setModalOpen(false);
  }, [resetForm, setModalOpen]);

  useEffect(() => {
    if (!modalOpen) return;

    let isCurrent = true;
    setOrganizationRoles([]);
    setProjectRoles([]);
    setOrganizationRoleDefinitionIds([]);
    setProjectRoleRows([]);
    setIsRoleCatalogReady(false);
    setIsRoleCatalogLoading(true);
    Promise.all(
      ['ORGANIZATION', 'PROJECT'].map(async scope => {
        const roles: RoleDefinition[] = [];
        let page = 1;
        let received = 0;
        let total = 0;
        const expectedScope =
          scope === 'ORGANIZATION'
            ? RoleScope.ROLE_SCOPE_ORGANIZATION
            : RoleScope.ROLE_SCOPE_PROJECT;

        do {
          const response = await listAccessRoles({
            page,
            pageSize: 100,
            criteria: [
              { key: 'scope', logic: '=', value: scope },
              { key: 'subject', logic: '=', value: 'USER' },
            ],
            auth: { organizationId, token, userId: authId },
          });
          if (!response.getSuccess()) {
            throw new Error(
              response.getError()?.getHumanmessage() ||
                `${scope === 'ORGANIZATION' ? 'Organization' : 'Project'} roles could not be loaded.`,
            );
          }
          const pageRoles = response.getRolesList();
          received += pageRoles.length;
          roles.push(
            ...pageRoles.filter(
              role =>
                role.getScope() === expectedScope &&
                role.getSubject() === RoleSubject.ROLE_SUBJECT_USER &&
                (scope !== 'ORGANIZATION' || role.getId() !== '101'),
            ),
          );
          total = response.getPaginated()?.getTotalitem() || received;
          if (pageRoles.length === 0 && received < total) {
            throw new Error('The role catalog returned an incomplete result.');
          }
          page += 1;
        } while (received < total);

        return roles;
      }),
    )
      .then(([loadedOrganizationRoles, loadedProjectRoles]) => {
        if (!isCurrent) return;
        setOrganizationRoles(loadedOrganizationRoles);
        setProjectRoles(loadedProjectRoles);
        setIsRoleCatalogReady(true);
        setIsRoleCatalogLoading(false);
      })
      .catch((requestError: ServiceError) => {
        if (isCurrent) {
          setIsRoleCatalogLoading(false);
          toast.error(requestError.message || 'Roles could not be loaded.');
        }
      });

    return () => {
      isCurrent = false;
    };
  }, [authId, modalOpen, token]);

  const submitInvite = async () => {
    if (!email) {
      setError('Please provide a valid email to invite user.');
      return;
    }
    if (organizationRoleDefinitionIds.length === 0) {
      setError('Select at least one organization role.');
      return;
    }
    if (
      projectRoleRows.some(
        row => !row.projectId || row.roleDefinitionIds.length === 0,
      )
    ) {
      setError('Select a project and at least one role for each row.');
      return;
    }
    const selectedProjectIds = projectRoleRows
      .filter(row => row.projectId)
      .map(row => row.projectId);
    if (new Set(selectedProjectIds).size !== selectedProjectIds.length) {
      setError('A project can only be assigned once.');
      return;
    }

    setError('');
    showLoader('overlay');

    try {
      const response = await inviteOrganizationUser({
        email,
        organizationRoleDefinitionIds,
        projectRoles: projectRoleRows.filter(
          row => row.projectId && row.roleDefinitionIds.length > 0,
        ),
        auth: { organizationId, token, userId: authId },
      });
      hideLoader();

      const responseError = response.getError();
      const message = responseError?.getHumanmessage() || PROJECT_ACTION_ERROR;

      if (response.getSuccess()) {
        closeDialog();
        toast.success('The organization invitation was sent successfully.');
        onSuccess?.();
        return;
      }

      toast.error(message);
    } catch (err: unknown) {
      hideLoader();
      const inviteError = err as OrganizationInviteFailure;
      const message =
        inviteError.getHumanmessage?.() ||
        inviteError.message ||
        PROJECT_ACTION_ERROR;
      toast.error(message);
    }
  };

  const projectOptions = useMemo(
    () =>
      (availableProjects || []).map(project => ({
        name: project.name,
        value: project.id,
      })),
    [availableProjects],
  );

  return (
    <Modal
      open={modalOpen}
      onClose={closeDialog}
      size="md"
      preventCloseOnClickOutside
    >
      <ModalHeader
        label="User Management"
        labelClassName={typographyClassNames['helper-text-01']}
        title="Invite User to Organization"
        titleClassName={typographyClassNames['heading-03']}
        onClose={closeDialog}
      />
      <ModalBody hasForm hasScrollingContent>
        <Stack gap={6}>
          <TextInput
            id="organization-invite-email"
            labelText="Email Address"
            value={email}
            type="email"
            placeholder="eg: john@deo.io"
            onChange={e => {
              setError('');
              setEmail(e.target.value);
            }}
          />
          <MultiSelect
            id="organization-invite-role"
            autoAlign
            titleText="Organization Roles"
            label="Select roles"
            items={organizationRoles}
            clearSelectionDescription="Total items selected: "
            clearSelectionText="To clear selection, press Delete or Backspace."
            disabled={!isRoleCatalogReady}
            selectedItems={organizationRoles.filter(role =>
              organizationRoleDefinitionIds.includes(role.getId()),
            )}
            selectionFeedback="top-after-reopen"
            itemToString={(item: RoleDefinition) => item?.getName() || ''}
            onChange={({ selectedItems }) => {
              setError('');
              setOrganizationRoleDefinitionIds(
                (selectedItems || []).map(role => role.getId()),
              );
            }}
            size="md"
            type="default"
          />
          <div>
            <div className="mb-3 flex items-center gap-1">
              <Text as="h3" variant="heading-compact-01">
                Project Access
              </Text>
              <HelpToggletip
                label="Project access"
                helpText="Optional. Assign project roles now, or invite the user with organization access only."
              />
            </div>
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
              defaultProjectId={projectOptions[0]?.value || ''}
              addButtonLabel="Add project access"
            />
          </div>
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
