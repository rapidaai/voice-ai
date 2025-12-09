import { PageHeaderBlock } from '@/app/components/blocks/page-header-block';
import { PageTitleBlock } from '@/app/components/blocks/page-title-block';
import { YellowNoticeBlock } from '@/app/components/container/message/notice-block';
import { FormLabel } from '@/app/components/form-label';
import {
  IBlueBorderPlusButton,
  IBlueButton,
  IButton,
} from '@/app/components/form/button';
import { CopyButton } from '@/app/components/form/button/copy-button';
import { FieldSet } from '@/app/components/form/fieldset';
import { Helmet } from '@/app/components/helmet';
import { useCurrentCredential } from '@/hooks/use-credential';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import {
  Bug,
  Code,
  ExternalLink,
  Eye,
  Globe,
  Info,
  Phone,
  Plus,
  RotateCw,
  Speech,
} from 'lucide-react';
import { FC, useCallback, useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { Speaker } from 'lucide-react';
import {
  Assistant,
  AssistantDefinition,
  ConnectionConfig,
  DeploymentAudioProvider,
  GetAssistant,
  GetAssistantRequest,
} from '@rapidaai/react';
import toast from 'react-hot-toast/headless';
import { connectionConfig } from '@/configs';
import { ProviderPill } from '@/app/components/pill/provider-model-pill';
import { PlusIcon } from '@/app/components/Icon/plus';
import { Popover } from '@/app/components/popover';
import { ActionableEmptyMessage } from '@/app/components/container/message/actionable-empty-message';
import { toHumanReadableDateTime } from '@/utils/date';
import { InputHelper } from '@/app/components/input-helper';
import { useRapidaStore } from '@/hooks';
import { AssistantPhoneCallDeploymentDialog } from '@/app/components/base/modal/assistant-phone-call-deployment-modal';
import { AssistantDebugDeploymentDialog } from '@/app/components/base/modal/assistant-debug-deployment-modal';
import { AssistantWebWidgetlDeploymentDialog } from '@/app/components/base/modal/assistant-web-widget-deployment-modal';
import { AssistantApiDeploymentDialog } from '@/app/components/base/modal/assistant-api-deployment-modal';
export const ConfigureAssistantDeploymentPage = () => {
  /**
   * assistant ID
   */
  const { assistantId } = useParams();

  /**
   * current assistant
   */
  const [assistant, setAssistant] = useState<Assistant | null>(null);

  /**
   * navigation
   */
  const navi = useGlobalNavigation();

  /**
   * authentication params
   */
  const { token, authId, projectId } = useCurrentCredential();

  /**
   * global loading
   */
  const { showLoader, hideLoader } = useRapidaStore();

  const get = useCallback(assistantId => {
    if (assistantId) {
      showLoader('block');
      const request = new GetAssistantRequest();
      const assistantDef = new AssistantDefinition();
      assistantDef.setAssistantid(assistantId);
      request.setAssistantdefinition(assistantDef);
      GetAssistant(
        connectionConfig,
        request,
        ConnectionConfig.WithDebugger({
          authorization: token,
          userId: authId,
          projectId: projectId,
        }),
      )
        .then(epmr => {
          hideLoader();
          if (epmr?.getSuccess()) {
            let assistant = epmr.getData();
            if (assistant) setAssistant(assistant);
          } else {
            const errorMessage =
              'Unable to get your assistant. please try again later.';
            const error = epmr?.getError();
            if (error) {
              toast.error(error.getHumanmessage());
              return;
            }
            toast.error(errorMessage);
            return;
          }
        })
        .catch(err => {
          hideLoader();
        });
    }
  }, []);

  useEffect(() => {
    get(assistantId);
  }, [assistantId]);

  const [isExpanded, setIsExpanded] = useState(false);
  const [isApiExpanded, setIsApiExpanded] = useState(false);
  const [isPhoneExpanded, setIsPhoneExpanded] = useState(false);

  const [isWidgetExpanded, setIsWidgetExpanded] = useState(false);
  const [createDeploymentPopover, setCreateDeploymentPopover] = useState(false);
  const [actionCreateDeploymentPopover, setActionCreateDeploymentPopover] =
    useState(false);
  /**
   *
   */
  return (
    <div className="flex flex-col w-full flex-1 overflow-auto bg-white dark:bg-gray-900">
      {assistant?.getPhonedeployment() && (
        <AssistantPhoneCallDeploymentDialog
          modalOpen={isPhoneExpanded}
          setModalOpen={setIsPhoneExpanded}
          deployment={assistant?.getPhonedeployment()!}
        />
      )}

      {assistant?.getDebuggerdeployment() && (
        <AssistantDebugDeploymentDialog
          modalOpen={isExpanded}
          setModalOpen={setIsExpanded}
          deployment={assistant?.getDebuggerdeployment()!}
        />
      )}

      {assistant?.getWebplugindeployment() && (
        <AssistantWebWidgetlDeploymentDialog
          modalOpen={isWidgetExpanded}
          setModalOpen={setIsWidgetExpanded}
          deployment={assistant?.getWebplugindeployment()!}
        />
      )}

      {assistant?.getApideployment() && (
        <AssistantApiDeploymentDialog
          modalOpen={isApiExpanded}
          setModalOpen={setIsApiExpanded}
          deployment={assistant?.getApideployment()!}
        />
      )}
      <Helmet title="Assistant deployment" />
      <PageHeaderBlock>
        <div className="flex items-center gap-3">
          <PageTitleBlock>Deployments</PageTitleBlock>
        </div>
        <div className="flex border-l">
          <IBlueButton
            className="px-4"
            onClick={() => {
              setCreateDeploymentPopover(true);
            }}
          >
            Add new deployment
            <PlusIcon className="w-4 h-4 ml-2" />
          </IBlueButton>
          <Popover
            align={'bottom-end'}
            className="w-72 py-0.5 px-0.5"
            open={createDeploymentPopover}
            setOpen={setCreateDeploymentPopover}
          >
            <div className="space-y-0.5 text-sm/6">
              <IButton
                className="w-full justify-start"
                onClick={() => navi.goToConfigureWeb(assistantId!)}
              >
                <Globe className="w-4 h-4 mr-2" strokeWidth={1.5} />
                <span className="ml-2">Add to Your Website</span>
              </IButton>
              <hr className="w-full h-[1px] bg-gray-300 dark:border-gray-800" />
              <IButton
                className="w-full justify-start"
                onClick={() => navi.goToConfigureApi(assistantId!)}
              >
                <Code className="w-4 h-4 mr-2" strokeWidth={1.5} />
                <span className="ml-2">Integrate with SDK</span>
              </IButton>
              <hr className="w-full h-[1px] bg-gray-300 dark:border-gray-800" />
              <IButton
                className="w-full justify-start"
                onClick={() => navi.goToConfigureCall(assistantId!)}
              >
                <Phone className="w-4 h-4 mr-2" strokeWidth={1.5} />
                <span className="ml-2">Deploy on Phone Call</span>
              </IButton>
              <hr className="w-full h-[1px] bg-gray-300 dark:border-gray-800" />
              <IButton
                className="w-full justify-start"
                onClick={() => navi.goToConfigureDebugger(assistantId!)}
              >
                <Bug className="w-4 h-4 mr-2" strokeWidth={1.5} />
                <span className="ml-2">Deploy for Debugging</span>
              </IButton>
            </div>
          </Popover>
          <IButton
            type="button"
            onClick={() => get(assistantId)}
            className="border-l"
          >
            <RotateCw className="w-4 h-4" strokeWidth={1.5} />
          </IButton>
        </div>
      </PageHeaderBlock>
      <YellowNoticeBlock className="flex items-center">
        <Info className="shrink-0 w-4 h-4" />
        <div className="ms-3 text-sm font-medium">
          Rapida Assistant enables you to deploy intelligent conversational
          agents across multiple channels.
        </div>
        <a
          target="_blank"
          href="https://doc.rapida.ai/assistants/overview"
          className="h-7 flex items-center font-medium hover:underline ml-auto text-yellow-600"
          rel="noreferrer"
        >
          Read documentation
          <ExternalLink className="shrink-0 w-4 h-4 ml-1.5" strokeWidth={1.5} />
        </a>
      </YellowNoticeBlock>
      <div className="flex flex-col gap-2 p-4">
        {/* debugger */}
        {assistant?.hasDebuggerdeployment() && (
          <div className="bg-white dark:bg-gray-900 border">
            <div className="flex items-center justify-between border-b">
              <div className="flex items-center gap-2 justify-between px-4">
                <h3 className="font-semibold truncate">Debugger</h3>
                <span className="text-xs">
                  vrsn_dpl_{assistant.getDebuggerdeployment()?.getId()}
                </span>
              </div>
              <div className="flex shrink-0 border-x divide-x">
                <IButton
                  onClick={() => {
                    navi.goToConfigureDebugger(assistantId!);
                  }}
                >
                  <span className="mr-2">Edit Debugger</span>
                  <Plus className="w-4 h-4 " />
                </IButton>
                <IButton
                  onClick={() => {
                    navi.goToAssistantPreview(assistantId!);
                  }}
                >
                  <span className="mr-2">Preview</span>
                  <ExternalLink className="w-4 h-4 " />
                </IButton>
                <IButton onClick={() => setIsExpanded(!isExpanded)}>
                  <Eye className="w-4 h-4" strokeWidth={1.5} />
                </IButton>
              </div>
            </div>

            <div className="grid grid-cols-2 gap-6 text-sm px-4 py-4 text-muted">
              <FieldSet className="col-span-2">
                <FormLabel>Public Url</FormLabel>
                <div className="flex items-center gap-2">
                  <code className="flex-1 dark:bg-gray-950 bg-gray-100 px-3 py-2 font-mono text-xs min-w-0 overflow-hidden">
                    {`https://www.rapida.ai/public/assistant/${assistantId}?token={{PROJECT_CRDENTIAL_KEY}}`}
                  </code>
                  <div className="flex shrink-0 border divide-x">
                    <CopyButton className="h-7 w-7">
                      {`https://www.rapida.ai/public/assistant/2214276472644829184?token={{PROJECT_CRDENTIAL_KEY}}`}
                    </CopyButton>
                  </div>
                </div>
                <InputHelper>
                  You can add all the additional agent arguments in query
                  parameters for example if you are expecting argument
                  <code className="text-red-600">`name`</code>
                  add <code className="text-red-600">`?name=your-name`</code>
                </InputHelper>
              </FieldSet>
              <div className="space-y-4">
                <div>
                  <div className="text-muted-foreground">Input Mode</div>
                  <div className="font-medium">
                    Text
                    {assistant?.getDebuggerdeployment()?.getInputaudio() &&
                      ', Audio'}
                  </div>
                </div>
                <div>
                  <div className="text-muted-foreground">Output Mode</div>
                  <div className="font-medium">
                    Text
                    {assistant?.getDebuggerdeployment()?.getOutputaudio() &&
                      ', Audio'}
                  </div>
                </div>
              </div>

              <div className="space-y-4">
                <div>
                  <div className="text-muted-foreground">Updated on</div>
                  <div className="font-medium">
                    {toHumanReadableDateTime(
                      assistant.getDebuggerdeployment()?.getCreateddate()!,
                    )}
                  </div>
                </div>
                <div>
                  <div className="text-muted-foreground">Version</div>
                  <div className="font-medium">
                    vrsn_dpl_{assistant.getDebuggerdeployment()?.getId()}
                  </div>
                </div>
              </div>
            </div>
          </div>
        )}
        {/* phone */}
        {assistant?.hasApideployment() && (
          <div className="bg-white dark:bg-gray-900 border">
            <div className="flex items-center justify-between border-b">
              <div className="flex items-center gap-2 justify-between px-4">
                <h3 className="font-semibold truncate">API</h3>
                <span className="text-xs">
                  vrsn_dpl_{assistant.getApideployment()?.getId()}
                </span>
              </div>
              <div className="flex shrink-0 border-x divide-x">
                <IButton
                  onClick={() => {
                    navi.goToConfigureApi(assistantId!);
                  }}
                >
                  <span className="mr-2">Edit Api</span>
                  <Plus className="w-4 h-4 " />
                </IButton>
                <IButton onClick={() => setIsApiExpanded(!isApiExpanded)}>
                  <span className="mr-2">Instruction</span>
                  <Code className="w-4 h-4 " />
                </IButton>
              </div>
            </div>
            <div className="grid grid-cols-2 gap-6 text-sm px-4 py-4 text-muted">
              <div className="space-y-4">
                <div>
                  <div className="text-muted-foreground">SDK</div>
                  <div className="font-medium capitalize text-primary">
                    React
                  </div>
                </div>

                <div>
                  <div className="text-muted-foreground">Deployment type</div>
                  <div className="font-medium">Global Standard</div>
                </div>

                <div>
                  <div className="text-muted-foreground">Input Mode</div>
                  <div className="font-medium">
                    Text
                    {assistant?.getApideployment()?.getInputaudio() &&
                      ', Audio'}
                  </div>
                </div>
                <div>
                  <div className="text-muted-foreground">Output Mode</div>
                  <div className="font-medium">
                    Text
                    {assistant?.getApideployment()?.getOutputaudio() &&
                      ', Audio'}
                  </div>
                </div>

                <div>
                  <div className="text-muted-foreground">Concurrency</div>
                  <div className="font-medium">10</div>
                </div>
              </div>

              <div className="space-y-4">
                <div>
                  <div className="text-muted-foreground">Updated on</div>
                  <div className="font-medium">
                    {toHumanReadableDateTime(
                      assistant.getApideployment()?.getCreateddate()!,
                    )}
                  </div>
                </div>
                <div>
                  <div className="text-muted-foreground">Version</div>
                  <div className="font-medium">
                    vrsn_dpl_{assistant.getApideployment()?.getId()}
                  </div>
                </div>
                <div>
                  <div className="text-muted-foreground">
                    Version upgrade policy
                  </div>
                  <div className="font-medium">
                    Once a new default version is available
                  </div>
                </div>
              </div>
            </div>
          </div>
        )}
        {assistant?.hasPhonedeployment() && (
          <div className="bg-white dark:bg-gray-900 border">
            <div className="flex items-center justify-between border-b">
              <div className="flex items-center gap-2 justify-between px-4">
                <h3 className="font-semibold truncate">Phone Call</h3>
                <span className="text-xs">
                  vrsn_dpl_{assistant.getPhonedeployment()?.getId()}
                </span>
              </div>

              <div className="flex shrink-0 border-x divide-x">
                <IButton
                  onClick={() => {
                    navi.goToConfigureCall(assistantId!);
                  }}
                >
                  <span className="mr-2">Edit Phone Call</span>
                  <Plus className="w-4 h-4 " />
                </IButton>
                <IButton
                  onClick={() => {
                    navi.goToAssistantPreviewCall(assistantId!);
                  }}
                >
                  <span className="mr-2">Preview</span>
                  <ExternalLink className="w-4 h-4 " />
                </IButton>
                <IButton onClick={() => setIsPhoneExpanded(!isPhoneExpanded)}>
                  <span className="mr-2">Inbound Instruction</span>
                  <Code className="w-4 h-4 " />
                </IButton>
              </div>
            </div>
            {/*  */}
            <div className="grid grid-cols-2 gap-6 text-sm px-4 py-4 text-muted">
              <div className="space-y-4">
                <div>
                  <div className="text-muted-foreground">Telephony</div>
                  <div className="font-medium capitalize text-primary">
                    {assistant.getPhonedeployment()?.getPhoneprovidername()}
                  </div>
                </div>

                <div>
                  <div className="text-muted-foreground">Deployment type</div>
                  <div className="font-medium">Global Standard</div>
                </div>

                <div>
                  <div className="text-muted-foreground">Input Mode</div>
                  <div className="font-medium">Audio</div>
                </div>

                <div>
                  <div className="text-muted-foreground">Output Mode</div>
                  <div className="font-medium">Audio</div>
                </div>

                <div>
                  <div className="text-muted-foreground">Concurrency</div>
                  <div className="font-medium">10</div>
                </div>
              </div>

              <div className="space-y-4">
                <div>
                  <div className="text-muted-foreground">Updated on</div>
                  <div className="font-medium">
                    {toHumanReadableDateTime(
                      assistant.getPhonedeployment()?.getCreateddate()!,
                    )}
                  </div>
                </div>
                <div>
                  <div className="text-muted-foreground">Version</div>
                  <div className="font-medium">
                    vrsn_dpl_{assistant.getPhonedeployment()?.getId()}
                  </div>
                </div>
                <div>
                  <div className="text-muted-foreground">
                    Version upgrade policy
                  </div>
                  <div className="font-medium">
                    Once a new default version is available
                  </div>
                </div>
              </div>
            </div>
            {/*  */}
          </div>
        )}
        {/* web widget */}
        {assistant?.hasWebplugindeployment() && (
          <div className="bg-white dark:bg-gray-900 border">
            <div className="flex items-center justify-between border-b">
              <div className="flex items-center gap-2 justify-between px-4">
                <h3 className="font-semibold truncate">Web widget</h3>
                <span className="text-xs">
                  vrsn_dpl_{assistant.getWebplugindeployment()?.getId()}
                </span>
              </div>
              <div className="flex shrink-0 border-x divide-x">
                <IButton
                  onClick={() => {
                    navi.goToConfigureWeb(assistantId!);
                  }}
                >
                  <span className="mr-2">Edit Widget</span>
                  <Plus className="w-4 h-4 " />
                </IButton>
                <IButton onClick={() => setIsWidgetExpanded(!isWidgetExpanded)}>
                  <span className="mr-2">Instruction</span>
                  <Code className="w-4 h-4 " />
                </IButton>
              </div>
            </div>
            <div className="grid grid-cols-2 gap-6 text-sm px-4 py-4 text-muted">
              <div className="space-y-4">
                <div>
                  <div className="text-muted-foreground">SDK</div>
                  <div className="font-medium capitalize text-primary">JS</div>
                </div>

                <div>
                  <div className="text-muted-foreground">Deployment type</div>
                  <div className="font-medium">Global Standard</div>
                </div>

                <div>
                  <div className="text-muted-foreground">Input Mode</div>
                  <div className="font-medium">
                    Text
                    {assistant?.getWebplugindeployment()?.getInputaudio() &&
                      ', Audio'}
                  </div>
                </div>
                <div>
                  <div className="text-muted-foreground">Output Mode</div>
                  <div className="font-medium">
                    Text
                    {assistant?.getWebplugindeployment()?.getOutputaudio() &&
                      ', Audio'}
                  </div>
                </div>

                <div>
                  <div className="text-muted-foreground">Concurrency</div>
                  <div className="font-medium">10</div>
                </div>
              </div>

              <div className="space-y-4">
                <div>
                  <div className="text-muted-foreground">Updated on</div>
                  <div className="font-medium">
                    {toHumanReadableDateTime(
                      assistant.getWebplugindeployment()?.getCreateddate()!,
                    )}
                  </div>
                </div>
                <div>
                  <div className="text-muted-foreground">Version</div>
                  <div className="font-medium">
                    vrsn_dpl_{assistant.getWebplugindeployment()?.getId()}
                  </div>
                </div>
                <div>
                  <div className="text-muted-foreground">
                    Version upgrade policy
                  </div>
                  <div className="font-medium">
                    Once a new default version is available
                  </div>
                </div>
              </div>
            </div>
          </div>
        )}
      </div>
      {!assistant?.getApideployment() &&
        !assistant?.getWebplugindeployment() &&
        !assistant?.getDebuggerdeployment() &&
        !assistant?.getPhonedeployment() && (
          <div className="flex flex-1 w-full justify-center items-center">
            <ActionableEmptyMessage
              title="No Deployment"
              subtitle="There are no assistant deployments found."
              actionComponent={
                <div className="relative mt-3">
                  <IBlueBorderPlusButton
                    className="px-4 bg-white"
                    onClick={() => {
                      setActionCreateDeploymentPopover(true);
                    }}
                  >
                    Add new deployment
                  </IBlueBorderPlusButton>
                  <Popover
                    align={'bottom-end'}
                    className="w-72 py-0.5 px-0.5"
                    open={actionCreateDeploymentPopover}
                    setOpen={setActionCreateDeploymentPopover}
                  >
                    <div className="space-y-0.5 text-sm/6">
                      <IButton
                        className="w-full justify-start"
                        onClick={() => navi.goToConfigureWeb(assistantId!)}
                      >
                        <Globe className="w-4 h-4 mr-2" strokeWidth={1.5} />
                        <span className="ml-2">Add to Your Website</span>
                      </IButton>
                      <hr className="w-full h-[1px] bg-gray-300 dark:border-gray-800" />
                      <IButton
                        className="w-full justify-start"
                        onClick={() => navi.goToConfigureApi(assistantId!)}
                      >
                        <Code className="w-4 h-4 mr-2" strokeWidth={1.5} />
                        <span className="ml-2">Integrate with SDK</span>
                      </IButton>
                      <hr className="w-full h-[1px] bg-gray-300 dark:border-gray-800" />
                      <IButton
                        className="w-full justify-start"
                        onClick={() => navi.goToConfigureCall(assistantId!)}
                      >
                        <Phone className="w-4 h-4 mr-2" strokeWidth={1.5} />
                        <span className="ml-2">Deploy on Phone Call</span>
                      </IButton>
                      <hr className="w-full h-[1px] bg-gray-300 dark:border-gray-800" />
                      <IButton
                        className="w-full justify-start"
                        onClick={() => navi.goToConfigureDebugger(assistantId!)}
                      >
                        <Bug className="w-4 h-4 mr-2" strokeWidth={1.5} />
                        <span className="ml-2">Deploy for Debugging</span>
                      </IButton>
                    </div>
                  </Popover>
                </div>
              }
            />
          </div>
        )}
    </div>
  );
};

// Helper components

const VoiceInput: FC<{ deployment?: DeploymentAudioProvider }> = ({
  deployment,
}) => (
  <div className="bg-gray-50 dark:bg-gray-950">
    <div className="flex items-center space-x-2 border-b py-1 px-4 h-10">
      <Speech className="w-4 h-4" />
      <h4 className="font-medium">Speech to text</h4>
    </div>
    {deployment?.getAudiooptionsList() ? (
      deployment?.getAudiooptionsList().length > 0 && (
        <div className="text-xs text-gray-500 dark:text-gray-400 py-3 px-3 space-y-6">
          <FieldSet>
            <FormLabel>Provider</FormLabel>
            <ProviderPill provider={deployment?.getAudioprovider()} />
          </FieldSet>
          <div className="grid grid-cols-2 gap-4">
            {deployment
              ?.getAudiooptionsList()
              .filter(d => d.getValue())
              .filter(d => d.getKey().startsWith('listen.'))
              .map((detail, index) => (
                <FieldSet key={index}>
                  <FormLabel>{detail.getKey()}</FormLabel>
                  <div className="flex items-center gap-2">
                    <code className="flex-1 dark:bg-gray-900 bg-gray-100 px-3 py-2 font-mono text-xs min-w-0 overflow-hidden">
                      {detail.getValue()}
                    </code>
                    <div className="flex shrink-0 border divide-x">
                      <CopyButton className="h-7 w-7">
                        {detail.getValue()}
                      </CopyButton>
                    </div>
                  </div>
                </FieldSet>
              ))}
          </div>
        </div>
      )
    ) : (
      <YellowNoticeBlock>Voice input is not enabled</YellowNoticeBlock>
    )}
  </div>
);

const VoiceOutput: FC<{ deployment?: DeploymentAudioProvider }> = ({
  deployment,
}) => (
  <div className="bg-gray-50 dark:bg-gray-950">
    <div className="flex items-center space-x-2 border-b py-2 px-4  h-10">
      <Speaker className="w-4 h-4" />
      <h4 className="font-medium">Text to speech</h4>
    </div>
    {deployment?.getAudiooptionsList() ? (
      deployment?.getAudiooptionsList().length > 0 && (
        <div className="text-xs text-gray-500 dark:text-gray-400 py-3 px-3 space-y-6">
          <FieldSet>
            <FormLabel>Provider</FormLabel>
            <ProviderPill provider={deployment?.getAudioprovider()} />
          </FieldSet>
          <div className="grid grid-cols-2 gap-4">
            {deployment
              ?.getAudiooptionsList()
              .filter(d => d.getValue())
              .filter(d => d.getKey().startsWith('speak.'))
              .map((detail, index) => (
                <FieldSet key={index}>
                  <FormLabel>{detail.getKey()}</FormLabel>
                  <div className="flex items-center gap-2">
                    <code className="flex-1 dark:bg-gray-900 bg-gray-100 px-3 py-2 font-mono text-xs min-w-0 overflow-hidden">
                      {detail.getValue()}
                    </code>

                    <div className="flex shrink-0 border divide-x">
                      <CopyButton className="h-7 w-7">
                        {detail.getValue()}
                      </CopyButton>
                    </div>
                  </div>
                </FieldSet>
              ))}
          </div>
        </div>
      )
    ) : (
      <YellowNoticeBlock>Voice output is not enabled</YellowNoticeBlock>
    )}
  </div>
);
