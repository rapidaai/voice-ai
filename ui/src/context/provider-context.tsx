import { createContext, useContext } from 'use-context-selector';
import { useCallback, useEffect, useRef, useState } from 'react';
import type { VaultCredential } from '@rapidaai/react';
import { useCurrentCredential } from '@/hooks/use-credential';
import { listOrganizationCredentials } from '@/clients';

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
  const latestRequestRef = useRef(0);

  const getAllOrganizationCredential = useCallback(() => {
    if (!token || !authId || !projectId) return;
    const requestId = ++latestRequestRef.current;
    listOrganizationCredentials({
      page: 1,
      pageSize: 100,
      criteria: [],
      auth: { token, userId: authId, projectId },
      callback: (_error, response) => {
        if (requestId !== latestRequestRef.current) return;
        if (response?.getSuccess()) {
          setProviderCredentials(response.getDataList());
        }
      },
    });
  }, [authId, projectId, token]);

  useEffect(() => {
    latestRequestRef.current += 1;
    setProviderCredentials([]);
    getAllOrganizationCredential();

    return () => {
      latestRequestRef.current += 1;
    };
  }, [getAllOrganizationCredential]);

  return (
    <ProviderContext.Provider
      value={{
        providerCredentials,
        reloadProviderCredentials: getAllOrganizationCredential,
      }}
    >
      {children}
    </ProviderContext.Provider>
  );
};
