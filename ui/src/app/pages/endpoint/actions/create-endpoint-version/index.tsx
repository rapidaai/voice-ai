import { FC, useCallback, useEffect, useState } from 'react';
import toast from 'react-hot-toast/headless';
import { Helmet } from '@/app/components/helmet';
import { PrimaryButton, SecondaryButton } from '@/app/components/carbon/button';
import { ButtonSet } from '@carbon/react';
import { TabForm } from '@/app/components/form/tab-form';
import ConfirmDialog from '@/app/components/base/modal/confirm-ui';
import { useCurrentCredential } from '@/hooks/use-credential';
import { useRapidaStore } from '@/hooks';
import { useAllProviderCredentials } from '@/hooks/use-model';
import {
  GetDefaultTextProviderConfigIfInvalid,
  GetDefaultTextProviderConfigOnProviderSwitch,
  TextProvider,
  ValidateTextProviderDefaultOptions,
} from '@/app/components/providers/text';
import { useNavigate, useParams } from 'react-router-dom';
import {
  ConnectionConfig,
  CreateEndpointProviderModel,
  GetEndpoint,
  Metadata,
} from '@rapidaai/react';
import { ConfigPrompt } from '@/app/components/configuration/config-prompt';
import {
  EndpointProviderModelAttribute,
  GetEndpointResponse,
} from '@rapidaai/react';
import { ServiceError } from '@rapidaai/react';
import { ChatCompletePrompt, Prompt } from '@/utils/prompt';
import { CreateEndpointProviderModelResponse } from '@rapidaai/react';
import { randomString } from '@/utils';
import { FieldSet } from '@/app/components/form/fieldset';
import { FormLabel } from '@/app/components/form-label';
import { Textarea } from '@/app/components/form/textarea';
import { connectionConfig } from '@/configs';
import { DocNoticeBlock } from '@/app/components/container/message/notice-block/doc-notice-block';
import { InputHelper } from '@/app/components/input-helper';

