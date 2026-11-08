import {
  CreateProjectCredentialResponse,
  GetAllProjectCredentialResponse,
} from '@rapidaai/react';
import type { ProjectCredential } from '@rapidaai/react';
import { EmptyState } from '@/app/components/ui/feedback';
import { CopyButton } from '@/app/components/ui/primitives';
import { ReloadButton } from '@/app/components/ui/primitives';
import { useRapidaStore } from '@/stores/app';
import { useCurrentCredential } from '@/hooks/use-credential';
import { toHumanReadableRelativeDay } from '@/utils/date';
import { useCallback, useEffect, useState } from 'react';
import toast from 'react-hot-toast/headless';
import {
  createProjectPublishableCredential,
  listProjectCredentials,
} from '@/clients';

export const RapidaCredentialCard = () => {
  const [ourKeys, setOurKeys] = useState<ProjectCredential[]>([]);
  const { authId, token, projectId } = useCurrentCredential();
  const { loading, showLoader, hideLoader } = useRapidaStore();

  const onCreateProjectCredential = () => {
    if (!projectId) return;
    createProjectPublishableCredential({
      projectId,
      name: 'publishable key',
      auth: { token, userId: authId },
      callback: afterCreateProjectCredential,
    });
  };

  const afterCreateProjectCredential = useCallback(
    (_err: unknown, data: CreateProjectCredentialResponse | null) => {
      if (data?.getSuccess()) {
        getAllProjectCredential();
      } else {
        const errorMessage = data?.getError();
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

  const afterGetAllProjectCredential = useCallback(
    (_err: unknown, data: GetAllProjectCredentialResponse | null) => {
      hideLoader();
      if (data?.getSuccess()) {
        setOurKeys(data.getDataList());
      } else {
        const errorMessage = data?.getError();
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

  const getAllProjectCredential = () => {
    showLoader();
    listProjectCredentials({
      projectId,
      auth: { token, userId: authId },
      callback: afterGetAllProjectCredential,
    });
  };

  return (
    <div className="border border-border-subtle ">
      <div className="h-10 px-4 flex items-center justify-between border-b border-border-subtle ">
        <span className="text-xs font-medium uppercase tracking-[0.08em] text-muted ">
          SDK Credentials
        </span>
        <ReloadButton
          className="h-7 text-xs"
          isLoading={loading}
          onClick={getAllProjectCredential}
        />
      </div>

      {ourKeys.length === 0 ? (
        <div className="px-4 flex justify-center">
          <EmptyState
            title="No credentials"
            subtitle="There are no SDK Authentication Credentials found to display"
            action="Create new credential"
            onAction={onCreateProjectCredential}
          />
        </div>
      ) : (
        <div className="divide-y divide-border-subtle ">
          {ourKeys.map((x, idx) => (
            <div
              key={idx}
              className="h-12 px-4 flex items-center justify-between gap-4"
            >
              <div className="flex items-center gap-3 min-w-0">
                <span className="text-xs font-medium uppercase tracking-[0.08em] text-muted shrink-0">
                  Publishable key
                </span>
                <span className="font-mono text-xs text-foreground truncate">
                  {x.getKey()}
                </span>
              </div>
              <div className="flex items-center gap-3 shrink-0">
                <span className="text-xs tabular-nums text-muted ">
                  {x.getCreateddate()
                    ? toHumanReadableRelativeDay(x.getCreateddate()!)
                    : '—'}
                </span>
                <CopyButton>{x.getKey()}</CopyButton>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
};
