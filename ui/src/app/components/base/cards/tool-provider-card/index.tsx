import { ToolProvider } from '@rapidaai/react';
import { Card, CardDescription, CardTitle } from '@/app/components/base/cards';

import {
  BlueBorderButton,
  IBlueBGButton,
  ILinkButton,
} from '@/app/components/Form/Button';
import { ReloadIcon } from '@/app/components/Icon/Reload';
import { CreateToolProviderCredentialDialog } from '@/app/components/base/modal/connect-tool-provider-credential-modal';
import { WEB_API } from '@/configs';
import { cn } from '@/styles/media';
import { FC, HTMLAttributes, memo, useState } from 'react';
import { StatusIndicator } from '@/app/components/indicators/status';

//
export interface ToolProviderConnectParams {
  linker: 'organization' | 'user';
  linkerId: string;
  redirectTo: string;
}
//

interface ToolProviderCardProps extends HTMLAttributes<HTMLDivElement> {
  toolProvider: ToolProvider;
  isConnected: boolean;
  toolConnectParams?: ToolProviderConnectParams;
}

export const ToolProviderCard: FC<ToolProviderCardProps> = memo(
  ({ toolProvider, toolConnectParams, className, isConnected = false }) => {
    return (
      <Card
        className={cn(
          'relative shadow-sm group flex flex-col rounded-[2px]',
          className,
          toolProvider.getId(),
        )}
      >
        <header className="flex justify-between">
          <div className="rounded-[2px] flex items-center justify-center shrink-0 h-10 w-10 dark:bg-gray-600/50 border dark:border-gray-700">
            <img
              src={toolProvider.getImage()}
              alt={toolProvider.getName()}
              className="rounded-[2px] p-1"
            />
          </div>
        </header>
        <div className="mt-4 flex-1">
          <CardTitle>{toolProvider.getName()} </CardTitle>
          <CardDescription>{toolProvider?.getDescription()}</CardDescription>
          <div className="flex mt-4 gap-2">
            {isConnected && <StatusIndicator state={'Complete'} />}
            {isConnected && toolConnectParams && (
              <ToolConnectIconButton
                toolProvider={toolProvider}
                {...toolConnectParams}
              />
            )}
            {!isConnected && toolConnectParams && (
              <ToolProviderConnectButton
                toolProvider={toolProvider}
                {...toolConnectParams}
              />
            )}
          </div>
        </div>
      </Card>
    );
  },
);

const ToolProviderConnectButton: FC<
  {
    toolProvider: ToolProvider;
  } & ToolProviderConnectParams
> = ({ toolProvider, linker, linkerId, redirectTo }) => {
  const [open, setOpen] = useState(false);

  if (
    toolProvider.getConnectconfigurationMap().get('connect_type') === 'oauth2'
  ) {
    //
    const connectUrl = toolProvider
      .getConnectconfigurationMap()
      .get('connect_url');
    // Constructing query parameters
    const queryParams = new URLSearchParams({
      link: linker,
      link_id: linkerId,
      redirect_to: redirectTo,
      tool_id: toolProvider.getId(),
    });

    return (
      <ILinkButton
        className="h-8 invisible group-hover:visible rounded-[2px]"
        target="_blank"
        href={`${WEB_API}${connectUrl}?${queryParams.toString()}`}
      >
        Connect {toolProvider.getName()}
      </ILinkButton>
    );
  }

  if (
    toolProvider.getConnectconfigurationMap().get('connect_type') === 'form'
  ) {
    return (
      <>
        <CreateToolProviderCredentialDialog
          toolProvider={toolProvider}
          modalOpen={open}
          setModalOpen={setOpen}
        />
        <IBlueBGButton
          className="h-8 invisible group-hover:visible rounded-[2px]"
          onClick={() => {
            setOpen(true);
          }}
        >
          Connect
        </IBlueBGButton>
      </>
    );
  }
  return <></>;
};

const ToolConnectIconButton: FC<
  {
    toolProvider: ToolProvider;
  } & ToolProviderConnectParams
> = ({ toolProvider, linker, linkerId, redirectTo }) => {
  const [open, setOpen] = useState(false);

  if (
    toolProvider.getConnectconfigurationMap().get('connect_type') === 'oauth2'
  ) {
    //
    const connectUrl = toolProvider
      .getConnectconfigurationMap()
      .get('connect_url');
    // Constructing query parameters
    const queryParams = new URLSearchParams({
      link: linker,
      link_id: linkerId,
      redirect_to: redirectTo,
      tool_id: toolProvider.getId(),
    });

    return (
      <ILinkButton
        className="h-7 w-7 p-1 text-sm font-medium border invisible group-hover:visible text-gray-500!"
        target="_blank"
        href={`${WEB_API}${connectUrl}?${queryParams.toString()}`}
      >
        <ReloadIcon className="w-4 h-4" />
      </ILinkButton>
    );
  }

  if (
    toolProvider.getConnectconfigurationMap().get('connect_type') === 'form'
  ) {
    return (
      <>
        <CreateToolProviderCredentialDialog
          toolProvider={toolProvider}
          modalOpen={open}
          setModalOpen={setOpen}
        />
        <BlueBorderButton
          className="h-8 text-sm font-medium border invisible group-hover:visible"
          onClick={() => {
            setOpen(true);
          }}
        >
          Connect
        </BlueBorderButton>
      </>
    );
  }
  return <></>;
};
