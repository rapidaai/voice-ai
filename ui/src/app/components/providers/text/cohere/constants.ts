import { Metadata } from '@rapidaai/react';
import { SetMetadata } from '@/utils/metadata';

export const COHERE_TEXT_MODEL = [
  {
    id: 'cohere/command-a-03-2025',
    created_date: '2025-03-01 10:00:00.000000',
    updated_date: null,
    provider_id: '298796716894742123',
    name: 'command-a-03-2025',
    description:
      'Command A is our most performant model to date, excelling at tool use, agents, retrieval augmented generation (RAG), and multilingual use cases. Command A has a context length of 256K, only requires two GPUs to run, and has 150% higher throughput compared to Command R+ 08-2024.',
    human_name: 'Cohere',
    category: 'text',
    status: 'ACTIVE',
    owner: 'rapida',
  },
  {
    id: 'cohere/command-r7b-12-2024',
    created_date: '2024-12-15 09:00:00.000000',
    updated_date: null,
    provider_id: '298796716894742123',
    name: 'command-r7b-12-2024',
    description:
      'command-r7b-12-2024 is a small, fast update delivered in December 2024. It excels at RAG, tool use, agents, and similar tasks requiring complex reasoning and multiple steps.',
    human_name: 'Cohere',
    category: 'text',
    status: 'ACTIVE',
    owner: 'rapida',
  },
  {
    id: 'cohere/command-r-plus-04-2024',
    created_date: '2024-04-01 08:00:00.000000',
    updated_date: null,
    provider_id: '298796716894742123',
    name: 'command-r-plus-04-2024',
    description:
      'Command R+ is an instruction-following conversational model that performs language tasks at a higher quality, more reliably, and with a longer context than previous models. It is best suited for complex RAG workflows and multi-step tool use.',
    human_name: 'Cohere',
    category: 'text',
    status: 'ACTIVE',
    owner: 'rapida',
  },
  {
    id: 'cohere/command-r-plus',
    created_date: '2024-04-01 08:00:00.000000',
    updated_date: null,
    provider_id: '298796716894742123',
    name: 'command-r-plus',
    description:
      "command-r-plus is an alias for command-r-plus-04-2024, so if you use command-r-plus in the API, that's the model you're pointing to.",
    human_name: 'Cohere',
    category: 'text',
    status: 'ACTIVE',
    owner: 'rapida',
  },
  {
    id: 'cohere/command-r-08-2024',
    created_date: '2024-08-15 11:00:00.000000',
    updated_date: null,
    provider_id: '298796716894742123',
    name: 'command-r-08-2024',
    description:
      'command-r-08-2024 is an update of the Command R model, delivered in August 2024.',
    human_name: 'Cohere',
    category: 'text',
    status: 'ACTIVE',
    owner: 'rapida',
  },
  {
    id: 'cohere/command-r-03-2024',
    created_date: '2024-03-15 07:00:00.000000',
    updated_date: null,
    provider_id: '298796716894742123',
    name: 'command-r-03-2024',
    description:
      'Command R is an instruction-following conversational model that performs language tasks at a higher quality, more reliably, and with a longer context than previous models. It can be used for complex workflows like code generation, retrieval augmented generation (RAG), tool use, and agents.',
    human_name: 'Cohere',
    category: 'text',
    status: 'ACTIVE',
    owner: 'rapida',
  },
  {
    id: 'cohere/command-r',
    created_date: '2024-03-15 07:00:00.000000',
    updated_date: null,
    provider_id: '298796716894742123',
    name: 'command-r',
    description:
      "command-r is an alias for command-r-03-2024, so if you use command-r in the API, that's the model you're pointing to.",
    human_name: 'Cohere',
    category: 'text',
    status: 'ACTIVE',
    owner: 'rapida',
  },
  {
    id: 'cohere/command',
    created_date: '2023-09-01 12:00:00.000000',
    updated_date: null,
    provider_id: '298796716894742123',
    name: 'command',
    description:
      'An instruction-following conversational model that performs language tasks with high quality, more reliably and with a longer context than our base generative models.',
    human_name: 'Cohere',
    category: 'text',
    status: 'ACTIVE',
    owner: 'rapida',
  },
  {
    id: 'cohere/command-nightly',
    created_date: '2024-01-01 00:00:00.000000',
    updated_date: null,
    provider_id: '298796716894742123',
    name: 'command-nightly',
    description:
      'To reduce the time between major releases, we put out nightly versions of command models. For command, that is command-nightly. Be advised that command-nightly is the latest, most experimental, and (possibly) unstable version of its default counterpart. Nightly releases are updated regularly, without warning, and are not recommended for production use.',
    human_name: 'Cohere',
    category: 'text',
    status: 'ACTIVE',
    owner: 'rapida',
  },
  {
    id: 'cohere/command-light',
    created_date: '2023-10-01 12:00:00.000000',
    updated_date: null,
    provider_id: '298796716894742123',
    name: 'command-light',
    description:
      'A smaller, faster version of command. Almost as capable, but a lot faster.',
    human_name: 'Cohere',
    category: 'text',
    status: 'ACTIVE',
    owner: 'rapida',
  },
  {
    id: 'cohere/command-light-nightly',
    created_date: '2024-01-01 00:00:00.000000',
    updated_date: null,
    provider_id: '298796716894742123',
    name: 'command-light-nightly',
    description:
      'To reduce the time between major releases, we put out nightly versions of command models. For command-light, that is command-light-nightly. Be advised that command-light-nightly is the latest, most experimental, and (possibly) unstable version of its default counterpart. Nightly releases are updated regularly, without warning, and are not recommended for production use.',
    human_name: 'Cohere',
    category: 'text',
    status: 'ACTIVE',
    owner: 'rapida',
  },
];

