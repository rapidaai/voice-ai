import { Metadata } from '@rapidaai/react';

export const COHERE_RERANKER_MODEL = [
  {
    id: '1967168435716096',
    created_date: '2024-06-22 07:00:17.733664',
    updated_date: '2024-06-22 07:00:17.733664',
    provider_id: '1987967168435716096',
    name: 'rerank-english-v2.0',
    description:
      'This model is designed for re-ranking English text data. It supports a context size of up to 5120 tokens, making it efficient for re-ranking tasks in natural language processing applications.',
    human_name: 'Rerank English v2.0',
    category: 'rerank',
    status: 'ACTIVE',
    owner: 'rapida',
  },
  {
    id: '1967168435716097',
    created_date: '2024-06-22 07:07:39.691102',
    updated_date: '2024-06-22 07:07:39.691102',
    provider_id: '1987967168435716096',
    name: 'rerank-english-v3.0',
    description:
      'This model is designed for re-ranking English text data. It supports a context size of up to 5120 tokens, making it efficient for re-ranking tasks in natural language processing applications.',
    human_name: 'Rerank English v3.0',
    category: 'rerank',
    status: 'ACTIVE',
    owner: 'rapida',
  },
  {
    id: '1967168435716098',
    created_date: '2024-06-22 07:08:43.51891',
    updated_date: '2024-06-22 07:08:43.51891',
    provider_id: '1987967168435716096',
    name: 'rerank-multilingual-v2.0',
    description:
      'This model is designed for re-ranking multilingual text data. It supports a context size of up to 5120 tokens, making it efficient for re-ranking tasks in natural language processing applications.',
    human_name: 'Rerank multilingual v2.0',
    category: 'rerank',
    status: 'ACTIVE',
    owner: 'rapida',
  },
  {
    id: '1967168435716099',
    created_date: '2024-06-22 07:09:09.401838',
    updated_date: '2024-06-22 07:09:09.401838',
    provider_id: '1987967168435716096',
    name: 'rerank-multilingual-v3.0',
    description:
      'This model is designed for re-ranking multilingual text data. It supports a context size of up to 5120 tokens, making it efficient for re-ranking tasks in natural language processing applications.',
    human_name: 'Rerank multilingual v3.0',
    category: 'rerank',
    status: 'ACTIVE',
    owner: 'rapida',
  },
];

export const GetCohereRerankerDefaultOptions = (
  current: Metadata[],
): Metadata[] => {
  const mtds: Metadata[] = [];
  const keysToKeep = ['model.id', 'model.name'];

  const setMetadata = (
    key: string,
    defaultValue: string,
    validationFn?: (value: string) => boolean,
  ) => {
    const existingMetadata = current.find(m => m.getKey() === key);
    const value = existingMetadata ? existingMetadata.getValue() : defaultValue;
    const validValue = validationFn
      ? validationFn(value)
        ? value
        : defaultValue
      : value;
    const metadata = new Metadata();
    metadata.setKey(key);
    metadata.setValue(validValue);
    mtds.push(metadata);
  };

  setMetadata('model.id', COHERE_RERANKER_MODEL[0].id, value =>
    COHERE_RERANKER_MODEL.some(model => model.id === value),
  );

  // Add validation for model.name
  setMetadata('model.name', COHERE_RERANKER_MODEL[0].name, value =>
    COHERE_RERANKER_MODEL.some(model => model.name === value),
  );

  return mtds.filter(m => keysToKeep.includes(m.getKey()));
};

export const ValidateCohereRerankerDefaultOptions = (
  options: Metadata[],
): boolean => {
  const modelIdOption = options.find(opt => opt.getKey() === 'model.id');
  if (
    !modelIdOption ||
    !COHERE_RERANKER_MODEL.some(model => model.id === modelIdOption.getValue())
  ) {
    return false;
  }

  return true;
};
