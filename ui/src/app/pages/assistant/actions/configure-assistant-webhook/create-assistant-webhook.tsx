import React, { FC, useState } from 'react';
import { useConfirmDialog } from '@/app/pages/assistant/actions/hooks/use-confirmation';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import {
  IBlueBGArrowButton,
  IBlueBorderButton,
  ICancelButton,
} from '@/app/components/Form/Button';
import { InputGroup } from '@/app/components/input-group';
import { cn } from '@/styles/media';
import { FieldSet } from '@/app/components/Form/Fieldset';
import { FormLabel } from '@/app/components/form-label';
import { Input } from '@/app/components/Form/Input';
import { Select } from '@/app/components/Form/Select';
import { Textarea } from '@/app/components/Form/Textarea';
import { InputHelper } from '@/app/components/input-helper';
import { ArrowRight, Plus, Trash2 } from 'lucide-react';
import { Slider } from '@/app/components/Form/Slider';
import { CreateWebhook } from '@rapidaai/react';
import { useCurrentCredential } from '@/hooks/use-credential';
import { PageActionButtonBlock } from '@/app/components/blocks/page-action-button-block';
import { InputCheckbox } from '@/app/components/Form/Checkbox';
import toast from 'react-hot-toast/headless';
import { useRapidaStore } from '@/hooks';
import { connectionConfig } from '@/configs';
import { PageHeaderBlock } from '@/app/components/blocks/page-header-block';
import { PageTitleBlock } from '@/app/components/blocks/page-title-block';

