import { Metadata } from '@rapidaai/react';
import { SetMetadata } from '@/utils/metadata';

export const OPENAI_TEXT_MODEL = [
  {
    id: 'openai/gpt-4',
    created_date: '2023-11-18 22:21:47.702183',
    updated_date: null,
    provider_id: '1987967168452493312',
    name: 'gpt-4',
    description:
      'GPT-4 from OpenAI has broad general knowledge and domain expertise allowing it to follow complex instructions in natural language and solve difficult problems accurately.',
    human_name: 'OpenAI',
    category: 'text',
    status: 'ACTIVE',
    owner: 'rapida',
  },
  {
    id: 'openai/gpt-4-0613',
    created_date: '2023-11-18 22:21:47.713746',
    updated_date: null,
    provider_id: '1987967168452493312',
    name: 'gpt-4-0613',
    description:
      'Snapshot of gpt-4 from June 13th 2023 with function calling data. Unlike gpt-4, this model does not receive updates, and is deprecated 3 months after a new version is released.',
    human_name: 'OpenAI',
    category: 'text',
    status: 'ACTIVE',
    owner: 'rapida',
  },
  {
    id: 'openai/gpt-3.5-turbo',
    created_date: '2023-11-18 22:21:47.718933',
    updated_date: null,
    provider_id: '1987967168452493312',
    name: 'gpt-3.5-turbo',
    description:
      "OpenAI's most capable and cost effective model in the GPT-3.5 family optimized for chat purposes, but also works well for traditional completions tasks.",
    human_name: 'OpenAI',
    category: 'text',
    status: 'ACTIVE',
    owner: 'rapida',
  },
  {
    id: 'openai/gpt-3.5-turbo-16k',
    created_date: '2023-11-18 22:21:47.724166',
    updated_date: null,
    provider_id: '1987967168452493312',
    name: 'gpt-3.5-turbo-16k',
    description:
      'Same capabilities as the standard gpt-3.5-turbo model but with 4 times the context.',
    human_name: 'OpenAI',
    category: 'text',
    status: 'ACTIVE',
    owner: 'rapida',
  },
  {
    id: 'openai/gpt-3.5-turbo-16k-0613',
    created_date: '2023-11-18 22:21:47.72892',
    updated_date: null,
    provider_id: '1987967168452493312',
    name: 'gpt-3.5-turbo-16k-0613',
    description:
      'Snapshot of gpt-3.5-turbo-16k from June 13th 2023. Unlike gpt-3.5-turbo-16k, this model does not receive updates, and will be deprecated 3 months after a new version is released.',
    human_name: 'OpenAI',
    category: 'text',
    status: 'ACTIVE',
    owner: 'rapida',
  },
  {
    id: 'openai/gpt-4o',
    created_date: '2024-07-08 05:42:22.075867',
    updated_date: '2024-07-08 05:42:22.075867',
    provider_id: '1987967168452493312',
    name: 'gpt-4o',
    description:
      'GPT-4o, an optimized version of the GPT-4 model designed for efficient and high-performance text processing tasks.',
    human_name: 'GPT-4o',
    category: 'text',
    status: 'ACTIVE',
    owner: 'rapida',
  },
  {
    id: 'openai/gpt-4-turbo-preview',
    created_date: '2024-07-08 05:42:22.075867',
    updated_date: '2024-07-08 05:42:22.075867',
    provider_id: '1987967168452493312',
    name: 'gpt-4-turbo-preview',
    description:
      'GPT-4 Turbo Preview, a preview version of the GPT-4 Turbo model offering advanced features and higher efficiency for text processing.',
    human_name: 'GPT-4 Turbo Preview',
    category: 'text',
    status: 'ACTIVE',
    owner: 'rapida',
  },
  {
    id: 'openai/gpt-4-turbo',
    created_date: '2024-07-08 05:42:22.075867',
    updated_date: '2024-07-08 05:42:22.075867',
    provider_id: '1987967168452493312',
    name: 'gpt-4-turbo',
    description:
      'GPT-4 Turbo, a high-performance variant of the GPT-4 model optimized for speed and accuracy in text processing tasks.',
    human_name: 'GPT-4 Turbo',
    category: 'text',
    status: 'ACTIVE',
    owner: 'rapida',
  },

  // new serise model
  // added on 15 july

  {
    id: 'openai/gpt-4-turbo',
    created_date: '2024-07-08 05:42:22.075867',
    updated_date: '2024-07-08 05:42:22.075867',
    provider_id: '1987967168452493312',
    name: 'gpt-4-turbo',
    description:
      'GPT-4 Turbo, a high-performance variant of the GPT-4 model optimized for speed and accuracy in text processing tasks.',
    human_name: 'GPT-4 Turbo',
    category: 'text',
    status: 'ACTIVE',
    owner: 'rapida',
  },

  {
    id: 'openai/gpt-4.1-mini',
    created_date: '2024-07-09 10:01:00.000000',
    updated_date: null,
    provider_id: '1987967168452493312',
    name: 'gpt-4.1-mini',
    description: 'Balanced for intelligence, speed, and cost',
    human_name: 'GPT-4.1 Mini',
    category: 'text',
    status: 'ACTIVE',
    owner: 'rapida',
  },
  {
    id: 'openai/gpt-4.1-nano',
    created_date: '2024-07-09 10:02:00.000000',
    updated_date: null,
    provider_id: '1987967168452493312',
    name: 'gpt-4.1-nano',
    description: 'Fastest, most cost-effective GPT-4.1 model',
    human_name: 'GPT-4.1 Nano',
    category: 'text',
    status: 'ACTIVE',
    owner: 'rapida',
  },
  {
    id: 'openai/o3-mini',
    created_date: '2024-07-09 10:03:00.000000',
    updated_date: null,
    provider_id: '1987967168452493312',
    name: 'o3-mini',
    description: 'A small model alternative to o3',
    human_name: 'O3 Mini',
    category: 'text',
    status: 'ACTIVE',
    owner: 'rapida',
  },
  {
    id: 'openai/gpt-4o-mini',
    created_date: '2024-07-09 10:04:00.000000',
    updated_date: null,
    provider_id: '1987967168452493312',
    name: 'gpt-4o-mini',
    description: 'Fast, affordable small model for focused tasks',
    human_name: 'GPT-4o Mini',
    category: 'text',
    status: 'ACTIVE',
    owner: 'rapida',
  },

  //  o serise

  {
    id: 'openai/o4-mini',
    created_date: '2024-07-09 10:00:00.000000',
    updated_date: null,
    provider_id: '1987967168452493312',
    name: 'o4-mini',
    description: 'Faster, more affordable reasoning model',
    human_name: 'O4 Mini',
    category: 'text',
    status: 'ACTIVE',
    owner: 'rapida',
  },
  {
    id: 'openai/o3',
    created_date: '2024-07-09 10:01:00.000000',
    updated_date: null,
    provider_id: '1987967168452493312',
    name: 'o3',
    description: 'Our most powerful reasoning model',
    human_name: 'O3',
    category: 'text',
    status: 'ACTIVE',
    owner: 'rapida',
  },
  {
    id: 'openai/o3-pro',
    created_date: '2024-07-09 10:02:00.000000',
    updated_date: null,
    provider_id: '1987967168452493312',
    name: 'o3-pro',
    description: 'Version of o3 with more compute for better responses',
    human_name: 'O3 Pro',
    category: 'text',
    status: 'ACTIVE',
    owner: 'rapida',
  },
  {
    id: 'openai/o3-mini',
    created_date: '2024-07-09 10:03:00.000000',
    updated_date: null,
    provider_id: '1987967168452493312',
    name: 'o3-mini',
    description: 'A small model alternative to o3',
    human_name: 'O3 Mini',
    category: 'text',
    status: 'ACTIVE',
    owner: 'rapida',
  },
  {
    id: 'openai/o1',
    created_date: '2024-07-09 10:04:00.000000',
    updated_date: null,
    provider_id: '1987967168452493312',
    name: 'o1',
    description: 'Previous full o-series reasoning model',
    human_name: 'O1',
    category: 'text',
    status: 'ACTIVE',
    owner: 'rapida',
  },
  {
    id: 'openai/o1-pro',
    created_date: '2024-07-09 10:05:00.000000',
    updated_date: null,
    provider_id: '1987967168452493312',
    name: 'o1-pro',
    description: 'Version of o1 with more compute for better responses',
    human_name: 'O1 Pro',
    category: 'text',
    status: 'ACTIVE',
    owner: 'rapida',
  },
];

