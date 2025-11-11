import { Metadata, VaultCredential } from '@rapidaai/react';
import { Dropdown } from '@/app/components/Dropdown';
import { FormLabel } from '@/app/components/form-label';
import { FieldSet } from '@/app/components/Form/Fieldset';
import {
  ConfigureAwsStorage,
  ValidateAwsStorageOptions,
} from '@/app/components/providers/storage/aws';
import {
  ConfigureAzureStorage,
  ValidateAzureStorageOptions,
} from '@/app/components/providers/storage/azure';
import {
  ConfigureGoogleStorage,
  ValidateGoogleCloudStorageOptions,
} from '@/app/components/providers/storage/google';
import { cn } from '@/styles/media';
import { ProviderConfig, STORAGE_PROVIDER } from '@/app/components/providers';
import { CredentialDropdown } from '@/app/components/Dropdown/credential-dropdown';
import { useCallback } from 'react';

export const ValidateStorageOptions = (
  provider: string,
  parameters: Metadata[],
): boolean => {
  switch (provider) {
    case 'azure-cloud':
      return ValidateAzureStorageOptions(parameters);
    case 'google-cloud':
      return ValidateGoogleCloudStorageOptions(parameters);
    case 'aws-cloud':
      return ValidateAwsStorageOptions(parameters);
    default:
      return false;
  }
};
export const CloudStorageProvider: React.FC<{
  onChangeConfig: (config: ProviderConfig | null) => void;
  config: ProviderConfig | null;
}> = ({ onChangeConfig, config }) => {
  const updateConfig = (newConfig: Partial<ProviderConfig>) => {
    onChangeConfig({ ...config, ...newConfig } as ProviderConfig);
  };

  const getParamValue = useCallback(
    (key: string) => {
      if (config)
        return (
          config.parameters?.find(p => p.getKey() === key)?.getValue() ?? ''
        );
    },
    [config],
  );

  const updateParameter = (key: string, value: string) => {
    if (config) {
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
    }
  };

  const renderConfigComponent = () => {
    switch (config?.provider) {
      case 'azure-cloud':
        return (
          <ConfigureAzureStorage
            parameters={config?.parameters || []}
            onParameterChange={(params: Metadata[]) =>
              updateConfig({ parameters: params })
            }
          />
        );
      case 'google-cloud':
        return (
          <ConfigureGoogleStorage
            parameters={config?.parameters || []}
            onParameterChange={(params: Metadata[]) =>
              updateConfig({ parameters: params })
            }
          />
        );

      case 'aws-cloud':
        return (
          <ConfigureAwsStorage
            parameters={config?.parameters || []}
            onParameterChange={(params: Metadata[]) =>
              updateConfig({ parameters: params })
            }
          />
        );

      default:
        return null;
    }
  };

  console.log(config);
  return (
    <div className={cn('px-6 pb-6 pt-2 flex gap-8 pl-8')}>
      <div className="space-y-6 w-full max-w-6xl">
        <FieldSet className="relative col-span-1">
          <FormLabel>Provider</FormLabel>
          <Dropdown
            className="bg-light-background max-w-full dark:bg-gray-950"
            currentValue={STORAGE_PROVIDER.find(
              x => x.code === config?.provider,
            )}
            setValue={v => {
              updateConfig({
                provider: v.code,
                providerId: v.id,
              });
            }}
            allValue={STORAGE_PROVIDER}
            placeholder="Select storage provide"
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
        </FieldSet>
        {config && (
          <CredentialDropdown
            className="bg-light-background max-w-full dark:bg-gray-950"
            onChangeCredential={(c: VaultCredential) => {
              updateParameter('rapida.credential_id', c.getId());
            }}
            currentCredential={getParamValue('rapida.credential_id')}
            providerId={config.providerId}
          />
        )}
        <div className="grid grid-cols-3 gap-x-6 gap-y-3">
          {renderConfigComponent()}
        </div>
      </div>
    </div>
  );
};
