import {
  ConfigureExperience,
  DEFAULT_UNCLEAR_INPUT_MESSAGE,
  DEFAULT_UNCLEAR_INPUT_TIMEOUT,
  ExperienceConfig,
} from '@/app/pages/assistant/actions/create-deployment/commons/configure-experience';
import { ConfigureAudioInputProvider } from '@/app/pages/assistant/actions/create-deployment/commons/configure-audio-input';
import { ConfigureAudioOutputProvider } from '@/app/pages/assistant/actions/create-deployment/commons/configure-audio-output';
import { useRapidaStore } from '@/hooks';
import { useAllProviderCredentials } from '@/hooks/use-model';
import { FC, useEffect, useRef, useState } from 'react';
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
import {
  TelephonyProvider,
  GetDefaultTelephonyConfigIfInvalid,
  ValidateTelephonyOptions,
} from '@/app/components/providers/telephony';
import { useConfirmDialog } from '@/app/pages/assistant/actions/hooks/use-confirmation';
import { Tabs } from '@/app/components/carbon/tabs';
import { PrimaryButton, SecondaryButton } from '@/app/components/carbon/button';
import { Notification } from '@/app/components/carbon/notification';
import { ButtonSet } from '@carbon/react';

const EDIT_TABS = [
  {
    code: 'telephony',
    name: 'Telephony',
    description:
      'Select and configure your telephony provider for inbound and outbound calls.',
  },
  {
    code: 'experience',
    name: 'General Experience',
    description: 'Define how the assistant greets users and handles sessions.',
  },
  {
    code: 'voice-input',
    name: 'Voice Input',
    description:
      'Configure the speech-to-text provider for capturing caller audio.',
  },
  {
    code: 'voice-output',
    name: 'Voice Output',
    description: 'Configure the text-to-speech provider for audio responses.',
  },
] as const;

export function EditAssistantCallDeploymentPage() {
  const { assistantId, deploymentId } = useParams();
  void deploymentId;
  return (
    <>
      <Helmet title="Edit phone deployment" />
      {assistantId && <EditAssistantCallDeployment assistantId={assistantId} />}
    </>
  );
}

