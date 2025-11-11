import { useState } from 'react';
import { Helmet } from '@/app/components/Helmet';
import { useRapidaStore } from '@/hooks';
import { TabForm } from '@/app/components/Form/tab-form';
import {
  IBlueBGArrowButton,
  IBlueBorderButton,
  ICancelButton,
} from '@/app/components/Form/Button';
import {
  Assistant,
  CreateAssistant,
  CreateAssistantProviderRequest,
  CreateAssistantRequest,
  GetAssistantResponse,
} from '@rapidaai/react';
import { useConfirmDialog } from '@/app/pages/assistant/actions/hooks/use-confirmation';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { useCurrentCredential } from '@/hooks/use-credential';
import { randomMeaningfullName } from '@/utils';
import { FieldSet } from '@/app/components/Form/Fieldset';
import { FormLabel } from '@/app/components/form-label';
import { Input } from '@/app/components/Form/Input';
import { Textarea } from '@/app/components/Form/Textarea';
import { TagInput } from '@/app/components/Form/tag-input';
import { AssistantTag } from '@/app/components/Form/tag-input/assistant-tags';
import {
  Bug,
  ChevronRight,
  Code,
  ExternalLink,
  Info,
  PhoneCall,
} from 'lucide-react';
import { YellowNoticeBlock } from '@/app/components/container/message/notice-block';
import { Phone, Globe } from 'lucide-react';
import { InputGroup } from '@/app/components/input-group';
import { APiParameter } from '@/app/components/external-api/api-parameter';
import { connectionConfig } from '@/configs';
import toast from 'react-hot-toast';

