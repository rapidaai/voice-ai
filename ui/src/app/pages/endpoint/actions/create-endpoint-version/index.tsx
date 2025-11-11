import { FC, useCallback, useEffect, useState } from 'react';
import toast from 'react-hot-toast/headless';
import { Helmet } from '@/app/components/Helmet';
import {
  IBlueBGArrowButton,
  ICancelButton,
} from '@/app/components/Form/Button';
import { TabForm } from '@/app/components/Form/tab-form';
import ConfirmDialog from '@/app/components/base/modal/confirm-ui';
import { useCurrentCredential } from '@/hooks/use-credential';
import { useRapidaStore } from '@/hooks';
import { ProviderConfig } from '@/app/components/providers';
import {
  GetDefaultTextProviderConfigIfInvalid,
  TextProvider,
  ValidateTextProviderDefaultOptions,
} from '@/app/components/providers/text';
import { useNavigate, useParams } from 'react-router-dom';
import { CreateEndpointProviderModel, GetEndpoint } from '@rapidaai/react';
import { ConfigPrompt } from '@/app/components/configuration/config-prompt';
import {
  EndpointProviderModelAttribute,
  GetEndpointResponse,
} from '@rapidaai/react';
import { ServiceError } from '@rapidaai/react';
import { ChatCompletePrompt, Prompt } from '@/utils/prompt';
import { CreateEndpointProviderModelResponse } from '@rapidaai/react';
import { randomString } from '@/utils';
import { FieldSet } from '@/app/components/Form/Fieldset';
import { FormLabel } from '@/app/components/form-label';
import { Textarea } from '@/app/components/Form/Textarea';
import { connectionConfig } from '@/configs';
import { YellowNoticeBlock } from '@/app/components/container/message/notice-block';
import { ExternalLink, Info } from 'lucide-react';
import { InputHelper } from '@/app/components/input-helper';

export const CreateNewVersionEndpointPage: FC = () => {
  const { endpointId } = useParams();
  const { authId, token, projectId } = useCurrentCredential();
  const [activeTab, setActiveTab] = useState('choose-model');

  //

  const currentDate = new Date().toLocaleDateString();
  const [commitMessage, setCommitMessage] = useState(
    `Changed on ${currentDate}`,
  );
  //
  const [errorMessage, setErrorMessage] = useState('');
  const { loading, showLoader, hideLoader } = useRapidaStore();

  //
  const [textProviderModel, setTextProviderModel] = useState<ProviderConfig>({
    providerId: '198796716894742122',
    provider: 'azure-openai',
    parameters: GetDefaultTextProviderConfigIfInvalid('azure-openai', []),
  });

  const onChangeTextProvider = (providerId: string, providerName: string) => {
    setTextProviderModel({
      providerId: providerId,
      provider: providerName,
      parameters: GetDefaultTextProviderConfigIfInvalid(
        providerName,
        textProviderModel.parameters,
      ),
    });
  };

  const [promptConfig, setPromptConfig] = useState<{
    prompt: { role: string; content: string }[];
    variables: { name: string; type: string; defaultvalue: string }[];
  }>({
    prompt: [],
    variables: [],
  });

  let navigator = useNavigate();

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
        navigator(`/deployment/endpoint/${ep?.getEndpointid()}`);
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
    endpointProviderModelAttr.setModelproviderid(textProviderModel.providerId);
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
      { 'x-auth-id': authId, authorization: token, 'x-project-id': projectId },
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
                providerId: endpointProvider.getModelproviderid(),
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
        formHeading="Complete the step to create new version of endpoint"
        errorMessage={errorMessage}
        form={[
          {
            name: 'Modify Endpoint',
            description:
              'Change endpoint defnition, change model, instruction and variables for the endpoint',
            code: 'choose-model',
            body: (
              <>
                <YellowNoticeBlock className="flex items-center">
                  <Info className="shrink-0 w-4 h-4" strokeWidth={1.5} />
                  <div className="ms-3 text-sm font-medium">
                    Please note that new versions of the endpoint will not be
                    deployed automatically.
                  </div>
                  <a
                    target="_blank"
                    href="https://doc.rapida.ai/endpoint/create-new-version"
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
                <div className="space-y-6 px-8 pb-8 max-w-4xl ">
                  <TextProvider
                    onChangeProvider={onChangeTextProvider}
                    onChangeConfig={setTextProviderModel}
                    config={textProviderModel}
                  />
                  <ConfigPrompt
                    instanceId={randomString(10)}
                    existingPrompt={promptConfig}
                    onChange={prompt => {
                      setPromptConfig(prompt);
                    }}
                  />
                </div>
              </>
            ),
            actions: [
              <ICancelButton
                className="px-4 rounded-[2px]"
                onClick={() => setIsShow(true)}
              >
                Cancel
              </ICancelButton>,
              <IBlueBGArrowButton
                type="button"
                isLoading={loading}
                className="px-4 rounded-[2px]"
                onClick={onvalidateEndpointInstruction}
              >
                Configure instruction
              </IBlueBGArrowButton>,
            ],
          },

          {
            code: 'commit-endpoint',
            name: 'Change definition',
            description:
              'Give a change definition that will help people to understand what has been change in this version',
            actions: [
              <ICancelButton
                className="px-4 rounded-[2px]"
                onClick={() => setIsShow(true)}
              >
                Cancel
              </ICancelButton>,
              <IBlueBGArrowButton
                className="px-4 rounded-[2px]"
                type="button"
                isLoading={loading}
                onClick={() => {
                  createEndpointProviderModel();
                }}
              >
                Create new version
              </IBlueBGArrowButton>,
            ],
            body: (
              <>
                <YellowNoticeBlock className="flex items-center">
                  <Info className="shrink-0 w-4 h-4" strokeWidth={1.5} />
                  <div className="ms-3 text-sm font-medium">
                    Please note that new versions of the endpoint will not be
                    deployed automatically.
                  </div>
                  <a
                    target="_blank"
                    href="https://doc.rapida.ai/endpoint/create-new-version"
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
                <div className="space-y-6 px-8 pb-8 max-w-4xl ">
                  <FieldSet>
                    <FormLabel>Change description</FormLabel>
                    <Textarea
                      row={5}
                      value={commitMessage}
                      placeholder={
                        'Provide a clear and detailed explanation of the purpose and modifications made to the endpoint.'
                      }
                      onChange={t => setCommitMessage(t.target.value)}
                    />
                    <InputHelper>
                      Use this field to summarize the changes made to the
                      endpoint, highlight key updates, and specify why these
                      modifications are necessary.
                    </InputHelper>
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
