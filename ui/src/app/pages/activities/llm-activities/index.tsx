import React, { useState, useEffect } from 'react';
import { Helmet } from '@/app/components/app-shell/helmet';
import { useCredential } from '@/hooks/use-credential';
import toast from 'react-hot-toast/headless';
import { useRapidaStore } from '@/stores/app';
import { Metadata } from '@rapidaai/react';
import { useActivityLogPage } from '@/stores/activity/activity-log.store';
import {
  formatNanoToReadableMilli,
  toHumanReadableDateTime,
} from '@/utils/date';
import { getMetadataValue, getMetricValueOrDefault } from '@/utils/metadata';
import { LLMLogDialog } from '@/app/components/dialogs/activity';
import { HttpStatusSpanIndicator } from '@/app/components/domain/indicators/http-status';
import {
  PageHeader,
  PageHeaderStickyToolbar,
} from '@/app/components/layout/page-header';
import { CarbonStatusIndicator } from '@/app/components/ui/feedback';
import {
  IconOnlyButton,
  Pagination,
  Text,
} from '@/app/components/ui/primitives';
import { Renew, View, Launch, Ai } from '@carbon/icons-react';
import { ProviderTag } from '@/app/components/domain/provider-tag';
import { EmptyState } from '@/app/components/ui/feedback';
import { LLMLogQuerySearch } from './llm-query-search';
import {
  Table,
  TableHead,
  TableRow,
  TableHeader,
  TableBody,
  TableCell,
  TableToolbar,
  TableToolbarContent,
  Loading,
  Link,
} from '@carbon/react';

