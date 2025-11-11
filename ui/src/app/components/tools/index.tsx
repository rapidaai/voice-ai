import { Metadata } from '@rapidaai/react';
import { Dropdown } from '@/app/components/Dropdown';
import { ConfigureAPIRequest } from '@/app/components/tools/api-request';
import {
  APIRequestToolDefintion,
  GetAPIRequestDefaultOptions,
  ValidateAPIRequestDefaultOptions,
} from '@/app/components/tools/api-request/constant';
import { ConfigureEndOfConversation } from '@/app/components/tools/end-of-conversation';
import {
  EndOfConverstaionToolDefintion,
  GetEndOfConversationDefaultOptions,
  ValidateEndOfConversationDefaultOptions,
} from '@/app/components/tools/end-of-conversation/constant';
import { ConfigureEndpoint } from '@/app/components/tools/endpoint';
import {
  EndpointToolDefintion,
  GetEndpointDefaultOptions,
  ValidateEndpointDefaultOptions,
} from '@/app/components/tools/endpoint/constant';
import { ConfigureKnowledgeRetrieval } from '@/app/components/tools/knowledge-retrieval';
import {
  GetKnowledgeRetrievalDefaultOptions,
  KnowledgeRetrievalToolDefintion,
  ValidateKnowledgeRetrievalDefaultOptions,
} from '@/app/components/tools/knowledge-retrieval/constant';
import { ConfigurePutOnHold } from '@/app/components/tools/put-on-hold';
import {
  GetPutOnHoldDefaultOptions,
  PutOnHoldToolDefintion,
  ValidatePutOnHoldDefaultOptions,
} from '@/app/components/tools/put-on-hold/constant';
import { cn } from '@/styles/media';
import { FC } from 'react';
import { FieldSet } from '@/app/components/Form/Fieldset';
import { FormLabel } from '@/app/components/form-label';

export const BUILDIN_TOOLS = [
  {
    icon: 'https://cdn-01.rapida.ai/partners/tools/knowledge-retrieval.webp',
    code: 'knowledge_retrieval',
    name: 'Knowledge Retrieval',
  },
  //   {
  //     icon: 'https://cdn-01.rapida.ai/partners/tools/web_search.png',
  //     code: 'web_search',
  //     name: 'Web Search',
  //   },
  {
    icon: 'https://cdn-01.rapida.ai/partners/tools/api_call.png',
    code: 'api_request',
    name: 'API request',
  },
  {
    icon: 'https://cdn-01.rapida.ai/partners/tools/api_call.png',
    code: 'endpoint',
    name: 'Endpoint (LLM Call)',
  },
  {
    icon: 'https://cdn-01.rapida.ai/partners/tools/waiting.png',
    code: 'put_on_hold',
    name: 'Put on hold',
  },
  {
    icon: 'https://cdn-01.rapida.ai/partners/tools/stop.png',
    code: 'end_of_conversation',
    name: 'End of conversation',
  },

  // Add more tools as needed
];

export interface BuildinToolConfig {
  code: string;
  parameters: Metadata[];
}

export const GetDefaultToolDefintion = (
  code: string,
  existing: { name: string; description: string; parameters: string },
): { name: string; description: string; parameters: string } => {
  const isValidDefinition =
    existing && existing.name && existing.description && existing.parameters;

  switch (code) {
    case 'knowledge_retrieval':
      return isValidDefinition ? existing : KnowledgeRetrievalToolDefintion;
    case 'api_request':
      return isValidDefinition ? existing : APIRequestToolDefintion;
    case 'endpoint':
      return isValidDefinition ? existing : EndpointToolDefintion;
    case 'put_on_hold':
      return isValidDefinition ? existing : PutOnHoldToolDefintion;
    case 'end_of_conversation':
      return isValidDefinition ? existing : EndOfConverstaionToolDefintion;
    default:
      return isValidDefinition ? existing : EndpointToolDefintion; // Fallback for default case
  }
};

export const GetDefaultToolConfigIfInvalid = (
  code: string,
  parameters: Metadata[],
): Metadata[] => {
  switch (code) {
    case 'knowledge_retrieval':
      return GetKnowledgeRetrievalDefaultOptions(parameters);
    case 'api_request':
      return GetAPIRequestDefaultOptions(parameters);
    case 'endpoint':
      return GetEndpointDefaultOptions(parameters);
    case 'put_on_hold':
      return GetPutOnHoldDefaultOptions(parameters);
    case 'end_of_conversation':
      return GetEndOfConversationDefaultOptions(parameters);
    default:
      return parameters;
  }
};

export const ValidateToolDefaultOptions = (
  code: string,
  parameters: Metadata[],
): boolean => {
  switch (code) {
    case 'knowledge_retrieval':
      return ValidateKnowledgeRetrievalDefaultOptions(parameters);
    case 'api_request':
      return ValidateAPIRequestDefaultOptions(parameters);
    case 'endpoint':
      return ValidateEndpointDefaultOptions(parameters);
    case 'put_on_hold':
      return ValidatePutOnHoldDefaultOptions(parameters);
    case 'end_of_conversation':
      return ValidateEndOfConversationDefaultOptions(parameters);
    default:
      return false;
  }
};

