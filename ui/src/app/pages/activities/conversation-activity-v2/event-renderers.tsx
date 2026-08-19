import type { FC } from 'react';
import { Tag } from '@carbon/react';
import type { TimelineDocument } from './types';
import { getDocumentComponent } from './utils';

type EventChip = {
  label: string;
  type: 'blue' | 'cool-gray' | 'gray' | 'purple' | 'red';
};

type EventDisplayModel = {
  chips: EventChip[];
  primary: string;
  secondary?: string;
};

type EventRendererProps = {
  document: TimelineDocument;
  occurredAtLabel?: string;
};

type EventRenderer = {
  Inspector: FC<EventRendererProps>;
  MainTable: FC<EventRendererProps>;
};

const getEventAttribute = (
  document: TimelineDocument,
  keys: string[],
): string => {
  for (const key of keys) {
    const value = document.attributes?.[key]?.trim();
    if (value) return value;
  }
  return '';
};

const formatEventSummary = (
  eventName: string,
  value: string,
  confidence?: string,
): string => {
  const confidenceLabel = confidence ? ` (${confidence})` : '';
  return `[${eventName}] ${value}${confidenceLabel}`;
};

const formatSeconds = (value: string): string => {
  const numericValue = Number(value);
  if (!Number.isFinite(numericValue)) return value;
  return `${numericValue.toFixed(3)}s`;
};

const EventInspectorDisplay: FC<
  EventRendererProps & { display: EventDisplayModel }
> = ({ document, display, occurredAtLabel }) => (
  <>
    <div className="mb-2 flex min-w-0 items-center gap-2">
      <div className="flex min-w-0 flex-wrap items-center gap-1.5">
        {display.chips.map(chip => (
          <Tag key={`${document.id}-${chip.label}`} type={chip.type}>
            {chip.label}
          </Tag>
        ))}
      </div>
      {occurredAtLabel && (
        <span className="ml-auto whitespace-nowrap font-mono text-xs text-gray-500">
          {occurredAtLabel}
        </span>
      )}
    </div>
    <p className="truncate font-mono text-sm text-gray-900 dark:text-gray-100">
      {display.primary}
    </p>
    {display.secondary && (
      <p className="mt-1 truncate font-mono text-xs text-gray-500">
        {display.secondary}
      </p>
    )}
  </>
);

const getSttEventDisplay = (document: TimelineDocument): EventDisplayModel => {
  const script = getEventAttribute(document, ['script', 'transcript', 'text']);
  const error = getEventAttribute(document, ['error', 'message']);
  const confidence = getEventAttribute(document, ['confidence']);
  const language = getEventAttribute(document, ['language']);
  const wordCount = getEventAttribute(document, ['word_count', 'wordCount']);
  const threshold = getEventAttribute(document, ['threshold']);
  const provider = getEventAttribute(document, ['provider']);
  const operation = getEventAttribute(document, ['operation']);
  const chips: EventChip[] = [];

  if (document.name === 'stt.low_confidence') {
    chips.push({ label: 'low confidence', type: 'red' });
  }
  if (language && document.name === 'stt.completed') {
    chips.push({ label: `lang ${language}`, type: 'cool-gray' });
  }
  if (wordCount) {
    chips.push({
      label: `${wordCount} ${wordCount === '1' ? 'word' : 'words'}`,
      type: 'cool-gray',
    });
  }
  if (threshold) {
    chips.push({
      label: `threshold ${threshold}`,
      type: 'cool-gray',
    });
  }
  if (provider) chips.push({ label: provider, type: 'cool-gray' });
  if (operation) chips.push({ label: operation, type: 'cool-gray' });

  const secondary = [
    document.name,
    document.messageId && `message ${document.messageId}`,
  ]
    .filter(Boolean)
    .join(' · ');

  if (document.name === 'stt.closed') {
    return {
      chips,
      primary: formatEventSummary(document.name, 'STT stream closed'),
      secondary,
    };
  }

  if (document.name === 'stt.error') {
    return {
      chips,
      primary: formatEventSummary(document.name, error || 'STT error'),
      secondary,
    };
  }

  return {
    chips,
    primary: formatEventSummary(
      document.name,
      script || 'Transcript unavailable',
      confidence,
    ),
    secondary,
  };
};

