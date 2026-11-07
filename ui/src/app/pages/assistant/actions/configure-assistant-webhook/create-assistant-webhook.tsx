import React, { FC, useState } from 'react';
import { useConfirmDialog } from '@/app/pages/assistant/actions/hooks/use-confirmation';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { PrimaryButton, SecondaryButton } from '@/app/components/ui/primitives';
import { TextInput, TextArea, Stack } from '@/app/components/ui/primitives';
import { InputGroup } from '@/app/components/ui/primitives';
import {
  ButtonSet,
  Select as CarbonSelect,
  SelectItem,
  NumberInput,
  Checkbox,
  Tooltip,
} from '@carbon/react';
import { Information } from '@carbon/icons-react';
import { Slider } from '@/app/components/ui/primitives';
import { APiHeader } from '@/app/components/domain/external-api/api-header';
import { Metadata } from '@rapidaai/react';
import { useCurrentCredential } from '@/hooks/use-credential';
import toast from 'react-hot-toast/headless';
import { useRapidaStore } from '@/stores/app';
import { CreateFullPageWithHeader } from '@/app/components/ui/composites';
import { WebhookEventSelector } from './webhook-event-selector';
import { WebhookEventGroup, webhookEvents } from './webhook-events';
import { createAssistantConfigurationForAssistant } from '@/clients/assistant.client';

const renderLabelWithTooltip = (label: string, tooltip: string) => (
  <span className="inline-flex items-center gap-1">
    {label}
    <Tooltip align="right" label={tooltip}>
      <Information size={14} />
    </Tooltip>
  </span>
);

const WEBHOOK_OPTION_KEYS = {
  method: 'http_method',
  url: 'http_url',
  headers: 'http_headers',
  retryStatusCodes: 'retry_status_codes',
  maxRetryCount: 'max_retry_count',
  timeoutSeconds: 'timeout_seconds',
  events: 'assistant_events',
  executionPriority: 'execution_priority',
  description: 'description',
};

const webhookConfigurationType = 'webhook';

const getEventGroupTitle = (
  group: WebhookEventGroup,
  selectedEvents: string[],
) => {
  const groupEvents = webhookEvents.filter(event => event.group === group);
  const selectedCount = groupEvents.filter(event =>
    selectedEvents.includes(event.id),
  ).length;
  return `${group} Events (${selectedCount}/${groupEvents.length})`;
};

const toJsonMap = (rows: { key: string; value: string }[]) => {
  return JSON.stringify(
    rows.reduce<Record<string, string>>((acc, current) => {
      if (!current.key) {
        return acc;
      }
      acc[current.key] = current.value;
      return acc;
    }, {}),
  );
};

const buildWebhookOptions = ({
  method,
  endpoint,
  headers,
  retryOnStatus,
  maxRetries,
  requestTimeout,
  priority,
  events,
  description,
}: {
  method: string;
  endpoint: string;
  headers: { key: string; value: string }[];
  retryOnStatus: string[];
  maxRetries: number;
  requestTimeout: number;
  priority: number;
  events: string[];
  description: string;
}): Metadata[] => {
  return [
    { key: WEBHOOK_OPTION_KEYS.method, value: method || 'POST' },
    { key: WEBHOOK_OPTION_KEYS.url, value: endpoint || '' },
    { key: WEBHOOK_OPTION_KEYS.headers, value: toJsonMap(headers) },
    {
      key: WEBHOOK_OPTION_KEYS.retryStatusCodes,
      value: JSON.stringify(retryOnStatus || []),
    },
    { key: WEBHOOK_OPTION_KEYS.maxRetryCount, value: String(maxRetries || 0) },
    {
      key: WEBHOOK_OPTION_KEYS.timeoutSeconds,
      value: String(requestTimeout || 0),
    },
    { key: WEBHOOK_OPTION_KEYS.events, value: JSON.stringify(events || []) },
    {
      key: WEBHOOK_OPTION_KEYS.executionPriority,
      value: String(priority || 0),
    },
    { key: WEBHOOK_OPTION_KEYS.description, value: description || '' },
  ].map(({ key, value }) => {
    const option = new Metadata();
    option.setKey(key);
    option.setValue(value);
    return option;
  });
};

