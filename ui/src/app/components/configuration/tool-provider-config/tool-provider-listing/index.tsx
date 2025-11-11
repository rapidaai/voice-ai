import type { FC, HTMLAttributes } from 'react';
import { useCallback, useContext, useEffect } from 'react';
import cn from 'classnames';
import { useRapidaStore } from '@/hooks';
import toast from 'react-hot-toast/headless';
import { SearchIconInput } from '@/app/components/Form/Input/IconInput';
import { BluredWrapper } from '@/app/components/Wrapper/BluredWrapper';
import { TablePagination } from '@/app/components/base/tables/table-pagination';
import { ToolProviderContext } from '@/hooks/use-tool-provider-page-store';
import { ToolProvider } from '@rapidaai/react';
import {
  ToolProviderCard,
  ToolProviderConnectParams,
} from '@/app/components/base/cards/tool-provider-card';
import { useAllToolProviderCredentials } from '@/hooks/use-tool-provider';
import { useCurrentCredential } from '@/hooks/use-credential';

/**
 *
 * @param param0
 * @returns
 */

interface ToolProviderListingProps extends HTMLAttributes<HTMLDivElement> {
  toolFeature?: 'data.knowledge' | 'action' | 'connection';
  toolbarClassName?: string;
  toolListClassName?: string;
  connectParam?: ToolProviderConnectParams;
}

/**
 *
 * @param param0
 * @returns
 */
export const ToolProviderListing: FC<ToolProviderListingProps> = ({
  className,
  toolbarClassName,
  toolFeature,
  connectParam,
  toolListClassName,
}) => {
  //
  const toolActions = useContext(ToolProviderContext);
  const { toolProviderCredentials } = useAllToolProviderCredentials();
  const { showLoader, hideLoader } = useRapidaStore();
  const { projectId, token, authId } = useCurrentCredential();
  const onError = useCallback((err: string) => {
    hideLoader();
    toast.error(err);
  }, []);

  //
  const onSuccess = useCallback((data: ToolProvider[]) => {
    hideLoader();
  }, []);
  /**
   * call the api
   */
  const getToolProviders = useCallback((projectId, token, userId) => {
    showLoader('block');
    toolActions.getAllToolProvider(token, userId, onError, onSuccess);
  }, []);

  //
  useEffect(() => {
    toolActions.clearCriteria();
    getToolProviders(projectId, token, authId);
  }, [projectId, toolActions.page, toolActions.pageSize, toolFeature]);

  return (
    <div className={cn('space-y-4', className)}>
      <BluredWrapper className={cn(toolbarClassName, 'sticky top-0 z-1')}>
        <SearchIconInput className="bg-light-background" />
        <TablePagination
          currentPage={toolActions.page}
          onChangeCurrentPage={toolActions.setPage}
          totalItem={toolActions.totalCount}
          pageSize={toolActions.pageSize}
          onChangePageSize={toolActions.setPageSize}
        />
      </BluredWrapper>
      {toolActions.toolProviders && toolActions.toolProviders?.length > 0 && (
        <div
          className={cn(
            'overflow-y-auto sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 gap-4 grid p-4',
            toolListClassName,
          )}
        >
          {toolActions.toolProviders
            .filter(
              x =>
                toolFeature !== undefined &&
                x.getFeatureList().includes(toolFeature),
            )
            .map((item, idx) => (
              <ToolProviderCard
                key={idx}
                toolProvider={item}
                toolConnectParams={connectParam}
                isConnected={toolProviderCredentials.some(
                  x => x.getVaulttypeid() === item.getId(),
                )}
              />
            ))}
        </div>
      )}
    </div>
  );
};