const getVadEventDisplay = (document: TimelineDocument): EventDisplayModel => {
  const provider = getEventAttribute(document, ['provider']);
  const event = getEventAttribute(document, ['event']);
  const startAt = getEventAttribute(document, ['start_at', 'startAt']);
  const endAt = getEventAttribute(document, ['end_at', 'endAt']);
  const segmentCount = getEventAttribute(document, [
    'segment_count',
    'segmentCount',
  ]);
  const error = getEventAttribute(document, ['error', 'message']);
  const chips: EventChip[] = [];

  if (document.name === 'vad.error') {
    chips.push({ label: 'error', type: 'red' });
  }
  if (provider) chips.push({ label: provider, type: 'cool-gray' });
  if (event) chips.push({ label: event, type: 'cool-gray' });
  if (segmentCount) {
    chips.push({
      label: `${segmentCount} ${segmentCount === '1' ? 'segment' : 'segments'}`,
      type: 'cool-gray',
    });
  }

  const secondary = [
    document.name,
    document.messageId && `message ${document.messageId}`,
    document.contextId && `context ${document.contextId}`,
  ]
    .filter(Boolean)
    .join(' · ');

  switch (document.name) {
    case 'vad.speech_started':
      return {
        chips,
        primary: formatEventSummary(
          document.name,
          startAt
            ? `speech started at ${formatSeconds(startAt)}`
            : 'speech started',
        ),
        secondary,
      };
    case 'vad.speech_ended':
      return {
        chips,
        primary: formatEventSummary(
          document.name,
          startAt && endAt
            ? `speech ended ${formatSeconds(startAt)} - ${formatSeconds(endAt)}`
            : 'speech ended',
        ),
        secondary,
      };
    case 'vad.closed':
      return {
        chips,
        primary: formatEventSummary(document.name, 'VAD stream closed'),
        secondary,
      };
    case 'vad.error':
      return {
        chips,
        primary: formatEventSummary(document.name, error || 'VAD error'),
        secondary,
      };
    default:
      return {
        chips,
        primary: formatEventSummary(
          document.name,
          document.title || document.name,
        ),
        secondary,
      };
  }
};

const getEosEventDisplay = (document: TimelineDocument): EventDisplayModel => {
  const speech = getEventAttribute(document, ['speech', 'text']);
  const provider = getEventAttribute(document, ['provider']);
  const confidence = getEventAttribute(document, ['confidence']);
  const wordCount = getEventAttribute(document, ['word_count', 'wordCount']);
  const charCount = getEventAttribute(document, ['char_count', 'charCount']);
  const textToTriggerMs = getEventAttribute(document, [
    'text_to_trigger_ms',
    'textToTriggerMs',
  ]);
  const waitToTriggerMs = getEventAttribute(document, [
    'wait_to_trigger_ms',
    'waitToTriggerMs',
  ]);
  const chips: EventChip[] = [];

  if (provider) chips.push({ label: provider, type: 'cool-gray' });
  if (wordCount) {
    chips.push({
      label: `${wordCount} ${wordCount === '1' ? 'word' : 'words'}`,
      type: 'cool-gray',
    });
  }
  if (charCount) chips.push({ label: `${charCount} chars`, type: 'cool-gray' });
  if (textToTriggerMs) {
    chips.push({ label: `trigger ${textToTriggerMs}ms`, type: 'cool-gray' });
  }
  if (waitToTriggerMs) {
    chips.push({ label: `wait ${waitToTriggerMs}ms`, type: 'cool-gray' });
  }

  const secondary = [
    document.name,
    document.messageId && `message ${document.messageId}`,
    document.contextId && `context ${document.contextId}`,
  ]
    .filter(Boolean)
    .join(' · ');

  switch (document.name) {
    case 'eos.started':
      return {
        chips,
        primary: formatEventSummary(
          document.name,
          speech || 'end-of-speech analysis started',
        ),
        secondary,
      };
    case 'eos.completed':
      return {
        chips,
        primary: formatEventSummary(
          document.name,
          speech || 'end-of-speech detected',
          confidence,
        ),
        secondary,
      };
    case 'eos.closed':
      return {
        chips,
        primary: formatEventSummary(document.name, 'EOS stream closed'),
        secondary,
      };
    default:
      return {
        chips,
        primary: formatEventSummary(
          document.name,
          document.title || document.name,
        ),
        secondary,
      };
  }
};

