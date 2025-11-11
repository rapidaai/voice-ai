import {
  CreateProjectCredential,
  GetAllProjectCredential,
} from '@rapidaai/react';
import {
  CreateProjectCredentialResponse,
  GetAllProjectCredentialResponse,
  ProjectCredential,
} from '@rapidaai/react';
import { ActionableEmptyMessage } from '@/app/components/container/message/actionable-empty-message';
import { CopyButton } from '@/app/components/Form/Button/copy-button';
import { ReloadButton } from '@/app/components/Form/Button/ReloadButton';
import { useRapidaStore } from '@/hooks';
import { useCurrentCredential } from '@/hooks/use-credential';
import { toHumanReadableRelativeDay } from '@/styles/media';
import { useCallback, useEffect, useState } from 'react';
import toast from 'react-hot-toast/headless';
import { connectionConfig } from '@/configs';

export const RapidaCredentialCard = () => {
  /**
   * all the result
   */
  const [ourKeys, setOurKeys] = useState<ProjectCredential[]>([]);

  /**
   * Current project credential
   */
  //   const { currentProjectRole } = useContext(useAuthenticationStore);

  /**
   * authentication
   */
  //   const [userId, token] = useCredential();
  const { authId, token, projectId } = useCurrentCredential();

  /**
   *
   */
  const { loading, showLoader, hideLoader } = useRapidaStore();

  /**
   * on create project credential
   */
  const onCreateProjectCredential = () => {
    if (!projectId) return;
    CreateProjectCredential(
      connectionConfig,
      projectId,
      'publishable key',
      afterCreateProjectCredential,
      {
        authorization: token,
        'x-auth-id': authId,
      },
    );
  };

  /**
   * after create project credential
   */
  const afterCreateProjectCredential = useCallback(
    (err, data: CreateProjectCredentialResponse | null) => {
      if (data?.getSuccess()) {
        getAllProjectCredential();
      } else {
        let errorMessage = data?.getError();
        if (errorMessage) {
          toast.error(errorMessage.getHumanmessage());
        } else {
          toast.error(
            'Unable to process your request. please try again later.',
          );
        }
      }
    },
    [],
  );

  /**
   * after get all the project credentials
   */
  const afterGetAllProjectCredential = useCallback(
    (err, data: GetAllProjectCredentialResponse | null) => {
      hideLoader();
      if (data?.getSuccess()) {
        setOurKeys(data.getDataList());
      } else {
        let errorMessage = data?.getError();
        if (errorMessage) {
          toast.error(errorMessage.getHumanmessage());
        } else {
          toast.error(
            'Unable to process your request. please try again later.',
          );
        }
      }
    },
    [],
  );

  useEffect(() => {
    getAllProjectCredential();
  }, [projectId]);

  //   when someone add things then reload the state

  const getAllProjectCredential = () => {
    showLoader();
    GetAllProjectCredential(
      connectionConfig,
      projectId,
      afterGetAllProjectCredential,
      {
        authorization: token,
        'x-auth-id': authId,
      },
    );
  };

  return (
    <div className="shadow-xs rounded-lg border dark:border-slate-800">
      <div className="flex justify-between items-center border-b dark:border-slate-800 px-4 py-2">
        <h1 className="font-medium text-base">
          SDK Authentication Credentials
        </h1>
        <div className="flex">
          <ReloadButton
            className="h-7 text-xs"
            isLoading={loading}
            onClick={getAllProjectCredential}
          />
        </div>
      </div>
      {ourKeys.length === 0 && (
        <div className="px-4 flex justify-center">
          <ActionableEmptyMessage
            title="No credentials"
            subtitle="There are no SDK Authentication Credential found to display"
            action="Create new credential"
            onActionClick={() => {
              onCreateProjectCredential();
              // navigator('/deployment/endpoint/create-endpoint');
            }}
          />
        </div>
      )}

      {ourKeys.map((x, idx) => {
        return (
          <div className="space-x-4 px-4 py-2 flex opacity-80" key={idx}>
            <div className="flex flex-col justify-between items-start w-2/3 max-w-full">
              <p className="text-sm font-medium mb-1">Publishable key</p>
              <div className="flex items-center space-x-2 justify-between max-w-full">
                <div className="truncate max-w-full">{x.getKey()}</div>
                <CopyButton className="shrink-0">{x.getKey()}</CopyButton>
              </div>
            </div>
            <div className="w-1/3">
              <p className="text-sm font-medium mb-1">Created on</p>
              <p className="text-sm text-gray-500">
                {x.getCreateddate()
                  ? toHumanReadableRelativeDay(x.getCreateddate()!)
                  : 'Not enabled'}
              </p>
            </div>
          </div>
        );
      })}
    </div>
  );
};
