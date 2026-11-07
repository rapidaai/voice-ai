import { FC, useCallback, useEffect, useState } from 'react';
import toast from 'react-hot-toast/headless';
import { useParams } from 'react-router-dom';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { EmptyState } from '@/app/components/ui/feedback';
import { Edit, TrashCan, ToolKit } from '@carbon/icons-react';
import { CreateTool } from '@/app/pages/assistant/actions/configure-assistant-tool/create-assistant-tool';
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
  DataTable,
  DataTableSkeleton,
  Table,
  TableHead,
  TableRow,
  TableHeader,
  TableBody,
  TableCell,
  TableToolbar,
  TableToolbarContent,
  TableToolbarSearch,
  TableBatchActions,
  TableBatchAction,
  TableSelectRow,
  Tag,
} from '@carbon/react';
import { toHumanReadableDateTime } from '@/utils/date';
import { AssistantSettingsShellHeader } from '@/app/pages/assistant/components/settings-shell-header';
import { TableSection } from '@/app/components/layout/sections/table-section';

const headers = [
  { key: 'name', header: 'Name' },
  { key: 'type', header: 'Type' },
  { key: 'description', header: 'Description' },
  { key: 'date', header: 'Date' },
];

const skeletonHeaders = [{ key: 'select', header: '' }, ...headers];

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
  const navigation = useGlobalNavigation();
  const action = useAssistantToolPageStore();
  const { authId, token, projectId } = useCurrentCredential();
  const [isFetching, setIsFetching] = useState(true);

  const fetchTools = useCallback(() => {
    setIsFetching(true);
    action.getAssistantTool(
      assistantId,
      projectId,
      token,
      authId,
      e => {
        toast.error(e);
        setIsFetching(false);
      },
      () => {
        setIsFetching(false);
      },
    );
  }, [action.getAssistantTool, assistantId, projectId, token, authId]);

  useEffect(() => {
    fetchTools();
  }, [fetchTools]);

  const { showDialog, ConfirmDialogComponent } = useConfirmDialog({
    title: 'Are you sure?',
    content: 'You want to delete? The tool will be removed from assistant.',
  });

  const deleteAssistantTool = (
    assistantId: string,
    assistantToolId: string,
  ) => {
    setIsFetching(true);
    action.deleteAssistantTool(
      assistantId,
      assistantToolId,
      projectId,
      token,
      authId,
      e => {
        toast.error(e);
        setIsFetching(false);
      },
      () => {
        toast.success('Assistant tool deleted successfully');
        fetchTools();
      },
    );
  };

  const refreshTools = () => {
    fetchTools();
  };

  const tableRows = action.tools.map(tool => {
    const method = tool.getExecutionmethod();
    const methodMeta = BUILDIN_TOOLS.find(
      candidate => candidate.code === method,
    );

    return {
      id: tool.getId(),
      name: tool.getName(),
      type:
        methodMeta?.name || (method === 'mcp' ? 'MCP' : method || 'Unknown'),
      description: tool.getDescription(),
      date: tool.getCreateddate()
        ? toHumanReadableDateTime(tool.getCreateddate()!)
        : '—',
    };
  });

  return (
    <div className="relative flex h-full min-h-0 flex-1 flex-col bg-surface">
      <ConfirmDialogComponent />

      <AssistantSettingsShellHeader
        title="Tool & MCP"
        href={`/deployment/assistant/${assistantId}/configure-tool`}
        onReload={refreshTools}
        reloadLabel="Reload tools"
        primaryAction={{
          label: 'Add tool',
          onClick: () => navigation.goToCreateAssistantTool(assistantId),
        }}
      />

      {isFetching ? (
        <TableSection className="min-h-0 overflow-hidden bg-surface">
          <DataTableSkeleton
            headers={skeletonHeaders}
            columnCount={skeletonHeaders.length}
            rowCount={5}
            showHeader={false}
            showToolbar
            size="lg"
            className="w-full"
          />
        </TableSection>
      ) : action.tools.length > 0 ? (
        <TableSection className="min-h-0 bg-surface">
          <DataTable rows={tableRows} headers={headers} radio>
            {({
              rows,
              headers: tableHeaders,
              selectedRows,
              getBatchActionProps,
              getHeaderProps,
              getRowProps,
              getSelectionProps,
              getTableProps,
              getToolbarProps,
              onInputChange,
            }) => {
              const batchActionProps = getBatchActionProps();
              const selectedTool = selectedRows[0]
                ? action.tools.find(tool => tool.getId() === selectedRows[0].id)
                : undefined;

              return (
                <>
                  <TableToolbar {...getToolbarProps()}>
                    {selectedTool && (
                      <TableBatchActions
                        {...batchActionProps}
                        shouldShowBatchActions
                      >
                        <TableBatchAction
                          renderIcon={Edit}
                          iconDescription="Edit"
                          onClick={() => {
                            navigation.goToEditAssistantTool(
                              assistantId,
                              selectedTool.getId(),
                            );
                            batchActionProps.onCancel();
                          }}
                        >
                          Edit
                        </TableBatchAction>
                        <TableBatchAction
                          renderIcon={TrashCan}
                          iconDescription="Delete"
                          onClick={() => {
                            showDialog(() => {
                              deleteAssistantTool(
                                assistantId,
                                selectedTool.getId(),
                              );
                            });
                            batchActionProps.onCancel();
                          }}
                        >
                          Delete
                        </TableBatchAction>
                      </TableBatchActions>
                    )}
                    <TableToolbarContent>
                      <TableToolbarSearch
                        placeholder="Search tools..."
                        onChange={onInputChange}
                      />
                    </TableToolbarContent>
                  </TableToolbar>

                  <div className="min-h-0 flex-1 overflow-auto">
                    <Table {...getTableProps()} className="min-w-max">
                      <TableHead>
                        <TableRow>
                          <TableHeader className="!w-12" />
                          {tableHeaders.map(header => (
                            <TableHeader
                              {...getHeaderProps({ header })}
                              key={header.key}
                            >
                              {header.header}
                            </TableHeader>
                          ))}
                        </TableRow>
                      </TableHead>
                      <TableBody>
                        {rows.map(tableRow => {
                          const tool = action.tools.find(
                            candidate => candidate.getId() === tableRow.id,
                          );
                          if (!tool) return null;

                          const method = tool.getExecutionmethod();
                          const methodMeta = BUILDIN_TOOLS.find(
                            candidate => candidate.code === method,
                          );
                          const isMcp = method === 'mcp';
                          const conditionSource = getToolConditionSource(
                            tool.getExecutionoptionsList(),
                          );

                          return (
                            <TableRow
                              {...getRowProps({ row: tableRow })}
                              key={tableRow.id}
                            >
                              <TableSelectRow
                                {...getSelectionProps({ row: tableRow })}
                              />
                              <TableCell className="text-sm">
                                {tool.getName()}
                              </TableCell>
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
                                      {getToolConditionSourceLabel(
                                        conditionSource,
                                      )}
                                    </Tag>
                                  )}
                                </div>
                              </TableCell>
                              <TableCell className="max-w-[360px] truncate text-sm">
                                {tool.getDescription()}
                              </TableCell>
                              <TableCell className="text-[13px] whitespace-nowrap">
                                {tool.getCreateddate()
                                  ? toHumanReadableDateTime(
                                      tool.getCreateddate()!,
                                    )
                                  : '—'}
                              </TableCell>
                            </TableRow>
                          );
                        })}
                      </TableBody>
                    </Table>
                  </div>
                </>
              );
            }}
          </DataTable>
        </TableSection>
      ) : (
        <div className="flex flex-1 items-center justify-center">
          <EmptyState
            icon={ToolKit}
            title="No tools found"
            subtitle="Any tools or MCPs you add will be listed here."
          />
        </div>
      )}
    </div>
  );
};
