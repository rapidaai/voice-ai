import type { GetAssistantResponse, ServiceError } from '@rapidaai/react';
import { ErrorContainer } from '@/app/components/ui/feedback';
import { useDeleteConfirmDialog } from '@/app/pages/assistant/actions/hooks/use-delete-confirmation';
import { useRapidaStore } from '@/stores/app';
import { useCurrentCredential } from '@/hooks/use-credential';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { FC, ReactNode, useEffect, useState } from 'react';
import toast from 'react-hot-toast/headless';
import { useParams } from 'react-router-dom';
import { Notification } from '@/app/components/ui/feedback';
import {
  Form,
  Stack,
  TextInput,
  TextArea,
} from '@/app/components/ui/primitives';
import { PrimaryButton, DangerButton } from '@/app/components/ui/primitives';
import { CopyButton } from '@/app/components/ui/primitives';
import {
  Toggletip as CarbonToggletip,
  ToggletipButton,
  ToggletipContent,
} from '@carbon/react';
import { Information, WarningAlt } from '@carbon/icons-react';
import { InputGroup } from '@/app/components/ui/primitives';
import {
  deleteAssistantById,
  getAssistantById,
  updateAssistantDescription,
} from '@/clients/assistant.client';
import { AssistantSettingsShellHeader } from '@/app/pages/assistant/components/settings-shell-header';

const Toggletip = (CarbonToggletip as any).default || CarbonToggletip;

const FieldLabelWithToggletip: FC<{
  label: string;
  description: ReactNode;
}> = ({ label, description }) => (
  <span className="inline-flex items-center gap-1.5">
    <span>{label}</span>
    <Toggletip align="right">
      <ToggletipButton
        label={`${label} information`}
        title={`${label} information`}
      >
        <Information size={14} />
      </ToggletipButton>
      <ToggletipContent>{description}</ToggletipContent>
    </Toggletip>
  </span>
);

export function EditAssistantPage() {
  const { assistantId } = useParams();
  const { goToAssistantListing } = useGlobalNavigation();

  if (!assistantId)
    return (
      <div className="flex flex-1">
        <ErrorContainer
          onAction={goToAssistantListing}
          code="403"
          actionLabel="Go to listing"
          title="Assistant not available"
          description="This assistant may be archived or you don't have access to it."
        />
      </div>
    );

  return <EditAssistant assistantId={assistantId!} />;
}

