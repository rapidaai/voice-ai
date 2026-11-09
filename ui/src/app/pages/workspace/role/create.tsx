import { RefObject, useEffect, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Information } from '@carbon/icons-react';
import {
  PermissionDefinition,
  RoleScope,
  RoleSubject,
  ServiceError,
} from '@rapidaai/react';
import {
  InlineLoading,
  ProgressIndicator,
  ProgressStep,
  Search,
  Toggletip,
  ToggletipButton,
  ToggletipContent,
} from '@carbon/react';
import {
  NoDataEmptyState,
  NotFoundEmptyState,
  preview__AddSelect as AddSelect,
  preview__Tearsheet as Tearsheet,
} from '@carbon/ibm-products';
import { Helmet } from '@/app/components/app-shell/helmet';
import { ActionNotification, Notification } from '@/app/components/ui/feedback';
import {
  FieldSet,
  FormLabel,
  Input,
  Select,
  Text,
  Textarea,
} from '@/app/components/ui/primitives';
import { createAccessRole, listAccessPermissions } from '@/clients';
import { useCredential } from '@/hooks/use-credential';

const roleKeyPattern = /^[a-z][a-z0-9_]{2,63}$/;

interface CreateRoleModalProps {
  launcherButtonRef?: RefObject<HTMLButtonElement | null>;
  open: boolean;
  onClose: () => void;
}

