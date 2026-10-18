import { useEffect, useId, useState } from 'react';
import { useCurrentCredential } from '@/hooks/use-credential';
import { ErrorMessage } from '@/app/components/ui/feedback';
import { useRapidaStore } from '@/stores/app';
import toast from 'react-hot-toast/headless';
import type { ModalProps } from '@/app/components/ui/primitives';
import {
  Modal,
  ModalHeader,
  ModalBody,
  ModalFooter,
} from '@/app/components/ui/primitives';
import {
  PrimaryButton,
  SecondaryButton,
  TertiaryButton,
} from '@/app/components/ui/primitives';
import { Stack, TextInput, TextArea } from '@/app/components/ui/primitives';
import {
  Dropdown,
  Button,
  Select as CarbonSelect,
  SelectItem,
} from '@carbon/react';
import { Add, TrashCan } from '@carbon/icons-react';
import { useProviderContext } from '@/context/provider-context';
import { INTEGRATION_PROVIDER } from '@/providers';
import type { RapidaProvider } from '@/providers';
import { createPortal } from 'react-dom';
import { createProviderCredential } from '@/clients';
import {
  TableForm,
  TableFormCell,
  TableFormHeaderCell,
  TableFormRow,
} from '@/app/components/ui/table';

interface CreateProviderCredentialDialogProps extends ModalProps {
  currentProvider?: string | null;
}

type ProviderConfiguration = NonNullable<
  RapidaProvider['configurations']
>[number];

const fallbackError = 'Unable to process your request. Please try again later.';

export function CreateProviderCredentialDialog({
  modalOpen,
  setModalOpen,
  currentProvider,
}: CreateProviderCredentialDialogProps) {
  const keyNameInputId = useId();
  const { authId, projectId, token } = useCurrentCredential();
  const [provider, setProvider] = useState<RapidaProvider | null>(null);
  const providerCtx = useProviderContext();
  const { loading, showLoader, hideLoader } = useRapidaStore();
  const [error, setError] = useState('');
  const [keyName, setKeyName] = useState('');
  const [config, setConfig] = useState<Record<string, string>>({});

  useEffect(() => {
    setProvider(
      INTEGRATION_PROVIDER.slice()
        .reverse()
        .find(x => x.code === currentProvider) || null,
    );
  }, [currentProvider]);

  const handleConfigChange = (name: string, value: string) => {
    setConfig(prev => ({ ...prev, [name]: value }));
  };

  const isConfigFieldRequired = (field: ProviderConfiguration) => {
    if (field.required === false) return false;
    if (field.label?.toLowerCase().includes('(optional)')) return false;
    if (provider?.code === 'sip' && field.name === 'sip_headers') return false;
    return true;
  };

  const validateAndSubmit = () => {
    if (!provider) {
      setError('Please select the provider which you want to create the key.');
      return;
    }
    if (!keyName.trim()) {
      setError('Please provide a valid key name for the credential.');
      return;
    }
    const missingFields = provider.configurations?.filter(
      configOption =>
        isConfigFieldRequired(configOption) &&
        !config[configOption.name]?.trim(),
    );
    if (missingFields && missingFields.length > 0) {
      setError(
        `Please fill out the following fields: ${missingFields
          .map(field => field.label)
          .join(', ')}`,
      );
      return;
    }

    showLoader();
    createProviderCredential({
      provider: provider.code,
      name: keyName,
      config,
      auth: { projectId, token, userId: authId },
    })
      .then(cpkr => {
        hideLoader();
        if (cpkr?.getSuccess()) {
          toast.success(
            'Provider credential have been successfully added to the vault.',
          );
          providerCtx.reloadProviderCredentials();
          setModalOpen(false);
          setError('');
          setKeyName('');
          setConfig({});
        } else {
          const errorMessage = cpkr?.getError();
          setError(errorMessage?.getHumanmessage() ?? fallbackError);
        }
      })
      .catch(() => {
        hideLoader();
        toast.error(
          'Unable to create provider credential, please try again later.',
        );
      });
  };

  const modalContent = (
    <Modal
      open={modalOpen}
      onClose={() => setModalOpen(false)}
      size="sm"
      className="!z-999999"
      containerClassName="!z-999999"
      selectorPrimaryFocus={`[id="${keyNameInputId}"]`}
      preventCloseOnClickOutside
    >
      <ModalHeader
        label="Credentials"
        title="Create provider credential"
        onClose={() => setModalOpen(false)}
      />
      <ProviderCredentialForm
        config={config}
        error={error}
        isConfigFieldRequired={isConfigFieldRequired}
        keyName={keyName}
        keyNameInputId={keyNameInputId}
        onCancel={() => setModalOpen(false)}
        onConfigChange={handleConfigChange}
        onKeyNameChange={setKeyName}
        onProviderChange={nextProvider => {
          setError('');
          setKeyName('');
          setConfig({});
          setProvider(nextProvider);
        }}
        onSubmit={validateAndSubmit}
        provider={provider}
        providers={INTEGRATION_PROVIDER}
        loading={loading}
      />
    </Modal>
  );

  if (typeof document === 'undefined') return modalContent;

  return createPortal(modalContent, document.body);
}

