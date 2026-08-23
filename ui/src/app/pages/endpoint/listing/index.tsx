import { useEffect, useCallback, useState } from 'react';
import { SingleEndpoint } from './single-endpoint';
import { useCredential } from '@/hooks/use-credential';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { useEndpointPageStore } from '@/hooks';
import { Helmet } from '@/app/components/helmet';
import { Endpoint } from '@rapidaai/react';
import toast from 'react-hot-toast/headless';
import { useRapidaStore } from '@/hooks';
import { PrimaryButton } from '@/app/components/carbon/button';
import { Pagination } from '@/app/components/carbon/pagination';
import { EmptyState } from '@/app/components/carbon/empty-state';
import { Add, Renew, Connect } from '@carbon/icons-react';
import { PageLoading } from '@/app/components/carbon/loading';
import { PageHeaderBlock } from '@/app/components/blocks/page-header-block';
import { PageTitleBlock } from '@/app/components/blocks/page-title-block';
import {
  Table,
  TableHead,
  TableRow,
  TableHeader,
  TableBody,
  TableToolbar,
  TableToolbarContent,
  Button,
} from '@carbon/react';
import {
  EndpointQuerySearch,
  getEndpointSearchCriteria,
} from './endpoint-query-search';

const CREATE_ENDPOINT_LABEL = 'Create new endpoint';

const endpointColumnClassName: Record<string, string> = {
  getName: 'min-w-56 whitespace-nowrap',
  getId: 'min-w-64 whitespace-nowrap',
  getStatus: 'min-w-28 whitespace-nowrap',
  getCurrentModel: 'min-w-36 whitespace-nowrap',
  getVersion: 'min-w-44 whitespace-nowrap',
  getTags: 'min-w-40 whitespace-nowrap',
  getCount: 'min-w-28 whitespace-nowrap',
  getErrorRate: 'min-w-36 whitespace-nowrap',
  getP50: 'min-w-32 whitespace-nowrap',
  getP99: 'min-w-32 whitespace-nowrap',
  getCost: 'min-w-28 whitespace-nowrap',
  getMRR: 'min-w-36 whitespace-nowrap',
  getCreatedActor: 'min-w-32 whitespace-nowrap',
  action: 'w-16 min-w-16 whitespace-nowrap',
};

const formatQuerySearchValue = (key: string, value: string): string =>
  /\s/.test(value)
    ? `${key}:"${value.replace(/"/g, '\\"')}"`
    : `${key}:${value}`;

export function EndpointPage() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const [userId, token, projectId] = useCredential();
  const endpointActions = useEndpointPageStore();
  const { loading, showLoader, hideLoader } = useRapidaStore();
  const [querySearchValue, setQuerySearchValue] = useState('');

  useEffect(() => {
    if (searchParams) {
      const searchParamMap = Object.fromEntries(searchParams.entries());
      const nextQuerySearchValue = Object.entries(searchParamMap)
        .filter(([, value]) => value)
        .map(([key, value]) => formatQuerySearchValue(key, value))
        .join(' ');

      setQuerySearchValue(nextQuerySearchValue);
      endpointActions.setCriterias(
        getEndpointSearchCriteria(nextQuerySearchValue),
      );
    }
  }, [searchParams]);

  const onError = useCallback((err: string) => {
    hideLoader();
    toast.error(err);
  }, []);

  const onSuccess = useCallback((_data: Endpoint[]) => {
    hideLoader();
  }, []);

  const getEndpoints = useCallback(
    (projectId: string, token: string, userId: string) => {
      showLoader();
      endpointActions.onGetAllEndpoint(
        projectId,
        token,
        userId,
        onError,
        onSuccess,
      );
    },
    [],
  );

  useEffect(() => {
    getEndpoints(projectId, token, userId);
  }, [
    projectId,
    endpointActions.page,
    endpointActions.pageSize,
    endpointActions.criteria,
  ]);

  const visibleColumns = endpointActions.columns.filter(x => x.visible);

  return (
    <div className="h-full flex flex-col overflow-hidden">
      <Helmet title="Hosted endpoints" />

      <PageHeaderBlock>
        <div className="flex items-center gap-3">
          <PageTitleBlock>Hosted Endpoints</PageTitleBlock>
          <span className="text-xs text-gray-500 dark:text-gray-400 tabular-nums">
            {endpointActions.endpoints.length}/{endpointActions.totalCount}
          </span>
        </div>
      </PageHeaderBlock>

      <TableToolbar>
        <TableToolbarContent>
          <EndpointQuerySearch
            value={querySearchValue}
            onChange={setQuerySearchValue}
            onApply={criteria => endpointActions.setCriterias(criteria)}
          />
          <Button
            hasIconOnly
            renderIcon={Renew}
            iconDescription="Refresh"
            kind="ghost"
            onClick={() => getEndpoints(projectId, token, userId)}
            tooltipPosition="bottom"
          />
          <PrimaryButton
            size="md"
            renderIcon={Add}
            onClick={() => navigate('/deployment/endpoint/create-endpoint')}
          >
            {CREATE_ENDPOINT_LABEL}
          </PrimaryButton>
        </TableToolbarContent>
      </TableToolbar>

      {loading ? (
        <PageLoading />
      ) : endpointActions.endpoints && endpointActions.endpoints.length > 0 ? (
        <div className="overflow-auto flex-1">
          <Table>
            <TableHead>
              <TableRow>
                {visibleColumns.map(col => (
                  <TableHeader
                    key={col.key}
                    className={endpointColumnClassName[col.key]}
                  >
                    {col.name}
                  </TableHeader>
                ))}
              </TableRow>
            </TableHead>
            <TableBody>
              {endpointActions.endpoints.map(ed => (
                <SingleEndpoint
                  key={`endpoint_row_${ed.getId()}`}
                  endpoint={ed}
                />
              ))}
            </TableBody>
          </Table>
        </div>
      ) : endpointActions.criteria.length > 0 ? (
        <EmptyState
          icon={Connect}
          title="No endpoints found"
          subtitle="No endpoints match your current filters."
          action={CREATE_ENDPOINT_LABEL}
          actionIcon={Add}
          onAction={() => navigate('/deployment/endpoint/create-endpoint')}
        />
      ) : (
        <EmptyState
          icon={Connect}
          title="No endpoints"
          subtitle="Deploy governed APIs for client, brand, or business-unit workflows with audit trails and access control."
          action={CREATE_ENDPOINT_LABEL}
          actionIcon={Add}
          onAction={() => navigate('/deployment/endpoint/create-endpoint')}
        />
      )}

      {endpointActions.endpoints && endpointActions.endpoints.length > 0 && (
        <Pagination
          className="shrink-0"
          totalItems={endpointActions.totalCount}
          page={endpointActions.page}
          pageSize={endpointActions.pageSize}
          pageSizes={[10, 20, 50]}
          onChange={({ page, pageSize }) => {
            if (pageSize !== endpointActions.pageSize) {
              endpointActions.setPageSize(pageSize);
            } else {
              endpointActions.setPage(page);
            }
          }}
        />
      )}
    </div>
  );
}
