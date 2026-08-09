import {
  ConfigureExperience,
  DEFAULT_UNCLEAR_INPUT_MESSAGE,
  DEFAULT_UNCLEAR_INPUT_TIMEOUT,
  ExperienceConfig,
} from '@/app/pages/assistant/actions/create-deployment/commons/configure-experience';
import { ConfigureAudioOutputProvider } from '@/app/pages/assistant/actions/create-deployment/commons/configure-audio-output';
import { ConfigureAudioInputProvider } from '@/app/pages/assistant/actions/create-deployment/commons/configure-audio-input';
import { useRapidaStore } from '@/hooks';
import { useAllProviderCredentials } from '@/hooks/use-model';
import { useCurrentCredential } from '@/hooks/use-credential';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { FC, useEffect, useMemo, useRef, useState } from 'react';
import { useParams, useSearchParams } from 'react-router-dom';
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
import { DebuggerDeploymentSuccessDialog } from '@/app/components/base/modal/debugger-deployment-success-modal';
import { TabForm } from '@/app/components/form/tab-form';
import {
  PrimaryButton,
  SecondaryButton,
  GhostButton,
} from '@/app/components/carbon/button';
import { ButtonSet, CheckboxGroup } from '@carbon/react';
import { InputCheckbox } from '@/app/components/carbon/form/input-checkbox';

type SectionCode = 'experience' | 'stt' | 'tts';
type ExistingDebuggerConfig = {
  experience: ExperienceConfig;
  inputAudio?: { provider: string; parameters: Metadata[] };
  outputAudio?: { provider: string; parameters: Metadata[] };
};

const STEPS = [
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
];

export function ConfigureAssistantDebuggerDeploymentPage() {
  const { assistantId } = useParams();
  return (
    <>
      <Helmet title="Configure debugger deployment" />
      {assistantId && (
        <ConfigureAssistantDebuggerDeployment assistantId={assistantId} />
      )}
    </>
  );
}

