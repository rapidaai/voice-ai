import { ToolProviderContext } from '@/hooks/use-tool-provider-page-store';
import { ToolProviderType } from '@/types/types.tool-provider';
import React from 'react';

export const ToolProviderContextProvider: React.FC<{
  children;
  contextValue: ToolProviderType;
}> = ({ children, contextValue }) => {
  return (
    <ToolProviderContext.Provider value={contextValue}>
      {children}
    </ToolProviderContext.Provider>
  );
};