export function CreateWebsocket() {
  const { authId, token, projectId } = useCurrentCredential();
  const {
    goToAssistant,
    goToConfigureDebugger,
    goToConfigureWeb,
    goToConfigureCall,
    goToConfigureApi,
    goToCreateAssistantAnalysis,
    goToCreateAssistantWebhook,
  } = useGlobalNavigation();
  const [assistant, setAssistant] = useState<null | Assistant>(null);
  const [activeTab, setActiveTab] = useState<
    'configure-websocket' | 'define-assistant' | 'deployment'
  >('configure-websocket');
  const [errorMessage, setErrorMessage] = useState('');
  const [name, setName] = useState(randomMeaningfullName('assistant'));
  const [description, setDescription] = useState('');
  const [tags, setTags] = useState<string[]>([]);
  const onAddTag = (tag: string) => {
    setTags([...tags, tag]);
  };
  const onRemoveTag = (tag: string) => {
    setTags(tags.filter(t => t !== tag));
  };

  const [websocketUrl, setWebscoketUrl] = useState('');
  const [headers, setHeaders] = useState<{ key: string; value: string }[]>([
    { key: '', value: '' },
  ]);
  const [parameters, setParameters] = useState<
    { key: string; value: string }[]
  >([{ key: '', value: '' }]);

  const validateWebsocket = (): boolean => {
    setErrorMessage('');
    if (!websocketUrl.trim()) {
      setErrorMessage('Please provide a valid websocket url.');
      return false; // websocketUrl must not be empty or whitespace
    }

    const websocketPattern = /^wss?:\/\/[\w.-]+(:\d+)?(\/.*)?$/i;
    if (!websocketPattern.test(websocketUrl.trim())) {
      setErrorMessage(
        'Please provide a valid WebSocket URL (ws:// or wss://).',
      );
      return false; // websocketUrl must match websocket URL structure
    }

    // Validate headers: key and value must not be empty if the array isn't empty
    for (const header of headers) {
      if (!header.key.trim() || !header.value.trim()) {
        setErrorMessage(
          'Please provide valid values for headers key and value.',
        );
        return false;
      }
    }

    // Validate parameters: key and value must not be empty if the array isn't empty
    for (const param of parameters) {
      if (!param.key.trim() || !param.value.trim()) {
        setErrorMessage(
          'Please provide valid values for parameters key and value.',
        );
        return false;
      }
    }
    return true;
  };

  //
  const { showDialog, ConfirmDialogComponent } = useConfirmDialog({});
  const { loading, showLoader, hideLoader } = useRapidaStore();
  let navigator = useGlobalNavigation();

  const createAssistant = () => {
    showLoader('overlay');
    if (!name) {
      setErrorMessage('Please provide a valid name for assistant.');
      return false;
    }

    // Create assistant provider model
    const assistantProvider = new CreateAssistantProviderRequest();
    const assistantWebsocket =
      new CreateAssistantProviderRequest.CreateAssistantProviderWebsocket();
    assistantWebsocket.setWebsocketurl(websocketUrl);

    // adding header
    headers.forEach(p => {
      assistantWebsocket.getHeadersMap().set(p.key, p.value);
    });
    // connection parameters
    parameters.forEach(p => {
      assistantWebsocket.getConnectionparametersMap().set(p.key, p.value);
    });
    assistantProvider.setWebsocket(assistantWebsocket);
    const request = new CreateAssistantRequest();
    request.setAssistantprovider(assistantProvider);
    request.setName(name);
    request.setTagsList(tags);
    request.setDescription(description);
    CreateAssistant(connectionConfig, request, {
      authorization: token,
      'x-auth-id': authId,
      'x-project-id': projectId,
    })
      .then((car: GetAssistantResponse) => {
        hideLoader();
        if (car?.getSuccess()) {
          let ast = car.getData();
          if (ast) {
            toast.success(
              'Assistant Created Successfully, Your AI assistant is ready to be deployed.',
            );
            setAssistant(ast);
            setActiveTab('deployment');
          }
        } else {
          const errorMessage =
            'Unable to create assistant. please try again later.';
          const error = car?.getError();
          if (error) {
            setErrorMessage(error.getHumanmessage());
            return;
          }
          setErrorMessage(errorMessage);
          return;
        }
      })
      .catch(er => {
        hideLoader();
        const errorMessage =
          'Unable to create assistant. please try again later.';
        setErrorMessage(errorMessage);
        return;
      });
  };

  return (
    <>
      <Helmet title="Create an assistant"></Helmet>
      <ConfirmDialogComponent />

      <TabForm
        formHeading="Complete all steps to connect with a websocket"
        activeTab={activeTab}
        onChangeActiveTab={() => {}}
        errorMessage={errorMessage}
        form={[
          {
            name: 'Configuration',
            description: 'Configure and connect the agent using a websocket',
            code: 'configure-websocket',
            body: (
              <div className="">
                <YellowNoticeBlock className="flex items-center">
                  <Info className="shrink-0 w-4 h-4" />
                  <div className="ms-3 text-sm font-medium">
                    Rapida Assistant enables you to deploy intelligent
                    conversational agents across multiple channels.
                  </div>
                  <a
                    target="_blank"
                    href="https://doc.rapida.ai/assistants/overview"
                    className="h-7 flex items-center font-medium hover:underline ml-auto text-yellow-600"
                    rel="noreferrer"
                  >
                    Read documentation
                    <ExternalLink
                      className="shrink-0 w-4 h-4 ml-1.5"
                      strokeWidth={1.5}
                    />
                  </a>
                </YellowNoticeBlock>
                <div className="space-y-6 p-8 max-w-4xl">
                  <FieldSet className="relative w-full">
                    <FormLabel>Websocket Endpoint</FormLabel>
                    <Input
                      placeholder="wss://your-agent-server.com/ws"
                      value={websocketUrl}
                      onChange={v => {
                        setWebscoketUrl(v.target.value);
                      }}
                    />
                  </FieldSet>
                  <FieldSet>
                    <FormLabel>Headers</FormLabel>
                    <APiParameter
                      initialValues={headers}
                      setParameterValue={h => {
                        setHeaders(h);
                      }}
                      actionButtonLabel="Add Header"
                      inputClass="bg-white dark:bg-gray-950!"
                    />
                  </FieldSet>
                  <FieldSet>
                    <FormLabel>Connection Parameters</FormLabel>
                    <APiParameter
                      initialValues={parameters}
                      setParameterValue={v => {
                        setParameters(v);
                      }}
                      actionButtonLabel="Add Parameter"
                      inputClass="bg-white dark:bg-gray-950"
                    />
                  </FieldSet>
                </div>
              </div>
            ),
            actions: [
              <ICancelButton
                className="px-4 rounded-[2px]"
                onClick={() => showDialog(navigator.goBack)}
              >
                Cancel
              </ICancelButton>,
              <IBlueBGArrowButton
                type="button"
                isLoading={loading}
                className="px-4 rounded-[2px]"
                onClick={() => {
                  if (validateWebsocket()) setActiveTab('define-assistant');
                }}
              >
                Continue
              </IBlueBGArrowButton>,
            ],
          },
          {
            code: 'define-assistant',
            name: 'Profile',
            description:
              'Provide the name, a brief description, and relevant tags for your assistant to help identify and categorize it.',
            actions: [
              <ICancelButton
                className="px-4 rounded-[2px]"
                onClick={() => showDialog(navigator.goBack)}
              >
                Cancel
              </ICancelButton>,
              <IBlueBGArrowButton
                isLoading={loading}
                type="button"
                onClick={createAssistant}
                className="px-4 rounded-[2px]"
              >
                Continue
              </IBlueBGArrowButton>,
            ],
            body: (
              <div className="space-y-6 px-8 py-8 max-w-4xl">
                <div className="h-fit pt-4 rounded-[2px] space-y-4">
                  <FieldSet>
                    <FormLabel>Name</FormLabel>
                    <Input
                      name="agent_name"
                      onChange={e => {
                        setName(e.target.value);
                      }}
                      value={name}
                      className="form-input"
                      placeholder="eg: your emotion detector"
                    ></Input>
                  </FieldSet>

                  <FieldSet>
                    <FormLabel>Description</FormLabel>
                    <Textarea
                      row={5}
                      value={description}
                      placeholder={"What's the purpose of the assistant?"}
                      onChange={t => setDescription(t.target.value)}
                    />
                  </FieldSet>
                  <TagInput
                    tags={tags}
                    addTag={onAddTag}
                    removeTag={onRemoveTag}
                    allTags={AssistantTag}
                  />
                </div>
              </div>
            ),
          },
          {
            name: 'Deployment',
            description: 'Enable assistant to start engaging with user',
            code: 'deployment',
            actions: [
              <ICancelButton
                className="px-4 rounded-[2px]"
                onClick={() => {
                  if (assistant) goToAssistant(assistant.getId());
                }}
              >
                Skip
              </ICancelButton>,
              <IBlueBGArrowButton
                type="button"
                isLoading={loading}
                className="px-4 rounded-[2px]"
                onClick={() => {
                  if (assistant) goToAssistant(assistant.getId());
                }}
              >
                Complete deployment
              </IBlueBGArrowButton>,
            ],
            body: (
              <div className="">
                <YellowNoticeBlock className="flex items-center">
                  <Info className="shrink-0 w-4 h-4" />
                  <div className="ms-3 text-sm font-medium">
                    Choose how you’d like to start engaging with users and add
                    advanced features to customize user's experience.
                  </div>
                  <a
                    target="_blank"
                    href="https://doc.rapida.ai/assistants/overview"
                    className="h-7 flex items-center font-medium hover:underline ml-auto text-yellow-600"
                    rel="noreferrer"
                  >
                    Read documentation
                    <ExternalLink
                      className="shrink-0 w-4 h-4 ml-1.5"
                      strokeWidth={1.5}
                    />
                  </a>
                </YellowNoticeBlock>
                <div className="border-gray-500">
                  <div className="grid grid-cols-1 gap-10">
                    <div className="group">
                      <h3 className="px-4 py-2 sm:px-2 font-medium text-pretty text-gray-600 dark:text-gray-400">
                        Deployments
                      </h3>
                      <dl className="bg-white dark:bg-gray-950">
                        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 divide-x">
                          <div className="border-y border-gray-300 dark:border-gray-800 grid grid-rows-[1fr_auto] max-md:border-t max-xl:last:hidden max-lg:nth-[3]:hidden last:border-r-0 max-xl:nth-[3]:border-r-0 max-lg:nth-[2]:border-r-0">
                            <div className="grid grid-cols-1 items-center">
                              <div className="px-4 py-2 sm:px-2">
                                <PhoneCall
                                  className="w-6 h-6 opacity-70 mt-4"
                                  strokeWidth={1.5}
                                />
                                <div className="flex items-center gap-2 mt-4">
                                  <h3 className="text-base/7 font-semibold">
                                    Phone call
                                  </h3>
                                </div>
                                <p className="text-sm/6 text-gray-600 md:max-w-2xs dark:text-gray-400">
                                  Enable voice conversations over phone call
                                </p>
                              </div>
                            </div>
                            <button
                              onClick={() => {
                                if (assistant)
                                  goToConfigureCall(assistant.getId());
                              }}
                              className="cursor-pointer flex justify-between items-center border-t border-gray-200 dark:border-gray-800 px-4 py-2 max-md:border-y sm:px-2 text-sm/6 text-blue-500 hover:bg-blue-600 hover:text-white transition-all delay-200"
                            >
                              Enable phone call
                              <ChevronRight className="w-4 h-4" />
                            </button>
                          </div>
                          {/*  */}
                          <div className="border-y border-gray-300 dark:border-gray-800 grid grid-rows-[1fr_auto] max-md:border-t max-xl:last:hidden max-lg:nth-[3]:hidden last:border-r-0 max-xl:nth-[3]:border-r-0 max-lg:nth-[2]:border-r-0">
                            <div className="grid grid-cols-1 items-center">
                              <div className="px-4 py-2 sm:px-2">
                                <Code
                                  className="w-6 h-6 opacity-70 mt-4"
                                  strokeWidth={1.5}
                                />
                                <div className="flex items-center gap-2 mt-4">
                                  <h3 className="text-base/7 font-semibold">
                                    API
                                  </h3>
                                </div>
                                <p className="text-sm/6 text-gray-600 md:max-w-2xs dark:text-gray-400">
                                  Integrate into your application using sdks
                                </p>
                              </div>
                            </div>
                            <button
                              onClick={() => {
                                if (assistant)
                                  goToConfigureApi(assistant.getId());
                              }}
                              className="cursor-pointer flex justify-between items-center border-t border-gray-200 dark:border-gray-800 px-4 py-2 max-md:border-y sm:px-2 text-sm/6 text-blue-500 hover:bg-blue-600 hover:text-white transition-all delay-200"
                            >
                              Enable Api
                              <ChevronRight className="w-4 h-4" />
                            </button>
                          </div>

                          {/*  */}
                          <div className="border-y border-gray-300 dark:border-gray-800 grid grid-rows-[1fr_auto] max-md:border-t max-xl:last:hidden max-lg:nth-[3]:hidden last:border-r-0 max-xl:nth-[3]:border-r-0 max-lg:nth-[2]:border-r-0">
                            <div className="grid grid-cols-1 items-center">
                              <div className="px-4 py-2 sm:px-2">
                                <Globe
                                  className="w-6 h-6 opacity-70 mt-4"
                                  strokeWidth={1.5}
                                />
                                <div className="flex items-center gap-2 mt-4">
                                  <h3 className="text-base/7 font-semibold">
                                    Web Widget
                                  </h3>
                                </div>
                                <p className="text-sm/6 text-gray-600 md:max-w-2xs dark:text-gray-400">
                                  Embed on your website to handle text and voice
                                  customer query.
                                </p>
                              </div>
                            </div>
                            <button
                              onClick={() => {
                                if (assistant)
                                  goToConfigureWeb(assistant.getId());
                              }}
                              className="cursor-pointer flex justify-between items-center border-t border-gray-200 dark:border-gray-800 px-4 py-2 max-md:border-y sm:px-2 text-sm/6 text-blue-500 hover:bg-blue-600 hover:text-white transition-all delay-200"
                            >
                              Deploy to Web Widget
                              <ChevronRight className="w-4 h-4" />
                            </button>
                          </div>
                          {/* Web Widget */}

                          <div className="border-y border-gray-300 dark:border-gray-800 grid grid-rows-[1fr_auto] max-md:border-t max-xl:last:hidden max-lg:nth-[3]:hidden last:border-r-0 max-xl:nth-[3]:border-r-0 max-lg:nth-[2]:border-r-0">
                            <div className="grid grid-cols-1 items-center">
                              <div className="px-4 py-2 sm:px-2">
                                <Bug
                                  className="w-6 h-6 opacity-70 mt-4"
                                  strokeWidth={1.5}
                                />
                                <div className="flex items-center gap-2 mt-4">
                                  <h3 className="text-base/7 font-semibold">
                                    Debugger / Testing
                                  </h3>
                                </div>
                                <p className="text-sm/6 text-gray-600 md:max-w-2xs dark:text-gray-400">
                                  Deploy the agent for testing and debugging.
                                </p>
                              </div>
                            </div>
                            <button
                              onClick={() => {
                                if (assistant)
                                  goToConfigureDebugger(assistant.getId());
                              }}
                              className="cursor-pointer flex justify-between items-center border-t border-gray-200 dark:border-gray-800 px-4 py-2 max-md:border-y sm:px-2 text-sm/6 text-blue-500 hover:bg-blue-600 hover:text-white transition-all delay-200"
                            >
                              Deploy to Debugger / Testing
                              <ChevronRight className="w-4 h-4" />
                            </button>
                          </div>
                          {/* Debugger / Testing */}
                        </div>
                        {/*  */}
                      </dl>
                    </div>
                    <div className="group">
                      <h3 className="px-4 py-2 sm:px-2 font-medium text-pretty text-gray-600 dark:text-gray-400">
                        Analysis
                      </h3>
                      <div
                        className="bg-white dark:bg-gray-950"
                        onClick={() => {
                          if (assistant)
                            goToCreateAssistantAnalysis(assistant.getId());
                        }}
                      >
                        <div className="flex w-full cursor-pointer justify-between gap-4 select-none border-y px-4 py-3 sm:px-2">
                          <div className="text-left text-sm/7 font-semibold text-pretty">
                            Gain insights from every interaction eg: Automatic
                            conversation transcripts Quality, sentiment, and SOP
                            adherence analysis Custom reporting and dashboards
                          </div>
                          <ChevronRight className="w-5 h-5" strokeWidth={1.5} />
                        </div>
                      </div>
                    </div>
                    <div className="group">
                      <h3 className="px-4 py-2 sm:px-2  font-medium text-pretty  text-gray-600 dark:text-gray-400">
                        Webhook & Integration
                      </h3>
                      <div
                        className="bg-white dark:bg-gray-950"
                        onClick={() => {
                          if (assistant)
                            goToCreateAssistantWebhook(assistant.getId());
                        }}
                      >
                        <div className="flex w-full cursor-pointer justify-between gap-4 select-none border-y px-4 py-3 sm:px-2">
                          <div className="text-left text-sm/7 font-semibold text-pretty">
                            Keep your workflows connected by triggering events
                            when key actions happen: eg: Conversation started /
                            ended Escalation to a human agent Custom events for
                            analytics or CRM sync
                          </div>
                          <ChevronRight className="w-5 h-5" strokeWidth={1.5} />
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            ),
          },
        ]}
      />
    </>
  );
}
