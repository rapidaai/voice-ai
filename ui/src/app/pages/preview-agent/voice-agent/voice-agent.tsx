import React, { FC, memo, useEffect, useMemo, useRef, useState } from 'react';
import {
  VoiceAgent as VI,
  ConnectionConfig,
  AgentConfig,
  AgentCallback,
  Assistant,
  Variable,
  ConversationError,
} from '@rapidaai/react';
import { MessagingAction } from '@/app/pages/preview-agent/voice-agent/actions';
import { ConversationMessages } from '@/app/pages/preview-agent/voice-agent/text/conversations';
import { cn } from '@/utils';
import { Panel, PanelGroup, PanelResizeHandle } from 'react-resizable-panels';
import { InputVarType } from '@/models/common';
import {
  Notification,
  LinkNotification,
} from '@/app/components/carbon/notification';
import { GhostButton, IconOnlyButton } from '@/app/components/carbon/button';
import { EmptyState } from '@/app/components/carbon/empty-state';
import { Activity, Copy, FilterRemove } from '@carbon/icons-react';
import { DismissibleTag, Tag } from '@carbon/react';
import { Tabs } from '@/app/components/carbon/tabs';
import { Text } from '@/app/components/carbon/text';
import { ArrowLeft } from '@carbon/icons-react';
import { TextArea } from '@/app/components/carbon/form';
import { PreviewAgentHeader } from './preview-agent-header';

export { PreviewAgentHeader } from './preview-agent-header';

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

type EventEntry = {
  type:
    | 'directive'
    | 'configuration'
    | 'userMessage'
    | 'assistantMessage'
    | 'interrupt'
    | 'pipelineEvent'
    | 'tool_call'
    | 'metric';
  ts: Date;
  payload: any;
};

type MsgTab = 'messages' | 'events';

const EDITABLE_ARGUMENT_TYPES = [
  InputVarType.stringInput,
  InputVarType.textInput,
  InputVarType.paragraph,
  InputVarType.number,
  InputVarType.json,
  InputVarType.url,
];

/** Returns the display label for an event — matches the 2nd column in the events table. */
function getEventLabel(entry: EventEntry): string {
  if (entry.type === 'pipelineEvent') return entry.payload?.name ?? 'pipeline';
  if (entry.type === 'userMessage') return 'user';
  if (entry.type === 'assistantMessage') return 'assistant';
  if (entry.type === 'configuration') return 'session';
  if (entry.type === 'interrupt') return 'interrupt';
  if (entry.type === 'metric') return 'metric';
  return entry.type;
}

// ---------------------------------------------------------------------------
// Conversation event row
// ---------------------------------------------------------------------------

const EVENT_COLORS: Record<string, string> = {
  session: 'text-gray-500 dark:text-gray-400',
  stt: 'text-green-600 dark:text-green-400',
  llm: 'text-blue-600 dark:text-blue-400',
  tts: 'text-violet-600 dark:text-violet-400',
  vad: 'text-yellow-600 dark:text-yellow-400',
  eos: 'text-cyan-600 dark:text-cyan-400',
  denoise: 'text-orange-600 dark:text-orange-400',
  audio: 'text-slate-600 dark:text-slate-400',
  tool: 'text-pink-600 dark:text-pink-400',
  behavior: 'text-rose-600 dark:text-rose-400',
  knowledge: 'text-teal-600 dark:text-teal-400',
};