export const GetOpenaiTextProviderDefaultOptions = (
  current: Metadata[],
): Metadata[] => {
  const mtds: Metadata[] = [];
  const keysToKeep = [
    'rapida.credential_id',
    'model.id',
    'model.name',
    'model.frequency_penalty',
    'model.temperature',
    'model.top_p',
    'model.presence_penalty',
    'model.max_completion_tokens',
    'model.response_format',
    'model.reasoning_effort',
    'model.seed',
    'model.service_tier',
    'model.top_logprobs',
    'model.metadata',
  ];

  const addMetadata = (
    key: string,
    defaultValue?: string,
    validationFn?: (value: string) => boolean,
  ) => {
    const metadata = SetMetadata(current, key, defaultValue, validationFn);
    if (metadata) mtds.push(metadata);
  };
  addMetadata('model.id', OPENAI_TEXT_MODEL[0].id, value =>
    OPENAI_TEXT_MODEL.some(model => model.id === value),
  );

  // Add validation for model.name
  addMetadata('model.name', OPENAI_TEXT_MODEL[0].name, value =>
    OPENAI_TEXT_MODEL.some(model => model.name === value),
  );
  addMetadata('model.frequency_penalty', '0');
  addMetadata('model.temperature', '0.7');
  addMetadata('model.top_p', '1');
  addMetadata('model.presence_penalty', '0');
  addMetadata('model.max_completion_tokens', '2048');
  addMetadata('model.response_format');
  addMetadata('model.stop');
  addMetadata('model.tool_choice');
  addMetadata('model.user');
  addMetadata('model.metadata');
  addMetadata('model.seed');
  addMetadata('model.reasoning_effort');
  addMetadata('model.service_tier');
  addMetadata('model.top_logprobs');
  addMetadata('rapida.credential_id');
  return mtds.filter(m => keysToKeep.includes(m.getKey()));
};

