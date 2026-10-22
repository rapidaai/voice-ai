import {
  PrimaryButton,
  GhostButton,
  IconOnlyButton,
} from '@/app/components/ui/primitives';
import { Dropdown } from '@/app/components/ui/primitives';
import { Form, Stack, TextInput } from '@/app/components/ui/primitives';
import { ArrowLeft, Close, Model, PhoneOutgoing } from '@carbon/icons-react';
import { Notification } from '@/app/components/ui/feedback';
import { Text } from '@/app/components/ui/primitives';
import {
  ArgumentList,
  ConfigEmpty,
  ConfigBlock,
  DebuggerTabHeader,
  InfoRow,
  PreviewAgentHeader,
  VoiceAgent,
} from '@/app/pages/preview-agent/voice-agent/voice-agent';
import {
  PHONE_COUNTRIES,
  DEFAULT_COUNTRY,
  Country,
} from '@/app/pages/preview-agent/voice-agent/phone-agent-constants';
import { useCurrentCredential } from '@/hooks/use-credential';
import { randomMeaningfullName } from '@/utils';
import { getStatusMetric } from '@/utils/metadata';
import {
  AgentConfig,
  Channel,
  InputOptions,
  StringToAny,
  Assistant,
  Variable,
} from '@rapidaai/react';
import { getAssistantByIdWithApi } from '@/clients/assistant.client';
import {
  createPreviewPhoneCall,
  createVoiceAgentDebuggerConnection,
  createVoiceAgentSDKConnection,
} from '@/clients/runtime.client';
import React, { useCallback, useEffect, useState } from 'react';
import { Navigate, useParams, useSearchParams } from 'react-router-dom';

/**
 *
 * @returns
 */
export const PublicPreviewVoiceAgent = () => {
  const [searchParams] = useSearchParams();
  const { assistantId } = useParams();
  const authId = searchParams.get('authId');
  const token = searchParams.get('token');

  if (!assistantId || !token) {
    return <Navigate to="/404" replace />;
  }

  return (
    <VoiceAgent
      debug={false}
      connectConfig={createVoiceAgentSDKConnection({
        apiKey: token,
        userId: '' + (authId || 'public_user'),
      })}
      agentConfig={new AgentConfig(
        assistantId,
        new InputOptions([Channel.Audio, Channel.Text], Channel.Text),
      )
        .addMetadata('authId', StringToAny('' + (authId || 'public_user')))
        .setUserIdentifier(authId || randomMeaningfullName('public'))}
    />
  );
};

//
export const PreviewVoiceAgent = () => {
  const { user, authId, token, projectId } = useCurrentCredential();
  const { assistantId } = useParams();

  if (!assistantId || !user?.name) {
    return <Navigate to="/404" replace />;
  }

  return (
    <VoiceAgent
      debug={true}
      connectConfig={createVoiceAgentDebuggerConnection({
        token,
        userId: authId,
        projectId,
      })}
      agentConfig={new AgentConfig(
        assistantId,
        new InputOptions([Channel.Audio, Channel.Text], Channel.Text),
      )
        .setUserIdentifier(authId, user.name)
        .addKeywords([user.name])
        .addMetadata('authId', StringToAny(authId))
        .addMetadata('projectId', StringToAny(projectId))}
      // .addCustomOption('listen.language', StringToAny('en'))
      // .addCustomOption('speak.language', StringToAny('en'))
      // .addCustomOption('listen.model', StringToAny('nova-3'))}
    />
  );
};

// ---------------------------------------------------------------------------
// Phone Agent
// ---------------------------------------------------------------------------

type PhoneCallStatus = 'idle' | 'calling' | 'success' | 'failed';

