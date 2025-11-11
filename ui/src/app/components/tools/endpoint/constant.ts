import { Metadata } from '@rapidaai/react';
import { SetMetadata } from '@/utils/metadata';

export const GetEndpointDefaultOptions = (current: Metadata[]): Metadata[] => {
  const mtds: Metadata[] = [];
  const keysToKeep = ['tool.endpoint_id', 'tool.parameters'];

  const addMetadata = (
    key: string,
    defaultValue?: string,
    validationFn?: (value: string) => boolean,
  ) => {
    const metadata = SetMetadata(current, key, defaultValue, validationFn);
    if (metadata) mtds.push(metadata);
  };

  addMetadata('tool.endpoint_id');
  addMetadata('tool.parameters');
  return mtds.filter(m => keysToKeep.includes(m.getKey()));
};
export const ValidateEndpointDefaultOptions = (
  options: Metadata[],
): boolean => {
  const requiredKeys = ['tool.endpoint_id', 'tool.parameters'];
  const foundKeys = new Set<string>();

  for (const option of options) {
    const key = option.getKey();
    foundKeys.add(key);

    if (key === 'tool.endpoint_id') {
      const value = option.getValue();
      if (typeof value !== 'string' || value === '') {
        return false;
      }
    }

    if (key === 'tool.parameters') {
      const value = option.getValue();
      if (typeof value !== 'string' || value === '') {
        return false;
      }

      try {
        const parameters = JSON.parse(value);

        if (
          typeof parameters !== 'object' ||
          parameters === null ||
          Array.isArray(parameters)
        ) {
          return false;
        }

        const entries = Object.entries(parameters);

        if (entries.length === 0) {
          return false;
        }

        for (const [paramKey, paramValue] of entries) {
          const [type, key] = paramKey.split('.');
          if (
            !type ||
            !key ||
            typeof paramValue !== 'string' ||
            paramValue === ''
          ) {
            return false;
          }
        }

        const values = entries.map(([, value]) => value);
        const uniqueValues = new Set(values);
        if (values.length !== uniqueValues.size) {
          return false;
        }
      } catch (e) {
        return false;
      }
    }
  }

  return requiredKeys.every(key => foundKeys.has(key));
};
export const EndpointToolDefintion = {
  name: 'llm_call',
  description:
    'Use it to make calls to a Language Learning Model. Specify the prompt and optional parameters such as temperature or max_tokens.',
  parameters: JSON.stringify(
    {
      properties: {
        prompt: {
          description: 'The input text or prompt for the LLM.',
          type: 'string',
        },
      },
      required: ['prompt'],
      type: 'object',
    },
    null,
    2,
  ),
};
