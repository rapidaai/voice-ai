import {
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
  Accordion,
  AccordionItem,
  Dropdown,
  FluidForm,
  FluidTextInput,
  IconButton as CarbonIconButton,
  InlineLoading,
  Layer,
  StructuredListBody,
  StructuredListCell,
  StructuredListRow,
  StructuredListWrapper,
} from '@carbon/react';
import { IconOnlyButton } from '@/app/components/ui/primitives';
import { ConfirmDialog } from '@/app/components/dialogs/shared';
import { useTheme } from '@/theme/theme-provider';
import { cn } from '@/utils';
import { PreviewAgentHeader } from './preview-agent-header';

const conversationHistory: HistoryItem[] = [
  {
    time: '2026-10-18T10:42:08.000Z',
    message: {
      id: 'delivery-request-1',
      input: {
        message_type: MessageInputType.TEXT,
        text: 'I need to move my delivery to Friday afternoon. Can you help?',
      },
    },
  },
  {
    time: '2026-10-18T10:42:10.000Z',
    message: {
      id: 'delivery-response-1',
      request_id: 'delivery-request-1',
      output: {
        generic: [
          {
            response_type: MessageResponseTypes.TEXT,
            text: 'I found your next delivery scheduled for Wednesday. I can move it to Friday between 1:00 PM and 5:00 PM. Would you like me to confirm that change?',
          },
        ],
      },
    },
  },
  {
    time: '2026-10-18T10:42:16.000Z',
    message: {
      id: 'delivery-request-2',
      input: {
        message_type: MessageInputType.TEXT,
        text: 'Yes, please confirm it.',
      },
    },
  },
  {
    time: '2026-10-18T10:42:17.000Z',
    message: {
      id: 'delivery-response-2',
      request_id: 'delivery-request-2',
      output: {
        generic: [
          {
            response_type: MessageResponseTypes.TEXT,
            text: 'Done. Your delivery is now scheduled for Friday between 1:00 PM and 5:00 PM.',
          },
        ],
      },
    },
  },
];

const voiceStack = [
  {
    id: 'language-model',
    icon: Model,
    label: 'Language model',
    configuration: [
      { id: 'provider', label: 'Provider', key: 'provider', value: 'OpenAI' },
      {
        id: 'model',
        label: 'Model',
        key: 'model.name',
        value: 'gpt-4.1-mini',
      },
      {
        id: 'temperature',
        label: 'Temperature',
        key: 'model.temperature',
        value: '0.7',
      },
      { id: 'top-p', label: 'Top P', key: 'model.top_p', value: '1' },
      {
        id: 'max-completion-tokens',
        label: 'Max completion tokens',
        key: 'model.max_completion_tokens',
        value: '1048',
      },
    ],
  },
  {
    id: 'speech-to-text',
    icon: VoiceActivate,
    label: 'Speech to text',
    configuration: [
      {
        id: 'provider',
        label: 'Provider',
        key: 'provider',
        value: 'Deepgram',
      },
      { id: 'model', label: 'Model', key: 'listen.model', value: 'nova-3' },
    ],
  },
  {
    id: 'text-to-speech',
    icon: VolumeUp,
    label: 'Text to speech',
    configuration: [
      {
        id: 'provider',
        label: 'Provider',
        key: 'provider',
        value: 'Cartesia',
      },
      { id: 'model', label: 'Model', key: 'speak.model', value: 'sonic-2' },
      {
        id: 'voice',
        label: 'Voice',
        key: 'speak.voice.id',
        value: '6ccbfb76-1fc6-48f7-b71d-91ac6298247b',
        displayValue: 'Tessa - Kind Companion',
      },
      {
        id: 'language',
        label: 'Language',
        key: 'speak.language',
        value: 'en',
      },
    ],
  },
];

type VoicePhase = 'idle' | 'connecting' | 'listening';

const microphoneOptions = [
  { id: 'built-in', label: 'Built-in Microphone' },
  { id: 'usb', label: 'USB Microphone' },
];

