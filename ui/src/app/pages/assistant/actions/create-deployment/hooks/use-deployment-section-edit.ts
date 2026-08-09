import { useCallback, useState } from 'react';
import {
  AssistantApiDeployment,
  AssistantDebuggerDeployment,
  AssistantPhoneDeployment,
  AssistantWebpluginDeployment,
  ConnectionConfig,
  CreateAssistantDebuggerDeployment,
  CreateAssistantApiDeployment,
  CreateAssistantPhoneDeployment,
  CreateAssistantWebpluginDeployment,
  CreateAssistantDeploymentRequest,
  DeploymentAudioProvider,
  GetAssistantDebuggerDeployment,
  GetAssistantApiDeployment,
  GetAssistantPhoneDeployment,
  GetAssistantWebpluginDeployment,
  GetAssistantDeploymentRequest,
  Metadata,
} from '@rapidaai/react';
import toast from 'react-hot-toast/headless';
import { connectionConfig } from '@/configs';
import { useCurrentCredential } from '@/hooks/use-credential';
import { useAllProviderCredentials } from '@/hooks/use-model';
import { useRapidaStore } from '@/hooks';
import {
  ExperienceConfig,
} from '@/app/pages/assistant/actions/create-deployment/commons/configure-experience';
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
import {
  GetDefaultTelephonyConfigIfInvalid,
  ValidateTelephonyOptions,
} from '@/app/components/providers/telephony';

export type EditSection =
  | 'telephony'
  | 'experience'
  | 'voice-input'
  | 'voice-output';
export type DeploymentType = 'debugger' | 'api' | 'web' | 'phone';

type AudioConfig = { provider: string; parameters: Metadata[] };

type TelephonyConfig = { provider: string; parameters: Metadata[] };

type ExistingConfig = {
  experience: ExperienceConfig;
  inputAudio?: AudioConfig;
  outputAudio?: AudioConfig;
  telephony?: TelephonyConfig;
};

const DEFAULT_EXPERIENCE: ExperienceConfig = {
  greeting: undefined,
  greetingInterruptible: true,
  messageOnError: undefined,
  unclearInputTimeout: undefined,
  unclearInputMessage: undefined,
  idealTimeout: '30',
  idealMessage: 'Are you there?',
  maxCallDuration: '300',
  idleTimeoutBackoffTimes: '2',
};

const getDeploymentFetcher = (type: DeploymentType) => {
  switch (type) {
    case 'debugger':
      return GetAssistantDebuggerDeployment;
    case 'api':
      return GetAssistantApiDeployment;
    case 'web':
      return GetAssistantWebpluginDeployment;
    case 'phone':
      return GetAssistantPhoneDeployment;
  }
};

const getDeploymentCreator = (type: DeploymentType) => {
  switch (type) {
    case 'debugger':
      return CreateAssistantDebuggerDeployment;
    case 'api':
      return CreateAssistantApiDeployment;
    case 'web':
      return CreateAssistantWebpluginDeployment;
    case 'phone':
      return CreateAssistantPhoneDeployment;
  }
};

const DEPLOYMENT_LABELS: Record<DeploymentType, string> = {
  debugger: 'Debugger',
  api: 'SDK / API',
  web: 'Web Widget',
  phone: 'Phone Call',
};