const EditAssistantCallDeployment: FC<{ assistantId: string }> = ({
  assistantId,
}) => {
  const { goToDeploymentAssistant } = useGlobalNavigation();
  const { showLoader, hideLoader } = useRapidaStore();
  const { providerCredentials } = useAllProviderCredentials();
  const { authId, projectId, token } = useCurrentCredential();

  const [activeTab, setActiveTab] = useState('telephony');
  const [errorMessage, setErrorMessage] = useState('');
  const [isDeploying, setIsDeploying] = useState(false);
  const { showDialog, ConfirmDialogComponent } = useConfirmDialog({});

  const [experienceConfig, setExperienceConfig] = useState<ExperienceConfig>({
    greeting: undefined,
    greetingInterruptible: true,
    messageOnError: undefined,
    unclearInputTimeout: DEFAULT_UNCLEAR_INPUT_TIMEOUT,
    unclearInputMessage: DEFAULT_UNCLEAR_INPUT_MESSAGE,
    idealTimeout: '30',
    idealMessage: 'Are you there?',
    maxCallDuration: '300',
    idleTimeoutBackoffTimes: '2',
  });

  const [telephonyConfig, setTelephonyConfig] = useState<{
    provider: string;
    parameters: Metadata[];
  }>({
    provider: 'twilio',
    parameters: [],
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
    GetAssistantPhoneDeployment(
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
            : DEFAULT_UNCLEAR_INPUT_TIMEOUT,
          unclearInputMessage: deployment.hasUnclearinputmessage?.()
            ? deployment.getUnclearinputmessage()
            : DEFAULT_UNCLEAR_INPUT_MESSAGE,
          idealTimeout: deployment.getIdealtimeout(),
          idealMessage: deployment.getIdealtimeoutmessage(),
          maxCallDuration: deployment.getMaxsessionduration(),
          idleTimeoutBackoffTimes: deployment.getIdealtimeoutbackoff(),
        });

        if (deployment.getPhoneprovidername()) {
          const provider = deployment.getPhoneprovidername() || '';
          setTelephonyConfig({
            provider,
            parameters: GetDefaultTelephonyConfigIfInvalid(
              provider,
              deployment.getPhoneoptionsList() || [],
            ),
          });
        }

        if (deployment.getInputaudio()) {
          const provider = deployment.getInputaudio()!;
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
        toast.error(
          err?.message ||
            'Error loading phone deployment configuration. Please try again.',
        );
      });
  }, [assistantId, token, authId, projectId]);

  const getProviderCredentials = (provider: string) =>
    providerCredentials.filter(c => c.getProvider() === provider);

  const activeIndex = EDIT_TABS.findIndex(tab => tab.code === activeTab);

  const handleTabChange = (index: number) => {
    if (index < 0 || index >= EDIT_TABS.length) return;
    setActiveTab(EDIT_TABS[index].code);
    setErrorMessage('');
  };

  const handleDeployPhone = () => {
    setIsDeploying(true);
    setErrorMessage('');

    const resolvedTelephonyParameters = GetDefaultTelephonyConfigIfInvalid(
      telephonyConfig.provider,
      telephonyConfig.parameters,
    );
    if (
      !ValidateTelephonyOptions(
        telephonyConfig.provider,
        resolvedTelephonyParameters,
      )
    ) {
      setIsDeploying(false);
      setErrorMessage('Please provide a valid telephony configuration.');
      return;
    }

    if (!audioInputConfig.provider) {
      setIsDeploying(false);
      setErrorMessage('Please provide a speech-to-text provider.');
      return;
    }

    const sttError = ValidateSpeechToTextIfInvalid(
      audioInputConfig.provider,
      audioInputConfig.parameters,
      getProviderCredentials(audioInputConfig.provider),
    );
    if (sttError) {
      setIsDeploying(false);
      setErrorMessage(sttError);
      return;
    }

    if (!audioOutputConfig.provider) {
      setIsDeploying(false);
      setErrorMessage('Please provide a text-to-speech provider.');
      return;
    }

    const ttsError = ValidateTextToSpeechIfInvalid(
      audioOutputConfig.provider,
      audioOutputConfig.parameters,
      getProviderCredentials(audioOutputConfig.provider),
    );
    if (ttsError) {
      setIsDeploying(false);
      setErrorMessage(ttsError);
      return;
    }

    const req = new CreateAssistantDeploymentRequest();
    const deployment = new AssistantPhoneDeployment();
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

    deployment.setPhoneprovidername(telephonyConfig.provider);
    deployment.setPhoneoptionsList(resolvedTelephonyParameters);

    const inputAudio = new DeploymentAudioProvider();
    inputAudio.setAudioprovider(audioInputConfig.provider);
    inputAudio.setAudiooptionsList(audioInputConfig.parameters);
    deployment.setInputaudio(inputAudio);

    const outputAudio = new DeploymentAudioProvider();
    outputAudio.setAudioprovider(audioOutputConfig.provider);
    outputAudio.setAudiooptionsList(audioOutputConfig.parameters);
    deployment.setOutputaudio(outputAudio);

    req.setPhone(deployment);

    CreateAssistantPhoneDeployment(
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
          toast.success('Phone call deployment updated successfully.');
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
          err?.message || 'Error deploying phone call. Please try again.',
        );
      })
      .finally(() => {
        setIsDeploying(false);
      });
  };

  return (
    <div className="flex flex-col flex-1 min-h-0 bg-white dark:bg-gray-900">
      <ConfirmDialogComponent />
      <header className="px-8 pt-8 pb-6 border-b border-gray-200 dark:border-gray-800">
        <p className="text-[10px] font-semibold tracking-[0.12em] uppercase text-gray-500 dark:text-gray-400 mb-1.5">
          Phone Deployment
        </p>
        <h1 className="text-xl font-semibold text-gray-900 dark:text-gray-100 leading-tight">
          Edit phone deployment
        </h1>
        <p className="text-sm text-gray-500 dark:text-gray-500 mt-1.5 leading-relaxed">
          Update telephony, voice input, general experience, and voice output.
        </p>
      </header>

      <div className="border-b border-gray-200 dark:border-gray-800 shrink-0">
        <Tabs
          contained
          aria-label="Phone deployment edit tabs"
          tabs={EDIT_TABS.map(tab => tab.name)}
          selectedIndex={Math.max(activeIndex, 0)}
          onChange={handleTabChange}
          className="w-full [&_.cds--tab--list]:w-full [&_.cds--tabs__nav]:w-full [&_.cds--tabs__nav-item]:flex-1 [&_.cds--tabs__nav-link]:w-full [&_.cds--tabs__nav-link]:justify-center [&_.cds--tabs__nav-item]:mx-0 [&_.cds--tabs__nav-link]:ml-0"
          panelClassName="!px-0"
        />
      </div>
      <div className="flex-1 min-h-0 overflow-auto">
        {activeTab === 'telephony' && (
          <div className="flex flex-col gap-6 max-w-4xl p-6">
            <TelephonyProvider
              provider={telephonyConfig.provider}
              parameters={telephonyConfig.parameters}
              onChangeProvider={provider =>
                setTelephonyConfig({
                  provider,
                  parameters: GetDefaultTelephonyConfigIfInvalid(provider, []),
                })
              }
              onChangeParameter={parameters =>
                setTelephonyConfig(c => ({
                  ...c,
                  parameters: GetDefaultTelephonyConfigIfInvalid(
                    c.provider,
                    parameters,
                  ),
                }))
              }
            />
          </div>
        )}
        {activeTab === 'voice-input' && (
          <ConfigureAudioInputProvider
            audioInputConfig={audioInputConfig}
            setAudioInputConfig={setAudioInputConfig}
          />
        )}
        {activeTab === 'experience' && (
          <ConfigureExperience
            experienceConfig={experienceConfig}
            setExperienceConfig={setExperienceConfig}
          />
        )}
        {activeTab === 'voice-output' && (
          <ConfigureAudioOutputProvider
            audioOutputConfig={audioOutputConfig}
            setAudioOutputConfig={setAudioOutputConfig}
          />
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
            onClick={handleDeployPhone}
          >
            Save Changes
          </PrimaryButton>
        </ButtonSet>
      </div>
    </div>
  );
};