const webhookEvents = [
  {
    id: 'conversation.begin',
    name: 'conversation.begin',
    description: 'Triggered when a new conversation begins.',
    category: 'Conversation',
  },
  {
    id: 'conversation.completed',
    name: 'conversation.completed',
    description: 'Triggered when a conversation ends successfully.',
    category: 'Conversation',
  },
  {
    id: 'conversation.failed',
    name: 'conversation.failed',
    description: 'Triggered when a conversation fails.',
    category: 'Conversation',
  },
];
export const CreateAssistantWebhook: FC<{ assistantId: string }> = ({
  assistantId,
}) => {
  const navigator = useGlobalNavigation();
  const { authId, token, projectId } = useCurrentCredential();
  const { showDialog, ConfirmDialogComponent } = useConfirmDialog({});
  const { loading, showLoader, hideLoader } = useRapidaStore();
  const [method, setMethod] = useState('POST');
  const [endpoint, setEndpoint] = useState('');
  const [description, setDescription] = useState('');
  const [retryOnStatus, setRetryOnStatus] = useState<string[]>(['500']);
  const [maxRetries, setMaxRetries] = useState(3);
  const [requestTimeout, setRequestTimeout] = useState(180);
  const [headers, setHeaders] = useState<{ key: string; value: string }[]>([]);
  const [priority, setPriority] = useState<number>(0);
  const [parameters, setParameters] = useState<
    {
      type:
        | 'event'
        | 'assistant'
        | 'conversation'
        | 'argument'
        | 'metadata'
        | 'option'
        | 'analysis';
      key: string;
      value: string;
    }[]
  >([
    { type: 'event', key: 'type', value: 'event' },
    { type: 'event', key: 'data', value: 'data' },
  ]);

  const [events, setEvents] = useState<string[]>([]);
  const [errorMessage, setErrorMessage] = useState('');

  const validateForm = () => {
    if (!endpoint) {
      setErrorMessage('Please provide a server url for the webhook.');
      return false;
    }
    if (!/^https?:\/\/.+/.test(endpoint)) {
      setErrorMessage('Please provide a valid server url for the webhook.');
      return false;
    }

    if (parameters.length === 0) {
      setErrorMessage(
        'Please provide one or more parameters which can be passed as data to your server.',
      );
      return false;
    }

    // Check for duplicate keys
    const keys = parameters.map(param => `${param.type}.${param.key}`);
    const uniqueKeys = new Set(keys);
    if (keys.length !== uniqueKeys.size) {
      setErrorMessage(`Duplicate parameter keys  are not allowed.`);
      return false;
    }

    const emptyKeysOrValues = parameters.filter(
      param => param.key.trim() === '' || param.value.trim() === '',
    );
    if (emptyKeysOrValues.length > 0) {
      setErrorMessage(`Empty parameter keys or values are not allowed.`);
      return false;
    }
    const values = parameters.map(param => param.value.trim());
    const uniqueValues = new Set(values);
    if (values.length !== uniqueValues.size) {
      setErrorMessage(`Duplicate parameter values are not allowed.`);
      return false;
    }

    if (Object.values(events).every(v => !v)) {
      setErrorMessage(
        'Please select at least one event when the webhook will get triggered.',
      );
      return false;
    }

    return true;
  };

  const onSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!validateForm()) return;
    showLoader();
    try {
      // Create key-value pairs for parameters
      const parameterKeyValuePairs = parameters.map(param => ({
        key: `${param.type}.${param.key}`,
        value: param.value,
      }));

      CreateWebhook(
        connectionConfig,
        assistantId,
        method,
        endpoint,
        headers,
        parameterKeyValuePairs,
        events,
        retryOnStatus,
        maxRetries,
        requestTimeout,
        priority,
        (err, response) => {
          hideLoader();
          if (err) {
            setErrorMessage(
              'Unable to create assistant webhook, please check and try again.',
            );
            return;
          }
          if (response?.getSuccess()) {
            toast.success(`Assistant's webhook created successfully`);
            navigator.goToAssistantWebhook(assistantId);
          } else {
            if (response?.getError()) {
              let err = response.getError();
              const message = err?.getHumanmessage();
              if (message) {
                setErrorMessage(message);
                return;
              }
              setErrorMessage(
                'Unable to create assistant webhook, please check and try again.',
              );
              return;
            }
            setErrorMessage(
              'Unable to create assistant webhook, please check and try again.',
            );
          }
        },
        {
          'x-auth-id': authId,
          authorization: token,
          'x-project-id': projectId,
        },
        description,
      );
    } catch (error) {
      setErrorMessage('Failed to configure webhook. Please try again.');
      console.error('Error configuring webhook:', error);
    }
  };

  const updateHeader = (index: number, key: string, value: string) => {
    const newHeaders = [...headers];
    newHeaders[index] = { ...newHeaders[index], [key]: value };
    setHeaders(newHeaders);
  };

  const updateParameter = (index: number, field: string, value: string) => {
    setParameters(prevParams =>
      prevParams.map((param, i) => {
        if (i === index) {
          const updatedParam = { ...param, [field]: value };
          if (field === 'type') {
            updatedParam.key = '';
            updatedParam.value = '';
          }

          return updatedParam;
        }
        return param;
      }),
    );
  };

  return (
    <form
      onSubmit={onSubmit}
      method="POST"
      className="relative flex flex-col flex-1"
    >
      <ConfirmDialogComponent />
      <div className="overflow-auto flex flex-col flex-1 pb-20">
        <PageHeaderBlock className="border-b">
          <div className="flex items-center gap-3">
            <PageTitleBlock>Creating new webhook</PageTitleBlock>
          </div>
        </PageHeaderBlock>
        <div
          className={cn(
            'px-6 pb-6 pt-2 flex flex-col gap-8 py-8 w-full max-w-6xl',
          )}
        >
          <div className="flex space-x-2">
            <FieldSet className="relative w-40">
              <FormLabel>Method</FormLabel>
              <Select
                value={method}
                onChange={e => setMethod(e.target.value)}
                className=""
                options={[
                  { name: 'POST', value: 'POST' },
                  { name: 'PUT', value: 'PUT' },
                  { name: 'PATCH', value: 'PATCH' },
                ]}
              />
            </FieldSet>
            <FieldSet className="relative w-full">
              <FormLabel>Server Url</FormLabel>
              <Input
                value={endpoint}
                onChange={e => setEndpoint(e.target.value)}
                placeholder="https://your-domain.com/webhook"
                className=""
              />
            </FieldSet>
          </div>
          <FieldSet className="relative w-full">
            <FormLabel>Description</FormLabel>
            <Textarea
              value={description}
              onChange={e => setDescription(e.target.value)}
              className=""
              placeholder="An optional description of the destination..."
              rows={2}
            />
          </FieldSet>
          <FieldSet>
            <FormLabel>Headers ({headers.length})</FormLabel>
            <div className="text-sm grid w-full ">
              {headers.map((header, index) => (
                <div
                  key={index}
                  className="grid grid-cols-2 border-b border-gray-400 dark:border-gray-600"
                >
                  <div className="flex col-span-1 items-center border-r">
                    <Input
                      value={header.key}
                      onChange={e => updateHeader(index, 'key', e.target.value)}
                      placeholder="Key"
                      className=" w-full border-none"
                    />
                  </div>
                  <div className="col-span-1 flex">
                    <Input
                      value={header.value}
                      onChange={e =>
                        updateHeader(index, 'value', e.target.value)
                      }
                      placeholder="Value"
                      className=" w-full border-none"
                    />
                    <ICancelButton
                      className="border-none outline-hidden dark:bg-gray-950"
                      onClick={() =>
                        setHeaders(headers.filter((_, i) => i !== index))
                      }
                      type="button"
                    >
                      <Trash2 className="w-4 h-4" strokeWidth={1.5} />
                    </ICancelButton>
                  </div>
                </div>
              ))}
            </div>
            <IBlueBorderButton
              onClick={() => setHeaders([...headers, { key: '', value: '' }])}
              className="justify-between space-x-8"
            >
              <span>Add header</span> <Plus className="h-4 w-4 ml-1.5" />
            </IBlueBorderButton>
          </FieldSet>

          <FieldSet>
            <FormLabel>Parameters ({parameters.length})</FormLabel>
            <div className="text-sm grid w-full ">
              {parameters.map((params, index) => (
                <div
                  key={index}
                  className="grid grid-cols-2 border-b border-gray-400 dark:border-gray-600"
                >
                  <div className="flex col-span-1 items-center">
                    <Select
                      value={params.type}
                      onChange={e =>
                        updateParameter(index, 'type', e.target.value)
                      }
                      className=" border-none"
                      options={[
                        { name: 'Event', value: 'event' },
                        { name: 'Assistant', value: 'assistant' },
                        { name: 'Conversation', value: 'conversation' },
                        { name: 'Argument', value: 'argument' },
                        { name: 'Metadata', value: 'metadata' },
                        { name: 'Option', value: 'option' },
                        { name: 'Analysis', value: 'analysis' },
                        { name: 'Custom', value: 'custom' },
                      ]}
                    />
                    <TypeKeySelector
                      type={
                        params.type as
                          | 'event'
                          | 'assistant'
                          | 'conversation'
                          | 'argument'
                          | 'metadata'
                          | 'option'
                          | 'analysis'
                          | 'custom'
                      }
                      key={`type-key-selector-${index}`}
                      value={params.key}
                      onChange={newKey => updateParameter(index, 'key', newKey)}
                    />
                    <div className=" dark:bg-gray-950 bg-white h-full flex items-center justify-center">
                      <ArrowRight strokeWidth={1.5} className="w-4 h-4" />
                    </div>
                  </div>
                  <div className="col-span-1 flex">
                    <Input
                      value={params.value}
                      onChange={e =>
                        updateParameter(index, 'value', e.target.value)
                      }
                      placeholder="Value"
                      className=" w-full border-none"
                    />
                    <ICancelButton
                      className="border-none outline-hidden "
                      onClick={() =>
                        setParameters(parameters.filter((_, i) => i !== index))
                      }
                      type="button"
                    >
                      <Trash2 className="w-4 h-4" strokeWidth={1.5} />
                    </ICancelButton>
                  </div>
                </div>
              ))}
            </div>
            <IBlueBorderButton
              onClick={() =>
                setParameters([
                  ...parameters,
                  { type: 'assistant', key: '', value: '' },
                ])
              }
              className="justify-between space-x-8"
            >
              <span>Add parameters</span> <Plus className="h-4 w-4 ml-1.5" />
            </IBlueBorderButton>
          </FieldSet>
        </div>

        <InputGroup title="Advanced configuration" initiallyExpanded={false}>
          <div className={cn('px-6 pb-6 pt-2 pl-8 w-full max-w-6xl space-y-6')}>
            <FieldSet className="relative w-60 shrink-0">
              <FormLabel className="normal-case">Max retry count</FormLabel>
              <Select
                value={maxRetries.toString()}
                onChange={e => setMaxRetries(parseInt(e.target.value))}
                className=""
                options={[
                  { name: '1', value: '1' },
                  { name: '2', value: '2' },
                  { name: '3', value: '3' },
                ]}
              />
            </FieldSet>
            <FieldSet>
              <FormLabel className="normal-case">
                Retry on status codes
              </FormLabel>
              <div className="flex flex-wrap gap-2 space-x-6">
                {['40X', '50X'].map(status => (
                  <label key={status} className="flex items-center space-x-2">
                    <InputCheckbox
                      checked={retryOnStatus.includes(status)}
                      onChange={e => {
                        if (e.target.checked) {
                          setRetryOnStatus([...retryOnStatus, status]);
                        } else {
                          setRetryOnStatus(
                            retryOnStatus.filter(s => s !== status),
                          );
                        }
                      }}
                    />
                    <span>{status}</span>
                  </label>
                ))}
              </div>
            </FieldSet>
            <FieldSet>
              <FormLabel>Timeout (in seconds)</FormLabel>
              <div className="flex items-center space-x-4">
                <Slider
                  min={180}
                  max={300}
                  step={1}
                  value={requestTimeout}
                  onSlide={value => setRequestTimeout(value)}
                  className="w-64"
                />
                <Input
                  type="number"
                  min={180}
                  max={300}
                  step={1}
                  value={requestTimeout}
                  onChange={e => {
                    setRequestTimeout(Number(e.target.value));
                  }}
                  className="w-16 ml-2 h-9 "
                />
              </div>
            </FieldSet>
            <FieldSet className="relative w-40">
              <FormLabel>Execution Priority</FormLabel>
              <Input
                type="number"
                min={0}
                value={priority}
                onChange={e => setPriority(Number(e.target.value))}
                className=""
              />
            </FieldSet>
          </div>
        </InputGroup>

        <InputGroup title="Assistant Events">
          <div className={cn('px-6 pb-6 pt-2 flex gap-8 pl-8')}>
            <div className="space-y-6 w-full max-w-6xl">
              <FieldSet>
                <div className="grid grid-cols-2 gap-4">
                  {webhookEvents.map(event => (
                    <div key={event.id} className="flex items-start">
                      <div className="flex h-4 items-center mt-2">
                        <InputCheckbox
                          id={event.id}
                          checked={events.includes(event.id)}
                          onChange={e => {
                            if (e.target.checked) {
                              setEvents([...events, event.id]);
                            } else {
                              setEvents(events.filter(id => id !== event.id));
                            }
                          }}
                          className=""
                        />
                      </div>
                      <FieldSet className="ml-3 space-y-0.5!">
                        <FormLabel
                          htmlFor={event.id}
                          className="font-medium text-base dark:text-gray-400"
                        >
                          {event.name}
                        </FormLabel>
                        <InputHelper>{event.description}</InputHelper>
                      </FieldSet>
                    </div>
                  ))}
                </div>
              </FieldSet>
            </div>
          </div>
        </InputGroup>
      </div>

      <PageActionButtonBlock errorMessage={errorMessage}>
        <ICancelButton
          className="px-4 rounded-[2px]"
          onClick={() => showDialog(navigator.goBack)}
          type="button"
        >
          Cancel
        </ICancelButton>
        <IBlueBGArrowButton
          isLoading={loading}
          type="submit"
          className="px-4 rounded-[2px]"
        >
          Configure webhook
        </IBlueBGArrowButton>
      </PageActionButtonBlock>
    </form>
  );
};

