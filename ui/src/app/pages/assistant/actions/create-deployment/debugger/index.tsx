import {
  ConfigureExperience,
  ExperienceConfig,
} from '@/app/pages/assistant/actions/create-deployment/commons/configure-experience';
import { ConfigureAudioOutputProvider } from '@/app/pages/assistant/actions/create-deployment/commons/configure-audio-output';
import { ConfigureAudioInputProvider } from '@/app/pages/assistant/actions/create-deployment/commons/configure-audio-input';
import { useRapidaStore } from '@/hooks';
import { useCurrentCredential } from '@/hooks/use-credential';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { FC, useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import {
  AssistantDebuggerDeployment,
  ConnectionConfig,
  CreateAssistantDebuggerDeployment,
  CreateAssistantDeploymentRequest,
  DeploymentAudioProvider,
  GetAssistantDeploymentRequest,
  Metadata,
} from '@rapidaai/react';
import { GetAssistantDebuggerDeployment } from '@rapidaai/react';
import {
  IBlueBGArrowButton,
  ICancelButton,
} from '@/app/components/form/button';
import toast from 'react-hot-toast/headless';
import { Helmet } from '@/app/components/helmet';
import {
  GetDefaultMicrophoneConfig,
  GetDefaultSpeechToTextIfInvalid,
  ValidateSpeechToTextIfInvalid,
} from '@/app/components/providers/speech-to-text/provider';
import {
  GetDefaultSpeakerConfig,
  GetDefaultTextToSpeechIfInvalid,
} from '@/app/components/providers/text-to-speech/provider';
import { PageActionButtonBlock } from '@/app/components/blocks/page-action-button-block';
import { connectionConfig } from '@/configs';
import { ValidateTextToSpeechIfInvalid } from '@/app/components/providers/text-to-speech/provider';
import { useConfirmDialog } from '@/app/pages/assistant/actions/hooks/use-confirmation';
import { DebuggerDeploymentSuccessDialog } from '@/app/components/base/modal/debugger-deployment-success-modal';

/**
 *
 * @returns
 */
export function ConfigureAssistantDebuggerDeploymentPage() {
  const { assistantId } = useParams();
  return (
    <>
      <Helmet title="Configure deubgger deployment" />
      {assistantId && (
        <ConfigureAssistantDebuggerDeployment assistantId={assistantId} />
      )}
    </>
  );
}

/**
 * Configure assistant web deployment
 * this provide a list of web deployment configuration
 * @param param0
 * @returns
 */
const ConfigureAssistantDebuggerDeployment: FC<{ assistantId: string }> = ({
  assistantId,
}) => {
  /**
   * global naviagtion
   */
  const { goToDeploymentAssistant } = useGlobalNavigation();

  /**
   * global loading
   */
  const { loading, showLoader, hideLoader } = useRapidaStore();

  /**
   * cradentials
   */
  const { authId, projectId, token } = useCurrentCredential();

  /**
   * error messages
   */
  const [errorMessage, setErrorMessage] = useState('');

  /**
   * voice enabled
   */
  const [voiceInputEnable, setVoiceInputEnable] = useState(false);

  /**
   * voice output enabled
   */
  const [voiceOutputEnable, setVoiceOutputEnable] = useState(false);

  /**
   * experience for voice
   */
  const [experienceConfig, setExperienceConfig] = useState<ExperienceConfig>({
    greeting: undefined,
    messageOnError: undefined,
    idealTimeout: '5000',
    idealMessage: 'Are you there?',
    maxCallDuration: '10000',
  });

  /**
   * input audio config
   */
  const [audioInputConfig, setAudioInputConfig] = useState<{
    provider: string;
    parameters: Metadata[];
  }>({
    provider: 'deepgram',
    parameters: GetDefaultSpeechToTextIfInvalid('deepgram', []),
  });
  const [audioOutputConfig, setAudioOutputConfig] = useState<{
    provider: string;
    parameters: Metadata[];
  }>({
    provider: 'cartesia',
    parameters: GetDefaultSpeechToTextIfInvalid('cartesia', []),
  });

  // show dialog and confirmation
  const { showDialog, ConfirmDialogComponent } = useConfirmDialog({});
  const [success, setSuccess] = useState(false);

  // Fetch existing deployment on component mount
  useEffect(() => {
    showLoader('block');
    const request = new GetAssistantDeploymentRequest();
    request.setAssistantid(assistantId);
    GetAssistantDebuggerDeployment(
      connectionConfig,
      request,
      ConnectionConfig.WithDebugger({
        authorization: token,
        projectId: projectId,
        userId: authId,
      }),
    )
      .then(response => {
        hideLoader();
        if (response && response.getData()) {
          const deployment = response.getData();
          setExperienceConfig({
            greeting: deployment?.getGreeting(),
            messageOnError: deployment?.getMistake(),
            idealTimeout: deployment?.getIdealtimeout(),
            idealMessage: deployment?.getIdealtimeoutmessage(),
            maxCallDuration: deployment?.getMaxsessionduration(),
          });

          // Audio providers configuration
          if (deployment && deployment.getInputaudio()) {
            const provider = deployment.getInputaudio();
            setVoiceInputEnable(true);
            setAudioInputConfig({
              provider: provider?.getAudioprovider() || 'deepgram',
              parameters: GetDefaultSpeechToTextIfInvalid(
                provider?.getAudioprovider() || 'deepgram',
                GetDefaultMicrophoneConfig(
                  provider?.getAudiooptionsList() || [],
                ),
              ),
            });
          }

          //
          if (deployment && deployment.getOutputaudio()) {
            const provider = deployment?.getOutputaudio();
            setVoiceOutputEnable(true);
            setAudioOutputConfig({
              provider: provider?.getAudioprovider() || 'cartesia',
              parameters: GetDefaultTextToSpeechIfInvalid(
                provider?.getAudioprovider() || 'cartesia',
                GetDefaultSpeakerConfig(provider?.getAudiooptionsList() || []),
              ),
            });
          }
        }
      })
      .catch(x => {
        hideLoader();
        setErrorMessage(
          'Unable to get deployment configurarion for debugger, please try again in sometime.',
        );
        return;
      });
  }, [assistantId, showLoader, token, authId, projectId]);

  // Handle deployment update
  const handleDeployDebugger = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    showLoader('block');
    setErrorMessage('');
    if (voiceInputEnable) {
      if (!audioInputConfig.provider) {
        hideLoader();
        setErrorMessage(
          'Please provide a provider for interpreting input audio of user.',
        );
        return;
      }
      let error = ValidateSpeechToTextIfInvalid(
        audioInputConfig.provider,
        audioInputConfig.parameters,
      );
      if (error) {
        hideLoader();
        setErrorMessage(error);
        return;
      }
    }

    if (voiceOutputEnable) {
      if (!audioOutputConfig.provider) {
        hideLoader();
        setErrorMessage(
          'Please provide a provider for interpreting output audio of user.',
        );
        return;
      }
      let error = ValidateTextToSpeechIfInvalid(
        audioOutputConfig.provider,
        audioOutputConfig.parameters,
      );
      if (error) {
        hideLoader();
        setErrorMessage(error);
        return;
      }
    }

    // audio input is set and working
    const req = new CreateAssistantDeploymentRequest();
    const deployment = new AssistantDebuggerDeployment();

    deployment.setAssistantid(assistantId);
    if (experienceConfig.greeting)
      deployment.setGreeting(experienceConfig.greeting);
    if (experienceConfig?.messageOnError)
      deployment.setMistake(experienceConfig?.messageOnError);
    if (experienceConfig?.idealTimeout)
      deployment.setIdealtimeout(experienceConfig?.idealTimeout);
    if (experienceConfig?.idealMessage)
      deployment.setIdealtimeoutmessage(experienceConfig?.idealMessage);
    if (experienceConfig?.maxCallDuration)
      deployment.setMaxsessionduration(experienceConfig?.maxCallDuration);

    if (voiceInputEnable) {
      const inputAudioProvider = new DeploymentAudioProvider();
      inputAudioProvider.setAudioprovider(audioInputConfig.provider);
      inputAudioProvider.setAudiooptionsList(audioInputConfig.parameters);
      deployment.setInputaudio(inputAudioProvider);
    }

    if (voiceOutputEnable) {
      const outputAudioProvider = new DeploymentAudioProvider();
      outputAudioProvider.setAudioprovider(audioOutputConfig.provider);
      outputAudioProvider.setAudiooptionsList(audioOutputConfig.parameters);
      deployment.setOutputaudio(outputAudioProvider);
    }
    req.setDebugger(deployment);
    CreateAssistantDebuggerDeployment(
      connectionConfig,
      req,
      ConnectionConfig.WithDebugger({
        authorization: token,
        userId: authId,
        projectId: projectId,
      }),
    )
      .then(response => {
        hideLoader();
        if (response?.getData() && response.getSuccess()) {
          toast.success(
            'Assistant deployment config for debugger has been updated successfully.',
          );
          setSuccess(true);
        } else {
          let err =
            response?.getError()?.getHumanmessage() ||
            'Unable to create deployment, please try again';
          toast.error(err);
        }
      })
      .catch(x => {
        hideLoader();
        setErrorMessage(
          'Error while deploying assistant as debugger, please check and try again.',
        );
        return;
      });
  };
  //

  return (
    <>
      <ConfirmDialogComponent />
      <DebuggerDeploymentSuccessDialog
        modalOpen={success}
        setModalOpen={() => {
          setSuccess(false);
          goToDeploymentAssistant(assistantId);
        }}
        assistantId={assistantId}
      />
      <form
        onSubmit={handleDeployDebugger}
        method="POST"
        className="relative flex flex-col flex-1 bg-white dark:bg-gray-900"
      >
        <div className="overflow-auto flex flex-col flex-1 pb-20">
          <ConfigureExperience
            experienceConfig={experienceConfig}
            setExperienceConfig={setExperienceConfig}
          />

          <ConfigureAudioInputProvider
            voiceInputEnable={voiceInputEnable}
            onChangeVoiceInputEnable={setVoiceInputEnable}
            audioInputConfig={audioInputConfig}
            setAudioInputConfig={setAudioInputConfig}
          />
          <ConfigureAudioOutputProvider
            voiceOutputEnable={voiceOutputEnable}
            onChangeVoiceOutputEnable={setVoiceOutputEnable}
            audioOutputConfig={audioOutputConfig}
            setAudioOutputConfig={setAudioOutputConfig}
          />
        </div>

        <PageActionButtonBlock errorMessage={errorMessage}>
          <ICancelButton
            className="px-4 rounded-[2px]"
            onClick={() => {
              showDialog(() => {
                goToDeploymentAssistant(assistantId);
              });
            }}
          >
            Cancel
          </ICancelButton>
          <IBlueBGArrowButton
            type="submit"
            className="px-4 rounded-[2px]"
            isLoading={loading}
          >
            Deploy Debugger
          </IBlueBGArrowButton>
        </PageActionButtonBlock>
      </form>
    </>
  );
};