export function ListingPage() {
  const { loading, showLoader, hideLoader } = useRapidaStore();
  const [userId, token, projectId] = useCredential();
  const [currentActivityId, setCurrentActivityId] = useState('');
  const [querySearchValue, setQuerySearchValue] = useState('');
  const [showLogModal, setShowLogModal] = useState(false);

  const {
    getActivities,
    setCriterias,
    activities,
    columns,
    page,
    setPage,
    totalCount,
    criteria,
    pageSize,
    visibleColumn,
    setPageSize,
  } = useActivityLogPage();

  useEffect(() => {
    showLoader();
    onGetAcitvities();
  }, [projectId, page, pageSize, JSON.stringify(criteria)]);

  const onGetAcitvities = () => {
    getActivities(
      projectId,
      token,
      userId,
      err => {
        hideLoader();
        toast.error(err);
      },
      logs => {
        hideLoader();
      },
    );
  };

  const visibleColumns = columns.filter(c => c.visible);

  return (
    <>
      {currentActivityId && (
        <LLMLogDialog
          modalOpen={showLogModal}
          setModalOpen={setShowLogModal}
          currentActivityId={currentActivityId}
        />
      )}

      <div
        data-page-scroll
        className="flex min-h-0 flex-1 flex-col overflow-y-auto overflow-x-hidden"
      >
        <Helmet title="LLM Logs" />
        <PageHeader
          eyebrow={
            <Text variant="label-01" className="text-primary">
              Observability
            </Text>
          }
          title="LLM Logs"
          description="Inspect language model requests, responses, latency, usage, and failures across your workspace."
          contextualActions={
            <Text variant="label-01" className="text-muted tabular-nums">
              {activities.length} of {totalCount}
            </Text>
          }
        />

        <PageHeaderStickyToolbar>
          <TableToolbar>
            <TableToolbarContent>
              <LLMLogQuerySearch
                value={querySearchValue}
                onChange={setQuerySearchValue}
                onApply={setCriterias}
              />
              <IconOnlyButton
                kind="ghost"
                size="lg"
                renderIcon={Renew}
                iconDescription="Refresh"
                onClick={() => onGetAcitvities()}
              />
            </TableToolbarContent>
          </TableToolbar>
        </PageHeaderStickyToolbar>

        {loading ? (
          <div className="flex items-center justify-center py-16">
            <Loading withOverlay={false} small />
          </div>
        ) : activities.length > 0 ? (
          <div className="shrink-0 overflow-x-auto">
            <Table className="min-w-max">
              <TableHead>
                <TableRow>
                  {visibleColumns.map(col => (
                    <TableHeader key={col.key}>{col.name}</TableHeader>
                  ))}
                </TableRow>
              </TableHead>
              <TableBody>
                {activities.map((at, idx) => (
                  <TableRow key={idx}>
                    {visibleColumn('Source') && (
                      <TableCell>
                        <ActivitySource
                          metadatas={at.getExternalauditmetadatasList()}
                        />
                      </TableCell>
                    )}
                    {visibleColumn('Provider Name') && (
                      <TableCell>
                        <ProviderTag
                          provider={getMetadataValue(
                            at.getExternalauditmetadatasList(),
                            'provider_name',
                          )}
                        />
                      </TableCell>
                    )}
                    {visibleColumn('Model Name') && (
                      <TableCell>
                        {getMetadataValue(
                          at.getExternalauditmetadatasList(),
                          'model_name',
                        )}
                      </TableCell>
                    )}
                    {visibleColumn('Created Date') && (
                      <TableCell className="text-[13px] whitespace-nowrap">
                        {at.getCreateddate() &&
                          toHumanReadableDateTime(at.getCreateddate()!)}
                      </TableCell>
                    )}
                    {visibleColumn('Action') && (
                      <TableCell>
                        <div className="flex items-center gap-0">
                          <IconOnlyButton
                            kind="ghost"
                            size="md"
                            renderIcon={View}
                            iconDescription="View detail"
                            onClick={() => {
                              setCurrentActivityId(at.getId());
                              setShowLogModal(true);
                            }}
                          />
                          <IconOnlyButton
                            kind="ghost"
                            size="md"
                            renderIcon={Launch}
                            iconDescription="View conversation"
                            onClick={() => {
                              const link = getActivityLink(
                                at.getExternalauditmetadatasList(),
                              ).link;
                              if (link) window.location.href = link;
                            }}
                          />
                        </div>
                      </TableCell>
                    )}
                    {visibleColumn('Status') && (
                      <TableCell>
                        <CarbonStatusIndicator state={at.getStatus()} />
                      </TableCell>
                    )}
                    {visibleColumn('TTFT') && (
                      <TableCell className="font-mono text-[13px]">
                        {formatNanoToReadableMilli(
                          getMetricValueOrDefault(
                            at.getMetricsList(),
                            'time_to_first_token',
                            '0',
                          ),
                        )}
                      </TableCell>
                    )}
                    {visibleColumn('TRT') && (
                      <TableCell className="font-mono text-[13px]">
                        {formatNanoToReadableMilli(at.getTimetaken())}
                      </TableCell>
                    )}

                    {visibleColumn('Http_status') && (
                      <TableCell>
                        <HttpStatusSpanIndicator
                          status={at.getResponsestatus()}
                        />
                      </TableCell>
                    )}
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        ) : (
          <EmptyState
            icon={Ai}
            title="No LLM activities found"
            subtitle="Requests made to LLM providers like OpenAI, Anthropic, and Google will appear here as your assistants process conversations."
          />
        )}

        {activities.length > 0 && (
          <Pagination
            className="mt-auto shrink-0"
            totalItems={totalCount}
            page={page}
            pageSize={pageSize}
            pageSizes={[10, 20, 25, 50, 100]}
            onChange={({ page: p, pageSize: ps }) => {
              if (ps !== pageSize) setPageSize(ps);
              else setPage(p);
            }}
          />
        )}
      </div>
    </>
  );
}

function ActivitySource(props: { metadatas: Metadata[] }) {
  const { source, link } = getActivityLink(props.metadatas);
  return link ? (
    <Link href={link} className="!text-sm !inline-flex !items-center !gap-1">
      <span>{source}</span>
      <Launch size={12} />
    </Link>
  ) : (
    <span className="text-sm">{source}</span>
  );
}

function getActivityLink(metadatas: Metadata[]): {
  source: string;
  link: string;
} {
  const endpoint = getMetadataValue(metadatas, 'endpoint_id');
  if (endpoint)
    return { source: endpoint, link: `/deployment/endpoint/${endpoint}` };

  const assistant = getMetadataValue(metadatas, 'assistant_id');
  if (assistant)
    return { source: assistant, link: `/deployment/assistant/${assistant}` };

  const knowledge = getMetadataValue(metadatas, 'knowledge_id');
  if (knowledge) return { source: knowledge, link: `/knowledge/${knowledge}` };

  const source = getMetadataValue(metadatas, 'source');
  if (source) return { source, link: '' };

  return { source: '', link: '' };
}