const ConversationEventRow: FC<{ entry: EventEntry }> = ({ entry }) => {
  const [expanded, setExpanded] = useState(false);
  const ts = entry.ts.toISOString().slice(11, 23);
  const toggle = () => setExpanded(p => !p);

  if (entry.type === 'pipelineEvent') {
    const { name, dataMap, id, time } = entry.payload as {
      name: string;
      dataMap: Array<[string, string]>;
      id?: string;
      time?: unknown;
    };
    const data = Object.fromEntries(dataMap ?? []);
    const color = EVENT_COLORS[name] ?? 'text-gray-500 dark:text-gray-400';
    const jsonPayload = { id, time, ...data };

    return (
      <>
        <tr
          className="hover:bg-gray-100 dark:hover:bg-gray-800/60 cursor-pointer select-text"
          onClick={toggle}
        >
          <td className="pl-3 pr-2 py-[3px] whitespace-nowrap tabular-nums text-gray-400 dark:text-gray-500">
            {ts}
          </td>
          <td
            className={cn(
              'px-2 py-[3px] whitespace-nowrap font-semibold',
              color,
            )}
          >
            {name}
          </td>
          <td
            colSpan={2}
            className="px-2 pr-3 py-[3px] text-gray-600 dark:text-gray-300 max-w-0 overflow-hidden truncate"
          >
            {JSON.stringify(jsonPayload)}
          </td>
        </tr>
        {expanded && (
          <tr className="bg-gray-50 dark:bg-gray-800/40">
            <td />
            <td colSpan={3} className="pl-2 pr-3 pt-1 pb-2">
              <pre className="whitespace-pre-wrap break-all text-gray-700 dark:text-gray-200 text-sm/6">
                {JSON.stringify(jsonPayload, null, 2)}
              </pre>
            </td>
          </tr>
        )}
      </>
    );
  }

  // Non-pipeline events — time | role | json
  const label =
    entry.type === 'userMessage'
      ? 'user'
      : entry.type === 'assistantMessage'
        ? 'assistant'
        : entry.type === 'configuration'
          ? 'session'
          : entry.type === 'interrupt'
            ? 'interrupt'
            : entry.type === 'metric'
              ? 'metric'
              : entry.type;

  const labelColor =
    entry.type === 'userMessage'
      ? 'text-emerald-600 dark:text-emerald-400'
      : entry.type === 'assistantMessage'
        ? 'text-indigo-600 dark:text-indigo-400'
        : entry.type === 'interrupt'
          ? 'text-orange-600 dark:text-orange-400'
          : entry.type === 'configuration'
            ? 'text-sky-600 dark:text-sky-400'
            : entry.type === 'metric'
              ? 'text-lime-600 dark:text-lime-400'
              : 'text-red-600 dark:text-red-400';

  return (
    <>
      <tr
        className="hover:bg-gray-100 dark:hover:bg-gray-800/60 cursor-pointer select-text"
        onClick={toggle}
      >
        <td className="pl-3 pr-2 py-[3px] whitespace-nowrap tabular-nums text-gray-400 dark:text-gray-500">
          {ts}
        </td>
        <td
          className={cn(
            'px-2 py-[3px] whitespace-nowrap font-semibold',
            labelColor,
          )}
        >
          {label}
        </td>
        <td
          colSpan={2}
          className="px-2 pr-3 py-[3px] text-gray-600 dark:text-gray-300 max-w-0 overflow-hidden truncate"
        >
          {JSON.stringify(entry.payload)}
        </td>
      </tr>
      {expanded && (
        <tr className="bg-gray-50 dark:bg-gray-800/40">
          <td />
          <td colSpan={3} className="pl-2 pr-3 pt-1 pb-2">
            <pre className="whitespace-pre-wrap break-all text-gray-700 dark:text-gray-200 text-sm/6">
              {JSON.stringify(entry.payload, null, 2)}
            </pre>
          </td>
        </tr>
      )}
    </>
  );
};

// ---------------------------------------------------------------------------
// Main layout
// ---------------------------------------------------------------------------

