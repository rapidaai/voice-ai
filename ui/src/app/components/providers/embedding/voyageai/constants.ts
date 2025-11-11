import { SetMetadata } from '@/utils/metadata';
import { Metadata } from '@rapidaai/react';

export const VOYAGE_EMBEDDING_MODEL = [
  {
    id: '6141923083544365385',
    created_date: '2024-07-08 06:08:56.78687',
    updated_date: '2024-07-08 06:08:56.78687',
    provider_id: '5212367370329048775',
    name: 'voyage-2',
    description:
      'Voyage 2, a general-purpose embedding model optimized for a balance between cost, latency, and retrieval quality.',
    human_name: 'Voyage 2',
    category: 'embedding',
    status: 'ACTIVE',
    owner: 'rapida',
  },
  {
    id: '3647863464842661959',
    created_date: '2024-07-08 06:09:09.503759',
    updated_date: '2024-07-08 06:09:09.503759',
    provider_id: '5212367370329048775',
    name: 'voyage-code-2',
    description:
      'Voyage Code 2, an embedding model optimized for code retrieval with significant performance improvements over alternatives.',
    human_name: 'Voyage Code 2',
    category: 'embedding',
    status: 'ACTIVE',
    owner: 'rapida',
  },
  {
    id: '1656300772847215061',
    created_date: '2024-07-08 06:08:38.529226',
    updated_date: '2024-07-08 06:08:38.529226',
    provider_id: '5212367370329048775',
    name: 'voyage-finance-2',
    description:
      'Voyage Finance 2, an embedding model optimized for finance retrieval and Retrieval-Augmented Generation (RAG).',
    human_name: 'Voyage Finance 2',
    category: 'embedding',
    status: 'ACTIVE',
    owner: 'rapida',
  },
  {
    id: '2897667082301135905',
    created_date: '2024-07-08 06:09:02.827288',
    updated_date: '2024-07-08 06:09:02.827288',
    provider_id: '5212367370329048775',
    name: 'voyage-large-2',
    description:
      'Voyage Large 2, a general-purpose embedding model optimized for retrieval quality, surpassing OpenAI V3 Large.',
    human_name: 'Voyage Large 2',
    category: 'embedding',
    status: 'ACTIVE',
    owner: 'rapida',
  },
  {
    id: '5586918793397992342',
    created_date: '2024-07-08 06:08:38.458332',
    updated_date: '2024-07-08 06:08:38.458332',
    provider_id: '5212367370329048775',
    name: 'voyage-large-2-instruct',
    description:
      'Voyage Large 2 Instruct, a top-of-the-line embedding model optimized for clustering, classification, and retrieval, especially suited for instruction-tuned general-purpose tasks.',
    human_name: 'Voyage Large 2 Instruct',
    category: 'embedding',
    status: 'ACTIVE',
    owner: 'rapida',
  },
  {
    id: '4120601392584502602',
    created_date: '2024-07-08 06:09:25.250838',
    updated_date: '2024-07-08 06:09:25.250838',
    provider_id: '5212367370329048775',
    name: 'voyage-law-2',
    description:
      'Voyage Law 2, an embedding model optimized for legal and long-context retrieval and Retrieval-Augmented Generation (RAG).',
    human_name: 'Voyage Law 2',
    category: 'embedding',
    status: 'ACTIVE',
    owner: 'rapida',
  },
  {
    id: '5791646024042041680',
    created_date: '2024-07-08 06:09:32.090351',
    updated_date: '2024-07-08 06:09:32.090351',
    provider_id: '5212367370329048775',
    name: 'voyage-multilingual-2',
    description:
      'Voyage Multilingual 2, an embedding model optimized for multilingual retrieval and Retrieval-Augmented Generation (RAG).',
    human_name: 'Voyage Multilingual 2',
    category: 'embedding',
    status: 'ACTIVE',
    owner: 'rapida',
  },
];

export const GetVoyageEmbeddingDefaultOptions = (
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

  addMetadata('model.id', VOYAGE_EMBEDDING_MODEL[0].id, value =>
    VOYAGE_EMBEDDING_MODEL.some(model => model.id === value),
  );

  // Add validation for model.name
  addMetadata('model.name', VOYAGE_EMBEDDING_MODEL[0].name, value =>
    VOYAGE_EMBEDDING_MODEL.some(model => model.name === value),
  );

  return mtds.filter(m => keysToKeep.includes(m.getKey()));
};

export const ValidateVoyageEmbeddingDefaultOptions = (
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
    return 'Please provide valid credential for voyage ai.';
  }
  const modelIdOption = options.find(opt => opt.getKey() === 'model.id');
  if (
    !modelIdOption ||
    !VOYAGE_EMBEDDING_MODEL.some(model => model.id === modelIdOption.getValue())
  ) {
    return 'Please select a valid embedding model.';
  }

  return undefined;
};
