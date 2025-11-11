import {
  ConnectionConfig,
  GeneralConnect,
  GeneralConnectResponse,
} from '@rapidaai/react';
import { useCredential } from '@/hooks';
import { FC, useCallback, useEffect } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { ServiceError } from '@rapidaai/react';
import { PageLoader } from '@/app/components/Loader/page-loader';
import toast from 'react-hot-toast/headless';
import { useProviderContext } from '@/context/provider-context';
import { connectionConfig } from '@/configs';

export const ConnectHubspotCRMPage: FC = () => {
  const [searchParams] = useSearchParams();
  const { state, code, scope } = Object.fromEntries(searchParams.entries());
  const [userId, token, projectId] = useCredential();
  const providerCtx = useProviderContext();

  const navigator = useNavigate();
  const onComplete = useCallback(
    (err: ServiceError | null, uvcr: GeneralConnectResponse | null) => {
      if (!uvcr || !uvcr.getSuccess()) {
        toast.error('Unable to connect hubspot, please try again later.');
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
    GeneralConnect(
      connectionConfig,
      'hubspot',
      code,
      state,
      scope,
      ConnectionConfig.WithDebugger({
        authorization: token,
        projectId: projectId,
        userId: userId,
      }),
      onComplete,
    );
  }, [state, code, scope]);

  //
  return <PageLoader />;
};
