import { useProviderContext } from '@/context/provider-context';

/**
 *
 * @returns
 */
export const useAllProviderCredentials = () => {
  const { providerCredentials } = useProviderContext();
  return {
    providerCredentials,
  };
};
