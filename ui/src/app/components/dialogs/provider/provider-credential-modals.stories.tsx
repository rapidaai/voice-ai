import type { Meta, StoryObj } from '@storybook/react-vite';
import { useState } from 'react';
import type { VaultCredential } from '@rapidaai/react';
import type { RapidaProvider } from '@/providers';
import {
  CredentialKeyValueField,
  ProviderCredentialForm,
} from './create-provider-credential-modal';
import { ProviderCredentialList } from './view-provider-credential-modal';

const provider: RapidaProvider = {
  code: 'openai',
  name: 'OpenAI',
  featureList: ['llm'],
  image: '/providers/openai.svg',
  configurations: [
    { name: 'api_key', type: 'password', label: 'API key' },
    {
      name: 'base_url',
      type: 'text',
      label: 'Base URL (optional)',
      required: false,
    },
    {
      name: 'mode',
      type: 'select',
      label: 'Mode',
      choices: [
        { label: 'Chat', value: 'chat' },
        { label: 'Responses', value: 'responses' },
      ],
    },
    { name: 'headers', type: 'key_value', label: 'Headers', required: false },
  ],
};

const credential = {
  getCreateddate: () => '2026-09-19T10:00:00Z',
  getId: () => 'credential-openai-prod',
  getLastuseddate: () => undefined,
  getName: () => 'Production key',
  getProvider: () => 'openai',
} as unknown as VaultCredential;

const meta = {
  title: 'Dialogs/Provider/Credentials',
  tags: ['autodocs'],
  parameters: {
    docs: {
      description: {
        component:
          'Provider credential dialog building blocks for creating and viewing provider keys.',
      },
    },
  },
} satisfies Meta;

export default meta;

type Story = StoryObj;

export const CreateForm: Story = {
  render: function Render() {
    const [config, setConfig] = useState<Record<string, string>>({
      api_key: '',
      mode: '',
    });
    const [keyName, setKeyName] = useState('Production key');
    const [selectedProvider, setSelectedProvider] =
      useState<RapidaProvider | null>(provider);

    return (
      <div className="max-w-xl border border-border-subtle bg-layer text-foreground">
        <ProviderCredentialForm
          config={config}
          error=""
          isConfigFieldRequired={field => field.required !== false}
          keyName={keyName}
          keyNameInputId="storybook-provider-key-name"
          onCancel={() => undefined}
          onConfigChange={(name, value) =>
            setConfig(prev => ({ ...prev, [name]: value }))
          }
          onKeyNameChange={setKeyName}
          onProviderChange={setSelectedProvider}
          onSubmit={() => undefined}
          provider={selectedProvider}
          providers={[provider]}
        />
      </div>
    );
  },
};

export const KeyValueField: Story = {
  render: function Render() {
    const [value, setValue] = useState('{"Authorization":"Bearer token"}');

    return (
      <div className="max-w-xl bg-layer p-4 text-foreground">
        <CredentialKeyValueField
          name="headers"
          label="Headers"
          value={value}
          onChange={setValue}
        />
      </div>
    );
  },
};

export const CredentialList: Story = {
  render: () => (
    <div className="max-w-xl border border-border-subtle bg-layer p-4 text-foreground">
      <ProviderCredentialList
        credentials={[credential]}
        currentProvider={provider}
        onDelete={() => undefined}
        onSetupCredential={() => undefined}
      />
    </div>
  ),
};

export const EmptyCredentialList: Story = {
  render: () => (
    <div className="max-w-xl border border-border-subtle bg-layer text-foreground">
      <ProviderCredentialList
        credentials={[]}
        currentProvider={provider}
        onDelete={() => undefined}
        onSetupCredential={() => undefined}
      />
    </div>
  ),
};
