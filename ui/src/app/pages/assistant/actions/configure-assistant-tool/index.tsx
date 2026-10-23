import { useRapidaStore } from '@/stores/app';
import { FC, useEffect, useState } from 'react';
import toast from 'react-hot-toast/headless';
import { useParams } from 'react-router-dom';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { EmptyState } from '@/app/components/ui/feedback';
import { Edit, TrashCan, ToolKit } from '@carbon/icons-react';
import { CreateTool } from '@/app/pages/assistant/actions/configure-assistant-tool/create-assistant-tool';
import { SectionLoader } from '@/app/components/ui/feedback';
import { useAssistantToolPageStore } from '@/stores/assistant/actions';
import { useCurrentCredential } from '@/hooks/use-credential';
import { UpdateTool } from '@/app/pages/assistant/actions/configure-assistant-tool/update-assistant-tool';
import { useConfirmDialog } from '@/app/pages/assistant/actions/hooks/use-confirmation';
import { BUILDIN_TOOLS } from '@/llm-tools';
import {
  getToolConditionSource,
  getToolConditionSourceLabel,
} from '@/app/components/domain/tools/common';
import {
  Button,
  Table,
  TableHead,
  TableRow,
  TableHeader,
  TableBody,
  TableCell,
  TableToolbar,
  TableBatchActions,
  TableBatchAction,
  RadioButton,
  Tag,
} from '@carbon/react';
import { toHumanReadableDateTime } from '@/utils/date';
import { AssistantSettingsShellHeader } from '@/app/pages/assistant/components/settings-shell-header';

export function ConfigureAssistantToolPage() {
  const { assistantId } = useParams();
  return (
    <>{assistantId && <ConfigureAssistantTool assistantId={assistantId} />}</>
  );
}

export function CreateAssistantToolPage() {
  const { assistantId } = useParams();
  return <>{assistantId && <CreateTool assistantId={assistantId} />}</>;
}

export function UpdateAssistantToolPage() {
  const { assistantId } = useParams();
  return <>{assistantId && <UpdateTool assistantId={assistantId} />}</>;
}

