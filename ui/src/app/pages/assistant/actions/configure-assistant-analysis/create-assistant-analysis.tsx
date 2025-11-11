import React, { FC, useState } from 'react';
import { useConfirmDialog } from '@/app/pages/assistant/actions/hooks/use-confirmation';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import {
  IBlueBGArrowButton,
  IBlueBorderButton,
  ICancelButton,
} from '@/app/components/Form/Button';
import { cn } from '@/styles/media';
import { FieldSet } from '@/app/components/Form/Fieldset';
import { FormLabel } from '@/app/components/form-label';
import { Input } from '@/app/components/Form/Input';
import { Select } from '@/app/components/Form/Select';
import { Textarea } from '@/app/components/Form/Textarea';
import { ArrowRight, Plus, Trash2 } from 'lucide-react';
import { useCurrentCredential } from '@/hooks/use-credential';
import { PageActionButtonBlock } from '@/app/components/blocks/page-action-button-block';
import { randomMeaningfullName } from '@/utils';
import { EndpointDropdown } from '@/app/components/Dropdown/endpoint-dropdown';
import { CreateAnalysis, Endpoint } from '@rapidaai/react';
import toast from 'react-hot-toast/headless';
import { connectionConfig } from '@/configs';
import { PageHeaderBlock } from '@/app/components/blocks/page-header-block';
import { PageTitleBlock } from '@/app/components/blocks/page-title-block';

