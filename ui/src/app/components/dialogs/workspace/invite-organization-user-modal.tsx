import { useCallback, useContext, useMemo, useState } from 'react';
import { DefinitionTooltip, Dropdown } from '@carbon/react';
import toast from 'react-hot-toast/headless';
import {
  Modal,
  ModalBody,
  ModalFooter,
  ModalHeader,
  type ModalProps,
  PrimaryButton,
  SecondaryButton,
  Stack,
  TextInput,
  typographyClassNames,
} from '@/app/components/ui/primitives';
import { ErrorMessage } from '@/app/components/ui/feedback';
import { AuthContext } from '@/context/auth-context';
import { useCurrentCredential } from '@/hooks/use-credential';
import { useRapidaStore } from '@/stores/app';
import {
  ProjectRoleRow,
  ProjectRoleTable,
} from '@/app/components/domain/project-role-table';
import { inviteOrganizationUser } from '@/clients';

const PROJECT_ACTION_ERROR =
  'Unable to process your request. please try again later.';

type OrganizationInviteFailure = {
  getHumanmessage?: () => string;
  message?: string;
};

const organizationRoles = [
  { name: 'Admin', value: 'admin' },
  { name: 'Member', value: 'member' },
];
const projectRoles = [
  { name: 'Super Admin', value: 'super admin' },
  { name: 'Admin', value: 'admin' },
  { name: 'Writer', value: 'writer' },
  { name: 'Reader', value: 'reader' },
];

interface InviteOrganizationUserDialogProps extends ModalProps {
  onSuccess?: () => void;
}

export function InviteOrganizationUserDialog({
  modalOpen,
  setModalOpen,
  onSuccess,
}: InviteOrganizationUserDialogProps) {
  const { authId, token } = useCurrentCredential();
  const { projectRoles: availableProjects } = useContext(AuthContext);
  const { loading, showLoader, hideLoader } = useRapidaStore();
  const [email, setEmail] = useState('');
  const [organizationRole, setOrganizationRole] = useState('');
  const [projectRoleRows, setProjectRoleRows] = useState<ProjectRoleRow[]>([]);
  const [error, setError] = useState('');

  const closeDialog = useCallback(() => {
    setModalOpen(false);
  }, [setModalOpen]);

  const resetForm = useCallback(() => {
    setEmail('');
    setOrganizationRole('');
    setProjectRoleRows([]);
    setError('');
  }, []);

  const submitInvite = async () => {
    if (!email) {
      setError('Please provide a valid email to invite user.');
      return;
    }
    if (!organizationRole) {
      setError('Please select an organization role.');
      return;
    }
    if (projectRoleRows.some(row => !row.projectId || !row.projectRole)) {
      setError('Select both project and role for each project role row.');
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
        organizationRole,
        projectRoles: projectRoleRows.filter(
          row => row.projectId && row.projectRole,
        ),
        auth: { token, userId: authId },
      });
      hideLoader();

      const responseError = response.getError();
      const message = responseError?.getHumanmessage() || PROJECT_ACTION_ERROR;

      if (response.getSuccess()) {
        resetForm();
        closeDialog();
        toast.success('The organization invitation was sent successfully.');
        onSuccess?.();
        return;
      }

      toast.error(message);
      setError(message);
    } catch (err: unknown) {
      hideLoader();
      const inviteError = err as OrganizationInviteFailure;
      const message =
        inviteError.getHumanmessage?.() ||
        inviteError.message ||
        PROJECT_ACTION_ERROR;
      toast.error(message);
      setError(message);
    }
  };

  const projectOptions = useMemo(
    () =>
      (availableProjects || []).map(project => ({
        name: project.projectname,
        value: project.projectid,
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
        label="User management"
        labelClassName={typographyClassNames.helper}
        title="Invite user to organization"
        titleClassName={typographyClassNames.sectionTitle}
        onClose={closeDialog}
      />
      <ModalBody hasForm hasScrollingContent>
        <Stack gap={6}>
          <TextInput
            id="organization-invite-email"
            labelText="Email address"
            value={email}
            type="email"
            placeholder="eg: john@deo.io"
            onChange={e => {
              setError('');
              setEmail(e.target.value);
            }}
          />
          <Dropdown
            id="organization-invite-role"
            titleText="Organization role"
            label="Select organization role"
            items={organizationRoles}
            selectedItem={
              organizationRoles.find(role => role.value === organizationRole) ||
              null
            }
            itemToString={(item: { name: string; value: string } | null) =>
              item?.name || ''
            }
            onChange={({ selectedItem }) => {
              setError('');
              setOrganizationRole(selectedItem?.value || '');
            }}
          />
          <div>
            <div className="mb-3">
              <h3 className={typographyClassNames.componentTitle}>
                <DefinitionTooltip
                  align="right"
                  definition="Optional. Assign project roles now, or invite the user with organization access only."
                  openOnHover
                >
                  Project access
                </DefinitionTooltip>
              </h3>
            </div>
            <ProjectRoleTable
              rows={projectRoleRows}
              onChange={next => {
                setError('');
                setProjectRoleRows(next);
              }}
              projectOptions={projectOptions}
              roleOptions={projectRoles}
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
        <PrimaryButton size="lg" onClick={submitInvite} isLoading={loading}>
          Invite user
        </PrimaryButton>
      </ModalFooter>
    </Modal>
  );
}