export const GetCohereTextProviderDefaultOptions = (
  current: Metadata[],
): Metadata[] => {
  const mtds: Metadata[] = [];
  const keysToKeep = [
    'rapida.credential_id',
    'model.id',
    'model.name',
    'model.max_tokens',
    'model.temperature',
    'model.p',
    'model.k',
    'model.frequency_penalty',
    'model.presence_penalty',
    'model.stop_sequences',
    'model.safety_mode',
    'model.seed',
    'model.response_format',
  ];
  const addMetadata = (
    key: string,
    defaultValue?: string,
    validationFn?: (value: string) => boolean,
  ) => {
    const metadata = SetMetadata(current, key, defaultValue, validationFn);
    if (metadata) mtds.push(metadata);
  };
  addMetadata('model.id', COHERE_TEXT_MODEL[0].id, value =>
    COHERE_TEXT_MODEL.some(model => model.id === value),
  );

  // Add validation for model.name
  addMetadata('model.name', COHERE_TEXT_MODEL[0].name, value =>
    COHERE_TEXT_MODEL.some(model => model.name === value),
  );
  addMetadata('rapida.credential_id');
  addMetadata('model.max_tokens', '2048');
  addMetadata('model.temperature');
  addMetadata('model.p');
  addMetadata('model.k');
  addMetadata('model.frequency_penalty');
  addMetadata('model.presence_penalty');
  addMetadata('model.stop_sequences', '');
  addMetadata('model.safety_mode', 'CONTEXTUAL');
  addMetadata('model.seed');
  addMetadata('model.response_format');

  return mtds.filter(m => keysToKeep.includes(m.getKey()));
};

export const ValidateCohereTextProviderDefaultOptions = (
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
    !COHERE_TEXT_MODEL.some(model => model.id === modelIdOption.getValue())
  ) {
    return 'Please check and select valid model from dropdown.';
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
      return 'Please check and provide a correct value for frequency_penalty any decimal value between -2 to 2';
    }
  }

  const presence_penalty = options.find(
    opt => opt.getKey() === 'model.presence_penalty',
  );
  if (presence_penalty) {
    const presencepenalty = parseFloat(presence_penalty.getValue());
    if (isNaN(presencepenalty) || presencepenalty < -2 || presencepenalty > 2) {
      console.log('Invalid model.presence_penalty');
      return 'Please check and provide a correct value for presence_penalty any decimal value between -2 to 2';
    }
  }

  const topPOption = options.find(opt => opt.getKey() === 'model.p');
  if (topPOption)
    if (
      isNaN(parseFloat(topPOption.getValue())) ||
      parseFloat(topPOption.getValue()) < 0 ||
      parseFloat(topPOption.getValue()) > 1
    ) {
      return 'Please check and provide a correct value for top_p any decimal value between 0 to 1';
    }

  const k = options.find(opt => opt.getKey() === 'model.k');
  if (k)
    if (
      isNaN(parseFloat(k.getValue())) ||
      parseFloat(k.getValue()) < -2 ||
      parseFloat(k.getValue()) > 2
    ) {
      return 'Please check and provide a correct value for top_k any decimal value between -2 to 2';
    }

  const maxCompletionTokensOption = options.find(
    opt => opt.getKey() === 'model.max_tokens',
  );
  if (
    !maxCompletionTokensOption ||
    isNaN(parseInt(maxCompletionTokensOption.getValue())) ||
    parseInt(maxCompletionTokensOption.getValue()) < 1
  ) {
    return 'Please check and provide a correct value for max_completion_token.';
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
