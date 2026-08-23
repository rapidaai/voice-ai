import React, { FC, useCallback, useEffect, useState } from 'react';
import { ConnectionConfig } from '@rapidaai/react';
import { useCurrentCredential } from '@/hooks/use-credential';
import { DeleteProviderKey } from '@rapidaai/react';
import { GetCredentialResponse, VaultCredential } from '@rapidaai/react';
import { useRapidaStore } from '@/hooks';
import toast from 'react-hot-toast/headless';
import { ModalProps } from '@/app/components/base/modal';
import { useAllProviderCredentials } from '@/hooks/use-model';
import { useProviderContext } from '@/context/provider-context';
import { toHumanReadableRelativeTime } from '@/utils/date';
import { ServiceError } from '@rapidaai/react';
import { connectionConfig } from '@/configs';
import { RapidaProvider } from '@/providers';
import {
  Modal,
  ModalHeader,
  ModalBody,
  ModalFooter,
} from '@/app/components/carbon/modal';
import {
  PrimaryButton,
  TertiaryButton,
  DangerButton,
} from '@/app/components/carbon/button';
import { Stack } from '@/app/components/carbon/form';
import { TrashCan } from '@carbon/icons-react';
import { CopyButton } from '@/app/components/carbon/button/copy-button';

interface ViewProviderCredentialDialogProps extends ModalProps {
  currentProvider: RapidaProvider;
  onSetupCredential: () => void;
}

export const ViewProviderCredentialDialog: FC<
  ViewProviderCredentialDialogProps
> = props => {
  const { authId, projectId, token } = useCurrentCredential();
  const { showLoader, hideLoader } = useRapidaStore();
  const { providerCredentials } = useAllProviderCredentials();
  const providerCtx = useProviderContext();
  const [currentProviderCredentials, setCurrentProviderCredentials] = useState<
    Array<VaultCredential>
  >([]);

  useEffect(() => {
    setCurrentProviderCredentials(
      providerCredentials.filter(
        y => y.getProvider() === props.currentProvider.code,
      ),
    );
  }, [providerCredentials, props.currentProvider]);

  const afterCredentialDelete = useCallback(
    (err: ServiceError | null, gapcr: GetCredentialResponse | null) => {
      hideLoader();
      if (gapcr?.getSuccess()) {
        providerCtx.reloadProviderCredentials();
      } else {
        let errorMessage = gapcr?.getError();
        if (errorMessage) {
          toast.error(errorMessage.getHumanmessage());
        } else
          toast.error(
            'Unable to process your request. please try again later.',
          );
        return;
      }
    },
    [],
  );

  const onDelete = (credId: string) => {
    showLoader();
    DeleteProviderKey(
      connectionConfig,
      credId,
      afterCredentialDelete,
      ConnectionConfig.WithDebugger({
        authorization: token,
        userId: authId,
        projectId: projectId,
      }),
    );
  };

  return (
    <Modal
      open={props.modalOpen}
      onClose={() => props.setModalOpen(false)}
      size="sm"
    >
      <ModalHeader
        label="Credentials"
        title="View provider credential"
        onClose={() => props.setModalOpen(false)}
      />
      <ModalBody>
        {currentProviderCredentials.length > 0 ? (
          <Stack gap={4}>
            {currentProviderCredentials.map(x => (
              <div
                className="group border border-border-subtle bg-layer"
                key={x.getId()}
              >
                <div className="flex items-center px-4 py-3">
                  <div className="border border-border-subtle bg-surface flex items-center justify-center shrink-0 h-10 w-10 p-1 mr-3">
                    <img
                      src={props.currentProvider.image}
                      alt={props.currentProvider.name}
                    />
                  </div>
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-semibold capitalize truncate">
                      {x.getName()}
                    </p>
                    <div className="flex gap-2 text-xs text-muted">
                      <span>
                        Updated{' '}
                        {x.getCreateddate() &&
                          toHumanReadableRelativeTime(x.getCreateddate()!)}
                      </span>
                      <span>·</span>
                      <span>
                        Last activity{' '}
                        {x.getLastuseddate()
                          ? toHumanReadableRelativeTime(x.getLastuseddate()!)
                          : 'No activity'}
                      </span>
                    </div>
                  </div>
                  <DangerButton
                    size="sm"
                    renderIcon={TrashCan}
                    className="opacity-0 group-hover:opacity-100 transition-opacity"
                    onClick={() => onDelete(x.getId())}
                  >
                    Delete
                  </DangerButton>
                </div>
                <div className="flex items-center gap-3 border-t border-border-subtle px-4 py-2">
                  <span className="shrink-0 text-xs font-medium text-muted">
                    Credential ID
                  </span>
                  <code
                    className="min-w-0 flex-1 truncate text-xs text-foreground"
                    title={x.getId()}
                  >
                    {x.getId()}
                  </code>
                  <CopyButton
                    className="h-7 w-7 shrink-0"
                    copyDescription="Copy credential ID"
                    copiedDescription="Credential ID copied"
                  >
                    {x.getId()}
                  </CopyButton>
                </div>
              </div>
            ))}
          </Stack>
        ) : (
          <div className="px-4 py-8 flex flex-col items-center text-center">
            <p className="text-sm font-semibold mb-1">No Credential</p>
            <p className="text-sm text-muted mb-4">
              No provider credential to display
            </p>
            <TertiaryButton size="sm" onClick={() => props.onSetupCredential()}>
              Setup Credential
            </TertiaryButton>
          </div>
        )}
      </ModalBody>
      <ModalFooter>
        <PrimaryButton size="lg" onClick={() => props.setModalOpen(false)}>
          Got it
        </PrimaryButton>
      </ModalFooter>
    </Modal>
  );
};
