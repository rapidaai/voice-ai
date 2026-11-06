import { Metadata } from '@rapidaai/react';
import { loadProviderConfig } from '@/providers/config-loader';
import {
  getDefaultsFromConfig,
  validateFromConfig,
} from '@/providers/config-defaults';
import { ConfigRenderer } from '@/app/components/domain/providers/config-renderer';
import { useCallback, useMemo } from 'react';
import { CredentialDropdown } from '@/app/components/domain/dropdowns/credential-dropdown';
import { TEXT_PROVIDERS } from '@/providers';
import { MapTextProviderModelSelection } from './text/model-mapping';
import { Dropdown, Stack } from '@carbon/react';
import { HelpToggletip } from '@/app/components/domain/providers/help-label';
import type { VaultCredential } from '@rapidaai/react';
import type {
  ProviderComponentProps,
  ProviderSelectionChange,
} from '@/app/components/domain/providers/provider-component-props';
import type { RapidaProvider } from '@/providers';

export const GetDefaultTextProviderConfigIfInvalid = (
  provider: string,
  parameters: Metadata[],
): Metadata[] => {
  const config = loadProviderConfig(provider);
  if (!config?.text) return parameters;
  const mappedParameters = MapTextProviderModelSelection(provider, parameters);
  return getDefaultsFromConfig(config, 'text', mappedParameters, provider);
};

export const GetDefaultTextProviderConfigOnProviderSwitch = (
  provider: string,
  parameters: Metadata[],
): Metadata[] => {
  const resetParameters = parameters.filter(
    p =>
      p.getKey() !== 'rapida.credential_id' && !p.getKey().startsWith('model.'),
  );

  return GetDefaultTextProviderConfigIfInvalid(provider, resetParameters);
};

export const ValidateTextProviderDefaultOptions = (
  provider: string,
  parameters: Metadata[],
  providerCredentialIds?: string[],
): string | undefined => {
  const config = loadProviderConfig(provider);
  if (!config?.text) return 'Please select a valid model and provider.';
  const mappedParameters = MapTextProviderModelSelection(provider, parameters);
  const validationError = validateFromConfig(
    config,
    'text',
    provider,
    mappedParameters,
  );
  if (validationError) return validationError;

  if (!providerCredentialIds) return undefined;
  const credentialID = mappedParameters
    .find(opt => opt.getKey() === 'rapida.credential_id')
    ?.getValue();
  if (!credentialID) return `Please provide a valid ${provider} credential.`;
  if (!providerCredentialIds.includes(credentialID))
    return `Please select a valid ${provider} credential.`;
  return undefined;
};

const getProviderName = (item: RapidaProvider | null): string =>
  item?.name ?? '';

function TextProviderModelPlaceholder() {
  return (
    <div className="flex-1 min-w-0">
      <Dropdown
        id="text-model-placeholder"
        titleText="Model"
        hideLabel
        label="Select model"
        size="md"
        items={[]}
        itemToString={() => ''}
        disabled
      />
    </div>
  );
}

export function TextProviderConfigComponent({
  provider,
  parameters,
  onChangeParameter,
}: ProviderComponentProps) {
  const config = loadProviderConfig(provider);
  if (!config?.text) return null;
  return (
    <ConfigRenderer
      provider={provider}
      category="text"
      config={config.text}
      parameters={parameters}
      onParameterChange={onChangeParameter}
    />
  );
}

export function TextProvider({
  provider,
  parameters,
  onChangeProvider,
  onChangeParameter,
}: ProviderComponentProps) {
  const textProviders = useMemo(
    () => TEXT_PROVIDERS.filter(p => Boolean(loadProviderConfig(p.code)?.text)),
    [],
  );

  const getParamValue = useCallback(
    (key: string) => parameters.find(p => p.getKey() === key)?.getValue() ?? '',
    [parameters],
  );

  const updateParameter = (key: string, value: string) => {
    const updatedParams = [...parameters];
    const existingIndex = updatedParams.findIndex(p => p.getKey() === key);
    const newParam = new Metadata();
    newParam.setKey(key);
    newParam.setValue(value);
    if (existingIndex >= 0) {
      updatedParams[existingIndex] = newParam;
    } else {
      updatedParams.push(newParam);
    }
    onChangeParameter(updatedParams);
  };

  const selectedProvider = textProviders.find(x => x.code === provider) || null;

  return (
    <>
      <Stack>
        <div className="w-full">
          <div className="cds--label text-provider-label-row !flex w-full items-center">
            <div className="w-48 shrink-0">
              <span className="inline-flex items-center gap-1">
                Model provider
                <HelpToggletip
                  label="Model provider"
                  helpText="Select the provider and model configuration used by this agent."
                />
              </span>
            </div>
            <div className="flex-1 min-w-0">
              <span>Model</span>
            </div>
          </div>
          <div className="text-provider-combo-row flex w-full items-stretch border-b border-b-border-strong bg-field">
            <div className="w-48 shrink-0 border-r border-border-subtle">
              <Dropdown
                id="text-provider"
                titleText="Model provider"
                hideLabel
                label="Select provider"
                size="md"
                items={textProviders}
                selectedItem={selectedProvider}
                itemToString={getProviderName}
                onChange={({
                  selectedItem,
                }: ProviderSelectionChange<RapidaProvider>) => {
                  if (selectedItem) onChangeProvider(selectedItem.code);
                }}
              />
            </div>
            {provider ? (
              <TextProviderConfigComponent
                parameters={parameters}
                provider={provider}
                onChangeParameter={onChangeParameter}
                onChangeProvider={onChangeProvider}
              />
            ) : (
              <TextProviderModelPlaceholder />
            )}
          </div>
        </div>
      </Stack>
      {provider && (
        <CredentialDropdown
          onChangeCredential={(c: VaultCredential) => {
            updateParameter('rapida.credential_id', c.getId());
          }}
          provider={provider}
          currentCredential={getParamValue('rapida.credential_id')}
        />
      )}
    </>
  );
}
