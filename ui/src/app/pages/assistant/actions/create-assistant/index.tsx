import { useState } from 'react';
import { Helmet } from '@/app/components/Helmet';
import { useRapidaStore } from '@/hooks';
import { TabForm } from '@/app/components/Form/tab-form';
import {
  IBlueBGArrowButton,
  IBlueButton,
  ICancelButton,
} from '@/app/components/Form/Button';
import {
  Assistant,
  ConnectionConfig,
  CreateAssistantProviderRequest,
  CreateAssistantRequest,
  GetAssistantResponse,
} from '@rapidaai/react';
import { useConfirmDialog } from '@/app/pages/assistant/actions/hooks/use-confirmation';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { useCurrentCredential } from '@/hooks/use-credential';
import { ConfigPrompt } from '@/app/components/configuration/config-prompt';
import { randomMeaningfullName, randomString } from '@/utils';
import { FieldSet } from '@/app/components/Form/Fieldset';
import { FormLabel } from '@/app/components/form-label';
import { Input } from '@/app/components/Form/Input';
import { Textarea } from '@/app/components/Form/Textarea';
import { TagInput } from '@/app/components/Form/tag-input';
import { AssistantTag } from '@/app/components/Form/tag-input/assistant-tags';
import {
  GetDefaultTextProviderConfigIfInvalid,
  TextProvider,
  ValidateTextProviderDefaultOptions,
} from '@/app/components/providers/text';
import { ProviderConfig } from '@/app/components/providers';
import { BuildinToolConfig } from '@/app/components/tools';
import { Card, CardDescription, CardTitle } from '@/app/components/base/cards';
import {
  Bug,
  ChevronRight,
  Code,
  ExternalLink,
  Info,
  PhoneCall,
  Plus,
  SquareFunction,
} from 'lucide-react';
import { ActionableEmptyMessage } from '@/app/components/container/message/actionable-empty-message';
import { ConfigureAssistantToolDialog } from '@/app/components/base/modal/assistant-configure-tool-modal';
import { YellowNoticeBlock } from '@/app/components/container/message/notice-block';
import { CardOptionMenu } from '@/app/components/Menu';
import { CreateAssistant } from '@rapidaai/react';
import { CreateAssistantToolRequest } from '@rapidaai/react';
import { Struct } from 'google-protobuf/google/protobuf/struct_pb';
import { connectionConfig } from '@/configs';
import { Phone, Globe } from 'lucide-react';
import { InputGroup } from '@/app/components/input-group';
import { ChatCompletePrompt } from '@/utils/prompt';
import toast from 'react-hot-toast/headless';

/**
 *
 * @returns
 */