const ConfigureAssistantTool: FC<{ assistantId: string }> = ({
  assistantId,
}) => {
  const navigator = useGlobalNavigation();
  const navigation = useGlobalNavigation();
  const axtion = useAssistantToolPageStore();
  const { authId, token, projectId } = useCurrentCredential();
  const { loading, showLoader, hideLoader } = useRapidaStore();
  const [selectedToolId, setSelectedToolId] = useState<string | null>(null);

  useEffect(() => {
    showLoader('block');
    get();
  }, []);

  const get = () => {
    axtion.getAssistantTool(
      assistantId,
      projectId,
      token,
      authId,
      e => {
        toast.error(e);
        hideLoader();
      },
      v => {
        hideLoader();
      },
    );
  };

  const { showDialog, ConfirmDialogComponent } = useConfirmDialog({
    title: 'Are you sure?',
    content: 'You want to delete? The tool will be removed from assistant.',
  });

  const deleteAssistantTool = (
    assistantId: string,
    assistantToolId: string,
  ) => {
    showLoader('block');
    axtion.deleteAssistantTool(
      assistantId,
      assistantToolId,
      projectId,
      token,
      authId,
      e => {
        toast.error(e);
        hideLoader();
      },
      v => {
        toast.success('Assistant tool deleted successfully');
        get();
      },
    );
  };

  const refreshTools = () => {
    setSelectedToolId(null);
    showLoader('block');
    get();
  };

  if (loading) {
    return (
      <div className="h-full w-full flex flex-col items-center justify-center">
        <SectionLoader />
      </div>
    );
  }

  return (
    <div className="relative flex flex-col flex-1">
      <ConfirmDialogComponent />

      <AssistantSettingsShellHeader
        title="Tools & MCP"
        href={`/deployment/assistant/${assistantId}/configure-tool`}
        onReload={refreshTools}
        reloadLabel="Reload tools"
        primaryAction={{
          label: 'Add tool',
          onClick: () => navigator.goToCreateAssistantTool(assistantId),
        }}
      />

      {selectedToolId && (
        <TableToolbar>
          <TableBatchActions
            shouldShowBatchActions
            totalSelected={1}
            totalCount={axtion.tools.length}
            onCancel={() => setSelectedToolId(null)}
          >
            <TableBatchAction
              renderIcon={Edit}
              onClick={() => {
                navigation.goToEditAssistantTool(assistantId, selectedToolId);
                setSelectedToolId(null);
              }}
            >
              Edit tool
            </TableBatchAction>
            <TableBatchAction
              renderIcon={TrashCan}
              onClick={() => {
                showDialog(() => {
                  deleteAssistantTool(assistantId, selectedToolId);
                });
                setSelectedToolId(null);
              }}
            >
              Delete tool
            </TableBatchAction>
          </TableBatchActions>
        </TableToolbar>
      )}

      <div className="overflow-auto flex flex-col flex-1">
        {axtion.tools.length > 0 ? (
          <Table>
            <TableHead>
              <TableRow>
                <TableHeader className="!w-12" />
                <TableHeader>Name</TableHeader>
                <TableHeader>Type</TableHeader>
                <TableHeader>Description</TableHeader>
                <TableHeader>Date</TableHeader>
                <TableHeader>Actions</TableHeader>
              </TableRow>
            </TableHead>
            <TableBody>
              {axtion.tools.map((itm, idx) => {
                const method = itm.getExecutionmethod();
                const methodMeta = BUILDIN_TOOLS.find(x => x.code === method);
                const isMcp = method === 'mcp';
                const selected = selectedToolId === itm.getId();
                const conditionSource = getToolConditionSource(
                  itm.getExecutionoptionsList(),
                );

                return (
                  <TableRow
                    key={`tool-row-${idx}`}
                    isSelected={selected}
                    onClick={() =>
                      setSelectedToolId(selected ? null : itm.getId())
                    }
                    className="cursor-pointer"
                  >
                    <TableCell
                      className="!w-12 !pr-0"
                      onClick={e => e.stopPropagation()}
                    >
                      <RadioButton
                        id={`tool-select-${itm.getId()}`}
                        name="tool-select"
                        labelText=""
                        hideLabel
                        checked={selected}
                        onChange={() =>
                          setSelectedToolId(selected ? null : itm.getId())
                        }
                      />
                    </TableCell>
                    <TableCell className="text-sm">{itm.getName()}</TableCell>
                    <TableCell className="text-sm">
                      <div className="flex items-center gap-1">
                        {methodMeta && (
                          <Tag size="sm" type="gray">
                            {methodMeta.name}
                          </Tag>
                        )}
                        {isMcp && (
                          <Tag size="sm" type="purple">
                            MCP
                          </Tag>
                        )}
                        {!methodMeta && !isMcp && (
                          <Tag size="sm" type="gray">
                            {(method || 'Unknown').replace(/_/g, ' ')}
                          </Tag>
                        )}
                        {conditionSource !== 'all' && (
                          <Tag size="sm" type="blue">
                            Source:{' '}
                            {getToolConditionSourceLabel(conditionSource)}
                          </Tag>
                        )}
                      </div>
                    </TableCell>
                    <TableCell className="max-w-[360px] truncate text-sm">
                      {itm.getDescription()}
                    </TableCell>
                    <TableCell className="text-[13px] whitespace-nowrap">
                      {itm.getCreateddate()
                        ? toHumanReadableDateTime(itm.getCreateddate()!)
                        : '—'}
                    </TableCell>
                    <TableCell
                      className="text-sm"
                      onClick={e => e.stopPropagation()}
                    >
                      <div className="flex items-center gap-0">
                        <Button
                          hasIconOnly
                          renderIcon={Edit}
                          iconDescription="Edit tool"
                          kind="ghost"
                          size="sm"
                          onClick={() =>
                            navigation.goToEditAssistantTool(
                              assistantId,
                              itm.getId(),
                            )
                          }
                        />
                        <Button
                          hasIconOnly
                          renderIcon={TrashCan}
                          iconDescription="Delete tool"
                          kind="ghost"
                          size="sm"
                          onClick={() =>
                            showDialog(() => {
                              deleteAssistantTool(assistantId, itm.getId());
                            })
                          }
                        />
                      </div>
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        ) : (
          <EmptyState
            icon={ToolKit}
            title="No tools found"
            subtitle="Any tools or MCPs you add will be listed here."
          />
        )}
      </div>
    </div>
  );
};