//
export const PreviewPhoneAgent = ({
  embedded = false,
}: {
  embedded?: boolean;
}) => {
  const { authId, token, projectId } = useCurrentCredential();

  const { assistantId } = useParams();
  const [assistant, setAssistant] = useState<Assistant | null>(null);
  const [variables, setVariables] = useState<Variable[]>([]);
  const [country, setCountry] = useState<Country>(DEFAULT_COUNTRY);
  const [phoneNumber, setPhoneNumber] = useState('');
  const [callStatus, setCallStatus] = useState<PhoneCallStatus>('idle');
  const [errorMessage, setErrorMessage] = useState('');
  const [isConfigurationOpen, setIsConfigurationOpen] = useState(false);
  const [argumentMap, setArgumentMap] = useState<Map<string, string>>(
    new Map(),
  );

  const onChangeArgument = useCallback((k: string, vl: string) => {
    setArgumentMap(prev => {
      const m = new Map(prev);
      m.set(k, vl);
      return m;
    });
  }, []);

  useEffect(() => {
    if (!assistantId) return;
    let isMounted = true;
    setAssistant(null);
    setVariables([]);
    setArgumentMap(new Map());
    setErrorMessage('');

    getAssistantByIdWithApi({
      assistantId,
      auth: {
        token,
        projectId,
        userId: authId,
      },
    })
      .then(response => {
        if (!isMounted) return;
        if (response?.getSuccess()) {
          setAssistant(response.getData()!);
          const pmtVars = response
            .getData()
            ?.getAssistantprovidermodel()
            ?.getTemplate()
            ?.getPromptvariablesList();
          if (pmtVars) {
            setVariables(pmtVars);
            const defaults = new Map<string, string>();
            pmtVars.forEach(v => {
              if (v.getDefaultvalue()) {
                defaults.set(v.getName(), v.getDefaultvalue());
              }
            });
            setArgumentMap(defaults);
          }
        }
      })
      .catch(() => {
        if (isMounted) {
          setErrorMessage('Unable to load assistant configuration.');
        }
      });

    return () => {
      isMounted = false;
    };
  }, [assistantId, authId, projectId, token]);

  if (!assistantId) {
    return <Navigate to="/404" replace />;
  }

  const validatePhoneNumber = () => {
    if (!country.name) {
      setErrorMessage('Please select a country.');
      return false;
    }
    if (
      (country.name !== 'Other' && phoneNumber.length < 7) ||
      phoneNumber.length > 15
    ) {
      setErrorMessage('Please enter a valid phone number.');
      return false;
    }
    return true;
  };

  const handleSubmit = () => {
    if (!validatePhoneNumber()) return;
    setErrorMessage('');
    setCallStatus('calling');

    createPreviewPhoneCall({
      assistantId,
      toNumber: country.value + phoneNumber,
      args: argumentMap,
      auth: {
        token,
        userId: authId,
        projectId,
      },
    })
      .then(x => {
        if (x.getSuccess()) {
          const status = getStatusMetric(x.getData()?.getMetricsList());
          if (status === 'FAILED') {
            setCallStatus('failed');
            setErrorMessage('Unable to start the call, please try again.');
            return;
          }
          setCallStatus('success');
          return;
        }
        setCallStatus('failed');
        const err = x.getError();
        setErrorMessage(
          err?.getHumanmessage() ||
            'Unable to start the call, please try again.',
        );
      })
      .catch(() => {
        setCallStatus('failed');
        setErrorMessage('Unable to start the call, please try again.');
      });
  };

  const handleReset = () => {
    setPhoneNumber('');
    setCallStatus('idle');
    setErrorMessage('');
  };

  const deployment = assistant?.getPhonedeployment() ?? null;
  const stt = deployment?.getInputaudio() ?? null;
  const tts = deployment?.getOutputaudio() ?? null;
  const model = assistant?.getAssistantprovidermodel() ?? null;

  return (
    <div
      className={`flex min-h-0 w-full flex-col text-sm/6 ${
        embedded ? 'h-full' : 'h-screen'
      }`}
    >
      {!embedded ? <PreviewAgentHeader /> : null}
      <div
        role="toolbar"
        aria-label="Phone debugger toolbar"
        className="flex h-12 shrink-0 items-center border-b border-[var(--cds-border-subtle-01)] bg-[var(--cds-layer-01)]"
      >
        <h1 className="min-w-0 flex-1 truncate px-4 text-sm font-semibold text-[var(--cds-text-primary)]">
          {assistant?.getName() || 'Phone debugger'}
        </h1>
        <IconOnlyButton
          kind="ghost"
          size="md"
          renderIcon={Model}
          iconDescription="Configuration"
          aria-controls="phone-debugger-details-panel"
          aria-expanded={isConfigurationOpen}
          aria-pressed={isConfigurationOpen}
          className={`!h-12 !w-12 shrink-0 !border-b-2 ${
            isConfigurationOpen
              ? '!border-b-[var(--cds-border-interactive)] !bg-[var(--cds-layer-selected-01)]'
              : '!border-b-transparent'
          }`}
          onClick={() => setIsConfigurationOpen(open => !open)}
        />
      </div>

      <div className="relative flex min-h-0 flex-1">
        <main className="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden bg-[var(--cds-background)]">
          {!embedded ? (
            <div className="flex items-center gap-1.5 px-3 py-2 border-b border-gray-200 dark:border-gray-800 shrink-0">
              <IconOnlyButton
                kind="ghost"
                size="sm"
                renderIcon={ArrowLeft}
                iconDescription="Back to Assistant"
                onClick={() => {
                  window.location.href = `/deployment/assistant/${assistantId}/overview`;
                }}
              />
              <span className="text-sm font-medium text-gray-900 dark:text-gray-100">
                Back to Assistant
              </span>
            </div>
          ) : null}

          <Form
            className="flex min-h-0 flex-1 flex-col"
            onSubmit={e => e.preventDefault()}
          >
            <div className="flex flex-1 items-center justify-center px-5 md:px-8">
              <Stack gap={7} className="w-full max-w-[42rem]">
                <div className="space-y-1">
                  <Text
                    as="h2"
                    isLoading={!assistant}
                    heading
                    skeletonWidth="60%"
                    className="text-xl font-semibold text-gray-900 dark:text-gray-100"
                  >
                    Debug Phone Call
                  </Text>
                  <Text
                    as="p"
                    isLoading={!assistant}
                    skeletonWidth="80%"
                    className="text-[var(--cds-text-secondary)]"
                  >
                    Place a live test call to validate your phone deployment
                    end-to-end.
                  </Text>
                </div>

                <div>
                  <Text as="label" className="block mb-2 text-sm font-medium">
                    Phone number
                  </Text>
                  <div className="grid grid-cols-1 gap-3 sm:grid-cols-[16rem_minmax(0,1fr)]">
                    <Dropdown<Country>
                      id="phone-country"
                      titleText=""
                      label="Select country code"
                      items={PHONE_COUNTRIES}
                      selectedItem={country}
                      onChange={({ selectedItem }) =>
                        setCountry(selectedItem ?? DEFAULT_COUNTRY)
                      }
                      itemToString={item =>
                        item ? `${item.name} (${item.value})` : ''
                      }
                      hideLabel
                    />
                    <TextInput
                      id="phone-number"
                      labelText="Phone number"
                      hideLabel
                      type="tel"
                      placeholder="Enter your phone number"
                      value={phoneNumber}
                      onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                        setPhoneNumber(e.target.value);
                        setErrorMessage('');
                      }}
                      invalid={Boolean(errorMessage)}
                    />
                  </div>
                </div>

                {errorMessage && (
                  <Notification
                    kind="error"
                    title="Error"
                    subtitle={errorMessage}
                  />
                )}

                {callStatus === 'success' && (
                  <Notification
                    kind="success"
                    title="Success"
                    subtitle="Call has been created successfully."
                  />
                )}
              </Stack>
            </div>

            <div className="flex h-12 shrink-0 items-stretch justify-end border-t border-[var(--cds-border-subtle-01)] bg-[var(--cds-layer-01)]">
              {callStatus === 'success' ? (
                <GhostButton
                  size="md"
                  className="mr-auto !h-12"
                  onClick={handleReset}
                >
                  Make another call
                </GhostButton>
              ) : null}
              <PrimaryButton
                size="md"
                className="!h-12"
                renderIcon={PhoneOutgoing}
                onClick={handleSubmit}
                isLoading={callStatus === 'calling'}
              >
                Start Call
              </PrimaryButton>
            </div>
          </Form>
        </main>

        {isConfigurationOpen ? (
          <>
            <button
              type="button"
              aria-label="Close configuration overlay"
              className="absolute inset-0 z-10 bg-[var(--cds-overlay)] xl:hidden"
              onClick={() => setIsConfigurationOpen(false)}
            />
            <aside
              id="phone-debugger-details-panel"
              aria-label="Session configuration"
              className="absolute inset-y-0 right-0 z-20 flex w-full flex-col border-l border-[var(--cds-border-subtle-01)] bg-[var(--cds-layer-01)] sm:w-[24rem] xl:relative xl:z-auto"
            >
              <div className="flex h-12 shrink-0 items-center border-b border-[var(--cds-border-subtle-01)] px-4">
                <h2 className="min-w-0 flex-1 truncate text-sm font-semibold text-[var(--cds-text-primary)]">
                  Session configuration
                </h2>
                <IconOnlyButton
                  kind="ghost"
                  size="sm"
                  renderIcon={Close}
                  iconDescription="Close configuration"
                  onClick={() => setIsConfigurationOpen(false)}
                />
              </div>
              <PhoneAgentDebugger
                assistant={assistant}
                deployment={deployment ? deployment : undefined}
                stt={stt}
                tts={tts}
                model={model}
                variables={variables}
                onChangeArgument={onChangeArgument}
              />
            </aside>
          </>
        ) : null}
      </div>
    </div>
  );
};