const getTtsEventDisplay = (document: TimelineDocument): EventDisplayModel => {
  const text = getEventAttribute(document, ['text', 'script', 'message']);
  const provider = getEventAttribute(document, ['provider']);
  const type = getEventAttribute(document, ['type']);
  const operation = getEventAttribute(document, ['operation']);
  const error = getEventAttribute(document, ['error', 'message']);
  const errorType = getEventAttribute(document, ['error_type', 'errorType']);
  const path = getEventAttribute(document, ['path']);
  const chips: EventChip[] = [];

  if (provider) chips.push({ label: provider, type: 'cool-gray' });
  if (type) chips.push({ label: type, type: 'blue' });
  if (operation) chips.push({ label: operation, type: 'cool-gray' });
  if (errorType) chips.push({ label: errorType, type: 'red' });

  const secondary = [
    document.name,
    document.messageId && `message ${document.messageId}`,
    document.contextId && `context ${document.contextId}`,
    path && `path ${path}`,
  ]
    .filter(Boolean)
    .join(' · ');

  switch (document.name) {
    case 'tts.speaking':
      return {
        chips,
        primary: formatEventSummary(document.name, text || 'speech started'),
        secondary,
      };
    case 'tts.completed':
      return {
        chips,
        primary: formatEventSummary(document.name, 'speech completed'),
        secondary,
      };
    case 'tts.interrupted':
      return {
        chips,
        primary: formatEventSummary(document.name, 'speech interrupted'),
        secondary,
      };
    case 'tts.closed':
      return {
        chips,
        primary: formatEventSummary(
          document.name,
          provider ? `${provider} stream closed` : 'TTS stream closed',
        ),
        secondary,
      };
    case 'tts.error':
      return {
        chips,
        primary: formatEventSummary(document.name, error || 'TTS error'),
        secondary,
      };
    default:
      return {
        chips,
        primary: formatEventSummary(
          document.name,
          text || document.title || document.name,
        ),
        secondary,
      };
  }
};

