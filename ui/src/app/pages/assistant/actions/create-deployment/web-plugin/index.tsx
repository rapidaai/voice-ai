import { ConfigureAudioInputProvider } from '@/app/pages/assistant/actions/create-deployment/commons/configure-audio-input';
import { ConfigureAudioOutputProvider } from '@/app/pages/assistant/actions/create-deployment/commons/configure-audio-output';
import {
  ConfigureExperience,
  WebWidgetExperienceConfig,
} from '@/app/pages/assistant/actions/create-deployment/web-plugin/configure-experience';
import {
  DEFAULT_IDEAL_TIMEOUT,
  DEFAULT_UNCLEAR_INPUT_MESSAGE,
  DEFAULT_UNCLEAR_INPUT_TIMEOUT,
} from '@/app/pages/assistant/actions/create-deployment/commons/configure-experience';
import { useRapidaStore } from '@/hooks';
import { useAllProviderCredentials } from '@/hooks/use-model';
import { useCurrentCredential } from '@/hooks/use-credential';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { FC, useEffect, useRef, useState } from 'react';
import { useParams } from 'react-router-dom';
import {
  AssistantWebpluginDeployment,
  ConnectionConfig,
  CreateAssistantDeploymentRequest,
  CreateAssistantWebpluginDeployment,
  DeploymentAudioProvider,
  GetAssistantDeploymentRequest,
  Metadata,
} from '@rapidaai/react';
import { GetAssistantWebpluginDeployment } from '@rapidaai/react';
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
import { AssistantWebwidgetDeploymentDialog } from '@/app/components/base/modal/assistant-instruction-modal';
import { TabForm } from '@/app/components/form/tab-form';
import { useConfirmDialog } from '@/app/pages/assistant/actions/hooks/use-confirmation';
import {
  PrimaryButton,
  SecondaryButton,
  GhostButton,
} from '@/app/components/carbon/button';
import { InputCheckbox } from '@/app/components/carbon/form/input-checkbox';
import { ButtonSet, CheckboxGroup } from '@carbon/react';

const STEPS = [
  {
    code: 'experience',
    name: 'Experience',
    description:
      'Define the greeting, quick-start questions, and session behaviour.',
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

const ConfigureAssistantWebDeployment: FC<{ assistantId: string }> = ({
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
  const [showInstruction, setShowInstruction] = useState(false);
  const [deploymentId, setDeploymentId] = useState<string | null>(null);
  const [voiceInputEnable, setVoiceInputEnable] = useState(false);
  const [voiceOutputEnable, setVoiceOutputEnable] = useState(true);

  const [experienceConfig, setExperienceConfig] =
    useState<WebWidgetExperienceConfig>({
      greeting: undefined,
      greetingInterruptible: true,
      messageOnError: undefined,
      unclearInputTimeout: DEFAULT_UNCLEAR_INPUT_TIMEOUT,
      unclearInputMessage: DEFAULT_UNCLEAR_INPUT_MESSAGE,
      idealTimeout: DEFAULT_IDEAL_TIMEOUT,
      idealMessage: 'Are you there?',
      maxCallDuration: '300',
      idleTimeoutBackoffTimes: '2',
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
    const req = new GetAssistantDeploymentRequest();
    req.setAssistantid(assistantId);
    GetAssistantWebpluginDeployment(
      connectionConfig,
      req,
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

        setDeploymentId(deployment.getId() ?? null);
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

  const handleTabChange = (code: string) => {
    const clickedIndex = STEPS.findIndex(s => s.code === code);
    const currentIndex = STEPS.findIndex(s => s.code === activeTab);
    if (clickedIndex < currentIndex) {
      setActiveTab(code);
      setErrorMessage('');
    }
  };

  const handleNext = () => {
    setErrorMessage('');
    const idx = STEPS.findIndex(s => s.code === activeTab);

    if (activeTab === 'experience') {
      if (!experienceConfig.greeting) {
        setErrorMessage('Please provide a greeting for the assistant.');
        return;
      }
    }

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

  const handlePrevious = () => {
    setErrorMessage('');
    const idx = STEPS.findIndex(s => s.code === activeTab);
    if (idx > 0) {
      setActiveTab(STEPS[idx - 1].code);
    }
  };

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

    const req = new CreateAssistantDeploymentRequest();
    const webDeployment = new AssistantWebpluginDeployment();
    webDeployment.setAssistantid(assistantId);
    webDeployment.setGreetinginterruptible(
      experienceConfig.greetingInterruptible ?? true,
    );
    if (experienceConfig.greeting)
      webDeployment.setGreeting(experienceConfig.greeting);
    if (experienceConfig.messageOnError)
      webDeployment.setMistake(experienceConfig.messageOnError);
    if (experienceConfig.unclearInputTimeout)
      webDeployment.setUnclearinputtimeout(
        Number(experienceConfig.unclearInputTimeout),
      );
    if (experienceConfig.unclearInputMessage)
      webDeployment.setUnclearinputmessage(
        experienceConfig.unclearInputMessage,
      );
    if (experienceConfig.idealTimeout)
      webDeployment.setIdealtimeout(experienceConfig.idealTimeout);
    if (experienceConfig.idleTimeoutBackoffTimes)
      webDeployment.setIdealtimeoutbackoff(
        experienceConfig.idleTimeoutBackoffTimes,
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

    req.setPlugin(webDeployment);
    CreateAssistantWebpluginDeployment(
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
          if (deploymentId) {
            toast.success('Web widget deployment updated successfully.');
            goToDeploymentAssistant(assistantId);
            return;
          }
          setShowInstruction(true);
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
      <AssistantWebwidgetDeploymentDialog
        assistantId={assistantId}
        modalOpen={showInstruction}
        setModalOpen={() => {
          setShowInstruction(false);
          goToDeploymentAssistant(assistantId);
        }}
      />
      <div className="flex flex-col flex-1 min-h-0 bg-white dark:bg-gray-900">
        <TabForm
          formHeading="Complete all steps to configure your web widget deployment."
          activeTab={activeTab}
          onChangeActiveTab={handleTabChange}
          errorMessage={errorMessage}
          form={[
            {
              code: 'experience',
              name: 'General Experience',
              description:
                'Define the greeting, quick-start questions, and session behaviour.',
              body: (
                <div className="pt-6">
                  <ConfigureExperience
                    experienceConfig={experienceConfig}
                    setExperienceConfig={setExperienceConfig}
                  />
                </div>
              ),
              actions: [
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
                    onClick={handleNext}
                  >
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
              body: (
                <div className="pt-6">
                  <div className="px-6">
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
              ),
              actions: [
                <ButtonSet className="!w-full [&>button]:!flex-1 [&>button]:!max-w-none">
                  <GhostButton size="lg" onClick={handlePrevious}>
                    Previous
                  </GhostButton>
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
                    onClick={handleNext}
                  >
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
              body: (
                <div className="pt-6">
                  <div className="px-6">
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
              ),
              actions: [
                <ButtonSet className="!w-full [&>button]:!flex-1 [&>button]:!max-w-none">
                  <GhostButton size="lg" onClick={handlePrevious}>
                    Previous
                  </GhostButton>
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
                    Deploy Web Widget
                  </PrimaryButton>
                </ButtonSet>,
              ],
            },
          ]}
        />
      </div>
    </>
  );
};
