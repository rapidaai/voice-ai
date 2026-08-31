import { FC, useEffect, useState } from 'react';
import { useRapidaStore } from '@/hooks';
import { useCredential } from '@/hooks/use-credential';
import { useParams } from 'react-router-dom';
import { Helmet } from '@/app/components/helmet';
import { PrimaryButton, SecondaryButton } from '@/app/components/carbon/button';
import {
  ButtonSet,
  Slider,
  Toggletip,
  ToggletipButton,
  ToggletipContent,
} from '@carbon/react';
import { ChevronDown, Information } from '@carbon/icons-react';
import { TabForm } from '@/app/components/form/tab-form';
import { FieldSet } from '@/app/components/form/fieldset';
import { useConfirmDialog } from '@/app/pages/assistant/actions/hooks/use-confirmation';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import {
  AssistantDefinition,
  ConnectionConfig,
  CreateAssistantProvider,
  CreateAssistantProviderRequest,
  GetAssistantProviderResponse,
  GetAssistantRequest,
} from '@rapidaai/react';
import { FormLabel } from '@/app/components/form-label';
import { Textarea } from '@/app/components/form/textarea';
import { ErrorContainer } from '@/app/components/error-container';
import { GetAssistant } from '@rapidaai/react';
import { connectionConfig } from '@/configs';
import { DocNoticeBlock } from '@/app/components/container/message/notice-block/doc-notice-block';
import { Input } from '@/app/components/form/input';
import { Select } from '@/app/components/form/select';
import { APiParameter } from '@/app/components/external-api/api-parameter';
import { CodeEditor } from '@/app/components/form/editor/code-editor';
import toast from 'react-hot-toast/headless';

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

export function CreateAgentKitVersion() {
  /**
   * get all the models when type change
   */
  const { assistantId } = useParams();
  const { goToAssistantListing } = useGlobalNavigation();

  if (!assistantId)
    return (
      <div className="flex flex-1">
        <ErrorContainer
          onAction={goToAssistantListing}
          code="403"
          actionLabel="Go to listing"
          title="Assistant not available"
          description="This assistant may be archived or you don't have access to it. Please check with your administrator or try another assistant."
        />
      </div>
    );

  return <CreateNewVersion assistantId={assistantId!} />;
}

/**
 *
 * @param props
 * @returns
 */
