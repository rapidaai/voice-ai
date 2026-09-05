import { useState } from 'react';
import { Helmet } from '@/app/components/helmet';
import { useRapidaStore } from '@/hooks';
import { TabForm } from '@/app/components/form/tab-form';
import { PrimaryButton, SecondaryButton } from '@/app/components/carbon/button';
import { ButtonSet, Slider } from '@carbon/react';
import { ChevronDown } from '@carbon/icons-react';
import {
  Assistant,
  CreateAssistantProviderRequest,
  CreateAssistantRequest,
  GetAssistantResponse,
} from '@rapidaai/react';
import { useConfirmDialog } from '@/app/pages/assistant/actions/hooks/use-confirmation';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { useCurrentCredential } from '@/hooks/use-credential';
import { randomMeaningfullName } from '@/utils';
import { FieldSet } from '@/app/components/form/fieldset';
import { FormLabel } from '@/app/components/form-label';
import { Input } from '@/app/components/form/input';
import { Select } from '@/app/components/form/select';
import { Textarea } from '@/app/components/form/textarea';
import { TagInput } from '@/app/components/form/tag-input';
import { AssistantTag } from '@/app/components/form/tag-input/assistant-tags';
import { Bug, ChevronRight, Code, PhoneCall } from 'lucide-react';
import { DocNoticeBlock } from '@/app/components/container/message/notice-block/doc-notice-block';
import { CreateAssistant } from '@rapidaai/react';
import { connectionConfig } from '@/configs';
import { Globe } from 'lucide-react';
import { APiParameter } from '@/app/components/external-api/api-parameter';
import { InputHelper } from '@/app/components/input-helper';
import { CodeEditor } from '@/app/components/form/editor/code-editor';
import toast from 'react-hot-toast/headless';
import { SectionDivider } from '@/app/components/blocks/section-divider';
import { useTheme } from '@/theme/theme-provider';

const TRANSPORT_SECURITY_OPTIONS = [
  { name: 'Default', value: '' },
  { name: 'TLS', value: 'TLS' },
  { name: 'Plaintext', value: 'PLAINTEXT' },
];

const TLS_VERIFICATION_OPTIONS = [
  { name: 'Default', value: '' },
  { name: 'Verify', value: 'VERIFY' },
  { name: 'Skip verification', value: 'SKIP_VERIFY' },
];

const MILLISECONDS_PER_SECOND = 1000;
const BYTES_PER_MB = 1048576;
const DEFAULT_CONNECT_TIMEOUT_SECONDS = 10;
const DEFAULT_KEEPALIVE_TIME_SECONDS = 30;
const DEFAULT_KEEPALIVE_TIMEOUT_SECONDS = 10;
const DEFAULT_MAX_RECV_MESSAGE_MB = 16;
const DEFAULT_MAX_SEND_MESSAGE_MB = 16;

