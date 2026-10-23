import { ConfigureAudioInputProvider } from '@/app/pages/assistant/actions/create-deployment/commons/configure-audio-input';
import { ConfigureAudioOutputProvider } from '@/app/pages/assistant/actions/create-deployment/commons/configure-audio-output';
import {
  ConfigureExperience,
  WebWidgetExperienceConfig,
} from '@/app/pages/assistant/actions/create-deployment/web-plugin/configure-experience';
import {
  DEFAULT_IDEAL_TIMEOUT,
  DEFAULT_IDLE_PROMPT_COUNT,
  validateIdleTimeoutConfig,
  DEFAULT_UNCLEAR_INPUT_MESSAGE,
  DEFAULT_UNCLEAR_INPUT_TIMEOUT,
} from '@/app/pages/assistant/actions/create-deployment/commons/configure-experience';
import { useRapidaStore } from '@/stores/app';
import { useAllProviderCredentials } from '@/hooks/use-model';
import { useCurrentCredential } from '@/hooks/use-credential';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { FC, useEffect, useRef, useState } from 'react';
import { useParams } from 'react-router-dom';
import {
  AssistantWebpluginDeployment,
  DeploymentAudioProvider,
  Metadata,
} from '@rapidaai/react';
import toast from 'react-hot-toast/headless';
import { Helmet } from '@/app/components/app-shell/helmet';
import {
  GetDefaultMicrophoneConfig,
  GetDefaultSpeechToTextIfInvalid,
  ValidateSpeechToTextIfInvalid,
} from '@/app/components/domain/providers/speech-to-text/provider';
import {
  GetDefaultSpeakerConfig,
  GetDefaultTextToSpeechIfInvalid,
  ValidateTextToSpeechIfInvalid,
} from '@/app/components/domain/providers/text-to-speech/provider';
import { useConfirmDialog } from '@/app/pages/assistant/actions/hooks/use-confirmation';
import { PrimaryButton, SecondaryButton } from '@/app/components/ui/primitives';
import { InputCheckbox } from '@/app/components/ui/primitives';
import { ButtonSet, CheckboxGroup } from '@carbon/react';
import { Notification } from '@/app/components/ui/feedback';
import {
  createAssistantDeploymentByType,
  getAssistantDeploymentByType,
} from '@/clients/assistant.client';
import { DeploymentEditTabs } from '@/app/pages/assistant/actions/create-deployment/commons/deployment-edit-tabs';
import { DeploymentEditHeader } from '@/app/pages/assistant/actions/create-deployment/commons/deployment-edit-header';

const EDIT_TABS = [
  { code: 'experience', name: 'Experience' },
  { code: 'voice-input', name: 'Voice Input' },
  { code: 'voice-output', name: 'Voice Output' },
] as const;

export function EditAssistantWebDeploymentPage() {
  const { assistantId, deploymentId } = useParams();
  void deploymentId;
  return (
    <>
      <Helmet title="Edit web widget deployment" />
      {assistantId && <EditAssistantWebDeployment assistantId={assistantId} />}
    </>
  );
}