export function useDeploymentSectionEdit(
  assistantId: string | undefined,
  onSuccess: () => void,
) {
  const { token, authId, projectId } = useCurrentCredential();
  const { providerCredentials } = useAllProviderCredentials();
  const { showLoader, hideLoader } = useRapidaStore();

  const [activeEdit, setActiveEdit] = useState<{
    type: DeploymentType;
    section: EditSection;
  } | null>(null);
  const [editError, setEditError] = useState('');
  const [isSaving, setIsSaving] = useState(false);

  const [voiceInputEnable, setVoiceInputEnable] = useState(true);
  const [voiceOutputEnable, setVoiceOutputEnable] = useState(true);

  const [experienceConfig, setExperienceConfig] = useState<ExperienceConfig>({
    ...DEFAULT_EXPERIENCE,
  });

  const [audioInputConfig, setAudioInputConfig] = useState<AudioConfig>({
    provider: 'deepgram',
    parameters: GetDefaultSpeechToTextIfInvalid(
      'deepgram',
      GetDefaultMicrophoneConfig(),
    ),
  });

  const [audioOutputConfig, setAudioOutputConfig] = useState<AudioConfig>({
    provider: 'cartesia',
    parameters: GetDefaultTextToSpeechIfInvalid(
      'cartesia',
      GetDefaultSpeakerConfig(),
    ),
  });

  const [telephonyConfig, setTelephonyConfig] = useState<TelephonyConfig>({
    provider: 'twilio',
    parameters: [],
  });

  const [existingConfig, setExistingConfig] = useState<ExistingConfig>({
    experience: { ...DEFAULT_EXPERIENCE },
  });

  const authHeaders = useCallback(
    () =>
      ConnectionConfig.WithDebugger({
        authorization: token,
        userId: authId,
        projectId,
      }),
    [token, authId, projectId],
  );

  const getProviderCredentials = (provider: string) =>
    providerCredentials.filter(c => c.getProvider() === provider);

  const loadConfig = useCallback(
    (type: DeploymentType) => {
      if (!assistantId) return Promise.resolve();
      const request = new GetAssistantDeploymentRequest();
      request.setAssistantid(assistantId);
      const fetcher = getDeploymentFetcher(type);
      return fetcher(connectionConfig, request, authHeaders()).then(
        (response: any) => {
          const deployment = response?.getData();
          if (!deployment) return;

          const fetchedExperience: ExperienceConfig = {
            greeting: deployment.getGreeting(),
            greetingInterruptible: deployment.hasGreetinginterruptible?.()
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
            ...(type === 'web' && deployment.getSuggestionList
              ? { suggestions: deployment.getSuggestionList() || [] }
              : {}),
          };
          setExperienceConfig(fetchedExperience);

          let fetchedInputAudio: AudioConfig | undefined;
          if (deployment.getInputaudio()) {
            const p = deployment.getInputaudio()!;
            fetchedInputAudio = {
              provider: p.getAudioprovider() || 'deepgram',
              parameters: p.getAudiooptionsList() || [],
            };
            setVoiceInputEnable(true);
            setAudioInputConfig({
              provider: p.getAudioprovider() || 'deepgram',
              parameters: GetDefaultSpeechToTextIfInvalid(
                p.getAudioprovider() || 'deepgram',
                GetDefaultMicrophoneConfig(p.getAudiooptionsList() || []),
              ),
            });
          } else {
            setVoiceInputEnable(false);
          }

          let fetchedOutputAudio: AudioConfig | undefined;
          if (deployment.getOutputaudio()) {
            const p = deployment.getOutputaudio()!;
            fetchedOutputAudio = {
              provider: p.getAudioprovider() || 'cartesia',
              parameters: p.getAudiooptionsList() || [],
            };
            setVoiceOutputEnable(true);
            setAudioOutputConfig({
              provider: p.getAudioprovider() || 'cartesia',
              parameters: GetDefaultTextToSpeechIfInvalid(
                p.getAudioprovider() || 'cartesia',
                GetDefaultSpeakerConfig(p.getAudiooptionsList() || []),
              ),
            });
          } else {
            setVoiceOutputEnable(false);
          }

          let fetchedTelephony: TelephonyConfig | undefined;
          if (type === 'phone' && deployment.getPhoneprovidername?.()) {
            const provider = deployment.getPhoneprovidername() || '';
            fetchedTelephony = {
              provider,
              parameters: GetDefaultTelephonyConfigIfInvalid(
                provider,
                deployment.getPhoneoptionsList?.() || [],
              ),
            };
            setTelephonyConfig(fetchedTelephony);
          }

          setExistingConfig({
            experience: fetchedExperience,
            inputAudio: fetchedInputAudio,
            outputAudio: fetchedOutputAudio,
            telephony: fetchedTelephony,
          });
        },
      );
    },
    [assistantId, authHeaders],
  );

  const openEditModal = useCallback(
    (type: DeploymentType, section: EditSection) => {
      if (!assistantId) return;
      setEditError('');
      showLoader('block');
      loadConfig(type)
        .then(() => setActiveEdit({ type, section }))
        .catch(() => {
          toast.error(`Unable to load ${DEPLOYMENT_LABELS[type]} settings.`);
        })
        .finally(() => hideLoader());
    },
    [assistantId, loadConfig, showLoader, hideLoader],
  );

  const closeEditModal = () => setActiveEdit(null);

  const saveSection = () => {
    if (!assistantId || !activeEdit) return;
    const { type, section } = activeEdit;
    setIsSaving(true);
    setEditError('');

    if (section === 'telephony') {
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
        setIsSaving(false);
        setEditError('Please provide a valid telephony configuration.');
        return;
      }
    }

    if (section === 'voice-input' && voiceInputEnable) {
      if (!audioInputConfig.provider) {
        setIsSaving(false);
        setEditError('Please select a speech-to-text provider.');
        return;
      }
      const err = ValidateSpeechToTextIfInvalid(
        audioInputConfig.provider,
        audioInputConfig.parameters,
        getProviderCredentials(audioInputConfig.provider),
      );
      if (err) {
        setIsSaving(false);
        setEditError(err);
        return;
      }
    }

    if (section === 'voice-output' && voiceOutputEnable) {
      if (!audioOutputConfig.provider) {
        setIsSaving(false);
        setEditError('Please select a text-to-speech provider.');
        return;
      }
      const err = ValidateTextToSpeechIfInvalid(
        audioOutputConfig.provider,
        audioOutputConfig.parameters,
        getProviderCredentials(audioOutputConfig.provider),
      );
      if (err) {
        setIsSaving(false);
        setEditError(err);
        return;
      }
    }

    const resolvedExperience =
      section === 'experience' ? experienceConfig : existingConfig.experience;

    const buildAudioInput = () => {
      if (section === 'voice-input') {
        if (!voiceInputEnable) return undefined;
        const a = new DeploymentAudioProvider();
        a.setAudioprovider(audioInputConfig.provider);
        a.setAudiooptionsList(audioInputConfig.parameters);
        return a;
      }
      if (existingConfig.inputAudio) {
        const a = new DeploymentAudioProvider();
        a.setAudioprovider(existingConfig.inputAudio.provider);
        a.setAudiooptionsList(existingConfig.inputAudio.parameters);
        return a;
      }
      return undefined;
    };

    const buildAudioOutput = () => {
      if (section === 'voice-output') {
        if (!voiceOutputEnable) return undefined;
        const a = new DeploymentAudioProvider();
        a.setAudioprovider(audioOutputConfig.provider);
        a.setAudiooptionsList(audioOutputConfig.parameters);
        return a;
      }
      if (existingConfig.outputAudio) {
        const a = new DeploymentAudioProvider();
        a.setAudioprovider(existingConfig.outputAudio.provider);
        a.setAudiooptionsList(existingConfig.outputAudio.parameters);
        return a;
      }
      return undefined;
    };

    const inputAudio = buildAudioInput();
    const outputAudio = buildAudioOutput();

    const applyCommonFields = (deployment: any) => {
      deployment.setAssistantid(assistantId);
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
        deployment.setUnclearinputmessage(
          resolvedExperience.unclearInputMessage,
        );
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
      if (inputAudio) deployment.setInputaudio(inputAudio);
      if (outputAudio) deployment.setOutputaudio(outputAudio);
    };

    const req = new CreateAssistantDeploymentRequest();

    if (type === 'debugger') {
      const d = new AssistantDebuggerDeployment();
      applyCommonFields(d);
      req.setDebugger(d);
    } else if (type === 'api') {
      const d = new AssistantApiDeployment();
      applyCommonFields(d);
      req.setApi(d);
    } else if (type === 'web') {
      const d = new AssistantWebpluginDeployment();
      applyCommonFields(d);
      d.setSuggestionList(resolvedExperience.suggestions || []);
      d.setHelpcenterenabled(false);
      d.setProductcatalogenabled(false);
      d.setArticlecatalogenabled(false);
      d.setUploadfileenabled(false);
      req.setPlugin(d);
    } else if (type === 'phone') {
      const d = new AssistantPhoneDeployment();
      applyCommonFields(d);
      const resolvedTelephony =
        section === 'telephony' ? telephonyConfig : existingConfig.telephony;
      if (resolvedTelephony) {
        d.setPhoneprovidername(resolvedTelephony.provider);
        d.setPhoneoptionsList(
          GetDefaultTelephonyConfigIfInvalid(
            resolvedTelephony.provider,
            resolvedTelephony.parameters,
          ),
        );
      }
      req.setPhone(d);
    }

    const creator = getDeploymentCreator(type);
    creator(connectionConfig, req, authHeaders())
      .then((response: any) => {
        if (response?.getData() && response.getSuccess()) {
          toast.success(
            `${DEPLOYMENT_LABELS[type]} deployment updated successfully.`,
          );
          setActiveEdit(null);
          onSuccess();
          return;
        }
        setEditError(
          response?.getError()?.getHumanmessage() ||
            'Unable to update deployment. Please try again.',
        );
      })
      .catch(() =>
        setEditError(
          'Unable to update deployment configuration. Please try again.',
        ),
      )
      .finally(() => setIsSaving(false));
  };

  return {
    activeEdit,
    editError,
    isSaving,
    voiceInputEnable,
    setVoiceInputEnable,
    voiceOutputEnable,
    setVoiceOutputEnable,
    experienceConfig,
    setExperienceConfig,
    audioInputConfig,
    setAudioInputConfig,
    audioOutputConfig,
    setAudioOutputConfig,
    telephonyConfig,
    setTelephonyConfig,
    openEditModal,
    closeEditModal,
    saveSection,
  };
}
