import type { FC } from 'react';
import { Tag } from '@carbon/react';
import type { TimelineDocument } from './types';
import { getDocumentComponent } from './utils';

type LogChip = {
  label: string;
  type: 'blue' | 'cool-gray' | 'gray' | 'red';
};

type LogDisplayModel = {
  chips: LogChip[];
  primary: string;
  secondary?: string;
};

type LogRendererProps = {
  document: TimelineDocument;
  occurredAtLabel?: string;
};

type LogRenderer = {
  Inspector: FC<LogRendererProps>;
  MainTable: FC<LogRendererProps>;
};

const getLogAttribute = (
  document: TimelineDocument,
  keys: string[],
): string => {
  for (const key of keys) {
    const value = document.attributes?.[key]?.trim();
    if (value) return value;
  }
  return '';
};

const formatLogSummary = (
  level: string,
  component: string,
  message: string,
): string => `[${level.toLowerCase()}] ${component} ${message}`;

const LogInspectorDisplay: FC<
  LogRendererProps & { display: LogDisplayModel }
> = ({ document, display, occurredAtLabel }) => (
  <>
    <div className="mb-2 flex min-w-0 items-center gap-2">
      <span
        className={[
          'font-mono text-xs text-gray-500',
          document.level.toLowerCase() === 'error'
            ? 'text-red-600 dark:text-red-400'
            : '',
        ].join(' ')}
      >
        {document.level.toLowerCase()}
      </span>
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

const getGenericLogDisplay = (document: TimelineDocument): LogDisplayModel => ({
  chips: [{ label: getDocumentComponent(document), type: 'cool-gray' }],
  primary: `[${document.level.toLowerCase()}] ${
    document.title || document.name
  }`,
});

const getEosLogDisplay = (document: TimelineDocument): LogDisplayModel => {
  const provider = getLogAttribute(document, ['provider']);
  const operation = getLogAttribute(document, ['operation']);
  const packet = getLogAttribute(document, ['packet']);
  const source = getLogAttribute(document, ['source']);
  const error = getLogAttribute(document, ['error']);
  const errorType = getLogAttribute(document, ['error_type', 'errorType']);
  const chips: LogChip[] = [];

  if (provider) chips.push({ label: provider, type: 'cool-gray' });
  if (operation) chips.push({ label: operation, type: 'cool-gray' });
  if (packet) chips.push({ label: packet, type: 'cool-gray' });
  if (source) chips.push({ label: source, type: 'cool-gray' });
  if (errorType) chips.push({ label: errorType, type: 'red' });

  const message = document.title || document.name;
  const lowerMessage = message.toLowerCase();
  const level = document.level.toLowerCase();
  const summary =
    level === 'error'
      ? formatLogSummary(
          document.level,
          'eos',
          `${operation || 'operation'} failed: ${error || message}`,
        )
      : lowerMessage.includes('initialization completed')
        ? formatLogSummary(
            document.level,
            'eos',
            `initialized${provider ? `: ${provider}` : ''}`,
          )
        : formatLogSummary(document.level, 'eos', message);

  const secondary = [
    document.contextId && `context ${document.contextId}`,
    document.messageRole && `role ${document.messageRole}`,
  ]
    .filter(Boolean)
    .join(' · ');

  return {
    chips,
    primary: summary,
    secondary,
  };
};

const GenericLogInspector: FC<LogRendererProps> = ({
  document,
  occurredAtLabel,
}) => (
  <LogInspectorDisplay
    document={document}
    display={getGenericLogDisplay(document)}
    occurredAtLabel={occurredAtLabel}
  />
);

const GenericLogMainTable: FC<LogRendererProps> = ({ document }) => {
  const display = getGenericLogDisplay(document);

  return <p className="truncate font-mono text-[13px]">{display.primary}</p>;
};

const EosLogInspector: FC<LogRendererProps> = ({
  document,
  occurredAtLabel,
}) => (
  <LogInspectorDisplay
    document={document}
    display={getEosLogDisplay(document)}
    occurredAtLabel={occurredAtLabel}
  />
);

const EosLogMainTable: FC<LogRendererProps> = ({ document }) => {
  const display = getEosLogDisplay(document);

  return (
    <p className="truncate font-mono text-[13px] text-gray-900 dark:text-gray-100">
      {display.primary}
    </p>
  );
};

const GENERIC_LOG_RENDERER: LogRenderer = {
  Inspector: GenericLogInspector,
  MainTable: GenericLogMainTable,
};

const EOS_LOG_RENDERER: LogRenderer = {
  Inspector: EosLogInspector,
  MainTable: EosLogMainTable,
};

const getLogRenderer = (document: TimelineDocument): LogRenderer => {
  switch (getDocumentComponent(document)) {
    case 'eos':
      return EOS_LOG_RENDERER;
    default:
      return GENERIC_LOG_RENDERER;
  }
};

export const LogInspectorContent: FC<LogRendererProps> = ({
  document,
  occurredAtLabel,
}) => {
  const Renderer = getLogRenderer(document);
  return (
    <Renderer.Inspector document={document} occurredAtLabel={occurredAtLabel} />
  );
};

export const LogMainTableSummary: FC<LogRendererProps> = ({ document }) => {
  const Renderer = getLogRenderer(document);
  return <Renderer.MainTable document={document} />;
};