// ---------------------------------------------------------------------------
// Phone Agent Debugger (right panel)
// ---------------------------------------------------------------------------

const PhoneAgentDebugger: React.FC<{
  assistant: Assistant | null;
  deployment: ReturnType<Assistant['getPhonedeployment']>;
  stt: any;
  tts: any;
  model: any;
  variables: Variable[];
  onChangeArgument: (k: string, v: string) => void;
}> = ({
  assistant,
  deployment,
  stt,
  tts,
  model,
  variables,
  onChangeArgument,
}) => {
  const loading = !assistant;
  const inputMode = 'Text' + (deployment?.getInputaudio() ? ', Audio' : '');
  const outputMode = 'Text' + (deployment?.getOutputaudio() ? ', Audio' : '');

  return (
    <div className="flex flex-col h-full overflow-hidden text-sm">
      <div className="flex-1 min-h-0 overflow-y-auto">
        <ConfigBlock title="deployment" isLoading={loading} skeletonRows={4}>
          {assistant && (
            <>
              <InfoRow
                label="telephony"
                value={deployment?.getPhoneprovidername() || 'not configured'}
              />
              <InfoRow
                label="model"
                value={model?.getModelprovidername() || 'not configured'}
              />
              <InfoRow label="input mode" value={inputMode} />
              <InfoRow label="output mode" value={outputMode} />
            </>
          )}
        </ConfigBlock>

        <ConfigBlock title="assistant" isLoading={loading} skeletonRows={2}>
          {assistant && (
            <>
              <InfoRow label="name" value={assistant.getName()} />
              <InfoRow label="arguments" value={String(variables.length)} />
              {assistant.getDescription() && (
                <InfoRow
                  label="description"
                  value={assistant.getDescription()}
                />
              )}
            </>
          )}
        </ConfigBlock>

        <ConfigBlock title="stt" isLoading={loading} skeletonRows={2}>
          {stt ? (
            <>
              <InfoRow label="provider" value={stt.getAudioprovider()} />
              {stt.getAudiooptionsList().map((d: any) => (
                <InfoRow
                  key={d.getKey()}
                  label={d.getKey()}
                  value={d.getValue()}
                />
              ))}
            </>
          ) : (
            <ConfigEmpty label="Audio input is not configured." />
          )}
        </ConfigBlock>

        <ConfigBlock title="tts" isLoading={loading} skeletonRows={2}>
          {tts ? (
            <>
              <InfoRow label="provider" value={tts.getAudioprovider()} />
              {tts.getAudiooptionsList().map((d: any) => (
                <InfoRow
                  key={d.getKey()}
                  label={d.getKey()}
                  value={d.getValue()}
                />
              ))}
            </>
          ) : (
            <ConfigEmpty label="Audio output is not configured." />
          )}
        </ConfigBlock>

        <ConfigBlock title="llm" isLoading={loading} skeletonRows={2}>
          {model ? (
            <>
              <InfoRow label="provider" value={model.getModelprovidername()} />
              {model.getAssistantmodeloptionsList().map((m: any) => (
                <InfoRow
                  key={m.getKey()}
                  label={m.getKey()}
                  value={m.getValue()}
                />
              ))}
            </>
          ) : (
            <ConfigEmpty label="Model configuration is not available." />
          )}
        </ConfigBlock>
        <DebuggerTabHeader
          title="Arguments"
          subtitle={`${variables.length} prompt variables`}
        />
        <ArgumentList
          variables={variables}
          onChangeArgument={onChangeArgument}
        />
      </div>
    </div>
  );
};
