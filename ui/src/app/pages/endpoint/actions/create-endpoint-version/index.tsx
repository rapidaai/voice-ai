import { FC, useCallback, useEffect, useState } from 'react';
import toast from 'react-hot-toast/headless';
import { Helmet } from '@/app/components/app-shell/helmet';
import { SourceControl } from '@carbon/icons-react';
import {
  CreateFullPageStep,
  CreateFullPageWithGlobalHeader,
} from '@/app/components/ui/composites';
import { ConfirmDialog } from '@/app/components/dialogs/shared';
import { useCurrentCredential } from '@/hooks/use-credential';
import { useRapidaStore } from '@/stores/app';
import { useAllProviderCredentials } from '@/hooks/use-model';
import {
  GetDefaultTextProviderConfigIfInvalid,
  GetDefaultTextProviderConfigOnProviderSwitch,
  TextProvider,
  ValidateTextProviderDefaultOptions,
} from '@/app/components/domain/providers/text';
import { useNavigate, useParams } from 'react-router-dom';
import { Metadata } from '@rapidaai/react';
import { ConfigPrompt } from '@/app/components/domain/configuration/config-prompt';
import {
  EndpointProviderModelAttribute,
  GetEndpointResponse,
} from '@rapidaai/react';
import { ServiceError } from '@rapidaai/react';
import { ChatCompletePrompt, Prompt } from '@/utils/prompt';
import { CreateEndpointProviderModelResponse } from '@rapidaai/react';
import { randomString } from '@/utils';
import { FieldSet } from '@/app/components/ui/primitives';
import { FormLabel } from '@/app/components/ui/primitives';
import { Textarea } from '@/app/components/ui/primitives';
import { InputHelper } from '@/app/components/ui/primitives';
import {
  createEndpointProviderModel as createEndpointProviderModelRequest,
  getEndpoint,
} from '@/clients';

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
   * Global navigator
   */
  const navigator = useNavigate();

  /**
   * error message
   */
  const [errorMessage, setErrorMessage] = useState('');
  const [versionNoteError, setVersionNoteError] = useState('');

  /**
   * global loading
   */
  const { showLoader, hideLoader } = useRapidaStore();
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
  const [promptInstanceId] = useState(() => randomString(10));
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

  const validateEndpointInstruction = (): boolean => {
    const error = ValidateTextProviderDefaultOptions(
      textProviderModel.provider,
      textProviderModel.parameters,
      providerCredentials
        .filter(c => c.getProvider() === textProviderModel.provider)
        .map(c => c.getId()),
    );
    if (error) {
      setErrorMessage(error);
      return false;
    }

    if (promptConfig.variables.length === 0) {
      setErrorMessage('Please define at least one variable.');
      return false;
    }

    // Check if the content is not empty
    const hasNonEmptyContent = promptConfig.prompt.some(
      item => item.content.trim() !== '',
    );
    if (!hasNonEmptyContent) {
      setErrorMessage('Please provide content for at least one prompt item.');
      return false;
    }

    // If all validations pass, proceed to the next tab
    setErrorMessage('');
    return true;
  };

  const createEndpointProviderModel = () =>
    new Promise<void>(resolve => {
      if (commitMessage.trim() === '') {
        setVersionNoteError('Enter a version note.');
        resolve();
        return;
      }
      setErrorMessage('');
      showLoader('overlay');
      const endpointProviderModelAttr = new EndpointProviderModelAttribute();
      endpointProviderModelAttr.setModelprovidername(
        textProviderModel.provider,
      );
      endpointProviderModelAttr.setEndpointmodeloptionsList(
        textProviderModel.parameters,
      );
      endpointProviderModelAttr.setDescription(commitMessage);
      endpointProviderModelAttr.setChatcompleteprompt(
        ChatCompletePrompt(promptConfig),
      );
      createEndpointProviderModelRequest({
        endpointId: endpointId!,
        endpointProviderModel: endpointProviderModelAttr,
        auth: {
          userId: authId,
          token,
          projectId,
        },
        callback: (error, response) => {
          afterCreateEndpointProviderModel(error, response);
          resolve();
        },
      });
    });

  useEffect(() => {
    showLoader('block');
    if (endpointId) {
      getEndpoint({
        endpointId,
        endpointProviderModelId: null,
        auth: {
          userId: authId,
          token,
          projectId,
        },
        callback: (
          err: ServiceError | null,
          response: GetEndpointResponse | null,
        ) => {
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
      });
    }
  }, [endpointId]);

  const [isShow, setIsShow] = useState(false);
  return (
    <>
      <Helmet title="Create endpoint version"></Helmet>
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

      <CreateFullPageWithGlobalHeader
        title="Create endpoint version"
        description="Update the endpoint configuration, then add a version note describing what changed. New versions are not deployed automatically."
        errorMessage={errorMessage}
        eyebrow="Endpoint version"
        headerIcon={SourceControl}
        nextButtonText="Continue"
        onCancel={() => setIsShow(true)}
        onRequestSubmit={createEndpointProviderModel}
        submitButtonText="Create"
      >
        <CreateFullPageStep
          title="Update Endpoint"
          description="Update the model, parameters, and prompt template for this endpoint."
          hideHeader
          onNext={validateEndpointInstruction}
        >
          <div className="flex max-w-4xl flex-col gap-8 px-8 pb-8 pt-6">
            <div className="flex flex-col gap-6">
              <TextProvider
                onChangeProvider={onChangeTextProvider}
                parameters={textProviderModel.parameters}
                provider={textProviderModel.provider}
                onChangeParameter={onChangeTextProviderParameter}
              />
            </div>

            <div className="flex flex-col gap-6">
              <ConfigPrompt
                instanceId={promptInstanceId}
                existingPrompt={promptConfig}
                onChange={prompt => setPromptConfig(prompt)}
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
                <FormLabel htmlFor="endpoint-version-note">
                  Version note
                </FormLabel>
                <Textarea
                  id="endpoint-version-note"
                  invalid={Boolean(versionNoteError)}
                  invalidText={versionNoteError}
                  row={5}
                  value={commitMessage}
                  placeholder="Provide a clear and detailed explanation of the purpose and modifications made to the endpoint."
                  onChange={event => {
                    setCommitMessage(event.target.value);
                    if (versionNoteError) {
                      setVersionNoteError('');
                    }
                  }}
                />
                <InputHelper>
                  Summarize the changes made to the endpoint, highlight key
                  updates, and specify why these modifications are necessary.
                </InputHelper>
              </FieldSet>
            </div>
          </div>
        </CreateFullPageStep>
      </CreateFullPageWithGlobalHeader>
    </>
  );
};