const getAgentEventDisplay = (
  document: TimelineDocument,
): EventDisplayModel => {
  const script = getEventAttribute(document, [
    'script',
    'message',
    'text',
    'input_text',
    'question',
  ]);
  const provider = getEventAttribute(document, ['provider']);
  const inputCharCount = getEventAttribute(document, [
    'input_char_count',
    'inputCharCount',
  ]);
  const historyCount = getEventAttribute(document, [
    'history_count',
    'historyCount',
  ]);
  const responseCharCount = getEventAttribute(document, [
    'response_char_count',
    'responseCharCount',
  ]);
  const finishReason = getEventAttribute(document, [
    'finish_reason',
    'finishReason',
  ]);
  const toolCallCount = getEventAttribute(document, [
    'tool_call_count',
    'toolCallCount',
  ]);
  const reason = getEventAttribute(document, ['reason']);
  const source = getEventAttribute(document, ['source']);
  const packet = getEventAttribute(document, ['packet']);
  const error = getEventAttribute(document, ['error', 'message']);
  const code = getEventAttribute(document, ['code']);
  const fromNodeLabel = getEventAttribute(document, ['from_node_label']);
  const fromNodeId = getEventAttribute(document, ['from_node_id']);
  const toNodeLabel = getEventAttribute(document, ['to_node_label']);
  const toNodeId = getEventAttribute(document, ['to_node_id']);
  const transitionName = getEventAttribute(document, ['transition_name']);
  const routeHandles = getEventAttribute(document, ['route_handles']);
  const result = getEventAttribute(document, ['result']);
  const chips: EventChip[] = [];

  if (provider) chips.push({ label: provider, type: 'cool-gray' });
  if (inputCharCount) {
    chips.push({ label: `${inputCharCount} chars in`, type: 'cool-gray' });
  }
  if (historyCount) {
    chips.push({ label: `${historyCount} history`, type: 'cool-gray' });
  }
  if (responseCharCount) {
    chips.push({ label: `${responseCharCount} chars out`, type: 'cool-gray' });
  }
  if (finishReason) chips.push({ label: finishReason, type: 'cool-gray' });
  if (toolCallCount) {
    chips.push({ label: `${toolCallCount} tools`, type: 'cool-gray' });
  }
  if (reason) chips.push({ label: reason, type: 'cool-gray' });
  if (source) chips.push({ label: source, type: 'cool-gray' });
  if (code) chips.push({ label: `code ${code}`, type: 'red' });
  if (result) chips.push({ label: result, type: 'cool-gray' });

  const secondary = [
    document.messageId && `message ${document.messageId}`,
    document.contextId && `context ${document.contextId}`,
  ]
    .filter(Boolean)
    .join(' · ');

  switch (document.name) {
    case 'agent.started': {
      const details = [
        inputCharCount && `${inputCharCount} chars`,
        historyCount && `${historyCount} history`,
      ]
        .filter(Boolean)
        .join(', ');

      return {
        chips,
        primary: formatEventSummary(
          document.name,
          script || 'agent request started',
          details,
        ),
        secondary,
      };
    }
    case 'agent.completed': {
      const details = [
        responseCharCount && `${responseCharCount} chars`,
        finishReason,
        toolCallCount && `${toolCallCount} tools`,
      ]
        .filter(Boolean)
        .join(', ');

      return {
        chips,
        primary: formatEventSummary(
          document.name,
          script || 'agent response completed',
          details,
        ),
        secondary,
      };
    }
    case 'agent.discarded':
      return {
        chips,
        primary: formatEventSummary(
          document.name,
          script ||
            [
              reason || 'agent response discarded',
              source && `from ${source}`,
              packet && `packet ${packet}`,
            ]
              .filter(Boolean)
              .join(', '),
        ),
        secondary,
      };
    case 'agent.error':
      return {
        chips,
        primary: formatEventSummary(
          document.name,
          error || script || 'agent error',
          code && `code ${code}`,
        ),
        secondary,
      };
    case 'agent.transition.triggered':
      return {
        chips,
        primary: formatEventSummary(
          document.name,
          `${fromNodeLabel || fromNodeId || 'agent'} -> ${
            transitionName || 'transition'
          }`,
        ),
        secondary,
      };
    case 'agent.transition.matched':
      return {
        chips,
        primary: formatEventSummary(
          document.name,
          `${fromNodeLabel || fromNodeId || 'agent'} -> ${
            toNodeLabel || toNodeId || routeHandles || 'next node'
          }`,
        ),
        secondary,
      };
    case 'agent.transition.missing_edge':
      return {
        chips,
        primary: formatEventSummary(
          document.name,
          `${fromNodeLabel || fromNodeId || 'agent'} missing edge for ${
            routeHandles || transitionName || 'route'
          }`,
        ),
        secondary,
      };
    default:
      return {
        chips,
        primary: formatEventSummary(
          document.name,
          script || document.title || document.name,
        ),
        secondary,
      };
  }
};

const getTurnEventDisplay = (document: TimelineDocument): EventDisplayModel => {
  const oldContext = getEventAttribute(document, [
    'old_context_id',
    'previous_context_id',
  ]);
  const newContext = getEventAttribute(document, [
    'new_context_id',
    'context_id',
  ]);
  const source = getEventAttribute(document, ['source']);
  const reason = getEventAttribute(document, ['reason']);
  const previousState = getEventAttribute(document, [
    'previous_state',
    'state',
  ]);
  const trigger = getEventAttribute(document, ['trigger']);
  const text = getEventAttribute(document, ['text', 'script', 'message']);
  const contextPair =
    oldContext && newContext
      ? `${oldContext} -> ${newContext}`
      : newContext || oldContext || 'turn';
  const details = [
    text && `context=${contextPair}`,
    source && `source=${source}`,
    reason && `reason=${reason}`,
    previousState && `state=${previousState}`,
    trigger && `trigger=${trigger}`,
  ]
    .filter(Boolean)
    .join(' | ');

  return {
    chips: [
      { label: 'turn', type: 'cool-gray' },
      { label: document.name.replace('turn.', ''), type: 'blue' },
    ],
    primary: formatEventSummary(document.name, text || contextPair),
    secondary: details || undefined,
  };
};

