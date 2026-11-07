import { FC, useEffect, useState } from 'react';
import { useRapidaStore } from '@/stores/app';
import { useCurrentCredential } from '@/hooks/use-credential';
import { useParams } from 'react-router-dom';
import { Helmet } from '@/app/components/app-shell/helmet';
import { Toggletip, ToggletipButton, ToggletipContent } from '@carbon/react';
import { Information, SourceControl } from '@carbon/icons-react';
import {
  CreateFullPageStep,
  CreateFullPageWithGlobalHeader,
} from '@/app/components/ui/composites';
import { FieldSet } from '@/app/components/ui/primitives';
import { useConfirmDialog } from '@/app/pages/assistant/actions/hooks/use-confirmation';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { CreateAssistantProviderRequest, Metadata } from '@rapidaai/react';
import type { GetAssistantProviderResponse } from '@rapidaai/react';
import { FormLabel } from '@/app/components/ui/primitives';
import { Textarea } from '@/app/components/ui/primitives';
import { ConfigPrompt } from '@/app/components/domain/configuration/config-prompt';
import { ErrorContainer } from '@/app/components/ui/feedback';
import { ChatCompletePrompt, Prompt } from '@/utils/prompt';
import {
  GetDefaultTextProviderConfigIfInvalid,
  GetDefaultTextProviderConfigOnProviderSwitch,
  TextProvider,
} from '@/app/components/domain/providers/text';
import { randomString } from '@/utils';
import { ValidateTextProviderDefaultOptions } from '@/app/components/domain/providers/text';
import { useAllProviderCredentials } from '@/hooks/use-model';
import { DocNoticeBlock } from '@/app/components/layout/container/message/notice-block/doc-notice-block';
import toast from 'react-hot-toast/headless';
import {
  createAssistantProviderWithDebuggerFromRequest,
  getAssistantById,
} from '@/clients/assistant.client';

/**
 *
 * @returns
 */
export function CreateVersionAssistantPage() {
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
          description="This assistant may be archived or you don't have access to it. Please check with your administrator or try another assistant."
        />
      </div>
    );
  return <CreateNewVersion assistantId={assistantId!} />;
}

/**
 *
 * @param props
 * @returns
 */
