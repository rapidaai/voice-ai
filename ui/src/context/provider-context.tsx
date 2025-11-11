import { createContext, useContext } from 'use-context-selector';
import { useCallback, useEffect, useState } from 'react';
import { Provider } from '@rapidaai/react';
import { GetAllToolProviderResponse, ToolProvider } from '@rapidaai/react';
import { ServiceError } from '@rapidaai/react';
import {
  GetAllOrganizationCredentialResponse,
  VaultCredential,
} from '@rapidaai/react';
import { useCredential } from '@/hooks';
import { GetAllOrganizationCredential } from '@rapidaai/react';
import { GetAllToolProvider } from '@rapidaai/react';

import {
  LOCAL_STORAGE_PROVIDERS,
  LOCAL_STORAGE_PROVIDER_CREDENTIALS,
  LOCAL_STORAGE_TOOL_CREDENTIALS,
  LOCAL_STORAGE_TOOLS,
  serializeProto,
  useLocalStorageSync,
} from '@/hooks/use-storage-sync';
import { connectionConfig } from '@/configs';

const ProviderContext = createContext<{
  toolProviders: ToolProvider[];
  providerCredentials: VaultCredential[];
  toolProviderCredentials: VaultCredential[];
  providers: Provider[];
  reloadProviderCredentials: () => void;
  reloadToolCredentials: () => void;
}>({
  toolProviders: [],
  providerCredentials: [],
  toolProviderCredentials: [],
  providers: [],
  reloadProviderCredentials: () => {
    throw new Error('Function not implemented.');
  },
  reloadToolCredentials: () => {
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
  const [toolProviders, setTools] = useState<ToolProvider[]>([]);
  const [providerCredentials, setProviderCredentials] = useState<
    VaultCredential[]
  >([]);
  const [toolProviderCredentials, setToolProviderCredentials] = useState<
    VaultCredential[]
  >([]);
  const [providers, setProviders] = useState<Provider[]>([]);
  const [userId, token] = useCredential();

  useLocalStorageSync(
    LOCAL_STORAGE_PROVIDER_CREDENTIALS,
    setProviderCredentials,
    VaultCredential,
  );
  useLocalStorageSync(LOCAL_STORAGE_PROVIDERS, setProviders, Provider);
  useLocalStorageSync(
    LOCAL_STORAGE_TOOL_CREDENTIALS,
    setToolProviderCredentials,
    VaultCredential,
  );

  /**
   *
   */
  const afterGetAllToolProvider = useCallback(
    (err: ServiceError | null, gapr: GetAllToolProviderResponse | null) => {
      if (gapr?.getSuccess()) {
        const tools = gapr.getDataList();
        setTools(tools);
        localStorage.setItem(
          LOCAL_STORAGE_TOOLS,
          JSON.stringify(tools.map((tl: ToolProvider) => tl.toObject())),
        );
      }
    },
    [],
  );

  //   const afterGetAllProvider = useCallback(
  //     (err: ServiceError | null, gapr: GetAllModelProviderResponse | null) => {
  //       if (gapr?.getSuccess()) {
  //         console.dir(gapr.toObject());
  //         const providers = gapr.getDataList();
  //         setProviders(providers);
  //         localStorage.setItem(
  //           LOCAL_STORAGE_PROVIDERS,
  //           JSON.stringify(providers.map((tl: Provider) => tl.toObject())),
  //         );
  //       }
  //     },
  //     [],
  //   );
  /**
   *
   */
  useEffect(() => {
    if (token && userId) {
      GetAllToolProvider(connectionConfig, 1, 50, [], afterGetAllToolProvider, {
        authorization: token,
        'x-auth-id': userId,
      });

      //   GetAllProvider(connectionConfig, afterGetAllProvider, {
      //     authorization: token,
      //     'x-auth-id': userId,
      //   });
      getAllOrganizationCredential();
    }
  }, [token, userId, afterGetAllToolProvider]);

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
        setProviderCredentials(
          credentials.filter(
            (x: VaultCredential) => x.getVaulttype() === 'provider-vault',
          ),
        );
        setToolProviderCredentials(
          credentials.filter(
            (x: VaultCredential) => x.getVaulttype() === 'tool-vault',
          ),
        );
        localStorage.setItem(
          LOCAL_STORAGE_PROVIDER_CREDENTIALS,
          JSON.stringify(
            credentials.map((cred: any) => Array.from(serializeProto(cred))),
          ),
        );
        localStorage.setItem(
          LOCAL_STORAGE_TOOL_CREDENTIALS,
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
      50,
      [],
      afterGettingAllCredential,
      {
        authorization: token,
        'x-auth-id': userId,
      },
    );
  };

  /**
   * reload provider credentials
   */
  const reloadProviderCredentials = () => {
    getAllOrganizationCredential();
  };

  /**
   * reloading the tool credentials
   */
  const reloadToolCredentials = () => {
    getAllOrganizationCredential();
  };

  return (
    <ProviderContext.Provider
      value={{
        toolProviders,
        providerCredentials,
        toolProviderCredentials,
        providers,
        reloadProviderCredentials,
        reloadToolCredentials,
      }}
    >
      {children}
    </ProviderContext.Provider>
  );
};

export default ProviderContext;
