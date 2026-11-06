import React, { FC, useState } from 'react';
import { Metadata } from '@rapidaai/react';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { useCurrentCredential } from '@/hooks/use-credential';
import { useRapidaStore } from '@/stores/app';
import toast from 'react-hot-toast/headless';
import { PrimaryButton, SecondaryButton } from '@/app/components/ui/primitives';
import { Stack } from '@/app/components/ui/primitives';
import { ButtonSet } from '@carbon/react';
import { useConfirmDialog } from '@/app/pages/assistant/actions/hooks/use-confirmation';
import {
  CloudStorageProvider,
  defaultStorageFiles,
  GetDefaultStorageConfigIfInvalid,
  preserveStorageConfigurationOptions,
  StorageFileSelector,
  upsertStorageFilesOption,
  ValidateStorageOptions,
} from '@/app/components/domain/providers/storage';
import { STORAGE_PROVIDER } from '@/providers';
import { InputGroup } from '@/app/components/ui/primitives';
import { Notification } from '@/app/components/ui/feedback';
import { createAssistantConfigurationForAssistant } from '@/clients/assistant.client';

const storageConfigurationType = 'storage';

export const CreateAssistantStorage: FC<{ assistantId: string }> = ({
  assistantId,
}) => {
  const navigator = useGlobalNavigation();
  const { authId, token, projectId } = useCurrentCredential();
  const { loading, showLoader, hideLoader } = useRapidaStore();
  const { showDialog, ConfirmDialogComponent } = useConfirmDialog({});

  const defaultProvider = STORAGE_PROVIDER[0]?.code || '';
  const [provider, setProvider] = useState(defaultProvider);
  const [parameters, setParameters] = useState<Metadata[]>(
    GetDefaultStorageConfigIfInvalid(defaultProvider, []),
  );
  const [selectedFiles, setSelectedFiles] =
    useState<string[]>(defaultStorageFiles);
  const [errorMessage, setErrorMessage] = useState('');

  const onChangeProvider = (providerCode: string) => {
    setProvider(providerCode);
    setParameters(
      GetDefaultStorageConfigIfInvalid(
        providerCode,
        preserveStorageConfigurationOptions(parameters),
      ),
    );
  };

  const onSubmit = () => {
    setErrorMessage('');

    if (!provider) {
      setErrorMessage('Please select a storage provider.');
      return;
    }

    if (!ValidateStorageOptions(provider, parameters)) {
      setErrorMessage('Please complete the required storage fields.');
      return;
    }
    if (selectedFiles.length === 0) {
      setErrorMessage('Please select at least one file to push.');
      return;
    }

    showLoader();
    createAssistantConfigurationForAssistant({
      assistantId,
      configurationType: storageConfigurationType,
      provider,
      enabled: true,
      options: upsertStorageFilesOption(parameters, selectedFiles),
      auth: { projectId, token, userId: authId },
    })
      .then(response => {
        hideLoader();
        if (response?.getSuccess()) {
          toast.success('Storage created successfully');
          navigator.goToAssistantStorage(assistantId);
          return;
        }

        setErrorMessage(
          response?.getError()?.getHumanmessage() ||
            'Unable to create storage. Please try again.',
        );
      })
      .catch(() => {
        hideLoader();
        setErrorMessage('Unable to create storage. Please try again.');
      });
  };

  return (
    <>
      <ConfirmDialogComponent />
      <section className="flex flex-1 min-h-0 overflow-hidden">
        <div className="flex-1 min-h-0 flex flex-col bg-surface ">
          <div className="flex-1 min-h-0 overflow-y-auto flex flex-col">
            <div className="flex flex-col flex-1">
              <header className="px-4 pt-8 pb-6 border-b border-border-subtle ">
                <h1 className="text-xl font-semibold text-foreground leading-tight">
                  Storage
                </h1>
                <p className="text-sm text-muted mt-1.5 leading-relaxed">
                  Configure the provider and destination.
                </p>
              </header>

              <div className="pb-8 flex flex-col">
                <InputGroup
                  childClass="p-0! m-0! px-4!"
                  title={`Recording Files (${selectedFiles.length}/${defaultStorageFiles.length})`}
                >
                  <StorageFileSelector
                    group="Recording"
                    selectedFiles={selectedFiles}
                    onChange={setSelectedFiles}
                  />
                </InputGroup>

                <InputGroup title="Destination">
                  <Stack gap={6}>
                    <CloudStorageProvider
                      provider={provider}
                      onChangeProvider={onChangeProvider}
                      parameters={parameters}
                      onChangeParameter={setParameters}
                    />
                  </Stack>
                </InputGroup>
              </div>
            </div>
          </div>

          <div className="shrink-0">
            {errorMessage && (
              <Notification
                kind="error"
                title="Error"
                subtitle={errorMessage}
              />
            )}
            <ButtonSet className="!w-full [&>button]:!flex-1 [&>button]:!max-w-none">
              <SecondaryButton
                size="lg"
                onClick={() => showDialog(navigator.goBack)}
              >
                Cancel
              </SecondaryButton>
              <PrimaryButton size="lg" isLoading={loading} onClick={onSubmit}>
                Configure storage
              </PrimaryButton>
            </ButtonSet>
          </div>
        </div>
      </section>
    </>
  );
};