export const VoiceAgent: FC<{
  debug: boolean;
  connectConfig: ConnectionConfig;
  agentConfig: AgentConfig;
  agentCallback?: AgentCallback;
}> = ({ debug, connectConfig, agentConfig, agentCallback }) => {
  const voiceAgentContextValue = React.useMemo(
    () => new VI(connectConfig, agentConfig, agentCallback),
    [connectConfig, agentConfig, agentCallback],
  );
  const [assistant, setAssistant] = useState<Assistant | null>(null);
  const [events, setEvents] = useState<EventEntry[]>([]);
  const [variables, setVariables] = useState<Variable[]>([]);
  const [msgTab, setMsgTab] = useState<MsgTab>('messages');
  const [eventFilters, setEventFilters] = useState<Set<string>>(new Set());
  const [conversationError, setConversationError] =
    useState<ConversationError.AsObject | null>(null);
  const [connectionStatus, setConnectionStatus] = useState<
    'idle' | 'connecting' | 'connected'
  >('idle');
  const eventsBottomRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    let isMounted = true;
    setAssistant(null);
    setVariables([]);
    setEvents([]);
    setConversationError(null);

    voiceAgentContextValue
      .getAssistant()
      .then(ex => {
        if (isMounted) {
          if (ex.getSuccess()) setAssistant(ex.getData()!);
        }
      })
      .catch(() => {
        if (isMounted) {
          setConversationError({
            message: 'Unable to load assistant configuration.',
          } as ConversationError.AsObject);
        }
      });

    return () => {
      isMounted = false;
    };
  }, [voiceAgentContextValue]);

  useEffect(() => {
    return () => {
      voiceAgentContextValue.disconnect().catch(() => {});
    };
  }, [voiceAgentContextValue]);

  // Load variables from assistant
  useEffect(() => {
    if (!assistant) return;
    const pmtVar = assistant
      .getAssistantprovidermodel()
      ?.getTemplate()
      ?.getPromptvariablesList();
    if (pmtVar) {
      pmtVar.forEach(v => {
        if (v.getDefaultvalue())
          voiceAgentContextValue.agentConfiguration.addArgument(
            v.getName(),
            v.getDefaultvalue(),
          );
      });
      setVariables(pmtVar);
    }
  }, [assistant, voiceAgentContextValue]);

  // Track connection state from agent events
  useEffect(() => {
    const handler = (state: string) => {
      if (state === 'connecting') {
        setConnectionStatus('connecting');
      } else if (state === 'connected') {
        setConnectionStatus('connected');
        setTimeout(() => setConnectionStatus('idle'), 2000);
      } else {
        setConnectionStatus('idle');
      }
    };
    voiceAgentContextValue.on('onConnectionStateEvent', handler);
    return () => {
      voiceAgentContextValue.off('onConnectionStateEvent', handler);
    };
  }, [voiceAgentContextValue]);

  useEffect(() => {
    let isActive = true;
    const pushEvent = (entry: Omit<EventEntry, 'ts'>) => {
      if (!isActive) return;
      setEvents(p => [...p, { ...entry, ts: new Date() }]);
    };

    voiceAgentContextValue.registerCallback({
      onToolCall: toolCall => {
        pushEvent({ type: 'tool_call', payload: toolCall });
      },
      onConfiguration: args =>
        pushEvent({ type: 'configuration', payload: args }),
      onUserMessage: args => pushEvent({ type: 'userMessage', payload: args }),
      onAssistantMessage: args => {
        if (args?.messageText)
          pushEvent({ type: 'assistantMessage', payload: args });
      },
      onInterrupt: args => pushEvent({ type: 'interrupt', payload: args }),
      onConversationEvent: event =>
        pushEvent({ type: 'pipelineEvent', payload: event }),
      onMetric: metric => pushEvent({ type: 'metric', payload: metric }),
      onConversationError: error => {
        if (isActive) setConversationError(error);
      },
    });

    return () => {
      isActive = false;
    };
  }, [voiceAgentContextValue]);

  // Auto-scroll events tab when new events arrive
  useEffect(() => {
    if (msgTab === 'events') {
      setTimeout(
        () => eventsBottomRef.current?.scrollIntoView({ behavior: 'smooth' }),
        50,
      );
    }
  }, [events.length, msgTab]);

  // Derive unique labels from events for the filter bar
  const availableEventLabels = useMemo(() => {
    const labels = new Set<string>();
    events.forEach(e => labels.add(getEventLabel(e)));
    return Array.from(labels);
  }, [events]);

  // Filter events — empty set means show all
  const filteredEvents = useMemo(() => {
    if (eventFilters.size === 0) return events;
    return events.filter(e => eventFilters.has(getEventLabel(e)));
  }, [events, eventFilters]);

  const toggleEventFilter = (label: string) => {
    setEventFilters(prev => {
      const next = new Set(prev);
      if (next.has(label)) {
        next.delete(label);
      } else {
        next.add(label);
      }
      return next;
    });
  };

  const voiceWarning = assistant
    ? debug
      ? !assistant.getDebuggerdeployment()?.hasInputaudio()
      : !assistant.getApideployment()?.hasInputaudio()
    : false;

  const enableVoiceHref = `/deployment/assistant/${agentConfig.id}/deployment/debugger`;

  return (
    <div className="flex h-dvh min-h-0 flex-col bg-white text-sm/6 dark:bg-gray-950">
      <PreviewAgentHeader />
      <PanelGroup
        className="!flex !min-h-0 !flex-1 !overflow-hidden"
        direction="horizontal"
      >
        {/* ── Left: messaging ─────────────────────────────────────────── */}
        <Panel
          defaultSize={70}
          minSize={55}
          className="flex h-full flex-col overflow-hidden bg-white dark:bg-gray-950"
        >
          {/* Header */}
          <div className="shrink-0">
            {debug && (
              <div className="flex items-center gap-1.5 px-3 py-2 border-b border-gray-200 dark:border-gray-800">
                <IconOnlyButton
                  kind="ghost"
                  size="sm"
                  renderIcon={ArrowLeft}
                  iconDescription="Back to Assistant"
                  onClick={() => {
                    window.location.href = `/deployment/assistant/${agentConfig.id}/overview`;
                  }}
                />
                <span className="text-sm font-medium text-gray-900 dark:text-gray-100">
                  Back to Assistant
                </span>
              </div>
            )}
            {voiceWarning && debug && (
              <LinkNotification
                kind="warning"
                title="Voice disabled"
                subtitle="Enable voice to enjoy a voice experience with your assistant."
                linkText="Enable voice"
                onLinkClick={() => window.open(enableVoiceHref, '_blank')}
                hideCloseButton
              />
            )}
            {voiceWarning && !debug && (
              <Notification
                kind="warning"
                title="Voice disabled"
                subtitle="This assistant is currently available for text conversations."
                hideCloseButton
              />
            )}
            {connectionStatus === 'connecting' && (
              <Notification
                kind="info"
                title="Establishing connection to the assistant..."
                hideCloseButton
              />
            )}
            {connectionStatus === 'connected' && (
              <Notification
                kind="success"
                title="Connected"
                subtitle="You are now connected. Start speaking."
                hideCloseButton
              />
            )}
            {conversationError && (
              <Notification
                kind="error"
                title="Error"
                subtitle={
                  conversationError.message ||
                  'An error occurred during the conversation.'
                }
                hideCloseButton={false}
                onClose={() => setConversationError(null)}
              />
            )}
            {/* Tab bar */}
            <div className="border-b border-gray-200 dark:border-gray-800">
              <Tabs
                tabs={[
                  'Messages',
                  events.length > 0 ? `Events (${events.length})` : 'Events',
                ]}
                selectedIndex={msgTab === 'messages' ? 0 : 1}
                onChange={idx => setMsgTab(idx === 0 ? 'messages' : 'events')}
                contained
                isLoading={!assistant}
                aria-label="Message tabs"
              />
            </div>
          </div>

          {/* Messages tab */}
          {msgTab === 'messages' &&
            (() => {
              const hasMessages = events.some(
                e => e.type === 'userMessage' || e.type === 'assistantMessage',
              );
              return hasMessages ? (
                <div className="flex flex-col grow min-h-0 overflow-y-auto px-4 py-4">
                  <ConversationMessages vag={voiceAgentContextValue} />
                </div>
              ) : (
                <AssistantPlaceholder assistant={assistant} />
              );
            })()}

          {/* Events tab — structured conversation event rows */}
          {msgTab === 'events' && (
            <div className="flex flex-col flex-1 min-h-0">
              {/* Filter bar */}
              {availableEventLabels.length > 0 && (
                <div className="shrink-0 flex flex-wrap items-center gap-2 px-3 py-2 border-b border-gray-200 dark:border-gray-800">
                  <span className="text-xs font-medium text-gray-500 dark:text-gray-400 select-none">
                    Filter
                  </span>
                  {availableEventLabels.map(label =>
                    eventFilters.has(label) ? (
                      <DismissibleTag
                        key={label}
                        text={label}
                        type="blue"
                        size="md"
                        onClose={() => toggleEventFilter(label)}
                      />
                    ) : (
                      <Tag
                        key={label}
                        type={eventFilters.size === 0 ? 'blue' : 'cool-gray'}
                        size="md"
                        onClick={() => toggleEventFilter(label)}
                        className="cursor-pointer"
                      >
                        {label}
                      </Tag>
                    ),
                  )}
                  {eventFilters.size > 0 && (
                    <GhostButton
                      size="sm"
                      onClick={() => setEventFilters(new Set())}
                    >
                      Clear all
                    </GhostButton>
                  )}
                </div>
              )}

              <div className="flex-1 min-h-0 overflow-y-auto py-1">
                {filteredEvents.length === 0 ? (
                  <EmptyState
                    icon={events.length === 0 ? Activity : FilterRemove}
                    title={
                      events.length === 0
                        ? 'No events yet'
                        : 'No events match the selected filters'
                    }
                    subtitle={
                      events.length === 0
                        ? 'Events will appear here once a conversation starts.'
                        : 'Try removing some filters to see more events.'
                    }
                    className="h-full"
                  />
                ) : (
                  <table className="w-full table-fixed font-mono text-sm/6 border-collapse">
                    <colgroup>
                      <col className="w-[9rem]" />
                      <col className="w-[6rem]" />
                      <col className="w-[10rem]" />
                      <col />
                    </colgroup>
                    <tbody>
                      {filteredEvents.map((entry, i) => (
                        <ConversationEventRow key={i} entry={entry} />
                      ))}
                    </tbody>
                  </table>
                )}
                <div ref={eventsBottomRef} />
              </div>
            </div>
          )}

          {/* Messaging action — always visible */}
          <MessagingAction
            assistant={assistant}
            placeholder="How can I help you?"
            className=" border-t"
            voiceAgent={voiceAgentContextValue}
          />
        </Panel>

        <PanelResizeHandle className="flex w-px! bg-gray-200 dark:bg-gray-800 hover:bg-blue-700 dark:hover:bg-blue-500 items-stretch"></PanelResizeHandle>
        {/* ── Right: assistant + metrics ──────────────────────────────── */}
        <Panel
          defaultSize={30}
          minSize={25}
          className="shrink-0 flex flex-col overflow-hidden"
        >
          <VoiceAgentDebugger
            debug={debug}
            voiceAgent={voiceAgentContextValue}
            assistant={assistant}
            variables={variables}
            events={events}
            onChangeArgument={(k, v) =>
              voiceAgentContextValue.agentConfiguration.addArgument(k, v)
            }
          />
        </Panel>
      </PanelGroup>
    </div>
  );
};

