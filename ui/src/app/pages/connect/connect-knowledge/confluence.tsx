import { KnowledgeConnect } from '@rapidaai/react';
import { KnowledgeConnectResponse } from '@rapidaai/react';
import { useCredential } from '@/hooks';
import { FC, useCallback, useEffect } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { ServiceError } from '@rapidaai/react';
import { PageLoader } from '@/app/components/Loader/page-loader';
import toast from 'react-hot-toast/headless';
import { useProviderContext } from '@/context/provider-context';
import { connectionConfig } from '@/configs';

export const ConnectConfluencePage: FC = () => {
  const [searchParams] = useSearchParams();
  const { state, code, scope } = Object.fromEntries(searchParams.entries());
  const [userId, token, projectId] = useCredential();
  const providerCtx = useProviderContext();

  const navigator = useNavigate();
  const onComplete = useCallback(
    (err: ServiceError | null, uvcr: KnowledgeConnectResponse | null) => {
      if (!uvcr || !uvcr.getSuccess()) {
        toast.error('Unable to connect confluence, please try again later.');
        // making default route
      }
      if (uvcr?.getSuccess()) {
        providerCtx.reloadToolCredentials();
        let redirectTo: string = uvcr.getRedirectto();
        navigator(redirectTo);
        return;
      }

      navigator('/integration/tools');
      return;
    },
    [],
  );
  useEffect(() => {
    KnowledgeConnect(
      connectionConfig,
      'atlassian',
      code,
      state,
      scope,
      {
        authorization: token,
        'x-project-id': projectId,
        'x-auth-id': userId,
      },
      onComplete,
    );
  }, [state, code, scope]);

  //
  return <PageLoader />;
};
