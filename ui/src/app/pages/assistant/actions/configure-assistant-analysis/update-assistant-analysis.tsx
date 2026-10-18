import React, { FC, useEffect, useState } from 'react';
import { useConfirmDialog } from '@/app/pages/assistant/actions/hooks/use-confirmation';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { PrimaryButton, SecondaryButton } from '@/app/components/ui/primitives';
import { Stack, TextInput, TextArea } from '@/app/components/ui/primitives';
import { ButtonSet, NumberInput } from '@carbon/react';
import { useCurrentCredential } from '@/hooks/use-credential';
import { randomMeaningfullName } from '@/utils';
import { EndpointDropdown } from '@/app/components/domain/dropdowns/endpoint-dropdown';
import { Endpoint, Metadata } from '@rapidaai/react';
import { useParams } from 'react-router-dom';
import toast from 'react-hot-toast/headless';
import { TabForm } from '@/app/components/ui/composites';
import {
  ASSISTANT_CONDITION_KEY_OPTIONS,
  ASSISTANT_CONDITION_OPERATOR_OPTIONS,
  ASSISTANT_CONDITION_SOURCE_OPTIONS,
  ASSISTANT_CONDITION_VALUE_OPTIONS_BY_KEY,
  AssistantConditionEntry,
  AssistantMappingTable,
  normalizeAssistantConditionEntries,
} from '@/app/components/domain/tools/common';
import { SourceConditionRule } from '@/app/components/domain/conditions/source-condition-rule';
import { InputGroup } from '@/app/components/ui/primitives';
import {
  getAssistantConfigurationById,
  updateAssistantConfigurationById,
} from '@/clients/assistant.client';

type ParamType =
  | 'client'
  | 'assistant'
  | 'conversation'
  | 'argument'
  | 'metadata'
  | 'option'
  | 'custom'
  | 'analysis';

interface Parameter {
  type: ParamType;
  key: string;
  value: string;
}

const PARAM_TYPE_OPTIONS = [
  { value: 'client', name: 'Client' },
  { value: 'assistant', name: 'Assistant' },
  { value: 'conversation', name: 'Conversation' },
  { value: 'argument', name: 'Argument' },
  { value: 'metadata', name: 'Metadata' },
  { value: 'option', name: 'Option' },
  { value: 'custom', name: 'Custom' },
  { value: 'analysis', name: 'Analysis' },
];
const ANALYSIS_CONDITION_OPTION_KEY = 'analysis.condition';
const analysisConfigurationType = 'analysis';

const RESERVED_OPTION_KEYS = new Set([
  'option.endpoint_id',
  'option.endpoint_version',
  'option.endpoint_parameters',
  `option.${ANALYSIS_CONDITION_OPTION_KEY}`,
]);

const DEFAULT_CONDITIONS: AssistantConditionEntry[] = [
  { key: 'source', condition: '=', value: 'all' },
];

const parseConditions = (
  raw?: string,
): AssistantConditionEntry[] | undefined => {
  if (!raw) return undefined;
  try {
    const parsed = JSON.parse(raw);
    const normalized = normalizeAssistantConditionEntries(parsed);
    return normalized.length > 0 ? normalized : DEFAULT_CONDITIONS;
  } catch {
    return DEFAULT_CONDITIONS;
  }
};

const parseEndpointParameters = (raw?: string): Parameter[] => {
  if (!raw) return [];
  try {
    const parsed = JSON.parse(raw);
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
      return [];
    }
    return Object.entries(parsed).flatMap(([key, value]) => {
      if (typeof value !== 'string') return [];
      const [type, ...parts] = key.split('.');
      if (!type || parts.length === 0) return [];
      return [
        {
          type: type as ParamType,
          key: parts.join('.'),
          value,
        },
      ];
    });
  } catch {
    return [];
  }
};

