import { ActionConnect } from '@rapidaai/react';
import { ActionConnectResponse } from '@rapidaai/react';
import { useCredential } from '@/hooks';
import { FC, useCallback, useEffect } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { ServiceError } from '@rapidaai/react';
import { PageLoader } from '@/app/components/Loader/page-loader';
import toast from 'react-hot-toast/headless';
import { useProviderContext } from '@/context/provider-context';
import { connectionConfig } from '@/configs';

export const ConnectSlackActionPage: FC = () => {
  const [searchParams] = useSearchParams();
  const { state, code, scope } = Object.fromEntries(searchParams.entries());
  const [userId, token, projectId] = useCredential();
  const providerCtx = useProviderContext();

  const navigator = useNavigate();

  const onComplete = useCallback(
    (err: ServiceError | null, uvcr: ActionConnectResponse | null) => {
      if (uvcr?.getSuccess()) {
        providerCtx.reloadToolCredentials();
      }

      if (!uvcr?.getSuccess()) {
        toast.error('Unable to connect knwoledgebase, please try again later.');
        return;
      }
      let redirectTo: string = uvcr.getRedirectto();
      navigator(redirectTo);
    },
    [],
  );
  useEffect(() => {
    ActionConnect(
      connectionConfig,
      'slack',
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
