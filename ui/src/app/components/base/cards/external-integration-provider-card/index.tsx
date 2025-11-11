import { Card, CardDescription, CardTitle } from '@/app/components/base/cards';
import { IBlueBGButton } from '@/app/components/Form/Button';
import { CardOptionMenu } from '@/app/components/Menu';
import { useAllProviderCredentials } from '@/hooks/use-model';
import { cn } from '@/styles/media';
import { FC, HTMLAttributes, memo, useEffect, useState } from 'react';
import { StatusIndicator } from '@/app/components/indicators/status';
import { IntegrationProvider } from '@/app/components/providers';
import { CreateProviderCredentialDialog } from '@/app/components/base/modal/create-provider-credential-modal';
import { ViewProviderCredentialDialog } from '@/app/components/base/modal/view-provider-credential-modal';

interface ExternalIntegrationProviderCardProps
  extends HTMLAttributes<HTMLDivElement> {
  provider: IntegrationProvider;
}

export const ExternalIntegrationProviderCard: FC<ExternalIntegrationProviderCardProps> =
  memo(({ provider, className }) => {
    const { providerCredentials } = useAllProviderCredentials();
    const [createProviderModalOpen, setCreateProviderModalOpen] =
      useState(false);
    const [viewProviderModalOpen, setViewProviderModalOpen] = useState(false);
    const [connected, setConnected] = useState(false);
    useEffect(() => {
      let isFoundCredential = providerCredentials.find(
        x => x.getVaulttypeid() === provider.id,
      );
      if (isFoundCredential) setConnected(true);
    }, [JSON.stringify(provider), JSON.stringify(providerCredentials)]);

    return (
      <>
        <CreateProviderCredentialDialog
          modalOpen={createProviderModalOpen}
          setModalOpen={setCreateProviderModalOpen}
          currentProviderId={provider.id}
        ></CreateProviderCredentialDialog>
        <ViewProviderCredentialDialog
          modalOpen={viewProviderModalOpen}
          setModalOpen={setViewProviderModalOpen}
          currentProvider={provider}
          onSetupCredential={() => {
            setViewProviderModalOpen(!viewProviderModalOpen);
            setCreateProviderModalOpen(!createProviderModalOpen);
          }}
        ></ViewProviderCredentialDialog>
        <Card
          className={cn('shadow-sm group flex flex-col rounded-[2px]', className)}
          data-id={provider.id}
        >
          <header className="flex justify-between">
            <div className="rounded-[2px] flex items-center justify-center shrink-0 h-9 w-9 dark:bg-gray-600 border dark:border-gray-700">
              <img
                src={provider.image}
                alt={provider.name}
                className="rounded-[2px]"
              />
            </div>
            <CardOptionMenu
              options={[
                {
                  option: 'Create a credential',
                  onActionClick: () => {
                    setCreateProviderModalOpen(!createProviderModalOpen);
                  },
                },
                {
                  option: 'View Credential',
                  onActionClick: () => {
                    setViewProviderModalOpen(!viewProviderModalOpen);
                  },
                },
              ]}
              classNames="h-8 rounded-[2px] opacity-60"
            />
          </header>
          <div className="mt-3 flex-1">
            <CardTitle>{provider.name} </CardTitle>
            <CardDescription>{provider.description}</CardDescription>
            <div className="flex mt-4 justify-between">
              <div>{connected && <StatusIndicator state={'Connected'} />}</div>
              {!connected && (
                <IBlueBGButton
                  className="h-8 text-sm rounded-[2px] invisible group-hover:visible"
                  onClick={() => {
                    setCreateProviderModalOpen(!createProviderModalOpen);
                  }}
                >
                  Setup Credential
                </IBlueBGButton>
              )}
            </div>
          </div>
        </Card>
      </>
    );
  });
