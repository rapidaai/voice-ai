import { Metadata } from '@rapidaai/react';
import { SetMetadata } from '@/utils/metadata';

export const APIRequestToolDefintion = {
  name: 'api_call',
  description:
    'Use it to perform API calls by specifying the URL, HTTP verb, and other configurations. Parameters should be simple key-value pairs.',
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

export const GetAPIRequestDefaultOptions = (
  current: Metadata[],
): Metadata[] => {
  const mtds: Metadata[] = [];

  const keysToKeep = [
    'tool.method',
    'tool.endpoint',
    'tool.headers',
    'tool.parameters',
  ];

  const addMetadata = (
    key: string,
    defaultValue?: string,
    validationFn?: (value: string) => boolean,
  ) => {
    const metadata = SetMetadata(current, key, defaultValue, validationFn);
    if (metadata) mtds.push(metadata);
  };

  addMetadata('tool.method', 'POST');
  addMetadata('tool.endpoint');
  addMetadata('tool.headers');
  addMetadata('tool.parameters');
  return mtds.filter(m => keysToKeep.includes(m.getKey()));
};

export const ValidateAPIRequestDefaultOptions = (
  options: Metadata[],
): boolean => {
  const requiredKeys = ['tool.method', 'tool.endpoint', 'tool.parameters'];
  const validMethods = ['GET', 'POST', 'PUT', 'DELETE', 'PATCH'];

  // Check if all required keys are present
  const hasAllRequiredKeys = requiredKeys.every(key =>
    options.some(option => option.getKey() === key),
  );

  if (!hasAllRequiredKeys) {
    console.error('Missing required metadata keys');
    return false;
  }

  // Validate method
  const methodOption = options.find(
    option => option.getKey() === 'tool.method',
  );
  if (
    methodOption &&
    !validMethods.includes(methodOption.getValue().toUpperCase())
  ) {
    console.error('Invalid HTTP method');
    return false;
  }

  // Validate endpoint (check for valid URL)
  const endpointOption = options.find(
    option => option.getKey() === 'tool.endpoint',
  );
  if (endpointOption) {
    try {
      new URL(endpointOption.getValue());
    } catch (error) {
      console.error('Invalid endpoint URL');
      return false;
    }
  }
  // Validate headers
  const headersOption = options.find(
    option => option.getKey() === 'tool.headers',
  );
  if (headersOption) {
    if (
      !headersOption.getValue() ||
      typeof headersOption.getValue() !== 'string'
    ) {
      console.error('Invalid headers: must be a string');
      return false;
    }
    try {
      const headers = JSON.parse(headersOption.getValue());
      for (const [key, value] of Object.entries(headers)) {
        if (
          typeof key !== 'string' ||
          typeof value !== 'string' ||
          key.trim() === '' ||
          value.trim() === ''
        ) {
          console.error('Invalid header:', key, value);
          return false;
        }
      }
    } catch (error) {
      console.log(error);
      console.error('Invalid JSON for headers');
      return false;
    }
  }

  const parameters = options.find(
    option => option.getKey() === 'tool.parameters',
  );
  const value = parameters?.getValue();
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

  return true;
};
