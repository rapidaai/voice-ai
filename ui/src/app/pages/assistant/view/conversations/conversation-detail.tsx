import { useEffect, useState } from 'react';
import {
  AssistantChatContext,
  useAssistantChat,
} from '@/hooks/use-assistant-chat';
import { ConversationMessages } from '@/app/pages/assistant/view/conversations/conversation-messages';
import {
  ConnectionConfig,
  FieldSelector,
  GetAssistantConversation,
  GetAssistantConversationRequest,
} from '@rapidaai/react';
import { useParams } from 'react-router-dom';
import { useCurrentCredential } from '@/hooks/use-credential';
import { AssistantConversation } from '@rapidaai/react';
import { useRapidaStore } from '@/hooks';
import { SideTab } from '@/app/components/tab-link';
import { PageLoader } from '@/app/components/Loader/page-loader';
import {
  Activity,
  BookText,
  ChartArea,
  ChevronLeft,
  DownloadIcon,
  MessagesSquare,
  Parentheses,
  RotateCw,
} from 'lucide-react';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { IBlueButton } from '@/app/components/Form/Button';
import { PageHeaderBlock } from '@/app/components/blocks/page-header-block';
import { PageTitleBlock } from '@/app/components/blocks/page-title-block';
import { Table } from '@/app/components/base/tables/table';
import { TableHead } from '@/app/components/base/tables/table-head';
import { TableBody } from '@/app/components/base/tables/table-body';
import { TableRow } from '@/app/components/base/tables/table-row';
import { TableCell } from '@/app/components/base/tables/table-cell';
import { BlueNoticeBlock } from '@/app/components/container/message/notice-block';
import { PaginationButtonBlock } from '@/app/components/blocks/pagination-button-block';
import { ActionableEmptyMessage } from '@/app/components/container/message/actionable-empty-message';
import { connectionConfig } from '@/configs';

/**
 *
 * @param param0
 * @returns
 */