export const CreateAssistantWebhook: FC<{ assistantId: string }> = ({
  assistantId,
}) => {
  const navigator = useGlobalNavigation();
  const { authId, token, projectId } = useCurrentCredential();
  const { showDialog, ConfirmDialogComponent } = useConfirmDialog({});
  const { loading, showLoader, hideLoader } = useRapidaStore();

  const [activeTab, setActiveTab] = useState('events');
  const [errorMessage, setErrorMessage] = useState('');

  const [method, setMethod] = useState('POST');
  const [endpoint, setEndpoint] = useState('');
  const [description, setDescription] = useState('');
  const [retryOnStatus, setRetryOnStatus] = useState<string[]>(['50X']);
  const [maxRetries, setMaxRetries] = useState(3);
  const [requestTimeout, setRequestTimeout] = useState(180);
  const [headers, setHeaders] = useState<{ key: string; value: string }[]>([]);
  const [priority, setPriority] = useState<number>(0);
  const [events, setEvents] = useState<string[]>([]);

  const validateEvents = (): boolean => {
    setErrorMessage('');
    if (events.length === 0) {
      setErrorMessage(
        'Please select at least one event when the webhook will get triggered.',
      );
      return false;
    }
    return true;
  };

  const validateDestination = (): boolean => {
    setErrorMessage('');
    if (!endpoint) {
      setErrorMessage('Please provide a server URL for the webhook.');
      return false;
    }
    if (!/^https?:\/\/.+/.test(endpoint)) {
      setErrorMessage('Please provide a valid server URL for the webhook.');
      return false;
    }
    return true;
  };

  const validateHeaders = (): boolean => {
    setErrorMessage('');
    const headersMissingValue = headers.some(
      header => header.key.trim() !== '' && header.value.trim() === '',
    );
    if (headersMissingValue) {
      setErrorMessage('Headers with a key must also include a value.');
      return false;
    }
    return true;
  };

  const onSubmit = async () => {
    setErrorMessage('');
    if (!validateEvents() || !validateDestination() || !validateHeaders()) {
      return;
    }
    showLoader();
    const options = buildWebhookOptions({
      method,
      endpoint,
      headers,
      retryOnStatus,
      maxRetries,
      requestTimeout,
      priority,
      events,
      description,
    });

    try {
      const response = await createAssistantConfigurationForAssistant({
        assistantId,
        configurationType: webhookConfigurationType,
        provider: 'http',
        enabled: true,
        options,
        auth: { projectId, token, userId: authId },
      });

      hideLoader();
      if (response?.getSuccess()) {
        toast.success(`Assistant's webhook created successfully`);
        navigator.goToAssistantWebhook(assistantId);
        return;
      }
      if (response?.getError()) {
        const message = response.getError()?.getHumanmessage();
        if (message) {
          setErrorMessage(message);
          return;
        }
      }
      setErrorMessage(
        'Unable to create assistant webhook, please check and try again.',
      );
    } catch {
      hideLoader();
      setErrorMessage(
        'Unable to create assistant webhook, please check and try again.',
      );
    }
  };

  return (
    <>
      <ConfirmDialogComponent />
      <CreateFullPageWithHeader
        title="Configure webhook"
        eyebrow="Assistant"
        description="Complete all steps to configure your webhook."
        activeStep={activeTab}
        onChangeStep={() => {}}
        errorMessage={errorMessage}
        steps={[
          {
            code: 'events',
            name: 'Events',
            description:
              'Choose which call, conversation, and WebRTC events trigger the webhook.',
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
                    if (validateEvents()) setActiveTab('delivery');
                  }}
                >
                  Continue
                </PrimaryButton>
              </ButtonSet>,
            ],
            body: (
              <div className="pb-8 flex flex-col">
                <InputGroup
                  childClass="p-0! m-0! px-4!"
                  title={renderLabelWithTooltip(
                    getEventGroupTitle('Call', events),
                    'Choose which call lifecycle events trigger this webhook.',
                  )}
                >
                  <WebhookEventSelector
                    group="Call"
                    selectedEvents={events}
                    onChange={setEvents}
                  />
                </InputGroup>

                <InputGroup
                  childClass="p-0! m-0! px-4!"
                  title={renderLabelWithTooltip(
                    getEventGroupTitle('Conversation', events),
                    'Choose which conversation lifecycle events trigger this webhook.',
                  )}
                >
                  <WebhookEventSelector
                    group="Conversation"
                    selectedEvents={events}
                    onChange={setEvents}
                  />
                </InputGroup>

                <InputGroup
                  childClass="p-0! m-0! px-4!"
                  title={renderLabelWithTooltip(
                    getEventGroupTitle('WebRTC', events),
                    'Choose which WebRTC media events trigger this webhook.',
                  )}
                >
                  <WebhookEventSelector
                    group="WebRTC"
                    selectedEvents={events}
                    onChange={setEvents}
                  />
                </InputGroup>
              </div>
            ),
          },
          {
            code: 'delivery',
            name: 'Delivery',
            description:
              'Configure the HTTP destination, headers, and delivery behavior.',
            actions: [
              <ButtonSet className="!w-full [&>button]:!flex-1 [&>button]:!max-w-none">
                <SecondaryButton
                  size="lg"
                  onClick={() => showDialog(navigator.goBack)}
                >
                  Cancel
                </SecondaryButton>
                <PrimaryButton size="lg" isLoading={loading} onClick={onSubmit}>
                  Configure webhook
                </PrimaryButton>
              </ButtonSet>,
            ],
            body: (
              <div className="pb-8 flex flex-col">
                <InputGroup
                  title={renderLabelWithTooltip(
                    'Destination',
                    'Configure the HTTP destination that receives the webhook request.',
                  )}
                >
                  <Stack gap={6}>
                    <div className="flex gap-2">
                      <div className="w-36 shrink-0">
                        <CarbonSelect
                          id="webhook-method"
                          labelText="Method"
                          value={method}
                          onChange={e => setMethod(e.target.value)}
                        >
                          <SelectItem value="POST" text="POST" />
                          <SelectItem value="PUT" text="PUT" />
                          <SelectItem value="PATCH" text="PATCH" />
                        </CarbonSelect>
                      </div>
                      <div className="flex-1">
                        <TextInput
                          id="webhook-endpoint"
                          labelText="Server URL"
                          value={endpoint}
                          onChange={e => setEndpoint(e.target.value)}
                          placeholder="https://your-domain.com/webhook"
                        />
                      </div>
                    </div>
                    <TextArea
                      id="webhook-description"
                      labelText="Description (Optional)"
                      value={description}
                      onChange={e => setDescription(e.target.value)}
                      placeholder="An optional description of this webhook destination..."
                      rows={2}
                    />
                  </Stack>
                </InputGroup>

                <InputGroup
                  title={renderLabelWithTooltip(
                    `Headers (${headers.length})`,
                    'HTTP headers included with every webhook request.',
                  )}
                >
                  <APiHeader headers={headers} setHeaders={setHeaders} />
                </InputGroup>

                <div className="grid gap-6 lg:grid-cols-2">
                  <InputGroup
                    title={renderLabelWithTooltip(
                      'Retry',
                      'Control how the webhook retries after failed responses.',
                    )}
                  >
                    <Stack gap={5}>
                      <div className="max-w-xs">
                        <CarbonSelect
                          id="webhook-max-retries"
                          labelText={renderLabelWithTooltip(
                            'Max retry count',
                            'Maximum number of retry attempts after a matching failure response.',
                          )}
                          value={maxRetries.toString()}
                          onChange={e =>
                            setMaxRetries(parseInt(e.target.value))
                          }
                        >
                          <SelectItem value="1" text="1" />
                          <SelectItem value="2" text="2" />
                          <SelectItem value="3" text="3" />
                        </CarbonSelect>
                      </div>
                      <div className="flex flex-wrap gap-4">
                        {['40X', '50X'].map(status => (
                          <Checkbox
                            key={status}
                            id={`retry-status-${status}`}
                            labelText={status}
                            checked={retryOnStatus.includes(status)}
                            onChange={(_, { checked }) => {
                              if (checked) {
                                setRetryOnStatus([...retryOnStatus, status]);
                              } else {
                                setRetryOnStatus(
                                  retryOnStatus.filter(s => s !== status),
                                );
                              }
                            }}
                          />
                        ))}
                      </div>
                    </Stack>
                  </InputGroup>

                  <div className="grid">
                    <InputGroup
                      childClass="space-y-4"
                      title={renderLabelWithTooltip(
                        'Timeout',
                        'Set how long the webhook waits before the request times out.',
                      )}
                    >
                      <div className="flex flex-col gap-4 sm:flex-row sm:items-center">
                        <Slider
                          min={180}
                          max={300}
                          step={1}
                          value={requestTimeout}
                          onSlide={value => setRequestTimeout(value)}
                          className="w-full sm:flex-1"
                        />
                        <NumberInput
                          id="webhook-timeout"
                          hideLabel
                          label={renderLabelWithTooltip(
                            'Timeout (seconds)',
                            'Webhook request timeout in seconds.',
                          )}
                          min={180}
                          max={300}
                          step={1}
                          value={requestTimeout}
                          onChange={(e: any, { value }: any) =>
                            setRequestTimeout(Number(value))
                          }
                          className="!w-full sm:!w-24"
                        />
                      </div>
                      <div className="max-w-[12rem]">
                        <NumberInput
                          id="webhook-priority"
                          label={renderLabelWithTooltip(
                            'Priority',
                            'Execution order when multiple webhooks trigger at the same time.',
                          )}
                          min={0}
                          value={priority}
                          onChange={(e: any, { value }: any) =>
                            setPriority(Number(value))
                          }
                          helperText="Lower numbers execute first when multiple webhooks are triggered."
                        />
                      </div>
                    </InputGroup>
                  </div>
                </div>
              </div>
            ),
          },
        ]}
      />
    </>
  );
};
