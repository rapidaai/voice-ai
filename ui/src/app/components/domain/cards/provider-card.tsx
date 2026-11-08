import {
  BaseCard,
  CardDescription,
  CardTitle,
} from '@/app/components/ui/primitives';
import { useAllProviderCredentials } from '@/hooks/use-model';
import { cn } from '@/utils';
import type { HTMLAttributes } from 'react';
import { memo, useMemo, useState } from 'react';
import { CreateProviderCredentialDialog } from '@/app/components/dialogs/provider';
import { ViewProviderCredentialDialog } from '@/app/components/dialogs/provider';
import { IntegrationProvider } from '@/providers';
import { Launch } from '@carbon/icons-react';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { OverflowMenu, OverflowMenuItem } from '@/app/components/ui/primitives';
import { PrimaryButton, GhostButton } from '@/app/components/ui/primitives';
import IconIndicator from '@carbon/react/es/components/IconIndicator';

interface ProviderCardProps extends HTMLAttributes<HTMLDivElement> {
  provider: IntegrationProvider;
}

function ProviderCardComponent({ provider, className }: ProviderCardProps) {
  const { goTo } = useGlobalNavigation();
  const { providerCredentials } = useAllProviderCredentials();
  const [createProviderModalOpen, setCreateProviderModalOpen] = useState(false);
  const [viewProviderModalOpen, setViewProviderModalOpen] = useState(false);
  const [menuOpen, setMenuOpen] = useState(false);

  const isConnected = useMemo(
    () =>
      providerCredentials.some(
        credential => credential.getProvider() === provider.code,
      ),
    [provider.code, providerCredentials],
  );

  return (
    <>
      <CreateProviderCredentialDialog
        modalOpen={createProviderModalOpen}
        setModalOpen={setCreateProviderModalOpen}
        currentProvider={provider.code}
      />
      <ViewProviderCredentialDialog
        modalOpen={viewProviderModalOpen}
        setModalOpen={setViewProviderModalOpen}
        currentProvider={provider}
        onSetupCredential={() => {
          setViewProviderModalOpen(false);
          setCreateProviderModalOpen(true);
        }}
      />
      <BaseCard className={cn('p-4 md:p-5', className)} data-id={provider.code}>
        <header className="flex items-start justify-between">
          <div className="flex items-center justify-center shrink-0 h-10 w-10 border border-border-subtle bg-layer">
            <img
              src={provider.image}
              alt={provider.name}
              className="h-8 w-8 object-contain"
            />
          </div>
          <OverflowMenu
            size="sm"
            flipped
            iconDescription="Actions"
            open={menuOpen}
            onOpen={() => setMenuOpen(true)}
            onClose={() => setMenuOpen(false)}
          >
            <OverflowMenuItem
              itemText="Create a credential"
              onClick={() => {
                setMenuOpen(false);
                setCreateProviderModalOpen(true);
              }}
            />
            <OverflowMenuItem
              itemText="View credential"
              onClick={() => {
                setMenuOpen(false);
                setViewProviderModalOpen(true);
              }}
            />
          </OverflowMenu>
        </header>

        <div className="mt-3 flex-1">
          <CardTitle>{provider.name}</CardTitle>
          <CardDescription>{provider.description}</CardDescription>
        </div>

        <div className="flex items-center justify-between mt-4">
          <div>
            {isConnected && (
              <IconIndicator kind="succeeded" label="Connected" size={16} />
            )}
          </div>
          <div className="flex items-center gap-1">
            {!isConnected && (
              <PrimaryButton
                size="sm"
                className="invisible group-hover:visible"
                onClick={() => setCreateProviderModalOpen(true)}
              >
                Setup Credential
              </PrimaryButton>
            )}
            {provider.url && (
              <GhostButton
                size="sm"
                hasIconOnly
                renderIcon={Launch}
                iconDescription="Open provider"
                onClick={() => goTo(provider.url!)}
              />
            )}
          </div>
        </div>
      </BaseCard>
    </>
  );
}

export const ProviderCard = memo(ProviderCardComponent);