export function CreateAssistantPage() {
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
    'tools' | 'choose-model' | 'define-assistant' | 'deployment'
  >('choose-model');
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
  const [tools, setTools] = useState<
    {
      name: string;
      description: string;
      fields: string;
      buildinToolConfig: BuildinToolConfig;
    }[]
  >([]);

  //
  const [editingTool, setEditingTool] = useState<{
    name: string;
    description: string;
    fields: string;
    buildinToolConfig: BuildinToolConfig;
  } | null>(null);

  //
  const [selectedModel, setSelectedModel] = useState<ProviderConfig>({
    providerId: '198796716894742122',
    provider: 'azure',
    parameters: GetDefaultTextProviderConfigIfInvalid('azure', []),
  });

  const { showDialog, ConfirmDialogComponent } = useConfirmDialog({});
  const [configureToolOpen, setConfigureToolOpen] = useState(false);

  //
  const onChangeProvider = (providerId: string, providerName: string) => {
    setSelectedModel({
      providerId: providerId,
      provider: providerName,
      parameters: GetDefaultTextProviderConfigIfInvalid(
        providerName,
        selectedModel.parameters,
      ),
    });
  };

  //
  const [template, setTemplate] = useState<{
    prompt: { role: string; content: string }[];
    variables: { name: string; type: string; defaultvalue: string }[];
  }>({
    prompt: [{ role: 'system', content: '' }],
    variables: [],
  });

  const { loading, showLoader, hideLoader } = useRapidaStore();
  let navigator = useGlobalNavigation();

  const createAssistant = () => {
    showLoader('overlay');
    if (!name) {
      setErrorMessage('Please provide a valid name for assistant.');
      return false;
    }

    // Create tool configuration
    const assistantToolConfig = tools.map(t => {
      const req = new CreateAssistantToolRequest();
      req.setName(t.name);
      req.setDescription(t.description);
      req.setFields(Struct.fromJavaScript(JSON.parse(t.fields)));
      req.setExecutionmethod(t.buildinToolConfig.code);
      req.setExecutionoptionsList(t.buildinToolConfig.parameters);
      return req;
    });

    // Create assistant provider model
    const assistantProvider = new CreateAssistantProviderRequest();
    const assistantModel =
      new CreateAssistantProviderRequest.CreateAssistantProviderModel();
    assistantModel.setTemplate(ChatCompletePrompt(template));
    assistantModel.setModelproviderid(selectedModel.providerId);
    assistantModel.setModelprovidername(selectedModel.provider);
    assistantModel.setAssistantmodeloptionsList(selectedModel.parameters);
    assistantProvider.setModel(assistantModel);
    const request = new CreateAssistantRequest();
    request.setAssistantprovider(assistantProvider);
    request.setAssistanttoolsList(assistantToolConfig);
    request.setName(name);
    request.setTagsList(tags);
    request.setDescription(description);
    CreateAssistant(
      connectionConfig,
      request,
      ConnectionConfig.WithDebugger({
        authorization: token,
        userId: authId,
        projectId: projectId,
      }),
    )
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

  const validateInstruction = (): boolean => {
    setErrorMessage('');
    let err = ValidateTextProviderDefaultOptions(
      selectedModel.provider,
      selectedModel.parameters,
    );
    if (err) {
      setErrorMessage(err);
      return false;
    }

    // Add template prompt validation
    if (!template.prompt || template.prompt.length === 0) {
      setErrorMessage('Please provide a valid template prompt.');
      return false;
    }

    // Validate each prompt message in the template
    for (const message of template.prompt) {
      if (!message.role || !message.content || message.content.trim() === '') {
        setErrorMessage(
          'Each prompt message must have a valid role and non-empty content.',
        );
        return false;
      }
    }
    return true;
  };

  const validateTool = (): boolean => {
    setErrorMessage('');
    if (tools.length === 0) {
      setErrorMessage('Please add atleast one tool for the assistant.');
      return false;
    }
    return true;
  };

  //
  return (
    <>
      <Helmet title="Create an assistant"></Helmet>
      <ConfirmDialogComponent />
      <ConfigureAssistantToolDialog
        modalOpen={configureToolOpen}
        setModalOpen={v => {
          setEditingTool(null);
          setConfigureToolOpen(v);
        }}
        initialData={editingTool}
        onValidateConfig={updatedTool => {
          // Check for empty name
          if (!updatedTool.name.trim()) {
            return 'Tool name cannot be empty';
          }

          // Check for duplicate name
          const isDuplicate = tools.some(
            tool =>
              tool.name !== editingTool?.name && tool.name === updatedTool.name,
          );

          if (isDuplicate) {
            return 'A tool with this name already exists';
          }

          return null;
        }}
        onChange={updatedTool => {
          if (editingTool) {
            setTools(
              tools.map(tool =>
                tool.name === editingTool.name ? updatedTool : tool,
              ),
            );
          } else {
            setTools([...tools, updatedTool]);
          }
          setEditingTool(null);
          setConfigureToolOpen(false);
        }}
      />
      <TabForm
        formHeading="Complete all steps to create new assistant"
        activeTab={activeTab}
        onChangeActiveTab={() => {}}
        errorMessage={errorMessage}
        form={[
          {
            name: 'Configuration',
            description: 'Select the llm you want to use for your assistant.',
            code: 'choose-model',
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
                <div className="space-y-6 px-8 py-8 max-w-4xl">
                  <div className="space-y-6">
                    <TextProvider
                      config={selectedModel}
                      onChangeConfig={setSelectedModel}
                      onChangeProvider={onChangeProvider}
                    />
                  </div>
                  <ConfigPrompt
                    instanceId={randomString(10)}
                    existingPrompt={template}
                    onChange={prompt => setTemplate(prompt)}
                  />
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
                  if (validateInstruction()) setActiveTab('tools');
                }}
              >
                Continue
              </IBlueBGArrowButton>,
            ],
          },

          {
            code: 'tools',
            name: 'Tools (optional)',
            description:
              'Let your assistant work with given differnt tools on behalf of you',
            body: (
              <div className="flex grow flex-col">
                <div className="flex items-center justify-between pl-4 bg-white dark:bg-gray-900 border-b">
                  Tool and MCPs
                  <div className="flex divide-x">
                    <IBlueButton
                      onClick={() => {
                        setConfigureToolOpen(true);
                      }}
                    >
                      Add another tool
                      <Plus className="w-4 h-4 ml-1.5" />
                    </IBlueButton>
                  </div>
                </div>
                <YellowNoticeBlock className="flex items-center">
                  <Info className="shrink-0 w-4 h-4" />
                  <div className="ms-3 text-sm font-medium">
                    Activate the tools you want your assistant to use, allowing
                    it to perform actions like fetching real-time data,
                    processing complex tasks, and more.
                  </div>
                </YellowNoticeBlock>
                {tools.length > 0 ? (
                  <div className="overflow-y-auto grid-cols-2 md:grid-cols-4 grid gap-2 px-4 py-2">
                    {tools.map((itm, idx) => (
                      <Card key={idx}>
                        <header className="flex justify-between">
                          <SquareFunction
                            className="w-7 h-7"
                            strokeWidth={1.5}
                          />
                          <CardOptionMenu
                            options={[
                              {
                                option: (
                                  <span className="text-red-600">
                                    Delete tool
                                  </span>
                                ),
                                onActionClick: () => {
                                  setTools(prevTools =>
                                    prevTools.filter(tool => tool !== itm),
                                  );
                                },
                              },
                              {
                                option: 'Edit tool',
                                onActionClick: () => {
                                  setEditingTool(itm);
                                  setConfigureToolOpen(true);
                                },
                              },
                            ]}
                            classNames="h-8 w-8 p-1 opacity-60"
                          />
                        </header>
                        <div className="flex-1 mt-3">
                          <CardTitle>{itm.name}</CardTitle>
                          <CardDescription>{itm.description}</CardDescription>
                        </div>
                      </Card>
                    ))}
                  </div>
                ) : (
                  <div className="justify-self-center justify-center items-center mx-auto my-auto w-full">
                    <ActionableEmptyMessage
                      title="No Tools"
                      subtitle="There are no tools given added to the assistant"
                      action="Add a tool"
                      onActionClick={() => setConfigureToolOpen(true)}
                    />
                  </div>
                )}
              </div>
            ),
            actions: [
              <ICancelButton
                className="px-4 rounded-[2px]"
                onClick={() => showDialog(navigator.goBack)}
              >
                Cancel
              </ICancelButton>,
              <ICancelButton
                className="px-4 rounded-[2px]"
                onClick={() => {
                  setTools([]);
                  setErrorMessage('');
                  setActiveTab('define-assistant');
                }}
              >
                Skip tools
              </ICancelButton>,
              <IBlueBGArrowButton
                type="button"
                isLoading={loading}
                onClick={() => {
                  //
                  if (validateTool()) setActiveTab('define-assistant');
                }}
                className="px-4 rounded-[2px]"
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
