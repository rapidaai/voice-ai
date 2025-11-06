import { IBlueBorderButton } from '@/app/components/Form/Button';
import { useAssistantProviderPageStore } from '@/hooks';
import { useRapidaStore } from '@/hooks';
import { useCredential } from '@/hooks/use-credential';
import { useEffect } from 'react';
import toast from 'react-hot-toast/headless';
import { cn, toHumanReadableDateTime } from '@/styles/media';
import { Assistant, GetAllAssistantProviderResponse } from '@rapidaai/react';
import { CopyButton } from '@/app/components/Form/Button/copy-button';
import { RevisionIndicator } from '@/app/components/indicators/revision';
import { SectionLoader } from '@/app/components/Loader/section-loader';
import { Brain, ChevronsLeftRightEllipsis, Code, Rocket } from 'lucide-react';
import { TableSection } from '@/app/components/sections/table-section';
import { ScrollableResizableTable } from '@/app/components/data-table';
import { TableRow } from '@/app/components/base/tables/table-row';
import { TableCell } from '@/app/components/base/tables/table-cell';

interface VersionProps {
  assistant: Assistant;
  onReload: () => void;
}

export function Version(props: VersionProps) {
  const [userId, token, projectId] = useCredential();
  const rapidaContext = useRapidaStore();
  const assistantProviderAction = useAssistantProviderPageStore();

  useEffect(() => {
    rapidaContext.showLoader();
    assistantProviderAction.onChangeAssistant(props.assistant);
    assistantProviderAction.getAssistantProviders(
      props.assistant.getId(),
      projectId,
      token,
      userId,
      (err: string) => {
        rapidaContext.hideLoader();
        toast.error(err);
      },
      data => {
        rapidaContext.hideLoader();
      },
    );
  }, [
    props.assistant.getId(),
    projectId,
    assistantProviderAction.page,
    assistantProviderAction.pageSize,
    assistantProviderAction.criteria,
  ]);

  const deployRevision = (
    assistantProvider: string,
    assistantProviderId: string,
  ) => {
    rapidaContext.showLoader('overlay');
    assistantProviderAction.onReleaseVersion(
      assistantProvider,
      assistantProviderId,
      projectId,
      token,
      userId,
      error => {
        rapidaContext.hideLoader();
        toast.error(error);
      },
      e => {
        toast.success(
          'New version of assistant has been deployed successfully.',
        );
        assistantProviderAction.onChangeAssistant(e);
        props.onReload();
        rapidaContext.hideLoader();
      },
    );
  };
  if (rapidaContext.loading) {
    return (
      <div className="h-full flex flex-col items-center justify-center">
        <SectionLoader />
      </div>
    );
  }

  return (
    <TableSection>
      <ScrollableResizableTable
        isActionable={false}
        clms={assistantProviderAction.columns.filter(x => x.visible)}
      >
        {assistantProviderAction.assistantProviders.map((apm, idx) => {
          switch (apm.getAssistantproviderCase()) {
            case GetAllAssistantProviderResponse.AssistantProvider
              .AssistantproviderCase.ASSISTANTPROVIDERMODEL:
              return (
                <TableRow key={idx} className="cursor-pointer" data-id={idx}>
                  <TableCell>
                    <div className="flex items-center space-x-2 font-mono text-sm">
                      <span>
                        vrsn_{apm.getAssistantprovidermodel()?.getId()}
                      </span>
                      <CopyButton className="border-none">
                        {`vrsn_${apm.getAssistantprovidermodel()?.getId()}`}
                      </CopyButton>
                    </div>
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center">
                      <Brain className="w-4 h-4" strokeWidth={1.5} />
                      <span className="ml-2">LLM</span>
                    </div>
                  </TableCell>
                  <TableCell>
                    {apm.getAssistantprovidermodel()?.getDescription()
                      ? apm.getAssistantprovidermodel()?.getDescription()
                      : 'Initial assistant version'}
                  </TableCell>
                  <TableCell data-id={apm.getAssistantprovidermodel()?.getId()}>
                    {apm.getAssistantprovidermodel()?.getCreateduser() &&
                      apm
                        .getAssistantprovidermodel()
                        ?.getCreateduser()
                        ?.getName()!}
                  </TableCell>
                  <TableCell>
                    {apm.getAssistantprovidermodel()?.getCreateddate() &&
                      toHumanReadableDateTime(
                        apm.getAssistantprovidermodel()?.getCreateddate()!,
                      )}
                  </TableCell>
                  <TableCell>
                    {assistantProviderAction.assistant?.getAssistantproviderid() !==
                    apm.getAssistantprovidermodel()?.getId() ? (
                      <IBlueBorderButton
                        disabled={
                          assistantProviderAction.assistant?.getAssistantproviderid() ===
                          apm.getAssistantprovidermodel()?.getId()
                        }
                        className={cn(
                          'shrink-0 rounded-[2px] h-7 text-sm font-medium px-2 bg-blue-500/10 !border-blue-500/40 border-[0.1px]',
                        )}
                        onClick={() => {
                          deployRevision(
                            'MODEL',
                            apm.getAssistantprovidermodel()?.getId()!,
                          );
                        }}
                      >
                        <Rocket
                          className="w-3.5 h-3.5 mr-2"
                          strokeWidth={1.5}
                        />
                        <span>Deploy Version</span>
                      </IBlueBorderButton>
                    ) : (
                      <RevisionIndicator
                        status={
                          assistantProviderAction.assistant?.getAssistantproviderid() ===
                          apm.getAssistantprovidermodel()?.getId()
                            ? 'DEPLOYED'
                            : 'NOT_DEPLOYED'
                        }
                      />
                    )}
                  </TableCell>
                </TableRow>
              );
            case GetAllAssistantProviderResponse.AssistantProvider
              .AssistantproviderCase.ASSISTANTPROVIDERAGENTKIT:
              return (
                <TableRow key={idx} className="cursor-pointer" data-id={idx}>
                  <TableCell>
                    <div className="flex space-x-2 items-center font-mono text-sm">
                      <span className="font-mono">
                        vrsn_{apm.getAssistantprovideragentkit()?.getId()}
                      </span>
                      <CopyButton className="border-none">
                        {`vrsn_${apm.getAssistantprovideragentkit()?.getId()}`}
                      </CopyButton>
                    </div>
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center">
                      <Code className="w-5 h-5" strokeWidth={1.5} />
                      <span className="ml-2">AgentKit</span>
                    </div>
                  </TableCell>
                  <TableCell>
                    {apm.getAssistantprovideragentkit()?.getDescription()
                      ? apm.getAssistantprovideragentkit()?.getDescription()
                      : 'Initial assistant version'}
                  </TableCell>

                  <TableCell>
                    {
                      apm
                        .getAssistantprovideragentkit()
                        ?.getCreateduser()
                        ?.getName()!
                    }
                  </TableCell>
                  <TableCell>
                    {apm.getAssistantprovideragentkit()?.getCreateddate() &&
                      toHumanReadableDateTime(
                        apm.getAssistantprovideragentkit()?.getCreateddate()!,
                      )}
                  </TableCell>
                  <TableCell>
                    {assistantProviderAction.assistant?.getAssistantproviderid() !==
                    apm.getAssistantprovideragentkit()?.getId() ? (
                      <IBlueBorderButton
                        disabled={
                          assistantProviderAction.assistant?.getAssistantproviderid() ===
                          apm.getAssistantprovideragentkit()?.getId()
                        }
                        className={cn(
                          'shrink-0 rounded-[2px] h-7 text-sm font-medium px-2  bg-blue-500/10 !border-blue-500/40 border-[0.1px]',
                        )}
                        onClick={() => {
                          deployRevision(
                            'AGENTKIT',
                            apm.getAssistantprovideragentkit()?.getId()!,
                          );
                        }}
                      >
                        <Rocket
                          className="w-3.5 h-3.5 mr-2"
                          strokeWidth={1.5}
                        />
                        <span>Deploy Version</span>
                      </IBlueBorderButton>
                    ) : (
                      <RevisionIndicator
                        status={
                          assistantProviderAction.assistant?.getAssistantproviderid() ===
                          apm.getAssistantprovideragentkit()?.getId()
                            ? 'DEPLOYED'
                            : 'NOT_DEPLOYED'
                        }
                      />
                    )}
                  </TableCell>
                </TableRow>
              );
            case GetAllAssistantProviderResponse.AssistantProvider
              .AssistantproviderCase.ASSISTANTPROVIDERWEBSOCKET:
              return (
                <TableRow key={idx}>
                  <TableCell>
                    <div className="flex space-x-2 font-mono text-sm">
                      <span>
                        vrsn_{apm.getAssistantproviderwebsocket()?.getId()}
                      </span>
                      <CopyButton className="border-none">
                        {`vrsn_${apm.getAssistantproviderwebsocket()?.getId()}`}
                      </CopyButton>
                    </div>
                  </TableCell>

                  <TableCell>
                    <div className="flex">
                      <ChevronsLeftRightEllipsis
                        className="w-5 h-5"
                        strokeWidth={1.5}
                      />
                      <span className="ml-2">Websocket</span>
                    </div>
                  </TableCell>
                  <TableCell>
                    {apm.getAssistantproviderwebsocket()?.getDescription()
                      ? apm.getAssistantproviderwebsocket()?.getDescription()
                      : 'Initial assistant version'}
                  </TableCell>
                  <TableCell>
                    {apm.getAssistantproviderwebsocket()?.getCreateduser() &&
                      apm
                        .getAssistantproviderwebsocket()
                        ?.getCreateduser()
                        ?.getName()!}
                  </TableCell>
                  <TableCell>
                    {apm.getAssistantproviderwebsocket()?.getCreateddate() &&
                      toHumanReadableDateTime(
                        apm.getAssistantproviderwebsocket()?.getCreateddate()!,
                      )}
                  </TableCell>

                  <TableCell>
                    {assistantProviderAction.assistant?.getAssistantproviderid() !==
                    apm.getAssistantproviderwebsocket()?.getId() ? (
                      <IBlueBorderButton
                        disabled={
                          assistantProviderAction.assistant?.getAssistantproviderid() ===
                          apm.getAssistantproviderwebsocket()?.getId()
                        }
                        className={cn(
                          'shrink-0 rounded-[2px] h-7 text-sm font-medium px-2  bg-blue-500/10 !border-blue-500/40 border-[0.1px]',
                        )}
                        onClick={() => {
                          deployRevision(
                            'WEBSOCKET',
                            apm.getAssistantproviderwebsocket()?.getId()!,
                          );
                        }}
                      >
                        <Rocket
                          className="w-3.5 h-3.5 mr-2"
                          strokeWidth={1.5}
                        />
                        <span>Deploy Version</span>
                      </IBlueBorderButton>
                    ) : (
                      <RevisionIndicator
                        status={
                          assistantProviderAction.assistant?.getAssistantproviderid() ===
                          apm.getAssistantproviderwebsocket()?.getId()
                            ? 'DEPLOYED'
                            : 'NOT_DEPLOYED'
                        }
                      />
                    )}
                  </TableCell>
                </TableRow>
              );
          }
        })}
      </ScrollableResizableTable>
    </TableSection>
  );
}