const GenericEventInspector: FC<EventRendererProps> = ({
  document,
  occurredAtLabel,
}) => (
  <>
    <div className="mb-2 flex min-w-0 items-center gap-2">
      <Tag type="cool-gray">{getDocumentComponent(document)}</Tag>
      <Tag type="blue">{document.scope}</Tag>
      {occurredAtLabel && (
        <span className="ml-auto whitespace-nowrap font-mono text-xs text-gray-500">
          {occurredAtLabel}
        </span>
      )}
    </div>
    <p className="truncate font-mono text-sm text-gray-900 dark:text-gray-100">
      {document.title || document.name}
    </p>
  </>
);

const GenericEventMainTable: FC<EventRendererProps> = ({ document }) => (
  <p className="truncate font-mono text-[13px]">
    [{getDocumentComponent(document)}] {document.title || document.name}
  </p>
);

const SttEventInspector: FC<EventRendererProps> = ({
  document,
  occurredAtLabel,
}) => {
  const display = getSttEventDisplay(document);

  return (
    <EventInspectorDisplay
      document={document}
      display={display}
      occurredAtLabel={occurredAtLabel}
    />
  );
};

const SttEventMainTable: FC<EventRendererProps> = ({ document }) => {
  const display = getSttEventDisplay(document);

  return (
    <p className="truncate font-mono text-[13px] text-gray-900 dark:text-gray-100">
      {display.primary}
    </p>
  );
};

const VadEventInspector: FC<EventRendererProps> = ({
  document,
  occurredAtLabel,
}) => {
  const display = getVadEventDisplay(document);

  return (
    <EventInspectorDisplay
      document={document}
      display={display}
      occurredAtLabel={occurredAtLabel}
    />
  );
};

const VadEventMainTable: FC<EventRendererProps> = ({ document }) => {
  const display = getVadEventDisplay(document);

  return (
    <p className="truncate font-mono text-[13px] text-gray-900 dark:text-gray-100">
      {display.primary}
    </p>
  );
};

const EosEventInspector: FC<EventRendererProps> = ({
  document,
  occurredAtLabel,
}) => {
  const display = getEosEventDisplay(document);

  return (
    <EventInspectorDisplay
      document={document}
      display={display}
      occurredAtLabel={occurredAtLabel}
    />
  );
};

const EosEventMainTable: FC<EventRendererProps> = ({ document }) => {
  const display = getEosEventDisplay(document);

  return (
    <p className="truncate font-mono text-[13px] text-gray-900 dark:text-gray-100">
      {display.primary}
    </p>
  );
};

const TtsEventInspector: FC<EventRendererProps> = ({
  document,
  occurredAtLabel,
}) => {
  const display = getTtsEventDisplay(document);

  return (
    <>
      <div className="mb-2 flex min-w-0 items-center gap-2">
        <div className="flex min-w-0 flex-wrap items-center gap-1.5">
          {display.chips.map(chip => (
            <Tag key={`${document.id}-${chip.label}`} type={chip.type}>
              {chip.label}
            </Tag>
          ))}
        </div>
        {occurredAtLabel && (
          <span className="ml-auto whitespace-nowrap font-mono text-xs text-gray-500">
            {occurredAtLabel}
          </span>
        )}
      </div>
      <p className="whitespace-pre-wrap break-words font-mono text-sm text-gray-900 dark:text-gray-100">
        {display.primary}
      </p>
      {display.secondary && (
        <p className="mt-1 truncate font-mono text-xs text-gray-500">
          {display.secondary}
        </p>
      )}
    </>
  );
};

const TtsEventMainTable: FC<EventRendererProps> = ({ document }) => {
  const display = getTtsEventDisplay(document);

  return (
    <p className="truncate font-mono text-[13px] text-gray-900 dark:text-gray-100">
      {display.primary}
    </p>
  );
};

