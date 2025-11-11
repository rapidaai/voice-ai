import { Metadata, VaultCredential } from '@rapidaai/react';
import { Dropdown } from '@/app/components/Dropdown';
import { ProviderConfig, TEXT_PROVIDERS } from '@/app/components/providers';
import { ConfigureAnthropicTextProviderModel } from '@/app/components/providers/text/anthropic';
import {
  GetAnthropicTextProviderDefaultOptions,
  ValidateAnthropicTextProviderDefaultOptions,
} from '@/app/components/providers/text/anthropic/constants';
import {
  ConfigureAzureTextProviderModel,
  GetAzureTextProviderDefaultOptions,
  ValidateAzureTextProviderDefaultOptions,
} from '@/app/components/providers/text/azure';
import { ConfigureCohereTextProviderModel } from '@/app/components/providers/text/cohere';
import {
  GetCohereTextProviderDefaultOptions,
  ValidateCohereTextProviderDefaultOptions,
} from '@/app/components/providers/text/cohere/constants';
import { ConfigureGoogleTextProviderModel } from '@/app/components/providers/text/google';
import {
  GetGoogleTextProviderDefaultOptions,
  ValidateGoogleTextProviderDefaultOptions,
} from '@/app/components/providers/text/google/constants';
import { ConfigureOpenaiTextProviderModel } from '@/app/components/providers/text/openai';
import {
  GetOpenaiTextProviderDefaultOptions,
  ValidateOpenaiTextProviderDefaultOptions,
} from '@/app/components/providers/text/openai/constants';
import { cn } from '@/styles/media';
import { FC, useCallback } from 'react';
import { FieldSet } from '@/app/components/Form/Fieldset';
import { FormLabel } from '@/app/components/form-label';
import { CredentialDropdown } from '@/app/components/Dropdown/credential-dropdown';

export const GetDefaultTextProviderConfigIfInvalid = (
  provider: string,
  parameters: Metadata[],
): Metadata[] => {
  switch (provider) {
    case 'openai':
      return GetOpenaiTextProviderDefaultOptions(parameters);
    case 'azure-openai':
    case 'azure':
      return GetAzureTextProviderDefaultOptions(parameters);
    case 'google':
      return GetGoogleTextProviderDefaultOptions(parameters);
    case 'anthropic':
      return GetAnthropicTextProviderDefaultOptions(parameters);
    case 'cohere':
      return GetCohereTextProviderDefaultOptions(parameters);
    default:
      return parameters;
  }
};

export const ValidateTextProviderDefaultOptions = (
  provider: string,
  parameters: Metadata[],
): string | undefined => {
  switch (provider) {
    case 'openai':
      return ValidateOpenaiTextProviderDefaultOptions(parameters);
    case 'azure-openai':
    case 'azure':
      return ValidateAzureTextProviderDefaultOptions(parameters);
    case 'google':
      return ValidateGoogleTextProviderDefaultOptions(parameters);
    case 'anthropic':
      return ValidateAnthropicTextProviderDefaultOptions(parameters);
    case 'cohere':
      return ValidateCohereTextProviderDefaultOptions(parameters);
    default:
      return 'Please select a valid model and provider.';
  }
};

const TextProviderConfigComponent: FC<{
  config: ProviderConfig;
  updateConfig: (config: Partial<ProviderConfig>) => void;
}> = ({ config, updateConfig }) => {
  switch (config.provider) {
    case 'openai':
      return (
        <ConfigureOpenaiTextProviderModel
          parameters={config.parameters}
          onParameterChange={(params: Metadata[]) =>
            updateConfig({ parameters: params })
          }
        />
      );
    case 'azure':
    case 'azure-openai':
      return (
        <ConfigureAzureTextProviderModel
          parameters={config.parameters}
          onParameterChange={(params: Metadata[]) =>
            updateConfig({ parameters: params })
          }
        />
      );
    case 'google':
      return (
        <ConfigureGoogleTextProviderModel
          parameters={config.parameters}
          onParameterChange={(params: Metadata[]) =>
            updateConfig({ parameters: params })
          }
        />
      );
    case 'anthropic':
      return (
        <ConfigureAnthropicTextProviderModel
          parameters={config.parameters}
          onParameterChange={(params: Metadata[]) =>
            updateConfig({ parameters: params })
          }
        />
      );
    case 'cohere':
      return (
        <ConfigureCohereTextProviderModel
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

export const TextProvider: React.FC<{
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
            'outline-solid outline-transparent',
            'focus-within:outline-blue-600 focus:outline-blue-600 -outline-offset-1',
            'border-b border-gray-400 dark:border-gray-600',
            'dark:focus-within:border-blue-600 focus-within:border-blue-600',
            'transition-all duration-200 ease-in-out',
            'flex relative',
            'pt-px pl-px',
          )}
        >
          <div className="w-44 relative">
            <Dropdown
              className="bg-white max-w-full dark:bg-gray-950 focus-within:border-none! outline-none! border-none! outline-hidden"
              currentValue={TEXT_PROVIDERS.find(
                x => x.id === config.providerId,
              )}
              setValue={v => {
                onChangeProvider(v.id, v.code);
              }}
              allValue={TEXT_PROVIDERS}
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
          {/*  */}
          <TextProviderConfigComponent
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
