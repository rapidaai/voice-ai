import { SetMetadata } from '@/utils/metadata';
import { Metadata } from '@rapidaai/react';
export const GOOGLE_EMBEDDING_MODEL = [
  {
    id: '6835060815855074623',
    created_date: '2024-07-08 06:45:54.262232',
    updated_date: '2024-07-08 06:45:54.262232',
    provider_id: '198796716894742118',
    name: 'text-embedding-004',
    description:
      'Text Embedding 004, a model designed for high-performance text embedding tasks.',
    human_name: 'Text Embedding 004',
    category: 'embedding',
    status: 'ACTIVE',
    owner: 'rapida',
  },
  {
    id: '6002401716480832659',
    created_date: '2024-07-08 06:46:00.83089',
    updated_date: '2024-07-08 06:46:00.83089',
    provider_id: '198796716894742118',
    name: 'text-multilingual-embedding-002',
    description:
      'Text Multilingual Embedding 002, a multilingual model designed for high-performance text embedding tasks.',
    human_name: 'Text Multilingual Embedding 002',
    category: 'embedding',
    status: 'ACTIVE',
    owner: 'rapida',
  },
  {
    id: '9186655839646578916',
    created_date: '2024-07-08 06:46:13.746951',
    updated_date: '2024-07-08 06:46:13.746951',
    provider_id: '198796716894742118',
    name: 'textembedding-gecko-multilingual@001',
    description:
      'Text Embedding Gecko Multilingual 001, a multilingual model designed for high-performance text embedding tasks.',
    human_name: 'Text Embedding Gecko Multilingual 001',
    category: 'embedding',
    status: 'ACTIVE',
    owner: 'rapida',
  },
  {
    id: '49235357547542843',
    created_date: '2024-07-08 06:46:21.1319',
    updated_date: '2024-07-08 06:46:21.1319',
    provider_id: '198796716894742118',
    name: 'textembedding-gecko@001',
    description:
      'Text Embedding Gecko 001, a model designed for high-performance text embedding tasks.',
    human_name: 'Text Embedding Gecko 001',
    category: 'embedding',
    status: 'ACTIVE',
    owner: 'rapida',
  },
  {
    id: '9098129614847368134',
    created_date: '2024-07-08 06:46:07.741271',
    updated_date: '2024-07-08 06:46:07.741271',
    provider_id: '198796716894742118',
    name: 'textembedding-gecko@003',
    description:
      'Text Embedding Gecko 003, an updated model designed for high-performance text embedding tasks.',
    human_name: 'Text Embedding Gecko 003',
    category: 'embedding',
    status: 'ACTIVE',
    owner: 'rapida',
  },
];

export const GetGoogleEmbeddingDefaultOptions = (
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

  addMetadata('model.id', GOOGLE_EMBEDDING_MODEL[0].id, value =>
    GOOGLE_EMBEDDING_MODEL.some(model => model.id === value),
  );

  // Add validation for model.name
  addMetadata('model.name', GOOGLE_EMBEDDING_MODEL[0].name, value =>
    GOOGLE_EMBEDDING_MODEL.some(model => model.name === value),
  );

  return mtds.filter(m => keysToKeep.includes(m.getKey()));
};

export const ValidateGoogleEmbeddingDefaultOptions = (
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
    return 'Please provide valid credential for google.';
  }
  const modelIdOption = options.find(opt => opt.getKey() === 'model.id');
  if (
    !modelIdOption ||
    !GOOGLE_EMBEDDING_MODEL.some(model => model.id === modelIdOption.getValue())
  ) {
    return 'Please select a valid embedding model.';
  }

  return undefined;
};