const EditAssistantWebDeployment: FC<{ assistantId: string }> = ({
  assistantId,
}) => {
  const { goToDeploymentAssistant } = useGlobalNavigation();
  const { showLoader, hideLoader } = useRapidaStore();
  const { providerCredentials } = useAllProviderCredentials();
  const { authId, projectId, token } = useCurrentCredential();
  const { showDialog, ConfirmDialogComponent } = useConfirmDialog({});

  const [activeTab, setActiveTab] =
    useState<(typeof EDIT_TABS)[number]['code']>('experience');
  const [errorMessage, setErrorMessage] = useState('');
  const [isDeploying, setIsDeploying] = useState(false);
  const [voiceInputEnable, setVoiceInputEnable] = useState(false);
  const [voiceOutputEnable, setVoiceOutputEnable] = useState(true);

  const [experienceConfig, setExperienceConfig] =
    useState<WebWidgetExperienceConfig>({
      greeting: undefined,
      greetingInterruptible: true,
      messageOnError: undefined,
      unclearInputTimeout: undefined,
      unclearInputMessage: undefined,
      idealTimeout: DEFAULT_IDEAL_TIMEOUT,
      idealMessage: 'Are you there?',
      maxCallDuration: '300',
      idleTimeoutBackoffTimes: DEFAULT_IDLE_PROMPT_COUNT,
      suggestions: [],
    });

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
    parameters: GetDefaultTextToSpeechIfInvalid(
      'cartesia',
      GetDefaultSpeakerConfig(),
    ),
  });

  const hasFetched = useRef(false);

  useEffect(() => {
    if (hasFetched.current) return;
    hasFetched.current = true;

    showLoader('block');
    getAssistantDeploymentByType({
      assistantId,
      deploymentType: 'web',
      auth: { projectId, token, userId: authId },
    })
      .then(response => {
        hideLoader();
        const deployment = response?.getData();
        if (!deployment) return;

        setExperienceConfig({
          greeting: deployment.getGreeting(),
          greetingInterruptible: deployment.hasGreetinginterruptible()
            ? deployment.getGreetinginterruptible()
            : true,
          suggestions: deployment.getSuggestionList() || [],
          messageOnError: deployment.getMistake(),
          unclearInputTimeout: deployment.hasUnclearinputtimeout?.()
            ? deployment.getUnclearinputtimeout().toString()
            : undefined,
          unclearInputMessage: deployment.hasUnclearinputmessage?.()
            ? deployment.getUnclearinputmessage()
            : undefined,
          idealTimeout:
            deployment.getIdealtimeout() === '0'
              ? DEFAULT_IDEAL_TIMEOUT
              : deployment.getIdealtimeout() || DEFAULT_IDEAL_TIMEOUT,
          idealMessage: deployment.getIdealtimeoutmessage(),
          maxCallDuration: deployment.getMaxsessionduration(),
          idleTimeoutBackoffTimes: deployment.getIdealtimeoutbackoff(),
        });

        if (deployment.getInputaudio()) {
          const provider = deployment.getInputaudio()!;
          setVoiceInputEnable(true);
          setAudioInputConfig({
            provider: provider.getAudioprovider() || 'deepgram',
            parameters: GetDefaultSpeechToTextIfInvalid(
              provider.getAudioprovider() || 'deepgram',
              GetDefaultMicrophoneConfig(provider.getAudiooptionsList() || []),
            ),
          });
        }

        if (deployment.getOutputaudio()) {
          const provider = deployment.getOutputaudio()!;
          setVoiceOutputEnable(true);
          setAudioOutputConfig({
            provider: provider.getAudioprovider() || 'cartesia',
            parameters: GetDefaultTextToSpeechIfInvalid(
              provider.getAudioprovider() || 'cartesia',
              GetDefaultSpeakerConfig(provider.getAudiooptionsList() || []),
            ),
          });
        }
      })
      .catch(err => {
        hideLoader();
        setErrorMessage(
          err?.message || 'Failed to fetch deployment configuration',
        );
      });
  }, [assistantId, token, authId, projectId]);

  const getProviderCredentials = (provider: string) =>
    providerCredentials.filter(c => c.getProvider() === provider);

  const handleDeployWebPlugin = () => {
    setIsDeploying(true);
    setErrorMessage('');

    if (!experienceConfig.greeting) {
      setIsDeploying(false);
      setErrorMessage('Please provide a greeting for the assistant.');
      return;
    }

    if (voiceInputEnable) {
      if (!audioInputConfig.provider) {
        setIsDeploying(false);
        setErrorMessage(
          'Please provide a provider for interpreting input audio.',
        );
        return;
      }
      const err = ValidateSpeechToTextIfInvalid(
        audioInputConfig.provider,
        audioInputConfig.parameters,
        getProviderCredentials(audioInputConfig.provider),
      );
      if (err) {
        setIsDeploying(false);
        setErrorMessage(err);
        return;
      }
    }

    if (voiceOutputEnable) {
      if (!audioOutputConfig.provider) {
        setIsDeploying(false);
        setErrorMessage(
          'Please provide a provider for interpreting output audio.',
        );
        return;
      }
      const err = ValidateTextToSpeechIfInvalid(
        audioOutputConfig.provider,
        audioOutputConfig.parameters,
        getProviderCredentials(audioOutputConfig.provider),
      );
      if (err) {
        setIsDeploying(false);
        setErrorMessage(err);
        return;
      }
    }

    const idleTimeoutError = validateIdleTimeoutConfig(experienceConfig);
    if (idleTimeoutError) {
      setIsDeploying(false);
      setErrorMessage(idleTimeoutError);
      return;
    }
    const webDeployment = new AssistantWebpluginDeployment();
    webDeployment.setAssistantid(assistantId);
    webDeployment.setGreetinginterruptible(
      experienceConfig.greetingInterruptible ?? true,
    );
    if (experienceConfig.greeting)
      webDeployment.setGreeting(experienceConfig.greeting);
    if (experienceConfig.messageOnError)
      webDeployment.setMistake(experienceConfig.messageOnError);
    webDeployment.setUnclearinputtimeout(
      Number(
        experienceConfig.unclearInputTimeout || DEFAULT_UNCLEAR_INPUT_TIMEOUT,
      ),
    );
    webDeployment.setUnclearinputmessage(
      experienceConfig.unclearInputMessage || DEFAULT_UNCLEAR_INPUT_MESSAGE,
    );
    webDeployment.setIdealtimeout(
      experienceConfig.idealTimeout?.trim() || DEFAULT_IDEAL_TIMEOUT,
    );
    webDeployment.setIdealtimeoutbackoff(
      experienceConfig.idleTimeoutBackoffTimes?.trim() ||
        DEFAULT_IDLE_PROMPT_COUNT,
    );
    if (experienceConfig.idealMessage)
      webDeployment.setIdealtimeoutmessage(experienceConfig.idealMessage);
    if (experienceConfig.maxCallDuration)
      webDeployment.setMaxsessionduration(experienceConfig.maxCallDuration);

    webDeployment.setSuggestionList(experienceConfig.suggestions);
    webDeployment.setHelpcenterenabled(false);
    webDeployment.setProductcatalogenabled(false);
    webDeployment.setArticlecatalogenabled(false);
    webDeployment.setUploadfileenabled(false);

    if (voiceInputEnable) {
      const inputAudio = new DeploymentAudioProvider();
      inputAudio.setAudioprovider(audioInputConfig.provider);
      inputAudio.setAudiooptionsList(audioInputConfig.parameters);
      webDeployment.setInputaudio(inputAudio);
    }

    if (voiceOutputEnable) {
      const outputAudio = new DeploymentAudioProvider();
      outputAudio.setAudioprovider(audioOutputConfig.provider);
      outputAudio.setAudiooptionsList(audioOutputConfig.parameters);
      webDeployment.setOutputaudio(outputAudio);
    }

    createAssistantDeploymentByType({
      deployment: webDeployment,
      deploymentType: 'web',
      auth: { projectId, token, userId: authId },
    })
      .then(response => {
        if (response?.getData() && response.getSuccess()) {
          toast.success('Web widget deployment updated successfully.');
          goToDeploymentAssistant(assistantId);
        } else {
          setErrorMessage(
            response?.getError()?.getHumanmessage() ||
              'Unable to create deployment, please try again.',
          );
        }
      })
      .catch(err => {
        setErrorMessage(
          err?.message || 'Error deploying web widget. Please try again.',
        );
      })
      .finally(() => {
        setIsDeploying(false);
      });
  };

  return (
    <>
      <ConfirmDialogComponent />
      <div className="flex flex-col flex-1 min-h-0 bg-white dark:bg-gray-900">
        <DeploymentEditHeader
          eyebrow="Web widget deployment"
          title="Edit web widget deployment"
          description="Update experience, voice input, and voice output."
        />

        <DeploymentEditTabs
          aria-label="Web widget deployment edit tabs"
          tabs={EDIT_TABS}
          activeTab={activeTab}
          onChange={setActiveTab}
          panels={{
            experience: (
              <div className="py-6">
                <ConfigureExperience
                  experienceConfig={experienceConfig}
                  setExperienceConfig={setExperienceConfig}
                />
              </div>
            ),
            'voice-input': (
              <div>
                <div className="px-6 pt-6">
                  <CheckboxGroup
                    legendText=""
                    warn
                    warnText={
                      voiceInputEnable
                        ? 'Assistant can now receive user input via audio and text.'
                        : 'Assistant will now receive user input via text only.'
                    }
                  >
                    <InputCheckbox
                      checked={voiceInputEnable}
                      onChange={e => setVoiceInputEnable(e.target.checked)}
                      id="voice-input-toggle-edit"
                    >
                      Enable Voice Input (Speech-to-Text)
                    </InputCheckbox>
                  </CheckboxGroup>
                </div>
                {voiceInputEnable && (
                  <ConfigureAudioInputProvider
                    audioInputConfig={audioInputConfig}
                    setAudioInputConfig={setAudioInputConfig}
                  />
                )}
              </div>
            ),
            'voice-output': (
              <div>
                <div className="px-6 pt-6">
                  <CheckboxGroup
                    legendText=""
                    warn
                    warnText={
                      voiceOutputEnable
                        ? 'Assistant responses will now be delivered via audio and text.'
                        : 'Assistant responses will now be delivered via text.'
                    }
                  >
                    <InputCheckbox
                      checked={voiceOutputEnable}
                      onChange={e => setVoiceOutputEnable(e.target.checked)}
                      id="voice-output-toggle-edit"
                    >
                      Enable Voice Output (Text-to-Speech)
                    </InputCheckbox>
                  </CheckboxGroup>
                </div>
                {voiceOutputEnable && (
                  <ConfigureAudioOutputProvider
                    audioOutputConfig={audioOutputConfig}
                    setAudioOutputConfig={setAudioOutputConfig}
                  />
                )}
              </div>
            ),
          }}
        />

        <div className="shrink-0">
          {errorMessage && (
            <Notification kind="error" title="Error" subtitle={errorMessage} />
          )}
          <ButtonSet className="!w-full [&>button]:!flex-1 [&>button]:!max-w-none">
            <SecondaryButton
              size="lg"
              className="w-full h-full"
              onClick={() =>
                showDialog(() => goToDeploymentAssistant(assistantId))
              }
            >
              Cancel
            </SecondaryButton>
            <PrimaryButton
              size="lg"
              type="button"
              className="w-full h-full"
              isLoading={isDeploying}
              disabled={isDeploying}
              onClick={handleDeployWebPlugin}
            >
              Save Changes
            </PrimaryButton>
          </ButtonSet>
        </div>
      </div>
    </>
  );
};
