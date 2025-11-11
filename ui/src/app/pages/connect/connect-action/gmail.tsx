import { ActionConnectResponse } from '@rapidaai/react';
import { useCredential } from '@/hooks';
import { FC, useCallback, useEffect } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';

import { PageLoader } from '@/app/components/Loader/page-loader';
import toast from 'react-hot-toast/headless';
import { useProviderContext } from '@/context/provider-context';
import { ServiceError } from '@rapidaai/react';
import { connectionConfig } from '@/configs';
import { ActionConnect } from '@rapidaai/react';

export const ConnectGmailActionPage: FC = () => {
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
      'gmail',
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
