import type { VaultCredential } from '@rapidaai/react';
import { Add, Information, Renew } from '@carbon/icons-react';
import { useMemo, useState } from 'react';
import { CreateProviderCredentialDialog } from '@/app/components/dialogs/provider';
import { useAllProviderCredentials } from '@/hooks/use-model';
import { useProviderContext } from '@/context/provider-context';
import { allProvider } from '@/providers';
import {
  Button,
  Dropdown,
  Toggletip,
  ToggletipButton,
  ToggletipContent,
} from '@carbon/react';
import { cn } from '@/utils';

interface CredentialDropdownProps {
  className?: string;
  provider?: string;
  currentCredential?: string;
  onChangeCredential: (credential: VaultCredential) => void;
}

interface CredentialDropdownViewProps extends CredentialDropdownProps {
  credentials: VaultCredential[];
  getProviderName: (provider: string) => string | undefined;
  onCreateCredential: () => void;
  onReloadCredentials: () => void;
}

export function CredentialDropdownView({
  className,
  credentials,
  currentCredential,
  getProviderName,
  onChangeCredential,
  onCreateCredential,
  onReloadCredentials,
}: CredentialDropdownViewProps) {
  const selectedItem =
    credentials.find(credential => credential.getId() === currentCredential) ||
    null;

  const formatCredential = (credential: VaultCredential | null) => {
    if (!credential) {
      return '';
    }

    const providerName = getProviderName(credential.getProvider());
    return providerName
      ? `${providerName} / ${credential.getName()}`
      : credential.getName();
  };

  return (
    <div className={cn(className)}>
      <div className="cds--label domain-connected-dropdown-label !inline-flex items-center gap-1">
        Credential
        <Toggletip align="right">
          <ToggletipButton label="Show credential information">
            <Information size={14} />
          </ToggletipButton>
          <ToggletipContent>
            Select the saved provider credential used for model access.
          </ToggletipContent>
        </Toggletip>
      </div>
      <div className="domain-connected-dropdown-row flex w-full items-stretch border-b border-b-border-strong bg-field">
        <div className="min-w-0 flex-1">
          <Dropdown
            id="credential-dropdown"
            titleText="Credential"
            hideLabel
            label="Select credential"
            items={credentials}
            selectedItem={selectedItem}
            itemToString={formatCredential}
            onChange={({ selectedItem }) => {
              if (selectedItem) {
                onChangeCredential(selectedItem);
              }
            }}
          />
        </div>
        <Button
          hasIconOnly
          renderIcon={Renew}
          iconDescription="Refresh credentials"
          kind="ghost"
          size="md"
          onClick={onReloadCredentials}
          className="domain-connected-dropdown-action shrink-0"
        />
        <Button
          hasIconOnly
          renderIcon={Add}
          iconDescription="Create credential"
          kind="ghost"
          size="md"
          onClick={onCreateCredential}
          className="domain-connected-dropdown-action shrink-0"
        />
      </div>
    </div>
  );
}

export function CredentialDropdown({
  className,
  provider,
  currentCredential,
  onChangeCredential,
}: CredentialDropdownProps) {
  const { providerCredentials } = useAllProviderCredentials();
  const ctx = useProviderContext();
  const [createProviderModalOpen, setCreateProviderModalOpen] = useState(false);

  const currentProviderCredentials = useMemo(
    () =>
      providerCredentials.filter(
        credential => credential.getProvider() === provider,
      ),
    [provider, providerCredentials],
  );

  const providerNames = useMemo(
    () =>
      new Map(
        allProvider().map(currentProvider => [
          currentProvider.code,
          currentProvider.name,
        ]),
      ),
    [],
  );

  const getProviderName = (code: string) => providerNames.get(code);

  return (
    <>
      <CreateProviderCredentialDialog
        modalOpen={createProviderModalOpen}
        setModalOpen={setCreateProviderModalOpen}
        currentProvider={provider}
      />
      <CredentialDropdownView
        className={className}
        provider={provider}
        currentCredential={currentCredential}
        credentials={currentProviderCredentials}
        getProviderName={getProviderName}
        onChangeCredential={onChangeCredential}
        onReloadCredentials={ctx.reloadProviderCredentials}
        onCreateCredential={() => setCreateProviderModalOpen(true)}
      />
    </>
  );
}
