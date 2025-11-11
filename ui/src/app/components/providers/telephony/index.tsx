import { Metadata, VaultCredential } from '@rapidaai/react';
import { ProviderConfig, TELEPHONY_PROVIDER } from '@/app/components/providers';
import {
  ConfigureExotelTelephony,
  ValidateExotelTelephonyOptions,
} from '@/app/components/providers/telephony/exotel';
import {
  ConfigureTwilioTelephony,
  ValidateTwilioTelephonyOptions,
} from '@/app/components/providers/telephony/twilio';
import {
  ConfigureVonageTelephony,
  ValidateVonageTelephonyOptions,
} from '@/app/components/providers/telephony/vonage';
import { Dropdown } from '@/app/components/Dropdown';
import { FormLabel } from '@/app/components/form-label';
import { FieldSet } from '@/app/components/Form/Fieldset';
import { InputGroup } from '@/app/components/input-group';
import { InputHelper } from '@/app/components/input-helper';
import { cn } from '@/styles/media';
import { CredentialDropdown } from '@/app/components/Dropdown/credential-dropdown';
import { useCallback } from 'react';

export const ValidateTelephonyOptions = (
  provider: string,
  parameters: Metadata[],
): boolean => {
  switch (provider) {
    case 'vonage':
      return ValidateVonageTelephonyOptions(parameters);
    case 'twilio':
      return ValidateTwilioTelephonyOptions(parameters);
    case 'exotel':
      return ValidateExotelTelephonyOptions(parameters);
    default:
      return false;
  }
};

/**
 *
 * @param param0
 * @returns
 */

export const ConfigureTelephonyComponent: React.FC<{
  onConfigChange: (config: Partial<ProviderConfig>) => void;
  config: ProviderConfig | null;
}> = ({ onConfigChange, config }) => {
  switch (config?.provider) {
    case 'exotel':
      return (
        <ConfigureExotelTelephony
          parameters={config?.parameters || []}
          onParameterChange={(params: Metadata[]) =>
            onConfigChange({ parameters: params })
          }
        />
      );
    case 'vonage':
      return (
        <ConfigureVonageTelephony
          parameters={config?.parameters || []}
          onParameterChange={(params: Metadata[]) =>
            onConfigChange({ parameters: params })
          }
        />
      );

    case 'twilio':
      return (
        <ConfigureTwilioTelephony
          parameters={config?.parameters || []}
          onParameterChange={(params: Metadata[]) =>
            onConfigChange({ parameters: params })
          }
        />
      );

    default:
      return null;
  }
};

/**
 *
 * @param param0
 * @returns
 */

export const TelephonyProvider: React.FC<{
  onConfigChange: (config: ProviderConfig | null) => void;
  config: ProviderConfig | null;
  onChangeProvider: (providerId: string, providerName: string) => void;
}> = ({ onConfigChange, config, onChangeProvider }) => {
  //
  const updateConfig = (newConfig: Partial<ProviderConfig>) => {
    onConfigChange({ ...config, ...newConfig } as ProviderConfig);
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

  return (
    <InputGroup title="Telephony">
      <div className={cn('px-6 pb-6 pt-2 flex gap-8')}>
        <div className="flex flex-col space-y-6">
          <FieldSet>
            <FormLabel>Telephony provider</FormLabel>
            <Dropdown
              className="bg-light-background max-w-full dark:bg-gray-950"
              currentValue={TELEPHONY_PROVIDER.find(
                x => x.code === config?.provider,
              )}
              setValue={v => {
                onChangeProvider(v.id, v.code);
              }}
              allValue={TELEPHONY_PROVIDER}
              placeholder="Select telephony provider"
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
            <InputHelper>
              Choose a telephony provider to handle voice communication for your
              applications. Each provider offers different capabilities, pricing
              structures, and global coverage.
            </InputHelper>
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
            <ConfigureTelephonyComponent
              config={config}
              onConfigChange={updateConfig}
            />
          </div>
        </div>
      </div>
    </InputGroup>
  );
};