export const EditAssistant: FC<{ assistantId: string }> = ({ assistantId }) => {
  const { authId, token, projectId } = useCurrentCredential();
  const { loading, showLoader, hideLoader } = useRapidaStore();
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [errorMessage, setErrorMessage] = useState('');
  const { goToAssistantListing } = useGlobalNavigation();

  useEffect(() => {
    showLoader('block');
    getAssistantById({
      assistantId,
      auth: { projectId, token, userId: authId },
    })
      .then(car => {
        hideLoader();
        if (car?.getSuccess()) {
          const assistant = car.getData();
          if (assistant) {
            setName(assistant.getName());
            setDescription(assistant.getDescription());
          }
        } else {
          const error = car?.getError();
          if (error) {
            toast.error(error.getHumanmessage());
            return;
          }
          toast.error('Unable to load assistant. Please try again later.');
        }
      })
      .catch(() => {
        hideLoader();
      });
  }, [assistantId, authId, hideLoader, projectId, showLoader, token]);

  const onUpdateAssistantDetail = () => {
    setErrorMessage('');
    showLoader('block');
    const afterUpdateAssistant = (
      err: ServiceError | null,
      car: GetAssistantResponse | null,
    ) => {
      hideLoader();
      if (car?.getSuccess()) {
        toast.success('The assistant has been successfully updated.');
        const assistant = car.getData();
        if (assistant) {
          setName(assistant.getName());
          setDescription(assistant.getDescription());
        }
      } else {
        const error = car?.getError();
        if (error) {
          setErrorMessage(error.getHumanmessage());
          return;
        }
        setErrorMessage('Unable to update assistant. Please try again later.');
      }
    };
    updateAssistantDescription({
      assistantId,
      name,
      description,
      callback: afterUpdateAssistant,
      auth: { projectId, token, userId: authId },
    });
  };

  const Deletion = useDeleteConfirmDialog({
    onConfirm: () => {
      showLoader('block');
      const afterDeleteAssistant = (
        err: ServiceError | null,
        car: GetAssistantResponse | null,
      ) => {
        if (car?.getSuccess()) {
          toast.success('The assistant has been deleted successfully.');
          goToAssistantListing();
        } else {
          hideLoader();
          const error = car?.getError();
          if (error) {
            toast.error(error.getHumanmessage());
            return;
          }
          toast.error('Unable to delete assistant. Please try again later.');
        }
      };
      deleteAssistantById({
        assistantId,
        callback: afterDeleteAssistant,
        auth: { projectId, token, userId: authId },
      });
    },
    name: name,
  });

  return (
    <div className="w-full flex flex-col flex-1 overflow-auto bg-white dark:bg-gray-900">
      <Deletion.ConfirmDeleteDialogComponent />

      <AssistantSettingsShellHeader
        title="General settings"
        href={`/deployment/assistant/${assistantId}/edit-assistant`}
      />

      <InputGroup title="Identity">
        <div className="flex max-w-full flex-col items-start gap-2">
          <FieldLabelWithToggletip
            label="Assistant ID"
            description="Your assistant's unique identifier. This cannot be changed."
          />
          <div
            id="assistant-id"
            className="inline-flex max-w-full items-center gap-1"
          >
            <span className="min-w-0 max-w-[32rem] truncate font-mono text-sm text-gray-700 dark:text-gray-200">
              {assistantId}
            </span>
            <CopyButton className="h-6 w-6 shrink-0">{assistantId}</CopyButton>
          </div>
        </div>
      </InputGroup>
      <InputGroup title="General Information">
        <Form
          className="flex flex-col gap-8 max-w-2xl"
          onSubmit={e => {
            e.preventDefault();
            onUpdateAssistantDetail();
          }}
        >
          <Stack gap={6}>
            <TextInput
              id="assistant-name"
              labelText={
                <FieldLabelWithToggletip
                  label="Name"
                  description="The display name shown across the platform."
                />
              }
              value={name}
              onChange={e => setName(e.target.value)}
              placeholder="e.g. Customer support bot"
            />
            <TextArea
              id="assistant-description"
              labelText={
                <FieldLabelWithToggletip
                  label="Description"
                  description="Describe what this assistant does and its intended use case."
                />
              }
              value={description}
              rows={4}
              onChange={e => setDescription(e.target.value)}
              placeholder="What's the purpose of this assistant?"
            />
            {errorMessage && (
              <Notification
                kind="error"
                title="Error"
                subtitle={errorMessage}
              />
            )}
            <div>
              <PrimaryButton
                size="md"
                className="assistant-settings-save"
                isLoading={loading}
                onClick={onUpdateAssistantDetail}
              >
                Save changes
              </PrimaryButton>
            </div>
          </Stack>
        </Form>
      </InputGroup>
      <div className="px-4 pt-6 pb-12 flex flex-col gap-8 max-w-2xl">
        {/* ── General Information ── */}

        {/* ── Danger Zone ── */}
        <div>
          <h2 className="text-sm font-semibold uppercase tracking-wider text-[var(--cds-support-error)] mb-4">
            Danger Zone
          </h2>
          <div>
            <div className="flex items-start justify-between gap-6">
              <div className="flex flex-col gap-1">
                <p className="text-sm font-semibold flex items-center gap-2">
                  <WarningAlt
                    size={16}
                    className="text-[var(--cds-support-error)]"
                  />
                  Delete this assistant
                </p>
                <p className="text-xs leading-relaxed text-[var(--cds-text-secondary)]">
                  Once deleted, all active connections will be terminated
                  immediately and data will be permanently removed. This action
                  cannot be undone.
                </p>
              </div>
              <DangerButton
                size="md"
                isLoading={loading}
                onClick={Deletion.showDialog}
              >
                Delete assistant
              </DangerButton>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};