export const CreateNewVersionEndpointPage: FC = () => {
  /**
   * current endpointID for which the version is getting created
   */
  const { endpointId } = useParams();

  /**
   * authentication
   */
  const { authId, token, projectId } = useCurrentCredential();

  /**
   * mutli step form
   */
  const [activeTab, setActiveTab] = useState('choose-model');

  /**
   * Global navigator
   */
  const navigator = useNavigate();

  /**
   * error message
   */
  const [errorMessage, setErrorMessage] = useState('');

  /**
   * global loading
   */
  const { loading, showLoader, hideLoader } = useRapidaStore();
  const { providerCredentials } = useAllProviderCredentials();

  /**
   * form
   */
  const [commitMessage, setCommitMessage] = useState(
    `Changed on ${new Date().toLocaleDateString()}`,
  );
  const [promptConfig, setPromptConfig] = useState<{
    prompt: { role: string; content: string }[];
    variables: { name: string; type: string; defaultvalue: string }[];
  }>({
    prompt: [],
    variables: [],
  });
  const [textProviderModel, setTextProviderModel] = useState<{
    provider: string;
    parameters: Metadata[];
  }>({
    provider: 'azure-foundry',
    parameters: GetDefaultTextProviderConfigIfInvalid('azure-foundry', []),
  });
  const onChangeTextProvider = (providerName: string) => {
    setTextProviderModel({
      provider: providerName,
      parameters: GetDefaultTextProviderConfigOnProviderSwitch(
        providerName,
        textProviderModel.parameters,
      ),
    });
  };
  const onChangeTextProviderParameter = (parameters: Metadata[]) => {
    setTextProviderModel({ ...textProviderModel, parameters });
  };

  /**
   *
   */
  const afterCreateEndpointProviderModel = useCallback(
    (
      err: ServiceError | null,
      response: CreateEndpointProviderModelResponse | null,
    ) => {
      hideLoader();
      if (err) {
        setErrorMessage('Something went wrong, Please try again in sometime.');
        return;
      }
      if (response?.getSuccess() && response.getData()) {
        let ep = response.getData();
        toast.success('New version of endpoint successfully created.');
        navigator(`/deployment/endpoint/${ep?.getEndpointid()}/versions`);
        return;
      }
      if (response?.getError()) {
        let err = response.getError();
        const message = err?.getHumanmessage();
        if (message) {
          setErrorMessage(message);
          return;
        }
        setErrorMessage(
          'Unable to create endpoint, please check and try again.',
        );
      }
    },
    [],
  );
  /**
   *
   */

  const onvalidateEndpointInstruction = () => {
    const error = ValidateTextProviderDefaultOptions(
      textProviderModel.provider,
      textProviderModel.parameters,
      providerCredentials
        .filter(c => c.getProvider() === textProviderModel.provider)
        .map(c => c.getId()),
    );
    if (error) {
      setErrorMessage(error);
      return;
    }

    if (promptConfig.variables.length === 0) {
      setErrorMessage('Please define at least one variable.');
      return;
    }

    // Check if the content is not empty
    const hasNonEmptyContent = promptConfig.prompt.some(
      item => item.content.trim() !== '',
    );
    if (!hasNonEmptyContent) {
      setErrorMessage('Please provide content for at least one prompt item.');
      return;
    }

    // If all validations pass, proceed to the next tab
    setErrorMessage('');
    setActiveTab('commit-endpoint');
  };

  const createEndpointProviderModel = () => {
    if (commitMessage.trim() === '') {
      setErrorMessage(
        'Please a valid name for endpoint, that can help you indentify the endpoint',
      );
      return;
    }
    setErrorMessage('');
    showLoader('overlay');
    const endpointProviderModelAttr = new EndpointProviderModelAttribute();
    endpointProviderModelAttr.setModelprovidername(textProviderModel.provider);
    endpointProviderModelAttr.setEndpointmodeloptionsList(
      textProviderModel.parameters,
    );
    endpointProviderModelAttr.setDescription(commitMessage);
    endpointProviderModelAttr.setChatcompleteprompt(
      ChatCompletePrompt(promptConfig),
    );
    CreateEndpointProviderModel(
      connectionConfig,
      endpointId!,
      endpointProviderModelAttr,
      ConnectionConfig.WithDebugger({
        userId: authId,
        authorization: token,
        projectId: projectId,
      }),
      afterCreateEndpointProviderModel,
    );
  };

  useEffect(() => {
    showLoader('block');
    if (endpointId) {
      GetEndpoint(
        connectionConfig,
        endpointId,
        null,
        {
          'x-auth-id': authId,
          authorization: token,
          'x-project-id': projectId,
        },
        (err: ServiceError | null, response: GetEndpointResponse | null) => {
          hideLoader();
          if (err) {
            setErrorMessage(
              'Something went wrong, Please try again in sometime.',
            );
            return;
          }
          if (response?.getSuccess() && response.getData()) {
            const endpointProvider = response
              .getData()
              ?.getEndpointprovidermodel();
            if (endpointProvider) {
              setTextProviderModel({
                provider: endpointProvider.getModelprovidername(),
                parameters: GetDefaultTextProviderConfigIfInvalid(
                  endpointProvider.getModelprovidername(),
                  endpointProvider.getEndpointmodeloptionsList(),
                ),
              });
              const endpointPrompt = endpointProvider.getChatcompleteprompt();
              if (endpointPrompt) {
                setPromptConfig(Prompt(endpointPrompt));
              }
            }
            return;
          }
          if (response?.getError()) {
            let err = response.getError();
            const message = err?.getHumanmessage();
            if (message) {
              setErrorMessage(message);
              return;
            }
            setErrorMessage(
              'Unable to get endpoint, please check and try again.',
            );
          }
        },
      );
    }
  }, [endpointId]);

  const [isShow, setIsShow] = useState(false);
  return (
    <>
      <Helmet title="Create new version"></Helmet>
      <ConfirmDialog
        showing={isShow}
        type="warning"
        title={'Are you sure?'}
        content={
          'You want to cancel creating this endpoint? Any unsaved changes will be lost.'
        }
        confirmText={'Confirm'}
        cancelText="Cancel"
        onConfirm={() => {
          navigator(-1);
        }}
        onCancel={() => {
          setIsShow(false);
        }}
        onClose={() => {
          setIsShow(false);
        }}
      />

      <TabForm
        activeTab={activeTab}
        onChangeActiveTab={() => {}}
        formHeading="Create a new version of this endpoint."
        errorMessage={errorMessage}
        form={[
          {
            name: 'Configure model',
            description:
              'Modify the model, parameters, and prompt template for this endpoint.',
            code: 'choose-model',
            body: (
              <>
                <DocNoticeBlock
                  docPath="/endpoint/create-new-version"
                  linkText="Read docs"
                >
                  New versions of the endpoint will not be deployed
                  automatically. You must promote them manually.
                </DocNoticeBlock>
                <div className="px-8 pt-6 pb-8 max-w-4xl flex flex-col gap-8">
                  {/* Model configuration section */}
                  <div className="flex flex-col gap-6">
                    <TextProvider
                      onChangeProvider={onChangeTextProvider}
                      parameters={textProviderModel.parameters}
                      provider={textProviderModel.provider}
                      onChangeParameter={onChangeTextProviderParameter}
                    />
                  </div>

                  {/* Prompt template section */}
                  <div className="flex flex-col gap-6">
                    <ConfigPrompt
                      instanceId={randomString(10)}
                      existingPrompt={promptConfig}
                      onChange={prompt => {
                        setPromptConfig(prompt);
                      }}
                    />
                  </div>
                </div>
              </>
            ),
            actions: [
              <ButtonSet className="!w-full [&>button]:!flex-1 [&>button]:!max-w-none">
                <SecondaryButton size="lg" onClick={() => setIsShow(true)}>
                  Cancel
                </SecondaryButton>
                <PrimaryButton
                  size="lg"
                  isLoading={loading}
                  onClick={onvalidateEndpointInstruction}
                >
                  Configure instruction
                </PrimaryButton>
              </ButtonSet>,
            ],
          },

          {
            code: 'commit-endpoint',
            name: 'Version note',
            description:
              'Write a brief note describing what changed in this version.',
            actions: [
              <ButtonSet className="!w-full [&>button]:!flex-1 [&>button]:!max-w-none">
                <SecondaryButton size="lg" onClick={() => setIsShow(true)}>
                  Cancel
                </SecondaryButton>
                <PrimaryButton
                  size="lg"
                  isLoading={loading}
                  onClick={() => {
                    createEndpointProviderModel();
                  }}
                >
                  Create new version
                </PrimaryButton>
              </ButtonSet>,
            ],
            body: (
              <div className="px-8 pt-8 pb-8 max-w-2xl flex flex-col gap-10">
                <div className="flex flex-col gap-6">
                  <FieldSet>
                    <FormLabel>Version note</FormLabel>
                    <Textarea
                      row={5}
                      value={commitMessage}
                      placeholder="Provide a clear and detailed explanation of the purpose and modifications made to the endpoint."
                      onChange={t => setCommitMessage(t.target.value)}
                    />
                    <InputHelper>
                      Summarize the changes made to the endpoint, highlight key
                      updates, and specify why these modifications are
                      necessary.
                    </InputHelper>
                  </FieldSet>
                </div>
              </div>
            ),
          },
        ]}
      />
    </>
  );
};
