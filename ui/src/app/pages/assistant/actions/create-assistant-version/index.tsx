import { FC, useEffect, useState } from 'react';
import { useRapidaStore } from '@/hooks';
import { useCredential } from '@/hooks/use-credential';
import { useParams } from 'react-router-dom';
import { Helmet } from '@/app/components/Helmet';
import {
  IBlueBGArrowButton,
  ICancelButton,
} from '@/app/components/Form/Button';
import { TabForm } from '@/app/components/Form/tab-form';
import { FieldSet } from '@/app/components/Form/Fieldset';
import { useConfirmDialog } from '@/app/pages/assistant/actions/hooks/use-confirmation';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import {
  CreateAssistantProvider,
  GetAssistantProviderResponse,
  GetAssistantRequest,
  GetAssistant,
  CreateAssistantProviderRequest,
  AssistantDefinition,
} from '@rapidaai/react';
import { FormLabel } from '@/app/components/form-label';
import { Textarea } from '@/app/components/Form/Textarea';
import { ConfigPrompt } from '@/app/components/configuration/config-prompt';
import { ErrorContainer } from '@/app/components/error-container';
import { ProviderConfig } from '@/app/components/providers';
import { ChatCompletePrompt, Prompt } from '@/utils/prompt';
import {
  GetDefaultTextProviderConfigIfInvalid,
  TextProvider,
} from '@/app/components/providers/text';
import { randomString } from '@/utils';
import { ValidateTextProviderDefaultOptions } from '@/app/components/providers/text/index';
import { connectionConfig } from '@/configs';
import { YellowNoticeBlock } from '@/app/components/container/message/notice-block';
import { ExternalLink, Info } from 'lucide-react';
import toast from 'react-hot-toast/headless';
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
  const [userId, token, projectId] = useCredential();
  const [activeTab, setActiveTab] = useState('change-assistant');
  const navigator = useGlobalNavigation();
  const [errorMessage, setErrorMessage] = useState('');
  const { showDialog, ConfirmDialogComponent } = useConfirmDialog({});
  const [selectedModel, setSelectedModel] = useState<ProviderConfig>({
    providerId: '198796716894742122',
    provider: 'azure',
    parameters: GetDefaultTextProviderConfigIfInvalid('azure', []),
  });
  const onChangeProvider = (providerId: string, providerName: string) => {
    setSelectedModel({
      providerId: providerId,
      provider: providerName,
      parameters: GetDefaultTextProviderConfigIfInvalid(
        providerName,
        selectedModel.parameters,
      ),
    });
  };

  const [template, setTemplate] = useState<{
    prompt: { role: string; content: string }[];
    variables: { name: string; type: string; defaultvalue: string }[];
  }>({
    prompt: [],
    variables: [],
  });

  const currentDate = new Date().toLocaleDateString();
  const [versionMessage, setVersionMessage] = useState(
    `Changed on ${currentDate}`,
  );
  const { loading, showLoader, hideLoader } = useRapidaStore();

  const validateInstruction = (): boolean => {
    setErrorMessage('');
    let err = ValidateTextProviderDefaultOptions(
      selectedModel.provider,
      selectedModel.parameters,
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

  const createProviderModel = () => {
    setErrorMessage('');
    if (!versionMessage || versionMessage.trim() === '') {
      setErrorMessage('Please provide a valid version description.');
      return;
    }
    showLoader();
    const request = new CreateAssistantProviderRequest();
    const model =
      new CreateAssistantProviderRequest.CreateAssistantProviderModel();
    model.setAssistantmodeloptionsList(selectedModel.parameters);
    model.setTemplate(ChatCompletePrompt(template));

    model.setModelproviderid(selectedModel.providerId);
    model.setModelprovidername(selectedModel.provider);
    request.setModel(model);
    request.setAssistantid(assistantId);
    request.setDescription(versionMessage);
    CreateAssistantProvider(connectionConfig, request, {
      authorization: token,
      'x-auth-id': userId,
      'x-project-id': projectId,
    })
      .then((car: GetAssistantProviderResponse) => {
        hideLoader();
        if (car?.getSuccess()) {
          toast.success(
            'Assistant version with model has been created successfully.',
          );
          navigator.goToAssistantVersions(assistantId);
        } else {
          const errorMessage =
            'Unable to create assistant version. please try again later.';
          const error = car?.getError();
          if (error) {
            setErrorMessage(error.getHumanmessage());
            return;
          }
          setErrorMessage(errorMessage);
          return;
        }
      })
      .catch(err => {
        setErrorMessage(
          'Unable to create assistant version. please try again later.',
        );
      });
  };

  //
  useEffect(() => {
    showLoader();
    const request = new GetAssistantRequest();
    const assistantDef = new AssistantDefinition();
    assistantDef.setAssistantid(assistantId);
    request.setAssistantdefinition(assistantDef);
    GetAssistant(connectionConfig, request, {
      authorization: token,
      'x-auth-id': userId,
      'x-project-id': projectId,
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
              providerId: assistantProvider.getModelproviderid(),
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
      <Helmet title="Create new version"></Helmet>
      <TabForm
        className="bg-linear-to-r from-white hover:shadow-alternate to-violet-500/5 dark:from-gray-950/30 dark:via-gray-950/10 dark:to-violet-950/20"
        activeTab={activeTab}
        formHeading="Complete all steps to create a new assistant version."
        onChangeActiveTab={() => {}}
        errorMessage={errorMessage}
        form={[
          {
            code: 'change-assistant',
            name: 'Update Assistant',
            description:
              "Update the assistant's definition — including the model, instructions, and variables — as needed.",
            actions: [
              <ICancelButton
                className="px-4 rounded-[2px]"
                onClick={() => showDialog(navigator.goBack)}
              >
                Cancel
              </ICancelButton>,
              <IBlueBGArrowButton
                type="button"
                isLoading={loading}
                onClick={() => {
                  if (validateInstruction()) {
                    setActiveTab('commit-assistant');
                  }
                }}
                className="px-4 rounded-[2px]"
              >
                Continue
              </IBlueBGArrowButton>,
            ],
            body: (
              <>
                <YellowNoticeBlock className="flex items-center">
                  <Info className="shrink-0 w-4 h-4" strokeWidth={1.5} />
                  <div className="ms-3 text-sm font-medium">
                    Please note that new versions of the assistant will not be
                    deployed automatically.
                  </div>
                  <a
                    target="_blank"
                    href="https://doc.rapida.ai/assistant/create-new-version"
                    className="h-7 flex items-center font-medium hover:underline ml-auto text-yellow-600"
                    rel="noreferrer"
                  >
                    Read documentation
                    <ExternalLink
                      className="shrink-0 w-4 h-4 ml-1.5"
                      strokeWidth={1.5}
                    />
                  </a>
                </YellowNoticeBlock>
                <div className="space-y-6 px-8 max-w-4xl">
                  <TextProvider
                    config={selectedModel}
                    onChangeConfig={setSelectedModel}
                    onChangeProvider={onChangeProvider}
                  />

                  <ConfigPrompt
                    instanceId={randomString(10)}
                    existingPrompt={template}
                    onChange={prompt => setTemplate(prompt)}
                  />
                </div>
              </>
            ),
          },
          {
            code: 'commit-assistant',
            name: 'Change definition',
            description:
              'Provide a clear description of the changes made in this version to help others understand what has been updated.',
            actions: [
              <ICancelButton
                className="px-4 rounded-[2px]"
                onClick={() => showDialog(navigator.goBack)}
              >
                Cancel
              </ICancelButton>,
              <IBlueBGArrowButton
                isLoading={loading}
                type="button"
                onClick={() => {
                  createProviderModel();
                }}
                className="px-4 rounded-[2px]"
              >
                Create new version
              </IBlueBGArrowButton>,
            ],
            body: (
              <>
                <YellowNoticeBlock className="flex items-center">
                  <Info className="shrink-0 w-4 h-4" strokeWidth={1.5} />
                  <div className="ms-3 text-sm font-medium">
                    Please note that new versions of the assistant will not be
                    deployed automatically.
                  </div>
                  <a
                    target="_blank"
                    href="https://doc.rapida.ai/assistant/create-new-version"
                    className="h-7 flex items-center font-medium hover:underline ml-auto text-yellow-600"
                    rel="noreferrer"
                  >
                    Read documentation
                    <ExternalLink
                      className="shrink-0 w-4 h-4 ml-1.5"
                      strokeWidth={1.5}
                    />
                  </a>
                </YellowNoticeBlock>
                <div className="space-y-6 px-8 max-w-4xl">
                  <FieldSet>
                    <FormLabel>Change description</FormLabel>
                    <Textarea
                      row={5}
                      value={versionMessage}
                      placeholder={'Describe the changes made in this version'}
                      onChange={t => setVersionMessage(t.target.value)}
                    />
                  </FieldSet>
                </div>
              </>
            ),
          },
        ]}
      />
    </>
  );
};
