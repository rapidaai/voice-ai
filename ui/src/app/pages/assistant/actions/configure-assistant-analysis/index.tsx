import { FC, useEffect } from 'react';
import { useParams } from 'react-router-dom';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { cn, toHumanReadableDateTime } from '@/styles/media';
import { ExternalLink, Plus, RotateCw } from 'lucide-react';
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
import { IBlueButton, IButton } from '@/app/components/Form/Button';
import toast from 'react-hot-toast/headless';
import { CardOptionMenu } from '@/app/components/Menu';
import { ActionableEmptyMessage } from '@/app/components/container/message/actionable-empty-message';
import { CreateAssistantAnalysis } from '@/app/pages/assistant/actions/configure-assistant-analysis/create-assistant-analysis';
import { useAssistantAnalysisPageStore } from '@/app/pages/assistant/actions/store/use-analysis-page-store';
import { UpdateAssistantAnalysis } from '@/app/pages/assistant/actions/configure-assistant-analysis/update-assistant-analysis';
import { PaginationButtonBlock } from '@/app/components/blocks/pagination-button-block';
import { CustomLink } from '@/app/components/custom-link';

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
      v => {
        hideLoader();
      },
    );
  };
  useEffect(() => {
    get();
  }, []);

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
            onClick={() => navigation.goToCreateAssistantAnalysis(assistantId)}
          >
            Create new analysis
            <Plus className="w-4 h-4 ml-1.5" />
          </IBlueButton>
        </PaginationButtonBlock>
      </BluredWrapper>
      <TableSection>
        {axtion.analysises.length > 0 ? (
          <ScrollableResizableTable
            isActionable={false}
            isOptionable={true}
            clms={axtion.columns.filter(x => x.visible)}
          >
            {axtion.analysises.map((row, idx) => (
              <TableRow key={idx} data-id={row.getId()}>
                {axtion.visibleColumn('id') && (
                  <TableCell className="">{row.getId()}</TableCell>
                )}

                {axtion.visibleColumn('name') && (
                  <TableCell className="truncate text-blue-600 hover:underline">
                    {row.getName()}
                  </TableCell>
                )}
                {axtion.visibleColumn('endpointId') && (
                  <TableCell>
                    <CustomLink
                      to={`/deployment/endpoint/${row.getEndpointid()}`}
                      className="font-normal dark:text-blue-500 text-blue-600 hover:underline cursor-pointer text-left flex items-center space-x-1"
                    >
                      <span>{row.getEndpointid()}</span>
                      <ExternalLink className="w-3 h-3" />
                    </CustomLink>
                  </TableCell>
                )}
                {axtion.visibleColumn('endpointVersion') && (
                  <TableCell className="">{row.getEndpointversion()}</TableCell>
                )}
                {axtion.visibleColumn('executionPriority') && (
                  <TableCell className="">
                    {row.getExecutionpriority()}
                  </TableCell>
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
                            Update analysis
                          </div>
                        ),
                        onActionClick: () => {
                          navigation.goToEditAssistantAnalysis(
                            assistantId,
                            row.getId(),
                          );
                        },
                      },
                      {
                        option: (
                          <div className="flex items-center text-sm">
                            Delete analysis
                          </div>
                        ),
                        onActionClick: () => {
                          deleteAssistantAnalysis(assistantId, row.getId());
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
              title="No Analysis"
              subtitle="There are no assistant analysis."
              action="Create new analysis"
              onActionClick={() =>
                navigation.goToCreateAssistantAnalysis(assistantId)
              }
            />
          </div>
        )}
      </TableSection>
    </div>
  );
};
