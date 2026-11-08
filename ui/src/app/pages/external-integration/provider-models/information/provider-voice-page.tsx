import { Helmet } from '@/app/components/app-shell/helmet';
import { CreateProviderCredentialDialog } from '@/app/components/dialogs/provider';
import { ViewProviderCredentialDialog } from '@/app/components/dialogs/provider';
import { PageHeader } from '@/app/components/layout/page-header';
import {
  GhostButton,
  PrimaryButton,
  Text,
} from '@/app/components/ui/primitives';
import {
  VoiceCatalog,
  VoiceCatalogItem,
} from '@/app/pages/external-integration/provider-models/information/voice-catalog';
import { useAllProviderCredentials } from '@/hooks/use-model';
import { RapidaProvider } from '@/providers';
import { Add, ModelAlt } from '@carbon/icons-react';
import { Tag } from '@carbon/react';
import { useMemo, useState } from 'react';

export function ProviderVoicePage(props: {
  provider?: RapidaProvider;
  voices: VoiceCatalogItem[];
}) {
  const { provider, voices } = props;
  const { providerCredentials } = useAllProviderCredentials();
  const [createProviderModalOpen, setCreateProviderModalOpen] = useState(false);
  const [viewProviderModalOpen, setViewProviderModalOpen] = useState(false);
  const [imageFailed, setImageFailed] = useState(false);
  const connected = useMemo(
    () =>
      providerCredentials.some(
        credential => credential.getProvider() === provider?.code,
      ),
    [provider?.code, providerCredentials],
  );

  if (!provider) return null;

  return (
    <div
      data-page-scroll
      className="flex min-h-0 flex-1 flex-col overflow-y-auto overflow-x-hidden"
    >
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
      <Helmet title={`${provider.name} voices`} />
      <PageHeader
        eyebrow={
          <Text variant="label-01" className="text-primary">
            Integrations
          </Text>
        }
        title={provider.name}
        description={provider.description}
        titleStart={
          <div className="flex h-12 w-12 shrink-0 items-center justify-center border border-border-subtle bg-layer">
            {provider.image && !imageFailed ? (
              <img
                src={provider.image}
                alt={`${provider.name} logo`}
                className="max-h-10 max-w-10 object-contain"
                onError={() => setImageFailed(true)}
              />
            ) : (
              <ModelAlt
                aria-label={`${provider.name} logo unavailable`}
                data-testid="provider-logo-fallback"
                size={24}
              />
            )}
          </div>
        }
        contextualActions={
          <Tag type={connected ? 'green' : 'gray'} size="sm">
            {connected ? 'Connected' : 'Setup required'}
          </Tag>
        }
      />
      <VoiceCatalog
        voices={voices}
        actions={
          <div className="flex h-full shrink-0 items-stretch">
            {connected && (
              <GhostButton
                className="h-full!"
                size="lg"
                onClick={() => setViewProviderModalOpen(true)}
              >
                Manage credentials
              </GhostButton>
            )}
            <PrimaryButton
              className="h-full!"
              size="lg"
              renderIcon={Add}
              onClick={() => setCreateProviderModalOpen(true)}
            >
              Add new credential
            </PrimaryButton>
          </div>
        }
      />
    </div>
  );
}
