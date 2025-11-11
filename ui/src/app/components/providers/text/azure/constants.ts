import { Metadata } from '@rapidaai/react';
import { SetMetadata } from '@/utils/metadata';

export const AZURE_TEXT_MODEL = [
  {
    id: 'azure/gpt-4o',
    created_date: '2024-09-04 02:33:51.481301',
    updated_date: null,
    provider_id: '198796716894742122',
    name: 'gpt-4o',
    description:
      'GPT-4 from OpenAI has broad general knowledge and domain expertise allowing it to follow complex instructions in natural language and solve difficult problems accurately.',
    human_name: 'OpenAI',
    category: 'text',
    status: 'ACTIVE',
    owner: 'rapida',
  },
  {
    id: 'azure/gpt-4o-mini',
    created_date: '2024-11-22 12:57:51.100805',
    updated_date: '2024-11-22 12:57:51.100805',
    provider_id: '198796716894742122',
    name: 'gpt-4o-mini',
    description:
      'GPT-4o mini enable developers to build applications that fetch data or take actions with external systems, and improved long-context performance compared to GPT-3.5 Turbo.',
    human_name: 'gpt-4o-mini',
    category: 'text',
    status: 'ACTIVE',
    owner: 'rapida',
  },
  {
    id: 'azure/gpt-4.1-nano',
    created_date: '2024-11-22 13:00:00.000000',
    updated_date: '2024-11-22 13:00:00.000000',
    provider_id: '198796716894742123',
    name: 'gpt-4.1-nano',
    description:
      'GPT-4.1 nano is a compact and efficient version of GPT-4, offering improved performance and reduced resource requirements for various natural language processing tasks.',
    human_name: 'gpt-4.1-nano',
    category: 'text',
    status: 'ACTIVE',
    owner: 'rapida',
  },
];

export const GetAzureTextProviderDefaultOptions = (
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
    'model.metadata',
    'model.response_format',
    'model.stop',
    'model.tool_choice',
    'model.user',
  ];

  const addMetadata = (
    key: string,
    defaultValue?: string,
    validationFn?: (value: string) => boolean,
  ) => {
    const metadata = SetMetadata(current, key, defaultValue, validationFn);
    if (metadata) mtds.push(metadata);
  };

  addMetadata('model.id', AZURE_TEXT_MODEL[0].id, value =>
    AZURE_TEXT_MODEL.some(model => model.id === value),
  );

  // Add validation for model.name
  addMetadata('model.name', AZURE_TEXT_MODEL[0].name, value =>
    AZURE_TEXT_MODEL.some(model => model.name === value),
  );
  addMetadata('model.frequency_penalty', '0');
  addMetadata('model.temperature', '0.7');
  addMetadata('model.top_p', '1');
  addMetadata('model.presence_penalty');
  addMetadata('model.max_completion_tokens', '2048');
  addMetadata('model.response_format');
  addMetadata('model.stop');
  addMetadata('model.tool_choice');
  addMetadata('model.user');
  addMetadata('model.metadata');
  addMetadata('rapida.credential_id', '');
  return mtds.filter(m => keysToKeep.includes(m.getKey()));
};
export const ValidateAzureTextProviderDefaultOptions = (
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
    return 'Please check and provide a valid credentials for azure openai';
  }
  const modelIdOption = options.find(opt => opt.getKey() === 'model.id');
  if (
    !modelIdOption ||
    !AZURE_TEXT_MODEL.some(model => model.id === modelIdOption.getValue())
  ) {
    return 'Please check and select valid model from dropdown.';
  }

  const frequencyPenaltyOption = options.find(
    opt => opt.getKey() === 'model.frequency_penalty',
  );
  if (frequencyPenaltyOption) {
    const frequencyPenalty = parseFloat(frequencyPenaltyOption.getValue());
    if (
      isNaN(frequencyPenalty) ||
      frequencyPenalty < -2 ||
      frequencyPenalty > 2
    ) {
      console.log('Invalid model.frequency_penalty');
      return 'Please check and provide a correct value for frequency_penalty a valid value between -2 to 2.';
    }
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
    console.log('Invalid or missing model.temperature');
    return 'Please check and provide a correct value for temperature any decimal value between 0 to 1';
  }

  const topPOption = options.find(opt => opt.getKey() === 'model.top_p');
  if (
    !topPOption ||
    isNaN(parseFloat(topPOption.getValue())) ||
    parseFloat(topPOption.getValue()) < 0 ||
    parseFloat(topPOption.getValue()) > 1
  ) {
    console.log('Invalid or missing model.top_p');
    return 'Please check and provide a correct value for top_p any decimal value between 0 to 1';
  }

  const presencePenaltyOption = options.find(
    opt => opt.getKey() === 'model.presence_penalty',
  );
  if (presencePenaltyOption)
    if (
      isNaN(parseFloat(presencePenaltyOption.getValue())) ||
      parseFloat(presencePenaltyOption.getValue()) < -2 ||
      parseFloat(presencePenaltyOption.getValue()) > 2
    ) {
      console.log('Invalid or missing model.presence_penalty');
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
    console.log('Invalid or missing model.max_completion_tokens');
    return 'Please check and provide a correct value for max_completion_tokens it should be greater then 1.';
  }

  const responseFormatOption = options.find(
    opt => opt.getKey() === 'model.response_format',
  );
  if (responseFormatOption) {
    try {
      const parsedFormat = JSON.parse(responseFormatOption.getValue());
      if (typeof parsedFormat !== 'object' || !parsedFormat.type) {
        console.log(
          'Invalid model.response_format: not an object or missing type',
        );
        return 'Please check and provide a correct value for response_format it should be a valid json object.';
      }
      if (!['text', 'json_object', 'json_schema'].includes(parsedFormat.type)) {
        console.log('Invalid model.response_format: unsupported type');
        return 'Please check and provide a correct value for response_format it should have type with text, json_object, json_schema.';
      }
      if (parsedFormat.type === 'json_schema' && !parsedFormat.json_schema) {
        return 'Please check and provide a correct value for response_format it should have valid json_schema.';
      }
    } catch (error) {
      console.log('Invalid model.response_format: JSON parsing error');
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
