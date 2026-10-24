import React, { FC, useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { toHumanReadableDateTime } from '@/utils/date';
import { Activity, Edit, TrashCan } from '@carbon/icons-react';
import { useCurrentCredential } from '@/hooks/use-credential';
import { SectionLoader } from '@/app/components/ui/feedback';
import toast from 'react-hot-toast/headless';
import { EmptyState } from '@/app/components/ui/feedback';
import { CreateAssistantTelemetry } from './create-assistant-telemetry';
import { UpdateAssistantTelemetry } from './update-assistant-telemetry';
import { useAssistantTelemetryPageStore } from '@/stores/assistant/actions';
import { TELEMETRY_PROVIDER } from '@/providers';
import { CarbonShapeIndicator } from '@/app/components/ui/feedback';
import {
  Button,
  ComposedModal,
  ModalBody,
  ModalFooter,
  ModalHeader,
  Table,
  TableHead,
  TableRow,
  TableHeader,
  TableBody,
  TableCell,
  OverflowMenu,
  OverflowMenuItem,
  Tag,
} from '@carbon/react';
import { AssistantConfiguration, Metadata } from '@rapidaai/react';
import { Pagination } from '@/app/components/ui/primitives';
import {
  ScrollableTableSection,
  TableSection,
} from '@/app/components/layout/sections/table-section';
import { AssistantSettingsShellHeader } from '@/app/pages/assistant/components/settings-shell-header';

export function ConfigureAssistantTelemetryPage() {
  const { assistantId } = useParams();
  return (
    <>
      {assistantId && <ConfigureAssistantTelemetry assistantId={assistantId} />}
    </>
  );
}

export function CreateAssistantTelemetryPage() {
  const { assistantId } = useParams();
  return (
    <>{assistantId && <CreateAssistantTelemetry assistantId={assistantId} />}</>
  );
}

export function UpdateAssistantTelemetryPage() {
  const { assistantId } = useParams();
  return (
    <>{assistantId && <UpdateAssistantTelemetry assistantId={assistantId} />}</>
  );
}

const providerNameByCode = new Map(
  TELEMETRY_PROVIDER.map(p => [p.code, p.name]),
);

const getOptionValue = (options: Metadata[], key: string) =>
  options.find(option => option.getKey() === key)?.getValue() || '';

const getTelemetryTarget = (telemetry: AssistantConfiguration) => {
  return getOptionValue(telemetry.getOptionsList(), 'endpoint') || '-';
};

type TelemetryAction = {
  kind: 'enable' | 'disable' | 'delete';
  telemetry: AssistantConfiguration;
};

const ConfigureAssistantTelemetry: FC<{ assistantId: string }> = ({
  assistantId,
}) => {
  const navigation = useGlobalNavigation();
  const action = useAssistantTelemetryPageStore();
  const { authId, token, projectId } = useCurrentCredential();
  const [loading, setLoading] = useState(true);
  const [pendingTelemetryAction, setPendingTelemetryAction] =
    useState<TelemetryAction | null>(null);

  useEffect(() => {
    get();
  }, [assistantId, projectId, token, authId, action.page, action.pageSize]);

  const get = () => {
    setLoading(true);
    action.getAssistantTelemetry(
      assistantId,
      projectId,
      token,
      authId,
      e => {
        toast.error(e);
        setLoading(false);
      },
      () => {
        setLoading(false);
      },
    );
  };

  const deleteTelemetry = (telemetryId: string) => {
    setLoading(true);
    action.deleteAssistantTelemetry(
      assistantId,
      telemetryId,
      projectId,
      token,
      authId,
      e => {
        toast.error(e);
        setLoading(false);
      },
      () => {
        toast.success('Telemetry provider deleted successfully');
        setPendingTelemetryAction(null);
        get();
      },
    );
  };

  const updateTelemetryEnabled = (
    telemetry: AssistantConfiguration,
    enabled: boolean,
  ) => {
    setLoading(true);
    action.updateAssistantTelemetryEnabled(
      assistantId,
      telemetry,
      enabled,
      projectId,
      token,
      authId,
      e => {
        toast.error(e);
        setLoading(false);
      },
      () => {
        toast.success(
          `Telemetry ${enabled ? 'enabled' : 'disabled'} successfully`,
        );
        setPendingTelemetryAction(null);
        get();
      },
    );
  };

  const modalTitle =
    pendingTelemetryAction?.kind === 'enable'
      ? 'Enable telemetry?'
      : pendingTelemetryAction?.kind === 'disable'
        ? 'Disable telemetry?'
        : 'Delete telemetry?';
  const modalContent =
    pendingTelemetryAction?.kind === 'enable'
      ? 'Telemetry will start being pushed to this provider.'
      : pendingTelemetryAction?.kind === 'disable'
        ? 'Telemetry will stop being pushed to this provider until it is enabled again.'
        : 'This telemetry provider will be removed from the assistant.';
  const modalPrimaryLabel =
    pendingTelemetryAction?.kind === 'enable'
      ? 'Enable'
      : pendingTelemetryAction?.kind === 'disable'
        ? 'Disable'
        : 'Delete';

  return (
    <div className="flex h-full flex-1 flex-col bg-surface">
      <ComposedModal
        open={Boolean(pendingTelemetryAction)}
        onClose={() => setPendingTelemetryAction(null)}
        size="sm"
        danger={pendingTelemetryAction?.kind !== 'enable'}
      >
        <ModalHeader title={modalTitle} />
        <ModalBody>
          <p>{modalContent}</p>
        </ModalBody>
        <ModalFooter danger={pendingTelemetryAction?.kind !== 'enable'}>
          <Button
            kind="secondary"
            size="md"
            onClick={() => setPendingTelemetryAction(null)}
          >
            Cancel
          </Button>
          <Button
            kind={
              pendingTelemetryAction?.kind === 'enable' ? 'primary' : 'danger'
            }
            size="md"
            onClick={() => {
              if (!pendingTelemetryAction) return;
              if (pendingTelemetryAction.kind === 'delete') {
                deleteTelemetry(pendingTelemetryAction.telemetry.getId());
                return;
              }
              updateTelemetryEnabled(
                pendingTelemetryAction.telemetry,
                pendingTelemetryAction.kind === 'enable',
              );
            }}
          >
            {modalPrimaryLabel}
          </Button>
        </ModalFooter>
      </ComposedModal>

      <AssistantSettingsShellHeader
        title="Telemetry"
        href={`/deployment/assistant/${assistantId}/configure-telemetry`}
        onReload={get}
        reloadLabel="Reload telemetry"
        primaryAction={{
          label: 'Add telemetry',
          onClick: () => navigation.goToCreateAssistantTelemetry(assistantId),
        }}
      />

      <TableSection>
        {loading ? (
          <div className="flex flex-col flex-1 items-center justify-center">
            <SectionLoader />
          </div>
        ) : action.telemetries.length > 0 ? (
          <>
            <ScrollableTableSection>
              <Table className="min-w-max">
                <TableHead>
                  <TableRow>
                    <TableHeader>Provider</TableHeader>
                    <TableHeader>Type</TableHeader>
                    <TableHeader>Target</TableHeader>
                    <TableHeader>Status</TableHeader>
                    <TableHeader>Created</TableHeader>
                    <TableHeader>Actions</TableHeader>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {action.telemetries.map(row => {
                    const provider = row.getProvider();
                    const providerName =
                      providerNameByCode.get(provider) || provider;
                    return (
                      <TableRow key={row.getId()}>
                        <TableCell className="text-sm">
                          {providerName}
                        </TableCell>
                        <TableCell className="text-sm">
                          <Tag type="blue" size="sm">
                            {row.getConfigurationtype() || 'telemetry'}
                          </Tag>
                        </TableCell>
                        <TableCell className="text-sm">
                          {getTelemetryTarget(row)}
                        </TableCell>
                        <TableCell className="text-sm whitespace-nowrap">
                          <CarbonShapeIndicator
                            kind={row.getEnabled() ? 'stable' : 'draft'}
                            label={row.getEnabled() ? 'Enabled' : 'Disabled'}
                            textSize={14}
                          />
                        </TableCell>
                        <TableCell className="text-[13px] whitespace-nowrap">
                          {row.getCreateddate()
                            ? toHumanReadableDateTime(row.getCreateddate()!)
                            : '-'}
                        </TableCell>
                        <TableCell
                          className="text-sm"
                          onClick={e => e.stopPropagation()}
                        >
                          <OverflowMenu
                            size="sm"
                            flipped
                            aria-label="Telemetry actions"
                          >
                            <OverflowMenuItem
                              itemText="Edit"
                              onClick={() =>
                                navigation.goToEditAssistantTelemetry(
                                  assistantId,
                                  row.getId(),
                                )
                              }
                            />
                            <OverflowMenuItem
                              itemText={row.getEnabled() ? 'Disable' : 'Enable'}
                              onClick={() =>
                                setPendingTelemetryAction({
                                  kind: row.getEnabled() ? 'disable' : 'enable',
                                  telemetry: row,
                                })
                              }
                            />
                            <OverflowMenuItem
                              itemText="Delete"
                              isDelete
                              onClick={() =>
                                setPendingTelemetryAction({
                                  kind: 'delete',
                                  telemetry: row,
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
            </ScrollableTableSection>
            <Pagination
              totalItems={action.totalCount}
              page={action.page}
              pageSize={action.pageSize}
              pageSizes={[10, 20, 50]}
              onChange={({ page: newPage, pageSize: newSize }) => {
                if (newSize !== action.pageSize) {
                  action.setPageSize(newSize);
                  return;
                }
                action.setPage(newPage);
              }}
            />
          </>
        ) : (
          <EmptyState
            className="w-full"
            icon={Activity}
            title="No telemetry providers"
            subtitle="Any telemetry providers you add will be listed here."
          />
        )}
      </TableSection>
    </div>
  );
};
