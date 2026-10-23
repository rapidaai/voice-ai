import { Fragment, useCallback, useEffect, useState } from 'react';
import {
  AssistantChatContext,
  useAssistantChat,
} from '@/stores/assistant/assistant-chat.store';
import { ConversationMessages } from '@/app/pages/assistant/view/conversations/conversation-messages';
import { useParams } from 'react-router-dom';
import { useCurrentCredential } from '@/hooks/use-credential';
import type { AssistantConversation } from '@rapidaai/react';
import { useRapidaStore } from '@/stores/app';
import { ErrorContainer, PageLoader } from '@/app/components/ui/feedback';
import {
  ArrowLeft,
  Renew,
  DataBase,
  DataCheck,
  ChartLine,
} from '@carbon/icons-react';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { GhostButton } from '@/app/components/ui/primitives';
import { PageHeaderBlock } from '@/app/components/layout/blocks/page-header-block';
import { cn } from '@/utils';
import { CarbonStatusIndicator } from '@/app/components/ui/feedback';
import { getStatusMetric } from '@/utils/metadata';
import { Tabs } from '@/app/components/ui/primitives';
import { getAssistantConversationDetail } from '@/clients/assistant.client';
import { SessionEmptyState } from './session-empty-state';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@carbon/react';

// ── Tab definitions ───────────────────────────────────────────────────────────

const TABS = [
  { key: 'messages', label: 'Messages' },
  { key: 'context', label: 'Context' },
  { key: 'arguments', label: 'Arguments' },
  { key: 'analysis', label: 'Analysis' },
  { key: 'metrics', label: 'Metrics' },
] as const;

type TabKey = (typeof TABS)[number]['key'];

// ── Page component ────────────────────────────────────────────────────────────

