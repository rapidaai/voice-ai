import { useProviderContext } from '@/context/provider-context';

/**
 *
 *
 *
 * Started refactiruing
 * @param defaultModel
 * @returns
 */

export const useAllToolProviderCredentials = () => {
  const { toolProviderCredentials } = useProviderContext();
  return {
    toolProviderCredentials,
  };
};
