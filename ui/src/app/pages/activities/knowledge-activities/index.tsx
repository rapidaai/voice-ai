import { useState, useEffect } from 'react';
import { Helmet } from '@/app/components/app-shell/helmet';
import { DateFilter } from '@/app/components/ui/composites';
import { useCredential } from '@/hooks/use-credential';
import toast from 'react-hot-toast/headless';
import { useRapidaStore } from '@/stores/app';
import {
  formatNanoToReadableMilli,
  toDateString,
  toHumanReadableDateTime,
} from '@/utils/date';
import {
  PageHeader,
  PageHeaderStickyToolbar,
} from '@/app/components/layout/page-header';
import { useKnowledgeActivityLogPage } from '@/stores/knowledge/knowledge-activity-log.store';
import { KnowledgeLogDialog } from '@/app/components/dialogs/activity';
import { CarbonStatusIndicator } from '@/app/components/ui/feedback';
import {
  IconOnlyButton,
  Pagination,
  Text,
} from '@/app/components/ui/primitives';
import { Renew, View, DataBase } from '@carbon/icons-react';
import { EmptyState } from '@/app/components/ui/feedback';

import {
  Table,
  TableHead,
  TableRow,
  TableHeader,
  TableBody,
  TableCell,
  TableToolbar,
  TableToolbarContent,
  TableToolbarSearch,
  Loading,
} from '@carbon/react';
import { TableLink } from '@/app/components/ui/table';

export function ListingPage() {
  const { loading, showLoader, hideLoader } = useRapidaStore();
  const [userId, token, projectId] = useCredential();
  const [currentActivityId, setCurrentActivityId] = useState('');
  const [showLogModal, setShowLogModal] = useState(false);

  const {
    getActivities,
    addCriterias,
    activities,
    columns,
    page,
    setPage,
    totalCount,
    criteria,
    pageSize,
    visibleColumn,
    setPageSize,
    setColumns,
  } = useKnowledgeActivityLogPage();

  const onDateSelect = (to: Date, from: Date) => {
    addCriterias([
      { k: 'created_date', v: toDateString(to), logic: '<=' },
      { k: 'created_date', v: toDateString(from), logic: '>=' },
    ]);
  };

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
        <KnowledgeLogDialog
          modalOpen={showLogModal}
          setModalOpen={setShowLogModal}
          currentActivityId={currentActivityId}
        />
      )}

      <div
        data-page-scroll
        className="flex min-h-0 flex-1 flex-col overflow-y-auto overflow-x-hidden"
      >
        <Helmet title="Knowledge Logs" />
        <PageHeader
          eyebrow={
            <Text variant="label-01" className="text-primary">
              Observability
            </Text>
          }
          title="Knowledge Logs"
          description="Inspect knowledge retrieval activity, sources, latency, and results across assistant sessions."
          contextualActions={
            <Text variant="label-01" className="text-muted tabular-nums">
              {activities.length} of {totalCount}
            </Text>
          }
        />

        <PageHeaderStickyToolbar>
          <TableToolbar>
            <TableToolbarContent>
              <TableToolbarSearch placeholder="Search knowledge logs" />
              <DateFilter
                onApply={(from, to) => onDateSelect(to, from)}
                onReset={() => addCriterias([])}
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
            <Table>
              <TableHead>
                <TableRow>
                  {visibleColumns.map(col => (
                    <TableHeader key={col.key}>{col.name}</TableHeader>
                  ))}
                  <TableHeader>Actions</TableHeader>
                </TableRow>
              </TableHead>
              <TableBody>
                {activities.map((at, idx) => (
                  <TableRow key={idx}>
                    {visibleColumn('knowledge_id') && (
                      <TableCell>
                        <TableLink href={`/knowledge/${at.getKnowledgeid()}`}>
                          {at.getKnowledgeid()}
                        </TableLink>
                      </TableCell>
                    )}
                    {visibleColumn('retrieval_method') && (
                      <TableCell>{at.getRetrievalmethod()}</TableCell>
                    )}
                    {visibleColumn('top_k') && (
                      <TableCell>{at.getTopk()}</TableCell>
                    )}
                    {visibleColumn('score_threshold') && (
                      <TableCell>{at.getScorethreshold()}</TableCell>
                    )}
                    {visibleColumn('document_count') && (
                      <TableCell>{at.getDocumentcount()}</TableCell>
                    )}
                    {visibleColumn('time_taken') && (
                      <TableCell className="!font-mono !text-xs">
                        {formatNanoToReadableMilli(at.getTimetaken())}
                      </TableCell>
                    )}
                    {visibleColumn('status') && (
                      <TableCell>
                        <CarbonStatusIndicator state={at.getStatus()} />
                      </TableCell>
                    )}
                    {visibleColumn('created_date') && (
                      <TableCell className="!text-xs whitespace-nowrap">
                        {at.getCreateddate() &&
                          toHumanReadableDateTime(at.getCreateddate()!)}
                      </TableCell>
                    )}
                    <TableCell>
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
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        ) : (
          <EmptyState
            icon={DataBase}
            title="No knowledge activities found"
            subtitle="Knowledge base retrieval activities will appear here once your assistants query connected knowledge sources."
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