export function CreateRoleModal({
  launcherButtonRef,
  open,
  onClose,
}: CreateRoleModalProps) {
  const navigate = useNavigate();
  const [userId, token] = useCredential();
  const [key, setKey] = useState('');
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [scope, setScope] = useState<keyof typeof RoleScope>(
    'ROLE_SCOPE_ORGANIZATION',
  );
  const [subject, setSubject] =
    useState<keyof typeof RoleSubject>('ROLE_SUBJECT_USER');
  const [permissions, setPermissions] = useState<PermissionDefinition[]>([]);
  const [selectedPermissionIds, setSelectedPermissionIds] = useState<string[]>(
    [],
  );
  const [isLoadingPermissions, setIsLoadingPermissions] = useState(true);
  const [hasLoadedPermissionCatalog, setHasLoadedPermissionCatalog] =
    useState(false);
  const [permissionError, setPermissionError] = useState('');
  const [permissionLoadCycle, setPermissionLoadCycle] = useState(0);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [detailsError, setDetailsError] = useState('');
  const [createError, setCreateError] = useState('');
  const [currentStep, setCurrentStep] = useState(0);
  const [permissionSearch, setPermissionSearch] = useState('');
  const permissionRequestSequence = useRef(0);
  const isMounted = useRef(true);

  useEffect(() => {
    isMounted.current = true;
    return () => {
      isMounted.current = false;
      permissionRequestSequence.current += 1;
    };
  }, []);

  useEffect(() => {
    if (open) return;
    setKey('');
    setName('');
    setDescription('');
    setScope('ROLE_SCOPE_ORGANIZATION');
    setSubject('ROLE_SUBJECT_USER');
    setPermissions([]);
    setSelectedPermissionIds([]);
    setIsLoadingPermissions(true);
    setHasLoadedPermissionCatalog(false);
    setPermissionError('');
    setPermissionLoadCycle(0);
    setDetailsError('');
    setCreateError('');
    setCurrentStep(0);
    setPermissionSearch('');
  }, [open]);

  useEffect(() => {
    if (!open) return;
    const requestId = ++permissionRequestSequence.current;
    setSelectedPermissionIds([]);
    setPermissions([]);
    setIsLoadingPermissions(true);
    setHasLoadedPermissionCatalog(false);
    setPermissionError('');
    setPermissionSearch('');

    const loadPermissions = async () => {
      const loaded: PermissionDefinition[] = [];
      let page = 1;
      let total = 0;
      do {
        const response = await listAccessPermissions({
          page,
          pageSize: 100,
          scope,
          auth: { token, userId },
        });
        if (requestId !== permissionRequestSequence.current) return;
        if (!response.getSuccess()) {
          throw new Error(
            response.getError()?.getHumanmessage() ||
              'Unable to load permissions.',
          );
        }
        loaded.push(...response.getPermissionsList());
        total = response.getPaginated()?.getTotalitem() || 0;
        page += 1;
      } while (loaded.length < total);
      if (requestId === permissionRequestSequence.current) {
        setPermissions(loaded);
        setHasLoadedPermissionCatalog(true);
      }
    };

    loadPermissions()
      .catch((requestError: ServiceError | Error) => {
        if (requestId === permissionRequestSequence.current) {
          setPermissionError(
            requestError.message || 'Unable to load permissions.',
          );
        }
      })
      .finally(() => {
        if (requestId === permissionRequestSequence.current) {
          setIsLoadingPermissions(false);
        }
      });

    return () => {
      permissionRequestSequence.current += 1;
    };
  }, [open, permissionLoadCycle, scope, subject, token, userId]);

  const close = () => {
    if (isSubmitting) return;
    onClose();
  };

  const validateDetails = async () => {
    setDetailsError('');
    if (!roleKeyPattern.test(key)) {
      const message =
        'Key must start with a lowercase letter and contain 3 to 64 lowercase letters, numbers, or underscores.';
      setDetailsError(message);
      throw new Error(message);
    }
    if (!name.trim()) {
      const message = 'Role name is required.';
      setDetailsError(message);
      throw new Error(message);
    }
  };

  const submit = async () => {
    if (isSubmitting) {
      throw new Error('Role creation is already in progress.');
    }
    setCreateError('');
    if (!hasLoadedPermissionCatalog || permissionError) {
      const message =
        'Load the complete permission catalog before creating a role.';
      setCreateError(message);
      throw new Error(message);
    }
    setIsSubmitting(true);
    try {
      const response = await createAccessRole({
        key: key.trim(),
        name: name.trim(),
        description: description.trim(),
        scope,
        subject,
        permissionDefinitionIds: selectedPermissionIds,
        auth: { token, userId },
      });
      if (!isMounted.current) return { preventClose: true as const };
      if (!response.getSuccess() || !response.getRole()) {
        const message =
          response.getError()?.getHumanmessage() || 'Unable to create role.';
        setCreateError(message);
        throw new Error(message);
      }
      navigate('/organization/roles', {
        replace: true,
        state: { createdRoleDefinitionId: response.getRole()!.getId() },
      });
      return { preventClose: true as const };
    } catch (requestError) {
      if (!isMounted.current) return { preventClose: true as const };
      const message =
        (requestError as ServiceError).message || 'Unable to create role.';
      setCreateError(message);
      throw requestError instanceof Error ? requestError : new Error(message);
    } finally {
      if (isMounted.current) setIsSubmitting(false);
    }
  };

  const continueFlow = async () => {
    try {
      await validateDetails();
      setCurrentStep(1);
    } catch {
      return;
    }
  };

  const submitRole = () => {
    void submit().catch(() => undefined);
  };

  const selectedPermissionSet = new Set(selectedPermissionIds);
  const selectedPermissions = permissions.filter(permission =>
    selectedPermissionSet.has(permission.getId()),
  );
  const permissionSearchTerm = permissionSearch.trim().toLowerCase();
  const visiblePermissions = permissionSearchTerm
    ? permissions.filter(permission =>
        [
          permission.getName(),
          permission.getKey(),
          permission.getDescription() || '',
        ].some(value => value.toLowerCase().includes(permissionSearchTerm)),
      )
    : permissions;

  const updatePermissionSelection = (
    permissionId: string,
    selected: boolean,
  ) => {
    setSelectedPermissionIds(current =>
      selected
        ? Array.from(new Set([...current, permissionId]))
        : current.filter(id => id !== permissionId),
    );
  };

  const footerNotification =
    currentStep === 0 && detailsError ? (
      <Notification kind="error" title="Error" subtitle={detailsError} />
    ) : currentStep === 1 && permissionError ? (
      <ActionNotification
        actionButtonLabel="Retry permissions"
        kind="error"
        title="Error"
        subtitle={permissionError}
        onActionButtonClick={() => {
          setCreateError('');
          setPermissionLoadCycle(value => value + 1);
        }}
      />
    ) : currentStep === 1 && createError ? (
      <Notification kind="error" title="Error" subtitle={createError} />
    ) : null;

  return (
    <>
      {open && <Helmet title="Create Role" />}
      <Tearsheet
        className="role-create-tearsheet"
        keepMounted
        launcherButtonRef={launcherButtonRef}
        onClose={close}
        open={open}
        selectorPrimaryFocus="#role-name"
        variant="wide"
      >
        <Tearsheet.Header closeIconDescription="Close create role">
          <Tearsheet.HeaderContent
            description="Create a custom role for your organization and choose the permissions it grants."
            label="Roles & Permissions"
            title="Create Role"
          />
        </Tearsheet.Header>
        <Tearsheet.Influencer
          className="py-6"
          influencerPanelAriaLabel="Create role steps"
        >
          <ProgressIndicator
            aria-label="Create role steps"
            currentIndex={currentStep}
            vertical
          >
            <ProgressStep
              complete={currentStep > 0}
              current={currentStep === 0}
              invalid={Boolean(detailsError)}
              label="Role Details"
              secondaryLabel="Define access type and scope"
            />
            <ProgressStep
              current={currentStep === 1}
              invalid={Boolean(createError || permissionError)}
              label="Permissions"
              secondaryLabel="Choose granted access"
            />
          </ProgressIndicator>
        </Tearsheet.Influencer>
        <Tearsheet.Body>
          <Tearsheet.MainContent isFlush={currentStep === 1}>
            {currentStep === 0 ? (
              <section className="py-8">
                <Text as="h2" variant="heading-03">
                  Role Details
                </Text>
                <Text className="mt-2 text-muted" variant="body-compact-01">
                  Define where this role applies and who can receive it.
                </Text>
                <div className="mt-8">
                  <section className="grid gap-6 md:grid-cols-2">
                    <FieldSet>
                      <FormLabel htmlFor="role-name">Role name</FormLabel>
                      <Input
                        id="role-name"
                        value={name}
                        disabled={isSubmitting}
                        placeholder="Enter role name"
                        onChange={event => {
                          setName(event.target.value);
                          setDetailsError('');
                        }}
                      />
                    </FieldSet>
                    <FieldSet>
                      <div className="flex items-center gap-1">
                        <FormLabel htmlFor="role-key">Role key</FormLabel>
                        <Toggletip align="right">
                          <ToggletipButton label="Role key information">
                            <Information size={14} />
                          </ToggletipButton>
                          <ToggletipContent>
                            Lowercase letters, numbers, and underscores. The key
                            cannot be changed later.
                          </ToggletipContent>
                        </Toggletip>
                      </div>
                      <Input
                        id="role-key"
                        value={key}
                        disabled={isSubmitting}
                        placeholder="Enter role key"
                        onChange={event => {
                          setKey(event.target.value);
                          setDetailsError('');
                        }}
                      />
                    </FieldSet>
                    <FieldSet>
                      <FormLabel htmlFor="role-scope">Scope</FormLabel>
                      <Select
                        id="role-scope"
                        value={scope}
                        disabled={
                          isSubmitting ||
                          subject === 'ROLE_SUBJECT_PROJECT_TOKEN'
                        }
                        options={[
                          {
                            name: 'Organization',
                            value: 'ROLE_SCOPE_ORGANIZATION',
                          },
                          { name: 'Project', value: 'ROLE_SCOPE_PROJECT' },
                        ]}
                        onChange={event =>
                          setScope(event.target.value as keyof typeof RoleScope)
                        }
                      />
                    </FieldSet>
                    <FieldSet>
                      <div className="flex items-center gap-1">
                        <FormLabel htmlFor="role-subject">
                          Access type
                        </FormLabel>
                        <Toggletip align="right">
                          <ToggletipButton label="Access type information">
                            <Information size={14} />
                          </ToggletipButton>
                          <ToggletipContent>
                            <p>
                              <strong>Console access:</strong> For signed-in
                              users at organization or project scope.
                            </p>
                            <p className="mt-2">
                              <strong>API access:</strong> For API integrations
                              using project access tokens. Project scope only.
                            </p>
                          </ToggletipContent>
                        </Toggletip>
                      </div>
                      <Select
                        id="role-subject"
                        value={subject}
                        disabled={isSubmitting}
                        options={[
                          {
                            name: 'Console access (user)',
                            value: 'ROLE_SUBJECT_USER',
                          },
                          {
                            name: 'API access (project token)',
                            value: 'ROLE_SUBJECT_PROJECT_TOKEN',
                          },
                        ]}
                        onChange={event => {
                          const nextSubject = event.target
                            .value as keyof typeof RoleSubject;
                          setSubject(nextSubject);
                          if (nextSubject === 'ROLE_SUBJECT_PROJECT_TOKEN') {
                            setScope('ROLE_SCOPE_PROJECT');
                          }
                        }}
                      />
                    </FieldSet>
                    <FieldSet className="md:col-span-2">
                      <FormLabel htmlFor="role-description">
                        Description
                      </FormLabel>
                      <Textarea
                        id="role-description"
                        value={description}
                        disabled={isSubmitting}
                        placeholder="Enter role description"
                        rows={3}
                        onChange={event => setDescription(event.target.value)}
                      />
                    </FieldSet>
                  </section>
                </div>
              </section>
            ) : (
              <section className="h-full min-h-0">
                <div className="h-full min-h-[24rem]">
                  {isLoadingPermissions ? (
                    <InlineLoading
                      description="Loading permissions"
                      className="p-4"
                    />
                  ) : permissionError ? null : permissions.length === 0 ? (
                    <p className="border border-border-subtle bg-layer p-4 text-sm text-muted">
                      No permissions are available for this scope.
                    </p>
                  ) : (
                    <AddSelect
                      className="h-full min-h-0 overflow-hidden"
                      selectedItems={selectedPermissionSet}
                      onItemSelect={updatePermissionSelection}
                    >
                      <div className="grid h-full min-h-0 grid-cols-1 lg:grid-cols-[minmax(0,1fr)_20rem]">
                        <div className="flex min-h-0 flex-col">
                          <div className="border-b border-border-subtle p-4">
                            <Search
                              id="role-permission-search"
                              labelText="Search permissions"
                              placeholder="Search permissions"
                              size="lg"
                              value={permissionSearch}
                              onChange={event =>
                                setPermissionSearch(event.target.value)
                              }
                            />
                          </div>
                          {visiblePermissions.length > 0 ? (
                            <AddSelect.Column
                              {...{
                                'aria-label': 'Permissions',
                                role: 'grid',
                              }}
                              className="min-h-0 border-r-0 p-0!"
                              hideSearch
                              itemCount={visiblePermissions.length}
                              multi
                              title={
                                permissionSearchTerm
                                  ? 'Search Results'
                                  : 'Permissions'
                              }
                            >
                              {visiblePermissions.map(permission => (
                                <AddSelect.Row
                                  disabled={isSubmitting}
                                  itemId={permission.getId()}
                                  key={permission.getId()}
                                  subtitle={
                                    permission.getDescription() ||
                                    'No permission description.'
                                  }
                                  title={permission.getName()}
                                  value={permission.getId()}
                                />
                              ))}
                            </AddSelect.Column>
                          ) : (
                            <NotFoundEmptyState
                              className="p-5"
                              size="sm"
                              subtitle="Try a different search."
                              title="No Permissions Found"
                            />
                          )}
                        </div>
                        <AddSelect.SelectionSummary
                          className="min-h-0 overflow-y-auto border-t border-border-subtle lg:border-l lg:border-t-0"
                          emptyState={
                            <NoDataEmptyState
                              className="p-5"
                              size="sm"
                              subtitle="Select one or more permissions to add to this role."
                              title="No Permissions Selected"
                            />
                          }
                          selectedItemCount={selectedPermissions.length}
                          title="Selected Permissions"
                        >
                          {selectedPermissions.map(permission => (
                            <AddSelect.SelectionSummaryItem
                              item={{
                                id: permission.getId(),
                                subtitle:
                                  permission.getDescription() ||
                                  'No permission description.',
                                title: permission.getName(),
                                value: permission.getId(),
                              }}
                              key={permission.getId()}
                              removeButtonLabel={`Remove ${permission.getName()}`}
                              onRemove={permissionId =>
                                updatePermissionSelection(permissionId, false)
                              }
                            />
                          ))}
                        </AddSelect.SelectionSummary>
                      </div>
                    </AddSelect>
                  )}
                </div>
              </section>
            )}
          </Tearsheet.MainContent>
        </Tearsheet.Body>
        <Tearsheet.Footer
          actions={[
            {
              key: 'cancel',
              kind: 'ghost',
              label: 'Cancel',
              type: 'button',
              disabled: isSubmitting,
              onClick: close,
            },
            {
              key: 'back',
              kind: 'secondary',
              label: 'Back',
              type: 'button',
              disabled: currentStep === 0 || isSubmitting,
              onClick: () => {
                setCreateError('');
                setCurrentStep(0);
              },
            },
            {
              key: currentStep === 0 ? 'next' : 'submit',
              kind: 'primary',
              label: currentStep === 0 ? 'Next' : 'Create role',
              type: 'button',
              disabled:
                isSubmitting ||
                (currentStep === 1 &&
                  (isLoadingPermissions ||
                    !hasLoadedPermissionCatalog ||
                    Boolean(permissionError))),
              onClick: currentStep === 0 ? continueFlow : submitRole,
            },
          ]}
        >
          {footerNotification}
        </Tearsheet.Footer>
      </Tearsheet>
    </>
  );
}
