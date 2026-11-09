import { useEffect, useRef, useState } from 'react';
import { Navigate } from 'react-router-dom';
import {
  RoleDefinition,
  RoleDefinitionType,
  ServiceError,
} from '@rapidaai/react';
import { InlineLoading } from '@carbon/react';
import {
  Modal,
  ModalBody,
  ModalFooter,
  ModalHeader,
  PrimaryButton,
  SecondaryButton,
  Stack,
  TextArea,
  TextInput,
  TertiaryButton,
} from '@/app/components/ui/primitives';
import { ConfirmDeleteDialog } from '@/app/components/dialogs/shared';
import { Notification } from '@/app/components/ui/feedback';
import { archiveAccessRole, getAccessRole, updateAccessRole } from '@/clients';
import { useCredential } from '@/hooks/use-credential';
import toast from 'react-hot-toast/headless';

interface RoleActionModalProps {
  open: boolean;
  role: RoleDefinition;
  onClose: () => void;
  onUpdated: () => void;
}

interface DeleteRoleDialogProps {
  open: boolean;
  role: RoleDefinition;
  onClose: () => void;
  onDeleted: () => void;
}

const roleDeletionReason = 'Deleted from Roles & Permissions.';

function isCustomRole(role: RoleDefinition) {
  return (
    role.getDefinitiontype() === RoleDefinitionType.ROLE_DEFINITION_TYPE_CUSTOM
  );
}

export function EditRoleModal({
  open,
  role,
  onClose,
  onUpdated,
}: RoleActionModalProps) {
  const [userId, token] = useCredential();
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [metadataError, setMetadataError] = useState('');
  const [activeAction, setActiveAction] = useState('');
  const [loadedRole, setLoadedRole] = useState<RoleDefinition | null>(null);
  const [loadStatus, setLoadStatus] = useState<'loading' | 'ready' | 'error'>(
    'loading',
  );
  const [loadError, setLoadError] = useState('');
  const [loadCycle, setLoadCycle] = useState(0);
  const requestSequence = useRef(0);
  const nameInputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (!open) {
      requestSequence.current += 1;
      return;
    }
    const requestId = ++requestSequence.current;
    setLoadedRole(null);
    setLoadStatus('loading');
    setLoadError('');
    setName('');
    setDescription('');
    setMetadataError('');
    setActiveAction('');
    getAccessRole({
      roleDefinitionId: role.getId(),
      auth: { token, userId },
    })
      .then(response => {
        if (requestId !== requestSequence.current) return;
        const completeRole = response.getRole();
        if (!response.getSuccess() || !completeRole) {
          setLoadError(
            response.getError()?.getHumanmessage() || 'Unable to load role.',
          );
          setLoadStatus('error');
          return;
        }
        if (!isCustomRole(completeRole)) {
          setLoadError('Built-in roles cannot be edited.');
          setLoadStatus('error');
          return;
        }
        setLoadedRole(completeRole);
        setName(completeRole.getName());
        setDescription(completeRole.getDescription());
        setLoadStatus('ready');
      })
      .catch((requestError: ServiceError) => {
        if (requestId !== requestSequence.current) return;
        setLoadError(requestError.message || 'Unable to load role.');
        setLoadStatus('error');
      });
    return () => {
      requestSequence.current += 1;
    };
  }, [loadCycle, open, role, token, userId]);

  useEffect(() => {
    if (open && loadStatus === 'ready') nameInputRef.current?.focus();
  }, [loadStatus, open]);

  const close = () => {
    if (activeAction) return;
    requestSequence.current += 1;
    onClose();
  };

  const updateMetadata = async () => {
    if (!loadedRole || !isCustomRole(loadedRole)) return;
    setMetadataError('');
    if (!name.trim()) {
      setMetadataError('Role name is required.');
      return;
    }
    const requestId = ++requestSequence.current;
    setActiveAction('metadata');
    try {
      const response = await updateAccessRole({
        roleDefinitionId: loadedRole.getId(),
        name: name.trim(),
        description: description.trim(),
        auth: { token, userId },
      });
      if (requestId !== requestSequence.current) return;
      if (!response.getSuccess()) {
        setMetadataError(
          response.getError()?.getHumanmessage() || 'Unable to update role.',
        );
        return;
      }
      onUpdated();
      requestSequence.current += 1;
      onClose();
    } catch (requestError) {
      if (requestId !== requestSequence.current) return;
      setMetadataError(
        (requestError as ServiceError).message || 'Unable to update role.',
      );
    } finally {
      if (requestId === requestSequence.current) setActiveAction('');
    }
  };

  return (
    <Modal
      aria-label="Edit role"
      open={open}
      onClose={close}
      preventCloseOnClickOutside
      selectorPrimaryFocus="#role-name"
      size="sm"
    >
      <ModalHeader
        label="Roles & Permissions"
        title="Edit role"
        onClose={close}
      />
      <ModalBody hasForm hasScrollingContent>
        {loadStatus === 'loading' && (
          <InlineLoading description="Loading role" />
        )}
        {loadStatus === 'error' && (
          <Stack gap={5}>
            <Notification kind="error" title="Error" subtitle={loadError} />
            <TertiaryButton
              size="md"
              onClick={() => setLoadCycle(value => value + 1)}
            >
              Retry
            </TertiaryButton>
          </Stack>
        )}
        {loadStatus === 'ready' && loadedRole && (
          <Stack gap={6}>
            <TextInput
              ref={nameInputRef}
              id="role-name"
              labelText="Name"
              value={name}
              invalid={Boolean(metadataError)}
              invalidText={metadataError}
              onChange={event => {
                setName(event.target.value);
                setMetadataError('');
              }}
            />
            <TextArea
              id="role-description"
              labelText="Description"
              rows={4}
              value={description}
              onChange={event => {
                setDescription(event.target.value as string);
                setMetadataError('');
              }}
            />
          </Stack>
        )}
      </ModalBody>
      <ModalFooter>
        <SecondaryButton
          size="lg"
          type="button"
          disabled={Boolean(activeAction)}
          onClick={close}
        >
          Cancel
        </SecondaryButton>
        <PrimaryButton
          size="lg"
          type="button"
          disabled={Boolean(activeAction) || loadStatus !== 'ready'}
          isLoading={activeAction === 'metadata'}
          onClick={updateMetadata}
        >
          Save changes
        </PrimaryButton>
      </ModalFooter>
    </Modal>
  );
}

