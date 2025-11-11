import React, { FC, useEffect } from 'react';
import { useParams } from 'react-router-dom';

import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { cn, toHumanReadableDateTime } from '@/styles/media';
import { Plus, RotateCw } from 'lucide-react';
import { useCurrentCredential } from '@/hooks/use-credential';
import { useRapidaStore } from '@/hooks';
import { SectionLoader } from '@/app/components/Loader/section-loader';
import { BluredWrapper } from '@/app/components/Wrapper/BluredWrapper';
import { SearchIconInput } from '@/app/components/Form/Input/IconInput';
import { TablePagination } from '@/app/components/base/tables/table-pagination';
import { TableSection } from '@/app/components/sections/table-section';
import { ScrollableResizableTable } from '@/app/components/data-table';
import { TableRow } from '@/app/components/base/tables/table-row';
import { TableCell } from '@/app/components/base/tables/table-cell';
import { StatusIndicator } from '@/app/components/indicators/status';
import { CreateAssistantWebhook } from './create-assistant-webhook';
import { IBlueButton, IButton } from '@/app/components/Form/Button';
import toast from 'react-hot-toast/headless';
import { CardOptionMenu } from '@/app/components/Menu';
import { ActionableEmptyMessage } from '@/app/components/container/message/actionable-empty-message';
import { UpdateAssistantWebhook } from '@/app/pages/assistant/actions/configure-assistant-webhook/update-assistant-webhook';
import { useAssistantWebhookPageStore } from '@/app/pages/assistant/actions/store/use-webhook-page-store';
import { PaginationButtonBlock } from '@/app/components/blocks/pagination-button-block';

export function ConfigureAssistantWebhookPage() {
  const { assistantId } = useParams();
  return (
    <>
      {assistantId && <ConfigureAssistantWebhook assistantId={assistantId} />}
    </>
  );
}

export function CreateAssistantWebhookPage() {
  const { assistantId } = useParams();
  return (
    <>{assistantId && <CreateAssistantWebhook assistantId={assistantId} />}</>
  );
}

export function UpdateAssistantWebhookPage() {
  const { assistantId } = useParams();
  return (
    <>{assistantId && <UpdateAssistantWebhook assistantId={assistantId} />}</>
  );
}

const ConfigureAssistantWebhook: FC<{ assistantId: string }> = ({
  assistantId,
}) => {
  const navigation = useGlobalNavigation();
  const axtion = useAssistantWebhookPageStore();
  const { authId, token, projectId } = useCurrentCredential();
  const { loading, showLoader, hideLoader } = useRapidaStore();

  useEffect(() => {
    showLoader('block');
    get();
  }, []);

  const get = () => {
    axtion.getAssistantWebhook(
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
  const deleteAssistantWebhook = (assistantId: string, webhookId: string) => {
    showLoader('block');
    axtion.deleteAssistantWebhook(
      assistantId,
      webhookId,
      projectId,
      token,
      authId,
      e => {
        toast.error(e);
        hideLoader();
      },
      v => {
        get();
      },
    );
  };

  if (loading) {
    return (
      <div className="h-full w-full flex flex-col items-center justify-center">
        <SectionLoader />
      </div>
    );
  }
  return (
    <div className="h-full flex flex-col flex-1">
      <BluredWrapper className="border-t-0 p-0">
        <div className="flex space-x-2">
          <SearchIconInput className="bg-light-background" />
        </div>
        <PaginationButtonBlock>
          <TablePagination
            className="py-0"
            columns={axtion.columns}
            currentPage={axtion.page}
            onChangeCurrentPage={axtion.setPage}
            totalItem={axtion.totalCount}
            pageSize={axtion.pageSize}
            onChangePageSize={axtion.setPageSize}
            onChangeColumns={axtion.setColumns}
          />
          <IButton onClick={get}>
            <RotateCw className="w-4 h-4" strokeWidth={1.5} />
          </IButton>
          <IBlueButton
            onClick={() => navigation.goToCreateAssistantWebhook(assistantId)}
          >
            Create new webhook
            <Plus className="w-4 h-4 ml-1.5" />
          </IBlueButton>
        </PaginationButtonBlock>
      </BluredWrapper>
      <TableSection>
        {axtion.webhooks.length > 0 ? (
          <ScrollableResizableTable
            isActionable={false}
            isOptionable={true}
            clms={axtion.columns.filter(x => x.visible)}
          >
            {axtion.webhooks.map((row, idx) => (
              <TableRow key={idx} data-id={row.getId()}>
                {axtion.visibleColumn('id') && (
                  <TableCell className="">{row.getId()}</TableCell>
                )}

                {axtion.visibleColumn('httpUrl') && (
                  <TableCell className="truncate">
                    {row.getHttpmethod()}:{row.getHttpurl()}
                  </TableCell>
                )}
                {axtion.visibleColumn('events') && (
                  <TableCell className="gap">
                    <div className="flex flex-wrap gap-2">
                      {row.getAssistanteventsList().map((event, index) => (
                        <span
                          key={index}
                          className="px-2 py-1 text-sm font-mono bg-blue-600/10 text-blue-600"
                        >
                          {event}
                        </span>
                      ))}
                    </div>
                  </TableCell>
                )}
                {axtion.visibleColumn('maxRetryCount') && (
                  <TableCell className="">{row.getRetrycount()}</TableCell>
                )}

                {axtion.visibleColumn('timeoutSeconds') && (
                  <TableCell className="">{row.getTimeoutsecond()}</TableCell>
                )}
                {axtion.visibleColumn('executionPriority') && (
                  <TableCell>{row.getExecutionpriority()}</TableCell>
                )}
                {axtion.visibleColumn('status') && (
                  <TableCell className="">
                    <StatusIndicator state={row.getStatus()} />
                  </TableCell>
                )}
                {axtion.visibleColumn('created_date') && (
                  <TableCell>
                    {row.getCreateddate() &&
                      toHumanReadableDateTime(row.getCreateddate()!)}
                  </TableCell>
                )}
                <TableCell>
                  <CardOptionMenu
                    classNames={cn('w-9 h-9')}
                    options={[
                      {
                        option: (
                          <div className="flex items-center text-sm">
                            <span>Update webhook</span>
                          </div>
                        ),
                        onActionClick: () => {
                          navigation.goToEditAssistantWebhook(
                            assistantId,
                            row.getId(),
                          );
                        },
                      },
                      {
                        option: (
                          <div className="flex items-center text-sm justify-between">
                            <span>Delete webhook</span>
                          </div>
                        ),
                        onActionClick: () => {
                          deleteAssistantWebhook(assistantId, row.getId());
                        },
                      },
                    ]}
                  />
                </TableCell>
              </TableRow>
            ))}
          </ScrollableResizableTable>
        ) : (
          <div className="flex flex-1 w-full justify-center items-center">
            <ActionableEmptyMessage
              title="No Webhook"
              subtitle="There are no assistant webhook found."
              action="Create new webhook"
              onActionClick={() =>
                navigation.goToCreateAssistantWebhook(assistantId)
              }
            />
          </div>
        )}
      </TableSection>
    </div>
  );
};
