import { useEffect, useRef, useState } from 'react';
import type {
  CreateProjectCredentialResponse,
  RoleDefinition,
  ServiceError,
} from '@rapidaai/react';
import { InlineLoading, PasswordInput } from '@carbon/react';
import { CheckmarkFilled, Copy, InformationFilled } from '@carbon/icons-react';
import {
  ComboBox,
  FieldSet,
  FormLabel,
  Input,
  Modal,
  ModalBody,
  ModalFooter,
  ModalHeader,
  PrimaryButton,
  SecondaryButton,
  TertiaryButton,
  Text,
} from '@/app/components/ui/primitives';
import { HelpToggletip } from '@/app/components/ui/composites';
import { createProjectPublishableCredential, listAccessRoles } from '@/clients';

type CredentialAuth = {
  organizationId: string;
  token: string;
  userId: string;
};

interface CreateProjectTokenProps {
  open: boolean;
  projectId: string;
  auth: CredentialAuth;
  onClose: () => void;
  onCreated: () => void;
}

type LoadStatus = 'idle' | 'loading' | 'ready' | 'empty' | 'error';

export function CreateProjectToken({
  open,
  projectId,
  auth,
  onClose,
  onCreated,
}: CreateProjectTokenProps) {
  const [roles, setRoles] = useState<RoleDefinition[]>([]);
  const [applicationName, setApplicationName] = useState('');
  const [selectedRole, setSelectedRole] = useState<RoleDefinition | null>(null);
  const [listStatus, setListStatus] = useState<LoadStatus>('idle');
  const [error, setError] = useState('');
  const [listCycle, setListCycle] = useState(0);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [generatedKey, setGeneratedKey] = useState<string | null>(null);
  const [isCopied, setIsCopied] = useState(false);
  const listSequence = useRef(0);
  const submissionSequence = useRef(0);
  const copySequence = useRef(0);

  useEffect(() => {
    if (!open) {
      listSequence.current += 1;
      submissionSequence.current += 1;
      copySequence.current += 1;
      setRoles([]);
      setApplicationName('');
      setSelectedRole(null);
      setListStatus('idle');
      setError('');
      setIsSubmitting(false);
      setGeneratedKey(null);
      setIsCopied(false);
      return;
    }

    const requestId = ++listSequence.current;
    setRoles([]);
    setApplicationName('');
    setSelectedRole(null);
    setListStatus('loading');
    setError('');
    setIsSubmitting(false);
    setGeneratedKey(null);
    setIsCopied(false);

    const loadRoles = async () => {
      const loadedRoles: RoleDefinition[] = [];
      let page = 1;
      let totalItems = 0;
      do {
        const response = await listAccessRoles({
          page,
          pageSize: 100,
          criteria: [
            { key: 'scope', logic: '=', value: 'PROJECT' },
            { key: 'subject', logic: '=', value: 'PROJECT_TOKEN' },
          ],
          auth,
        });
        if (requestId !== listSequence.current) return;
        if (!response.getSuccess()) {
          throw new Error(
            response.getError()?.getHumanmessage() ||
              'Unable to load project roles.',
          );
        }
        const pageRoles = response.getRolesList();
        loadedRoles.push(...pageRoles);
        totalItems = response.getPaginated()?.getTotalitem() || 0;
        page += 1;
        if (pageRoles.length === 0) break;
      } while (loadedRoles.length < totalItems);

      if (requestId !== listSequence.current) return;
      setRoles(loadedRoles);
      setListStatus(loadedRoles.length === 0 ? 'empty' : 'ready');
    };

    loadRoles().catch((requestError: ServiceError | Error) => {
      if (requestId !== listSequence.current) return;
      setError(requestError.message || 'Unable to load project roles.');
      setListStatus('error');
    });

    return () => {
      listSequence.current += 1;
      submissionSequence.current += 1;
      copySequence.current += 1;
    };
  }, [auth.token, auth.userId, listCycle, open]);

  const submit = () => {
    if (isSubmitting || !applicationName.trim() || !selectedRole) return;
    const requestId = ++submissionSequence.current;
    setIsSubmitting(true);
    setError('');
    createProjectPublishableCredential({
      projectId,
      name: applicationName.trim(),
      roleDefinitionId: selectedRole.getId(),
      auth,
      callback: (
        requestError: ServiceError | null,
        response: CreateProjectCredentialResponse | null,
      ) => {
        if (requestId !== submissionSequence.current) return;
        setIsSubmitting(false);
        if (requestError) {
          setError(requestError.message || 'Unable to create API key.');
          return;
        }
        if (!response?.getSuccess()) {
          setError(
            response?.getError()?.getHumanmessage() ||
              'Unable to create API key.',
          );
          return;
        }
        const credentialKey = response.getData()?.getKey() || '';
        copySequence.current += 1;
        setGeneratedKey(credentialKey);
        setError(
          credentialKey
            ? ''
            : 'API key created, but its value was not returned.',
        );
        setIsCopied(false);
        onCreated();
      },
    });
  };

  const close = () => {
    submissionSequence.current += 1;
    copySequence.current += 1;
    setIsSubmitting(false);
    setGeneratedKey(null);
    setIsCopied(false);
    setError('');
    onClose();
  };

  const copyGeneratedKey = () => {
    if (!generatedKey || !navigator.clipboard) {
      setIsCopied(false);
      setError('Unable to copy the API key.');
      return;
    }
    const requestId = ++copySequence.current;
    const sessionId = submissionSequence.current;
    setIsCopied(false);
    setError('');
    try {
      navigator.clipboard
        .writeText(generatedKey)
        .then(() => {
          if (
            requestId !== copySequence.current ||
            sessionId !== submissionSequence.current
          )
            return;
          setIsCopied(true);
        })
        .catch(() => {
          if (
            requestId !== copySequence.current ||
            sessionId !== submissionSequence.current
          )
            return;
          setIsCopied(false);
          setError('Unable to copy the API key.');
        });
    } catch {
      if (
        requestId !== copySequence.current ||
        sessionId !== submissionSequence.current
      )
        return;
      setIsCopied(false);
      setError('Unable to copy the API key.');
    }
  };

  return (
    <Modal open={open} onClose={close} size="sm" preventCloseOnClickOutside>
      <ModalHeader
        label="Project Access"
        title="Create API key"
        onClose={close}
      />
      <ModalBody hasScrollingContent>
        {generatedKey === null ? (
          <div className="flex flex-col gap-4 py-2">
            <Text as="p" variant="body-compact-01" className="text-foreground">
              Create a project-scoped API key for an application or service.
              Access is governed by the assigned project role.
            </Text>
            <FieldSet>
              <Input
                id="project-api-key-application-name"
                labelText="Application Name"
                hideLabel={false}
                decorator={
                  <HelpToggletip
                    label="Application name"
                    helpText="Use a name that identifies the application, service, or integration that will use this key."
                  />
                }
                placeholder="Application name"
                value={applicationName}
                onChange={event => setApplicationName(event.target.value)}
                disabled={isSubmitting}
                required
                data-modal-primary-focus
              />
            </FieldSet>
            {listStatus === 'loading' && (
              <InlineLoading
                className="min-h-12"
                description="Loading project roles"
              />
            )}
            {listStatus === 'error' && (
              <div className="flex flex-col gap-3">
                <Text
                  as="p"
                  variant="label-01"
                  role="alert"
                  className="text-support-error"
                >
                  {error}
                </Text>
                <TertiaryButton
                  size="md"
                  type="button"
                  onClick={() => setListCycle(value => value + 1)}
                >
                  Retry
                </TertiaryButton>
              </div>
            )}
            {listStatus === 'empty' && (
              <Text as="p" variant="body-compact-01" className="text-muted">
                No active project roles are available for API access.
              </Text>
            )}
            {listStatus === 'ready' && (
              <ComboBox
                id="project-token-role"
                titleText="Project Role"
                placeholder="Search or select a role"
                items={roles}
                selectedItem={selectedRole}
                itemToString={role =>
                  role ? `${role.getName()} (${role.getKey()})` : ''
                }
                disabled={isSubmitting}
                onChange={({ selectedItem }) => {
                  setError('');
                  setSelectedRole(selectedItem || null);
                }}
              />
            )}
            {error && listStatus !== 'error' && (
              <Text
                as="p"
                variant="label-01"
                role="alert"
                className="text-support-error"
              >
                {error}
              </Text>
            )}
            {isSubmitting && (
              <InlineLoading
                className="min-h-12"
                description="Creating API key"
              />
            )}
          </div>
        ) : (
          <div className="flex flex-col gap-4 py-2">
            {generatedKey ? (
              <>
                <FieldSet>
                  <div className="flex items-center gap-1">
                    <FormLabel
                      id="generated-project-api-key-label"
                      htmlFor="generated-project-api-key"
                    >
                      API Key
                    </FormLabel>
                    <HelpToggletip
                      label="API key security"
                      helpText="Store this key in an approved secrets manager. For security, it cannot be retrieved after this dialog is closed."
                    />
                  </div>
                  <PasswordInput
                    id="generated-project-api-key"
                    labelText=""
                    hideLabel
                    aria-labelledby="generated-project-api-key-label"
                    value={generatedKey}
                    readOnly
                    showPasswordLabel="Show key"
                    hidePasswordLabel="Hide key"
                    tooltipPosition="left"
                  />
                </FieldSet>
                <div className="mt-2 flex gap-2 pr-[calc(20%-1rem)]">
                  <InformationFilled size={16} />
                  <Text as="p" variant="label-01" className="flex-1">
                    This API key grants the access defined by the assigned
                    project role.
                  </Text>
                </div>
                <div className="mt-2 flex gap-2 pr-[calc(20%-1rem)]">
                  <CheckmarkFilled size={16} className="text-primary" />
                  <Text
                    as="p"
                    variant="label-01"
                    className="flex-1"
                    role="status"
                    aria-live="polite"
                  >
                    API key created
                  </Text>
                </div>
              </>
            ) : (
              <Text
                as="p"
                variant="label-01"
                role="alert"
                className="text-support-error"
              >
                {error}
              </Text>
            )}
            {generatedKey && error && (
              <Text
                as="p"
                variant="label-01"
                role="alert"
                className="text-support-error"
              >
                {error}
              </Text>
            )}
          </div>
        )}
      </ModalBody>
      <ModalFooter>
        {generatedKey === null ? (
          <>
            <SecondaryButton size="lg" type="button" onClick={close}>
              Close
            </SecondaryButton>
            <PrimaryButton
              size="lg"
              type="button"
              disabled={
                isSubmitting || !applicationName.trim() || !selectedRole
              }
              onClick={submit}
            >
              Create API key
            </PrimaryButton>
          </>
        ) : (
          <>
            <SecondaryButton size="lg" type="button" onClick={close}>
              Close
            </SecondaryButton>
            <PrimaryButton
              size="lg"
              type="button"
              disabled={!generatedKey}
              onClick={copyGeneratedKey}
              renderIcon={isCopied ? CheckmarkFilled : Copy}
              iconDescription="Copy"
            >
              {isCopied ? 'Copied' : 'Copy'}
            </PrimaryButton>
          </>
        )}
      </ModalFooter>
    </Modal>
  );
}