export function DeleteRoleDialog({
  open,
  role,
  onClose,
  onDeleted,
}: DeleteRoleDialogProps) {
  const [userId, token] = useCredential();
  const [isDeleting, setIsDeleting] = useState(false);
  const requestSequence = useRef(0);

  useEffect(() => {
    requestSequence.current += 1;
    if (!open) return;
    setIsDeleting(false);
    return () => {
      requestSequence.current += 1;
    };
  }, [open, role]);

  const close = () => {
    if (isDeleting) return;
    requestSequence.current += 1;
    onClose();
  };

  const deleteRole = async () => {
    if (!isCustomRole(role)) return;
    const requestId = ++requestSequence.current;
    setIsDeleting(true);
    try {
      const response = await archiveAccessRole({
        roleDefinitionId: role.getId(),
        reason: roleDeletionReason,
        auth: { token, userId },
      });
      if (requestId !== requestSequence.current) return;
      if (!response.getSuccess()) {
        toast.error(
          response.getError()?.getHumanmessage() || 'Unable to delete role.',
        );
        return;
      }
      onDeleted();
      requestSequence.current += 1;
      onClose();
    } catch (requestError) {
      if (requestId !== requestSequence.current) return;
      toast.error(
        (requestError as ServiceError).message || 'Unable to delete role.',
      );
    } finally {
      if (requestId === requestSequence.current) setIsDeleting(false);
    }
  };

  return (
    <ConfirmDeleteDialog
      showing={open}
      title="Delete role"
      content="Deleting a role removes it from the active roles. Its key cannot be reused, and roles assigned to users cannot be deleted."
      objectName={role.getName()}
      confirmText={isDeleting ? 'Deleting...' : 'Delete role'}
      onConfirm={deleteRole}
      onCancel={close}
      onClose={close}
    />
  );
}

export function RoleDetailPage() {
  return <Navigate to="/organization/roles" replace />;
}