export function ConversationDetailPage({}) {
  const { assistantId, sessionId } = useParams();
  const { authId, token, projectId } = useCurrentCredential();
  const { loading, showLoader, hideLoader } = useRapidaStore();
  const actions = useAssistantChat();
  const navigator = useGlobalNavigation();
  const [currentConversation, setCurrentConversation] =
    useState<AssistantConversation | null>(null);

  const [activeTab, setActiveTab] = useState('messages');

  const get = () => {
    showLoader();
    const request = new GetAssistantConversationRequest();
    request.setAssistantid(assistantId!);
    request.setId(sessionId!);
    const filed = new FieldSelector();
    filed.setField('recording');
    request.addSelectors(filed);
    GetAssistantConversation(
      connectionConfig,
      request,
      ConnectionConfig.WithDebugger({
        authorization: token,
        userId: authId,
        projectId: projectId,
      }),
    )
      .then(response => {
        hideLoader();
        if (response?.getSuccess() && response.getData()) {
          setCurrentConversation(response.getData()!);
        }
      })
      .catch(err => {
        hideLoader();
      });
  };

  //
  useEffect(() => {
    if (!assistantId || !sessionId) return;
    get();
  }, [assistantId, sessionId]);

  if (loading || currentConversation == null) {
    return <PageLoader />;
  }
  const renderContent = () => {
    switch (activeTab) {
      case 'analysis':
        return (
          <div className="flex-1 flex flex-col h-full relative">
            <PageHeaderBlock className="border-b h-10">
              <PageTitleBlock>Analysis</PageTitleBlock>
              <PaginationButtonBlock>
                <IBlueButton onClick={() => {}}>
                  <DownloadIcon strokeWidth={1.5} className="h-4 w-4" />
                </IBlueButton>
                <IBlueButton
                  onClick={() => {
                    get();
                  }}
                >
                  <RotateCw strokeWidth={1.5} className="h-4 w-4" />
                </IBlueButton>
              </PaginationButtonBlock>
            </PageHeaderBlock>

            {currentConversation
              .getMetadataList()
              .filter(x => x.getKey().startsWith('analysis.')).length === 0 ? (
              <div className="my-auto mx-auto flex flex-1">
                <ActionableEmptyMessage
                  title="No Analysis"
                  subtitle="There are no analysis yet done for the conversation"
                />
              </div>
            ) : (
              <div className="flex flex-1 p-3 flex-col space-y-4 divide-y-2">
                {currentConversation
                  .getMetadataList()
                  .filter(x => x.getKey().startsWith('analysis.'))
                  .map((x, idx) => (
                    <div
                      key={idx}
                      className="space-y-3 px-4 py-3 bg-white dark:bg-gray-950"
                    >
                      <div className="capitalize font-medium text-base">
                        {x.getKey().replace('.', ' > ')}
                      </div>
                      <JsonViewer data={x.getValue()} />
                    </div>
                  ))}
              </div>
            )}
          </div>
        );

      case 'context':
        return (
          <div className="flex-1 flex flex-col h-full relative">
            <PageHeaderBlock className="border-b h-10">
              <PageTitleBlock>Knowledge Contexts</PageTitleBlock>
              <PaginationButtonBlock>
                <IBlueButton onClick={() => {}}>
                  <DownloadIcon strokeWidth={1.5} className="h-4 w-4" />
                </IBlueButton>
                <IBlueButton
                  onClick={() => {
                    get();
                  }}
                >
                  <RotateCw strokeWidth={1.5} className="h-4 w-4" />
                </IBlueButton>
              </PaginationButtonBlock>
            </PageHeaderBlock>

            {currentConversation.getContextsList().length === 0 ? (
              <div className="my-auto mx-auto flex flex-1">
                <ActionableEmptyMessage
                  title="No Context"
                  subtitle="There are no context yet for the conversation"
                />
              </div>
            ) : (
              <div className="flex flex-1 p-3 flex-col space-y-4 divide-y-2">
                {currentConversation.getContextsList().map((x, idx) => (
                  <div key={idx} className="space-y-3 px-4 py-3">
                    {/* Query */}
                    <div>
                      <h3 className="font-semibold mb-2">Query</h3>
                      <p className="opacity-80 leading-relaxed text-sm">
                        {x
                          .getQuery()
                          ?.getFieldsMap()
                          .get('query')
                          ?.getStringValue()}
                      </p>
                    </div>
                    <div>
                      <h3 className="font-semibold mb-2">Additional Filter</h3>
                      <p className="opacity-80 leading-relaxed text-sm">
                        {JSON.stringify(
                          x
                            .getQuery()
                            ?.getFieldsMap()
                            .get('additionalData')
                            ?.getStructValue()
                            ?.toJavaScript(),
                        )}
                      </p>
                    </div>
                    <div>
                      <h3 className="font-semibold mb-2">
                        Content
                        <small>
                          {x
                            .getResult()
                            ?.getFieldsMap()
                            .get('score')
                            ?.getNumberValue()}
                        </small>
                      </h3>
                      <p className="opacity-80 leading-relaxed text-sm mb-2">
                        {x
                          .getResult()
                          ?.getFieldsMap()
                          .get('content')
                          ?.getStringValue()}
                      </p>
                      <h3 className="font-semibold mb-2">Document</h3>
                      <p className="opacity-80 leading-relaxed text-sm">
                        {JSON.stringify(
                          x.getMetadata()?.toJavaScript(),
                          null,
                          2,
                        )}
                      </p>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        );
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
        return (
          <>
            <PageHeaderBlock className="border-b h-10">
              <PageTitleBlock>Metrics</PageTitleBlock>
              <PaginationButtonBlock>
                <IBlueButton onClick={() => {}}>
                  <DownloadIcon strokeWidth={1.5} className="h-4 w-4" />
                </IBlueButton>
                <IBlueButton
                  onClick={() => {
                    get();
                  }}
                >
                  <RotateCw strokeWidth={1.5} className="h-4 w-4" />
                </IBlueButton>
              </PaginationButtonBlock>
            </PageHeaderBlock>
            {currentConversation.getMetricsList() &&
            currentConversation.getMetricsList().length > 0 ? (
              <Table className="w-full bg-white dark:bg-gray-900">
                <TableHead
                  columns={[
                    { name: 'Name', key: 'Name' },
                    { name: 'Value', key: 'Value' },
                    { name: 'Description', key: 'Description' },
                  ]}
                />
                <TableBody>
                  {currentConversation
                    .getMetricsList()
                    .map((metadata, index) => {
                      return (
                        <TableRow key={index}>
                          <TableCell>{metadata.getName()}</TableCell>
                          <TableCell className="break-words break-all">
                            {metadata.getValue()}
                          </TableCell>
                          <TableCell className="break-words break-all">
                            {metadata.getDescription()}
                          </TableCell>
                        </TableRow>
                      );
                    })}
                </TableBody>
              </Table>
            ) : (
              <BlueNoticeBlock>
                No metadata has been captured for this conversation.
              </BlueNoticeBlock>
            )}
          </>
        );
      case 'arguments':
        return (
          <div className="flex flex-col w-full divide-x flex-1">
            <PageHeaderBlock className="border-b h-10">
              <PageTitleBlock>Arguments and Parameters</PageTitleBlock>
              <PaginationButtonBlock>
                <IBlueButton onClick={() => {}}>
                  <DownloadIcon strokeWidth={1.5} className="h-4 w-4" />
                </IBlueButton>
                <IBlueButton
                  onClick={() => {
                    get();
                  }}
                >
                  <RotateCw strokeWidth={1.5} className="h-4 w-4" />
                </IBlueButton>
              </PaginationButtonBlock>
            </PageHeaderBlock>
            <Table className="bg-white dark:bg-gray-900">
              <TableHead
                columns={[
                  { name: 'Type', key: 'Type' },
                  { name: 'Name', key: 'Name' },
                  { name: 'Value', key: 'Value' },
                ]}
              />
              <TableBody>
                {currentConversation
                  ?.getArgumentsList()
                  .map((metadata, index) => {
                    return (
                      <TableRow key={index}>
                        <TableCell>Argument</TableCell>
                        <TableCell>{metadata.getName()}</TableCell>
                        <TableCell>{metadata.getValue()}</TableCell>
                      </TableRow>
                    );
                  })}
                {currentConversation
                  ?.getOptionsList()
                  .map((metadata, index) => {
                    return (
                      <TableRow key={index}>
                        <TableCell>Option</TableCell>
                        <TableCell>{metadata.getKey()}</TableCell>
                        <TableCell>{metadata.getValue()}</TableCell>
                      </TableRow>
                    );
                  })}
                {currentConversation
                  ?.getMetadataList()
                  .map((metadata, index) => {
                    return (
                      <TableRow key={index}>
                        <TableCell>Metadata</TableCell>
                        <TableCell>{metadata.getKey()}</TableCell>
                        <TableCell className="break-words break-all">
                          {metadata.getValue()}
                        </TableCell>
                      </TableRow>
                    );
                  })}
              </TableBody>
            </Table>
          </div>
        );
    }
  };

  return (
    <div className="flex-1 flex relative grow h-full overflow-hidden">
      <aside
        className="w-80 border-r bg-white dark:bg-gray-900 z-1 overflow-auto shrink-0"
        aria-label="Sidebar"
      >
        <div className="h-full space-y-3">
          <ul className="text-sm">
            <li>
              <SideTab
                to="#"
                onClick={() => navigator.goToAssistantSessionList(assistantId!)}
                className="hover:text-red-600 hover:border-b hover:border-b-red-600 text-red-600 border-b h-10 flex items-center"
              >
                <ChevronLeft className="w-5 h-5 mr-1" strokeWidth={1.5} />
                <span className="">Back to sessions</span>
              </SideTab>
            </li>
            <li>
              <SideTab
                to="#"
                className="h-11"
                isActive={activeTab === 'messages'}
                onClick={() => setActiveTab('messages')}
              >
                <MessagesSquare className="w-4 h-4 mr-2" strokeWidth={1.5} />
                <span className="">Messages</span>
              </SideTab>
            </li>
            <li>
              <SideTab
                to="#"
                className="h-11"
                isActive={activeTab === 'context'}
                onClick={() => setActiveTab('context')}
              >
                <BookText className="w-4 h-4 mr-2" strokeWidth={1.5} />
                <span className="">Context</span>
              </SideTab>
            </li>
            <li>
              <SideTab
                to="#"
                className="h-11"
                isActive={activeTab === 'arguments'}
                onClick={() => setActiveTab('arguments')}
              >
                <Parentheses className="w-4 h-4 mr-2" strokeWidth={1.5} />
                <span className="">Arguments</span>
              </SideTab>
            </li>
            <li>
              <SideTab
                to="#"
                className="h-11"
                isActive={activeTab === 'analysis'}
                onClick={() => setActiveTab('analysis')}
              >
                <ChartArea className="w-4 h-4 mr-2" strokeWidth={1.5} />
                <span className="">Analysis</span>
              </SideTab>
            </li>
            <li>
              <SideTab
                to="#"
                className="h-11"
                isActive={activeTab === 'metrics'}
                onClick={() => setActiveTab('metrics')}
              >
                <Activity className="w-4 h-4 mr-2" strokeWidth={1.5} />
                <span className="">Metrics</span>
              </SideTab>
            </li>
          </ul>
        </div>
      </aside>
      <div className="flex-1 overflow-auto flex flex-col">
        {renderContent()}
      </div>
    </div>
  );
}

interface JsonViewerProps {
  data: string | any;
  preview?: boolean;
}

const JsonViewer = ({ data, preview = false }: JsonViewerProps) => {
  // Parse JSON string if data is a string
  const parsedData = typeof data === 'string' ? JSON.parse(data) : data;

  // Check if it's a simple result structure
  const isSimpleResult =
    parsedData &&
    typeof parsedData === 'object' &&
    parsedData.result &&
    Object.keys(parsedData).length === 1;

  // Extract key insights for business display
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
  const shouldTruncate = preview && insights.length > 3;
  const displayInsights = shouldTruncate ? insights.slice(0, 3) : insights;

  const formatValue = (value: any, type: string) => {
    if (type === 'array') {
      return Array.isArray(value) ? value.join(', ') : String(value);
    }
    if (type === 'number') {
      return typeof value === 'number'
        ? `${Math.round(value * 100)}%`
        : String(value);
    }
    return String(value);
  };

  // Simple result display
  if (isSimpleResult) {
    return (
      <div className="space-y-3">
        <div className="flex items-center gap-2">
          {/* <Badge variant="outline" className="text-xs"> */}
          Analysis Result
          {/* </Badge> */}
        </div>
        <p className="text-foreground leading-relaxed">{parsedData.result}</p>
      </div>
    );
  }

  // Complex data display
  return (
    <div className="space-y-3">
      {displayInsights.map((insight, index) => (
        <div
          key={index}
          className="border-l-2 border-primary bg-blue-600/5 pl-4 py-2"
        >
          <div className="mb-2">
            <h4 className="font-medium text-sm text-foreground capitalize leading-tight">
              {insight.key}
            </h4>
          </div>
          <p className="text-muted-foreground text-sm leading-relaxed">
            {formatValue(insight.value, insight.type)}
          </p>
        </div>
      ))}
    </div>
  );
};

export default JsonViewer;
