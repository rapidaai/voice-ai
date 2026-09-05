import { useState } from 'react';
import { Helmet } from '@/app/components/helmet';
import { useRapidaStore } from '@/hooks';
import { TabForm } from '@/app/components/form/tab-form';
import { PrimaryButton, SecondaryButton } from '@/app/components/carbon/button';
import {
  ButtonSet,
  Table,
  TableHead,
  TableRow,
  TableHeader,
  TableBody,
  TableCell,
  TableToolbar,
  TableToolbarContent,
  TableToolbarSearch,
  TableBatchActions,
  TableBatchAction,
  RadioButton,
  Tag,
  Button,
} from '@carbon/react';
import {
  Assistant,
  ConnectionConfig,
  CreateAssistantProviderRequest,
  CreateAssistantRequest,
  GetAssistantResponse,
  Metadata,
} from '@rapidaai/react';
import { useConfirmDialog } from '@/app/pages/assistant/actions/hooks/use-confirmation';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { useCurrentCredential } from '@/hooks/use-credential';
import { useAllProviderCredentials } from '@/hooks/use-model';
import { ConfigPrompt } from '@/app/components/configuration/config-prompt';
import { randomMeaningfullName, randomString } from '@/utils';
import { TextInput, TextArea, Stack } from '@/app/components/carbon/form';
import { TagInput } from '@/app/components/form/tag-input';
import { AssistantTag } from '@/app/components/form/tag-input/assistant-tags';
import {
  GetDefaultTextProviderConfigIfInvalid,
  GetDefaultTextProviderConfigOnProviderSwitch,
  TextProvider,
  ValidateTextProviderDefaultOptions,
} from '@/app/components/providers/text';
import { BuildinToolConfig } from '@/app/components/tools';
import {
  getToolConditionSource,
  getToolConditionSourceLabel,
} from '@/app/components/tools/common';
import { ArrowUpRight } from 'lucide-react';
import { BUILDIN_TOOLS } from '@/llm-tools';
import { EmptyState } from '@/app/components/carbon/empty-state';
import { ConfigureAssistantToolDialog } from '@/app/components/base/modal/assistant-configure-tool-modal';
import { DocNoticeBlock } from '@/app/components/container/message/notice-block/doc-notice-block';
import { CreateAssistant } from '@rapidaai/react';
import { CreateAssistantToolRequest } from '@rapidaai/react';
import { Struct } from 'google-protobuf/google/protobuf/struct_pb';
import { connectionConfig } from '@/configs';
import { ChatCompletePrompt } from '@/utils/prompt';
import toast from 'react-hot-toast/headless';
import { ConfigureAssistantNextDialog } from '@/app/components/base/modal/assistant-configure-next-modal';
import { SectionDivider } from '@/app/components/blocks/section-divider';
import { CornerBorderOverlay } from '@/app/components/base/corner-border';
import { Add, Edit, ToolKit, TrashCan } from '@carbon/icons-react';
import {
  AssistantTemplate,
  ConfigureAssistantTemplateDialog,
} from '@/app/components/base/modal/configure-assistant-template-modal';
import { useTheme } from '@/theme/theme-provider';

/**
 *
 * @returns
 */
