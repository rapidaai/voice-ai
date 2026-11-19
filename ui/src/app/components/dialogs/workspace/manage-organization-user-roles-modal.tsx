import { useCallback, useEffect, useState } from 'react';
import {
  RoleScope,
  RoleSubject,
  type RoleDefinition,
  type ServiceError,
  type User,
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
  TextInput,
  typographyClassNames,
} from '@/app/components/ui/primitives';
import { ErrorMessage } from '@/app/components/ui/feedback';
import { listAccessRoles, updateOrganizationUserRole } from '@/clients';
import { useCurrentCredential } from '@/hooks/use-credential';
import { useRapidaStore } from '@/stores/app';

const USER_ROLE_ACTION_ERROR =
  'Unable to update organization roles. Try again.';

interface ManageOrganizationUserRolesDialogProps extends ModalProps {
  user: User | null;
  onSuccess?: () => void;
}

export function ManageOrganizationUserRolesDialog({
  modalOpen,
  setModalOpen,
  user,
  onSuccess,
}: ManageOrganizationUserRolesDialogProps) {
  const { authId, organizationId, token } = useCurrentCredential();
  const { loading, showLoader, hideLoader } = useRapidaStore();
  const [roles, setRoles] = useState<RoleDefinition[]>([]);
  const [roleDefinitionIds, setRoleDefinitionIds] = useState<string[]>([]);
  const [isRoleCatalogLoading, setIsRoleCatalogLoading] = useState(false);
  const [isRoleCatalogReady, setIsRoleCatalogReady] = useState(false);
  const [error, setError] = useState('');

  const closeDialog = useCallback(() => {
    setRoles([]);
    setRoleDefinitionIds([]);
    setIsRoleCatalogReady(false);
    setError('');
    setModalOpen(false);
  }, [setModalOpen]);

  useEffect(() => {
    if (!modalOpen || !user) return;

    let isCurrent = true;
    setRoles([]);
    setRoleDefinitionIds([]);
    setIsRoleCatalogReady(false);
    setIsRoleCatalogLoading(true);
    setError('');

    (async () => {
      const loadedRoles: RoleDefinition[] = [];
      let page = 1;
      let received = 0;
      let total = 0;

      do {
        const response = await listAccessRoles({
          page,
          pageSize: 100,
          criteria: [
            { key: 'scope', logic: '=', value: 'ORGANIZATION' },
            { key: 'subject', logic: '=', value: 'USER' },
          ],
          auth: { organizationId, token, userId: authId },
        });
        if (!response.getSuccess()) {
          throw new Error(
            response.getError()?.getHumanmessage() ||
              'Organization roles could not be loaded.',
          );
        }
        const pageRoles = response.getRolesList();
        received += pageRoles.length;
        loadedRoles.push(
          ...pageRoles.filter(
            role =>
              role.getScope() === RoleScope.ROLE_SCOPE_ORGANIZATION &&
              role.getSubject() === RoleSubject.ROLE_SUBJECT_USER &&
              role.getId() !== '101',
          ),
        );
        total = response.getPaginated()?.getTotalitem() || received;
        if (pageRoles.length === 0 && received < total) {
          throw new Error('The role catalog returned an incomplete result.');
        }
        page += 1;
      } while (received < total);

      return loadedRoles;
    })()
      .then(loadedRoles => {
        if (!isCurrent) return;
        const availableRoleIds = new Set(loadedRoles.map(role => role.getId()));
        setRoles(loadedRoles);
        setRoleDefinitionIds(
          user
            .getOrganizationrolesList()
            .map(role => role.getRoledefinitionid())
            .filter(roleDefinitionId => availableRoleIds.has(roleDefinitionId)),
        );
        setIsRoleCatalogReady(true);
        setIsRoleCatalogLoading(false);
      })
      .catch((requestError: ServiceError) => {
        if (isCurrent) {
          setIsRoleCatalogLoading(false);
          toast.error(
            requestError.message || 'Organization roles could not be loaded.',
          );
        }
      });

    return () => {
      isCurrent = false;
    };
  }, [authId, modalOpen, token, user]);

  const saveRoles = async () => {
    if (!user) return;
    if (roleDefinitionIds.length === 0) {
      setError('Select at least one organization role.');
      return;
    }

    setError('');
    showLoader('overlay');
    try {
      const response = await updateOrganizationUserRole({
        userId: user.getId(),
        roleDefinitionIds,
        auth: { organizationId, token, userId: authId },
      });
      hideLoader();

      if (!response.getSuccess()) {
        toast.error(
          response.getError()?.getHumanmessage() || USER_ROLE_ACTION_ERROR,
        );
        return;
      }

      closeDialog();
      toast.success('Organization roles updated.');
      onSuccess?.();
    } catch (requestError: unknown) {
      hideLoader();
      toast.error(
        requestError instanceof Error && requestError.message
          ? requestError.message
          : USER_ROLE_ACTION_ERROR,
      );
    }
  };

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
        title="Manage Organization Roles"
        titleClassName={typographyClassNames['heading-03']}
        onClose={closeDialog}
      />
      <ModalBody hasForm hasScrollingContent>
        <Stack gap={6}>
          <TextInput
            id="organization-role-user"
            labelText="User"
            value={user ? `${user.getName()} (${user.getEmail()})` : ''}
            readOnly
          />
          <MultiSelect
            id="organization-role-selection"
            autoAlign
            titleText="Organization Roles"
            label="Select roles"
            items={roles}
            clearSelectionDescription="Total items selected: "
            clearSelectionText="To clear selection, press Delete or Backspace."
            disabled={!isRoleCatalogReady}
            selectedItems={roles.filter(role =>
              roleDefinitionIds.includes(role.getId()),
            )}
            selectionFeedback="top-after-reopen"
            itemToString={(item: RoleDefinition) => item?.getName() || ''}
            onChange={({ selectedItems }) => {
              setError('');
              setRoleDefinitionIds(
                (selectedItems || []).map(role => role.getId()),
              );
            }}
            size="md"
            type="default"
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
          onClick={saveRoles}
          disabled={!isRoleCatalogReady}
          isLoading={loading || isRoleCatalogLoading}
        >
          Save Roles
        </PrimaryButton>
      </ModalFooter>
    </Modal>
  );
}
