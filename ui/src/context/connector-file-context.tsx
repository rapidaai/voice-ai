import { ConnectorFileContext } from '@/hooks/use-connector-file-page-store';
import { ConnectorFileType } from '@/types/types.connector-file';
import React from 'react';

export const ConnectorFileContextProvider: React.FC<{
  children;
  contextValue: ConnectorFileType;
}> = ({ children, contextValue }) => {
  return (
    <ConnectorFileContext.Provider value={contextValue}>
      {children}
    </ConnectorFileContext.Provider>
  );
};
