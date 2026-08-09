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
  AssistantApiDeployment,
  ConnectionConfig,
  CreateAssistantApiDeployment,
  CreateAssistantDeploymentRequest,
  DeploymentAudioProvider,
  GetAssistantDeploymentRequest,
  Metadata,
} from '@rapidaai/react';
import { GetAssistantApiDeployment } from '@rapidaai/react';
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
import { InputCheckbox } from '@/app/components/carbon/form/input-checkbox';
import { ButtonSet, CheckboxGroup } from '@carbon/react';
import { Notification } from '@/app/components/carbon/notification';

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

  const [activeTab, setActiveTab] = useState('experience');
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

  const hasFetched = useRef(false);

  useEffect(() => {
    if (hasFetched.current) return;
    hasFetched.current = true;

    showLoader('block');
    const request = new GetAssistantDeploymentRequest();
    request.setAssistantid(assistantId);
    GetAssistantApiDeployment(
      connectionConfig,
      request,
      ConnectionConfig.WithDebugger({
        authorization: token,
        userId: authId,
        projectId,
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

  const activeIndex = useMemo(
    () =>
      Math.max(
        EDIT_TABS.findIndex(tab => tab.code === activeTab),
        0,
      ),
    [activeTab],
  );

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

    const req = new CreateAssistantDeploymentRequest();
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

    req.setApi(deployment);
    CreateAssistantApiDeployment(
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
        <header className="px-8 pt-8 pb-6 border-b border-gray-200 dark:border-gray-800">
          <p className="text-[10px] font-semibold tracking-[0.12em] uppercase text-gray-500 dark:text-gray-400 mb-1.5">
            SDK / API Deployment
          </p>
          <h1 className="text-xl font-semibold text-gray-900 dark:text-gray-100 leading-tight">
            Edit SDK / API deployment
          </h1>
          <p className="text-sm text-gray-500 dark:text-gray-500 mt-1.5 leading-relaxed">
            Update general experience, voice input, and voice output.
          </p>
        </header>

        <div className="border-b border-gray-200 dark:border-gray-800 shrink-0">
          <Tabs
            contained
            aria-label="API deployment edit tabs"
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
          )}
          {activeTab === 'voice-output' && (
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
