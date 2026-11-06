import { useCallback, useState } from 'react';
import { useRapidaStore } from '@/stores/app';
import { useCurrentCredential } from '@/hooks/use-credential';
import { useAllProviderCredentials } from '@/hooks/use-model';
import { useNavigate } from 'react-router-dom';
import toast from 'react-hot-toast/headless';
import { Helmet } from '@/app/components/app-shell/helmet';
import { PrimaryButton, SecondaryButton } from '@/app/components/ui/primitives';
import {
  ButtonSet,
  Toggletip,
  ToggletipButton,
  ToggletipContent,
} from '@carbon/react';
import { ArrowUpRight, Information } from '@carbon/icons-react';
import { TabForm } from '@/app/components/ui/composites';
import {
  CreateEndpointResponse,
  EndpointAttribute,
  EndpointProviderModelAttribute,
  Metadata,
} from '@rapidaai/react';
import { ConfirmDialog } from '@/app/components/dialogs/shared';
import { create_endpoint_success_message } from '@/utils/messages';
import {
  GetDefaultTextProviderConfigIfInvalid,
  GetDefaultTextProviderConfigOnProviderSwitch,
  TextProvider,
  ValidateTextProviderDefaultOptions,
} from '@/app/components/domain/providers/text';
import { ConfigPrompt } from '@/app/components/domain/configuration/config-prompt';
import { randomMeaningfullName, randomString } from '@/utils';
import { FieldSet } from '@/app/components/ui/primitives';
import { FormLabel } from '@/app/components/ui/primitives';
import { Input } from '@/app/components/ui/primitives';
import { TagInput } from '@/app/components/ui/composites';
import { EndpointTag } from '@/app/components/domain/tags/endpoint-tags';
import { Textarea } from '@/app/components/ui/primitives';
import { ServiceError } from '@rapidaai/react';
import { ChatCompletePrompt } from '@/utils/prompt';
import { InputHelper } from '@/app/components/ui/primitives';
import { ConfigureEndpointPromptDialog } from '@/app/components/dialogs/endpoint';
import { ActionCard } from '@/app/components/ui/primitives';
import { createEndpoint as createEndpointRequest } from '@/clients';