// ---------------------------------------------------------------------------
// Right panel: tabs — arguments | configuration | metrics
// ---------------------------------------------------------------------------

type RightTab = 'arguments' | 'configuration' | 'metrics';

export const VoiceAgentDebugger: FC<{
  debug: boolean;
  voiceAgent: VI;
  assistant: Assistant | null;
  variables: Variable[];
  events: EventEntry[];
  onChangeArgument: (k: string, v: string) => void;
}> = memo(({ debug, assistant, variables, events, onChangeArgument }) => {
  const RIGHT_TABS: RightTab[] = ['configuration', 'arguments', 'metrics'];
  const [tab, setTab] = useState<RightTab>('configuration');
  const metrics = useMemo(() => computeMetrics(events), [events]);
  const metricCount = Object.keys(metrics).length;

  const deployment = assistant
    ? ((debug
        ? assistant.getDebuggerdeployment()
        : assistant.getApideployment()) ?? null)
    : null;
  const stt = deployment?.getInputaudio() ?? null;
  const tts = deployment?.getOutputaudio() ?? null;
  const model = assistant?.getAssistantprovidermodel() ?? null;
  const executor = assistant
    ? assistant.hasAssistantprovideragentkit()
      ? 'agentkit'
      : assistant.hasAssistantproviderwebsocket()
        ? 'websocket'
        : 'model'
    : '';
  const inputMode = 'Text' + (stt ? ', Audio' : '');
  const outputMode = 'Text' + (tts ? ', Audio' : '');

  return (
    <div className="flex flex-col h-full overflow-hidden text-sm">
      {/* Tab bar */}
      <div className="border-b border-gray-200 dark:border-gray-800">
        <Tabs
          tabs={[
            'Configuration',
            variables.length > 0
              ? `Arguments (${variables.length})`
              : 'Arguments',
            metricCount > 0 ? `Metrics (${metricCount})` : 'Metrics',
          ]}
          selectedIndex={RIGHT_TABS.indexOf(tab)}
          onChange={idx => setTab(RIGHT_TABS[idx])}
          contained
          aria-label="Debugger tabs"
          isLoading={!assistant}
        />
      </div>

      {/* ── arguments tab ── */}
      {tab === 'arguments' && (
        <DebuggerScrollArea>
          <DebuggerTabHeader
            title="Arguments"
            subtitle={`${variables.length} prompt variables`}
          />
          <ArgumentList
            variables={variables}
            onChangeArgument={onChangeArgument}
          />
        </DebuggerScrollArea>
      )}

      {/* ── configuration tab ── */}
      {tab === 'configuration' && (
        <DebuggerScrollArea>
          <ConfigBlock
            title="deployment"
            isLoading={!assistant}
            skeletonRows={4}
          >
            {assistant && (
              <>
                <InfoRow label="executor" value={executor} />
                <InfoRow
                  label="model"
                  value={model?.getModelprovidername() || 'not configured'}
                />
                <InfoRow label="input mode" value={inputMode} />
                <InfoRow label="output mode" value={outputMode} />
              </>
            )}
          </ConfigBlock>
          <ConfigBlock
            title="assistant"
            isLoading={!assistant}
            skeletonRows={3}
          >
            {assistant && (
              <>
                <InfoRow label="name" value={assistant.getName()} />
                <InfoRow label="arguments" value={String(variables.length)} />
                {assistant.getDescription() && (
                  <InfoRow
                    label="description"
                    value={assistant.getDescription()}
                  />
                )}
              </>
            )}
          </ConfigBlock>

          <ConfigBlock title="stt" isLoading={!assistant} skeletonRows={2}>
            {stt ? (
              <>
                <InfoRow label="provider" value={stt.getAudioprovider()} />
                {stt.getAudiooptionsList().map(m => (
                  <InfoRow
                    key={m.getKey()}
                    label={m.getKey()}
                    value={m.getValue()}
                  />
                ))}
              </>
            ) : (
              <ConfigEmpty label="Audio input is not configured." />
            )}
          </ConfigBlock>

          <ConfigBlock title="tts" isLoading={!assistant} skeletonRows={2}>
            {tts ? (
              <>
                <InfoRow label="provider" value={tts.getAudioprovider()} />
                {tts.getAudiooptionsList().map(m => (
                  <InfoRow
                    key={m.getKey()}
                    label={m.getKey()}
                    value={m.getValue()}
                  />
                ))}
              </>
            ) : (
              <ConfigEmpty label="Audio output is not configured." />
            )}
          </ConfigBlock>

          <ConfigBlock title="llm" isLoading={!assistant} skeletonRows={2}>
            {model ? (
              <>
                <InfoRow
                  label="provider"
                  value={model.getModelprovidername()}
                />
                {model.getAssistantmodeloptionsList().map(m => (
                  <InfoRow
                    key={m.getKey()}
                    label={m.getKey()}
                    value={m.getValue()}
                  />
                ))}
              </>
            ) : (
              <ConfigEmpty label="Model configuration is not available." />
            )}
          </ConfigBlock>
        </DebuggerScrollArea>
      )}

      {/* ── metrics tab ── */}
      {tab === 'metrics' && (
        <DebuggerScrollArea>
          <DebuggerTabHeader
            title="Metrics"
            subtitle={`${metricCount} live signals`}
          />
          <MetricGrid metrics={metrics} />
        </DebuggerScrollArea>
      )}
    </div>
  );
});