export function PreviewVoiceAgentV2() {
  const { assistantId } = useParams();
  const navigate = useNavigate();
  const { resolvedMode } = useTheme();
  const chatContainerRef = useRef<HTMLDivElement>(null);
  const conversationRef = useRef<HTMLElement>(null);
  const panelRef = useRef<HTMLElement>(null);
  const panelTriggerRef = useRef<HTMLButtonElement | null>(null);
  const responseSequence = useRef(0);
  const [isConfigurationOpen, setIsConfigurationOpen] = useState(false);
  const [voicePhase, setVoicePhase] = useState<VoicePhase>('idle');
  const [isVoiceMuted, setIsVoiceMuted] = useState(false);
  const [isEndConfirmationOpen, setIsEndConfirmationOpen] = useState(false);
  const [selectedMicrophone, setSelectedMicrophone] = useState(
    microphoneOptions[0],
  );
  const [isOverlayPanel, setIsOverlayPanel] = useState(
    () =>
      typeof window !== 'undefined' &&
      window.matchMedia('(max-width: 1055px)').matches,
  );
  const [promptVariables, setPromptVariables] = useState([
    { name: 'customer_name', value: 'Prashant' },
    { name: 'account_tier', value: 'Enterprise' },
  ]);

  const loadConversationHistory = useCallback(
    async () => conversationHistory,
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

  useEffect(() => {
    const mediaQuery = window.matchMedia('(max-width: 1055px)');
    const updatePanelMode = () => setIsOverlayPanel(mediaQuery.matches);
    updatePanelMode();
    mediaQuery.addEventListener('change', updatePanelMode);

    return () => mediaQuery.removeEventListener('change', updatePanelMode);
  }, []);

  useEffect(() => {
    if (voicePhase !== 'connecting') return;

    const connectionTimer = window.setTimeout(
      () => setVoicePhase('listening'),
      600,
    );
    return () => window.clearTimeout(connectionTimer);
  }, [voicePhase]);

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

  const sendEvaluationMessage = useCallback(
    async (
      request: MessageRequest,
      options: CustomSendMessageOptions,
      instance: ChatInstance,
    ) => {
      if (options.signal.aborted) return;

      const message = request.input.text?.trim();
      if (!message) return;

      const customerName =
        promptVariables.find(variable => variable.name === 'customer_name')
          ?.value || 'there';
      responseSequence.current += 1;

      await instance.messaging.addMessage({
        id: `evaluation-response-${responseSequence.current}`,
        request_id: request.id,
        output: {
          generic: [
            {
              response_type: MessageResponseTypes.TEXT,
              text: `Thanks, ${customerName}. This evaluation received your message: “${message}”`,
            },
          ],
        },
      });
    },
    [promptVariables],
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

  const resetVoiceSession = useCallback((shouldFocusText = false) => {
    setVoicePhase('idle');
    setIsVoiceMuted(false);
    setIsEndConfirmationOpen(false);
    if (!shouldFocusText) return;

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

  const chatSlots = useMemo(
    () => ({
      [WriteableElementName.CUSTOM_HEADER]: (
        <div
          role="toolbar"
          aria-label="Conversation toolbar"
          className="flex h-12 w-full items-center border-b border-[var(--cds-border-subtle-01)] bg-[var(--cds-layer-01)]"
        >
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
          <h1 className="min-w-0 flex-1 truncate px-1 text-sm font-semibold text-[var(--cds-text-primary)]">
            Delivery concierge
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
                  onClick={() => resetVoiceSession()}
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
                      if (selectedItem) setSelectedMicrophone(selectedItem);
                    }}
                    className="w-full sm:w-48"
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
                      onClick={() => setIsVoiceMuted(current => !current)}
                    />
                    <IconOnlyButton
                      kind="ghost"
                      size="sm"
                      renderIcon={Chat}
                      iconDescription="Switch to text"
                      tooltipPosition="top"
                      onClick={() => resetVoiceSession(true)}
                    />
                    <IconOnlyButton
                      kind="ghost"
                      size="sm"
                      renderIcon={StopFilledAlt}
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
          <IconOnlyButton
            kind="ghost"
            size="sm"
            renderIcon={VoiceMode}
            iconDescription="Start voice conversation"
            tooltipPosition="top"
            onClick={() => setVoicePhase('connecting')}
          />
        ) : null,
    }),
    [
      assistantId,
      isConfigurationOpen,
      isVoiceMuted,
      navigate,
      resetVoiceSession,
      selectedMicrophone,
      toggleConfiguration,
      voicePhase,
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

  if (!assistantId) {
    return <Navigate to="/404" replace />;
  }

  return (
    <div className="flex h-dvh min-h-0 flex-col bg-white text-sm dark:bg-gray-950">
      <PreviewAgentHeader />

      <div className="flex min-h-0 flex-1">
        <main className="relative flex min-h-0 min-w-0 flex-1 overflow-hidden bg-white dark:bg-gray-950">
          <section
            ref={conversationRef}
            aria-label="Conversation"
            className="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden bg-white dark:bg-gray-950"
          >
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
                assistantName="Delivery concierge"
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
                  hasContentMaxWidth: true,
                  customProperties: {
                    'messages-max-width': '45rem',
                  },
                }}
                messaging={{
                  skipWelcome: true,
                  customLoadHistory: loadConversationHistory,
                  customSendMessage: sendEvaluationMessage,
                }}
                strings={{
                  input_placeholder: 'Message Delivery concierge',
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
                                    <StructuredListCell className="min-w-0 !py-2 !pl-2 !pr-4 text-right align-top">
                                      <span className="block break-all text-sm text-[var(--cds-text-primary)]">
                                        {'displayValue' in option
                                          ? option.displayValue
                                          : option.value}
                                      </span>
                                      {'displayValue' in option && (
                                        <span className="block break-all font-mono text-[0.6875rem] leading-4 text-[var(--cds-text-helper)]">
                                          {option.value}
                                        </span>
                                      )}
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
                          <FluidForm
                            aria-label="Arguments"
                            onSubmit={event => event.preventDefault()}
                          >
                            {promptVariables.map((variable, index) => (
                              <FluidTextInput
                                key={variable.name}
                                id={`prompt-variable-${variable.name}`}
                                labelText={`{{${variable.name}}}`}
                                value={variable.value}
                                onChange={event => {
                                  setPromptVariables(current =>
                                    current.map((item, itemIndex) =>
                                      itemIndex === index
                                        ? { ...item, value: event.target.value }
                                        : item,
                                    ),
                                  );
                                }}
                              />
                            ))}
                          </FluidForm>
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
        onConfirm={() => resetVoiceSession(true)}
        onCancel={() => setIsEndConfirmationOpen(false)}
        onClose={() => setIsEndConfirmationOpen(false)}
      />
    </div>
  );
}