const ConfigureAssistantDebuggerDeployment: FC<{ assistantId: string }> = ({
  assistantId,
}) => {
  const [searchParams] = useSearchParams();
  const { goToDeploymentAssistant } = useGlobalNavigation();
  const { showLoader, hideLoader } = useRapidaStore();
  const { providerCredentials } = useAllProviderCredentials();
  const { authId, projectId, token } = useCurrentCredential();

  const [activeTab, setActiveTab] = useState('experience');
  const [errorMessage, setErrorMessage] = useState('');
  const [isDeploying, setIsDeploying] = useState(false);
  const [voiceInputEnable, setVoiceInputEnable] = useState(true);
  const [voiceOutputEnable, setVoiceOutputEnable] = useState(true);
  const [success, setSuccess] = useState(false);
  const [existingConfig, setExistingConfig] = useState<ExistingDebuggerConfig>({
    experience: {
      greeting: undefined,
      greetingInterruptible: true,
      messageOnError: undefined,
      unclearInputTimeout: undefined,
      unclearInputMessage: undefined,
      idealTimeout: '30',
      idealMessage: 'Are you there?',
      maxCallDuration: '300',
      idleTimeoutBackoffTimes: '2',
    },
  });

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

  const section = searchParams.get('section');
  const editMode = searchParams.get('editMode');
  const editSection: SectionCode | null = useMemo(() => {
    if (editMode !== 'section') return null;
    if (section === 'experience' || section === 'stt' || section === 'tts') {
      return section;
    }
    return null;
  }, [editMode, section]);
  const isSectionMode = !!editSection;

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

        const fetchedExperience = {
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
        };
        setExperienceConfig(fetchedExperience);

        let fetchedInputAudio: ExistingDebuggerConfig['inputAudio'];
        if (deployment.getInputaudio()) {
          const provider = deployment.getInputaudio()!;
          fetchedInputAudio = {
            provider: provider.getAudioprovider() || 'deepgram',
            parameters: provider.getAudiooptionsList() || [],
          };
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

        let fetchedOutputAudio: ExistingDebuggerConfig['outputAudio'];
        if (deployment.getOutputaudio()) {
          const provider = deployment.getOutputaudio()!;
          fetchedOutputAudio = {
            provider: provider.getAudioprovider() || 'cartesia',
            parameters: provider.getAudiooptionsList() || [],
          };
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

        setExistingConfig({
          experience: fetchedExperience,
          inputAudio: fetchedInputAudio,
          outputAudio: fetchedOutputAudio,
        });
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

  const handleTabChange = (code: string) => {
    const clickedIndex = STEPS.findIndex(s => s.code === code);
    const currentIndex = STEPS.findIndex(s => s.code === activeTab);
    if (clickedIndex < currentIndex) {
      setActiveTab(code);
      setErrorMessage('');
    }
  };

  const handlePrevious = () => {
    setErrorMessage('');
    const idx = STEPS.findIndex(s => s.code === activeTab);
    if (idx > 0) setActiveTab(STEPS[idx - 1].code);
  };

  const handleNext = () => {
    setErrorMessage('');
    const idx = STEPS.findIndex(s => s.code === activeTab);

    if (activeTab === 'voice-input') {
      if (voiceInputEnable) {
        if (!audioInputConfig.provider) {
          setErrorMessage('Please select a speech-to-text provider.');
          return;
        }
        const err = ValidateSpeechToTextIfInvalid(
          audioInputConfig.provider,
          audioInputConfig.parameters,
          getProviderCredentials(audioInputConfig.provider),
        );
        if (err) {
          setErrorMessage(err);
          return;
        }
      }
    }

    if (idx < STEPS.length - 1) {
      setActiveTab(STEPS[idx + 1].code);
    }
  };

  const handleDeployDebugger = () => {
    setIsDeploying(true);
    setErrorMessage('');

    const shouldValidateVoiceInput = !isSectionMode || editSection === 'stt';
    const shouldValidateVoiceOutput = !isSectionMode || editSection === 'tts';

    if (shouldValidateVoiceInput && voiceInputEnable) {
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

    if (shouldValidateVoiceOutput && voiceOutputEnable) {
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
    const resolvedExperience =
      isSectionMode && editSection !== 'experience'
        ? existingConfig.experience
        : experienceConfig;
    deployment.setGreetinginterruptible(
      resolvedExperience.greetingInterruptible ?? true,
    );
    if (resolvedExperience.greeting)
      deployment.setGreeting(resolvedExperience.greeting);
    if (resolvedExperience.messageOnError)
      deployment.setMistake(resolvedExperience.messageOnError);
    if (resolvedExperience.unclearInputTimeout)
      deployment.setUnclearinputtimeout(
        Number(resolvedExperience.unclearInputTimeout),
      );
    if (resolvedExperience.unclearInputMessage)
      deployment.setUnclearinputmessage(resolvedExperience.unclearInputMessage);
    if (resolvedExperience.idealTimeout)
      deployment.setIdealtimeout(resolvedExperience.idealTimeout);
    if (resolvedExperience.idleTimeoutBackoffTimes)
      deployment.setIdealtimeoutbackoff(
        resolvedExperience.idleTimeoutBackoffTimes,
      );
    if (resolvedExperience.idealMessage)
      deployment.setIdealtimeoutmessage(resolvedExperience.idealMessage);
    if (resolvedExperience.maxCallDuration)
      deployment.setMaxsessionduration(resolvedExperience.maxCallDuration);

    if (isSectionMode && editSection !== 'stt') {
      if (existingConfig.inputAudio) {
        const inputAudio = new DeploymentAudioProvider();
        inputAudio.setAudioprovider(existingConfig.inputAudio.provider);
        inputAudio.setAudiooptionsList(existingConfig.inputAudio.parameters);
        deployment.setInputaudio(inputAudio);
      }
    } else if (voiceInputEnable) {
      const inputAudio = new DeploymentAudioProvider();
      inputAudio.setAudioprovider(audioInputConfig.provider);
      inputAudio.setAudiooptionsList(audioInputConfig.parameters);
      deployment.setInputaudio(inputAudio);
    }

    if (isSectionMode && editSection !== 'tts') {
      if (existingConfig.outputAudio) {
        const outputAudio = new DeploymentAudioProvider();
        outputAudio.setAudioprovider(existingConfig.outputAudio.provider);
        outputAudio.setAudiooptionsList(existingConfig.outputAudio.parameters);
        deployment.setOutputaudio(outputAudio);
      }
    } else if (voiceOutputEnable) {
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
          if (isSectionMode) {
            toast.success('Debugger deployment section updated successfully.');
            goToDeploymentAssistant(assistantId);
          } else {
            toast.success('Debugger deployment updated successfully.');
            setSuccess(true);
          }
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

  const experienceBody = (
    <ConfigureExperience
      experienceConfig={experienceConfig}
      setExperienceConfig={setExperienceConfig}
    />
  );

  const voiceInputBody = (
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
            id="voice-input-toggle"
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
  );

  const voiceOutputBody = (
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
            id="voice-output-toggle"
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
  );

  const sectionBodyMap: Record<SectionCode, React.ReactNode> = {
    experience: experienceBody,
    stt: voiceInputBody,
    tts: voiceOutputBody,
  };

  const sectionTitleMap: Record<SectionCode, string> = {
    experience: 'Experience',
    stt: 'STT',
    tts: 'TTS',
  };

  return (
    <>
      <ConfirmDialogComponent />
      <DebuggerDeploymentSuccessDialog
        modalOpen={success}
        setModalOpen={() => {
          setSuccess(false);
          goToDeploymentAssistant(assistantId);
        }}
        assistantId={assistantId}
      />
      <div className="flex flex-col flex-1 min-h-0 bg-white dark:bg-gray-900">
        {isSectionMode && editSection ? (
          <div className="flex flex-col flex-1 min-h-0">
            <div className="px-6 pt-6 pb-4 border-b border-gray-200 dark:border-gray-800">
              <h2 className="text-2xl font-light tracking-tight">
                Edit {sectionTitleMap[editSection]} settings
              </h2>
            </div>
            <div className="flex-1 overflow-auto">
              {sectionBodyMap[editSection]}
            </div>
            <div className="px-6 py-4 border-t border-gray-200 dark:border-gray-800">
              {errorMessage && (
                <p className="text-sm text-red-600 dark:text-red-400 mb-3">
                  {errorMessage}
                </p>
              )}
              <ButtonSet className="!w-full [&>button]:!flex-1 [&>button]:!max-w-none">
                <SecondaryButton
                  size="lg"
                  onClick={() =>
                    showDialog(() => goToDeploymentAssistant(assistantId))
                  }
                >
                  Cancel
                </SecondaryButton>
                <PrimaryButton
                  size="lg"
                  isLoading={isDeploying}
                  disabled={isDeploying}
                  onClick={handleDeployDebugger}
                >
                  Save
                </PrimaryButton>
              </ButtonSet>
            </div>
          </div>
        ) : (
          <TabForm
            formHeading="Complete all steps to configure your debugger deployment."
            activeTab={activeTab}
            onChangeActiveTab={handleTabChange}
            errorMessage={errorMessage}
            form={[
              {
                code: 'experience',
                name: 'General Experience',
                description:
                  'Define how the assistant greets users and handles sessions.',
                body: experienceBody,
                actions: [
                  <ButtonSet className="!w-full [&>button]:!flex-1 [&>button]:!max-w-none">
                    <SecondaryButton
                      size="lg"
                      onClick={() =>
                        showDialog(() => goToDeploymentAssistant(assistantId))
                      }
                    >
                      Cancel
                    </SecondaryButton>
                    <PrimaryButton size="lg" onClick={handleNext}>
                      Next
                    </PrimaryButton>
                  </ButtonSet>,
                ],
              },
              {
                code: 'voice-input',
                name: 'Voice Input',
                description:
                  'Configure the speech-to-text provider for capturing user audio.',
                body: voiceInputBody,
                actions: [
                  <ButtonSet className="!w-full [&>button]:!flex-1 [&>button]:!max-w-none">
                    <GhostButton size="lg" onClick={handlePrevious}>
                      Previous
                    </GhostButton>
                    <SecondaryButton
                      size="lg"
                      onClick={() =>
                        showDialog(() => goToDeploymentAssistant(assistantId))
                      }
                    >
                      Cancel
                    </SecondaryButton>
                    <PrimaryButton size="lg" onClick={handleNext}>
                      Next
                    </PrimaryButton>
                  </ButtonSet>,
                ],
              },
              {
                code: 'voice-output',
                name: 'Voice Output',
                description:
                  'Configure the text-to-speech provider for audio responses.',
                body: voiceOutputBody,
                actions: [
                  <ButtonSet className="!w-full [&>button]:!flex-1 [&>button]:!max-w-none">
                    <GhostButton size="lg" onClick={handlePrevious}>
                      Previous
                    </GhostButton>
                    <SecondaryButton
                      size="lg"
                      onClick={() =>
                        showDialog(() => goToDeploymentAssistant(assistantId))
                      }
                    >
                      Cancel
                    </SecondaryButton>
                    <PrimaryButton
                      size="lg"
                      isLoading={isDeploying}
                      disabled={isDeploying}
                      onClick={handleDeployDebugger}
                    >
                      Deploy Debugger
                    </PrimaryButton>
                  </ButtonSet>,
                ],
              },
            ]}
          />
        )}
      </div>
    </>
  );
};
