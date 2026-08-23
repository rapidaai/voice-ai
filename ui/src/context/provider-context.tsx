import { createContext, useContext } from 'use-context-selector';
import { useCallback, useEffect, useState } from 'react';
import { ConnectionConfig, ServiceError } from '@rapidaai/react';
import {
  GetAllOrganizationCredentialResponse,
  VaultCredential,
} from '@rapidaai/react';
import { GetAllOrganizationCredential } from '@rapidaai/react';

import {
  LOCAL_STORAGE_PROVIDER_CREDENTIALS,
  serializeProto,
  useLocalStorageSync,
} from '@/hooks/use-storage-sync';
import { connectionConfig } from '@/configs';
import { useCurrentCredential } from '@/hooks/use-credential';

const ProviderContext = createContext<{
  providerCredentials: VaultCredential[];
  reloadProviderCredentials: () => void;
}>({
  providerCredentials: [],
  reloadProviderCredentials: () => {
    throw new Error('Function not implemented.');
  },
});

export const useProviderContext = () => useContext(ProviderContext);
type ProviderContextProviderProps = {
  children: React.ReactNode;
};

export const ProviderContextProvider = ({
  children,
}: ProviderContextProviderProps) => {
  const [providerCredentials, setProviderCredentials] = useState<
    VaultCredential[]
  >([]);
  const { authId, projectId, token } = useCurrentCredential();
  useLocalStorageSync(
    LOCAL_STORAGE_PROVIDER_CREDENTIALS,
    setProviderCredentials,
    VaultCredential,
  );

  /**
   *
   */
  useEffect(() => {
    if (token && authId && projectId) {
      getAllOrganizationCredential();
    }
  }, [token, authId, projectId]);

  /**
   * after getting all the credentials to store in the local storage
   */
  const afterGettingAllCredential = useCallback(
    (
      err: ServiceError | null,
      gapcr: GetAllOrganizationCredentialResponse | null,
    ) => {
      if (gapcr?.getSuccess()) {
        const credentials = gapcr.getDataList();
        setProviderCredentials(credentials);
        localStorage.setItem(
          LOCAL_STORAGE_PROVIDER_CREDENTIALS,
          JSON.stringify(
            credentials.map((cred: any) => Array.from(serializeProto(cred))),
          ),
        );
      }
    },
    [],
  );

  /**
   * gettung all the organization
   */
  const getAllOrganizationCredential = () => {
    GetAllOrganizationCredential(
      connectionConfig,
      1,
      100,
      [],
      afterGettingAllCredential,
      ConnectionConfig.WithDebugger({
        authorization: token,
        userId: authId,
        projectId: projectId,
      }),
    );
  };

  /**
   * reload provider credentials
   */
  const reloadProviderCredentials = () => {
    getAllOrganizationCredential();
  };

  return (
    <ProviderContext.Provider
      value={{
        providerCredentials,
        reloadProviderCredentials,
      }}
    >
      {children}
    </ProviderContext.Provider>
  );
};