// ---------------------------------------------------------------------------
// Sub-components
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Empty-state placeholder — developer console style
// ---------------------------------------------------------------------------

const AssistantPlaceholder: FC<{
  assistant: Assistant | null;
}> = ({ assistant }) => {
  const isLoading = !assistant;
  return (
    <div className="flex flex-col flex-1 min-h-0 items-start justify-end gap-1 px-2 pb-6 select-none">
      <Text
        as="span"
        isLoading={isLoading}
        heading
        skeletonWidth="200px"
        className="text-2xl font-semibold text-gray-800 dark:text-gray-100 italic"
      >
        Hello,
      </Text>
      <Text
        as="span"
        isLoading={isLoading}
        skeletonWidth="280px"
        className="text-lg text-gray-400 dark:text-gray-500 font-semibold italic"
      >
        How can I help you today?
      </Text>
    </div>
  );
};

const DebuggerScrollArea: FC<{ children: React.ReactNode }> = ({
  children,
}) => (
  <div className="flex-1 min-h-0 overflow-y-auto bg-white dark:bg-gray-950">
    {children}
  </div>
);

export const DebuggerTabHeader: FC<{
  title: string;
  subtitle?: string;
}> = ({ title, subtitle }) => (
  <div className="border-b border-gray-200/90 px-4 py-3 dark:border-gray-800">
    <div className="min-w-0">
      <h3 className="truncate text-sm font-semibold text-gray-900 dark:text-gray-100">
        {title}
      </h3>
      {subtitle && (
        <p className="mt-0.5 truncate text-xs text-gray-500 dark:text-gray-400">
          {subtitle}
        </p>
      )}
    </div>
  </div>
);

