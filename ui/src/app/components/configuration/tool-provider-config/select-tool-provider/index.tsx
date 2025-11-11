import type { FC, HTMLAttributes } from 'react';
import React, { useCallback, useContext, useEffect } from 'react';
import cn from 'classnames';
import { useCredential, useRapidaStore } from '@/hooks';
import toast from 'react-hot-toast/headless';
import { SearchIconInput } from '@/app/components/Form/Input/IconInput';
import { BluredWrapper } from '@/app/components/Wrapper/BluredWrapper';
import { TablePagination } from '@/app/components/base/tables/table-pagination';
import { ToolProviderContext } from '@/hooks/use-tool-provider-page-store';
import {
  ToolProviderCard,
  ToolProviderConnectParams,
} from '@/app/components/base/cards/tool-provider-card';
import CheckboxCard from '@/app/components/Form/checkbox-card';
import { useAllToolProviderCredentials } from '@/hooks/use-tool-provider';
import { ToolProvider } from '@rapidaai/react';

/**
 *
 * @param param0
 * @returns
 */

interface MultiSelectProps {
  //
  selectedToolProviders: Array<ToolProvider>;

  //
  onSelectToolProviders: (tools: Array<ToolProvider>) => void;
}

interface SingleSelectProps {
  //
  selectedToolProvider: ToolProvider | null;

  //
  onSelectToolProvider: (tool: ToolProvider) => void;
}

interface SelectToolProviderProps extends HTMLAttributes<HTMLDivElement> {
  //
  toolFeature?: 'data.knowledge' | 'connection';
  toolbarClassName?: string;

  //
  connectParam?: ToolProviderConnectParams;
}

/**
 *
 * @param param0
 * @returns
 */
export const MultiSelectToolProvider: FC<
  SelectToolProviderProps & MultiSelectProps
> = ({
  className,
  toolbarClassName,
  toolFeature,
  connectParam,
  selectedToolProviders,
  onSelectToolProviders,
}) => {
  //
  const [userId, token, projectId] = useCredential();

  const { toolProviderCredentials } = useAllToolProviderCredentials();

  //
  const toolActions = useContext(ToolProviderContext);

  const toggleSelect = (tool: ToolProvider) => {
    const isSelected = selectedToolProviders.some(
      item => item.getId() === tool.getId(),
    );
    if (isSelected) {
      onSelectToolProviders(
        selectedToolProviders.filter(item => item.getId() !== tool.getId()),
      );
    } else {
      onSelectToolProviders([...selectedToolProviders, tool]);
    }
  };

  //
  const { showLoader, hideLoader } = useRapidaStore();

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

  useEffect(() => {
    getToolProviders(projectId, token, userId);
  }, [projectId, toolActions.page, toolActions.pageSize, toolFeature]);

  return (
    <div className={cn('space-y-4 overflow-auto relative', className)}>
      <BluredWrapper
        className={cn(toolbarClassName, 'border-none sticky top-0 z-1')}
      >
        <SearchIconInput iconClassName="w-4 h-4" className="pl-7" />
        <TablePagination
          currentPage={toolActions.page}
          onChangeCurrentPage={toolActions.setPage}
          totalItem={toolActions.totalCount}
          pageSize={toolActions.pageSize}
          onChangePageSize={toolActions.setPageSize}
        />
      </BluredWrapper>
      {toolActions.toolProviders && toolActions.toolProviders?.length > 0 && (
        <div className="overflow-y-auto grid-cols-3 grid gap-4 px-4 py-2">
          {toolActions.toolProviders
            .filter(
              x =>
                toolFeature !== undefined &&
                x.getFeatureList().includes(toolFeature),
            )
            .map((item, idx) => (
              <CheckboxCard
                key={`${idx}-checkbox-sd-tool`}
                id={`${idx}-checkbox-sd-tool`}
                name={`${idx}-checkbox-sd-tool`}
                checked={selectedToolProviders.some(
                  i => i.getId() === item.getId(),
                )}
                type="checkbox"
                disabled={
                  !toolProviderCredentials.some(
                    x => x.getVaulttypeid() === item.getId(),
                  )
                }
                onChange={() => {
                  toggleSelect(item);
                }}
              >
                <ToolProviderCard
                  key={idx}
                  toolProvider={item}
                  isConnected={toolProviderCredentials.some(
                    x => x.getVaulttypeid() === item.getId(),
                  )}
                  toolConnectParams={connectParam}
                />
              </CheckboxCard>
            ))}
        </div>
      )}
    </div>
  );
};

export const SelectToolProvider: FC<
  SelectToolProviderProps & SingleSelectProps
> = ({
  className,
  toolbarClassName,
  toolFeature,
  connectParam,
  selectedToolProvider,
  onSelectToolProvider,
}) => {
  //
  const [userId, token, projectId] = useCredential();
  const { toolProviderCredentials } = useAllToolProviderCredentials();
  const toolActions = useContext(ToolProviderContext);
  const { showLoader, hideLoader } = useRapidaStore();
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

  useEffect(() => {
    getToolProviders(projectId, token, userId);
  }, []);

  return (
    <div className={cn('space-y-4 overflow-auto relative', className)}>
      <BluredWrapper className={cn(toolbarClassName, 'sticky top-0 z-1')}>
        <SearchIconInput iconClassName="w-4 h-4" className="pl-7" />
        <TablePagination
          currentPage={toolActions.page}
          onChangeCurrentPage={toolActions.setPage}
          totalItem={toolActions.totalCount}
          pageSize={toolActions.pageSize}
          onChangePageSize={toolActions.setPageSize}
        />
      </BluredWrapper>
      {toolActions.toolProviders && toolActions.toolProviders?.length > 0 && (
        <div className="overflow-y-auto grid-cols-3 grid gap-4 px-4 py-2">
          {toolActions.toolProviders
            .filter(
              x =>
                toolFeature !== undefined &&
                x.getFeatureList().includes(toolFeature),
            )
            .map((item, idx) => (
              <CheckboxCard
                key={`${idx}-checkbox-sd-tool`}
                id={`${idx}-checkbox-sd-tool`}
                name={`radio-checkbox-sd-tool`}
                checked={
                  selectedToolProvider
                    ? selectedToolProvider.getId() === item.getId()
                    : false
                }
                type="radio"
                disabled={
                  !toolProviderCredentials.some(
                    x => x.getVaulttypeid() === item.getId(),
                  )
                }
                onChange={() => {
                  onSelectToolProvider(item);
                }}
              >
                <ToolProviderCard
                  key={idx}
                  toolProvider={item}
                  isConnected={toolProviderCredentials.some(
                    x => x.getVaulttypeid() === item.getId(),
                  )}
                  toolConnectParams={connectParam}
                />
              </CheckboxCard>
            ))}
        </div>
      )}
    </div>
  );
};