const CreateNewVersion: FC<{ assistantId: string }> = ({ assistantId }) => {
  /**
   *
   */
  const { authId, token, projectId } = useCurrentCredential();
  const navigator = useGlobalNavigation();
  /**
   * error message
   */
  const [errorMessage, setErrorMessage] = useState('');
  const [versionNoteError, setVersionNoteError] = useState('');
  /**
   * confirmation dialog
   */
  const { showDialog, ConfirmDialogComponent } = useConfirmDialog({});
  /**
   * selected model and parameters
   */
  const [selectedModel, setSelectedModel] = useState<{
    provider: string;
    parameters: Metadata[];
  }>({
    provider: 'azure-foundry',
    parameters: GetDefaultTextProviderConfigIfInvalid('azure-foundry', []),
  });

  /**
   * prompt template
   */
  const [template, setTemplate] = useState<{
    prompt: { role: string; content: string }[];
    variables: { name: string; type: string; defaultvalue: string }[];
  }>({
    prompt: [{ role: 'system', content: '' }],
    variables: [],
  });

  /**
   * current data used as commit message
   */
  const [versionMessage, setVersionMessage] = useState(
    `Changed on ${new Date().toLocaleDateString()}`,
  );
  const [promptInstanceId] = useState(() => randomString(10));

  /**
   * global loader
   */
  const { showLoader, hideLoader } = useRapidaStore();
  const { providerCredentials } = useAllProviderCredentials();

  const onChangeProvider = (providerName: string) => {
    setSelectedModel({
      provider: providerName,
      parameters: GetDefaultTextProviderConfigOnProviderSwitch(
        providerName,
        selectedModel.parameters,
      ),
    });
  };

  const onChangeProviderParameter = (parameters: Metadata[]) => {
    setSelectedModel({ ...selectedModel, parameters });
  };

  const validateInstruction = (): boolean => {
    setErrorMessage('');
    let err = ValidateTextProviderDefaultOptions(
      selectedModel.provider,
      selectedModel.parameters,
      providerCredentials
        .filter(c => c.getProvider() === selectedModel.provider)
        .map(c => c.getId()),
    );
    if (err) {
      setErrorMessage(err);
      return false;
    }

    // Add template prompt validation
    if (!template.prompt || template.prompt.length === 0) {
      setErrorMessage('Please provide a valid template prompt.');
      return false;
    }

    // Validate each prompt message in the template
    for (const message of template.prompt) {
      if (!message.role || !message.content || message.content.trim() === '') {
        setErrorMessage(
          'Each prompt message must have a valid role and non-empty content.',
        );
        return false;
      }
    }
    return true;
  };

  const createProviderModel = async () => {
    setErrorMessage('');
    if (!versionMessage || versionMessage.trim() === '') {
      const message = 'Enter a version note.';
      setVersionNoteError(message);
      return;
    }
    showLoader();
    const request = new CreateAssistantProviderRequest();
    const model =
      new CreateAssistantProviderRequest.CreateAssistantProviderModel();
    model.setAssistantmodeloptionsList(selectedModel.parameters);
    model.setTemplate(ChatCompletePrompt(template));
    model.setModelprovidername(selectedModel.provider);
    request.setModel(model);
    request.setAssistantid(assistantId);
    request.setDescription(versionMessage);
    try {
      const car: GetAssistantProviderResponse =
        await createAssistantProviderWithDebuggerFromRequest({
          request,
          auth: { projectId, token, userId: authId },
        });
      if (car?.getSuccess()) {
        toast.success(
          'Assistant version with model has been created successfully.',
        );
        navigator.goToAssistantVersions(assistantId);
        return;
      }

      const error = car?.getError();
      setErrorMessage(
        error?.getHumanmessage() ||
          'Unable to create assistant version. please try again later.',
      );
    } catch {
      setErrorMessage(
        'Unable to create assistant version. please try again later.',
      );
    } finally {
      hideLoader();
    }
  };

  //
  useEffect(() => {
    showLoader();
    getAssistantById({
      assistantId,
      auth: { projectId, token, userId: authId },
    })
      .then(response => {
        hideLoader();
        if (response?.getSuccess()) {
          const assistantProvider = response
            .getData()
            ?.getAssistantprovidermodel();
          if (assistantProvider) {
            setTemplate(Prompt(assistantProvider.getTemplate()!));
            setSelectedModel({
              parameters: GetDefaultTextProviderConfigIfInvalid(
                assistantProvider.getModelprovidername(),
                assistantProvider.getAssistantmodeloptionsList(),
              ),
              provider: assistantProvider.getModelprovidername(),
            });
          }
          return;
        }
        const error = response?.getError();
        const errorMsg = error
          ? error.getHumanmessage()
          : 'Unable to get latest assistant provider. Please try again later.';
        setErrorMessage(errorMsg);
      })
      .catch(err => {
        hideLoader();
        setErrorMessage(
          'Unable to get latest assistant provider. Please try again later.',
        );
      });
  }, [assistantId]);

  return (
    <>
      <ConfirmDialogComponent />
      <Helmet title="Create version"></Helmet>
      <CreateFullPageWithGlobalHeader
        title="Create version"
        description="Update the assistant configuration, then add a version note describing what changed. New versions are not deployed automatically."
        errorMessage={errorMessage}
        eyebrow="Assistant version"
        headerIcon={SourceControl}
        nextButtonText="Continue"
        onCancel={() => showDialog(navigator.goBack)}
        onRequestSubmit={createProviderModel}
        submitButtonText="Create"
      >
        <CreateFullPageStep
          title="Update Assistant"
          description="Update the assistant definition, including its model, instructions, and variables."
          hideHeader
          onNext={validateInstruction}
        >
          <div className="flex max-w-4xl flex-col gap-8 px-8 pb-8 pt-6">
            <div className="flex flex-col gap-6">
              <TextProvider
                onChangeParameter={onChangeProviderParameter}
                onChangeProvider={onChangeProvider}
                parameters={selectedModel.parameters}
                provider={selectedModel.provider}
              />
            </div>

            <div className="flex flex-col gap-6">
              <DocNoticeBlock docPath="/assistants/prompt-templating">
                Prompt variables and system arguments are resolved at runtime.
                Read the prompt templating guide before creating the version.
              </DocNoticeBlock>
              <ConfigPrompt
                instanceId={promptInstanceId}
                existingPrompt={template}
                showRuntimeReplacementHint
                enableReservedVariableSuggestions
                onChange={prompt => setTemplate(prompt)}
              />
            </div>
          </div>
        </CreateFullPageStep>

        <CreateFullPageStep
          title="Version Note"
          description="Write a brief note describing what changed in this version."
          hideHeader
        >
          <div className="flex max-w-2xl flex-col gap-10 px-8 pb-8 pt-8">
            <div className="flex flex-col gap-6">
              <FieldSet>
                <div className="mb-2 flex items-center gap-1">
                  <FormLabel htmlFor="model-version-note">
                    Version note
                  </FormLabel>
                  <Toggletip align="right">
                    <ToggletipButton label="Show information">
                      <Information size={14} />
                    </ToggletipButton>
                    <ToggletipContent>
                      Briefly describe what changed in this version.
                    </ToggletipContent>
                  </Toggletip>
                </div>
                <Textarea
                  invalid={Boolean(versionNoteError)}
                  invalidText={versionNoteError}
                  name="model-version-note"
                  row={5}
                  value={versionMessage}
                  placeholder="Provide a clear and detailed explanation of the changes made to this assistant."
                  onChange={event => {
                    setVersionMessage(event.target.value);
                    if (versionNoteError) {
                      setVersionNoteError('');
                    }
                  }}
                />
              </FieldSet>
            </div>
          </div>
        </CreateFullPageStep>
      </CreateFullPageWithGlobalHeader>
    </>
  );
};
