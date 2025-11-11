import { Metadata } from '@rapidaai/react';
import { SetMetadata } from '@/utils/metadata';

export const ANTHROPIC_TEXT_MODEL = [
  {
    id: 'anthropic/claude-opus-4-20250514',
    created_date: '2025-05-14',
    updated_date: null,
    provider_id: '1987967168347635712',
    name: 'claude-opus-4-20250514',
    description: 'Claude Opus 4, the most advanced model in the Claude series.',
    human_name: 'Claude Opus 4',
    category: 'text',
    status: 'ACTIVE',
    owner: 'rapida',
  },
  {
    id: 'anthropic/claude-sonnet-4-20250514',
    created_date: '2025-05-14',
    updated_date: null,
    provider_id: '1987967168347635712',
    name: 'claude-sonnet-4-20250514',
    description: 'Claude Sonnet 4, an improved version of the Sonnet series.',
    human_name: 'Claude Sonnet 4',
    category: 'text',
    status: 'ACTIVE',
    owner: 'rapida',
  },
  {
    id: 'anthropic/claude-3-7-sonnet-20250219',
    created_date: '2025-02-19',
    updated_date: null,
    provider_id: '1987967168347635712',
    name: 'claude-3-7-sonnet-20250219',
    description: 'Claude Sonnet 3.7, the latest version of the 3.7 series.',
    human_name: 'Claude Sonnet 3.7',
    category: 'text',
    status: 'ACTIVE',
    owner: 'rapida',
  },
  {
    id: 'anthropic/claude-3-5-haiku-20241022',
    created_date: '2024-10-22',
    updated_date: null,
    provider_id: '1987967168347635712',
    name: 'claude-3-5-haiku-20241022',
    description: 'Claude Haiku 3.5, the latest version of the Haiku series.',
    human_name: 'Claude Haiku 3.5',
    category: 'text',
    status: 'ACTIVE',
    owner: 'rapida',
  },
  {
    id: 'anthropic/claude-3-5-sonnet-20241022',
    created_date: '2024-10-22',
    updated_date: null,
    provider_id: '1987967168347635712',
    name: 'claude-3-5-sonnet-20241022',
    description:
      'Claude Sonnet 3.5 v2, the latest version of the 3.5 Sonnet series.',
    human_name: 'Claude Sonnet 3.5 v2',
    category: 'text',
    status: 'ACTIVE',
    owner: 'rapida',
  },
];

export const GetAnthropicTextProviderDefaultOptions = (
  current: Metadata[],
): Metadata[] => {
  const mtds: Metadata[] = [];
  const keysToKeep = [
    'rapida.credential_id',
    'model.id',
    'model.name',
    'model.max_tokens',
    'model.temperature',
    'model.top_k',
    'model.top_p',
    'model.stop_sequences',
    'model.metadata',
    'model.container',
    'model.service_tier',
    'model.thinking',
  ];

  const addMetadata = (
    key: string,
    defaultValue?: string,
    validationFn?: (value: string) => boolean,
  ) => {
    const metadata = SetMetadata(current, key, defaultValue, validationFn);
    if (metadata) mtds.push(metadata);
  };
  addMetadata('model.id', ANTHROPIC_TEXT_MODEL[0].id, value =>
    ANTHROPIC_TEXT_MODEL.some(model => model.id === value),
  );

  // Add validation for model.name
  addMetadata('model.name', ANTHROPIC_TEXT_MODEL[0].name, value =>
    ANTHROPIC_TEXT_MODEL.some(model => model.name === value),
  );
  addMetadata('model.max_tokens', '1028');
  addMetadata('model.temperature', '1.0');
  addMetadata('model.service_tier', 'auto');

  addMetadata('model.thinking');
  addMetadata('model.top_k');
  addMetadata('model.top_p');
  addMetadata('model.stop_sequences');
  addMetadata('model.metadata');
  addMetadata('model.container');
  addMetadata('rapida.credential_id');

  return mtds.filter(m => keysToKeep.includes(m.getKey()));
};

export const ValidateAnthropicTextProviderDefaultOptions = (
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
    return 'Please check and provide a valid credentials for anthropic.';
  }

  const modelIdOption = options.find(opt => opt.getKey() === 'model.id');
  if (
    !modelIdOption ||
    !ANTHROPIC_TEXT_MODEL.some(model => model.id === modelIdOption.getValue())
  ) {
    return 'Please check and select valid model from dropdown.';
  }

  const top_k = options.find(opt => opt.getKey() === 'model.top_k');
  if (top_k) {
    if (
      isNaN(parseFloat(top_k.getValue())) ||
      parseFloat(top_k.getValue()) < -2 ||
      parseFloat(top_k.getValue()) > 2
    ) {
      return 'Please check and provide a correct value for top_k a valid value between -2 to 2.';
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
    return 'Please check and provide a correct value for temperature any decimal value between 0 to 1';
  }

  const topPOption = options.find(opt => opt.getKey() === 'model.top_p');
  if (topPOption) {
    if (
      isNaN(parseFloat(topPOption.getValue())) ||
      parseFloat(topPOption.getValue()) < 0 ||
      parseFloat(topPOption.getValue()) > 1
    ) {
      return 'Please check and provide a correct value for top_p any decimal value between 0 to 1';
    }
  }

  const maxCompletionTokensOption = options.find(
    opt => opt.getKey() === 'model.max_tokens',
  );
  if (
    !maxCompletionTokensOption ||
    isNaN(parseInt(maxCompletionTokensOption.getValue())) ||
    parseInt(maxCompletionTokensOption.getValue()) < 1
  ) {
    return 'Please check and provide a correct value for max_tokens.';
  }

  const thinking = options.find(opt => opt.getKey() === 'model.thinking');
  if (thinking) {
    try {
      JSON.parse(thinking.getValue());
    } catch (error) {
      return 'Please check and provide a correct value for thinking.';
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