const AgentEventInspector: FC<EventRendererProps> = ({
  document,
  occurredAtLabel,
}) => {
  const display = getAgentEventDisplay(document);

  return (
    <>
      <div className="mb-2 flex min-w-0 items-center gap-2">
        <div className="flex min-w-0 flex-wrap items-center gap-1.5">
          {display.chips.map(chip => (
            <Tag key={`${document.id}-${chip.label}`} type={chip.type}>
              {chip.label}
            </Tag>
          ))}
        </div>
        {occurredAtLabel && (
          <span className="ml-auto whitespace-nowrap font-mono text-xs text-gray-500">
            {occurredAtLabel}
          </span>
        )}
      </div>
      <p className="whitespace-pre-wrap break-words font-mono text-sm text-gray-900 dark:text-gray-100">
        {display.primary}
      </p>
      {display.secondary && (
        <p className="mt-1 truncate font-mono text-xs text-gray-500">
          {display.secondary}
        </p>
      )}
    </>
  );
};

const AgentEventMainTable: FC<EventRendererProps> = ({ document }) => {
  const display = getAgentEventDisplay(document);

  return (
    <p className="truncate font-mono text-[13px] text-gray-900 dark:text-gray-100">
      {display.primary}
    </p>
  );
};

const TurnEventInspector: FC<EventRendererProps> = ({
  document,
  occurredAtLabel,
}) => {
  const display = getTurnEventDisplay(document);
  return (
    <EventInspectorDisplay
      document={document}
      display={display}
      occurredAtLabel={occurredAtLabel}
    />
  );
};

const TurnEventMainTable: FC<EventRendererProps> = ({ document }) => {
  const display = getTurnEventDisplay(document);

  return (
    <p className="truncate font-mono text-[13px] text-gray-900 dark:text-gray-100">
      {display.primary}
      {display.secondary ? ` ${display.secondary}` : ''}
    </p>
  );
};

const GENERIC_EVENT_RENDERER: EventRenderer = {
  Inspector: GenericEventInspector,
  MainTable: GenericEventMainTable,
};

const TURN_EVENT_RENDERER: EventRenderer = {
  Inspector: TurnEventInspector,
  MainTable: TurnEventMainTable,
};

const STT_EVENT_RENDERER: EventRenderer = {
  Inspector: SttEventInspector,
  MainTable: SttEventMainTable,
};

const VAD_EVENT_RENDERER: EventRenderer = {
  Inspector: VadEventInspector,
  MainTable: VadEventMainTable,
};

const EOS_EVENT_RENDERER: EventRenderer = {
  Inspector: EosEventInspector,
  MainTable: EosEventMainTable,
};

const TTS_EVENT_RENDERER: EventRenderer = {
  Inspector: TtsEventInspector,
  MainTable: TtsEventMainTable,
};

const AGENT_EVENT_RENDERER: EventRenderer = {
  Inspector: AgentEventInspector,
  MainTable: AgentEventMainTable,
};

const getEventRenderer = (eventName: string): EventRenderer => {
  switch (eventName) {
    case 'turn.interrupted':
    case 'turn.change':
    case 'turn.started':
      return TURN_EVENT_RENDERER;
    case 'stt.interim':
    case 'stt.completed':
    case 'stt.low_confidence':
    case 'stt.closed':
    case 'stt.error':
      return STT_EVENT_RENDERER;
    case 'vad.speech_started':
    case 'vad.speech_ended':
    case 'vad.closed':
    case 'vad.error':
      return VAD_EVENT_RENDERER;
    case 'eos.started':
    case 'eos.completed':
    case 'eos.closed':
      return EOS_EVENT_RENDERER;
    case 'tts.speaking':
    case 'tts.completed':
    case 'tts.interrupted':
    case 'tts.closed':
    case 'tts.error':
      return TTS_EVENT_RENDERER;
    case 'agent.started':
    case 'agent.completed':
    case 'agent.discarded':
    case 'agent.error':
    case 'agent.transition.triggered':
    case 'agent.transition.matched':
    case 'agent.transition.missing_edge':
      return AGENT_EVENT_RENDERER;
    default:
      return GENERIC_EVENT_RENDERER;
  }
};

export const EventInspectorContent: FC<EventRendererProps> = ({
  document,
  occurredAtLabel,
}) => {
  const Renderer = getEventRenderer(document.name);
  return (
    <Renderer.Inspector document={document} occurredAtLabel={occurredAtLabel} />
  );
};

export const EventMainTableSummary: FC<EventRendererProps> = ({ document }) => {
  const Renderer = getEventRenderer(document.name);
  return <Renderer.MainTable document={document} />;
};