export function ConversationDetailPage() {
  const { assistantId, sessionId } = useParams();
  const { authId, token, projectId } = useCurrentCredential();
  const { loading, showLoader, hideLoader } = useRapidaStore();
  const actions = useAssistantChat();
  const navigator = useGlobalNavigation();
  const [currentConversation, setCurrentConversation] =
    useState<AssistantConversation | null>(null);
  const [loadError, setLoadError] = useState(false);
  const [activeTab, setActiveTab] = useState<TabKey>('messages');
  const selectedTabIndex = Math.max(
    TABS.findIndex(({ key }) => key === activeTab),
    0,
  );

  const get = useCallback(() => {
    if (!assistantId || !sessionId) {
      setLoadError(true);
      return;
    }

    setLoadError(false);
    showLoader();
    getAssistantConversationDetail({
      assistantId,
      conversationId: sessionId,
      fields: ['recording'],
      auth: { projectId, token, userId: authId },
    })
      .then(response => {
        hideLoader();
        if (response?.getSuccess() && response.getData()) {
          setCurrentConversation(response.getData()!);
          return;
        }
        setLoadError(true);
      })
      .catch(() => {
        hideLoader();
        setLoadError(true);
      });
  }, [
    assistantId,
    authId,
    hideLoader,
    projectId,
    sessionId,
    showLoader,
    token,
  ]);

  useEffect(() => {
    get();
  }, [get]);

  if (loading || (!loadError && currentConversation == null)) {
    return <PageLoader />;
  }

  if (loadError || currentConversation == null) {
    return (
      <ErrorContainer
        className="h-full"
        code="503"
        title="Session unavailable"
        description="We couldn't load this session. Try again in a moment."
        actionLabel="Try again"
        actionIcon={Renew}
        onAction={get}
      />
    );
  }

  const renderContent = () => {
    switch (activeTab) {
      case 'analysis': {
        const items = currentConversation
          .getMetadataList()
          .filter(x => x.getKey().startsWith('analysis.'));
        return items.length === 0 ? (
          <SessionEmptyState
            icon={ChartLine}
            title="No analysis available"
            description="This session does not contain analysis data."
            className="h-full"
          />
        ) : (
          <div className="flex flex-col divide-y divide-gray-200 dark:divide-gray-800">
            {items.map((x, idx) => (
              <div
                key={idx}
                className="px-6 py-4 space-y-3 bg-white dark:bg-gray-900"
              >
                <p className="text-xs font-medium uppercase tracking-[0.08em] text-[var(--cds-text-secondary)]">
                  {x.getKey().replace('.', ' › ')}
                </p>
                <JsonViewer data={x.getValue()} />
              </div>
            ))}
          </div>
        );
      }

      case 'context': {
        const contexts = currentConversation.getContextsList();
        return contexts.length === 0 ? (
          <SessionEmptyState
            icon={DataBase}
            title="No context available"
            description="No knowledge context was retrieved for this session."
            className="h-full"
          />
        ) : (
          <div className="flex flex-col divide-y divide-gray-200 dark:divide-gray-800">
            {contexts.map((x, idx) => (
              <div key={idx} className="px-6 py-5 space-y-4">
                <DataField
                  label="Query"
                  value={x
                    .getQuery()
                    ?.getFieldsMap()
                    .get('query')
                    ?.getStringValue()}
                />
                <DataField
                  label="Additional Filter"
                  value={JSON.stringify(
                    x
                      .getQuery()
                      ?.getFieldsMap()
                      .get('additionalData')
                      ?.getStructValue()
                      ?.toJavaScript(),
                  )}
                />
                <DataField
                  label={`Content${
                    x
                      .getResult()
                      ?.getFieldsMap()
                      .get('score')
                      ?.getNumberValue() != null
                      ? ` · score ${x.getResult()?.getFieldsMap().get('score')?.getNumberValue()}`
                      : ''
                  }`}
                  value={x
                    .getResult()
                    ?.getFieldsMap()
                    .get('content')
                    ?.getStringValue()}
                />
                <DataField
                  label="Document"
                  value={JSON.stringify(
                    x.getMetadata()?.toJavaScript(),
                    null,
                    2,
                  )}
                  mono
                />
              </div>
            ))}
          </div>
        );
      }

      case 'messages':
        return (
          <AssistantChatContext.Provider value={actions}>
            <ConversationMessages
              conversation={currentConversation}
              assistantId={currentConversation.getAssistantid()}
              conversationId={currentConversation.getId()}
            />
          </AssistantChatContext.Provider>
        );

      case 'metrics':
        return currentConversation.getMetricsList().length > 0 ? (
          <div className="w-full">
            <Table
              aria-label="Session metrics"
              className="w-full table-fixed"
              size="lg"
              useZebraStyles={false}
            >
              <TableHead>
                <TableRow>
                  <TableHeader className="w-[30%]">Name</TableHeader>
                  <TableHeader className="w-1/5">Value</TableHeader>
                  <TableHeader className="w-1/2">Description</TableHeader>
                </TableRow>
              </TableHead>
              <TableBody>
                {currentConversation.getMetricsList().map((m, i) => (
                  <TableRow key={i}>
                    <TableCell className="align-top whitespace-normal break-words">
                      {m.getName()}
                    </TableCell>
                    <TableCell className="align-top whitespace-normal break-words">
                      {m.getValue()}
                    </TableCell>
                    <TableCell className="align-top whitespace-normal break-words">
                      {m.getDescription()}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        ) : (
          <SessionEmptyState
            icon={ChartLine}
            title="No metrics available"
            description="This session does not contain metrics."
            className="h-full"
          />
        );

      case 'arguments': {
        const hasArguments =
          currentConversation.getArgumentsList().length > 0 ||
          currentConversation.getOptionsList().length > 0 ||
          currentConversation.getMetadataList().length > 0;
        if (!hasArguments) {
          return (
            <SessionEmptyState
              icon={DataCheck}
              title="No arguments available"
              description="This session does not contain arguments, options, or metadata."
              className="h-full"
            />
          );
        }
        return (
          <div className="w-full">
            <Table
              aria-label="Session arguments"
              className="w-full table-fixed"
              size="lg"
              useZebraStyles={false}
            >
              <TableHead>
                <TableRow>
                  <TableHeader className="w-1/5">Type</TableHeader>
                  <TableHeader className="w-[30%]">Name</TableHeader>
                  <TableHeader className="w-1/2">Value</TableHeader>
                </TableRow>
              </TableHead>
              <TableBody>
                {currentConversation.getArgumentsList().map((m, i) => (
                  <TableRow key={`arg-${i}`}>
                    <TableCell className="align-top whitespace-normal">
                      Argument
                    </TableCell>
                    <TableCell className="align-top whitespace-normal break-words">
                      {m.getName()}
                    </TableCell>
                    <TableCell className="align-top whitespace-normal break-words">
                      {m.getValue()}
                    </TableCell>
                  </TableRow>
                ))}
                {currentConversation.getOptionsList().map((m, i) => (
                  <TableRow key={`opt-${i}`}>
                    <TableCell className="align-top whitespace-normal">
                      Option
                    </TableCell>
                    <TableCell className="align-top whitespace-normal break-words">
                      {m.getKey()}
                    </TableCell>
                    <TableCell className="align-top whitespace-normal break-words">
                      {m.getValue()}
                    </TableCell>
                  </TableRow>
                ))}
                {currentConversation.getMetadataList().map((m, i) => (
                  <TableRow key={`meta-${i}`}>
                    <TableCell className="align-top whitespace-normal">
                      Metadata
                    </TableCell>
                    <TableCell className="align-top whitespace-normal break-words">
                      {m.getKey()}
                    </TableCell>
                    <TableCell className="align-top whitespace-normal break-words">
                      {m.getValue()}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        );
      }
    }
  };

  return (
    <>
      {/* ── Page header: breadcrumb + status + refresh ── */}
      <PageHeaderBlock>
        <div className="flex items-center gap-1.5 min-w-0">
          <GhostButton
            size="sm"
            hasIconOnly
            renderIcon={ArrowLeft}
            iconDescription="Back to sessions"
            onClick={() => navigator.goToAssistantSessionList(assistantId!)}
          />
          <span className="text-sm font-medium text-[var(--cds-text-secondary)]">
            Sessions
          </span>
          <span className="text-gray-300 dark:text-gray-600">/</span>
          <span className="text-sm font-medium text-gray-900 dark:text-gray-100 font-mono truncate">
            {sessionId}
          </span>
        </div>
        <div className="flex items-center gap-2 px-2">
          <CarbonStatusIndicator
            state={getStatusMetric(currentConversation.getMetricsList())}
          />
          <div className="w-px h-4 bg-gray-200 dark:bg-gray-700" />
          <GhostButton
            size="sm"
            hasIconOnly
            renderIcon={Renew}
            iconDescription="Refresh"
            onClick={get}
          />
        </div>
      </PageHeaderBlock>

      <Tabs
        tabs={TABS.map(({ label }) => label)}
        selectedIndex={selectedTabIndex}
        onChange={idx => setActiveTab(TABS[idx]?.key ?? 'messages')}
        contained
        fill
        aria-label="Session detail tabs"
        className="w-full"
        tabListClassName="w-full border-y border-border-subtle"
        panelsClassName="h-full w-full"
        panelClassName="!block !h-full !min-h-0 !w-full !overflow-auto !p-0"
      >
        {TABS.map(({ key }) => (
          <Fragment key={key}>
            {key === activeTab ? renderContent() : null}
          </Fragment>
        ))}
      </Tabs>
    </>
  );
}

// ── Helper components ─────────────────────────────────────────────────────────

const DataField = ({
  label,
  value,
  mono = false,
}: {
  label: string;
  value?: string | null;
  mono?: boolean;
}) => (
  <div>
    <p className="text-xs font-medium uppercase tracking-[0.08em] text-[var(--cds-text-secondary)] mb-1">
      {label}
    </p>
    <p
      className={cn(
        'text-sm text-gray-900 dark:text-gray-100 leading-relaxed',
        mono && 'font-mono',
      )}
    >
      {value ?? '—'}
    </p>
  </div>
);

interface JsonViewerProps {
  data: string | any;
  preview?: boolean;
}

const JsonViewer = ({ data, preview = false }: JsonViewerProps) => {
  const parsedData = typeof data === 'string' ? JSON.parse(data) : data;

  const isSimpleResult =
    parsedData &&
    typeof parsedData === 'object' &&
    parsedData.result &&
    Object.keys(parsedData).length === 1;

  const extractInsights = (
    obj: any,
  ): { key: string; value: any; type: string }[] => {
    const insights: { key: string; value: any; type: string }[] = [];

    const processObject = (object: any, prefix = '') => {
      if (typeof object === 'object' && object !== null) {
        Object.entries(object).forEach(([key, value]) => {
          const formattedKey = prefix ? `${prefix}.${key}` : key;
          if (
            typeof value === 'object' &&
            value !== null &&
            !Array.isArray(value)
          ) {
            processObject(value, formattedKey);
          } else {
            insights.push({
              key: formattedKey.replace(/_/g, ' ').replace(/\./g, ' › '),
              value,
              type: Array.isArray(value) ? 'array' : typeof value,
            });
          }
        });
      }
    };

    processObject(obj);
    return insights;
  };

  const insights = extractInsights(parsedData);
  const displayInsights =
    preview && insights.length > 3 ? insights.slice(0, 3) : insights;

  const formatValue = (value: any, type: string) => {
    if (type === 'array')
      return Array.isArray(value) ? value.join(', ') : String(value);
    if (type === 'number')
      return typeof value === 'number'
        ? `${Math.round(value * 100)}%`
        : String(value);
    return String(value);
  };

  if (isSimpleResult) {
    return (
      <p className="text-sm text-gray-900 dark:text-gray-100 leading-relaxed">
        {parsedData.result}
      </p>
    );
  }

  return (
    <div className="space-y-2">
      {displayInsights.map((insight, index) => (
        <div
          key={index}
          className="border-l-2 border-primary bg-primary/5 pl-4 py-2"
        >
          <p className="text-xs font-medium uppercase tracking-[0.08em] text-[var(--cds-text-secondary)] mb-1 capitalize">
            {insight.key}
          </p>
          <p className="text-sm text-gray-900 dark:text-gray-100 leading-relaxed">
            {formatValue(insight.value, insight.type)}
          </p>
        </div>
      ))}
    </div>
  );
};

export default JsonViewer;