export function CreateAgentKit() {
  const { theme } = useTheme();
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

  //   steps for configuring agentkit
  const [activeTab, setActiveTab] = useState<
    'configure-agentkit' | 'define-assistant' | 'deployment'
  >('configure-agentkit');

  //
  const [errorMessage, setErrorMessage] = useState('');
  const [showAdvanced, setShowAdvanced] = useState(false);

  //
  const [name, setName] = useState(randomMeaningfullName('assistant'));
  const [description, setDescription] = useState('');
  const [tags, setTags] = useState<string[]>([]);
  const onAddTag = (tag: string) => {
    setTags([...tags, tag]);
  };
  const onRemoveTag = (tag: string) => {
    setTags(tags.filter(t => t !== tag));
  };
  const [agentKitUrl, setAgentKitUrl] = useState('');
  const [certificate, setCertificate] = useState('');
  const [transportSecurity, setTransportSecurity] = useState('');
  const [tlsVerification, setTlsVerification] = useState('');
  const [tlsServerName, setTlsServerName] = useState('');
  const [connectTimeoutSeconds, setConnectTimeoutSeconds] = useState('');
  const [keepaliveTimeSeconds, setKeepaliveTimeSeconds] = useState('');
  const [keepaliveTimeoutSeconds, setKeepaliveTimeoutSeconds] = useState('');
  const [maxRecvMessageMb, setMaxRecvMessageMb] = useState(
    DEFAULT_MAX_RECV_MESSAGE_MB.toString(),
  );
  const [maxSendMessageMb, setMaxSendMessageMb] = useState(
    DEFAULT_MAX_SEND_MESSAGE_MB.toString(),
  );
  const [parameters, setParameters] = useState<
    {
      key: string;
      value: string;
    }[]
  >([]);

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
    const assistantKit =
      new CreateAssistantProviderRequest.CreateAssistantProviderAgentkit();
    assistantKit.setAgentkiturl(agentKitUrl);
    assistantKit.setCertificate(certificate);

    if (transportSecurity) {
      assistantKit.setTransportsecurity(transportSecurity);
    }
    if (tlsVerification) {
      assistantKit.setTlsverification(tlsVerification);
    }
    if (tlsServerName.trim()) {
      assistantKit.setTlsservername(tlsServerName.trim());
    }
    if (connectTimeoutSeconds.trim()) {
      assistantKit.setConnecttimeoutms(
        Number(connectTimeoutSeconds) * MILLISECONDS_PER_SECOND,
      );
    }
    if (keepaliveTimeSeconds.trim()) {
      assistantKit.setKeepalivetimems(
        Number(keepaliveTimeSeconds) * MILLISECONDS_PER_SECOND,
      );
    }
    if (keepaliveTimeoutSeconds.trim()) {
      assistantKit.setKeepalivetimeoutms(
        Number(keepaliveTimeoutSeconds) * MILLISECONDS_PER_SECOND,
      );
    }
    if (maxRecvMessageMb.trim()) {
      assistantKit.setMaxrecvmessagebytes(
        Number(maxRecvMessageMb) * BYTES_PER_MB,
      );
    }
    if (maxSendMessageMb.trim()) {
      assistantKit.setMaxsendmessagebytes(
        Number(maxSendMessageMb) * BYTES_PER_MB,
      );
    }

    parameters.forEach(p => {
      assistantKit.getMetadataMap().set(p.key, p.value);
    });

    assistantProvider.setAgentkit(assistantKit);
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

  const validateAgentkit = (): boolean => {
    const grpcUrlPattern = /^[a-zA-Z0-9.-]+(:\d+)?$/; // Matches "hostname" or "hostname:port"
    const sslCertPattern =
      /^-----BEGIN CERTIFICATE-----[\s\S]+-----END CERTIFICATE-----$/;

    if (!grpcUrlPattern.test(agentKitUrl)) {
      setErrorMessage(
        'Illegal agentKit server url, please provide a valid host:port where agentkit is running.',
      );
      return false;
    }

    if (certificate === 'insecure' || certificate === 'skip-verify') {
      setErrorMessage(
        'Certificate must be a CA certificate. Use transport security and TLS verification for gRPC security options.',
      );
      return false;
    }

    if (certificate && !sslCertPattern.test(certificate)) {
      setErrorMessage(
        'Illegal certificate, please provide a valid certificate it should start with "-----BEGIN CERTIFICATE-----" and end with "-----END CERTIFICATE-----".',
      );
      return false;
    }

    if (
      transportSecurity &&
      !['TLS', 'PLAINTEXT'].includes(transportSecurity)
    ) {
      setErrorMessage('Transport security must be TLS or PLAINTEXT.');
      return false;
    }

    if (
      tlsVerification &&
      !['VERIFY', 'SKIP_VERIFY'].includes(tlsVerification)
    ) {
      setErrorMessage('TLS verification must be VERIFY or SKIP_VERIFY.');
      return false;
    }

    const numberFields = [
      {
        label: 'Connect timeout',
        value: connectTimeoutSeconds,
        min: 1,
        max: 300,
      },
      {
        label: 'Keepalive time',
        value: keepaliveTimeSeconds,
        min: 10,
        max: 3600,
      },
      {
        label: 'Keepalive timeout',
        value: keepaliveTimeoutSeconds,
        min: 1,
        max: 300,
      },
      {
        label: 'Max receive message MB',
        value: maxRecvMessageMb,
        min: 1,
        max: 100,
      },
      {
        label: 'Max send message MB',
        value: maxSendMessageMb,
        min: 1,
        max: 100,
      },
    ];

    for (const field of numberFields) {
      if (!field.value.trim()) {
        continue;
      }
      const parsedValue = Number(field.value);
      if (
        !Number.isInteger(parsedValue) ||
        parsedValue < field.min ||
        parsedValue > field.max
      ) {
        setErrorMessage(
          `${field.label} must be between ${field.min} and ${field.max}.`,
        );
        return false;
      }
    }

    const hasInvalidParameter = parameters.some(
      param => !param.key.trim() || !param.value.trim(),
    );
    if (hasInvalidParameter) {
      setErrorMessage('All parameters must have non-empty keys and values.');
      return false;
    }

    return true;
  };

  return (
    <>
      <Helmet title="Create an assistant"></Helmet>
      <ConfirmDialogComponent />
      <TabForm
        formHeading="Complete all steps to connect a new AgentKit."
        activeTab={activeTab}
        onChangeActiveTab={() => {}}
        errorMessage={errorMessage}
        form={[
          {
            code: 'configure-agentkit',
            name: 'Configuration',
            description:
              'Configure and connect the agent using an AgentKit endpoint.',
            body: (
              <>
                <DocNoticeBlock
                  docPath="/assistants/overview"
                  linkText="Read docs"
                >
                  Deploy your agent on-premises with the {theme.brand.name}{' '}
                  orchestration engine via AgentkitConnection.
                </DocNoticeBlock>
                <div className="px-8 pt-6 pb-8 max-w-4xl flex flex-col gap-8">
                  {/* Connection section */}
                  <div className="flex flex-col gap-6">
                    <SectionDivider label="Connection" />
                    <FieldSet className="relative w-full">
                      <FormLabel>AgentKit Endpoint</FormLabel>
                      <Input
                        placeholder="agent.your-domain.com:5051"
                        value={agentKitUrl}
                        onChange={v => {
                          setAgentKitUrl(v.target.value);
                        }}
                      />
                      <InputHelper>
                        The gRPC server address where your {theme.brand.name}{' '}
                        AgentKit is running.
                      </InputHelper>
                    </FieldSet>
                  </div>

                  <button
                    type="button"
                    onClick={() => setShowAdvanced(!showAdvanced)}
                    className="flex items-center gap-1.5 text-xs font-medium text-gray-500 hover:text-gray-800 dark:hover:text-gray-200 transition-colors"
                  >
                    <ChevronDown
                      size={16}
                      className={`transition-transform duration-200 ${
                        showAdvanced ? 'rotate-180' : ''
                      }`}
                    />
                    {showAdvanced ? 'Hide' : 'Show'} advanced settings
                  </button>

                  {showAdvanced && (
                    <div className="pt-6 border-t border-gray-200 dark:border-gray-800 flex flex-col gap-8">
                      <div className="flex flex-col gap-6">
                        <SectionDivider label="Connection tuning" />
                        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                          <FieldSet>
                            <div className="[&_.cds--slider-container]:!mt-0 [&_.cds--slider__range-label]:hidden">
                              <Slider
                                id="agentkit-connect-timeout-seconds"
                                labelText="Connect Timeout (seconds)"
                                min={1}
                                max={300}
                                step={1}
                                value={
                                  Number(connectTimeoutSeconds) ||
                                  DEFAULT_CONNECT_TIMEOUT_SECONDS
                                }
                                onChange={({ value }: { value: number }) => {
                                  setConnectTimeoutSeconds(value.toString());
                                }}
                              />
                            </div>
                          </FieldSet>
                          <FieldSet>
                            <div className="[&_.cds--slider-container]:!mt-0 [&_.cds--slider__range-label]:hidden">
                              <Slider
                                id="agentkit-keepalive-time-seconds"
                                labelText="Keepalive Time (seconds)"
                                min={10}
                                max={3600}
                                step={1}
                                value={
                                  Number(keepaliveTimeSeconds) ||
                                  DEFAULT_KEEPALIVE_TIME_SECONDS
                                }
                                onChange={({ value }: { value: number }) => {
                                  setKeepaliveTimeSeconds(value.toString());
                                }}
                              />
                            </div>
                          </FieldSet>
                          <FieldSet>
                            <div className="[&_.cds--slider-container]:!mt-0 [&_.cds--slider__range-label]:hidden">
                              <Slider
                                id="agentkit-keepalive-timeout-seconds"
                                labelText="Keepalive Timeout (seconds)"
                                min={1}
                                max={300}
                                step={1}
                                value={
                                  Number(keepaliveTimeoutSeconds) ||
                                  DEFAULT_KEEPALIVE_TIMEOUT_SECONDS
                                }
                                onChange={({ value }: { value: number }) => {
                                  setKeepaliveTimeoutSeconds(value.toString());
                                }}
                              />
                            </div>
                          </FieldSet>
                          <FieldSet>
                            <div className="[&_.cds--slider-container]:!mt-0 [&_.cds--slider__range-label]:hidden">
                              <Slider
                                id="agentkit-max-receive-message-mb"
                                labelText="Max Receive Message (MB)"
                                min={1}
                                max={100}
                                step={1}
                                value={
                                  Number(maxRecvMessageMb) ||
                                  DEFAULT_MAX_RECV_MESSAGE_MB
                                }
                                onChange={({ value }: { value: number }) => {
                                  setMaxRecvMessageMb(value.toString());
                                }}
                              />
                            </div>
                          </FieldSet>
                          <FieldSet>
                            <div className="[&_.cds--slider-container]:!mt-0 [&_.cds--slider__range-label]:hidden">
                              <Slider
                                id="agentkit-max-send-message-mb"
                                labelText="Max Send Message (MB)"
                                min={1}
                                max={100}
                                step={1}
                                value={
                                  Number(maxSendMessageMb) ||
                                  DEFAULT_MAX_SEND_MESSAGE_MB
                                }
                                onChange={({ value }: { value: number }) => {
                                  setMaxSendMessageMb(value.toString());
                                }}
                              />
                            </div>
                          </FieldSet>
                        </div>
                      </div>

                      <div className="flex flex-col gap-6">
                        <SectionDivider label="Security" />
                        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                          <FieldSet>
                            <FormLabel>Transport Security</FormLabel>
                            <Select
                              value={transportSecurity}
                              options={TRANSPORT_SECURITY_OPTIONS}
                              onChange={v => {
                                setTransportSecurity(v.target.value);
                              }}
                            />
                          </FieldSet>
                          <FieldSet>
                            <FormLabel>TLS Verification</FormLabel>
                            <Select
                              value={tlsVerification}
                              options={TLS_VERIFICATION_OPTIONS}
                              onChange={v => {
                                setTlsVerification(v.target.value);
                              }}
                            />
                          </FieldSet>
                          <FieldSet>
                            <FormLabel>TLS Server Name</FormLabel>
                            <Input
                              placeholder="agent.your-domain.com"
                              value={tlsServerName}
                              onChange={v => {
                                setTlsServerName(v.target.value);
                              }}
                            />
                          </FieldSet>
                        </div>
                        <FieldSet>
                          <FormLabel>TLS Certificate (Optional)</FormLabel>
                          <CodeEditor
                            placeholder="
                      -----BEGIN CERTIFICATE-----
...
-----END CERTIFICATE-----"
                            value={certificate}
                            onChange={value => {
                              setCertificate(value);
                            }}
                            className="min-h-40 max-h-dvh "
                          />
                        </FieldSet>
                      </div>
                    </div>
                  )}

                  <div className="flex flex-col gap-6">
                    <SectionDivider label="Metadata" />
                    <FieldSet>
                      <APiParameter
                        actionButtonLabel="Add Metadata"
                        setParameterValue={p => {
                          setParameters(p);
                        }}
                        initialValues={parameters}
                        inputClass="bg-light-background dark:bg-gray-950"
                      />
                    </FieldSet>
                  </div>
                </div>
              </>
            ),
            actions: [
              <ButtonSet className="!w-full [&>button]:!flex-1 [&>button]:!max-w-none">
                <SecondaryButton
                  size="lg"
                  onClick={() => showDialog(navigator.goBack)}
                >
                  Cancel
                </SecondaryButton>
                <PrimaryButton
                  size="lg"
                  isLoading={loading}
                  onClick={() => {
                    if (validateAgentkit()) setActiveTab('define-assistant');
                  }}
                >
                  Continue
                </PrimaryButton>
              </ButtonSet>,
            ],
          },

          {
            code: 'define-assistant',
            name: 'Profile',
            description:
              'Provide the name, a brief description, and relevant tags for your assistant to help identify and categorize it.',
            actions: [
              <ButtonSet className="!w-full [&>button]:!flex-1 [&>button]:!max-w-none">
                <SecondaryButton
                  size="lg"
                  onClick={() => showDialog(navigator.goBack)}
                >
                  Cancel
                </SecondaryButton>
                <PrimaryButton
                  size="lg"
                  isLoading={loading}
                  onClick={createAssistant}
                >
                  Continue
                </PrimaryButton>
              </ButtonSet>,
            ],
            body: (
              <div className="px-8 pt-8 pb-8 max-w-2xl flex flex-col gap-10">
                {/* Identity section */}
                <div className="flex flex-col gap-6">
                  <SectionDivider label="Identity" />

                  <FieldSet>
                    <FormLabel
                      htmlFor="agent_name"
                      className="text-xs tracking-wide uppercase"
                    >
                      Name{' '}
                      <span className="text-red-500 ml-0.5 normal-case">*</span>
                    </FormLabel>
                    <Input
                      name="agent_name"
                      onChange={e => {
                        setName(e.target.value);
                      }}
                      value={name}
                      placeholder="e.g. customer-support-assistant"
                    />
                    <InputHelper>
                      Provide a name that will appear in the assistant list and
                      help identify it.
                    </InputHelper>
                  </FieldSet>

                  <FieldSet>
                    <FormLabel
                      htmlFor="description"
                      className="text-xs tracking-wide uppercase"
                    >
                      Description (Optional)
                    </FormLabel>
                    <Textarea
                      row={4}
                      value={description}
                      placeholder="What's the purpose of the assistant?"
                      onChange={t => setDescription(t.target.value)}
                    />
                    <InputHelper>
                      Provide a description to explain what this assistant is
                      about.
                    </InputHelper>
                  </FieldSet>
                </div>

                {/* Labels section */}
                <div className="flex flex-col gap-6">
                  <SectionDivider label="Labels" />
                  <TagInput
                    tags={tags}
                    addTag={onAddTag}
                    removeTag={onRemoveTag}
                    allTags={AssistantTag}
                  />
                  <InputHelper>
                    Tags help you organize and filter assistants across your
                    workspace.
                  </InputHelper>
                </div>
              </div>
            ),
          },
          {
            code: 'deployment',
            name: 'Deployment',
            description: 'Enable the assistant to start engaging with users.',
            actions: [
              <ButtonSet className="!w-full [&>button]:!flex-1 [&>button]:!max-w-none">
                <SecondaryButton
                  size="lg"
                  onClick={() => {
                    if (assistant) goToAssistant(assistant.getId());
                  }}
                >
                  Skip
                </SecondaryButton>
                <PrimaryButton
                  size="lg"
                  isLoading={loading}
                  onClick={() => {
                    if (assistant) goToAssistant(assistant.getId());
                  }}
                >
                  Complete deployment
                </PrimaryButton>
              </ButtonSet>,
            ],
            body: (
              <>
                <DocNoticeBlock
                  docPath="/assistants/overview"
                  linkText="Read docs"
                >
                  Choose how you'd like to start engaging with users and add
                  advanced features to customize the user's experience.
                </DocNoticeBlock>
                <div className="px-8 pt-6 pb-8 flex flex-col gap-8">
                  {/* Deployments section */}
                  <div className="flex flex-col gap-6">
                    <SectionDivider label="Deployments" />
                    <dl className="bg-white dark:bg-gray-950">
                      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 divide-x divide-gray-200 dark:divide-gray-800">
                        <div className="border-y border-gray-200 dark:border-gray-800 grid grid-rows-[1fr_auto]">
                          <div className="px-4 py-2">
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
                          <button
                            onClick={() => {
                              if (assistant)
                                goToConfigureCall(assistant.getId());
                            }}
                            className="cursor-pointer flex justify-between items-center border-t border-gray-200 dark:border-gray-800 px-4 py-2 text-sm/6 text-blue-500 hover:bg-blue-600 hover:text-white transition-all delay-200"
                          >
                            Enable phone call
                            <ChevronRight className="w-4 h-4" />
                          </button>
                        </div>

                        <div className="border-y border-gray-200 dark:border-gray-800 grid grid-rows-[1fr_auto]">
                          <div className="px-4 py-2">
                            <Code
                              className="w-6 h-6 opacity-70 mt-4"
                              strokeWidth={1.5}
                            />
                            <div className="flex items-center gap-2 mt-4">
                              <h3 className="text-base/7 font-semibold">API</h3>
                            </div>
                            <p className="text-sm/6 text-gray-600 md:max-w-2xs dark:text-gray-400">
                              Integrate into your application using SDKs
                            </p>
                          </div>
                          <button
                            onClick={() => {
                              if (assistant)
                                goToConfigureApi(assistant.getId());
                            }}
                            className="cursor-pointer flex justify-between items-center border-t border-gray-200 dark:border-gray-800 px-4 py-2 text-sm/6 text-blue-500 hover:bg-blue-600 hover:text-white transition-all delay-200"
                          >
                            Enable API
                            <ChevronRight className="w-4 h-4" />
                          </button>
                        </div>

                        <div className="border-y border-gray-200 dark:border-gray-800 grid grid-rows-[1fr_auto]">
                          <div className="px-4 py-2">
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
                              customer queries.
                            </p>
                          </div>
                          <button
                            onClick={() => {
                              if (assistant)
                                goToConfigureWeb(assistant.getId());
                            }}
                            className="cursor-pointer flex justify-between items-center border-t border-gray-200 dark:border-gray-800 px-4 py-2 text-sm/6 text-blue-500 hover:bg-blue-600 hover:text-white transition-all delay-200"
                          >
                            Deploy to Web Widget
                            <ChevronRight className="w-4 h-4" />
                          </button>
                        </div>

                        <div className="border-y border-gray-200 dark:border-gray-800 grid grid-rows-[1fr_auto]">
                          <div className="px-4 py-2">
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
                          <button
                            onClick={() => {
                              if (assistant)
                                goToConfigureDebugger(assistant.getId());
                            }}
                            className="cursor-pointer flex justify-between items-center border-t border-gray-200 dark:border-gray-800 px-4 py-2 text-sm/6 text-blue-500 hover:bg-blue-600 hover:text-white transition-all delay-200"
                          >
                            Deploy to Debugger / Testing
                            <ChevronRight className="w-4 h-4" />
                          </button>
                        </div>
                      </div>
                    </dl>
                  </div>

                  {/* Analysis section */}
                  <div className="flex flex-col gap-6">
                    <SectionDivider label="Analysis" />
                    <div
                      className="bg-white dark:bg-gray-950 cursor-pointer"
                      onClick={() => {
                        if (assistant)
                          goToCreateAssistantAnalysis(assistant.getId());
                      }}
                    >
                      <div className="flex w-full justify-between gap-4 select-none border-y border-gray-200 dark:border-gray-800 px-4 py-3">
                        <div className="text-left text-sm/7 font-semibold text-pretty">
                          Gain insights from every interaction — automatic
                          conversation transcripts, quality, sentiment, and SOP
                          adherence analysis, custom reporting and dashboards.
                        </div>
                        <ChevronRight className="w-5 h-5" strokeWidth={1.5} />
                      </div>
                    </div>
                  </div>

                  {/* Webhook & Integration section */}
                  <div className="flex flex-col gap-6">
                    <SectionDivider label="Webhook & Integration" />
                    <div
                      className="bg-white dark:bg-gray-950 cursor-pointer"
                      onClick={() => {
                        if (assistant)
                          goToCreateAssistantWebhook(assistant.getId());
                      }}
                    >
                      <div className="flex w-full justify-between gap-4 select-none border-y border-gray-200 dark:border-gray-800 px-4 py-3">
                        <div className="text-left text-sm/7 font-semibold text-pretty">
                          Keep your workflows connected by triggering events
                          when key actions happen — conversation started /
                          ended, escalation to a human agent, custom events for
                          analytics or CRM sync.
                        </div>
                        <ChevronRight className="w-5 h-5" strokeWidth={1.5} />
                      </div>
                    </div>
                  </div>
                </div>
              </>
            ),
          },
        ]}
      />
    </>
  );
}