export const TypeKeySelector: FC<{
  type:
    | 'assistant'
    | 'event'
    | 'conversation'
    | 'argument'
    | 'metadata'
    | 'option'
    | 'custom'
    | 'analysis';
  value: string;
  onChange: (newValue: string) => void;
}> = ({ type, value, onChange }) => {
  switch (type) {
    case 'event':
      return (
        <Select
          value={value}
          onChange={e => onChange(e.target.value)}
          className=" border-none"
          options={[
            { name: 'Type', value: 'type' },
            { name: 'Data', value: 'data' },
          ]}
        />
      );
    case 'assistant':
      return (
        <Select
          value={value}
          onChange={e => onChange(e.target.value)}
          className=" border-none"
          options={[
            { name: 'ID', value: 'id' },
            { name: 'Name', value: 'name' },
            { name: 'Version', value: 'version' },
          ]}
        />
      );
    case 'conversation':
      return (
        <Select
          value={value}
          onChange={e => onChange(e.target.value)}
          className=" border-none"
          options={[
            { name: 'Messages', value: 'messages' },
            { name: 'ID', value: 'id' },
          ]}
        />
      );
    default:
      return (
        <Input
          value={value}
          onChange={e => onChange(e.target.value)}
          placeholder="Key"
          className=" w-full border-none"
        />
      );
  }
};