export function CreateEndpointPage() {
  const { authId, token, projectId } = useCurrentCredential();

  const [activeTab, setActiveTab] = useState('choose-model');
  const [errorMessage, setErrorMessage] = useState('');
  const { loading, showLoader, hideLoader } = useRapidaStore();
  const { providerCredentials } = useAllProviderCredentials();
  const navigator = useNavigate();

  const [name, setName] = useState<string>(randomMeaningfullName('endpoint'));
  const [description, setDescription] = useState('');
  const [tags, setTags] = useState<string[]>([]);
  const [promptConfig, setPromptConfig] = useState<{
    prompt: { role: string; content: string }[];
    variables: { name: string; type: string; defaultvalue: string }[];
  }>({
    prompt: [{ role: 'system', content: '' }],
    variables: [],
  });

  const [textProviderModel, setTextProviderModel] = useState<{
    provider: string;
    parameters: Metadata[];
  }>({
    provider: 'azure-foundry',
    parameters: GetDefaultTextProviderConfigIfInvalid('azure-foundry', []),
  });

  const onChangeTextProvider = (providerName: string) => {
    setTextProviderModel({
      provider: providerName,
      parameters: GetDefaultTextProviderConfigOnProviderSwitch(
        providerName,
        textProviderModel.parameters,
      ),
    });
  };

  const onChangeTextProviderParameter = (parameters: Metadata[]) => {
    setTextProviderModel({ ...textProviderModel, parameters });
  };

  const onAddTag = (newTag: string) => {
    setTags(prevTags => [...prevTags, newTag]);
  };

  const onRemoveTag = (tagToRemove: string) => {
    setTags(prevTags => prevTags.filter(tag => tag !== tagToRemove));
  };

  const afterCreateEndpoint = useCallback(
    (err: ServiceError | null, response: CreateEndpointResponse | null) => {
      hideLoader();
      if (err) {
        setErrorMessage('Something went wrong, Please try again in sometime.');
        return;
      }
      if (response?.getSuccess() && response.getData()) {
        let ep = response.getData();
        toast.success(create_endpoint_success_message(name));
        navigator(`/deployment/endpoint/${ep?.getId()}`);
        return;
      }
      if (response?.getError()) {
        let err = response.getError();
        const message = err?.getHumanmessage();
        if (message) {
          setErrorMessage(message);
          return;
        }
        setErrorMessage(
          'Unable to create endpoint, please check and try again.',
        );
      }
    },
    [],
  );

  const onvalidateEndpointInstruction = () => {
    const err = ValidateTextProviderDefaultOptions(
      textProviderModel.provider,
      textProviderModel.parameters,
      providerCredentials
        .filter(c => c.getProvider() === textProviderModel.provider)
        .map(c => c.getId()),
    );
    if (err) {
      setErrorMessage(err);
      return;
    }

    if (promptConfig.variables.length === 0) {
      setErrorMessage(
        'Please provide a valid prompt template, it should at least have one variable.',
      );
      return;
    }

    const hasNonEmptyContent = promptConfig.prompt.some(
      item => item.content.trim() !== '',
    );
    if (!hasNonEmptyContent) {
      setErrorMessage('Please provide content for at least one prompt item.');
      return;
    }

    setErrorMessage('');
    setActiveTab('define-endpoint');
  };

  const createEndpoint = () => {
    if (name.trim() === '') {
      setErrorMessage(
        'Please a valid name for endpoint, that can help you indentify the endpoint',
      );
      return;
    }
    setErrorMessage('');
    showLoader('overlay');

    const endpointProviderModelAttr = new EndpointProviderModelAttribute();
    endpointProviderModelAttr.setModelprovidername(textProviderModel.provider);
    endpointProviderModelAttr.setEndpointmodeloptionsList(
      textProviderModel.parameters,
    );
    endpointProviderModelAttr.setChatcompleteprompt(
      ChatCompletePrompt(promptConfig),
    );

    const endpointattr = new EndpointAttribute();
    endpointattr.setName(name);
    if (description.trim() !== '') {
      endpointattr.setDescription(description);
    }

    createEndpointRequest({
      endpointProviderModel: endpointProviderModelAttr,
      endpoint: endpointattr,
      tags,
      auth: {
        userId: authId,
        token,
        projectId,
      },
      callback: afterCreateEndpoint,
    });
  };

  const [isShow, setIsShow] = useState(false);
  const [isConfigureEndpointPromptOpen, setIsConfigureEndpointPromptOpen] =
    useState(false);

  const handleSelectTemplate = (template: {
    name: string;
    description: string;
    provider: string;
    model: string;
    parameters: { temperature: number; response_format: string };
    instruction: { role: string; content: string }[];
  }) => {
    setName(template.name);
    setDescription(template.description);

    const promptMessages = template.instruction.map(inst => ({
      role: inst.role,
      content: inst.content,
    }));

    const variableRegex = /\{\{\s*(\w+)(?:\[.*?\])?\s*\}\}/g;
    const variables: { name: string; type: string; defaultvalue: string }[] =
      [];
    const seenVariables = new Set<string>();

    template.instruction.forEach(inst => {
      let match;
      while ((match = variableRegex.exec(inst.content)) !== null) {
        const varName = match[1];
        if (!seenVariables.has(varName)) {
          seenVariables.add(varName);
          variables.push({
            name: varName,
            type: 'string',
            defaultvalue: '',
          });
        }
      }
    });

    setPromptConfig({
      prompt: promptMessages,
      variables:
        variables.length > 0
          ? variables
          : [{ name: 'messages', type: 'string', defaultvalue: '' }],
    });

    const newParams = GetDefaultTextProviderConfigIfInvalid(
      template.provider,
      [],
    );

    const setOrCreateParam = (key: string, value: string) => {
      const existingParam = newParams.find(p => p.getKey() === key);
      if (existingParam) {
        existingParam.setValue(value);
      } else {
        const newParam = new Metadata();
        newParam.setKey(key);
        newParam.setValue(value);
        newParams.push(newParam);
      }
    };

    setOrCreateParam(
      'model.temperature',
      String(template.parameters.temperature),
    );
    if (template.parameters.response_format) {
      setOrCreateParam(
        'model.response_format',
        template.parameters.response_format,
      );
    }
    setOrCreateParam('model.name', template.model);
    setOrCreateParam('model.id', template.model);
    const normalizedParams = GetDefaultTextProviderConfigIfInvalid(
      template.provider,
      newParams,
    );

    setTextProviderModel({
      provider: template.provider,
      parameters: normalizedParams,
    });
  };

  return (
    <>
      <Helmet title="Create an Endpoint" />
      <ConfigureEndpointPromptDialog
        modalOpen={isConfigureEndpointPromptOpen}
        setModalOpen={setIsConfigureEndpointPromptOpen}
        onSelectTemplate={handleSelectTemplate}
      />
      <ConfirmDialog
        showing={isShow}
        type="warning"
        title="Are you sure?"
        content="You want to cancel creating this endpoint? Any unsaved changes will be lost."
        confirmText="Confirm"
        cancelText="Cancel"
        onConfirm={() => navigator(-1)}
        onCancel={() => setIsShow(false)}
        onClose={() => setIsShow(false)}
      />

      <TabForm
        formHeading="Complete all steps to create a new endpoint."
        activeTab={activeTab}
        onChangeActiveTab={() => {}}
        errorMessage={errorMessage}
        form={[
          {
            name: 'Choose Model',
            description:
              'Select the LLM provider and configure your prompt template.',
            code: 'choose-model',
            body: (
              <div className="px-8 pt-6 pb-8 max-w-4xl flex flex-col gap-8">
                {/* Carbon Tile Usecase Template slot */}
                <ActionCard
                  className="group relative w-full flex items-start justify-between gap-4 p-4 text-left bg-surface border border-border-subtle hover:bg-layer transition-colors duration-100"
                  onClick={() => setIsConfigureEndpointPromptOpen(true)}
                >
                  <div className="flex flex-col gap-1 min-w-0">
                    <span className="text-[10px] font-semibold tracking-[0.12em] uppercase text-muted">
                      Quick start
                    </span>
                    <span className="text-sm font-medium text-foreground ">
                      Usecase Template
                    </span>
                    <span className="text-xs text-muted leading-relaxed">
                      Browse pre-configured templates and auto-fill your form.
                    </span>
                  </div>
                  <ArrowUpRight
                    className="shrink-0 mt-0.5 text-muted group-hover:text-primary transition-colors"
                    size={16}
                  />
                </ActionCard>

                {/* Model configuration section */}
                <div className="flex flex-col gap-6">
                  <TextProvider
                    onChangeProvider={onChangeTextProvider}
                    onChangeParameter={onChangeTextProviderParameter}
                    parameters={textProviderModel.parameters}
                    provider={textProviderModel.provider}
                  />
                </div>

                {/* Prompt template section */}
                <div className="flex flex-col gap-6">
                  <ConfigPrompt
                    instanceId={randomString(10)}
                    existingPrompt={promptConfig}
                    onChange={prompt => setPromptConfig(prompt)}
                  />
                </div>
              </div>
            ),
            actions: [
              <ButtonSet className="!w-full [&>button]:!flex-1 [&>button]:!max-w-none">
                <SecondaryButton size="lg" onClick={() => setIsShow(true)}>
                  Cancel
                </SecondaryButton>
                <PrimaryButton
                  size="lg"
                  isLoading={loading}
                  onClick={onvalidateEndpointInstruction}
                >
                  Configure instruction
                </PrimaryButton>
              </ButtonSet>,
            ],
          },
          {
            code: 'define-endpoint',
            name: 'Define Endpoint Profile',
            description:
              'Give your endpoint a name, description, and labels to make it easy to find and manage.',
            actions: [
              <ButtonSet className="!w-full [&>button]:!flex-1 [&>button]:!max-w-none">
                <SecondaryButton size="lg" onClick={() => setIsShow(true)}>
                  Cancel
                </SecondaryButton>
                <PrimaryButton
                  size="lg"
                  isLoading={loading}
                  onClick={createEndpoint}
                >
                  Create endpoint
                </PrimaryButton>
              </ButtonSet>,
            ],
            body: (
              <div className="px-8 pt-8 pb-8 max-w-2xl flex flex-col gap-10">
                <div className="flex flex-col gap-6">
                  <FieldSet>
                    <div className="flex items-baseline justify-between">
                      <div className="flex items-center gap-1">
                        <FormLabel htmlFor="name">Endpoint name</FormLabel>
                        <Toggletip align="right">
                          <ToggletipButton label="Endpoint name information">
                            <Information size={14} />
                          </ToggletipButton>
                          <ToggletipContent>
                            Use a short, stable name that makes this endpoint
                            easy to identify.
                          </ToggletipContent>
                        </Toggletip>
                      </div>
                      <span className="text-xs text-muted tabular-nums">
                        {name.length}/100
                      </span>
                    </div>
                    <Input
                      id="name"
                      name="name"
                      maxLength={100}
                      onChange={e => setName(e.target.value)}
                      value={name}
                      placeholder="e.g. customer-support-v1"
                    />
                  </FieldSet>

                  <FieldSet>
                    <div className="flex items-center gap-1">
                      <FormLabel htmlFor="description">Description</FormLabel>
                      <Toggletip align="right">
                        <ToggletipButton label="Endpoint description information">
                          <Information size={14} />
                        </ToggletipButton>
                        <ToggletipContent>
                          Describe what this endpoint is for and when it should
                          be used.
                        </ToggletipContent>
                      </Toggletip>
                    </div>
                    <Textarea
                      id="description"
                      row={4}
                      name="description"
                      value={description}
                      placeholder="What does this endpoint do? When should it be used?"
                      onChange={t => setDescription(t.target.value)}
                    />
                  </FieldSet>
                </div>

                <div className="flex flex-col gap-6">
                  <TagInput
                    tags={tags}
                    addTag={onAddTag}
                    removeTag={onRemoveTag}
                    allTags={EndpointTag}
                  />
                  <InputHelper>
                    Tags help you organize and filter endpoints across your
                    workspace.
                  </InputHelper>
                </div>
              </div>
            ),
          },
        ]}
      />
    </>
  );
}
