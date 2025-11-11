import { useEffect, useCallback, useState } from 'react';
import { SingleEndpoint } from './single-endpoint';
import { useCredential } from '@/hooks/use-credential';
import { TablePagination } from '@/app/components/base/tables/table-pagination';
import { SearchIconInput } from '@/app/components/Form/Input/IconInput';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { useEndpointPageStore } from '@/hooks';
import { Helmet } from '@/app/components/Helmet';
import { BluredWrapper } from '@/app/components/Wrapper/BluredWrapper';
import { Endpoint } from '@rapidaai/react';
import toast from 'react-hot-toast/headless';
import { useRapidaStore } from '@/hooks';
import { Spinner } from '@/app/components/Loader/Spinner';
import { HowEndpointWorksDialog } from '@/app/components/base/modal/how-it-works-modal/how-endpoint-works';
import { ActionableEmptyMessage } from '@/app/components/container/message/actionable-empty-message';
import { IBlueButton, IButton } from '@/app/components/Form/Button';
import { Plus, RotateCw } from 'lucide-react';
import { PageHeaderBlock } from '@/app/components/blocks/page-header-block';
import { PageTitleBlock } from '@/app/components/blocks/page-title-block';
import { ScrollableResizableTable } from '@/app/components/data-table';
import { PaginationButtonBlock } from '@/app/components/blocks/pagination-button-block';

/**
 *
 * @returns
 */
export function EndpointPage() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const [userId, token, projectId] = useCredential();
  const endpointActions = useEndpointPageStore();
  const { loading, showLoader, hideLoader } = useRapidaStore();
  const navigator = useNavigate();

  /**
   *
   */
  useEffect(() => {
    if (searchParams) {
      const searchParamMap = Object.fromEntries(searchParams.entries());
      Object.entries(searchParamMap).forEach(([key, value]) =>
        endpointActions.addCriteria(key, value, '='),
      );
    }
  }, [searchParams]);

  const onError = useCallback((err: string) => {
    hideLoader();
    toast.error(err);
  }, []);

  const onSuccess = useCallback((data: Endpoint[]) => {
    hideLoader();
  }, []);
  /**
   * call the api
   */
  const getEndpoints = useCallback((projectId, token, userId) => {
    showLoader();
    endpointActions.onGetAllEndpoint(
      projectId,
      token,
      userId,
      onError,
      onSuccess,
    );
  }, []);

  /**
   *
   */
  useEffect(() => {
    getEndpoints(projectId, token, userId);
  }, [
    projectId,
    endpointActions.page,
    endpointActions.pageSize,
    endpointActions.criteria,
  ]);

  const [hiw, sethiw] = useState(false);
  return (
    <>
      <Helmet title="Hosted endpoints" />
      <HowEndpointWorksDialog setModalOpen={sethiw} modalOpen={hiw} />
      <PageHeaderBlock>
        <div className="flex items-center gap-3">
          <PageTitleBlock>Hosted Endpoints</PageTitleBlock>
          <div className="text-xs opacity-75">
            {`${endpointActions.endpoints.length}/${endpointActions.totalCount}`}
          </div>
        </div>
        <div className="flex">
          <IButton
            className="border-r"
            onClick={() => {
              sethiw(!hiw);
            }}
          >
            How it works?
          </IButton>
          <IBlueButton
            onClick={() => {
              navigate('/deployment/endpoint/create-endpoint');
            }}
          >
            Add new endpoint
            <Plus strokeWidth={1.5} className="ml-1.5 h-4 w-4" />
          </IBlueButton>
        </div>
      </PageHeaderBlock>
      <BluredWrapper>
        <SearchIconInput className="bg-light-background" />
        <PaginationButtonBlock>
          <TablePagination
            columns={endpointActions.columns}
            currentPage={endpointActions.page}
            onChangeCurrentPage={endpointActions.setPage}
            totalItem={endpointActions.totalCount}
            pageSize={endpointActions.pageSize}
            onChangePageSize={endpointActions.setPageSize}
            onChangeColumns={endpointActions.setColumns}
          />
          <IButton
            onClick={() => {
              getEndpoints(projectId, token, userId);
            }}
          >
            <RotateCw strokeWidth={1.5} className="h-4 w-4" />
          </IButton>
        </PaginationButtonBlock>
      </BluredWrapper>

      {endpointActions.endpoints && endpointActions.endpoints.length > 0 ? (
        <ScrollableResizableTable
          isActionable={false}
          clms={endpointActions.columns.filter(x => {
            return x.visible;
          })}
          className="w-[2800px]"
        >
          {endpointActions.endpoints.map((ed, idx) => {
            return (
              <SingleEndpoint
                key={`endpoint_row_${ed.getId()}`}
                endpoint={ed}
              ></SingleEndpoint>
            );
          })}
        </ScrollableResizableTable>
      ) : endpointActions.criteria.length > 0 ? (
        <div className="h-full flex justify-center items-center">
          <ActionableEmptyMessage
            title="No Endpoint"
            subtitle="There are no endpoints matching with your criteria to display"
            action="Create new endpoint"
            onActionClick={() =>
              navigator('/deployment/endpoint/create-endpoint')
            }
          />
        </div>
      ) : !loading ? (
        <div className="h-full flex justify-center items-center">
          <ActionableEmptyMessage
            title="No Endpoint"
            subtitle="There are no endpoints deployed to display"
            action="Create new endpoint"
            onActionClick={() =>
              navigator('/deployment/endpoint/create-endpoint')
            }
          />
        </div>
      ) : (
        <div className="h-full flex justify-center items-center">
          <Spinner size="md" />
        </div>
      )}
    </>
  );
}