export const UpdateAssistantAnalysis: FC<{ assistantId: string }> = ({
  assistantId,
}) => {
  const navigator = useGlobalNavigation();
  const { analysisId } = useParams();
  const { authId, token, projectId } = useCurrentCredential();
  const { showDialog, ConfirmDialogComponent } = useConfirmDialog({});

  const [activeTab, setActiveTab] = useState('configure');
  const [errorMessage, setErrorMessage] = useState('');

  const [name, setName] = useState(randomMeaningfullName());
  const [description, setDescription] = useState('');
  const [priority, setPriority] = useState<number>(0);
  const [endpointId, setEndpointId] = useState<string>('');
  const [parameters, setParameters] = useState<Parameter[]>([
    { type: 'conversation', key: 'messages', value: 'messages' },
  ]);
  const [sourceConditions, setSourceConditions] =
    useState<AssistantConditionEntry[]>(DEFAULT_CONDITIONS);

  useEffect(() => {
    const load = async () => {
      try {
        const res = await getAssistantConfigurationById({
          assistantId,
          configurationId: analysisId!,
          auth: { projectId, token, userId: authId },
        });
        const analysis = res?.getData();
        if (!analysis) return;

        const options = new Map<string, string>();
        const optionsList = (analysis as any).getOptionsList?.();
        if (Array.isArray(optionsList)) {
          optionsList.forEach((item: any) => {
            const key = item?.getKey?.();
            const value = item?.getValue?.();
            if (key && typeof value === 'string') {
              options.set(key, value);
            }
          });
        }

        setName(options.get('name') || '');
        setDescription(options.get('description') || '');
        const nextPriority = Number(options.get('execution_priority') || '0');
        setPriority(Number.isFinite(nextPriority) ? nextPriority : 0);
        setEndpointId(options.get('endpoint_id') || '');

        const parsedConditions =
          parseConditions(options.get(ANALYSIS_CONDITION_OPTION_KEY)) ||
          DEFAULT_CONDITIONS;
        setSourceConditions(parsedConditions);

        const nextParameters = parseEndpointParameters(
          options.get('endpoint_parameters'),
        );
        setParameters(
          nextParameters.length > 0
            ? nextParameters
            : [{ type: 'conversation', key: 'messages', value: 'messages' }],
        );
      } catch {
        toast.error('Unable to load analysis, please try again later.');
      }
    };

    load();
  }, [assistantId, analysisId, authId, token, projectId]);

  const validateConfigure = (): boolean => {
    setErrorMessage('');
    if (!endpointId) {
      setErrorMessage(
        'Please select a valid endpoint to be executed for analysis.',
      );
      return false;
    }
    if (parameters.length === 0) {
      setErrorMessage('Please provide one or more parameters.');
      return false;
    }
    const keys = parameters.map(p => `${p.type}.${p.key}`);
    const reservedKey = keys.find(key => RESERVED_OPTION_KEYS.has(key));
    if (reservedKey) {
      setErrorMessage(
        `${reservedKey} is reserved and managed by analysis options.`,
      );
      return false;
    }
    if (new Set(keys).size !== keys.length) {
      setErrorMessage('Duplicate parameter keys are not allowed.');
      return false;
    }
    if (parameters.some(p => !p.key.trim() || !p.value.trim())) {
      setErrorMessage('Empty parameter keys or values are not allowed.');
      return false;
    }
    const values = parameters.map(p => p.value.trim());
    if (new Set(values).size !== values.length) {
      setErrorMessage('Duplicate parameter values are not allowed.');
      return false;
    }
    return true;
  };

  const onSubmit = async () => {
    setErrorMessage('');
    if (!name) {
      setErrorMessage('Please provide a valid name for analysis.');
      return;
    }

    const keys = parameters.map(p => `${p.type}.${p.key}`);
    const reservedKey = keys.find(key => RESERVED_OPTION_KEYS.has(key));
    if (reservedKey) {
      setErrorMessage(
        `${reservedKey} is reserved and managed by analysis options.`,
      );
      return;
    }

    const endpointParameters = Object.fromEntries(
      parameters.map(p => [`${p.type}.${p.key}`, p.value]),
    );

    const options: Metadata[] = [];
    [
      { key: 'name', value: name },
      { key: 'description', value: description },
      { key: 'execution_priority', value: String(priority || 0) },
      { key: 'endpoint_id', value: endpointId },
      { key: 'endpoint_version', value: 'latest' },
      {
        key: 'endpoint_parameters',
        value: JSON.stringify(endpointParameters),
      },
      {
        key: ANALYSIS_CONDITION_OPTION_KEY,
        value: JSON.stringify(sourceConditions),
      },
    ].forEach(({ key, value }) => {
      const item = new Metadata();
      item.setKey(key);
      item.setValue(value);
      options.push(item);
    });
    try {
      const response = await updateAssistantConfigurationById({
        assistantId,
        configurationId: analysisId!,
        configurationType: analysisConfigurationType,
        provider: 'endpoint',
        enabled: true,
        options,
        auth: { projectId, token, userId: authId },
      });

      if (response?.getSuccess()) {
        toast.success(`Assistant's analysis updated successfully`);
        navigator.goToConfigureAssistantAnalysis(assistantId);
      } else {
        setErrorMessage(
          response?.getError()?.getHumanmessage() ||
            'Unable to update assistant analysis, please check and try again.',
        );
      }
    } catch {
      setErrorMessage(
        'Unable to update assistant analysis, please check and try again.',
      );
    }
  };

  return (
    <>
      <ConfirmDialogComponent />
      <TabForm
        formHeading="Update all steps to reconfigure your analysis."
        activeTab={activeTab}
        onChangeActiveTab={() => {}}
        errorMessage={errorMessage}
        form={[
          {
            code: 'configure',
            name: 'Configure',
            description:
              'Select the endpoint and map the data parameters for analysis.',
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
                    if (validateConfigure()) setActiveTab('profile');
                  }}
                >
                  Continue
                </PrimaryButton>
              </ButtonSet>,
            ],
            body: (
              <div>
                <InputGroup title="Execution conditions (Optional)">
                  <SourceConditionRule
                    conditions={sourceConditions}
                    onChangeConditions={setSourceConditions}
                    conditionOptions={ASSISTANT_CONDITION_OPERATOR_OPTIONS}
                    sourceOptions={ASSISTANT_CONDITION_SOURCE_OPTIONS}
                    keyOptions={ASSISTANT_CONDITION_KEY_OPTIONS}
                    valueOptionsByKey={ASSISTANT_CONDITION_VALUE_OPTIONS_BY_KEY}
                    keyTooltipText="The variable to evaluate before running this analysis."
                  />
                </InputGroup>
                <Stack gap={7} className="px-8 pt-6 pb-8 max-w-4xl">
                  <EndpointDropdown
                    currentEndpoint={endpointId}
                    onChangeEndpoint={(e: Endpoint) => {
                      if (e) setEndpointId(e.getId());
                    }}
                  />
                  <AssistantMappingTable
                    parameters={parameters}
                    onChange={setParameters}
                    typeOptions={PARAM_TYPE_OPTIONS}
                    includeEmptyKeyOption
                    createNewParameter={() => ({
                      type: 'assistant',
                      key: '',
                      value: '',
                    })}
                    title="Parameters"
                    addButtonLabel="Add parameter"
                    valuePlaceholder="Variable name"
                    isRemoveDestructive
                  />
                </Stack>
              </div>
            ),
          },
          {
            code: 'profile',
            name: 'Profile',
            description: 'Provide a name and set the execution priority.',
            actions: [
              <ButtonSet className="!w-full [&>button]:!flex-1 [&>button]:!max-w-none">
                <SecondaryButton
                  size="lg"
                  onClick={() => showDialog(navigator.goBack)}
                >
                  Cancel
                </SecondaryButton>
                <PrimaryButton size="lg" onClick={onSubmit}>
                  Update analysis
                </PrimaryButton>
              </ButtonSet>,
            ],
            body: (
              <div className="px-8 pt-6 pb-8 max-w-2xl">
                <Stack gap={6}>
                  <TextInput
                    id="analysis-name"
                    labelText="Name"
                    value={name}
                    onChange={e => setName(e.target.value)}
                    placeholder="A name for your analysis"
                    helperText="A unique name to identify this analysis configuration."
                  />
                  <TextArea
                    id="analysis-description"
                    labelText="Description (Optional)"
                    value={description}
                    onChange={e => setDescription(e.target.value)}
                    placeholder="An optional description of this analysis..."
                    rows={2}
                  />
                  <NumberInput
                    id="analysis-priority"
                    label="Execution Priority"
                    min={0}
                    value={priority}
                    onChange={(e: any, { value }: any) => setPriority(value)}
                    helperText="Lower numbers execute first when multiple analyses are triggered."
                  />
                </Stack>
              </div>
            ),
          },
        ]}
      />
    </>
  );
};
