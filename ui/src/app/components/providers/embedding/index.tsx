import { Metadata, VaultCredential } from '@rapidaai/react';
import { Dropdown } from '@/app/components/Dropdown';
import {
  EMBEDDING_PROVIDERS,
  ProviderConfig,
} from '@/app/components/providers';
import { ConfigureCohereEmbeddingModel } from '@/app/components/providers/embedding/cohere';
import {
  GetCohereEmbeddingDefaultOptions,
  ValidateCohereEmbeddingDefaultOptions,
} from '@/app/components/providers/embedding/cohere/constants';
import { ConfigureGoogleEmbeddingModel } from '@/app/components/providers/embedding/google';
import {
  GetGoogleEmbeddingDefaultOptions,
  ValidateGoogleEmbeddingDefaultOptions,
} from '@/app/components/providers/embedding/google/constants';
import { ConfigureOpenaiEmbeddingModel } from '@/app/components/providers/embedding/openai';
import {
  GetOpenaiEmbeddingDefaultOptions,
  ValidateOpenaiEmbeddingDefaultOptions,
} from '@/app/components/providers/embedding/openai/constants';
import { ConfigureVoyageEmbeddingModel } from '@/app/components/providers/embedding/voyageai';
import {
  GetVoyageEmbeddingDefaultOptions,
  ValidateVoyageEmbeddingDefaultOptions,
} from '@/app/components/providers/embedding/voyageai/constants';
import { cn } from '@/styles/media';
import { FC, useCallback } from 'react';
import { CredentialDropdown } from '@/app/components/Dropdown/credential-dropdown';
import { FieldSet } from '@/app/components/Form/Fieldset';
import { FormLabel } from '@/app/components/form-label';

export const GetDefaultEmbeddingConfigIfInvalid = (
  provider: string,
  parameters: Metadata[],
): Metadata[] => {
  switch (provider) {
    case 'cohere':
      return GetCohereEmbeddingDefaultOptions(parameters);
    case 'openai':
      return GetOpenaiEmbeddingDefaultOptions(parameters);
    case 'google':
      return GetGoogleEmbeddingDefaultOptions(parameters);
    case 'voyageai':
      return GetVoyageEmbeddingDefaultOptions(parameters);
    default:
      return parameters;
  }
};

export const ValidateEmbeddingDefaultOptions = (
  provider: string,
  parameters: Metadata[],
): string | undefined => {
  switch (provider) {
    case 'cohere':
      return ValidateCohereEmbeddingDefaultOptions(parameters);
    case 'openai':
      return ValidateOpenaiEmbeddingDefaultOptions(parameters);
    case 'google':
      return ValidateGoogleEmbeddingDefaultOptions(parameters);
    case 'voyageai':
      return ValidateVoyageEmbeddingDefaultOptions(parameters);
    default:
      return 'Please select a valid provider and model for embedding';
  }
};

export const EmbeddingConfigComponent: FC<{
  config: ProviderConfig;
  updateConfig: (config: Partial<ProviderConfig>) => void;
}> = ({ config, updateConfig }) => {
  switch (config.provider) {
    case 'cohere':
      return (
        <ConfigureCohereEmbeddingModel
          parameters={config.parameters}
          onParameterChange={(params: Metadata[]) =>
            updateConfig({ parameters: params })
          }
        />
      );
    case 'openai':
      return (
        <ConfigureOpenaiEmbeddingModel
          parameters={config.parameters}
          onParameterChange={(params: Metadata[]) =>
            updateConfig({ parameters: params })
          }
        />
      );
    case 'voyageai':
      return (
        <ConfigureVoyageEmbeddingModel
          parameters={config.parameters}
          onParameterChange={(params: Metadata[]) =>
            updateConfig({ parameters: params })
          }
        />
      );
    case 'google':
      return (
        <ConfigureGoogleEmbeddingModel
          parameters={config.parameters}
          onParameterChange={(params: Metadata[]) =>
            updateConfig({ parameters: params })
          }
        />
      );
    default:
      return null;
  }
};

export const EmbeddingProvider: React.FC<{
  onChangeProvider: (i: string, v: string) => void;
  onChangeConfig: (config: ProviderConfig) => void;
  config: ProviderConfig;
}> = ({ onChangeProvider, onChangeConfig, config }) => {
  const updateConfig = (newConfig: Partial<ProviderConfig>) => {
    onChangeConfig({ ...config, ...newConfig } as ProviderConfig);
  };

  const getParamValue = useCallback(
    (key: string) => {
      return config.parameters?.find(p => p.getKey() === key)?.getValue() ?? '';
    },
    [config.parameters],
  );

  const updateParameter = (key: string, value: string) => {
    const updatedParams = [...(config.parameters || [])];
    const existingIndex = updatedParams.findIndex(p => p.getKey() === key);
    const newParam = new Metadata();
    newParam.setKey(key);
    newParam.setValue(value);
    if (existingIndex >= 0) {
      updatedParams[existingIndex] = newParam;
    } else {
      updatedParams.push(newParam);
    }
    updateConfig({ parameters: updatedParams });
  };
  return (
    <>
      <FieldSet>
        <FormLabel>Provider Model</FormLabel>
        <div
          className={cn(
            'p-px',
            'outline-solid outline-transparent',
            'focus-within:outline-blue-600 focus:outline-blue-600 -outline-offset-1',
            'border-b border-gray-400 dark:border-gray-600',
            'dark:focus-within:border-blue-600 focus-within:border-blue-600',
            'transition-all duration-200 ease-in-out',
            'flex relative',
          )}
        >
          <div className="w-44 relative">
            <Dropdown
              className="bg-white max-w-full dark:bg-gray-950 focus-within:border-none! focus-within:outline-hidden! border-none! outline-hidden"
              currentValue={EMBEDDING_PROVIDERS.find(
                x => x.id === config.providerId,
              )}
              setValue={v => {
                onChangeProvider(v.id, v.code);
              }}
              allValue={EMBEDDING_PROVIDERS}
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
                      src={c.image}
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
                      src={c.image}
                    />
                    <span className="truncate capitalize">{c.name}</span>
                  </span>
                );
              }}
            />
          </div>

          <EmbeddingConfigComponent
            config={config}
            updateConfig={updateConfig}
          />
        </div>
      </FieldSet>
      {config.providerId && (
        <CredentialDropdown
          className="bg-white"
          onChangeCredential={(c: VaultCredential) => {
            updateParameter('rapida.credential_id', c.getId());
          }}
          currentCredential={getParamValue('rapida.credential_id')}
          providerId={config.providerId}
        />
      )}
    </>
  );
};