interface ProviderCredentialFormProps {
  config: Record<string, string>;
  error: string;
  isConfigFieldRequired: (field: ProviderConfiguration) => boolean;
  keyName: string;
  keyNameInputId: string;
  loading?: boolean;
  onCancel: () => void;
  onConfigChange: (name: string, value: string) => void;
  onKeyNameChange: (value: string) => void;
  onProviderChange: (provider: RapidaProvider | null) => void;
  onSubmit: () => void;
  provider: RapidaProvider | null;
  providers: RapidaProvider[];
}

export function ProviderCredentialForm({
  config,
  error,
  isConfigFieldRequired,
  keyName,
  keyNameInputId,
  loading,
  onCancel,
  onConfigChange,
  onKeyNameChange,
  onProviderChange,
  onSubmit,
  provider,
  providers,
}: ProviderCredentialFormProps) {
  return (
    <>
      <ModalBody hasForm>
        <Stack gap={6}>
          <Dropdown
            id="credential-provider"
            titleText="Select your provider"
            label="Select the provider"
            items={providers}
            selectedItem={provider}
            itemToString={(item: RapidaProvider | null) => item?.name || ''}
            onChange={({ selectedItem }) => {
              onProviderChange(selectedItem || null);
            }}
          />
          <TextInput
            id={keyNameInputId}
            labelText="Key Name"
            placeholder="Assign a unique name to this provider key"
            value={keyName}
            required
            onChange={e => onKeyNameChange(e.target.value)}
          />
          {provider?.configurations?.map((field, idx) =>
            field.type === 'text' ? (
              <TextArea
                key={idx}
                id={`config-${field.name}`}
                labelText={field.label}
                placeholder={field.label}
                value={config[field.name] || ''}
                required={isConfigFieldRequired(field)}
                onChange={e => onConfigChange(field.name, e.target.value)}
              />
            ) : field.type === 'key_value' ? (
              <CredentialKeyValueField
                key={idx}
                name={field.name}
                label={field.label}
                value={config[field.name] || ''}
                onChange={value => onConfigChange(field.name, value)}
              />
            ) : field.type === 'select' ? (
              <CarbonSelect
                key={idx}
                id={`config-${field.name}`}
                labelText={field.label}
                value={config[field.name] || ''}
                onChange={e =>
                  onConfigChange(
                    field.name,
                    (e.target as HTMLSelectElement).value,
                  )
                }
              >
                <SelectItem
                  value=""
                  text={`Select ${field.label.toLowerCase()}`}
                />
                {(field.choices ?? []).map(choice => (
                  <SelectItem
                    key={choice.value}
                    value={choice.value}
                    text={choice.label}
                  />
                ))}
              </CarbonSelect>
            ) : (
              <TextInput
                key={idx}
                id={`config-${field.name}`}
                labelText={field.label}
                placeholder={field.label}
                value={config[field.name] || ''}
                required={isConfigFieldRequired(field)}
                onChange={e => onConfigChange(field.name, e.target.value)}
              />
            ),
          )}
          <ErrorMessage message={error} />
        </Stack>
      </ModalBody>
      <ModalFooter>
        <SecondaryButton size="lg" onClick={onCancel}>
          Cancel
        </SecondaryButton>
        <PrimaryButton size="lg" onClick={onSubmit} isLoading={loading}>
          Configure
        </PrimaryButton>
      </ModalFooter>
    </>
  );
}