const ConfigureBuildinTool: FC<{
  toolDefinition: {
    name: string;
    description: string;
    parameters: string;
  };
  onChangeToolDefinition: (vl: {
    name: string;
    description: string;
    parameters: string;
  }) => void;
  config: BuildinToolConfig;
  updateConfig: (config: Partial<BuildinToolConfig>) => void;
  inputClass?: string;
}> = ({
  config,
  updateConfig,
  inputClass,
  toolDefinition,
  onChangeToolDefinition,
}) => {
  switch (config.code) {
    case 'knowledge_retrieval':
      return (
        <ConfigureKnowledgeRetrieval
          toolDefinition={toolDefinition}
          onChangeToolDefinition={onChangeToolDefinition}
          parameters={config.parameters}
          inputClass={inputClass}
          onParameterChange={(params: Metadata[]) =>
            updateConfig({ parameters: params })
          }
        />
      );
    case 'api_request':
      return (
        <ConfigureAPIRequest
          toolDefinition={toolDefinition}
          onChangeToolDefinition={onChangeToolDefinition}
          parameters={config.parameters}
          inputClass={inputClass}
          onParameterChange={(params: Metadata[]) =>
            updateConfig({ parameters: params })
          }
        />
      );
    case 'endpoint':
      return (
        <ConfigureEndpoint
          toolDefinition={toolDefinition}
          onChangeToolDefinition={onChangeToolDefinition}
          parameters={config.parameters}
          inputClass={inputClass}
          onParameterChange={(params: Metadata[]) =>
            updateConfig({ parameters: params })
          }
        />
      );
    case 'put_on_hold':
      return (
        <ConfigurePutOnHold
          toolDefinition={toolDefinition}
          onChangeToolDefinition={onChangeToolDefinition}
          parameters={config.parameters}
          inputClass={inputClass}
          onParameterChange={(params: Metadata[]) =>
            updateConfig({ parameters: params })
          }
        />
      );
    case 'end_of_conversation':
      return (
        <ConfigureEndOfConversation
          toolDefinition={toolDefinition}
          onChangeToolDefinition={onChangeToolDefinition}
          parameters={config.parameters}
          inputClass={inputClass}
          onParameterChange={(params: Metadata[]) =>
            updateConfig({ parameters: params })
          }
        />
      );
    default:
      return null;
  }
};

export const BuildinTool: React.FC<{
  toolDefinition: {
    name: string;
    description: string;
    parameters: string;
  };
  onChangeToolDefinition: (vl: {
    name: string;
    description: string;
    parameters: string;
  }) => void;
  onChangeBuildinTool: (i: string) => void;
  onChangeConfig: (config: BuildinToolConfig) => void;
  inputClass?: string;
  config: BuildinToolConfig;
}> = ({
  toolDefinition,
  onChangeToolDefinition,
  onChangeBuildinTool,
  onChangeConfig,
  config,
  inputClass,
}) => {
  const updateConfig = (newConfig: Partial<BuildinToolConfig>) => {
    onChangeConfig({ ...config, ...newConfig } as BuildinToolConfig);
  };

  return (
    <>
      <div className="p-6">
        <FieldSet>
          <FormLabel>Action</FormLabel>
          <Dropdown
            className={cn('bg-light-background dark:bg-gray-950', inputClass)}
            currentValue={BUILDIN_TOOLS.find(x => x.code === config.code)}
            setValue={v => {
              onChangeBuildinTool(v.code);
            }}
            allValue={BUILDIN_TOOLS}
            placeholder="Select provider"
            option={c => {
              return (
                <span className="inline-flex items-center gap-2 sm:gap-2.5 max-w-full text-sm font-medium">
                  <img
                    alt=""
                    loading="lazy"
                    width={16}
                    height={16}
                    className="sm:h-4 sm:w-4 w-4 h-4 align-middle block shrink-0"
                    src={c.icon}
                  />
                  <span className="truncate capitalize">{c.name}</span>
                </span>
              );
            }}
            label={c => {
              return (
                <span className="inline-flex items-center gap-2 sm:gap-2.5 max-w-full text-sm font-medium">
                  <img
                    alt=""
                    loading="lazy"
                    width={16}
                    height={16}
                    className="sm:h-4 sm:w-4 w-4 h-4 align-middle block shrink-0"
                    src={c.icon}
                  />
                  <span className="truncate capitalize">{c.name}</span>
                </span>
              );
            }}
          />
        </FieldSet>
      </div>

      <ConfigureBuildinTool
        toolDefinition={GetDefaultToolDefintion(config.code, toolDefinition)}
        onChangeToolDefinition={onChangeToolDefinition}
        config={config}
        updateConfig={updateConfig}
        inputClass={inputClass}
      />
    </>
  );
};
