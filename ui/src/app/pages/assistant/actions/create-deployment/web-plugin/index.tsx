import { IBlueBGButton, ICancelButton } from '@/app/components/Form/Button';
import { useRapidaStore } from '@/hooks';
import { FC, useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import {
  AssistantWebpluginDeployment,
  ConnectionConfig,
  Content,
  CreateAssistantDeploymentRequest,
  DeploymentAudioProvider,
  GetAssistantDeploymentRequest,
} from '@rapidaai/react';
import { GetAssistantWebpluginDeployment } from '@rapidaai/react';
import { useCurrentCredential } from '@/hooks/use-credential';
import { CreateAssistantWebpluginDeployment } from '@rapidaai/react';
import {
  ConfigurePersona,
  PersonaConfig,
} from '@/app/pages/assistant/actions/create-deployment/web-plugin/configure-persona';
import {
  ConfigureExperience,
  ExperienceConfig,
} from '@/app/pages/assistant/actions/create-deployment/web-plugin/configure-experience';
import { ConfigureAudioOutputProvider } from '@/app/pages/assistant/actions/create-deployment/commons/configure-audio-output';
import { ConfigureAudioInputProvider } from '@/app/pages/assistant/actions/create-deployment/commons/configure-audio-input';
import {
  ConfigureFeature,
  FeatureConfig,
} from '@/app/pages/assistant/actions/create-deployment/web-plugin/configure-feature';
import toast from 'react-hot-toast/headless';
import { Helmet } from '@/app/components/Helmet';
import {
  GetDefaultSpeakerConfig,
  GetDefaultTextToSpeechIfInvalid,
  ValidateTextToSpeechIfInvalid,
} from '@/app/components/providers/text-to-speech/provider';
import {
  GetDefaultMicrophoneConfig,
  GetDefaultSpeechToTextIfInvalid,
  ValidateSpeechToTextIfInvalid,
} from '@/app/components/providers/speech-to-text/provider';
import { PageActionButtonBlock } from '@/app/components/blocks/page-action-button-block';
import { ProviderConfig } from '@/app/components/providers';
import { connectionConfig } from '@/configs';
import { Rocket } from 'lucide-react';
import { AssistantWebwidgetDeploymentDialog } from '@/app/components/base/modal/assistant-instruction-modal';

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
  const { goBack, goToDeploymentAssistant } = useGlobalNavigation();
  const { loading, showLoader, hideLoader } = useRapidaStore();
  const { authId, projectId, token } = useCurrentCredential();
  const [errorMessage, setErrorMessage] = useState('');
  const [voiceInputEnable, setVoiceInputEnable] = useState(false);
  const [voiceOutputEnable, setVoiceOutputEnable] = useState(false);
  const [showInstruction, setShowInstruction] = useState(false);
  //   data
  //   id of deployment
  const [deploymentId, setDeploymentId] = useState<string | null>(null);
  const [personaConfig, setPersonaConfig] = useState<PersonaConfig>({});
  const [experienceConfig, setExperienceConfig] = useState<ExperienceConfig>({
    greeting: '',
    suggestions: [],
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
        GetDefaultMicrophoneConfig(audioInputConfig?.parameters),
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
        GetDefaultSpeakerConfig(audioOutputConfig?.parameters),
      ),
    });
  };

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
            greeting: deployment?.getGreeting() || '',
            suggestions: deployment?.getSuggestionList() || [],
            messageOnError: deployment?.getMistake() || '',
          });

          // Persona configuration
          setPersonaConfig({
            name: deployment?.getName(),
            avatarUrl: deployment?.getUrl(),
          });

          // Audio providers configuration

          if (deployment && deployment.getInputaudio()) {
            const provider = deployment.getInputaudio();
            setAudioInputConfig({
              providerId: provider?.getAudioproviderid() || '',
              provider: provider?.getAudioprovider() || '',
              parameters: provider?.getAudiooptionsList() || [],
            });
          }

          if (deployment && deployment.getOutputaudio()) {
            const provider = deployment?.getOutputaudio();
            setAudioOutputConfig({
              providerId: provider?.getAudioproviderid() || '',
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

    if (!personaConfig.avatar?.file && !personaConfig.avatarUrl) {
      hideLoader();
      setErrorMessage('Please provide a valid icon for the web plugin.');
      return;
    }
    // validation

    if (!experienceConfig?.greeting) {
      hideLoader();
      setErrorMessage('Please provide a greeting for the assistant.');
      return;
    }

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
    const req = new CreateAssistantDeploymentRequest();
    const webDeployment = new AssistantWebpluginDeployment();
    webDeployment.setAssistantid(assistantId);
    webDeployment.setName(personaConfig.name || '');
    webDeployment.setGreeting(experienceConfig.greeting);
    webDeployment.setMistake(experienceConfig.messageOnError);
    webDeployment.setSuggestionList(experienceConfig.suggestions);
    webDeployment.setHelpcenterenabled(featureConfig.qAListing);
    webDeployment.setProductcatalogenabled(featureConfig.productCatalog);
    webDeployment.setArticlecatalogenabled(featureConfig.blogPost);
    webDeployment.setUploadfileenabled(false); // Not provided in the input, set to false by default

    if (audioInputConfig) {
      const inputAudioProvider = new DeploymentAudioProvider();
      inputAudioProvider.setId(audioInputConfig.providerId);
      inputAudioProvider.setAudioprovider(audioInputConfig.provider);
      inputAudioProvider.setAudiooptionsList(audioInputConfig.parameters);
      inputAudioProvider.setAudioproviderid(audioInputConfig.providerId);
      webDeployment.setInputaudio(inputAudioProvider);
    }

    if (audioOutputConfig) {
      const outputAudioProvider = new DeploymentAudioProvider();
      outputAudioProvider.setId(audioOutputConfig.providerId);
      outputAudioProvider.setAudioprovider(audioOutputConfig.provider);
      outputAudioProvider.setAudiooptionsList(audioOutputConfig.parameters);
      outputAudioProvider.setAudioproviderid(audioOutputConfig.providerId);
      webDeployment.setOutputaudio(outputAudioProvider);
    }

    if (personaConfig.avatar && personaConfig.avatar.file) {
      const cntn = new Content();
      cntn.setContent(personaConfig.avatar.file);
      cntn.setName(personaConfig.avatar.name);
      cntn.setContenttype(personaConfig.avatar.type);
      webDeployment.setRaw(cntn);
    } else if (personaConfig.avatarUrl) {
      webDeployment.setUrl(personaConfig.avatarUrl);
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
  };
  return (
    <form
      onSubmit={handleDeployWebPlugin}
      className="relative flex flex-col flex-1"
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
      <div className="overflow-auto flex flex-col flex-1 pb-20 bg-white dark:bg-gray-900">
        <ConfigurePersona
          onChangePersona={setPersonaConfig}
          personaConfig={personaConfig}
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
        <IBlueBGButton
          type="submit"
          className="px-4 rounded-[2px]"
          isLoading={loading}
        >
          Deploy web widget
          <Rocket className="w-4 h-4 ml-2" strokeWidth={1.5} />
        </IBlueBGButton>
      </PageActionButtonBlock>
    </form>
  );
};