export const ArgumentList: FC<{
  variables: Variable[];
  onChangeArgument: (k: string, v: string) => void;
}> = ({ variables, onChangeArgument }) => {
  if (variables.length === 0) {
    return (
      <EmptyState
        title="No arguments defined"
        subtitle="This assistant does not expose prompt variables."
        className="min-h-[18rem]"
      />
    );
  }

  return (
    <div className="divide-y divide-gray-200/90 border-b border-gray-200/90 dark:divide-gray-800 dark:border-gray-800">
      {variables.map(variable => (
        <ArgumentRow
          key={variable.getName()}
          variable={variable}
          onChangeArgument={onChangeArgument}
        />
      ))}
    </div>
  );
};

const ArgumentRow: FC<{
  variable: Variable;
  onChangeArgument: (k: string, v: string) => void;
}> = ({ variable, onChangeArgument }) => {
  const type = variable.getType() as InputVarType;
  const editable = EDITABLE_ARGUMENT_TYPES.includes(type);
  const rows =
    type === InputVarType.paragraph || type === InputVarType.json ? 4 : 2;

  return (
    <div className="px-4 py-3">
      <div className="mb-2 flex items-start justify-between gap-3">
        <div className="min-w-0">
          <div
            className="truncate font-mono text-sm font-semibold text-gray-900 dark:text-gray-100"
            title={variable.getName()}
          >
            {`{{${variable.getName()}}}`}
          </div>
          {variable.getDefaultvalue() && (
            <div className="mt-0.5 truncate text-xs text-gray-500 dark:text-gray-400">
              Default value loaded
            </div>
          )}
        </div>
        <ArgumentTypePill type={variable.getType()} />
      </div>

      {editable ? (
        <TextArea
          id={variable.getName()}
          labelText={`{{${variable.getName()}}}`}
          hideLabel
          rows={rows}
          defaultValue={variable.getDefaultvalue()}
          placeholder="Enter variable value"
          onChange={(e: React.ChangeEvent<HTMLTextAreaElement>) =>
            onChangeArgument(variable.getName(), e.target.value)
          }
        />
      ) : (
        <ConfigEmpty label="This argument type is not editable in preview." />
      )}
    </div>
  );
};

