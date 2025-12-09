import { useRapidaStore } from '@/hooks';
import { useCurrentCredential } from '@/hooks/use-credential';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { FC, useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import {
  AssistantDebuggerDeployment,
  ConnectionConfig,
  CreateAssistantApiDeployment,
  CreateAssistantDeploymentRequest,
  DeploymentAudioProvider,
  GetAssistantDeploymentRequest,
  Metadata,
} from '@rapidaai/react';
import { GetAssistantApiDeployment } from '@rapidaai/react';
import {
  ConfigureExperience,
  ExperienceConfig,
} from '@/app/pages/assistant/actions/create-deployment/commons/configure-experience';
import { ConfigureAudioOutputProvider } from '@/app/pages/assistant/actions/create-deployment/commons/configure-audio-output';
import { ConfigureAudioInputProvider } from '@/app/pages/assistant/actions/create-deployment/commons/configure-audio-input';
import {
  ICancelButton,
  IBlueBGArrowButton,
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
  ValidateTextToSpeechIfInvalid,
} from '@/app/components/providers/text-to-speech/provider';
import { PageActionButtonBlock } from '@/app/components/blocks/page-action-button-block';
import { connectionConfig } from '@/configs';
import { useConfirmDialog } from '@/app/pages/assistant/actions/hooks/use-confirmation';

/**
 *
 * @returns
 */
export function ConfigureAssistantApiDeploymentPage() {
  const { assistantId } = useParams();
  return (
    <>
      <Helmet title="Configure api deployment" />
      {assistantId && (
        <ConfigureAssistantApiDeployment assistantId={assistantId} />
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
const ConfigureAssistantApiDeployment: FC<{ assistantId: string }> = ({
  assistantId,
}) => {
  const { goToDeploymentAssistant } = useGlobalNavigation();
  const { loading, showLoader, hideLoader } = useRapidaStore();
  const { authId, projectId, token } = useCurrentCredential();
  const [errorMessage, setErrorMessage] = useState('');
  const { showDialog, ConfirmDialogComponent } = useConfirmDialog({});
  /**
   * if voice is enabled
   */
  const [voiceInputEnable, setVoiceInputEnable] = useState(false);
  const [voiceOutputEnable, setVoiceOutputEnable] = useState(false);

  /**
   * voice experience
   */
  const [experienceConfig, setExperienceConfig] = useState<ExperienceConfig>({
    greeting: undefined,
    messageOnError: undefined,
    idealTimeout: '5000',
    idealMessage: 'Are you there?',
    maxCallDuration: '10000',
  });

  /**
   * io for voice
   */
  const [audioInputConfig, setAudioInputConfig] = useState<{
    provider: string;
    parameters: Metadata[];
  }>({
    provider: 'deepgram',
    parameters: GetDefaultSpeechToTextIfInvalid(
      'deepgram',
      GetDefaultMicrophoneConfig(),
    ),
  });
  const [audioOutputConfig, setAudioOutputConfig] = useState<{
    provider: string;
    parameters: Metadata[];
  }>({
    provider: 'cartesia',
    parameters: GetDefaultSpeechToTextIfInvalid(
      'cartesia',
      GetDefaultMicrophoneConfig(),
    ),
  });

  // Fetch existing deployment on component mount
  useEffect(() => {
    showLoader('block');
    const request = new GetAssistantDeploymentRequest();
    request.setAssistantid(assistantId);
    GetAssistantApiDeployment(
      connectionConfig,
      request,
      ConnectionConfig.WithDebugger({
        authorization: token,
        userId: authId,
        projectId: projectId,
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
      .catch(err => {
        hideLoader();
        if (err) {
          setErrorMessage(
            err.message || 'Failed to fetch deployment configuration',
          );
          return;
        }
      });
  }, [assistantId, showLoader, token, authId, projectId]);

  // Handle deployment update
  const handleDeployApi = (event: React.FormEvent<HTMLFormElement>) => {
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
        setErrorMessage('Please provide a valid speech to text options.');
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
        setErrorMessage('Please provide a valid text to speech options.');
        return;
      }
    }

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

    req.setApi(deployment);
    CreateAssistantApiDeployment(
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
            'Assistant deployment config for api has been updated successfully.',
          );
          goToDeploymentAssistant(assistantId);
        } else {
          let err =
            response?.getError()?.getHumanmessage() ||
            'Unable to create deployment, please try again';
          toast.error(err);
        }
      })
      .catch(err => {
        hideLoader();
        if (err) {
          setErrorMessage(err.message || 'Failed to deploy api');
          toast.error(
            err.message ||
              'Error while deploying assistant as api, please check and try again.',
          );
          return;
        }
      });
  };

  return (
    <>
      <ConfirmDialogComponent />
      <form
        onSubmit={handleDeployApi}
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
            Deploy Api
          </IBlueBGArrowButton>
        </PageActionButtonBlock>
      </form>
    </>
  );
};
