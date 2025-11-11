import { Dropdown } from '@/app/components/Dropdown';
import { CredentialDropdown } from '@/app/components/Dropdown/credential-dropdown';
import { FormLabel } from '@/app/components/form-label';
import { FieldSet } from '@/app/components/Form/Fieldset';
import {
  ProviderConfig,
  TEXT_TO_SPEECH_PROVIDER,
} from '@/app/components/providers';
import { TextToSpeechConfigComponent } from '@/app/components/providers/text-to-speech/provider';
import { Metadata, VaultCredential } from '@rapidaai/react';
import { useCallback } from 'react';

export const TextToSpeechProvider: React.FC<{
  onChangeProvider: (i: string, v: string) => void;
  onChangeConfig: (config: ProviderConfig) => void;
  config: ProviderConfig | null;
}> = ({ onChangeProvider, onChangeConfig, config }) => {
  const updateConfig = (newConfig: Partial<ProviderConfig>) => {
    onChangeConfig({ ...config, ...newConfig } as ProviderConfig);
  };

  const getParamValue = useCallback(
    (key: string) => {
      return (
        config?.parameters?.find(p => p.getKey() === key)?.getValue() ?? ''
      );
    },
    [config?.parameters],
  );

  const updateParameter = (key: string, value: string) => {
    const updatedParams = [...(config?.parameters || [])];
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
    <div className="space-y-6 w-full max-w-6xl">
      <FieldSet className="relative">
        <FormLabel>Provider</FormLabel>
        <Dropdown
          className="bg-light-background max-w-full dark:bg-gray-950"
          currentValue={TEXT_TO_SPEECH_PROVIDER.find(
            x => x.code === config?.provider,
          )}
          setValue={v => {
            onChangeProvider(v.id, v.code);
          }}
          allValue={TEXT_TO_SPEECH_PROVIDER}
          placeholder="Select voice ouput provider"
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
      {config?.providerId && (
        <CredentialDropdown
          className="bg-light-background max-w-full dark:bg-gray-950"
          onChangeCredential={(c: VaultCredential) => {
            updateParameter('rapida.credential_id', c.getId());
          }}
          currentCredential={getParamValue('rapida.credential_id')}
          providerId={config?.providerId}
        />
      )}
      {config && (
        <div className="grid grid-cols-3 gap-x-6 gap-y-3">
          <TextToSpeechConfigComponent
            config={config}
            updateConfig={updateConfig}
          />
        </div>
      )}
    </div>
  );
};