export const CreateAssistantAnalysis: FC<{ assistantId: string }> = ({
  assistantId,
}) => {
  const navigator = useGlobalNavigation();
  const { authId, token, projectId } = useCurrentCredential();
  const { showDialog, ConfirmDialogComponent } = useConfirmDialog({});

  const [name, setName] = useState(randomMeaningfullName('analysis'));
  const [description, setDescription] = useState('');
  const [priority, setPriority] = useState<number>(0);
  const [endpointId, setEndpointId] = useState<string>('');
  const [parameters, setParameters] = useState<
    {
      type:
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
    {
      type: 'conversation',
      key: 'messages',
      value: 'messages',
    },
  ]);

  const [errorMessage, setErrorMessage] = useState('');

  // Validation function
  const validateForm = () => {
    if (!name) {
      setErrorMessage('Please provide a valid name for analysis.');
      return false;
    }

    if (!endpointId) {
      setErrorMessage(
        'Please select a valid endpoint to be executed for analysis.',
      );
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

    return true;
  };

  const onSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!validateForm()) return;

    try {
      const parameterKeyValuePairs = parameters.map(param => ({
        key: `${param.type}.${param.key}`,
        value: param.value,
      }));

      CreateAnalysis(
        connectionConfig,
        assistantId,
        name,
        endpointId,
        'latest',
        priority,
        parameterKeyValuePairs,
        (err, response) => {
          if (err) {
            setErrorMessage(
              'Unable to create assistant analysis, please check and try again.',
            );
          }
          if (response?.getSuccess()) {
            toast.success(`Analysis added to assistant successfully`);
            navigator.goToConfigureAssistantAnalysis(assistantId);
          } else {
            if (response?.getError()) {
              let err = response.getError();
              const message = err?.getHumanmessage();
              if (message) {
                setErrorMessage(message);
                return;
              }
              setErrorMessage(
                'Unable to create assistant analysis, please check and try again.',
              );
              return;
            }
            setErrorMessage(
              'Unable to create assistant analysis, please check and try again.',
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
      setErrorMessage('Failed to configure analysis. Please try again.');
      console.error('Error configuring analysis:', error);
    }
  };

  const updateParameter = (index: number, field: string, value: string) => {
    setParameters(prevParams =>
      prevParams.map((param, i) =>
        i === index ? { ...param, [field]: value } : param,
      ),
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
            <PageTitleBlock>Creating new analysis</PageTitleBlock>
          </div>
        </PageHeaderBlock>
        <div
          className={cn(
            'px-6 pb-6 pt-2 flex flex-col gap-8 py-8  w-full max-w-6xl',
          )}
        >
          <FieldSet className="relative w-full">
            <FormLabel>Name</FormLabel>
            <Input
              value={name}
              onChange={e => setName(e.target.value)}
              placeholder="A name for your analysis"
            />
          </FieldSet>
          <FieldSet className="relative w-full">
            <FormLabel>Description</FormLabel>
            <Textarea
              value={description}
              onChange={e => setDescription(e.target.value)}
              placeholder="An optional description of the destination..."
              rows={2}
            />
          </FieldSet>
          <EndpointDropdown
            className="bg-white"
            currentEndpoint={endpointId}
            onChangeEndpoint={(e: Endpoint) => {
              if (e) setEndpointId(e.getId());
            }}
          />
          <FieldSet>
            <FormLabel>Parameters ({parameters.length})</FormLabel>
            <div className="text-sm grid w-full">
              {parameters.map((param, index) => (
                <div
                  key={index}
                  className="grid grid-cols-2 border-b border-gray-400 dark:border-gray-600"
                >
                  <div className="flex col-span-1 items-center">
                    <Select
                      value={param.type}
                      onChange={e =>
                        updateParameter(index, 'type', e.target.value)
                      }
                      className="border-none"
                      options={[
                        { name: 'Assistant', value: 'assistant' },
                        { name: 'Conversation', value: 'conversation' },
                        { name: 'Argument', value: 'argument' },
                        { name: 'Metadata', value: 'metadata' },
                        { name: 'Option', value: 'option' },
                        { name: 'Analysis', value: 'analysis' },
                      ]}
                    />
                    <TypeKeySelector
                      type={param.type}
                      value={param.key}
                      onChange={newKey => updateParameter(index, 'key', newKey)}
                    />
                    <div className="bg-white dark:bg-gray-950 h-full flex items-center justify-center">
                      <ArrowRight strokeWidth={1.5} />
                    </div>
                  </div>

                  <div className="col-span-1 flex">
                    <Input
                      value={param.value}
                      onChange={e =>
                        updateParameter(index, 'value', e.target.value)
                      }
                      placeholder="Value"
                      className="w-full border-none"
                    />
                    <ICancelButton
                      className="border-none outline-hidden dark:bg-gray-950"
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
          <FieldSet className="relative w-40">
            <FormLabel>Execution Priority</FormLabel>
            <Input
              type="number"
              min={0}
              value={priority}
              onChange={e => setPriority(Number(e.target.value))}
            />
          </FieldSet>
        </div>
      </div>

      {/* onClick={() => showDialog(navigator.goBack)} */}
      {/* Error message and submit buttons */}
      <PageActionButtonBlock errorMessage={errorMessage}>
        <ICancelButton
          className="px-4 rounded-[2px]"
          onClick={() => showDialog(navigator.goBack)}
          type="button"
        >
          Cancel
        </ICancelButton>
        <IBlueBGArrowButton type="submit" className="px-4 rounded-[2px]">
          Configure analysis
        </IBlueBGArrowButton>
      </PageActionButtonBlock>
    </form>
  );
};

const TypeKeySelector: FC<{
  type:
    | 'assistant'
    | 'conversation'
    | 'argument'
    | 'metadata'
    | 'option'
    | 'analysis';
  value: string;
  onChange: (newValue: string) => void;
}> = ({ type, value, onChange }) => {
  switch (type) {
    case 'assistant':
      return (
        <Select
          value={value}
          onChange={e => onChange(e.target.value)}
          className="border-none"
          options={[
            { name: 'Name', value: 'name' },
            { name: 'Prompt', value: 'prompt' },
          ]}
        />
      );
    case 'conversation':
      return (
        <Select
          value={value}
          onChange={e => onChange(e.target.value)}
          className="border-none"
          options={[{ name: 'Messages', value: 'messages' }]}
        />
      );
    default:
      return (
        <Input
          value={value}
          onChange={e => onChange(e.target.value)}
          placeholder="Key"
          className="w-full border-none"
        />
      );
  }
};
