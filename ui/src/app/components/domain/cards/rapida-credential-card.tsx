import { GetAllProjectCredentialResponse } from '@rapidaai/react';
import type { ProjectCredential } from '@rapidaai/react';
import { EmptyState } from '@/app/components/ui/feedback';
import { CopyButton } from '@/app/components/ui/primitives';
import { ReloadButton } from '@/app/components/ui/primitives';
import { useRapidaStore } from '@/stores/app';
import { useCurrentCredential } from '@/hooks/use-credential';
import { toHumanReadableRelativeDay } from '@/utils/date';
import { useEffect, useState } from 'react';
import toast from 'react-hot-toast/headless';
import { listProjectCredentials } from '@/clients';
import { CreateProjectToken } from '@/app/components/domain/project-credentials/create-project-token';

export const RapidaCredentialCard = () => {
  const [ourKeys, setOurKeys] = useState<ProjectCredential[]>([]);
  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [error, setError] = useState('');
  const { authId, token, projectId } = useCurrentCredential();
  const { loading, showLoader, hideLoader } = useRapidaStore();

  const onCreateProjectCredential = () => {
    if (!projectId) return;
    setIsCreateOpen(true);
  };

  useEffect(() => {
    getAllProjectCredential();
  }, [projectId]);

  const getAllProjectCredential = () => {
    showLoader();
    setError('');
    const loadedCredentials: ProjectCredential[] = [];
    const loadPage = (page: number) => {
      listProjectCredentials({
        projectId,
        page,
        pageSize: 100,
        criteria: [],
        auth: { token, userId: authId },
        callback: (_err, data: GetAllProjectCredentialResponse | null) => {
          if (!data?.getSuccess()) {
            hideLoader();
            const message =
              data?.getError()?.getHumanmessage() ||
              'Project API keys could not be loaded.';
            setError(message);
            toast.error(message);
            return;
          }
          const pageCredentials = data.getDataList();
          loadedCredentials.push(...pageCredentials);
          const totalItems =
            data.getPaginated()?.getTotalitem() || loadedCredentials.length;
          if (
            pageCredentials.length === 0 &&
            loadedCredentials.length < totalItems
          ) {
            hideLoader();
            setError('Project API keys could not be loaded.');
            return;
          }
          if (loadedCredentials.length < totalItems) {
            loadPage(page + 1);
            return;
          }
          setOurKeys(loadedCredentials);
          hideLoader();
        },
      });
    };
    loadPage(1);
  };

  return (
    <div className="border border-border-subtle ">
      <div className="h-10 px-4 flex items-center justify-between border-b border-border-subtle ">
        <span className="text-xs font-medium uppercase tracking-[0.08em] text-muted ">
          Project API keys
        </span>
        <ReloadButton
          className="h-7 text-xs"
          isLoading={loading}
          onClick={getAllProjectCredential}
        />
      </div>

      {error ? (
        <div className="px-4 flex justify-center">
          <EmptyState
            title="API keys could not be loaded"
            subtitle={error}
            action="Try again"
            onAction={getAllProjectCredential}
          />
        </div>
      ) : ourKeys.length === 0 ? (
        <div className="px-4 flex justify-center">
          <EmptyState
            title="No project API keys"
            subtitle="Create an API key for an application or tool that needs access to this project."
            action="Create API key"
            onAction={onCreateProjectCredential}
          />
        </div>
      ) : (
        <div className="divide-y divide-border-subtle ">
          {ourKeys.map(x => (
            <div
              key={x.getId()}
              className="h-12 px-4 flex items-center justify-between gap-4"
            >
              <div className="flex items-center gap-3 min-w-0">
                <span className="text-xs font-medium uppercase tracking-[0.08em] text-muted shrink-0">
                  {x.getName()}
                </span>
                <span className="font-mono text-xs text-foreground truncate">
                  {x.getKey()}
                </span>
                <span className="text-xs text-muted truncate">
                  {x.getRolename()}
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
      <CreateProjectToken
        open={isCreateOpen}
        projectId={projectId}
        auth={{ token, userId: authId }}
        onClose={() => setIsCreateOpen(false)}
        onCreated={getAllProjectCredential}
      />
    </div>
  );
};