interface CredentialEntry {
  key: string;
  value: string;
}

export function parseCredentialEntries(raw: string): CredentialEntry[] {
  if (!raw) return [];
  try {
    const value = JSON.parse(raw);
    if (!value || typeof value !== 'object' || Array.isArray(value)) return [];
    return Object.entries(value).map(([key, entryValue]) => ({
      key,
      value: String(entryValue),
    }));
  } catch {
    return [];
  }
}

export function serializeCredentialEntries(entries: CredentialEntry[]): string {
  const value: Record<string, string> = {};
  for (const entry of entries) {
    if (entry.key) value[entry.key] = entry.value;
  }
  return Object.keys(value).length > 0 ? JSON.stringify(value) : '';
}

export function CredentialKeyValueField({
  name,
  label,
  value,
  onChange,
}: {
  name: string;
  label: string;
  value: string;
  onChange: (value: string) => void;
}) {
  const [entries, setEntries] = useState<CredentialEntry[]>(() =>
    parseCredentialEntries(value),
  );

  useEffect(() => {
    setEntries(parseCredentialEntries(value));
  }, [value]);

  const syncEntries = (next: CredentialEntry[]) => {
    setEntries(next);
    onChange(serializeCredentialEntries(next));
  };

  const updateEntry = (index: number, field: 'key' | 'value', val: string) => {
    const next = [...entries];
    next[index] = { ...next[index], [field]: val };
    syncEntries(next);
  };

  const removeEntry = (index: number) => {
    syncEntries(entries.filter((_, i) => i !== index));
  };

  const addEntry = () => {
    setEntries(prev => [...prev, { key: '', value: '' }]);
  };

  return (
    <div className="flex flex-col gap-4">
      <p className="text-[10px] font-semibold tracking-[0.12em] uppercase text-muted">
        {label} ({entries.length})
      </p>
      <TableForm>
        <thead>
          <tr className="bg-[var(--cds-layer-accent-01)]">
            <TableFormHeaderCell className="w-1/2">Key</TableFormHeaderCell>
            <TableFormHeaderCell className="w-1/2">Value</TableFormHeaderCell>
            <TableFormHeaderCell className="w-10" divider={false} />
          </tr>
        </thead>
        <tbody>
          {entries.length === 0 && (
            <TableFormRow>
              <td colSpan={3} className="px-4 py-3 text-xs text-muted">
                No entries yet. Click <strong>Add {label.toLowerCase()}</strong>{' '}
                below to add key-value pairs.
              </td>
            </TableFormRow>
          )}
          {entries.map((entry, index) => (
            <TableFormRow key={index}>
              <TableFormCell>
                <TextInput
                  id={`kv-key-${name}-${index}`}
                  labelText=""
                  hideLabel
                  value={entry.key}
                  onChange={e => updateEntry(index, 'key', e.target.value)}
                  placeholder="Key"
                  size="md"
                />
              </TableFormCell>
              <TableFormCell>
                <TextInput
                  id={`kv-val-${name}-${index}`}
                  labelText=""
                  hideLabel
                  value={entry.value}
                  onChange={e => updateEntry(index, 'value', e.target.value)}
                  placeholder="Value"
                  size="md"
                />
              </TableFormCell>
              <TableFormCell className="w-10 text-center" divider={false}>
                <Button
                  hasIconOnly
                  renderIcon={TrashCan}
                  iconDescription="Remove"
                  kind="ghost"
                  size="sm"
                  className="!text-[var(--cds-support-error)]"
                  onClick={() => removeEntry(index)}
                />
              </TableFormCell>
            </TableFormRow>
          ))}
        </tbody>
      </TableForm>
      <TertiaryButton
        size="md"
        renderIcon={Add}
        onClick={addEntry}
        className="!w-full !max-w-none"
      >
        Add {label.toLowerCase()}
      </TertiaryButton>
    </div>
  );
}
