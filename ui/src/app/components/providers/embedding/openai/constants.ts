import { SetMetadata } from '@/utils/metadata';
import { Metadata } from '@rapidaai/react';

export const OPENAI_EMBEDDING_MODEL = [
  {
    id: '87967168452493312',
    created_date: '2024-05-31 06:16:13.930444',
    updated_date: null,
    provider_id: '1987967168452493312',
    name: 'text-embedding-3-large',
    description:
      'This model is designed for generating high-quality text embeddings, suitable for a variety of natural language processing tasks. It supports a context size of up to 8191 tokens and can process up to 32 chunks in parallel, making it highly efficient for large-scale text analysis and machine learning applications.',
    human_name: 'Text Embedding 3 Large',
    category: 'embedding',
    status: 'ACTIVE',
    owner: 'rapida',
  },
  {
    id: '86967168452493312',
    created_date: '2024-05-31 06:34:59.294085',
    updated_date: null,
    provider_id: '1987967168452493312',
    name: 'text-embedding-3-small',
    description:
      'This model is designed for generating high-quality text embeddings, suitable for a variety of natural language processing tasks. It supports a context size of up to 8191 tokens and can process up to 32 chunks in parallel, making it highly efficient for large-scale text analysis and machine learning applications.',
    human_name: 'Text Embedding 3 Small',
    category: 'embedding',
    status: 'ACTIVE',
    owner: 'rapida',
  },
  {
    id: '85967168452493312',
    created_date: '2024-05-31 06:40:06.274429',
    updated_date: null,
    provider_id: '1987967168452493312',
    name: 'text-embedding-ada-002',
    description: 'Description of text-embedding-ada-002 model goes here',
    human_name: 'Text Embedding ADA 002',
    category: 'embedding',
    status: 'ACTIVE',
    owner: 'rapida',
  },
];

export const GetOpenaiEmbeddingDefaultOptions = (
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

  addMetadata('model.id', OPENAI_EMBEDDING_MODEL[0].id, value =>
    OPENAI_EMBEDDING_MODEL.some(model => model.id === value),
  );

  // Add validation for model.name
  addMetadata('model.name', OPENAI_EMBEDDING_MODEL[0].name, value =>
    OPENAI_EMBEDDING_MODEL.some(model => model.name === value),
  );
  return mtds.filter(m => keysToKeep.includes(m.getKey()));
};

export const ValidateOpenaiEmbeddingDefaultOptions = (
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
    !OPENAI_EMBEDDING_MODEL.some(model => model.id === modelIdOption.getValue())
  ) {
    return 'Please select a valid embedding model.';
  }

  return undefined;
};
