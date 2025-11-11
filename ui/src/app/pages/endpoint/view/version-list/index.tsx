import { Endpoint, EndpointProviderModel } from '@rapidaai/react';
import { BorderButton } from '@/app/components/Form/Button';
import { useEndpointProviderModelPageStore } from '@/hooks';
import { useRapidaStore } from '@/hooks';
import { useCurrentCredential } from '@/hooks/use-credential';
import React, { useEffect } from 'react';
import toast from 'react-hot-toast/headless';
import { cn, toHumanReadableRelativeTime } from '@/styles/media';
import { TextImage } from '@/app/components/Image/TextImage';
import { CopyButton } from '@/app/components/Form/Button/copy-button';
import { SingleDotIcon } from '@/app/components/Icon/single-dot';
import { RevisionIndicator } from '@/app/components/indicators/revision';
import { ReloadIcon } from '@/app/components/Icon/Reload';

export function Version(props: {
  currentEndpoint: Endpoint;
  onReload: () => void;
}) {
  const { authId, token, projectId } = useCurrentCredential();
  const rapidaContext = useRapidaStore();
  const endpointProviderAction = useEndpointProviderModelPageStore();

  useEffect(() => {
    rapidaContext.showLoader();
    endpointProviderAction.onChangeCurrentEndpoint(props.currentEndpoint);
    endpointProviderAction.getEndpointProviderModels(
      props.currentEndpoint.getId(),
      projectId,
      token,
      authId,
      (err: string) => {
        rapidaContext.hideLoader();
        toast.error(err);
      },
      (data: EndpointProviderModel[]) => {
        rapidaContext.hideLoader();
      },
    );
  }, [
    props.currentEndpoint,
    projectId,
    endpointProviderAction.page,
    endpointProviderAction.pageSize,
    endpointProviderAction.criteria,
  ]);

  const deployRevision = endpointProviderModelId => {
    rapidaContext.showLoader('overlay');
    endpointProviderAction.onReleaseVersion(
      endpointProviderModelId,
      projectId,
      token,
      authId,
      error => {
        rapidaContext.hideLoader();
        toast.error(error);
      },
      e => {
        toast.success(
          'New version of endpoint has been deployed successfully.',
        );
        endpointProviderAction.onChangeCurrentEndpoint(e);
        props.onReload();
        rapidaContext.hideLoader();
      },
    );
  };
  return (
    <div className="p-4 w-full">
      <div className="px-0 py-0 divide-y shadow-sm max-w-4xl mx-auto bg-white dark:bg-gray-950">
        {endpointProviderAction.endpointProviderModels.map((epm, idx) => {
          return (
            <article className="px-4 py-3.5 dark:border-gray-800" key={idx}>
              <div className="flex items-center justify-between">
                <div className="flex">
                  <div className="mr-3 truncate text-base font-semibold prose-a:hover:underline">
                    {epm.getDescription()
                      ? epm.getDescription()
                      : 'Initial endpoint version'}
                  </div>
                  <div className="inline-flex rounded-[2px] border leading-snug text-sm dark:border-gray-800">
                    <span className="flex items-center overflow-hidden whitespace-nowrap px-1.5 py-0.5 pl-2 tracking-wide">
                      vrsn_{epm.getId()}
                    </span>{' '}
                    <CopyButton className="border-none">
                      {`vrsn_${epm.getId()}`}
                    </CopyButton>
                  </div>
                </div>

                <div className="gap-2 flex items-center shrink-0">
                  <RevisionIndicator
                    status={
                      endpointProviderAction.currentEndpoint?.getEndpointprovidermodelid() ===
                      epm.getId()
                        ? 'DEPLOYED'
                        : 'NOT_DEPLOYED'
                    }
                  />
                  {endpointProviderAction.currentEndpoint?.getEndpointprovidermodelid() !==
                    epm.getId() && (
                    <BorderButton
                      className={cn(' h-7 w-7 p-0 shrink-0 rounded-[2px]')}
                      onClick={() => {
                        deployRevision(epm.getId());
                      }}
                    >
                      <ReloadIcon className="w-4 h-4" />
                    </BorderButton>
                  )}
                </div>
              </div>
              <div className="flex flex-wrap items-center whitespace-nowrap text-sm flex-1 mt-1">
                {epm.getCreateduser() && (
                  <>
                    <div className="shrink-0 mr-1.5">
                      <TextImage
                        size={4}
                        name={epm.getCreateduser()?.getName()!}
                      ></TextImage>
                    </div>
                    <div className="opacity-70 font-medium">
                      {epm.getCreateduser()?.getName()!}
                    </div>
                  </>
                )}
                <SingleDotIcon />
                <span className="truncate opacity-70">
                  Updated{' '}
                  <span>
                    {epm?.getCreateddate() &&
                      toHumanReadableRelativeTime(epm?.getCreateddate()!)}
                  </span>
                </span>
              </div>
            </article>
          );
        })}
      </div>
    </div>
  );
}
