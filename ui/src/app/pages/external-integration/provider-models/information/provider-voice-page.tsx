import { Helmet } from '@/app/components/app-shell/helmet';
import { CreateProviderCredentialDialog } from '@/app/components/dialogs/provider';
import { ViewProviderCredentialDialog } from '@/app/components/dialogs/provider';
import { PageHeaderBlock } from '@/app/components/layout/blocks/page-header-block';
import { PageTitleBlock } from '@/app/components/layout/blocks/page-title-block';
import { GhostButton, PrimaryButton } from '@/app/components/ui/primitives';
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
    <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
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
      <PageHeaderBlock className="min-h-24 shrink-0">
        <div className="flex min-w-0 items-center gap-3">
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
          <PageTitleBlock className="min-w-0">
            <div className="flex items-center gap-2">
              <span>{provider.name}</span>
              <Tag type={connected ? 'green' : 'gray'} size="sm">
                {connected ? 'Connected' : 'Setup required'}
              </Tag>
            </div>
            <p className="mt-1 line-clamp-2 max-w-3xl text-sm text-muted">
              {provider.description}
            </p>
          </PageTitleBlock>
        </div>
      </PageHeaderBlock>
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
