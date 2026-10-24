import {
  ConfigureExperience,
  DEFAULT_IDEAL_TIMEOUT,
  DEFAULT_IDLE_PROMPT_COUNT,
  validateIdleTimeoutConfig,
  DEFAULT_UNCLEAR_INPUT_MESSAGE,
  DEFAULT_UNCLEAR_INPUT_TIMEOUT,
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
  AssistantDebuggerDeployment,
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
import { ButtonSet, CheckboxGroup } from '@carbon/react';
import { InputCheckbox } from '@/app/components/ui/primitives';
import { Notification } from '@/app/components/ui/feedback';
import {
  createAssistantDeploymentByType,
  getAssistantDeploymentByType,
} from '@/clients/assistant.client';
import { DeploymentEditTabs } from '@/app/pages/assistant/actions/create-deployment/commons/deployment-edit-tabs';
import { DeploymentEditHeader } from '@/app/pages/assistant/actions/create-deployment/commons/deployment-edit-header';

const EDIT_TABS = [
  {
    code: 'experience',
    name: 'General Experience',
    description: 'Define how the assistant greets users and handles sessions.',
  },
  {
    code: 'voice-input',
    name: 'Voice Input',
    description:
      'Configure the speech-to-text provider for capturing user audio.',
  },
  {
    code: 'voice-output',
    name: 'Voice Output',
    description: 'Configure the text-to-speech provider for audio responses.',
  },
] as const;

export function EditAssistantDebuggerDeploymentPage() {
  const { assistantId, deploymentId } = useParams();
  void deploymentId;
  return (
    <>
      <Helmet title="Edit debugger deployment" />
      {assistantId && (
        <EditAssistantDebuggerDeployment assistantId={assistantId} />
      )}
    </>
  );
}

const EditAssistantDebuggerDeployment: FC<{ assistantId: string }> = ({
  assistantId,
}) => {
  const { goToDeploymentAssistant } = useGlobalNavigation();
  const { showLoader, hideLoader } = useRapidaStore();
  const { providerCredentials } = useAllProviderCredentials();
  const { authId, projectId, token } = useCurrentCredential();

  const [activeTab, setActiveTab] =
    useState<(typeof EDIT_TABS)[number]['code']>('experience');
  const [errorMessage, setErrorMessage] = useState('');
  const [isDeploying, setIsDeploying] = useState(false);
  const [voiceInputEnable, setVoiceInputEnable] = useState(true);
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

  const { showDialog, ConfirmDialogComponent } = useConfirmDialog({});
  const hasFetched = useRef(false);

  useEffect(() => {
    if (hasFetched.current) return;
    hasFetched.current = true;

    showLoader('block');
    getAssistantDeploymentByType({
      assistantId,
      deploymentType: 'debugger',
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
        } else {
          setVoiceInputEnable(false);
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
      .catch(() => {
        hideLoader();
        setErrorMessage(
          'Unable to load debugger deployment configuration. Please try again.',
        );
      });
  }, [assistantId, token, authId, projectId]);

  const getProviderCredentials = (provider: string) =>
    providerCredentials.filter(c => c.getProvider() === provider);

  const handleDeployDebugger = () => {
    setIsDeploying(true);
    setErrorMessage('');

    if (voiceInputEnable) {
      if (!audioInputConfig.provider) {
        setIsDeploying(false);
        setErrorMessage(
          'Please select a speech-to-text provider for voice input.',
        );
        return;
      }
      const inputError = ValidateSpeechToTextIfInvalid(
        audioInputConfig.provider,
        audioInputConfig.parameters,
        getProviderCredentials(audioInputConfig.provider),
      );
      if (inputError) {
        setIsDeploying(false);
        setErrorMessage(inputError);
        return;
      }
    }

    if (voiceOutputEnable) {
      if (!audioOutputConfig.provider) {
        setIsDeploying(false);
        setErrorMessage(
          'Please select a text-to-speech provider for voice output.',
        );
        return;
      }
      const outputError = ValidateTextToSpeechIfInvalid(
        audioOutputConfig.provider,
        audioOutputConfig.parameters,
        getProviderCredentials(audioOutputConfig.provider),
      );
      if (outputError) {
        setIsDeploying(false);
        setErrorMessage(outputError);
        return;
      }
    }

    const idleTimeoutError = validateIdleTimeoutConfig(experienceConfig);
    if (idleTimeoutError) {
      setIsDeploying(false);
      setErrorMessage(idleTimeoutError);
      return;
    }
    const deployment = new AssistantDebuggerDeployment();
    deployment.setAssistantid(assistantId);
    deployment.setGreetinginterruptible(
      experienceConfig.greetingInterruptible ?? true,
    );
    if (experienceConfig.greeting)
      deployment.setGreeting(experienceConfig.greeting);
    if (experienceConfig.messageOnError)
      deployment.setMistake(experienceConfig.messageOnError);
    deployment.setUnclearinputtimeout(
      Number(
        experienceConfig.unclearInputTimeout || DEFAULT_UNCLEAR_INPUT_TIMEOUT,
      ),
    );
    deployment.setUnclearinputmessage(
      experienceConfig.unclearInputMessage || DEFAULT_UNCLEAR_INPUT_MESSAGE,
    );
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
      deploymentType: 'debugger',
      auth: { projectId, token, userId: authId },
    })
      .then(response => {
        if (response?.getData() && response.getSuccess()) {
          toast.success('Debugger deployment updated successfully.');
          goToDeploymentAssistant(assistantId);
        } else {
          toast.error(
            response?.getError()?.getHumanmessage() ||
              'Unable to deploy. Please try again.',
          );
        }
      })
      .catch(() => {
        setErrorMessage(
          'Error deploying as debugger. Please check and try again.',
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
          eyebrow="Debugger deployment"
          title="Edit debugger deployment"
          description="Update general experience, voice input, and voice output."
        />

        <DeploymentEditTabs
          aria-label="Debugger deployment edit tabs"
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
                    legendText="Voice input"
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
                      Enable
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
                    legendText="Voice output"
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
                      Enable
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
              onClick={handleDeployDebugger}
            >
              Save Changes
            </PrimaryButton>
          </ButtonSet>
        </div>
      </div>
    </>
  );
};
