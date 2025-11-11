import type { FC, HTMLAttributes } from 'react';
import React, { useCallback, useContext, useEffect } from 'react';
import cn from 'classnames';
import { useCredential, useRapidaStore } from '@/hooks';
import toast from 'react-hot-toast/headless';
import { Dropdown } from '@/app/components/Dropdown';
import { BlueBorderButton } from '@/app/components/Form/Button';
import { AlertTriangle } from '@/app/components/Icon/alert-triangle';
import { ToolProviderConnectParams } from '@/app/components/base/cards/tool-provider-card';
import { ToolProvider } from '@rapidaai/react';
import { ToolProviderContext } from '@/hooks/use-tool-provider-page-store';
import { useAllToolProviderCredentials } from '@/hooks/use-tool-provider';

/**
 *
 * @param param0
 * @returns
 */

interface ToolDropdownProps extends HTMLAttributes<HTMLDivElement> {
  currentToolId?: string;
  setCurrentToolProvider: (tl: ToolProvider) => void;
  toolFeature?: 'knowledge' | 'action' | 'connection';
  toolbarClassName?: string;
  toolListClassName?: string;
  connectParam?: ToolProviderConnectParams;
}

/**
 *
 * @param param0
 * @returns
 */
export const ToolDropdown: FC<ToolDropdownProps> = ({
  currentToolId,
  setCurrentToolProvider,
  className,
  toolFeature,
  connectParam,
  toolListClassName,
}) => {
  //
  const [userId, token, projectId] = useCredential();
  const toolActions = useContext(ToolProviderContext);
  const { toolProviderCredentials } = useAllToolProviderCredentials();
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
  const getTools = useCallback((projectId, token, userId) => {
    showLoader('block');
    toolActions.getAllToolProvider(token, userId, onError, onSuccess);
  }, []);

  //
  useEffect(() => {
    toolActions.clearCriteria();
    if (toolFeature) toolActions.addCriteria('feature', toolFeature, '=');
    getTools(projectId, token, userId);
  }, [projectId, toolActions.page, toolActions.pageSize, toolFeature]);

  const dropdownItem = (tool: ToolProvider) => {
    return (
      <div className="flex justify-between items-center w-full group">
        <span
          data-id={tool.getId()}
          parent-data-id={tool.getId()}
          className={cn('inline-flex items-center gap-1.5 sm:gap-2 w-full')}
        >
          <img
            alt={tool.getName()}
            loading="lazy"
            className="w-5 h-5 align-middle block shrink-0"
            src={tool?.getImage()}
          />
          <span className="truncate capitalize">{tool.getName()}</span>
        </span>
        {!toolProviderCredentials.some(
          x => x.getVaulttypeid() === tool.getId(),
        ) && (
          <div className="flex justify-center items-center">
            <BlueBorderButton
              className="h-fit text-xs border-[0.5px] px-4 py-1 group-hover:visible invisible rounded-[2px]"
              onClick={() => {}}
            >
              Setup Credential
            </BlueBorderButton>
            <AlertTriangle className="text-yellow-600 group-hover:hidden block" />
          </div>
        )}
      </div>
    );
  };
  return (
    <Dropdown
      currentValue={
        toolActions.toolProviders.find(x => x.getId() === currentToolId) ||
        toolActions.toolProviders[0] ||
        null
      }
      setValue={setCurrentToolProvider}
      allValue={toolActions.toolProviders}
      placeholder="Select the provider"
      label={dropdownItem}
      option={dropdownItem}
    />
  );
};