const ArgumentTypePill: FC<{ type: string }> = ({ type }) => (
  <span className="shrink-0 border border-gray-200 bg-gray-50 px-2 py-1 text-xs font-medium lowercase text-gray-600 dark:border-gray-800 dark:bg-gray-900 dark:text-gray-300">
    {type || 'unknown'}
  </span>
);

export const ConfigBlock: FC<{
  title: string;
  children: React.ReactNode;
  isLoading?: boolean;
  skeletonRows?: number;
}> = ({ title, children, isLoading = false, skeletonRows = 3 }) => (
  <section>
    <div className="border-y border-gray-200/90 px-4 py-2 text-xs font-semibold uppercase tracking-widest text-gray-500 dark:border-gray-800 dark:text-gray-400">
      {title}
    </div>
    {isLoading ? (
      <ConfigRowsSkeleton rowCount={skeletonRows} />
    ) : (
      <div className="px-4 pb-1">{children}</div>
    )}
  </section>
);

export const InfoRow: FC<{ label: string; value: string }> = ({
  label,
  value,
}) => {
  const copyValue = () => {
    navigator.clipboard?.writeText(value).catch(() => {});
  };

  return (
    <div className="grid grid-cols-[8rem_minmax(0,1fr)_2rem] items-center gap-x-3 border-t border-gray-100/80 py-2.5 first:border-t-0 dark:border-gray-900/80">
      <span
        className="truncate text-sm lowercase tracking-wide text-gray-500 dark:text-gray-400"
        title={label}
      >
        {label}
      </span>
      <span
        className="truncate text-right text-sm font-medium text-gray-900 dark:text-gray-100"
        title={value}
      >
        {value}
      </span>
      <IconOnlyButton
        kind="ghost"
        size="sm"
        renderIcon={Copy}
        iconDescription={`Copy ${label}`}
        tooltipPosition="left"
        onClick={copyValue}
        className="justify-self-end"
      />
    </div>
  );
};

export const ConfigEmpty: FC<{ label: string }> = ({ label }) => (
  <div className="border-t border-gray-100/80 dark:border-gray-900/80 py-2.5 first:border-t-0">
    <span className="text-sm text-gray-400 dark:text-gray-500">{label}</span>
  </div>
);

const ConfigRowsSkeleton: FC<{ rowCount: number }> = ({ rowCount }) => (
  <div className="px-4 pb-2 animate-pulse">
    {Array.from({ length: rowCount }).map((_, idx) => (
      <div
        key={idx}
        className="grid grid-cols-[12rem_minmax(0,1fr)] gap-x-4 border-t border-gray-100/80 dark:border-gray-900/80 py-2.5 first:border-t-0"
      >
        <div className="h-4 w-24 rounded bg-gray-200 dark:bg-gray-800" />
        <div className="ml-auto h-4 w-36 rounded bg-gray-200 dark:bg-gray-800" />
      </div>
    ))}
  </div>
);

