import { PageHeaderBlock } from '@/app/components/blocks/page-header-block';
import { PageTitleBlock } from '@/app/components/blocks/page-title-block';
import { YellowNoticeBlock } from '@/app/components/container/message/notice-block';
import { FormLabel } from '@/app/components/form-label';
import {
  IBlueBorderPlusButton,
  IBlueButton,
  IButton,
} from '@/app/components/Form/Button';
import { CopyButton } from '@/app/components/Form/Button/copy-button';
import { FieldSet } from '@/app/components/Form/Fieldset';
import { Helmet } from '@/app/components/Helmet';
import { useCurrentCredential } from '@/hooks/use-credential';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import {
  Bug,
  ChevronDown,
  ChevronUp,
  Code,
  ExternalLink,
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
import { ProviderPill } from '@/app/components/Pill/provider-model-pill';
import { AnimatePresence, motion } from 'framer-motion';
import { PlusIcon } from '@/app/components/Icon/plus';
import { Popover } from '@/app/components/Popover';
import { ActionableEmptyMessage } from '@/app/components/container/message/actionable-empty-message';
import { cn, toHumanReadableDateTime } from '@/styles/media';
import { InputHelper } from '@/app/components/input-helper';
import { CodeHighlighting } from '@/app/components/code-highlighting';
import { useRapidaStore } from '@/hooks';
import { PaginationButtonBlock } from '@/app/components/blocks/pagination-button-block';

export const ConfigureAssistantDeploymentPage = () => {
  const { assistantId } = useParams();
  const [assistant, setAssistant] = useState<Assistant | null>(null);
  const navi = useGlobalNavigation();
  const { token, authId, projectId } = useCurrentCredential();
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
  /**
   *
   */
  useEffect(() => {
    get(assistantId);
  }, [assistantId]);

  const [isExpanded, setIsExpanded] = useState(false);
  const [isApiExpanded, setIsApiExpanded] = useState(false);
  const [isPhoneExpanded, setIsPhoneExpanded] = useState(false);
  const [isPhoneInboundCodeExpanded, setIsPhoneInboundCodeExpanded] =
    useState(false);
  //
  const [isWidgetExpanded, setIsWidgetExpanded] = useState(false);
  const [isWidgetCodeExpanded, setIsWidgetCodeExpanded] = useState(false);
  const [createDeploymentPopover, setCreateDeploymentPopover] = useState(false);
  const [actionCreateDeploymentPopover, setActionCreateDeploymentPopover] =
    useState(false);
  /**
   *
   */
  return (
    <div className="flex flex-col w-full flex-1 overflow-auto">
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
            className="w-auto"
            open={createDeploymentPopover}
            setOpen={setCreateDeploymentPopover}
          >
            <div className="space-y-0.5">
              <IButton
                className="w-full justify-start"
                onClick={() => navi.goToConfigureWeb(assistantId!)}
              >
                <Globe className="w-4 h-4 mr-2" /> Create new web widget
              </IButton>
              <IButton
                className="w-full justify-start"
                onClick={() => navi.goToConfigureApi(assistantId!)}
              >
                <Code className="w-4 h-4 mr-2" /> Create new api deployment
              </IButton>
              <IButton
                className="w-full justify-start"
                onClick={() => navi.goToConfigureCall(assistantId!)}
              >
                <Phone className="w-4 h-4 mr-2" /> Create new phone call
                deployment
              </IButton>
              <IButton
                className="w-full justify-start"
                onClick={() => navi.goToConfigureDebugger(assistantId!)}
              >
                <Bug className="w-4 h-4 mr-2" /> Create new debugger deployement
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
          <div className="bg-white dark:bg-gray-950 border">
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
                  {isExpanded ? (
                    <ChevronUp className="w-4 h-4" />
                  ) : (
                    <ChevronDown className="w-4 h-4" />
                  )}
                </IButton>
              </div>
            </div>

            <div className="grid grid-cols-2 gap-6 text-sm px-4 py-4">
              <FieldSet className="col-span-2">
                <FormLabel>Public Url</FormLabel>
                <div className="flex items-center gap-2">
                  <code className="flex-1 dark:bg-gray-900 bg-gray-100 px-3 py-2 font-mono text-xs min-w-0 overflow-hidden">
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
            <AnimatePresence>
              <motion.div
                initial={{ height: 0 }} // h-10 is 0px in most Tailwind configurations
                animate={{ height: isExpanded ? 'auto' : 0 }}
                transition={{ duration: 0.3 }}
                className="grid grid-cols-2 divide-x border-t overflow-hidden"
              >
                <VoiceInput
                  deployment={assistant
                    .getDebuggerdeployment()
                    ?.getInputaudio()}
                />

                <VoiceOutput
                  deployment={assistant
                    .getDebuggerdeployment()
                    ?.getOutputaudio()}
                />
              </motion.div>
            </AnimatePresence>
          </div>
        )}
        {/* phone */}
        {assistant?.hasApideployment() && (
          <div className="bg-white dark:bg-gray-950 border">
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
                <IButton
                  onClick={() => {
                    // navi.goToConfigureWeb(assistantId!);
                  }}
                >
                  <span className="mr-2">Instruction</span>
                  <Code className="w-4 h-4 " />
                </IButton>
                <IButton onClick={() => setIsApiExpanded(!isApiExpanded)}>
                  {isApiExpanded ? (
                    <ChevronUp className="w-4 h-4" />
                  ) : (
                    <ChevronDown className="w-4 h-4" />
                  )}
                </IButton>
              </div>
            </div>
            <div className="grid grid-cols-2 gap-6 text-sm px-4 py-4">
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
            <AnimatePresence>
              <motion.div
                initial={{ height: 0 }} // h-10 is 40px in most Tailwind configurations
                animate={{ height: isApiExpanded ? 'auto' : 0 }}
                transition={{ duration: 0.3 }}
                className="grid grid-cols-2 divide-x border-t overflow-hidden"
              >
                <VoiceInput
                  deployment={assistant.getApideployment()?.getInputaudio()}
                />

                <VoiceOutput
                  deployment={assistant.getApideployment()?.getOutputaudio()}
                />
              </motion.div>
            </AnimatePresence>
          </div>
        )}
        {assistant?.hasPhonedeployment() && (
          <div className="bg-white dark:bg-gray-950 border">
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
                <IButton
                  onClick={() => {
                    setIsPhoneInboundCodeExpanded(!isPhoneInboundCodeExpanded);
                  }}
                >
                  <span className="mr-2">Inbound Instruction</span>
                  <Code className="w-4 h-4 " />
                </IButton>
                <IButton onClick={() => setIsPhoneExpanded(!isPhoneExpanded)}>
                  {isPhoneExpanded ? (
                    <ChevronUp className="w-4 h-4" />
                  ) : (
                    <ChevronDown className="w-4 h-4" />
                  )}
                </IButton>
              </div>
            </div>
            {/*  */}
            <div className="grid grid-cols-2 gap-6 text-sm px-4 py-4">
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
            <AnimatePresence>
              <motion.div
                initial={{ height: 0, opacity: 0, zIndex: -1 }}
                animate={{
                  height: isPhoneInboundCodeExpanded ? 'auto' : 0,
                  opacity: isPhoneInboundCodeExpanded ? 1 : 0,
                }}
                transition={{ duration: 0.3 }}
                className={cn(
                  'overflow-hidden',
                  isPhoneInboundCodeExpanded && 'space-y-6 p-4 border-t',
                )}
              >
                <FieldSet className="col-span-2">
                  <FormLabel>Inbound webhook url</FormLabel>
                  <div className="flex items-center gap-2">
                    <code className="flex-1 dark:bg-gray-900 bg-gray-100 px-3 py-2 font-mono text-xs min-w-0 overflow-hidden">
                      {`https://assistant-01.rapida.ai/v1/talk/twilio/call/${assistantId}?x-api-key={{PROJECT_CRDENTIAL_KEY}}`}
                    </code>
                    <div className="flex shrink-0 border divide-x">
                      <CopyButton className="h-7 w-7">
                        {`https://assistant-01.rapida.ai/v1/talk/twilio/call/${assistantId}?x-api-key={{PROJECT_CRDENTIAL_KEY}}`}
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
              </motion.div>
            </AnimatePresence>
            <AnimatePresence>
              <motion.div
                initial={{ height: 0 }} // h-10 is 40px in most Tailwind configurations
                animate={{ height: isPhoneExpanded ? 'auto' : 0 }}
                transition={{ duration: 0.3 }}
                className="grid grid-cols-2 divide-x border-t overflow-hidden"
              >
                <VoiceInput
                  deployment={assistant.getPhonedeployment()?.getInputaudio()}
                />

                <VoiceOutput
                  deployment={assistant.getPhonedeployment()?.getOutputaudio()}
                />
              </motion.div>
            </AnimatePresence>
          </div>
        )}
        {/* web widget */}
        {assistant?.hasWebplugindeployment() && (
          <div className="bg-white dark:bg-gray-950 border">
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
                <IButton
                  onClick={() => {
                    setIsWidgetCodeExpanded(!isWidgetCodeExpanded);
                  }}
                >
                  <span className="mr-2">Instruction</span>
                  <Code className="w-4 h-4 " />
                </IButton>
                <IButton onClick={() => setIsWidgetExpanded(!isWidgetExpanded)}>
                  {isWidgetExpanded ? (
                    <ChevronUp className="w-4 h-4" />
                  ) : (
                    <ChevronDown className="w-4 h-4" />
                  )}
                </IButton>
              </div>
            </div>
            <div className="grid grid-cols-2 gap-6 text-sm px-4 py-4">
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
            <AnimatePresence>
              <motion.div
                initial={{ height: 0, opacity: 0 }}
                animate={{
                  height: isWidgetCodeExpanded ? 'auto' : 0,
                  opacity: isWidgetCodeExpanded ? 1 : 0,
                }}
                transition={{ duration: 0.3 }}
                className={cn(isWidgetCodeExpanded && 'space-y-6 p-4 border-t')}
              >
                <FieldSet>
                  <div className="text-muted-foreground">
                    Add the Rapida.js script to your HTML
                  </div>
                  <CodeHighlighting code='<script src="https://cdn-01.rapida.ai/public/scripts/app.min.js" defer></script>'></CodeHighlighting>
                </FieldSet>
                <FieldSet>
                  <div className="text-muted-foreground">
                    Add the chatbot configuration script
                  </div>
                  <CodeHighlighting
                    code={`
                        <script>window.chatbotConfig = {
        theme: {
          color: "black",
        },
        assistant_id: "2139456643765633024",
        token:
          "5e04b962dfd988ca0f4ac9e097f17841ebf34d7e38c50297e245c12bddce2117",
        user: {
          id: "ayan-global-user",
          name: "Guest",
        }
      }</script>`.trim()}
                  ></CodeHighlighting>
                </FieldSet>
              </motion.div>
            </AnimatePresence>
            <AnimatePresence>
              <motion.div
                initial={{ height: 0 }} // h-10 is 0px in most Tailwind configurations
                animate={{ height: isWidgetExpanded ? 'auto' : 0 }}
                transition={{ duration: 0.3 }}
                className="grid grid-cols-2 divide-x border-t overflow-hidden"
              >
                <VoiceInput
                  deployment={assistant
                    .getWebplugindeployment()
                    ?.getInputaudio()}
                />

                <VoiceOutput
                  deployment={assistant
                    .getWebplugindeployment()
                    ?.getOutputaudio()}
                />
              </motion.div>
            </AnimatePresence>
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
              //   action="Create deployment"
              //   onActionClick={() => navi.goToConfigureDebugger(assistantId!)}
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
                    className="w-auto"
                    open={actionCreateDeploymentPopover}
                    setOpen={setActionCreateDeploymentPopover}
                  >
                    <div className="space-y-0.5">
                      <IButton
                        className="w-full justify-start"
                        onClick={() => navi.goToConfigureWeb(assistantId!)}
                      >
                        <Globe className="w-4 h-4 mr-2" /> Create new web widget
                      </IButton>
                      <IButton
                        className="w-full justify-start"
                        onClick={() => navi.goToConfigureApi(assistantId!)}
                      >
                        <Code className="w-4 h-4 mr-2" /> Create new api
                        deployment
                      </IButton>
                      <IButton
                        className="w-full justify-start"
                        onClick={() => navi.goToConfigureCall(assistantId!)}
                      >
                        <Phone className="w-4 h-4 mr-2" /> Create new phone call
                        deployment
                      </IButton>
                      <IButton
                        className="w-full justify-start"
                        onClick={() => navi.goToConfigureDebugger(assistantId!)}
                      >
                        <Bug className="w-4 h-4 mr-2" /> Create new debugger
                        deployement
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
            <ProviderPill providerId={deployment?.getAudioproviderid()} />
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
            <ProviderPill providerId={deployment?.getAudioproviderid()} />
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
