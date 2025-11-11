import { Metadata } from '@rapidaai/react';
import { SetMetadata } from '@/utils/metadata';

export const GetKnowledgeRetrievalDefaultOptions = (
  current: Metadata[],
): Metadata[] => {
  const mtds: Metadata[] = [];

  const keysToKeep = [
    'tool.search_type',
    'tool.knowledge_id',
    'tool.top_k',
    'tool.score_threshold',
  ];

  const addMetadata = (
    key: string,
    defaultValue?: string,
    validationFn?: (value: string) => boolean,
  ) => {
    const metadata = SetMetadata(current, key, defaultValue, validationFn);
    if (metadata) mtds.push(metadata);
  };

  addMetadata('tool.search_type', 'hybrid');
  addMetadata('tool.top_k', '5');
  addMetadata('tool.score_threshold', '0.5');
  addMetadata('tool.knowledge_id');
  return mtds.filter(m => keysToKeep.includes(m.getKey()));
};

export const ValidateKnowledgeRetrievalDefaultOptions = (
  options: Metadata[],
): boolean => {
  const requiredKeys = [
    'tool.search_type',
    'tool.knowledge_id',
    'tool.top_k',
    'tool.score_threshold',
  ];
  const allowedSearchTypes = ['semantic', 'fullText', 'hybrid'];

  // Check if all required keys are present
  for (const key of requiredKeys) {
    if (!options.some(option => option.getKey() === key)) {
      return false;
    }
  }

  for (const option of options) {
    if (
      option.getKey() === 'search_type' &&
      !allowedSearchTypes.includes(option.getValue())
    ) {
      return false;
    }

    if (option.getKey() === 'top_k') {
      const topK = Number(option.getValue());
      if (isNaN(topK) || topK < 1 || topK > 10) {
        return false;
      }
    }

    if (option.getKey() === 'score_threshold') {
      const scoreThreshold = Number(option.getValue());
      if (
        isNaN(scoreThreshold) ||
        scoreThreshold < 0.1 ||
        scoreThreshold > 0.9
      ) {
        return false;
      }
    }
  }

  return true;
};

export const KnowledgeRetrievalToolDefintion = {
  name: 'knowledge_query',
  description:
    'Use this tool to retrieve specific information or data from provided queries before responding.',
  parameters: JSON.stringify(
    {
      properties: {
        context: {
          description:
            'Concise and searchable description of the users query or topic.',
          type: 'string',
        },
        organizations: {
          description:
            'Names of organizations or companies mentioned in the content',
          items: {
            type: 'string',
          },
          type: 'array',
        },
        products: {
          description: 'Names of products or services mentioned in the content',
          items: {
            type: 'string',
          },
          type: 'array',
        },
      },
      required: ['context'],
      type: 'object',
    },
    null,
    2,
  ),
};