const MetricGrid: FC<{
  metrics: Record<string, string | number>;
}> = ({ metrics }) => {
  const entries = Object.entries(metrics);
  const primaryKeys = ['messages_sent', 'messages_received', 'pipeline_events'];
  const primary = primaryKeys
    .map(key => entries.find(([metric]) => metric === key))
    .filter(Boolean) as Array<[string, string | number]>;
  const rest = entries.filter(([key]) => !primaryKeys.includes(key));

  if (entries.length === 0) {
    return (
      <EmptyState
        title="No metrics yet"
        subtitle="Metrics appear once a conversation starts."
        className="min-h-[18rem]"
      />
    );
  }

  return (
    <div className="space-y-4 p-4">
      <MetricSection title="conversation" entries={primary} emphasized />
      {rest.length > 0 && <MetricSection title="signals" entries={rest} />}
    </div>
  );
};

const MetricSection: FC<{
  title: string;
  entries: Array<[string, string | number]>;
  emphasized?: boolean;
}> = ({ title, entries, emphasized = false }) => {
  if (entries.length === 0) return null;

  return (
    <section>
      <div className="mb-2 text-xs font-semibold uppercase tracking-widest text-gray-500 dark:text-gray-400">
        {title}
      </div>
      <div
        className={cn('grid gap-2', emphasized ? 'grid-cols-3' : 'grid-cols-2')}
      >
        {entries.map(([k, v]) => (
          <MetricCard key={k} label={k} value={String(v)} />
        ))}
      </div>
    </section>
  );
};

export const MetricCard: FC<{ label: string; value: string }> = ({
  label,
  value,
}) => (
  <div className="min-w-0 border border-gray-200 bg-gray-50 px-3 py-2.5 dark:border-gray-800 dark:bg-gray-900">
    <span className="block truncate text-xs uppercase tracking-wide text-gray-400 dark:text-gray-500">
      {label.replace(/_/g, ' ')}
    </span>
    <span
      className="mt-1 block truncate text-sm font-semibold tabular-nums text-gray-900 dark:text-gray-100"
      title={value}
    >
      {value}
    </span>
  </div>
);

// ---------------------------------------------------------------------------
// Metrics computation
// ---------------------------------------------------------------------------

function computeMetrics(events: EventEntry[]): Record<string, string | number> {
  const m: Record<string, string | number> = {
    messages_sent: events.filter(
      e => e.type === 'userMessage' && e.payload?.completed,
    ).length,
    messages_received: events.filter(e => e.type === 'assistantMessage').length,
    pipeline_events: events.filter(e => e.type === 'pipelineEvent').length,
  };

  // Walk in reverse to get the latest value for each key.
  for (let i = events.length - 1; i >= 0; i--) {
    const e = events[i];

    // Server-emitted ConversationMetric packets (stt.latency_ms, agent.ttft_ms, etc.)
    if (e.type === 'metric') {
      const list: Array<{ name: string; value: string }> =
        e.payload?.metricsList ?? [];
      for (const { name, value } of list) {
        if (name && !(name in m)) m[name] = value;
      }
      continue;
    }

    // Pipeline events — extract well-known fields
    if (e.type !== 'pipelineEvent') continue;
    const { name, dataMap } = e.payload as {
      name: string;
      dataMap: Array<[string, string]>;
    };
    const data = Object.fromEntries(dataMap ?? []);
    const type = data['type'];

    if (
      name === 'llm' &&
      type === 'provider_metrics' &&
      !('llm_input_tokens' in m)
    ) {
      if (data['input_tokens']) m['llm_input_tokens'] = data['input_tokens'];
      if (data['output_tokens']) m['llm_output_tokens'] = data['output_tokens'];
    }
    if (name === 'stt' && type === 'completed' && !('stt_words' in m)) {
      if (data['word_count']) m['stt_words'] = data['word_count'];
    }
    if (name === 'tts' && type === 'completed' && !('tts_audio_kb' in m)) {
      if (data['audio_bytes'])
        m['tts_audio_kb'] =
          `${Math.round(Number(data['audio_bytes']) / 1024)} KB`;
    }
  }

  return m;
}
