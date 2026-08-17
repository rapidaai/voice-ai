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

const GENERIC_EVENT_RENDERER: EventRenderer = {
  Inspector: GenericEventInspector,
  MainTable: GenericEventMainTable,
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

const getEventRenderer = (eventName: string): EventRenderer => {
  switch (eventName) {
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
