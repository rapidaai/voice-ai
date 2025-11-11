import { IBlueBGButton, ICancelButton } from '@/app/components/Form/Button';
import { useRapidaStore } from '@/hooks';
import { FC, useCallback, useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import {
  AssistantPhoneDeployment,
  ConnectionConfig,
  CreateAssistantDeploymentRequest,
  CreateAssistantPhoneDeployment,
  DeploymentAudioProvider,
  GetAssistantDeploymentRequest,
} from '@rapidaai/react';
import { GetAssistantPhoneDeployment } from '@rapidaai/react';
import { useCurrentCredential } from '@/hooks/use-credential';
import {
  ConfigureExperience,
  ExperienceConfig,
} from '@/app/pages/assistant/actions/create-deployment/commons/configure-experience';
import { ConfigureAudioInputProvider } from '@/app/pages/assistant/actions/create-deployment/commons/configure-audio-input';
import { ConfigureAudioOutputProvider } from '@/app/pages/assistant/actions/create-deployment/commons/configure-audio-output';
import toast from 'react-hot-toast/headless';
import { Helmet } from '@/app/components/Helmet';
import { GetCartesiaDefaultOptions } from '@/app/components/providers/text-to-speech/cartesia';
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
import { ProviderConfig } from '@/app/components/providers';
import { connectionConfig } from '@/configs';
import {
  TelephonyProvider,
  ValidateTelephonyOptions,
} from '@/app/components/providers/telephony';
export function ConfigureAssistantCallDeploymentPage() {
  const { assistantId } = useParams();
  return (
    <>
      <Helmet title="Configure phone deployment" />
      {assistantId && (
        <ConfigureAssistantCallDeployment assistantId={assistantId} />
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
const ConfigureAssistantCallDeployment: FC<{ assistantId: string }> = ({
  assistantId,
}) => {
  const { goToDeploymentAssistant } = useGlobalNavigation();
  const { loading, showLoader, hideLoader } = useRapidaStore();
  const { authId, projectId, token } = useCurrentCredential();
  const [errorMessage, setErrorMessage] = useState('');
  const [voiceInputEnable, setVoiceInputEnable] = useState(false);
  const [voiceOutputEnable, setVoiceOutputEnable] = useState(false);
  const [experienceConfig, setExperienceConfig] = useState<ExperienceConfig>({
    greeting: '',
    messageOnError: '',
  });

  const [telephonyConfig, setTelephonyConfig] = useState<ProviderConfig | null>(
    null,
  );
  const onChangTelephonyProvider = (
    providerId: string,
    providerName: string,
  ) => {
    setTelephonyConfig({
      providerId: providerId,
      provider: providerName,
      parameters: [],
    });
  };

  /**
   * audio input
   */
  const [audioInputConfig, setAudioInputConfig] = useState<ProviderConfig>({
    providerId: '2123891723608588082',
    provider: 'deepgram',
    parameters: GetDefaultSpeechToTextIfInvalid(
      'deepgram',
      GetDefaultMicrophoneConfig(),
    ),
  });

  const onChangeAudioInputProvider = (
    providerId: string,
    providerName: string,
  ) => {
    setAudioInputConfig({
      providerId: providerId,
      provider: providerName,
      parameters: GetDefaultSpeechToTextIfInvalid(
        providerName,
        GetDefaultMicrophoneConfig(audioInputConfig.parameters),
      ),
    });
  };

  /**
   * audio output
   */
  const [audioOutputConfig, setAudioOutputConfig] = useState<ProviderConfig>({
    providerId: '2123891723608588082',
    provider: 'cartesia',
    parameters: GetCartesiaDefaultOptions(GetDefaultSpeakerConfig()),
  });

  const onChangeAudioOuputProvider = (
    providerId: string,
    providerName: string,
  ) => {
    setAudioOutputConfig({
      providerId: providerId,
      provider: providerName,
      parameters: GetDefaultTextToSpeechIfInvalid(
        providerName,
        GetDefaultSpeakerConfig(audioOutputConfig.parameters),
      ),
    });
  };

  // Fetch existing deployment on component mount
  useEffect(() => {
    showLoader('block');
    const request = new GetAssistantDeploymentRequest();
    request.setAssistantid(assistantId);
    GetAssistantPhoneDeployment(
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
            greeting: deployment?.getGreeting() || '',
            messageOnError: deployment?.getMistake() || '',
          });

          // Audio providers configuration

          if (deployment && deployment.getPhoneprovidername()) {
            setTelephonyConfig({
              providerId: deployment.getPhoneproviderid() || '',
              provider: deployment?.getPhoneprovidername() || '',
              parameters: deployment?.getPhoneoptionsList() || [],
            });
          }

          if (deployment && deployment.getInputaudio()) {
            const provider = deployment.getInputaudio();
            setVoiceInputEnable(true);
            setAudioInputConfig({
              providerId: provider?.getAudioproviderid() || '',
              provider: provider?.getAudioprovider() || '',
              parameters: provider?.getAudiooptionsList() || [],
            });
          }

          if (deployment && deployment.getOutputaudio()) {
            const provider = deployment?.getOutputaudio();
            setVoiceOutputEnable(true);
            setAudioOutputConfig({
              providerId: provider?.getAudioproviderid() || '',
              provider: provider?.getAudioprovider() || '',
              parameters: provider?.getAudiooptionsList() || [],
            });
          }
        }
      })
      .catch(err => {
        hideLoader();
        if (err) {
          setErrorMessage(err.message || 'Failed to deploy api');
          toast.error(
            err.message ||
              'Error while deploying assistant as phone call, please check and try again.',
          );
          return;
        }
      });

    //   afterGetAssistantPhoneDeployment,
    //
  }, [assistantId, showLoader, token, authId, projectId]);

  // Handle deployment update
  const handleDeployPhone = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();

    //
    showLoader('block');
    // setting error message to be empty when there is no data to submit
    setErrorMessage('');

    if (telephonyConfig == null) {
      hideLoader();
      setErrorMessage(
        'Please provide a valid telephony providers for phone call.',
      );
      return;
    }

    if (
      !ValidateTelephonyOptions(
        telephonyConfig.provider,
        telephonyConfig.parameters,
      )
    ) {
      hideLoader();
      setErrorMessage(
        'Please provide a valid telephony providers for phone call.',
      );
      return;
    }

    if (audioInputConfig == null) {
      hideLoader();
      setErrorMessage('Please provide a valid speech to text for phone voice');
      return;
    }

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

    if (audioOutputConfig == null) {
      hideLoader();
      setErrorMessage(
        'Please provide a provider for interpreting output audio of user.',
      );
      return;
    }

    // audio input is set and working
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
      setErrorMessage('Please provide a valid text to speech  options.');
      return;
    }

    const req = new CreateAssistantDeploymentRequest();
    const deployment = new AssistantPhoneDeployment();
    deployment.setAssistantid(assistantId);
    deployment.setGreeting(experienceConfig.greeting);
    deployment.setMistake(experienceConfig.messageOnError);

    if (telephonyConfig) {
      deployment.setPhoneoptionsList(telephonyConfig.parameters);
      deployment.setPhoneproviderid(telephonyConfig.providerId);
      deployment.setPhoneprovidername(telephonyConfig.provider);
    }

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

    req.setPhone(deployment);

    CreateAssistantPhoneDeployment(
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
            'Assistant deployment config for phone call has been updated successfully.',
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
              'Error while deploying assistant as phone call, please check and try again.',
          );
          return;
        }
      });

    //  hideLoader();
  };

  return (
    <form
      onSubmit={handleDeployPhone}
      method="POST"
      className="relative flex flex-col flex-1"
    >
      <div className="bg-white dark:bg-gray-900 overflow-auto flex flex-col flex-1 pb-20">
        <TelephonyProvider
          onConfigChange={setTelephonyConfig}
          config={telephonyConfig}
          onChangeProvider={onChangTelephonyProvider}
        />
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
          disabled={loading}
        >
          Deploy Phone
        </IBlueBGButton>
      </PageActionButtonBlock>
    </form>
  );
};
