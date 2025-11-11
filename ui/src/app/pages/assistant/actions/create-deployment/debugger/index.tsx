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
} from '@rapidaai/react';
import { GetAssistantDebuggerDeployment } from '@rapidaai/react';
import { IBlueBGButton, ICancelButton } from '@/app/components/Form/Button';
import toast from 'react-hot-toast/headless';
import { Helmet } from '@/app/components/Helmet';
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
import { ProviderConfig } from '@/app/components/providers';
import { connectionConfig } from '@/configs';
import { ValidateTextToSpeechIfInvalid } from '@/app/components/providers/text-to-speech/provider';

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
  const { goToDeploymentAssistant } = useGlobalNavigation();
  const { loading, showLoader, hideLoader } = useRapidaStore();
  const { authId, projectId, token } = useCurrentCredential();
  const [errorMessage, setErrorMessage] = useState('');
  const [voiceInputEnable, setVoiceInputEnable] = useState(false);
  const [voiceOutputEnable, setVoiceOutputEnable] = useState(false);

  //   const [personaConfig, setPersonaConfig] = useState<PersonaConfig>({});
  const [experienceConfig, setExperienceConfig] = useState<ExperienceConfig>({
    greeting: '',
    messageOnError: '',
  });

  /**
   * audio input
   */
  const [audioInputConfig, setAudioInputConfig] =
    useState<ProviderConfig | null>(null);

  const onChangeAudioInputProvider = (
    providerId: string,
    providerName: string,
  ) => {
    setAudioInputConfig({
      providerId: providerId,
      provider: providerName,
      parameters: GetDefaultSpeechToTextIfInvalid(
        providerName,
        GetDefaultMicrophoneConfig(
          audioInputConfig?.parameters ? audioInputConfig.parameters : [],
        ),
      ),
    });
  };

  /**
   * audio output
   */
  const [audioOutputConfig, setAudioOutputConfig] =
    useState<ProviderConfig | null>(null);

  const onChangeAudioOuputProvider = (
    providerId: string,
    providerName: string,
  ) => {
    setAudioOutputConfig({
      providerId: providerId,
      provider: providerName,
      parameters: GetDefaultTextToSpeechIfInvalid(
        providerName,
        GetDefaultSpeakerConfig(
          audioOutputConfig ? audioOutputConfig.parameters : [],
        ),
      ),
    });
  };

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
            greeting: deployment?.getGreeting() || '',
            messageOnError: deployment?.getMistake() || '',
          });

          // Audio providers configuration
          if (deployment && deployment.getInputaudio()) {
            const provider = deployment.getInputaudio();
            setVoiceInputEnable(true);
            setAudioInputConfig({
              providerId:
                provider?.getAudioproviderid() || '2123891723608588082',
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
              providerId:
                provider?.getAudioproviderid() || '2123891723608588082',
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
    if (audioInputConfig != null) {
      if (!audioInputConfig.provider || !audioInputConfig.providerId) {
        hideLoader();
        setErrorMessage(
          'Please provide a provider for interpreting input audio of user.',
        );
        return;
      }

      if (
        !ValidateSpeechToTextIfInvalid(
          audioInputConfig.provider,
          audioInputConfig.parameters,
        )
      ) {
        hideLoader();
        setErrorMessage('Please provide a valid speech to text options.');
        return;
      }
    }

    if (audioOutputConfig != null) {
      if (!audioOutputConfig.provider || !audioOutputConfig.providerId) {
        hideLoader();
        setErrorMessage(
          'Please provide a provider for interpreting output audio of user.',
        );
        return;
      }

      if (
        !ValidateTextToSpeechIfInvalid(
          audioOutputConfig.provider,
          audioOutputConfig.parameters,
        )
      ) {
        hideLoader();
        setErrorMessage('Please provide a valid text to speech options.');
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

    if (audioInputConfig) {
      const inputAudioProvider = new DeploymentAudioProvider();
      inputAudioProvider.setId(audioInputConfig.providerId);
      inputAudioProvider.setAudioprovider(audioInputConfig.provider);
      inputAudioProvider.setAudiooptionsList(audioInputConfig.parameters);
      inputAudioProvider.setAudioproviderid(audioInputConfig.providerId);
      deployment.setInputaudio(inputAudioProvider);
    }

    if (audioOutputConfig) {
      const outputAudioProvider = new DeploymentAudioProvider();
      outputAudioProvider.setId(audioOutputConfig.providerId);
      outputAudioProvider.setAudioprovider(audioOutputConfig.provider);
      outputAudioProvider.setAudiooptionsList(audioOutputConfig.parameters);
      outputAudioProvider.setAudioproviderid(audioOutputConfig.providerId);
      deployment.setOutputaudio(outputAudioProvider);
    }

    req.setDebugger(deployment);
    CreateAssistantDebuggerDeployment(connectionConfig, req, {
      authorization: token,
      'x-auth-id': authId,
      'x-project-id': projectId,
    })
      .then(response => {
        hideLoader();
        if (response?.getData() && response.getSuccess()) {
          toast.success(
            'Assistant deployment config for debugger has been updated successfully.',
          );
          goToDeploymentAssistant(assistantId);
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
    <form
      onSubmit={handleDeployDebugger}
      method="POST"
      className="relative flex flex-col flex-1"
    >
      <div className="bg-white dark:bg-gray-900 overflow-auto flex flex-col flex-1 pb-20">
        <ConfigureExperience
          experienceConfig={experienceConfig}
          setExperienceConfig={setExperienceConfig}
        />

        <ConfigureAudioInputProvider
          onChangeProvider={onChangeAudioInputProvider}
          onChangeConfig={setAudioInputConfig}
          config={audioInputConfig}
          voiceInputEnable={voiceInputEnable}
          onChangeVoiceInputEnable={setVoiceInputEnable}
        />
        <ConfigureAudioOutputProvider
          onChangeProvider={onChangeAudioOuputProvider}
          onChangeConfig={setAudioOutputConfig}
          config={audioOutputConfig}
          voiceOutputEnable={voiceOutputEnable}
          onChangeVoiceOutputEnable={setVoiceOutputEnable}
        />
      </div>

      <PageActionButtonBlock errorMessage={errorMessage}>
        <ICancelButton
          className="px-4 rounded-[2px]"
          onClick={() => {
            goToDeploymentAssistant(assistantId);
          }}
        >
          Cancel
        </ICancelButton>
        <IBlueBGButton
          type="submit"
          className="px-4 rounded-[2px]"
          isLoading={loading}
        >
          Deploy Debugger
        </IBlueBGButton>
      </PageActionButtonBlock>
    </form>
  );
};
