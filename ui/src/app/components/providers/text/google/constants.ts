import { Metadata } from '@rapidaai/react';
import { SetMetadata } from '@/utils/metadata';
export const GOOGLE_TEXT_MODEL = [
  {
    id: 'google/gemini-2.5-pro',
    created_date: '2024-07-09 10:00:00.000000',
    updated_date: null,
    provider_id: '198796716894742118',
    name: 'gemini-2.5-pro',
    description:
      'Enhanced thinking and reasoning, multimodal understanding, advanced coding, and more',
    human_name: 'Gemini 2.5 Pro',
    category: 'text',
    status: 'ACTIVE',
    owner: 'rapida',
    input_types: ['Audio', 'images', 'videos', 'text', 'PDF'],
    output_types: ['Text'],
  },
  {
    id: 'google/gemini-2.5-flash',
    created_date: '2024-07-09 10:00:00.000000',
    updated_date: null,
    provider_id: '198796716894742118',
    name: 'gemini-2.5-flash',
    description: 'Adaptive thinking, cost efficiency',
    human_name: 'Gemini 2.5 Flash',
    category: 'text',
    status: 'ACTIVE',
    owner: 'rapida',
    input_types: ['Audio', 'images', 'videos', 'text'],
    output_types: ['Text'],
  },
  {
    id: 'google/gemini-2.5-flash-lite-preview-06-17',
    created_date: '2024-07-09 10:00:00.000000',
    updated_date: null,
    provider_id: '198796716894742118',
    name: 'gemini-2.5-flash-lite-preview-06-17',
    description: 'Most cost-efficient model supporting high throughput',
    human_name: 'Gemini 2.5 Flash-Lite Preview',
    category: 'text',
    status: 'PREVIEW',
    owner: 'rapida',
    input_types: ['Text', 'image', 'video', 'audio'],
    output_types: ['Text'],
  },
  // ... existing code for gemini-2.5-flash-preview-native-audio-dialog and gemini-2.5-flash-exp-native-audio-thinking-dialog ...

  {
    id: 'google/gemini-2.0-flash',
    created_date: '2024-07-09 10:00:00.000000',
    updated_date: null,
    provider_id: '198796716894742118',
    name: 'gemini-2.0-flash',
    description: 'Next generation features, speed, and realtime streaming.',
    human_name: 'Gemini 2.0 Flash',
    category: 'text',
    status: 'ACTIVE',
    owner: 'rapida',
    input_types: ['Audio', 'images', 'videos', 'text'],
    output_types: ['Text'],
  },

  {
    id: 'google/gemini-2.0-flash-lite',
    created_date: '2024-07-09 10:00:00.000000',
    updated_date: null,
    provider_id: '198796716894742118',
    name: 'gemini-2.0-flash-lite',
    description: 'Cost efficiency and low latency',
    human_name: 'Gemini 2.0 Flash-Lite',
    category: 'text',
    status: 'ACTIVE',
    owner: 'rapida',
    input_types: ['Audio', 'images', 'videos', 'text'],
    output_types: ['Text'],
  },
];
export const GetGoogleTextProviderDefaultOptions = (
  current: Metadata[],
): Metadata[] => {
  const mtds: Metadata[] = [];
  const keysToKeep = [
    'rapida.credential_id',
    'model.id',
    'model.name',
    'model.frequency_penalty',
    'model.presence_penalty',

    'model.temperature',
    'model.top_p',
    'model.top_k',

    'model.candidate_count',
    'model.seed',

    'model.max_output_tokens',
    'model.response_format',
    'model.stop_sequences',
    //
  ];

  const addMetadata = (
    key: string,
    defaultValue?: string,
    validationFn?: (value: string) => boolean,
  ) => {
    const metadata = SetMetadata(current, key, defaultValue, validationFn);
    if (metadata) mtds.push(metadata);
  };
  addMetadata('model.id', GOOGLE_TEXT_MODEL[0].id, value =>
    GOOGLE_TEXT_MODEL.some(model => model.id === value),
  );

  addMetadata('model.name', GOOGLE_TEXT_MODEL[0].name, value =>
    GOOGLE_TEXT_MODEL.some(model => model.name === value),
  );

  addMetadata('model.candidate_count', '1');
  addMetadata('model.temperature', '1');
  addMetadata('model.max_output_tokens', '2048');
  addMetadata('model.stop_sequences', 'STOP!');
  addMetadata('model.presence_penalty', '0.0');
  addMetadata('model.frequency_penalty', '0.0');
  addMetadata('model.response_format');
  addMetadata('model.top_p');
  addMetadata('model.top_k');
  addMetadata('rapida.credential_id');
  addMetadata('model.seed');

  return mtds.filter(m => keysToKeep.includes(m.getKey()));
};
export const ValidateGoogleTextProviderDefaultOptions = (
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
    return 'Please check and provide a valid credentials for google.';
  }
  const modelIdOption = options.find(opt => opt.getKey() === 'model.id');
  if (
    !modelIdOption ||
    !GOOGLE_TEXT_MODEL.some(model => model.id === modelIdOption.getValue())
  ) {
    return 'Please check and select valid model from dropdown.';
  }

  const presence_penalty = options.find(
    opt => opt.getKey() === 'model.presence_penalty',
  );
  if (presence_penalty)
    if (
      isNaN(parseFloat(presence_penalty.getValue())) ||
      parseFloat(presence_penalty.getValue()) < -2 ||
      parseFloat(presence_penalty.getValue()) > 2
    ) {
      return 'Please check and provide a correct value for presence_penalty a valid value between -2 to 2.';
    }

  const frequencyPenaltyOption = options.find(
    opt => opt.getKey() === 'model.frequency_penalty',
  );
  if (frequencyPenaltyOption)
    if (
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
  if (topPOption)
    if (
      isNaN(parseFloat(topPOption.getValue())) ||
      parseFloat(topPOption.getValue()) < 0 ||
      parseFloat(topPOption.getValue()) > 1
    ) {
      return 'Please check and provide a correct value for top_p any decimal value between 0 to 1';
    }

  const maxCompletionTokensOption = options.find(
    opt => opt.getKey() === 'model.max_output_tokens',
  );
  if (
    !maxCompletionTokensOption ||
    isNaN(parseInt(maxCompletionTokensOption.getValue())) ||
    parseInt(maxCompletionTokensOption.getValue()) < 1
  ) {
    return 'Please check and provide a correct value for max_completion_tokens';
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

  return undefined;
};
