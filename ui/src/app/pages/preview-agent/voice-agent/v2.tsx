import {
  type ComponentProps,
  type ChangeEvent as ReactChangeEvent,
  type KeyboardEvent as ReactKeyboardEvent,
  type MouseEvent as ReactMouseEvent,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import { Navigate, useNavigate, useParams } from 'react-router-dom';
import {
  ArrowLeft,
  Chat,
  Close,
  Microphone,
  MicrophoneOff,
  Model,
  StopFilledAlt,
  ValueVariable,
  VoiceActivate,
  VoiceMode,
  VolumeUp,
} from '@carbon/icons-react';
import {
  CarbonTheme,
  ChatCustomElement,
  MessageInputType,
  MessageResponseTypes,
  type ChatInstance,
  type CustomSendMessageOptions,
  type HistoryItem,
  type MessageRequest,
  WriteableElementName,
} from '@carbon/ai-chat';
import {
  AgentConfig,
  type Assistant,
  Channel,
  InputOptions,
  StringToAny,
  VoiceAgent,
  useSelectInputDeviceAgent,
} from '@rapidaai/react';
import {
  Accordion,
  AccordionItem,
  Dropdown,
  FluidForm,
  FluidNumberInput,
  FluidTextArea,
  FluidTextInput,
  IconButton as CarbonIconButton,
  InlineLoading,
  InlineNotification,
  Layer,
  StructuredListBody,
  StructuredListCell,
  StructuredListRow,
  StructuredListWrapper,
  Toggletip as CarbonToggletip,
  ToggletipActions,
  ToggletipButton,
  ToggletipContent,
} from '@carbon/react';
import {
  CopyButton,
  IconOnlyButton,
  PrimaryButton,
} from '@/app/components/ui/primitives';
import { ConfirmDialog } from '@/app/components/dialogs/shared';
import { createVoiceAgentDebuggerConnection } from '@/clients/runtime.client';
import { useCurrentCredential } from '@/hooks/use-credential';
import { useTheme } from '@/theme/theme-provider';
import { InputVarType } from '@/models/common';
import { cn } from '@/utils';
import { PreviewAgentHeader } from './preview-agent-header';

const Toggletip =
  (CarbonToggletip as unknown as { default?: typeof CarbonToggletip })
    .default ?? CarbonToggletip;

type VoicePhase = 'idle' | 'connecting' | 'listening';

function VoiceActionIcon(props: ComponentProps<typeof VoiceMode>) {
  return (
    <VoiceMode
      {...props}
      style={{ ...props.style, fill: 'var(--cds-link-primary)' }}
    />
  );
}

function StopActionIcon(props: ComponentProps<typeof StopFilledAlt>) {
  return (
    <StopFilledAlt
      {...props}
      style={{ ...props.style, fill: 'var(--cds-support-error)' }}
    />
  );
}

type PromptVariable = {
  name: string;
  type: string;
  value: string;
};

type ActiveAssistantStream = {
  itemId: string;
  requestId?: string;
  text: string;
};

type ConfigurationOption = {
  id: string;
  label: string;
  key: string;
  value: string;
};

const EDITABLE_ARGUMENT_TYPES = new Set<string>([
  InputVarType.stringInput,
  InputVarType.textInput,
  InputVarType.paragraph,
  InputVarType.number,
  InputVarType.json,
  InputVarType.url,
]);

const MULTILINE_ARGUMENT_TYPES = new Set<string>([
  InputVarType.textInput,
  InputVarType.paragraph,
  InputVarType.json,
]);

function formatArgumentType(type: string) {
  if (type === InputVarType.json) return 'JSON';
  return type.charAt(0).toUpperCase() + type.slice(1);
}

function formatConfigurationLabel(key: string) {
  if (key.includes('voice')) return 'Voice';
  if (key.endsWith('model') || key === 'model.name') return 'Model';
  if (key.endsWith('top_p')) return 'Top P';

  const name = key.split('.').at(-1)?.replaceAll('_', ' ') || key;
  return name.charAt(0).toUpperCase() + name.slice(1);
}

function isVisibleConfigurationKey(key: string) {
  const configurationKey = key.toLowerCase();
  return (
    configurationKey !== 'credential' &&
    !configurationKey.endsWith('credential_id')
  );
}

function isCopyableConfigurationKey(key: string) {
  const configurationKey = key.toLowerCase();
  return (
    configurationKey === 'model.name' ||
    configurationKey.endsWith('.model') ||
    configurationKey.endsWith('.url') ||
    configurationKey.endsWith('_url') ||
    configurationKey.endsWith('.id') ||
    configurationKey.endsWith('_id')
  );
}

function VoiceMicrophoneSelector({
  voiceAgent,
  onError,
}: {
  voiceAgent: VoiceAgent;
  onError: (error: Error) => void;
}) {
  const { devices, activeDeviceId, setActiveMediaDevice } =
    useSelectInputDeviceAgent({
      voiceAgent,
      requestPermissions: true,
      onError,
    });
  const microphoneOptions = useMemo(
    () =>
      devices.map((device, index) => ({
        id: device.deviceId,
        label: device.label || `Microphone ${index + 1}`,
      })),
    [devices],
  );
  const selectedMicrophone =
    microphoneOptions.find(option => option.id === activeDeviceId) ??
    microphoneOptions[0] ??
    null;

  return (
    <Dropdown
      id="voice-microphone"
      aria-label="Microphone"
      titleText="Microphone"
      hideLabel
      label="Select microphone"
      size="sm"
      direction="top"
      autoAlign
      items={microphoneOptions}
      itemToString={item => item?.label ?? ''}
      selectedItem={selectedMicrophone}
      onChange={({ selectedItem }) => {
        if (!selectedItem) return;
        void setActiveMediaDevice(selectedItem.id).catch(onError);
      }}
      className="w-full sm:w-48"
    />
  );
}

export function PreviewVoiceAgentV2({
  embedded = false,
}: {
  embedded?: boolean;
}) {
  const { assistantId } = useParams();
  const { user, authId, token, projectId } = useCurrentCredential();

  if (!assistantId || !user?.name) {
    return <Navigate to="/404" replace />;
  }

  return (
    <PreviewVoiceAgentV2Workspace
      assistantId={assistantId}
      userName={user.name}
      authId={authId}
      token={token}
      projectId={projectId}
      embedded={embedded}
    />
  );
}

function PreviewVoiceAgentV2Workspace({
  assistantId,
  userName,
  authId,
  token,
  projectId,
  embedded,
}: {
  assistantId: string;
  userName: string;
  authId: string;
  token: string;
  projectId: string;
  embedded: boolean;
}) {
  const navigate = useNavigate();
  const { resolvedMode } = useTheme();
  const chatBackgroundColor = resolvedMode === 'dark' ? '#161616' : '#f4f4f4';
  const chatActionColor = resolvedMode === 'dark' ? '#262626' : '#ffffff';
  const chatContainerRef = useRef<HTMLDivElement>(null);
  const conversationRef = useRef<HTMLElement>(null);
  const panelRef = useRef<HTMLElement>(null);
  const panelTriggerRef = useRef<HTMLButtonElement | null>(null);
  const chatInstanceRef = useRef<ChatInstance | null>(null);
  const pendingRequestIdsRef = useRef<string[]>([]);
  const activeAssistantStreamsRef = useRef<Map<string, ActiveAssistantStream>>(
    new Map(),
  );
  const voiceTranscriptIdRef = useRef<string>();
  const pendingVoiceConnectionRef = useRef<Promise<void> | null>(null);
  const runtimeResetRef = useRef<Promise<void> | null>(null);
  const isVoiceRequestedRef = useRef(false);
  const argumentsRequireReconnectRef = useRef(false);
  const responseSequence = useRef(0);
  const voiceAgent = useMemo(
    () =>
      new VoiceAgent(
        createVoiceAgentDebuggerConnection({
          token,
          userId: authId,
          projectId,
        }),
        new AgentConfig(
          assistantId,
          new InputOptions([Channel.Audio, Channel.Text], Channel.Text),
        )
          .setUserIdentifier(authId, userName)
          .addKeywords([userName])
          .addMetadata('authId', StringToAny(authId))
          .addMetadata('projectId', StringToAny(projectId)),
      ),
    [assistantId, authId, projectId, token, userName],
  );
  const [assistant, setAssistant] = useState<Assistant | null>(null);
  const [isConfigurationOpen, setIsConfigurationOpen] = useState(false);
  const [isSessionConnected, setIsSessionConnected] = useState(false);
  const [voicePhase, setVoicePhase] = useState<VoicePhase>('idle');
  const [isVoiceMuted, setIsVoiceMuted] = useState(false);
  const [isEndConfirmationOpen, setIsEndConfirmationOpen] = useState(false);
  const [runtimeError, setRuntimeError] = useState('');
  const [isOverlayPanel, setIsOverlayPanel] = useState(
    () =>
      typeof window !== 'undefined' &&
      window.matchMedia('(max-width: 1055px)').matches,
  );
  const [isSmallViewport, setIsSmallViewport] = useState(
    () =>
      typeof window !== 'undefined' &&
      window.matchMedia('(max-width: 671px)').matches,
  );
  const [promptVariables, setPromptVariables] = useState<PromptVariable[]>([]);

  const voiceStack = useMemo(() => {
    const model = assistant?.getAssistantprovidermodel();
    const deployment = assistant?.getDebuggerdeployment();
    const speechToText = deployment?.getInputaudio();
    const textToSpeech = deployment?.getOutputaudio();
    const unavailableConfiguration: ConfigurationOption[] = [
      {
        id: 'status',
        label: 'Status',
        key: 'status',
        value: assistant ? 'Not configured' : 'Loading',
      },
    ];

    return [
      {
        id: 'language-model',
        icon: Model,
        label: 'Language model',
        configuration: model
          ? [
              {
                id: 'model-provider',
                label: 'Provider',
                key: 'provider',
                value: model.getModelprovidername(),
              },
              ...model
                .getAssistantmodeloptionsList()
                .filter(option => isVisibleConfigurationKey(option.getKey()))
                .map(option => ({
                  id: `model-${option.getKey()}`,
                  label: formatConfigurationLabel(option.getKey()),
                  key: option.getKey(),
                  value: option.getValue(),
                })),
            ]
          : unavailableConfiguration,
      },
      {
        id: 'speech-to-text',
        icon: VoiceActivate,
        label: 'Speech to text',
        configuration: speechToText
          ? [
              {
                id: 'stt-provider',
                label: 'Provider',
                key: 'provider',
                value: speechToText.getAudioprovider(),
              },
              ...speechToText
                .getAudiooptionsList()
                .filter(option => isVisibleConfigurationKey(option.getKey()))
                .map(option => ({
                  id: `stt-${option.getKey()}`,
                  label: formatConfigurationLabel(option.getKey()),
                  key: option.getKey(),
                  value: option.getValue(),
                })),
            ]
          : unavailableConfiguration,
      },
      {
        id: 'text-to-speech',
        icon: VolumeUp,
        label: 'Text to speech',
        configuration: textToSpeech
          ? [
              {
                id: 'tts-provider',
                label: 'Provider',
                key: 'provider',
                value: textToSpeech.getAudioprovider(),
              },
              ...textToSpeech
                .getAudiooptionsList()
                .filter(option => isVisibleConfigurationKey(option.getKey()))
                .map(option => ({
                  id: `tts-${option.getKey()}`,
                  label: formatConfigurationLabel(option.getKey()),
                  key: option.getKey(),
                  value: option.getValue(),
                })),
            ]
          : unavailableConfiguration,
      },
    ];
  }, [assistant]);

  const handleRuntimeError = useCallback((error: unknown) => {
    setRuntimeError(
      error instanceof Error && error.message
        ? error.message
        : 'The voice session could not complete this action.',
    );
  }, []);

  const resetRuntime = useCallback(() => {
    if (runtimeResetRef.current) return runtimeResetRef.current;

    const pendingVoiceConnection = pendingVoiceConnectionRef.current;
    const reset = (async () => {
      try {
        try {
          await voiceAgent.disconnect();
        } finally {
          if (pendingVoiceConnection) {
            await pendingVoiceConnection.catch(() => undefined);
            await voiceAgent.disconnect();
          }
        }
      } finally {
        voiceAgent.agentConfiguration.inputOptions.channel = Channel.Text;
        voiceAgent.agentConfiguration.outputOptions.channel = Channel.Text;
      }
    })();
    runtimeResetRef.current = reset;
    const clearReset = () => {
      if (runtimeResetRef.current === reset) runtimeResetRef.current = null;
    };
    reset.then(clearReset, clearReset);
    return reset;
  }, [voiceAgent]);

  const applyPendingArguments = useCallback(async () => {
    const activeReset = runtimeResetRef.current;
    if (activeReset) await activeReset;
    if (!argumentsRequireReconnectRef.current) return;
    if (voiceAgent.isConnected || pendingVoiceConnectionRef.current) {
      await resetRuntime();
    }
    argumentsRequireReconnectRef.current = false;
  }, [resetRuntime, voiceAgent]);

  const loadConversationHistory = useCallback(
    async (instance: ChatInstance): Promise<HistoryItem[]> => {
      chatInstanceRef.current = instance;
      return [];
    },
    [],
  );

  const makeTranscriptFocusable = useCallback(() => {
    const chatElement =
      chatContainerRef.current?.querySelector<HTMLElement>('cds-aichat-react');
    const transcript = chatElement?.shadowRoot?.querySelector<HTMLElement>(
      '.cds-aichat--messages__wrapper',
    );
    transcript?.setAttribute('tabindex', '0');
  }, []);

  const stopAssistantStreams = useCallback(async () => {
    const instance = chatInstanceRef.current;
    const streams = Array.from(activeAssistantStreamsRef.current.entries());
    activeAssistantStreamsRef.current.clear();
    if (!instance) return;

    await Promise.all(
      streams.map(([responseId, stream]) =>
        instance.messaging.addMessageChunk({
          streaming_metadata: { response_id: responseId },
          final_response: {
            id: responseId,
            request_id: stream.requestId,
            output: {
              generic: [
                {
                  response_type: MessageResponseTypes.TEXT,
                  text: stream.text,
                  streaming_metadata: {
                    id: stream.itemId,
                    stream_stopped: true,
                  },
                },
              ],
            },
          },
        }),
      ),
    );
  }, []);

  const disconnectChatSession = useCallback(async () => {
    setIsSessionConnected(false);
    setRuntimeError('');
    pendingRequestIdsRef.current = [];
    try {
      await stopAssistantStreams();
      await resetRuntime();
    } catch (error) {
      handleRuntimeError(error);
    }
  }, [handleRuntimeError, resetRuntime, stopAssistantStreams]);

  useEffect(() => {
    const panelMediaQuery = window.matchMedia('(max-width: 1055px)');
    const smallViewportMediaQuery = window.matchMedia('(max-width: 671px)');
    const updatePanelMode = () => setIsOverlayPanel(panelMediaQuery.matches);
    const updateSmallViewport = () =>
      setIsSmallViewport(smallViewportMediaQuery.matches);
    updatePanelMode();
    updateSmallViewport();
    panelMediaQuery.addEventListener('change', updatePanelMode);
    smallViewportMediaQuery.addEventListener('change', updateSmallViewport);

    return () => {
      panelMediaQuery.removeEventListener('change', updatePanelMode);
      smallViewportMediaQuery.removeEventListener(
        'change',
        updateSmallViewport,
      );
    };
  }, []);

  useEffect(() => {
    let isActive = true;
    setAssistant(null);
    setPromptVariables([]);
    setRuntimeError('');

    voiceAgent
      .getAssistant()
      .then(response => {
        if (!isActive) return;
        const loadedAssistant = response.getSuccess()
          ? response.getData()
          : null;
        if (!loadedAssistant) {
          setRuntimeError('Unable to load assistant configuration.');
          return;
        }

        setAssistant(loadedAssistant);
        const variables =
          loadedAssistant
            .getAssistantprovidermodel()
            ?.getTemplate()
            ?.getPromptvariablesList() ?? [];
        setPromptVariables(
          variables.map(variable => ({
            name: variable.getName(),
            type: variable.getType() || InputVarType.stringInput,
            value: variable.getDefaultvalue(),
          })),
        );
        variables.forEach(variable => {
          if (variable.getDefaultvalue()) {
            voiceAgent.agentConfiguration.addArgument(
              variable.getName(),
              variable.getDefaultvalue(),
            );
          }
        });
      })
      .catch(() => {
        if (isActive) {
          setRuntimeError('Unable to load assistant configuration.');
        }
      });

    return () => {
      isActive = false;
    };
  }, [voiceAgent]);

  useEffect(() => {
    let isActive = true;
    const handleConnectionState = (state: string) => {
      if (!isActive) return;
      const isVoiceConnection =
        isVoiceRequestedRef.current &&
        voiceAgent.agentConfiguration.inputOptions.channel === Channel.Audio;
      if (state === 'connecting' && isVoiceConnection) {
        setVoicePhase('connecting');
      } else if (state === 'connected' && isVoiceConnection) {
        setVoicePhase('listening');
      }
      if (state === 'connected') {
        setIsSessionConnected(true);
      } else if (state === 'disconnected') {
        pendingRequestIdsRef.current = [];
        setIsSessionConnected(false);
        setVoicePhase('idle');
        setIsVoiceMuted(false);
        void stopAssistantStreams().catch(handleRuntimeError);
      }
    };
    const handleAgentError = (_type: string, error: string) => {
      if (!isActive) return;
      isVoiceRequestedRef.current = false;
      pendingRequestIdsRef.current = [];
      setIsSessionConnected(false);
      voiceAgent.agentConfiguration.inputOptions.channel = Channel.Text;
      voiceAgent.agentConfiguration.outputOptions.channel = Channel.Text;
      setVoicePhase('idle');
      setIsVoiceMuted(false);
      setRuntimeError(error || 'Voice session failed.');
      void stopAssistantStreams().catch(handleRuntimeError);
    };

    voiceAgent.on('onConnectionStateEvent', handleConnectionState);
    voiceAgent.on('onErrorEvent', handleAgentError);
    voiceAgent.registerCallback({
      onAssistantMessage: message => {
        if (!isActive || !message) return;
        const instance = chatInstanceRef.current;
        if (!instance) return;

        const responseId =
          message.id || `assistant-response-${++responseSequence.current}`;
        if (message.completed === false) {
          const chunkText = message.messageText || '';
          if (!chunkText) return;

          let stream = activeAssistantStreamsRef.current.get(responseId);
          if (!stream) {
            stream = {
              itemId: `${responseId}-text`,
              requestId: pendingRequestIdsRef.current.shift(),
              text: '',
            };
            activeAssistantStreamsRef.current.set(responseId, stream);
          }
          stream.text += chunkText;
          void instance.messaging
            .addMessageChunk({
              streaming_metadata: { response_id: responseId },
              partial_item: {
                response_type: MessageResponseTypes.TEXT,
                text: chunkText,
                streaming_metadata: { id: stream.itemId },
              },
            })
            .catch(handleRuntimeError);
          return;
        }

        const storedMessage = voiceAgent.messages.find(
          current => current.id === message.id && current.role === 'system',
        );
        const stream = activeAssistantStreamsRef.current.get(responseId);
        const text =
          storedMessage?.messages.join('') ||
          stream?.text ||
          message.messageText?.trim();
        if (!text) return;

        const requestId =
          stream?.requestId ?? pendingRequestIdsRef.current.shift();
        if (stream) {
          activeAssistantStreamsRef.current.delete(responseId);
          void instance.messaging
            .addMessageChunk({
              streaming_metadata: { response_id: responseId },
              final_response: {
                id: responseId,
                request_id: requestId,
                output: {
                  generic: [
                    {
                      response_type: MessageResponseTypes.TEXT,
                      text,
                      streaming_metadata: { id: stream.itemId },
                    },
                  ],
                },
              },
            })
            .catch(handleRuntimeError);
          return;
        }

        void instance.messaging
          .addMessage({
            id: responseId,
            request_id: requestId,
            output: {
              generic: [
                {
                  response_type: MessageResponseTypes.TEXT,
                  text,
                },
              ],
            },
          })
          .catch(handleRuntimeError);
      },
      onUserMessage: message => {
        if (
          !isActive ||
          !message ||
          message.completed === false ||
          !message.messageText?.trim() ||
          voiceAgent.agentConfiguration.inputOptions.channel !== Channel.Audio
        ) {
          return;
        }
        const instance = chatInstanceRef.current;
        if (!instance) return;

        voiceTranscriptIdRef.current = message.id;
        if (message.id) pendingRequestIdsRef.current.push(message.id);
        void instance
          .send({
            id: message.id,
            input: {
              message_type: MessageInputType.TEXT,
              text: message.messageText,
            },
          })
          .catch(error => {
            pendingRequestIdsRef.current = pendingRequestIdsRef.current.filter(
              requestId => requestId !== message.id,
            );
            if (voiceTranscriptIdRef.current === message.id) {
              voiceTranscriptIdRef.current = undefined;
            }
            handleRuntimeError(error);
          });
      },
      onConversationError: error => {
        if (isActive) {
          pendingRequestIdsRef.current = [];
          setIsSessionConnected(false);
          setRuntimeError(error.message || 'Voice session failed.');
          void stopAssistantStreams().catch(handleRuntimeError);
        }
      },
    });

    return () => {
      isActive = false;
      voiceAgent.off('onConnectionStateEvent', handleConnectionState);
      voiceAgent.off('onErrorEvent', handleAgentError);
    };
  }, [handleRuntimeError, stopAssistantStreams, voiceAgent]);

  useEffect(
    () => () => {
      chatInstanceRef.current = null;
      isVoiceRequestedRef.current = false;
      voiceAgent.agentConfiguration.inputOptions.channel = Channel.Text;
      voiceAgent.agentConfiguration.outputOptions.channel = Channel.Text;
      void resetRuntime();
    },
    [resetRuntime, voiceAgent],
  );

  useEffect(() => {
    const conversation = conversationRef.current;
    if (!conversation) return;

    if (isConfigurationOpen && isOverlayPanel) {
      conversation.setAttribute('inert', '');
      window.requestAnimationFrame(() => {
        panelRef.current?.querySelector<HTMLButtonElement>('button')?.focus();
      });
      return () => conversation.removeAttribute('inert');
    }

    conversation.removeAttribute('inert');
  }, [isConfigurationOpen, isOverlayPanel]);

  const sendRuntimeMessage = useCallback(
    async (
      request: MessageRequest,
      options: CustomSendMessageOptions,
      instance: ChatInstance,
    ) => {
      if (options.signal.aborted) return;

      chatInstanceRef.current = instance;
      if (request.history?.is_welcome_request) {
        await instance.messaging.addMessage({
          id: request.id ? `welcome-${request.id}` : 'debugger-welcome',
          request_id: request.id,
          output: {
            generic: [
              {
                response_type: MessageResponseTypes.TEXT,
                text: 'Hello! How can I help you today?',
              },
            ],
          },
        });
        return;
      }

      const message = request.input.text?.trim();
      if (!message) return;

      if (voiceTranscriptIdRef.current === request.id) {
        voiceTranscriptIdRef.current = undefined;
        return;
      }

      setRuntimeError('');
      try {
        await applyPendingArguments();
        if (options.signal.aborted) return;
        if (request.id) pendingRequestIdsRef.current.push(request.id);
        await voiceAgent.onSendText(message);
      } catch (error) {
        pendingRequestIdsRef.current = pendingRequestIdsRef.current.filter(
          requestId => requestId !== request.id,
        );
        throw error;
      }
    },
    [applyPendingArguments, voiceAgent],
  );

  const closePanel = useCallback(() => {
    setIsConfigurationOpen(false);
    window.requestAnimationFrame(() => panelTriggerRef.current?.focus());
  }, []);

  const toggleConfiguration = useCallback(
    (event: ReactMouseEvent<HTMLButtonElement>) => {
      panelTriggerRef.current = event.currentTarget;
      if (isConfigurationOpen) {
        closePanel();
        return;
      }
      setIsConfigurationOpen(true);
    },
    [closePanel, isConfigurationOpen],
  );

  useEffect(() => {
    if (!isConfigurationOpen || !isOverlayPanel) return;

    const closeOnEscape = (event: KeyboardEvent) => {
      if (event.key !== 'Escape') return;
      event.preventDefault();
      closePanel();
    };
    document.addEventListener('keydown', closeOnEscape, true);
    return () => document.removeEventListener('keydown', closeOnEscape, true);
  }, [closePanel, isConfigurationOpen, isOverlayPanel]);

  const focusTextComposer = useCallback(() => {
    window.requestAnimationFrame(() => {
      const chatElement =
        chatContainerRef.current?.querySelector<HTMLElement>(
          'cds-aichat-react',
        );
      chatElement?.shadowRoot
        ?.querySelector<HTMLElement>('[contenteditable="true"]')
        ?.focus();
    });
  }, []);

  const startVoiceSession = useCallback(async () => {
    if (!assistant?.getDebuggerdeployment()?.getInputaudio()) return;

    isVoiceRequestedRef.current = true;
    setRuntimeError('');
    setVoicePhase('connecting');
    let pendingConnection: Promise<void> | null = null;
    try {
      await applyPendingArguments();
      if (!isVoiceRequestedRef.current) return;
      pendingConnection = voiceAgent.setInputChannel(Channel.Audio);
      pendingVoiceConnectionRef.current = pendingConnection;
      await pendingConnection;
    } catch (error) {
      if (isVoiceRequestedRef.current) {
        isVoiceRequestedRef.current = false;
        voiceAgent.agentConfiguration.inputOptions.channel = Channel.Text;
        voiceAgent.agentConfiguration.outputOptions.channel = Channel.Text;
        setVoicePhase('idle');
        handleRuntimeError(error);
      }
    } finally {
      if (pendingVoiceConnectionRef.current === pendingConnection) {
        pendingVoiceConnectionRef.current = null;
      }
    }
  }, [applyPendingArguments, assistant, handleRuntimeError, voiceAgent]);

  const switchToText = useCallback(async () => {
    isVoiceRequestedRef.current = false;
    try {
      await voiceAgent.setInputChannel(Channel.Text);
      setVoicePhase('idle');
      setIsVoiceMuted(false);
      focusTextComposer();
    } catch (error) {
      handleRuntimeError(error);
    }
  }, [focusTextComposer, handleRuntimeError, voiceAgent]);

  const endVoiceSession = useCallback(
    async (shouldFocusText = false) => {
      isVoiceRequestedRef.current = false;
      voiceAgent.agentConfiguration.inputOptions.channel = Channel.Text;
      voiceAgent.agentConfiguration.outputOptions.channel = Channel.Text;
      setVoicePhase('idle');
      setIsVoiceMuted(false);
      setIsEndConfirmationOpen(false);
      try {
        await resetRuntime();
      } catch (error) {
        handleRuntimeError(error);
      }
      if (shouldFocusText) focusTextComposer();
    },
    [focusTextComposer, handleRuntimeError, resetRuntime, voiceAgent],
  );

  const toggleMute = useCallback(() => {
    setIsVoiceMuted(voiceAgent.toggleMute());
  }, [voiceAgent]);

  const assistantName = assistant?.getName() || 'Voice assistant';
  const isVoiceAvailable = Boolean(
    assistant?.getDebuggerdeployment()?.getInputaudio(),
  );

  const updatePromptVariable = (
    index: number,
    variable: PromptVariable,
    value: string,
  ) => {
    setPromptVariables(current =>
      current.map((item, itemIndex) =>
        itemIndex === index ? { ...item, value } : item,
      ),
    );
    voiceAgent.agentConfiguration.addArgument(variable.name, value);
    argumentsRequireReconnectRef.current = true;
    if (voicePhase === 'idle') return;

    isVoiceRequestedRef.current = false;
    voiceAgent.agentConfiguration.inputOptions.channel = Channel.Text;
    voiceAgent.agentConfiguration.outputOptions.channel = Channel.Text;
    setVoicePhase('idle');
    setIsVoiceMuted(false);
    void resetRuntime()
      .then(() => {
        argumentsRequireReconnectRef.current = false;
      })
      .catch(handleRuntimeError);
  };

  const chatSlots = useMemo(
    () => ({
      [WriteableElementName.CUSTOM_HEADER]: (
        <div
          role="toolbar"
          aria-label="Conversation toolbar"
          className="flex h-12 w-full items-center border-b border-[var(--cds-border-subtle-01)] bg-[var(--cds-layer-01)]"
        >
          {!embedded ? (
            <IconOnlyButton
              kind="ghost"
              size="md"
              renderIcon={ArrowLeft}
              iconDescription="Back to assistant"
              className="!h-12 !w-12 shrink-0"
              onClick={() =>
                navigate(`/deployment/assistant/${assistantId}/overview`)
              }
            />
          ) : null}
          <h1 className="min-w-0 flex-1 truncate px-1 text-sm font-semibold text-[var(--cds-text-primary)]">
            {assistantName}
          </h1>
          <CarbonIconButton
            label="Configuration"
            kind="ghost"
            size="md"
            aria-controls="debugger-details-panel"
            aria-expanded={isConfigurationOpen}
            aria-pressed={isConfigurationOpen}
            className={cn(
              '!h-12 !w-12 shrink-0 !border-b-2 !border-b-transparent',
              isConfigurationOpen &&
                '!border-b-[var(--cds-border-interactive)] !bg-[var(--cds-layer-selected-01)]',
            )}
            onClick={toggleConfiguration}
          >
            <Model size={16} />
          </CarbonIconButton>
        </div>
      ),
      [WriteableElementName.BEFORE_INPUT_ELEMENT]:
        voicePhase === 'idle' ? null : (
          <div
            data-testid="voice-session-dock"
            className="bg-[var(--cds-chat-prompt-background,var(--cds-background))] px-2 py-2"
          >
            {voicePhase === 'connecting' ? (
              <div
                data-testid="voice-connecting-row"
                className="mx-auto flex h-10 w-full max-w-[45rem] flex-nowrap items-center justify-between gap-2"
              >
                <InlineLoading
                  className="min-w-0 flex-none whitespace-nowrap"
                  style={{ inlineSize: 'max-content', whiteSpace: 'nowrap' }}
                  status="active"
                  iconDescription="Connecting voice"
                  description="Connecting voice..."
                />
                <IconOnlyButton
                  kind="ghost"
                  size="sm"
                  renderIcon={Close}
                  iconDescription="Cancel voice connection"
                  tooltipPosition="left"
                  onClick={() => void endVoiceSession()}
                />
              </div>
            ) : (
              <div className="mx-auto flex w-full max-w-[45rem] flex-col gap-2 sm:min-h-10 sm:flex-row sm:items-center">
                <div
                  role="status"
                  aria-live="polite"
                  className="flex min-h-10 min-w-40 flex-1 items-center gap-2 px-2 text-sm text-[var(--cds-text-primary)]"
                >
                  {isVoiceMuted ? (
                    <MicrophoneOff size={16} />
                  ) : (
                    <VoiceMode size={16} />
                  )}
                  <span>{isVoiceMuted ? 'Microphone muted' : 'Listening'}</span>
                </div>
                <div className="flex min-w-0 flex-col gap-1 sm:flex-row sm:items-end">
                  <VoiceMicrophoneSelector
                    voiceAgent={voiceAgent}
                    onError={handleRuntimeError}
                  />
                  <div
                    role="toolbar"
                    aria-label="Voice conversation controls"
                    className="flex items-center justify-end gap-1"
                  >
                    <IconOnlyButton
                      kind="ghost"
                      size="sm"
                      renderIcon={isVoiceMuted ? MicrophoneOff : Microphone}
                      iconDescription={
                        isVoiceMuted ? 'Unmute microphone' : 'Mute microphone'
                      }
                      tooltipPosition="top"
                      onClick={toggleMute}
                    />
                    <IconOnlyButton
                      kind="ghost"
                      size="sm"
                      renderIcon={Chat}
                      iconDescription="Switch to text"
                      tooltipPosition="top"
                      onClick={() => void switchToText()}
                    />
                    <IconOnlyButton
                      kind="ghost"
                      size="sm"
                      renderIcon={StopActionIcon}
                      iconDescription="End voice conversation"
                      tooltipPosition="top"
                      className="!text-[var(--cds-support-error)]"
                      onClick={() => setIsEndConfirmationOpen(true)}
                    />
                  </div>
                </div>
              </div>
            )}
          </div>
        ),
      [WriteableElementName.PROMPT_LINE_SEND_BUTTON_START]:
        voicePhase === 'idle' ? (
          <div
            role="group"
            aria-label="Chat input controls"
            className="flex h-8 items-center gap-2"
          >
            {isVoiceAvailable ? (
              <IconOnlyButton
                kind="ghost"
                size="sm"
                renderIcon={VoiceActionIcon}
                iconDescription="Start voice conversation"
                tooltipPosition="top"
                onClick={() => void startVoiceSession()}
              />
            ) : (
              <Toggletip align="top-end" autoAlign>
                <ToggletipButton
                  label="Voice is not configured"
                  className="!flex !h-8 !w-8 items-center justify-center hover:bg-[var(--cds-background-hover)]"
                >
                  <VoiceActionIcon size={16} />
                </ToggletipButton>
                <ToggletipContent>
                  <p>Configure speech to text to start a voice conversation.</p>
                  <ToggletipActions>
                    <PrimaryButton
                      size="sm"
                      onClick={() =>
                        navigate(
                          `/deployment/assistant/${assistantId}/deployment/debugger`,
                        )
                      }
                    >
                      Configure voice
                    </PrimaryButton>
                  </ToggletipActions>
                </ToggletipContent>
              </Toggletip>
            )}
            {isSessionConnected ? (
              <IconOnlyButton
                kind="ghost"
                size="sm"
                renderIcon={StopActionIcon}
                iconDescription="Stop chat session"
                tooltipPosition="top"
                onClick={() => void disconnectChatSession()}
              />
            ) : null}
          </div>
        ) : null,
    }),
    [
      assistantId,
      assistantName,
      disconnectChatSession,
      endVoiceSession,
      handleRuntimeError,
      isConfigurationOpen,
      isSessionConnected,
      isVoiceMuted,
      isVoiceAvailable,
      navigate,
      startVoiceSession,
      switchToText,
      toggleConfiguration,
      toggleMute,
      voiceAgent,
      voicePhase,
      embedded,
    ],
  );

  const handlePanelKeyDown = (event: ReactKeyboardEvent<HTMLElement>) => {
    if (event.key === 'Escape') {
      event.preventDefault();
      closePanel();
      return;
    }
    if (event.key !== 'Tab' || !isOverlayPanel) return;

    const focusableElements = Array.from(
      event.currentTarget.querySelectorAll<HTMLElement>(
        'button:not([disabled]), input:not([disabled]), [href], [tabindex]:not([tabindex="-1"])',
      ),
    ).filter(element => {
      if (element.closest('[hidden], [aria-hidden="true"]')) return false;

      const accordionWrapper = element.closest('.cds--accordion__wrapper');
      return (
        !accordionWrapper ||
        accordionWrapper.previousElementSibling?.getAttribute(
          'aria-expanded',
        ) === 'true'
      );
    });
    if (focusableElements.length === 0) return;

    const firstElement = focusableElements[0];
    const lastElement = focusableElements[focusableElements.length - 1];
    if (event.shiftKey && document.activeElement === firstElement) {
      event.preventDefault();
      lastElement.focus();
    } else if (!event.shiftKey && document.activeElement === lastElement) {
      event.preventDefault();
      firstElement.focus();
    }
  };

  return (
    <div
      className={cn(
        'flex min-h-0 flex-col bg-white text-sm dark:bg-gray-950',
        embedded ? 'h-full' : 'h-dvh',
      )}
    >
      {!embedded ? <PreviewAgentHeader /> : null}

      <div className="flex min-h-0 flex-1">
        <main className="relative flex min-h-0 min-w-0 flex-1 overflow-hidden bg-white dark:bg-gray-950">
          <section
            ref={conversationRef}
            aria-label="Conversation"
            className="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden bg-white dark:bg-gray-950"
          >
            {runtimeError ? (
              <InlineNotification
                kind="error"
                title="Debugger error"
                subtitle={runtimeError}
                lowContrast
                onClose={() => setRuntimeError('')}
                className="!m-0 !max-w-none shrink-0"
              />
            ) : null}
            <div ref={chatContainerRef} className="min-h-0 flex-1">
              <ChatCustomElement
                className="block h-full min-h-0 w-full"
                onAfterRender={makeTranscriptFocusable}
                openChatByDefault
                shouldTakeFocusIfOpensAutomatically={false}
                disableCustomElementMobileEnhancements
                injectCarbonTheme={
                  resolvedMode === 'dark' ? CarbonTheme.G100 : CarbonTheme.WHITE
                }
                assistantName={assistantName}
                aiEnabled
                header={{ isOn: true }}
                history={{ isOn: false }}
                homescreen={{ isOn: false }}
                input={{
                  expanded: true,
                  isVisible: voicePhase === 'idle',
                }}
                renderWriteableElements={chatSlots}
                layout={{
                  showFrame: false,
                  hasContentMaxWidth: !isSmallViewport,
                  customProperties: {
                    'messages-max-width': '45rem',
                    '$chat-shell-background': chatBackgroundColor,
                    '$chat-prompt-background': chatActionColor,
                    '$ai-aura-start-sm': chatBackgroundColor,
                    '$ai-aura-end': chatBackgroundColor,
                  },
                }}
                messaging={{
                  skipWelcome: false,
                  customLoadHistory: loadConversationHistory,
                  customSendMessage: sendRuntimeMessage,
                }}
                strings={{
                  input_placeholder: `Message ${assistantName}`,
                }}
              />
            </div>
          </section>

          {isConfigurationOpen && isOverlayPanel ? (
            <button
              type="button"
              tabIndex={-1}
              aria-label="Close details panel"
              className="absolute inset-0 z-10 bg-black/40 lg:hidden"
              onClick={closePanel}
            />
          ) : null}

          {isConfigurationOpen ? (
            <aside
              ref={panelRef}
              id="debugger-details-panel"
              role={isOverlayPanel ? 'dialog' : undefined}
              aria-modal={isOverlayPanel ? true : undefined}
              aria-labelledby="debugger-details-title"
              aria-label="Session configuration"
              className="absolute inset-y-0 right-0 z-20 flex w-full shrink-0 flex-col border-l border-[var(--cds-border-subtle-01)] bg-[var(--cds-background)] shadow-2xl sm:w-96 lg:static lg:z-auto lg:shadow-none"
              onKeyDown={handlePanelKeyDown}
            >
              <Layer level={1} className="flex min-h-0 flex-1 flex-col">
                <header className="flex h-12 shrink-0 items-center justify-between gap-4 border-b border-[var(--cds-border-subtle-01)] bg-[var(--cds-layer-01)] pl-4">
                  <h2
                    id="debugger-details-title"
                    className="min-w-0 truncate text-sm font-semibold text-[var(--cds-text-primary)]"
                  >
                    Session configuration
                  </h2>
                  <IconOnlyButton
                    kind="ghost"
                    size="md"
                    renderIcon={Close}
                    iconDescription="Close session configuration"
                    className="!h-12 !w-12"
                    onClick={closePanel}
                  />
                </header>

                <div className="min-h-0 flex-1 overflow-y-auto bg-[var(--cds-layer-01)]">
                  <section aria-label="Session settings">
                    <Accordion
                      aria-label="Session configuration sections"
                      align="start"
                      className="debugger-configuration-accordion"
                      size="md"
                    >
                      {voiceStack.map((item, index) => {
                        const summaryValues = item.configuration
                          .slice(0, 2)
                          .map(option => option.value);
                        const summary = summaryValues.join(' · ');
                        const ItemIcon = item.icon;

                        return (
                          <AccordionItem
                            key={item.id}
                            aria-label={`${item.label}, ${summaryValues.join(', ')}`}
                            open={index === 0}
                            title={
                              <span className="flex min-w-0 items-center gap-2">
                                <ItemIcon
                                  size={16}
                                  className="shrink-0 text-[var(--cds-icon-secondary)]"
                                />
                                <span className="min-w-0 flex-1 truncate text-sm font-medium text-[var(--cds-text-primary)]">
                                  {item.label}
                                </span>
                                <span className="max-w-36 shrink truncate text-xs text-[var(--cds-text-secondary)]">
                                  {summary}
                                </span>
                              </span>
                            }
                          >
                            <StructuredListWrapper
                              aria-label={`${item.label} configuration`}
                              className="!mb-0 !w-full"
                              isCondensed
                              selection={false}
                            >
                              <StructuredListBody>
                                {item.configuration.map(option => (
                                  <StructuredListRow key={option.id}>
                                    <StructuredListCell className="!w-[48%] !py-2 !pl-4 !pr-2 align-top">
                                      <span className="block text-sm text-[var(--cds-text-primary)]">
                                        {option.label}
                                      </span>
                                      <span className="block break-all font-mono text-[0.6875rem] leading-4 text-[var(--cds-text-helper)]">
                                        {option.key}
                                      </span>
                                    </StructuredListCell>
                                    <StructuredListCell className="min-w-0 !py-2 !pl-2 !pr-4 align-top">
                                      <div className="flex min-w-0 items-start justify-end gap-1 text-right">
                                        <span className="min-w-0 break-all pt-1.5 text-sm text-[var(--cds-text-primary)]">
                                          {option.value}
                                        </span>
                                        {isCopyableConfigurationKey(
                                          option.key,
                                        ) ? (
                                          <CopyButton
                                            className="shrink-0"
                                            copyDescription={`Copy ${option.label}`}
                                            copiedDescription={`${option.label} copied`}
                                          >
                                            {option.value}
                                          </CopyButton>
                                        ) : null}
                                      </div>
                                    </StructuredListCell>
                                  </StructuredListRow>
                                ))}
                              </StructuredListBody>
                            </StructuredListWrapper>
                          </AccordionItem>
                        );
                      })}
                      <AccordionItem
                        aria-label={`Arguments, ${promptVariables.length} variables`}
                        title={
                          <span className="flex min-w-0 items-center gap-2">
                            <ValueVariable
                              size={16}
                              className="shrink-0 text-[var(--cds-icon-secondary)]"
                            />
                            <span className="min-w-0 flex-1 truncate text-sm font-medium text-[var(--cds-text-primary)]">
                              Arguments
                            </span>
                            <span className="max-w-36 shrink truncate text-xs text-[var(--cds-text-secondary)]">
                              {promptVariables.length} variables
                            </span>
                          </span>
                        }
                      >
                        <div
                          data-testid="arguments-accordion-content"
                          className="px-4 pb-4 pt-2"
                        >
                          {promptVariables.length === 0 ? (
                            <div className="flex min-h-32 items-center justify-center px-4 text-center">
                              <div>
                                <p className="text-sm font-semibold text-[var(--cds-text-primary)]">
                                  No arguments defined
                                </p>
                                <p className="mt-1 text-sm text-[var(--cds-text-secondary)]">
                                  This assistant does not expose prompt
                                  variables.
                                </p>
                              </div>
                            </div>
                          ) : (
                            <FluidForm
                              aria-label="Arguments"
                              onSubmit={event => event.preventDefault()}
                            >
                              {promptVariables.map((variable, index) => {
                                const id = `prompt-variable-${variable.name}`;
                                const label = `{{${variable.name}}}`;
                                const helperText = formatArgumentType(
                                  variable.type,
                                );
                                const labelContent = (
                                  <span className="flex items-center justify-between gap-3">
                                    <span className="truncate font-mono">
                                      {label}
                                    </span>
                                    <span
                                      aria-hidden="true"
                                      className="text-xs font-normal text-[var(--cds-text-helper)]"
                                    >
                                      {helperText}
                                    </span>
                                  </span>
                                );
                                const onChange = (
                                  event: ReactChangeEvent<
                                    HTMLInputElement | HTMLTextAreaElement
                                  >,
                                ) =>
                                  updatePromptVariable(
                                    index,
                                    variable,
                                    event.target.value,
                                  );

                                if (
                                  !EDITABLE_ARGUMENT_TYPES.has(variable.type)
                                ) {
                                  return (
                                    <div
                                      key={variable.name}
                                      className="border-b border-[var(--cds-border-subtle-01)] bg-[var(--cds-field-01)] px-4 py-3 last:border-b-0"
                                    >
                                      <p className="mb-2 font-mono text-sm font-medium text-[var(--cds-text-primary)]">
                                        {label}
                                      </p>
                                      <InlineNotification
                                        kind="warning"
                                        lowContrast
                                        hideCloseButton
                                        title="Not editable in debugger"
                                        subtitle={`${helperText} arguments cannot be edited in this debugger.`}
                                        className="!m-0 !max-w-full"
                                      />
                                    </div>
                                  );
                                }

                                if (
                                  MULTILINE_ARGUMENT_TYPES.has(variable.type)
                                ) {
                                  return (
                                    <FluidTextArea
                                      key={variable.name}
                                      id={id}
                                      labelText={labelContent}
                                      rows={
                                        variable.type === InputVarType.json
                                          ? 6
                                          : 4
                                      }
                                      value={variable.value}
                                      className={cn(
                                        variable.type === InputVarType.json &&
                                          '[&_textarea]:font-mono',
                                      )}
                                      onChange={onChange}
                                    />
                                  );
                                }

                                if (variable.type === InputVarType.number) {
                                  return (
                                    <FluidNumberInput
                                      key={variable.name}
                                      id={id}
                                      label={labelContent}
                                      type="number"
                                      allowEmpty
                                      value={variable.value}
                                      onChange={(_event, { value }) =>
                                        updatePromptVariable(
                                          index,
                                          variable,
                                          String(value),
                                        )
                                      }
                                    />
                                  );
                                }

                                return (
                                  <FluidTextInput
                                    key={variable.name}
                                    id={id}
                                    labelText={labelContent}
                                    value={variable.value}
                                    onChange={onChange}
                                  />
                                );
                              })}
                            </FluidForm>
                          )}
                        </div>
                      </AccordionItem>
                    </Accordion>
                  </section>
                </div>
              </Layer>
            </aside>
          ) : null}
        </main>
      </div>

      <ConfirmDialog
        showing={isEndConfirmationOpen}
        type="warning"
        title="End voice conversation?"
        content="This will stop the current voice session and return to the text composer."
        confirmText="End conversation"
        onConfirm={() => void endVoiceSession(true)}
        onCancel={() => setIsEndConfirmationOpen(false)}
        onClose={() => setIsEndConfirmationOpen(false)}
      />
    </div>
  );
}
