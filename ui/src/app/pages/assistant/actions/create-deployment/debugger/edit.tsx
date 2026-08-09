import {
  ConfigureExperience,
  DEFAULT_IDEAL_TIMEOUT,
  ExperienceConfig,
} from '@/app/pages/assistant/actions/create-deployment/commons/configure-experience';
import { ConfigureAudioOutputProvider } from '@/app/pages/assistant/actions/create-deployment/commons/configure-audio-output';
import { ConfigureAudioInputProvider } from '@/app/pages/assistant/actions/create-deployment/commons/configure-audio-input';
import { useRapidaStore } from '@/hooks';
import { useAllProviderCredentials } from '@/hooks/use-model';
import { useCurrentCredential } from '@/hooks/use-credential';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { FC, useEffect, useMemo, useRef, useState } from 'react';
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
import { connectionConfig } from '@/configs';
import { useConfirmDialog } from '@/app/pages/assistant/actions/hooks/use-confirmation';
import { Tabs } from '@/app/components/carbon/tabs';
import { PrimaryButton, SecondaryButton } from '@/app/components/carbon/button';
import { ButtonSet, CheckboxGroup } from '@carbon/react';
import { InputCheckbox } from '@/app/components/carbon/form/input-checkbox';
import { Notification } from '@/app/components/carbon/notification';

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

  const [activeTab, setActiveTab] = useState('experience');
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
    idleTimeoutBackoffTimes: '2',
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
    const request = new GetAssistantDeploymentRequest();
    request.setAssistantid(assistantId);
    GetAssistantDebuggerDeployment(
      connectionConfig,
      request,
      ConnectionConfig.WithDebugger({
        authorization: token,
        projectId,
        userId: authId,
      }),
    )
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
          idealTimeout: deployment.getIdealtimeout(),
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

  const activeIndex = useMemo(
    () =>
      Math.max(
        EDIT_TABS.findIndex(tab => tab.code === activeTab),
        0,
      ),
    [activeTab],
  );

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

    const deployment = new AssistantDebuggerDeployment();
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
    if (experienceConfig.idealTimeout)
      deployment.setIdealtimeout(experienceConfig.idealTimeout);
    if (experienceConfig.idleTimeoutBackoffTimes)
      deployment.setIdealtimeoutbackoff(
        experienceConfig.idleTimeoutBackoffTimes,
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

    const req = new CreateAssistantDeploymentRequest();
    req.setDebugger(deployment);

    CreateAssistantDebuggerDeployment(
      connectionConfig,
      req,
      ConnectionConfig.WithDebugger({
        authorization: token,
        userId: authId,
        projectId,
      }),
    )
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
        <header className="px-8 pt-8 pb-6 border-b border-gray-200 dark:border-gray-800">
          <p className="text-[10px] font-semibold tracking-[0.12em] uppercase text-gray-500 dark:text-gray-400 mb-1.5">
            Debugger Deployment
          </p>
          <h1 className="text-xl font-semibold text-gray-900 dark:text-gray-100 leading-tight">
            Edit debugger deployment
          </h1>
          <p className="text-sm text-gray-500 dark:text-gray-500 mt-1.5 leading-relaxed">
            Update general experience, voice input, and voice output.
          </p>
        </header>

        <div className="border-b border-gray-200 dark:border-gray-800 shrink-0">
          <Tabs
            contained
            aria-label="Debugger deployment edit tabs"
            tabs={EDIT_TABS.map(tab => tab.name)}
            selectedIndex={activeIndex}
            onChange={index => setActiveTab(EDIT_TABS[index].code)}
            className="w-full [&_.cds--tab--list]:w-full [&_.cds--tabs__nav]:w-full [&_.cds--tabs__nav-item]:flex-1 [&_.cds--tabs__nav-link]:w-full [&_.cds--tabs__nav-link]:justify-center [&_.cds--tabs__nav-item]:mx-0 [&_.cds--tabs__nav-link]:ml-0"
            panelClassName="!px-0"
          />
        </div>

        <div className="flex-1 min-h-0 overflow-auto">
          {activeTab === 'experience' && (
            <div className="w-full">
              <ConfigureExperience
                experienceConfig={experienceConfig}
                setExperienceConfig={setExperienceConfig}
              />
            </div>
          )}
          {activeTab === 'voice-input' && (
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
          )}
          {activeTab === 'voice-output' && (
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
          )}
        </div>

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
