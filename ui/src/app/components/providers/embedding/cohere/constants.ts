import { SetMetadata } from '@/utils/metadata';
import { Metadata } from '@rapidaai/react';

export const COHERE_EMBEDDING_MODEL = [
  {
    id: '987967168435716096',
    created_date: '2024-05-31 06:50:36.130312',
    updated_date: null,
    provider_id: '1987967168435716096',
    name: 'embed-multilingual-v3.0',
    description:
      'This model provides multilingual text embeddings, supporting a context size of 1024 tokens and up to 48 chunks in parallel. It is suitable for tasks requiring multilingual text representation learning.',
    human_name: 'Multilingual Embedding v3.0',
    category: 'embedding',
    status: 'ACTIVE',
    owner: 'rapida',
  },
  {
    id: '2662160914657700600',
    created_date: '2024-07-08 06:29:42.597405',
    updated_date: null,
    provider_id: '1987967168435716096',
    name: 'embed-english-v3.0',
    description:
      'A model that allows for text to be classified or turned into embeddings. English only.',
    human_name: 'English Embedding v3.0',
    category: 'embedding',
    status: 'ACTIVE',
    owner: 'rapida',
  },
  {
    id: '147378794724762721',
    created_date: '2024-07-08 06:29:49.625961',
    updated_date: null,
    provider_id: '1987967168435716096',
    name: 'embed-english-light-v3.0',
    description:
      'A smaller, faster version of embed-english-v3.0. Almost as capable, but a lot faster. English only.',
    human_name: 'English Light Embedding v3.0',
    category: 'embedding',
    status: 'ACTIVE',
    owner: 'rapida',
  },
  {
    id: '2091619879999091992',
    created_date: '2024-07-08 06:29:55.033843',
    updated_date: null,
    provider_id: '1987967168435716096',
    name: 'embed-multilingual-light-v3.0',
    description:
      'A smaller, faster version of embed-multilingual-v3.0. Almost as capable, but a lot faster. Supports multiple languages.',
    human_name: 'Multilingual Light Embedding v3.0',
    category: 'embedding',
    status: 'ACTIVE',
    owner: 'rapida',
  },
  {
    id: '523212506935277337',
    created_date: '2024-07-08 06:30:02.191672',
    updated_date: null,
    provider_id: '1987967168435716096',
    name: 'embed-english-v2.0',
    description:
      'Our older embeddings model that allows for text to be classified or turned into embeddings. English only.',
    human_name: 'English Embedding v2.0',
    category: 'embedding',
    status: 'ACTIVE',
    owner: 'rapida',
  },
  {
    id: '5180946549999591272',
    created_date: '2024-07-08 06:30:08.230456',
    updated_date: null,
    provider_id: '1987967168435716096',
    name: 'embed-english-light-v2.0',
    description:
      'A smaller, faster version of embed-english-v2.0. Almost as capable, but a lot faster. English only.',
    human_name: 'English Light Embedding v2.0',
    category: 'embedding',
    status: 'ACTIVE',
    owner: 'rapida',
  },
  {
    id: '246348370119325392',
    created_date: '2024-07-08 06:30:14.407878',
    updated_date: null,
    provider_id: '1987967168435716096',
    name: 'embed-multilingual-v2.0',
    description:
      'Provides multilingual classification and embedding support. See supported languages here.',
    human_name: 'Multilingual Embedding v2.0',
    category: 'embedding',
    status: 'ACTIVE',
    owner: 'rapida',
  },
];

export const GetCohereEmbeddingDefaultOptions = (
  current: Metadata[],
): Metadata[] => {
  const mtds: Metadata[] = [];
  const keysToKeep = ['rapida.credential_id', 'model.id', 'model.name'];

  const addMetadata = (
    key: string,
    defaultValue?: string,
    validationFn?: (value: string) => boolean,
  ) => {
    const metadata = SetMetadata(current, key, defaultValue, validationFn);
    if (metadata) mtds.push(metadata);
  };
  addMetadata('rapida.credential_id');

  addMetadata('model.id', COHERE_EMBEDDING_MODEL[0].id, value =>
    COHERE_EMBEDDING_MODEL.some(model => model.id === value),
  );

  // Add validation for model.name
  addMetadata('model.name', COHERE_EMBEDDING_MODEL[0].name, value =>
    COHERE_EMBEDDING_MODEL.some(model => model.name === value),
  );

  return mtds.filter(m => keysToKeep.includes(m.getKey()));
};

export const ValidateCohereEmbeddingDefaultOptions = (
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
    return 'Please provide valid credential for cohere.';
  }
  const modelIdOption = options.find(opt => opt.getKey() === 'model.id');
  if (
    !modelIdOption ||
    !COHERE_EMBEDDING_MODEL.some(model => model.id === modelIdOption.getValue())
  ) {
    return 'Please select valid embedding model.';
  }
  return undefined;
};