export function CreateAssistantPage() {
  const { theme } = useTheme();
  /**
   * credentils and authentication parameters
   */
  const { authId, token, projectId } = useCurrentCredential();

  /**
   * global reloading
   */
  const { loading, showLoader, hideLoader } = useRapidaStore();
  const { providerCredentials } = useAllProviderCredentials();

  /**
   * after creation of assistant maintaining stage
   */
  const [assistant, setAssistant] = useState<null | Assistant>(null);

  /**
   * navigation
   */
  const { goBack, goToAssistant } = useGlobalNavigation();

  /**
   *
   */
  const [createAssistantSuccess, setCreateAssistantSuccess] = useState(false);

  /**
   * multi step form
   */
  const [activeTab, setActiveTab] = useState<
    'choose-model' | 'tools' | 'define-assistant'
  >('choose-model');

  /**
   * Error message
   */
  const [errorMessage, setErrorMessage] = useState('');

  /**
   * Form fields
   */
  const [name, setName] = useState(randomMeaningfullName('assistant'));
  const [description, setDescription] = useState('');
  const [tags, setTags] = useState<string[]>([]);
  const [tools, setTools] = useState<
    {
      name: string;
      description: string;
      fields: string;
      buildinToolConfig: BuildinToolConfig;
    }[]
  >([]);
  const [editingTool, setEditingTool] = useState<{
    name: string;
    description: string;
    fields: string;
    buildinToolConfig: BuildinToolConfig;
  } | null>(null);
  const [selectedModel, setSelectedModel] = useState<{
    provider: string;
    parameters: Metadata[];
  }>({
    provider: 'azure-foundry',
    parameters: GetDefaultTextProviderConfigIfInvalid('azure-foundry', []),
  });
  const [template, setTemplate] = useState<{
    prompt: { role: string; content: string }[];
    variables: { name: string; type: string; defaultvalue: string }[];
  }>({
    prompt: [{ role: 'system', content: '' }],
    variables: [],
  });
  const { showDialog, ConfirmDialogComponent } = useConfirmDialog({});
  const [configureToolOpen, setConfigureToolOpen] = useState(false);
  const [templateModalOpen, setTemplateModalOpen] = useState(false);
  const [toolSearchTerm, setToolSearchTerm] = useState('');
  const [selectedToolName, setSelectedToolName] = useState<string | null>(null);

  /**
   * Applies a selected usecase template to pre-fill the form state.
   */
  const handleSelectTemplate = (tmpl: AssistantTemplate) => {
    setName(tmpl.name);
    setDescription(tmpl.description);

    // Build prompt messages
    const promptMessages = tmpl.instruction.map(inst => ({
      role: inst.role,
      content: inst.content,
    }));

    // Extract {{variable}} references from all prompt content
    const variableRegex = /\{\{\s*(\w+)\s*\}\}/g;
    const variables: { name: string; type: string; defaultvalue: string }[] =
      [];
    const seen = new Set<string>();
    tmpl.instruction.forEach(inst => {
      let match;
      while ((match = variableRegex.exec(inst.content)) !== null) {
        const varName = match[1];
        if (!seen.has(varName)) {
          seen.add(varName);
          variables.push({ name: varName, type: 'string', defaultvalue: '' });
        }
      }
    });

    setTemplate({
      prompt: promptMessages,
      variables: variables.length > 0 ? variables : [],
    });

    // Apply model/provider settings
    const newParams = GetDefaultTextProviderConfigIfInvalid(tmpl.provider, []);
    const setOrCreate = (key: string, value: string) => {
      const existing = newParams.find(p => p.getKey() === key);
      if (existing) {
        existing.setValue(value);
      } else {
        const m = new Metadata();
        m.setKey(key);
        m.setValue(value);
        newParams.push(m);
      }
    };
    setOrCreate('model.temperature', String(tmpl.parameters.temperature));
    setOrCreate('model.name', tmpl.model);
    setOrCreate('model.id', tmpl.model);
    const normalizedParams = GetDefaultTextProviderConfigIfInvalid(
      tmpl.provider,
      newParams,
    );

    setSelectedModel({ provider: tmpl.provider, parameters: normalizedParams });
  };
  const onAddTag = (tag: string) => {
    setTags([...tags, tag]);
  };
  const onRemoveTag = (tag: string) => {
    setTags(tags.filter(t => t !== tag));
  };
  const onChangeProvider = (providerName: string) => {
    setSelectedModel({
      provider: providerName,
      parameters: GetDefaultTextProviderConfigOnProviderSwitch(
        providerName,
        selectedModel.parameters,
      ),
    });
  };
  const onChangeParameter = (parameters: Metadata[]) => {
    setSelectedModel({ ...selectedModel, parameters });
  };

  /**
   *
   * @returns
   */
  const createAssistant = () => {
    if (!name) {
      setErrorMessage('Please provide a valid name for assistant.');
      return false;
    }
    showLoader('overlay');
    const assistantToolConfig = tools.map(t => {
      const req = new CreateAssistantToolRequest();
      req.setName(t.name);
      req.setDescription(t.description);
      req.setFields(Struct.fromJavaScript(JSON.parse(t.fields)));
      req.setExecutionmethod(t.buildinToolConfig.code);
      req.setExecutionoptionsList(t.buildinToolConfig.parameters);
      return req;
    });
    const assistantProvider = new CreateAssistantProviderRequest();
    const assistantModel =
      new CreateAssistantProviderRequest.CreateAssistantProviderModel();
    assistantModel.setTemplate(ChatCompletePrompt(template));
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
            setCreateAssistantSuccess(true);
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

  /**
   * validate instruction
   * @returns
   */
  const validateInstruction = (): boolean => {
    setErrorMessage('');
    let err = ValidateTextProviderDefaultOptions(
      selectedModel.provider,
      selectedModel.parameters,
      providerCredentials
        .filter(c => c.getProvider() === selectedModel.provider)
        .map(c => c.getId()),
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

  /**
   * validation of tools
   * @returns
   */
  const validateTool = (): boolean => {
    setErrorMessage('');
    if (tools.length === 0) {
      setErrorMessage('Please add atleast one tool for the assistant.');
      return false;
    }
    return true;
  };

  const normalizedToolSearch = toolSearchTerm.trim().toLowerCase();
  const filteredTools = normalizedToolSearch
    ? tools.filter(tool =>
        [
          tool.name,
          tool.description,
          tool.buildinToolConfig.code.replace(/_/g, ' '),
        ]
          .join(' ')
          .toLowerCase()
          .includes(normalizedToolSearch),
      )
    : tools;

  //
  return (
    <>
      <Helmet title="Create an assistant"></Helmet>
      <ConfirmDialogComponent />

      <ConfigureAssistantTemplateDialog
        modalOpen={templateModalOpen}
        setModalOpen={setTemplateModalOpen}
        onSelectTemplate={handleSelectTemplate}
      />

      {assistant && (
        <ConfigureAssistantNextDialog
          assistant={assistant}
          modalOpen={createAssistantSuccess}
          setModalOpen={() => {
            setCreateAssistantSuccess(false);
            goToAssistant(assistant.getId());
          }}
        />
      )}

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
            return 'Please provide a valid tool name.';
          }

          // Check for duplicate name
          const isDuplicate = tools.some(
            tool =>
              tool.name !== editingTool?.name && tool.name === updatedTool.name,
          );

          if (isDuplicate) {
            return 'Please provide a unique tool name for tools.';
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
        formHeading="Complete all steps to create a new assistant."
        activeTab={activeTab}
        onChangeActiveTab={() => {}}
        errorMessage={errorMessage}
        form={[
          {
            name: 'Configuration',
            description:
              'Select the LLM provider and configure your prompt template.',
            code: 'choose-model',
            body: (
              <>
                <DocNoticeBlock
                  docPath="/assistants/overview"
                  linkText="Read docs"
                >
                  {theme.brand.name} enables you to deploy intelligent
                  conversational agents across multiple channels.
                </DocNoticeBlock>
                <div className="px-8 pt-6 pb-8 max-w-4xl flex flex-col gap-8">
                  {/* Quick start — Usecase Template tile */}
                  <button
                    type="button"
                    className="group relative w-full flex items-start justify-between gap-4 p-4 text-left bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-800 hover:bg-gray-50 dark:hover:bg-gray-800/60 transition-colors duration-100"
                    onClick={() => setTemplateModalOpen(true)}
                  >
                    <CornerBorderOverlay />
                    <div className="flex flex-col gap-1 min-w-0">
                      <span className="text-[10px] font-semibold tracking-[0.12em] uppercase text-gray-500 dark:text-gray-400">
                        Quick start
                      </span>
                      <span className="text-sm font-medium text-gray-900 dark:text-gray-100">
                        Usecase Template
                      </span>
                      <span className="text-xs text-gray-500 dark:text-gray-500 leading-relaxed">
                        Browse 8 pre-configured assistant templates and
                        auto-fill your form.
                      </span>
                    </div>
                    <ArrowUpRight
                      className="shrink-0 mt-0.5 text-gray-500 dark:text-gray-400 group-hover:text-primary transition-colors"
                      strokeWidth={1.5}
                      size={16}
                    />
                  </button>

                  {/* Model configuration section */}
                  <div className="flex flex-col gap-6">
                    <SectionDivider label="Model Configuration" />
                    <TextProvider
                      onChangeParameter={onChangeParameter}
                      onChangeProvider={onChangeProvider}
                      parameters={selectedModel.parameters}
                      provider={selectedModel.provider}
                    />
                  </div>

                  {/* Prompt template section */}
                  <div className="flex flex-col gap-6">
                    <SectionDivider label="Prompt Template" />
                    <DocNoticeBlock docPath="/assistants/prompt-templating">
                      Prompt variables and system arguments are resolved at
                      runtime. Read the prompt templating guide before
                      finalizing your instruction.
                    </DocNoticeBlock>
                    <ConfigPrompt
                      instanceId={randomString(10)}
                      existingPrompt={template}
                      showRuntimeReplacementHint
                      enableReservedVariableSuggestions
                      onChange={prompt => setTemplate(prompt)}
                    />
                  </div>
                </div>
              </>
            ),
            actions: [
              <ButtonSet className="!w-full [&>button]:!flex-1 [&>button]:!max-w-none">
                <SecondaryButton size="lg" onClick={() => showDialog(goBack)}>
                  Cancel
                </SecondaryButton>
                <PrimaryButton
                  size="lg"
                  isLoading={loading}
                  onClick={() => {
                    if (validateInstruction()) setActiveTab('tools');
                  }}
                >
                  Continue
                </PrimaryButton>
              </ButtonSet>,
            ],
          },
          {
            code: 'tools',
            name: 'Tools (optional)',
            description:
              'Let your assistant work with different tools on behalf of you.',
            actions: [
              <ButtonSet className="!w-full [&>button]:!flex-1 [&>button]:!max-w-none">
                <SecondaryButton size="lg" onClick={() => showDialog(goBack)}>
                  Cancel
                </SecondaryButton>
                <PrimaryButton
                  size="lg"
                  isLoading={loading}
                  onClick={() => {
                    if (tools.length === 0) {
                      setTools([]);
                      setErrorMessage('');
                      setActiveTab('define-assistant');
                      return;
                    }
                    if (validateTool()) setActiveTab('define-assistant');
                  }}
                >
                  {tools.length === 0 ? 'Skip for now' : 'Continue'}
                </PrimaryButton>
              </ButtonSet>,
            ],
            body: (
              <div className="relative flex flex-col flex-1">
                <TableToolbar>
                  <TableBatchActions
                    shouldShowBatchActions={!!selectedToolName}
                    totalSelected={selectedToolName ? 1 : 0}
                    totalCount={tools.length}
                    onCancel={() => setSelectedToolName(null)}
                    className="[&_[class*=divider]]:hidden [&_.cds--btn]:transition-colors [&_.cds--btn:hover]:!bg-primary [&_.cds--btn:hover]:!text-white"
                  >
                    {selectedToolName && (
                      <>
                        <TableBatchAction
                          renderIcon={Edit}
                          onClick={() => {
                            const selectedTool = tools.find(
                              itm => itm.name === selectedToolName,
                            );
                            if (!selectedTool) return;
                            setEditingTool(selectedTool);
                            setConfigureToolOpen(true);
                            setSelectedToolName(null);
                          }}
                        >
                          Edit tool
                        </TableBatchAction>
                        <TableBatchAction
                          renderIcon={TrashCan}
                          onClick={() => {
                            setTools(prevTools =>
                              prevTools.filter(
                                tool => tool.name !== selectedToolName,
                              ),
                            );
                            setSelectedToolName(null);
                          }}
                        >
                          Delete tool
                        </TableBatchAction>
                      </>
                    )}
                  </TableBatchActions>
                  <TableToolbarContent>
                    <TableToolbarSearch
                      placeholder="Search tools..."
                      onChange={(e: any) =>
                        setToolSearchTerm(e.target?.value || '')
                      }
                    />
                    <PrimaryButton
                      size="md"
                      renderIcon={Add}
                      onClick={() => setConfigureToolOpen(true)}
                    >
                      Add tool
                    </PrimaryButton>
                  </TableToolbarContent>
                </TableToolbar>
                <div className="overflow-auto flex flex-col flex-1">
                  {tools.length > 0 && filteredTools.length > 0 ? (
                    <Table>
                      <TableHead>
                        <TableRow>
                          <TableHeader className="!w-12" />
                          <TableHeader>Name</TableHeader>
                          <TableHeader>Type</TableHeader>
                          <TableHeader>Description</TableHeader>
                          <TableHeader>Actions</TableHeader>
                        </TableRow>
                      </TableHead>
                      <TableBody>
                        {filteredTools.map((itm, idx) => {
                          const method = itm.buildinToolConfig.code;
                          const methodMeta = BUILDIN_TOOLS.find(
                            x => x.code === method,
                          );
                          const isMCP = method === 'mcp';
                          const selected = selectedToolName === itm.name;
                          const conditionSource = getToolConditionSource(
                            itm.buildinToolConfig.parameters,
                          );

                          return (
                            <TableRow
                              key={`tool-row-${idx}`}
                              isSelected={selected}
                              onClick={() =>
                                setSelectedToolName(selected ? null : itm.name)
                              }
                              className="cursor-pointer"
                            >
                              <TableCell
                                className="!w-12 !pr-0"
                                onClick={e => e.stopPropagation()}
                              >
                                <RadioButton
                                  id={`tool-select-${itm.name}`}
                                  name="tool-select"
                                  labelText=""
                                  hideLabel
                                  checked={selected}
                                  onChange={() =>
                                    setSelectedToolName(
                                      selected ? null : itm.name,
                                    )
                                  }
                                />
                              </TableCell>
                              <TableCell>{itm.name}</TableCell>
                              <TableCell>
                                <div className="flex items-center gap-1">
                                  {methodMeta && (
                                    <Tag size="sm" type="gray">
                                      {methodMeta.name}
                                    </Tag>
                                  )}
                                  {isMCP && (
                                    <Tag size="sm" type="purple">
                                      MCP
                                    </Tag>
                                  )}
                                  {!methodMeta && !isMCP && (
                                    <Tag size="sm" type="gray">
                                      {(method || 'Unknown').replace(/_/g, ' ')}
                                    </Tag>
                                  )}
                                  {conditionSource !== 'all' && (
                                    <Tag size="sm" type="blue">
                                      Source:{' '}
                                      {getToolConditionSourceLabel(
                                        conditionSource,
                                      )}
                                    </Tag>
                                  )}
                                </div>
                              </TableCell>
                              <TableCell className="max-w-[360px] truncate">
                                {itm.description}
                              </TableCell>
                              <TableCell onClick={e => e.stopPropagation()}>
                                <div className="flex items-center gap-0">
                                  <Button
                                    hasIconOnly
                                    renderIcon={Edit}
                                    iconDescription="Edit tool"
                                    size="sm"
                                    kind="ghost"
                                    onClick={() => {
                                      setEditingTool(itm);
                                      setConfigureToolOpen(true);
                                    }}
                                  />
                                  <Button
                                    hasIconOnly
                                    renderIcon={TrashCan}
                                    iconDescription="Delete tool"
                                    size="sm"
                                    kind="ghost"
                                    onClick={() => {
                                      setTools(prevTools =>
                                        prevTools.filter(
                                          tool => tool.name !== itm.name,
                                        ),
                                      );
                                      if (selectedToolName === itm.name) {
                                        setSelectedToolName(null);
                                      }
                                    }}
                                  />
                                </div>
                              </TableCell>
                            </TableRow>
                          );
                        })}
                      </TableBody>
                    </Table>
                  ) : tools.length > 0 ? (
                    <EmptyState
                      icon={ToolKit}
                      title="No tools found"
                      subtitle="No tool matched your search."
                    />
                  ) : (
                    <EmptyState
                      icon={ToolKit}
                      title="No tools found"
                      subtitle="Any tools or MCPs you add will be listed here."
                    />
                  )}
                </div>
              </div>
            ),
          },
          {
            code: 'define-assistant',
            name: 'Profile',
            description:
              'Provide the name, a brief description, and relevant tags for your assistant to help identify and categorize it.',
            actions: [
              <ButtonSet className="!w-full [&>button]:!flex-1 [&>button]:!max-w-none">
                <SecondaryButton size="lg" onClick={() => showDialog(goBack)}>
                  Cancel
                </SecondaryButton>
                <PrimaryButton
                  size="lg"
                  isLoading={loading}
                  onClick={createAssistant}
                >
                  Create assistant
                </PrimaryButton>
              </ButtonSet>,
            ],
            body: (
              <div className="px-8 pt-8 pb-8 max-w-2xl">
                <Stack gap={7}>
                  <p className="text-[10px] font-semibold tracking-[0.12em] uppercase text-gray-500 dark:text-gray-400">
                    Identity
                  </p>
                  <TextInput
                    id="agent-name"
                    labelText="Name *"
                    value={name}
                    onChange={e => setName(e.target.value)}
                    placeholder="e.g. customer-support-assistant"
                    helperText="Provide a name that will appear in the assistant list and help identify it."
                  />
                  <TextArea
                    id="agent-description"
                    labelText="Description (Optional)"
                    value={description}
                    onChange={e => setDescription(e.target.value)}
                    placeholder="What's the purpose of the assistant?"
                    rows={4}
                    helperText="Provide a description to explain what this assistant is about."
                  />
                  <p className="text-[10px] font-semibold tracking-[0.12em] uppercase text-gray-500 dark:text-gray-400">
                    Labels
                  </p>
                  <TagInput
                    tags={tags}
                    addTag={onAddTag}
                    removeTag={onRemoveTag}
                    allTags={AssistantTag}
                  />
                </Stack>
              </div>
            ),
          },
        ]}
      />
    </>
  );
}
