import {
  ConfigureExperience,
  DEFAULT_IDEAL_TIMEOUT,
  DEFAULT_IDLE_PROMPT_COUNT,
  validateIdleTimeoutConfig,
  ExperienceConfig,
} from '@/app/pages/assistant/actions/create-deployment/commons/configure-experience';
import { ConfigureAudioOutputProvider } from '@/app/pages/assistant/actions/create-deployment/commons/configure-audio-output';
import { ConfigureAudioInputProvider } from '@/app/pages/assistant/actions/create-deployment/commons/configure-audio-input';
import { useRapidaStore } from '@/stores/app';
import { useAllProviderCredentials } from '@/hooks/use-model';
import { useCurrentCredential } from '@/hooks/use-credential';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { FC, useEffect, useRef, useState } from 'react';
import { useParams } from 'react-router-dom';
import {
  AssistantApiDeployment,
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
  { code: 'experience', name: 'General Experience' },
  { code: 'voice-input', name: 'Voice Input' },
  { code: 'voice-output', name: 'Voice Output' },
] as const;

export function EditAssistantApiDeploymentPage() {
  const { assistantId, deploymentId } = useParams();
  void deploymentId;
  return (
    <>
      <Helmet title="Edit API deployment" />
      {assistantId && <EditAssistantApiDeployment assistantId={assistantId} />}
    </>
  );
}

const EditAssistantApiDeployment: FC<{ assistantId: string }> = ({
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

  const [experienceConfig, setExperienceConfig] = useState<ExperienceConfig>({
    greeting: undefined,
    greetingInterruptible: true,
    messageOnError: undefined,
    unclearInputTimeout: undefined,
    unclearInputMessage: undefined,
    idealTimeout: DEFAULT_IDEAL_TIMEOUT,
    idealMessage: 'Are you there?',
    maxCallDuration: '300',
    idleTimeoutBackoffTimes: DEFAULT_IDLE_PROMPT_COUNT,
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
      deploymentType: 'api',
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
        } else {
          setVoiceOutputEnable(false);
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

  const handleDeployApi = () => {
    setIsDeploying(true);
    setErrorMessage('');

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
    const deployment = new AssistantApiDeployment();
    deployment.setAssistantid(assistantId);
    deployment.setGreetinginterruptible(
      experienceConfig.greetingInterruptible ?? true,
    );
    if (experienceConfig.greeting)
      deployment.setGreeting(experienceConfig.greeting);
    if (experienceConfig.messageOnError)
      deployment.setMistake(experienceConfig.messageOnError);
    if (experienceConfig.unclearInputTimeout)
      deployment.setUnclearinputtimeout(
        Number(experienceConfig.unclearInputTimeout),
      );
    if (experienceConfig.unclearInputMessage)
      deployment.setUnclearinputmessage(experienceConfig.unclearInputMessage);
    deployment.setIdealtimeout(
      experienceConfig.idealTimeout?.trim() || DEFAULT_IDEAL_TIMEOUT,
    );
    deployment.setIdealtimeoutbackoff(
      experienceConfig.idleTimeoutBackoffTimes?.trim() ||
        DEFAULT_IDLE_PROMPT_COUNT,
    );
    if (experienceConfig.idealMessage)
      deployment.setIdealtimeoutmessage(experienceConfig.idealMessage);
    if (experienceConfig.maxCallDuration)
      deployment.setMaxsessionduration(experienceConfig.maxCallDuration);

    if (voiceInputEnable) {
      const inputAudio = new DeploymentAudioProvider();
      inputAudio.setAudioprovider(audioInputConfig.provider);
      inputAudio.setAudiooptionsList(audioInputConfig.parameters);
      deployment.setInputaudio(inputAudio);
    }

    if (voiceOutputEnable) {
      const outputAudio = new DeploymentAudioProvider();
      outputAudio.setAudioprovider(audioOutputConfig.provider);
      outputAudio.setAudiooptionsList(audioOutputConfig.parameters);
      deployment.setOutputaudio(outputAudio);
    }

    createAssistantDeploymentByType({
      deployment,
      deploymentType: 'api',
      auth: { projectId, token, userId: authId },
    })
      .then(response => {
        if (response?.getData() && response.getSuccess()) {
          toast.success('SDK / API deployment updated successfully.');
          goToDeploymentAssistant(assistantId);
        } else {
          toast.error(
            response?.getError()?.getHumanmessage() ||
              'Unable to create deployment, please try again.',
          );
        }
      })
      .catch(err => {
        setErrorMessage(
          err?.message || 'Error deploying as API. Please try again.',
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
          eyebrow="SDK / API deployment"
          title="Edit SDK / API deployment"
          description="Update general experience, voice input, and voice output."
        />

        <DeploymentEditTabs
          aria-label="API deployment edit tabs"
          tabs={EDIT_TABS}
          activeTab={activeTab}
          onChange={setActiveTab}
          panels={{
            experience: (
              <ConfigureExperience
                experienceConfig={experienceConfig}
                setExperienceConfig={setExperienceConfig}
              />
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
              onClick={handleDeployApi}
            >
              Save Changes
            </PrimaryButton>
          </ButtonSet>
        </div>
      </div>
    </>
  );
};