const CreateNewVersion: FC<{ assistantId: string }> = ({ assistantId }) => {
  const [userId, token, projectId] = useCredential();
  const [activeTab, setActiveTab] = useState('change-assistant');
  const navigator = useGlobalNavigation();
  const [errorMessage, setErrorMessage] = useState('');
  const [showAdvanced, setShowAdvanced] = useState(false);
  const { showDialog, ConfirmDialogComponent } = useConfirmDialog({});
  const currentDate = new Date().toLocaleDateString();
  const [versionMessage, setVersionMessage] = useState(
    `Changed on ${currentDate}`,
  );
  const { showLoader, hideLoader } = useRapidaStore();

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

  const validateAgentKit = (): boolean => {
    const grpcUrlPattern = /^[a-zA-Z0-9.-]+(:\d+)?$/; // Matches "hostname" or "hostname:port"
    const sslCertPattern =
      /^-----BEGIN CERTIFICATE-----[\s\S]+-----END CERTIFICATE-----$/;

    if (!grpcUrlPattern.test(agentKitUrl)) {
      setErrorMessage(
        'Invalid gRPC URL. It should follow the format "hostname" or "hostname:port".',
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
        'Invalid SSL certificate format. It should start with "-----BEGIN CERTIFICATE-----" and end with "-----END CERTIFICATE-----".',
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

  const createProviderModel = () => {
    setErrorMessage('');
    if (!versionMessage || versionMessage.trim() === '') {
      setErrorMessage('Please provide a valid version description.');
      return;
    }
    showLoader();
    const request = new CreateAssistantProviderRequest();
    const agentKit =
      new CreateAssistantProviderRequest.CreateAssistantProviderAgentkit();

    agentKit.setAgentkiturl(agentKitUrl);
    agentKit.setCertificate(certificate);
    if (transportSecurity) {
      agentKit.setTransportsecurity(transportSecurity);
    }
    if (tlsVerification) {
      agentKit.setTlsverification(tlsVerification);
    }
    if (tlsServerName.trim()) {
      agentKit.setTlsservername(tlsServerName.trim());
    }
    if (connectTimeoutSeconds.trim()) {
      agentKit.setConnecttimeoutms(
        Number(connectTimeoutSeconds) * MILLISECONDS_PER_SECOND,
      );
    }
    if (keepaliveTimeSeconds.trim()) {
      agentKit.setKeepalivetimems(
        Number(keepaliveTimeSeconds) * MILLISECONDS_PER_SECOND,
      );
    }
    if (keepaliveTimeoutSeconds.trim()) {
      agentKit.setKeepalivetimeoutms(
        Number(keepaliveTimeoutSeconds) * MILLISECONDS_PER_SECOND,
      );
    }
    if (maxRecvMessageMb.trim()) {
      agentKit.setMaxrecvmessagebytes(Number(maxRecvMessageMb) * BYTES_PER_MB);
    }
    if (maxSendMessageMb.trim()) {
      agentKit.setMaxsendmessagebytes(Number(maxSendMessageMb) * BYTES_PER_MB);
    }

    parameters.forEach(p => {
      agentKit.getMetadataMap().set(p.key, p.value);
    });

    //
    request.setAgentkit(agentKit);
    request.setAssistantid(assistantId);
    request.setDescription(versionMessage);
    CreateAssistantProvider(connectionConfig, request, {
      authorization: token,
      'x-auth-id': userId,
      'x-project-id': projectId,
    })
      .then((car: GetAssistantProviderResponse) => {
        hideLoader();
        if (car?.getSuccess()) {
          toast.success(
            'Assistant version with model has been created successfully.',
          );
          navigator.goToAssistantVersions(assistantId);
        } else {
          const errorMessage =
            'Unable to create assistant version. please try again later.';
          const error = car?.getError();
          if (error) {
            setErrorMessage(error.getHumanmessage());
            return;
          }
          setErrorMessage(errorMessage);
          return;
        }
      })
      .catch(err => {
        setErrorMessage(
          'Unable to create assistant version. please try again later.',
        );
      });
  };

  useEffect(() => {
    showLoader();
    const request = new GetAssistantRequest();
    const assistantDef = new AssistantDefinition();
    assistantDef.setAssistantid(assistantId);
    request.setAssistantdefinition(assistantDef);
    GetAssistant(
      connectionConfig,
      request,
      ConnectionConfig.WithDebugger({
        authorization: token,
        userId: userId,
        projectId: projectId,
      }),
    )
      .then(response => {
        hideLoader();
        if (response?.getSuccess()) {
          const assistantProvider = response
            .getData()
            ?.getAssistantprovideragentkit();
          if (assistantProvider) {
            setAgentKitUrl(assistantProvider.getUrl());
            setCertificate(assistantProvider.getCertificate());
            setTransportSecurity(assistantProvider.getTransportsecurity());
            setTlsVerification(assistantProvider.getTlsverification());
            setTlsServerName(assistantProvider.getTlsservername());
            const connectTimeoutMs = assistantProvider.getConnecttimeoutms();
            const keepaliveTimeMs = assistantProvider.getKeepalivetimems();
            const keepaliveTimeoutMs =
              assistantProvider.getKeepalivetimeoutms();
            const maxRecvMessageBytes =
              assistantProvider.getMaxrecvmessagebytes();
            const maxSendMessageBytes =
              assistantProvider.getMaxsendmessagebytes();
            setConnectTimeoutSeconds(
              connectTimeoutMs
                ? String(Math.round(connectTimeoutMs / MILLISECONDS_PER_SECOND))
                : '',
            );
            setKeepaliveTimeSeconds(
              keepaliveTimeMs
                ? String(Math.round(keepaliveTimeMs / MILLISECONDS_PER_SECOND))
                : '',
            );
            setKeepaliveTimeoutSeconds(
              keepaliveTimeoutMs
                ? String(
                    Math.round(keepaliveTimeoutMs / MILLISECONDS_PER_SECOND),
                  )
                : '',
            );
            setMaxRecvMessageMb(
              maxRecvMessageBytes
                ? String(Math.round(maxRecvMessageBytes / BYTES_PER_MB))
                : DEFAULT_MAX_RECV_MESSAGE_MB.toString(),
            );
            setMaxSendMessageMb(
              maxSendMessageBytes
                ? String(Math.round(maxSendMessageBytes / BYTES_PER_MB))
                : DEFAULT_MAX_SEND_MESSAGE_MB.toString(),
            );
            const _parameters: { key: string; value: string }[] = [];
            assistantProvider.getMetadataMap().forEach((value, key) => {
              _parameters.push({ key, value });
            });
            setParameters(_parameters);
          }
          return;
        }
        const error = response?.getError();
        const errorMsg = error
          ? error.getHumanmessage()
          : 'Unable to get your assistant. Please try again later.';
        setErrorMessage(errorMsg);
      })
      .catch(err => {
        hideLoader();
        setErrorMessage(
          'Unable to get your assistant. Please try again later.',
        );
      });
  }, [assistantId]);

  return (
    <>
      <ConfirmDialogComponent />
      <Helmet title="Connect AgentKit"></Helmet>
      <TabForm
        formHeading="Create a new version of this AgentKit connection."
        activeTab={activeTab}
        onChangeActiveTab={() => {}}
        errorMessage={errorMessage}
        form={[
          {
            code: 'change-assistant',
            name: 'Connect configuration',
            description:
              'Provide the connection configuration for your Rapida AgentKit setup.',
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
                  onClick={() => {
                    if (validateAgentKit()) {
                      setActiveTab('commit-assistant');
                    }
                  }}
                >
                  Continue
                </PrimaryButton>
              </ButtonSet>,
            ],
            body: (
              <>
                <DocNoticeBlock
                  docPath="/assistants/create-new-version"
                  linkText="Read docs"
                >
                  New versions of the assistant will not be deployed
                  automatically. You must promote them manually.
                </DocNoticeBlock>
                <div className="px-4 pt-6 pb-8 max-w-4xl flex flex-col gap-8">
                  {/* Connection section */}
                  <div className="flex flex-col gap-6">
                    <FieldSet className="relative w-full">
                      <div className="mb-2 flex items-center gap-1">
                        <FormLabel htmlFor="agentkit-endpoint">
                          AgentKit Endpoint
                        </FormLabel>
                        <Toggletip align="right">
                          <ToggletipButton label="Show information">
                            <Information size={14} />
                          </ToggletipButton>
                          <ToggletipContent>
                            The gRPC server address where your Rapida AgentKit
                            is running.
                          </ToggletipContent>
                        </Toggletip>
                      </div>
                      <Input
                        name="agentkit-endpoint"
                        placeholder="agent.your-domain.com:5051"
                        value={agentKitUrl}
                        onChange={v => {
                          setAgentKitUrl(v.target.value);
                        }}
                      />
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
                        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                          <FieldSet>
                            <div className="[&_.cds--slider-container]:!mt-0 [&_.cds--slider__range-label]:hidden">
                              <Slider
                                id="agentkit-version-connect-timeout-seconds"
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
                                id="agentkit-version-keepalive-time-seconds"
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
                                id="agentkit-version-keepalive-timeout-seconds"
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
                                id="agentkit-version-max-receive-message-mb"
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
                                id="agentkit-version-max-send-message-mb"
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
                            placeholder="-----BEGIN CERTIFICATE-----
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
                    <FieldSet>
                      <div className="mb-2 flex items-center gap-1">
                        <FormLabel>Metadata</FormLabel>
                        <Toggletip align="right">
                          <ToggletipButton label="Show information">
                            <Information size={14} />
                          </ToggletipButton>
                          <ToggletipContent>
                            Additional key-value metadata sent with the AgentKit
                            connection.
                          </ToggletipContent>
                        </Toggletip>
                      </div>
                      <APiParameter
                        actionButtonLabel="Add Metadata"
                        setParameterValue={parameters => {
                          setParameters(parameters);
                        }}
                        initialValues={parameters}
                        inputClass="bg-light-background dark:bg-gray-950"
                      />
                    </FieldSet>
                  </div>
                </div>
              </>
            ),
          },
          {
            code: 'commit-assistant',
            name: 'Version note',
            description:
              'Write a brief note describing what changed in this version.',
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
                  onClick={() => {
                    createProviderModel();
                  }}
                >
                  Create new version
                </PrimaryButton>
              </ButtonSet>,
            ],
            body: (
              <div className="px-8 pt-8 pb-8 max-w-2xl flex flex-col gap-10">
                <div className="flex flex-col gap-6">
                  <FieldSet>
                    <div className="mb-2 flex items-center gap-1">
                      <FormLabel htmlFor="agentkit-version-note">
                        Version note
                      </FormLabel>
                      <Toggletip align="right">
                        <ToggletipButton label="Show information">
                          <Information size={14} />
                        </ToggletipButton>
                        <ToggletipContent>
                          Briefly describe what changed in this version.
                        </ToggletipContent>
                      </Toggletip>
                    </div>
                    <Textarea
                      name="agentkit-version-note"
                      row={5}
                      value={versionMessage}
                      placeholder="Provide a clear and detailed explanation of the changes made to this AgentKit connection."
                      onChange={t => setVersionMessage(t.target.value)}
                    />
                  </FieldSet>
                </div>
              </div>
            ),
          },
        ]}
      />
    </>
  );
};
