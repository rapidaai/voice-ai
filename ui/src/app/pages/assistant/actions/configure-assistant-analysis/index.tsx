import React, { FC, useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { toHumanReadableDateTime } from '@/utils/date';
import { ChartLine } from '@carbon/icons-react';
import { useCurrentCredential } from '@/hooks/use-credential';
import { useRapidaStore } from '@/stores/app';
import { SectionLoader } from '@/app/components/ui/feedback';
import toast from 'react-hot-toast/headless';
import { EmptyState } from '@/app/components/ui/feedback';
import { CreateAssistantAnalysis } from '@/app/pages/assistant/actions/configure-assistant-analysis/create-assistant-analysis';
import { useAssistantAnalysisPageStore } from '@/stores/assistant/actions';
import { UpdateAssistantAnalysis } from '@/app/pages/assistant/actions/configure-assistant-analysis/update-assistant-analysis';
import { RecordStatusIndicator } from '@/app/components/ui/feedback';
import { Pagination } from '@/app/components/ui/primitives';
import {
  Button,
  Table,
  TableHead,
  TableRow,
  TableHeader,
  TableBody,
  TableCell,
  ComposedModal,
  ModalBody,
  ModalFooter,
  ModalHeader,
  OverflowMenu,
  OverflowMenuItem,
} from '@carbon/react';
import { TableSection } from '@/app/components/layout/sections/table-section';
import { AssistantSettingsShellHeader } from '@/app/pages/assistant/components/settings-shell-header';

const getAnalysisOptionMap = (row: any): Map<string, string> => {
  const map = new Map<string, string>();
  const options = row?.getOptionsList?.() || [];
  options.forEach((option: any) => {
    const key = option?.getKey?.();
    const value = option?.getValue?.();
    if (key && typeof value === 'string') {
      map.set(key, value);
    }
  });
  return map;
};

const getAnalysisEndpointId = (row: any): string =>
  getAnalysisOptionMap(row).get('endpoint_id') || '';

const getAnalysisEndpointVersion = (row: any): string =>
  getAnalysisOptionMap(row).get('endpoint_version') || 'latest';

const getAnalysisName = (row: any): string =>
  getAnalysisOptionMap(row).get('name') || '';

const getAnalysisPriority = (row: any): number => {
  const value = Number(getAnalysisOptionMap(row).get('execution_priority'));
  return Number.isFinite(value) ? value : 0;
};

type AnalysisAction = {
  kind: 'enable' | 'disable' | 'delete';
  analysis: any;
};

export function ConfigureAssistantAnalysisPage() {
  const { assistantId } = useParams();
  return (
    <>
      {assistantId && <ConfigureAssistantAnalysis assistantId={assistantId} />}
    </>
  );
}

export function CreateAssistantAnalysisPage() {
  const { assistantId } = useParams();
  return (
    <>{assistantId && <CreateAssistantAnalysis assistantId={assistantId} />}</>
  );
}

export function UpdateAssistantAnalysisPage() {
  const { assistantId } = useParams();
  return (
    <>{assistantId && <UpdateAssistantAnalysis assistantId={assistantId} />}</>
  );
}

const ConfigureAssistantAnalysis: FC<{ assistantId: string }> = ({
  assistantId,
}) => {
  const navigation = useGlobalNavigation();
  const axtion = useAssistantAnalysisPageStore();
  const { authId, token, projectId } = useCurrentCredential();
  const { loading, showLoader, hideLoader } = useRapidaStore();
  const [pendingAnalysisAction, setPendingAnalysisAction] =
    useState<AnalysisAction | null>(null);

  const get = () => {
    showLoader('block');
    axtion.getAssistantAnalysis(
      assistantId,
      projectId,
      token,
      authId,
      e => {
        toast.error(e);
        hideLoader();
      },
      () => {
        hideLoader();
      },
    );
  };

  useEffect(() => {
    get();
  }, [assistantId, projectId, token, authId, axtion.page, axtion.pageSize]);

  const deleteAssistantAnalysis = (assistantId: string, analysisId: string) => {
    showLoader('block');
    axtion.deleteAssistantAnalysis(
      assistantId,
      analysisId,
      projectId,
      token,
      authId,
      e => {
        toast.error(e);
        hideLoader();
      },
      () => {
        toast.success('Analysis deleted successfully');
        setPendingAnalysisAction(null);
        get();
      },
    );
  };

  const updateAnalysisEnabled = (analysis: any, enabled: boolean) => {
    showLoader('block');
    axtion.updateAssistantAnalysisEnabled(
      assistantId,
      analysis,
      enabled,
      projectId,
      token,
      authId,
      e => {
        toast.error(e);
        hideLoader();
      },
      () => {
        toast.success(
          `Analysis ${enabled ? 'enabled' : 'disabled'} successfully`,
        );
        setPendingAnalysisAction(null);
        get();
      },
    );
  };

  const modalTitle =
    pendingAnalysisAction?.kind === 'enable'
      ? 'Enable analysis?'
      : pendingAnalysisAction?.kind === 'disable'
        ? 'Disable analysis?'
        : 'Delete analysis?';
  const modalContent =
    pendingAnalysisAction?.kind === 'enable'
      ? 'This analysis will run when assistant finalization triggers it.'
      : pendingAnalysisAction?.kind === 'disable'
        ? 'This analysis will stop running until it is enabled again.'
        : 'This analysis configuration will be removed from the assistant.';
  const modalPrimaryLabel =
    pendingAnalysisAction?.kind === 'enable'
      ? 'Enable'
      : pendingAnalysisAction?.kind === 'disable'
        ? 'Disable'
        : 'Delete';

  if (loading) {
    return (
      <div className="h-full w-full flex flex-col items-center justify-center">
        <SectionLoader />
      </div>
    );
  }

  return (
    <div className="flex h-full flex-1 flex-col bg-surface">
      <ComposedModal
        open={Boolean(pendingAnalysisAction)}
        onClose={() => setPendingAnalysisAction(null)}
        size="sm"
        danger={pendingAnalysisAction?.kind !== 'enable'}
      >
        <ModalHeader title={modalTitle} />
        <ModalBody>
          <p>{modalContent}</p>
        </ModalBody>
        <ModalFooter danger={pendingAnalysisAction?.kind !== 'enable'}>
          <Button
            kind="secondary"
            size="md"
            onClick={() => setPendingAnalysisAction(null)}
          >
            Cancel
          </Button>
          <Button
            kind={
              pendingAnalysisAction?.kind === 'enable' ? 'primary' : 'danger'
            }
            size="md"
            onClick={() => {
              if (!pendingAnalysisAction) return;
              if (pendingAnalysisAction.kind === 'delete') {
                deleteAssistantAnalysis(
                  assistantId,
                  pendingAnalysisAction.analysis.getId(),
                );
                return;
              }
              updateAnalysisEnabled(
                pendingAnalysisAction.analysis,
                pendingAnalysisAction.kind === 'enable',
              );
            }}
          >
            {modalPrimaryLabel}
          </Button>
        </ModalFooter>
      </ComposedModal>
      <AssistantSettingsShellHeader
        title="Analysis"
        href={`/deployment/assistant/${assistantId}/configure-analysis`}
        onReload={get}
        reloadLabel="Reload analysis"
        primaryAction={{
          label: 'Create analysis',
          onClick: () => navigation.goToCreateAssistantAnalysis(assistantId),
        }}
      />
      <TableSection>
        {axtion.analysises.length > 0 ? (
          <>
            <Table>
              <TableHead>
                <TableRow>
                  <TableHeader>Name</TableHeader>
                  <TableHeader>Endpoint</TableHeader>
                  <TableHeader>Version</TableHeader>
                  <TableHeader>Priority</TableHeader>
                  <TableHeader>Status</TableHeader>
                  <TableHeader>Date</TableHeader>
                  <TableHeader>Action</TableHeader>
                </TableRow>
              </TableHead>
              <TableBody>
                {axtion.analysises.map(row => {
                  return (
                    <TableRow key={row.getId()}>
                      <TableCell className="text-sm">
                        {getAnalysisName(row)}
                      </TableCell>
                      <TableCell className="text-sm">
                        <span className="font-mono text-[13px]">
                          {getAnalysisEndpointId(row) || '—'}
                        </span>
                      </TableCell>
                      <TableCell className="text-sm">
                        {getAnalysisEndpointVersion(row)}
                      </TableCell>
                      <TableCell className="text-sm">
                        {getAnalysisPriority(row)}
                      </TableCell>
                      <TableCell className="text-sm whitespace-nowrap">
                        <RecordStatusIndicator
                          state={row.getStatus()}
                          textSize={14}
                        />
                      </TableCell>
                      <TableCell className="text-[13px] whitespace-nowrap">
                        {row.getCreateddate()
                          ? toHumanReadableDateTime(row.getCreateddate()!)
                          : '—'}
                      </TableCell>
                      <TableCell
                        className="text-sm"
                        onClick={e => e.stopPropagation()}
                      >
                        <OverflowMenu
                          size="sm"
                          flipped
                          aria-label="Analysis actions"
                        >
                          <OverflowMenuItem
                            itemText="Edit"
                            onClick={() =>
                              navigation.goToEditAssistantAnalysis(
                                assistantId,
                                row.getId(),
                              )
                            }
                          />
                          <OverflowMenuItem
                            itemText={row.getEnabled() ? 'Disable' : 'Enable'}
                            onClick={() =>
                              setPendingAnalysisAction({
                                kind: row.getEnabled() ? 'disable' : 'enable',
                                analysis: row,
                              })
                            }
                          />
                          <OverflowMenuItem
                            itemText="Delete"
                            isDelete
                            onClick={() =>
                              setPendingAnalysisAction({
                                kind: 'delete',
                                analysis: row,
                              })
                            }
                          />
                        </OverflowMenu>
                      </TableCell>
                    </TableRow>
                  );
                })}
              </TableBody>
            </Table>
            <Pagination
              totalItems={axtion.totalCount}
              page={axtion.page}
              pageSize={axtion.pageSize}
              pageSizes={[10, 20, 50]}
              onChange={({ page, pageSize }) => {
                if (pageSize !== axtion.pageSize) {
                  axtion.setPageSize(pageSize);
                } else {
                  axtion.setPage(page);
                }
              }}
            />
          </>
        ) : (
          <EmptyState
            icon={ChartLine}
            title="No analysis"
            subtitle="Any analysis you add will be listed here."
          />
        )}
      </TableSection>
    </div>
  );
};
