import { FormEvent, useCallback, useEffect, useRef, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import {
  RoleDefinition,
  RoleDefinitionType,
  RoleScope,
  RoleSubject,
  ServiceError,
} from '@rapidaai/react';
import { Button, InlineLoading } from '@carbon/react';
import { Renew, Security } from '@carbon/icons-react';
import { Helmet } from '@/app/components/app-shell/helmet';
import { PageHeader } from '@/app/components/layout/page-header';
import { EmptyState } from '@/app/components/ui/feedback';
import {
  FieldSet,
  FormLabel,
  Input,
  InputCheckbox,
  Text,
  Textarea,
} from '@/app/components/ui/primitives';
import {
  archiveAccessRole,
  getAccessRole,
  reduceAccessRolePermissions,
  updateAccessRole,
} from '@/clients';
import { useCredential } from '@/hooks/use-credential';
import { toHumanReadableDateTime } from '@/utils/date';

export function RoleDetailPage() {
  const navigate = useNavigate();
  const { roleDefinitionId = '' } = useParams();
  const [userId, token] = useCredential();
  const [role, setRole] = useState<RoleDefinition | null>(null);
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [selectedPermissionIds, setSelectedPermissionIds] = useState<string[]>(
    [],
  );
  const [reductionReason, setReductionReason] = useState('');
  const [archiveReason, setArchiveReason] = useState('');
  const [isLoading, setIsLoading] = useState(true);
  const [loadError, setLoadError] = useState('');
  const [metadataError, setMetadataError] = useState('');
  const [reductionError, setReductionError] = useState('');
  const [archiveError, setArchiveError] = useState('');
  const [activeAction, setActiveAction] = useState('');
  const roleRequestSequence = useRef(0);
  const isMounted = useRef(true);

  const loadRole = useCallback(async () => {
    const requestId = ++roleRequestSequence.current;
    setIsLoading(true);
    setLoadError('');
    try {
      const response = await getAccessRole({
        roleDefinitionId,
        auth: { token, userId },
      });
      if (requestId !== roleRequestSequence.current) return;
      if (!response.getSuccess() || !response.getRole()) {
        setLoadError(
          response.getError()?.getHumanmessage() || 'Unable to load role.',
        );
        return;
      }
      const nextRole = response.getRole()!;
      setRole(nextRole);
      setName(nextRole.getName());
      setDescription(nextRole.getDescription());
      setSelectedPermissionIds([]);
    } catch (requestError) {
      if (requestId !== roleRequestSequence.current) return;
      const serviceError = requestError as ServiceError;
      setLoadError(
        serviceError.code === 7
          ? 'You do not have permission to view this role.'
          : serviceError.message || 'Unable to load role.',
      );
    } finally {
      if (requestId === roleRequestSequence.current) setIsLoading(false);
    }
  }, [roleDefinitionId, token, userId]);

  useEffect(() => {
    isMounted.current = true;
    loadRole();
    return () => {
      isMounted.current = false;
      roleRequestSequence.current += 1;
    };
  }, [loadRole]);

  const updateMetadata = async (event: FormEvent) => {
    event.preventDefault();
    setMetadataError('');
    if (!name.trim()) {
      setMetadataError('Role name is required.');
      return;
    }
    setActiveAction('metadata');
    try {
      const response = await updateAccessRole({
        roleDefinitionId,
        name: name.trim(),
        description: description.trim(),
        auth: { token, userId },
      });
      if (!isMounted.current) return;
      if (!response.getSuccess()) {
        setMetadataError(
          response.getError()?.getHumanmessage() || 'Unable to update role.',
        );
        return;
      }
      await loadRole();
    } catch (requestError) {
      if (!isMounted.current) return;
      setMetadataError(
        (requestError as ServiceError).message || 'Unable to update role.',
      );
    } finally {
      if (isMounted.current) setActiveAction('');
    }
  };

  const removePermissions = async (event: FormEvent) => {
    event.preventDefault();
    setReductionError('');
    if (selectedPermissionIds.length === 0) {
      setReductionError('Select at least one permission to remove.');
      return;
    }
    if (!reductionReason.trim()) {
      setReductionError('A reason is required.');
      return;
    }
    setActiveAction('permissions');
    try {
      const response = await reduceAccessRolePermissions({
        roleDefinitionId,
        permissionDefinitionIds: selectedPermissionIds,
        reason: reductionReason.trim(),
        auth: { token, userId },
      });
      if (!isMounted.current) return;
      if (!response.getSuccess()) {
        setReductionError(
          response.getError()?.getHumanmessage() ||
            'Unable to remove permissions.',
        );
        return;
      }
      setReductionReason('');
      await loadRole();
    } catch (requestError) {
      if (!isMounted.current) return;
      setReductionError(
        (requestError as ServiceError).message ||
          'Unable to remove permissions.',
      );
    } finally {
      if (isMounted.current) setActiveAction('');
    }
  };

  const archiveRole = async (event: FormEvent) => {
    event.preventDefault();
    setArchiveError('');
    if (!archiveReason.trim()) {
      setArchiveError('A reason is required.');
      return;
    }
    setActiveAction('archive');
    try {
      const response = await archiveAccessRole({
        roleDefinitionId,
        reason: archiveReason.trim(),
        auth: { token, userId },
      });
      if (!isMounted.current) return;
      if (!response.getSuccess()) {
        setArchiveError(
          response.getError()?.getHumanmessage() || 'Unable to archive role.',
        );
        return;
      }
      navigate('/organization/roles');
    } catch (requestError) {
      if (!isMounted.current) return;
      setArchiveError(
        (requestError as ServiceError).message || 'Unable to archive role.',
      );
    } finally {
      if (isMounted.current) setActiveAction('');
    }
  };

  if (isLoading && !role) {
    return <InlineLoading description="Loading role" className="m-8" />;
  }

  if (loadError || !role) {
    return (
      <EmptyState
        icon={Security}
        title={loadError || 'Role not found.'}
        action="Try again"
        actionIcon={Renew}
        onAction={loadRole}
      />
    );
  }

  const isBuiltIn =
    role.getDefinitiontype() ===
    RoleDefinitionType.ROLE_DEFINITION_TYPE_BUILT_IN;
  const createdActor = role.getCreatedactor();
  const updatedActor = role.getUpdatedactor();
  const createdBy = createdActor
    ? createdActor.getDisplayname() ||
      `${createdActor.getType()} ${createdActor.getId()}`
    : 'System';
  const updatedBy = updatedActor
    ? updatedActor.getDisplayname() ||
      `${updatedActor.getType()} ${updatedActor.getId()}`
    : 'Not updated';

  return (
    <div className="flex min-h-0 flex-1 flex-col overflow-y-auto bg-surface">
      <Helmet title={role.getName()} />
      <PageHeader
        eyebrow={<Text className="text-primary">Roles & Permissions</Text>}
        title={role.getName()}
        description={role.getDescription() || 'No description provided.'}
        contextualActions={
          <Text className="text-muted">
            {isBuiltIn ? 'Built-in role' : 'Custom role'}
          </Text>
        }
      />
      <div className="mx-auto flex w-full max-w-5xl flex-col gap-8 p-6 md:p-8">
        <section className="grid gap-4 border border-border-subtle bg-layer p-5 md:grid-cols-3">
          <div>
            <Text className="text-muted">Key</Text>
            <p className="mt-1 font-medium">{role.getKey()}</p>
          </div>
          <div>
            <Text className="text-muted">Scope</Text>
            <p className="mt-1 font-medium">
              {role.getScope() === RoleScope.ROLE_SCOPE_ORGANIZATION
                ? 'Organization'
                : 'Project'}
            </p>
          </div>
          <div>
            <Text className="text-muted">Subject</Text>
            <p className="mt-1 font-medium">
              {role.getSubject() === RoleSubject.ROLE_SUBJECT_PROJECT_TOKEN
                ? 'Project token'
                : 'User'}
            </p>
          </div>
          <div>
            <Text className="text-muted">Created by</Text>
            <p className="mt-1 font-medium">{createdBy}</p>
          </div>
          <div>
            <Text className="text-muted">Created</Text>
            <p className="mt-1 font-medium">
              {role.getCreateddate()
                ? toHumanReadableDateTime(role.getCreateddate()!)
                : 'Unknown'}
            </p>
          </div>
          <div>
            <Text className="text-muted">Last updated</Text>
            <p className="mt-1 font-medium">
              {role.getUpdateddate()
                ? `${toHumanReadableDateTime(role.getUpdateddate()!)} by ${updatedBy}`
                : 'Never'}
            </p>
          </div>
        </section>

        <form
          className="border border-border-subtle bg-layer p-5"
          onSubmit={updateMetadata}
        >
          <h2 className="text-base font-semibold">Role details</h2>
          {isBuiltIn && (
            <p className="mt-1 text-sm text-muted">
              Built-in role details cannot be changed.
            </p>
          )}
          {metadataError && (
            <p role="alert" className="mt-3 text-sm text-support-error">
              {metadataError}
            </p>
          )}
          <div className="mt-5 grid gap-5">
            <FieldSet>
              <FormLabel htmlFor="role-name">Name</FormLabel>
              <Input
                id="role-name"
                value={name}
                disabled={isBuiltIn}
                onChange={event => setName(event.target.value)}
              />
            </FieldSet>
            <FieldSet>
              <FormLabel htmlFor="role-description">Description</FormLabel>
              <Textarea
                id="role-description"
                value={description}
                disabled={isBuiltIn}
                onChange={event => setDescription(event.target.value)}
              />
            </FieldSet>
          </div>
          {!isBuiltIn && (
            <div className="mt-5 flex justify-end">
              <Button type="submit" disabled={activeAction !== ''}>
                {activeAction === 'metadata' ? 'Saving...' : 'Save details'}
              </Button>
            </div>
          )}
        </form>

        <form
          className="border border-border-subtle bg-layer"
          onSubmit={removePermissions}
        >
          <div className="border-b border-border-subtle p-5">
            <h2 className="text-base font-semibold">Permissions</h2>
            <p className="mt-1 text-sm text-muted">
              {isBuiltIn
                ? 'Built-in permissions are read-only.'
                : 'Select permissions that should no longer be granted by this role.'}
            </p>
            {reductionError && (
              <p role="alert" className="mt-3 text-sm text-support-error">
                {reductionError}
              </p>
            )}
          </div>
          {role.getPermissionsList().length === 0 ? (
            <p className="p-5 text-sm text-muted">
              This role has no permissions.
            </p>
          ) : (
            <div className="grid gap-px bg-border-subtle md:grid-cols-2">
              {role.getPermissionsList().map(permission => (
                <div
                  key={permission.getId()}
                  className="flex gap-3 bg-surface p-4"
                >
                  {!isBuiltIn && (
                    <InputCheckbox
                      aria-label={`Remove ${permission.getName()}`}
                      checked={selectedPermissionIds.includes(
                        permission.getId(),
                      )}
                      onChange={event =>
                        setSelectedPermissionIds(current =>
                          event.target.checked
                            ? [...current, permission.getId()]
                            : current.filter(id => id !== permission.getId()),
                        )
                      }
                    />
                  )}
                  <span>
                    <span className="block text-sm font-medium">
                      {permission.getName()}
                    </span>
                    <span className="block text-xs text-muted">
                      {permission.getKey()}
                    </span>
                  </span>
                </div>
              ))}
            </div>
          )}
          {!isBuiltIn && role.getPermissionsList().length > 0 && (
            <div className="grid gap-4 border-t border-border-subtle p-5 md:grid-cols-[1fr_auto] md:items-end">
              <FieldSet>
                <FormLabel htmlFor="reduction-reason">
                  Reason for removal
                </FormLabel>
                <Input
                  id="reduction-reason"
                  value={reductionReason}
                  onChange={event => setReductionReason(event.target.value)}
                />
              </FieldSet>
              <Button
                kind="danger--tertiary"
                type="submit"
                disabled={activeAction !== ''}
              >
                {activeAction === 'permissions'
                  ? 'Removing...'
                  : 'Remove selected'}
              </Button>
            </div>
          )}
        </form>

        {!isBuiltIn && (
          <form
            className="border border-support-error p-5"
            onSubmit={archiveRole}
          >
            <h2 className="text-base font-semibold">Archive role</h2>
            <p className="mt-1 text-sm text-muted">
              Archive this role when it is no longer required. This action
              removes it from the active catalog.
            </p>
            {archiveError && (
              <p role="alert" className="mt-3 text-sm text-support-error">
                {archiveError}
              </p>
            )}
            <div className="mt-5 grid gap-4 md:grid-cols-[1fr_auto] md:items-end">
              <FieldSet>
                <FormLabel htmlFor="archive-reason">
                  Reason for archive
                </FormLabel>
                <Input
                  id="archive-reason"
                  value={archiveReason}
                  onChange={event => setArchiveReason(event.target.value)}
                />
              </FieldSet>
              <Button
                kind="danger"
                type="submit"
                disabled={activeAction !== ''}
              >
                {activeAction === 'archive' ? 'Archiving...' : 'Archive role'}
              </Button>
            </div>
          </form>
        )}
      </div>
    </div>
  );
}