export const ValidateOpenaiTextProviderDefaultOptions = (
  options: Metadata[],
): string | undefined => {
  const credentialID = options.find(
    opt => opt.getKey() === 'rapida.credential_id',
  );
  if (
    !credentialID ||
    !credentialID.getValue() ||
    credentialID.getValue().length === 0
  ) {
    return 'Please check and provide a valid credentials for openai';
  }
  const modelIdOption = options.find(opt => opt.getKey() === 'model.id');
  if (
    !modelIdOption ||
    !OPENAI_TEXT_MODEL.some(model => model.id === modelIdOption.getValue())
  ) {
    return 'Please check and select valid model from dropdown.';
  }

  const frequencyPenaltyOption = options.find(
    opt => opt.getKey() === 'model.frequency_penalty',
  );
  if (
    !frequencyPenaltyOption ||
    isNaN(parseFloat(frequencyPenaltyOption.getValue())) ||
    parseFloat(frequencyPenaltyOption.getValue()) < -2 ||
    parseFloat(frequencyPenaltyOption.getValue()) > 2
  ) {
    return 'Please check and provide a correct value for frequency_penalty a valid value between -2 to 2.';
  }

  const temperatureOption = options.find(
    opt => opt.getKey() === 'model.temperature',
  );
  if (
    !temperatureOption ||
    isNaN(parseFloat(temperatureOption.getValue())) ||
    parseFloat(temperatureOption.getValue()) < 0 ||
    parseFloat(temperatureOption.getValue()) > 1
  ) {
    return 'Please check and provide a correct value for temperature any decimal value between 0 to 1';
  }

  const topPOption = options.find(opt => opt.getKey() === 'model.top_p');
  if (
    !topPOption ||
    isNaN(parseFloat(topPOption.getValue())) ||
    parseFloat(topPOption.getValue()) < 0 ||
    parseFloat(topPOption.getValue()) > 1
  ) {
    return 'Please check and provide a correct value for top_p any decimal value between 0 to 1';
  }

  const presencePenaltyOption = options.find(
    opt => opt.getKey() === 'model.presence_penalty',
  );
  if (
    !presencePenaltyOption ||
    isNaN(parseFloat(presencePenaltyOption.getValue())) ||
    parseFloat(presencePenaltyOption.getValue()) < -2 ||
    parseFloat(presencePenaltyOption.getValue()) > 2
  ) {
    return 'Please check and provide a correct value for presence_penalty any decimal value between -2 to 2';
  }

  const maxCompletionTokensOption = options.find(
    opt => opt.getKey() === 'model.max_completion_tokens',
  );
  if (
    !maxCompletionTokensOption ||
    isNaN(parseInt(maxCompletionTokensOption.getValue())) ||
    parseInt(maxCompletionTokensOption.getValue()) < 1
  ) {
    return 'Please check and provide a correct value for max_completion_tokens it should be greater then 1.';
  }

  const responseFormatOption = options.find(
    opt => opt.getKey() === 'model.response_format',
  );
  if (responseFormatOption) {
    try {
      const parsedFormat = JSON.parse(responseFormatOption.getValue());
      if (typeof parsedFormat !== 'object' || !parsedFormat.type) {
        return 'Please check and provide a correct value for response_format it should be a valid json object.';
      }
      if (!['text', 'json_object', 'json_schema'].includes(parsedFormat.type)) {
        return 'Please check and provide a correct value for response_format it should have type with text, json_object, json_schema.';
      }
      if (parsedFormat.type === 'json_schema' && !parsedFormat.json_schema) {
        return 'Please check and provide a correct value for response_format it should have valid json_schema.';
      }
    } catch (error) {
      return 'Please check and provide a correct value for response_format.';
    }
  }

  const metadata = options.find(opt => opt.getKey() === 'model.metadata');
  if (metadata) {
    try {
      JSON.parse(metadata.getValue());
    } catch (error) {
      return 'Please check and provide a correct value for metadata.';
    }
  }

  return undefined;
};
