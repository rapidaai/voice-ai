import {
  IBlueBGArrowButton,
  ICancelButton,
} from '@/app/components/form/button';
import { useRapidaStore } from '@/hooks';
import { FC, useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import {
  AssistantWebpluginDeployment,
  ConnectionConfig,
  CreateAssistantDeploymentRequest,
  DeploymentAudioProvider,
  GetAssistantDeploymentRequest,
  Metadata,
} from '@rapidaai/react';
import { GetAssistantWebpluginDeployment } from '@rapidaai/react';
import { useCurrentCredential } from '@/hooks/use-credential';
import { CreateAssistantWebpluginDeployment } from '@rapidaai/react';
import {
  ConfigureExperience,
  WebWidgetExperienceConfig,
} from '@/app/pages/assistant/actions/create-deployment/web-plugin/configure-experience';
import { ConfigureAudioOutputProvider } from '@/app/pages/assistant/actions/create-deployment/commons/configure-audio-output';
import { ConfigureAudioInputProvider } from '@/app/pages/assistant/actions/create-deployment/commons/configure-audio-input';
import {
  ConfigureFeature,
  FeatureConfig,
} from '@/app/pages/assistant/actions/create-deployment/web-plugin/configure-feature';
import toast from 'react-hot-toast/headless';
import { Helmet } from '@/app/components/helmet';
import { ValidateTextToSpeechIfInvalid } from '@/app/components/providers/text-to-speech/provider';
import {
  GetDefaultMicrophoneConfig,
  GetDefaultSpeechToTextIfInvalid,
  ValidateSpeechToTextIfInvalid,
} from '@/app/components/providers/speech-to-text/provider';
import { PageActionButtonBlock } from '@/app/components/blocks/page-action-button-block';
import { connectionConfig } from '@/configs';
import { AssistantWebwidgetDeploymentDialog } from '@/app/components/base/modal/assistant-instruction-modal';

/**
 *
 * @returns
 */
export function ConfigureAssistantWebDeploymentPage() {
  const { assistantId } = useParams();
  return (
    <>
      <Helmet title="Configure web-plugin deployment" />
      {assistantId && (
        <ConfigureAssistantWebDeployment assistantId={assistantId} />
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
const ConfigureAssistantWebDeployment: FC<{ assistantId: string }> = ({
  assistantId,
}) => {
  /**
   * global navigation
   */
  const { goToDeploymentAssistant } = useGlobalNavigation();

  /**
   * global loading state
   */
  const { loading, showLoader, hideLoader } = useRapidaStore();

  /**
   * authentication
   */
  const { authId, projectId, token } = useCurrentCredential();

  /**
   * error message
   */
  const [errorMessage, setErrorMessage] = useState('');

  /**
   * Post create instruction
   */
  const [showInstruction, setShowInstruction] = useState(false);
  const [deploymentId, setDeploymentId] = useState<string | null>(null);
  /**
   * voice io
   */
  const [voiceInputEnable, setVoiceInputEnable] = useState(false);
  const [voiceOutputEnable, setVoiceOutputEnable] = useState(false);

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

  const [experienceConfig, setExperienceConfig] =
    useState<WebWidgetExperienceConfig>({
      greeting: undefined,
      messageOnError: undefined,
      idealTimeout: '5000',
      idealMessage: 'Are you there?',
      maxCallDuration: '10000',
      suggestions: [],
    });

  const [featureConfig, setFeatureConfig] = useState<FeatureConfig>({
    qAListing: false,
    productCatalog: false,
    blogPost: false,
  });

  useEffect(() => {
    showLoader('block');
    const req = new GetAssistantDeploymentRequest();
    req.setAssistantid(assistantId);
    GetAssistantWebpluginDeployment(
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
        if (response && response.getData()) {
          const deployment = response.getData();
          setDeploymentId(deployment?.getId()!);
          setExperienceConfig({
            greeting: deployment?.getGreeting(),
            suggestions: deployment?.getSuggestionList() || [],
            messageOnError: deployment?.getMistake(),
            idealTimeout: deployment?.getIdealtimeout(),
            idealMessage: deployment?.getIdealtimeoutmessage(),
            maxCallDuration: deployment?.getMaxsessionduration(),
          });

          // Audio providers configuration

          if (deployment && deployment.getInputaudio()) {
            const provider = deployment.getInputaudio();
            setAudioInputConfig({
              provider: provider?.getAudioprovider() || '',
              parameters: provider?.getAudiooptionsList() || [],
            });
          }

          if (deployment && deployment.getOutputaudio()) {
            const provider = deployment?.getOutputaudio();
            setAudioOutputConfig({
              provider: provider?.getAudioprovider() || '',
              parameters: provider?.getAudiooptionsList() || [],
            });
          }

          // Feature configuration
          if (deployment) {
            setFeatureConfig({
              qAListing: deployment.getHelpcenterenabled(),
              productCatalog: deployment.getProductcatalogenabled(),
              blogPost: deployment.getArticlecatalogenabled(),
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
  const handleDeployWebPlugin = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    showLoader('block');
    // setting error message to be empty when there is no data to submit
    setErrorMessage('');
    // validation

    if (!experienceConfig?.greeting) {
      hideLoader();
      setErrorMessage('Please provide a greeting for the assistant.');
      return;
    }

    // voice input can be disabled for web widget
    if (voiceInputEnable)
      if (audioInputConfig != null) {
        if (!audioInputConfig.provider) {
          hideLoader();
          setErrorMessage(
            'Please provide a provider for interpreting input audio of user.',
          );
          return;
        }

        let err = ValidateSpeechToTextIfInvalid(
          audioInputConfig.provider,
          audioInputConfig.parameters,
        );
        if (err) {
          hideLoader();
          setErrorMessage(err);
          return;
        }
      }

    //   voice output can be disabled for web widget
    if (voiceOutputEnable)
      if (audioOutputConfig != null) {
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
    const req = new CreateAssistantDeploymentRequest();
    const webDeployment = new AssistantWebpluginDeployment();
    webDeployment.setAssistantid(assistantId);
    if (experienceConfig.greeting)
      webDeployment.setGreeting(experienceConfig.greeting);
    if (experienceConfig?.messageOnError)
      webDeployment.setMistake(experienceConfig?.messageOnError);
    if (experienceConfig?.idealTimeout)
      webDeployment.setIdealtimeout(experienceConfig?.idealTimeout);
    if (experienceConfig?.idealMessage)
      webDeployment.setIdealtimeoutmessage(experienceConfig?.idealMessage);
    if (experienceConfig?.maxCallDuration)
      webDeployment.setMaxsessionduration(experienceConfig?.maxCallDuration);

    webDeployment.setSuggestionList(experienceConfig.suggestions);
    webDeployment.setHelpcenterenabled(featureConfig.qAListing);
    webDeployment.setProductcatalogenabled(featureConfig.productCatalog);
    webDeployment.setArticlecatalogenabled(featureConfig.blogPost);
    webDeployment.setUploadfileenabled(false); // Not provided in the input, set to false by default

    if (voiceInputEnable) {
      const inputAudioProvider = new DeploymentAudioProvider();
      inputAudioProvider.setAudioprovider(audioInputConfig.provider);
      inputAudioProvider.setAudiooptionsList(audioInputConfig.parameters);
      webDeployment.setInputaudio(inputAudioProvider);
    }

    if (voiceOutputEnable) {
      const outputAudioProvider = new DeploymentAudioProvider();
      outputAudioProvider.setAudioprovider(audioOutputConfig.provider);
      outputAudioProvider.setAudiooptionsList(audioOutputConfig.parameters);
      webDeployment.setOutputaudio(outputAudioProvider);
    }

    req.setPlugin(webDeployment);
    CreateAssistantWebpluginDeployment(
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
          if (deploymentId) {
            toast.success(
              'Assistant deployment config for phone call has been updated successfully.',
            );
            goToDeploymentAssistant(assistantId);
            return;
          }
          setShowInstruction(true);
        } else {
          setErrorMessage(
            response?.getError()?.getHumanmessage() ||
              'Error while deploying assistant as web plugin, please check and try again.',
          );
        }
      })
      .catch(err => {
        hideLoader();
        if (err) {
          setErrorMessage(
            err.message ||
              'Error while deploying assistant as web plugin, please check and try again.',
          );
          return;
        }
      });
  };
  return (
    <form
      onSubmit={handleDeployWebPlugin}
      className="relative flex flex-col flex-1 bg-white dark:bg-gray-900"
      method="POST"
    >
      <AssistantWebwidgetDeploymentDialog
        assistantId={assistantId}
        setModalOpen={() => {
          setShowInstruction(!showInstruction);
          goToDeploymentAssistant(assistantId);
        }}
        modalOpen={showInstruction}
      />
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
        <ConfigureFeature
          onConfigChange={setFeatureConfig}
          config={featureConfig}
        />

        {/*  */}
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
        >
          Deploy web widget
        </IBlueBGArrowButton>
      </PageActionButtonBlock>
    </form>
  );
};
