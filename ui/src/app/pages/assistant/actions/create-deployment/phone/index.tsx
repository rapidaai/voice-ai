import {
  IBlueBGArrowButton,
  ICancelButton,
} from '@/app/components/form/button';
import { useRapidaStore } from '@/hooks';
import { FC, useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import {
  AssistantPhoneDeployment,
  ConnectionConfig,
  CreateAssistantDeploymentRequest,
  CreateAssistantPhoneDeployment,
  DeploymentAudioProvider,
  GetAssistantDeploymentRequest,
  Metadata,
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
import { Helmet } from '@/app/components/helmet';
import { GetCartesiaDefaultOptions } from '@/app/components/providers/text-to-speech/cartesia';
import {
  GetDefaultMicrophoneConfig,
  GetDefaultSpeechToTextIfInvalid,
  ValidateSpeechToTextIfInvalid,
} from '@/app/components/providers/speech-to-text/provider';
import {
  GetDefaultSpeakerConfig,
  ValidateTextToSpeechIfInvalid,
} from '@/app/components/providers/text-to-speech/provider';
import { PageActionButtonBlock } from '@/app/components/blocks/page-action-button-block';
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

  /**
   * enable for voice
   */
  const [voiceInputEnable, setVoiceInputEnable] = useState(true);
  const [voiceOutputEnable, setVoiceOutputEnable] = useState(true);
  const [experienceConfig, setExperienceConfig] = useState<ExperienceConfig>({
    greeting: undefined,
    messageOnError: undefined,
    idealTimeout: '5000',
    idealMessage: 'Are you there?',
    maxCallDuration: '10000',
  });

  const [telephonyConfig, setTelephonyConfig] = useState<{
    provider: string;
    parameters: Metadata[];
  }>({
    provider: 'twilio',
    parameters: [],
  });

  /**
   * audio input
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

  /**
   * audio output
   */
  const [audioOutputConfig, setAudioOutputConfig] = useState<{
    provider: string;
    parameters: Metadata[];
  }>({
    provider: 'cartesia',
    parameters: GetCartesiaDefaultOptions(GetDefaultSpeakerConfig()),
  });

  const onChangTelephonyProvider = (providerName: string) => {
    setTelephonyConfig({
      provider: providerName,
      parameters: [],
    });
  };

  const onChangeTelephonyParameter = (parameters: Metadata[]) => {
    if (telephonyConfig) setTelephonyConfig({ ...telephonyConfig, parameters });
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
            greeting: deployment?.getGreeting(),
            messageOnError: deployment?.getMistake(),
            idealTimeout: deployment?.getIdealtimeout(),
            idealMessage: deployment?.getIdealtimeoutmessage(),
            maxCallDuration: deployment?.getMaxsessionduration(),
          });

          // Audio providers configuration

          if (deployment && deployment.getPhoneprovidername()) {
            setTelephonyConfig({
              provider: deployment?.getPhoneprovidername() || '',
              parameters: deployment?.getPhoneoptionsList() || [],
            });
          }

          if (deployment && deployment.getInputaudio()) {
            const provider = deployment.getInputaudio();
            setVoiceInputEnable(true);
            setAudioInputConfig({
              provider: provider?.getAudioprovider() || '',
              parameters: provider?.getAudiooptionsList() || [],
            });
          }

          if (deployment && deployment.getOutputaudio()) {
            const provider = deployment?.getOutputaudio();
            setVoiceOutputEnable(true);
            setAudioOutputConfig({
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

    let error = ValidateTelephonyOptions(
      telephonyConfig.provider,
      telephonyConfig.parameters,
    );
    if (!error) {
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

    if (!audioInputConfig.provider) {
      hideLoader();
      setErrorMessage(
        'Please provide a provider for interpreting input audio of user.',
      );
      return;
    }

    let sttError = ValidateSpeechToTextIfInvalid(
      audioInputConfig.provider,
      audioInputConfig.parameters,
    );
    if (sttError) {
      hideLoader();
      setErrorMessage(sttError);
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
    if (!audioOutputConfig.provider) {
      hideLoader();
      setErrorMessage(
        'Please provide a provider for interpreting output audio of user.',
      );
      return;
    }

    let ttsErr = ValidateTextToSpeechIfInvalid(
      audioOutputConfig.provider,
      audioOutputConfig.parameters,
    );
    if (ttsErr) {
      hideLoader();
      setErrorMessage(ttsErr);
      return;
    }

    const req = new CreateAssistantDeploymentRequest();
    const deployment = new AssistantPhoneDeployment();
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

    if (telephonyConfig) {
      deployment.setPhoneoptionsList(telephonyConfig.parameters);
      deployment.setPhoneprovidername(telephonyConfig.provider);
    }

    if (audioInputConfig) {
      const inputAudioProvider = new DeploymentAudioProvider();
      inputAudioProvider.setAudioprovider(audioInputConfig.provider);
      inputAudioProvider.setAudiooptionsList(audioInputConfig.parameters);
      deployment.setInputaudio(inputAudioProvider);
    }

    if (audioOutputConfig) {
      const outputAudioProvider = new DeploymentAudioProvider();
      outputAudioProvider.setAudioprovider(audioOutputConfig.provider);
      outputAudioProvider.setAudiooptionsList(audioOutputConfig.parameters);
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
      className="relative flex flex-col flex-1 bg-white dark:bg-gray-900"
    >
      <div className="overflow-auto flex flex-col flex-1 pb-20">
        <TelephonyProvider
          provider={telephonyConfig.provider}
          parameters={telephonyConfig.parameters}
          onChangeProvider={onChangTelephonyProvider}
          onChangeParameter={onChangeTelephonyParameter}
        />
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
            goToDeploymentAssistant(assistantId);
          }}
        >
          Cancel
        </ICancelButton>
        <IBlueBGArrowButton
          type="submit"
          className="px-4 rounded-[2px]"
          isLoading={loading}
          disabled={loading}
        >
          Deploy Phone
        </IBlueBGArrowButton>
      </PageActionButtonBlock>
    </form>
  );
};
