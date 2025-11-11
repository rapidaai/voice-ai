import React, { FC, useCallback, useEffect, useState } from 'react';
import { ConnectionConfig, Provider } from '@rapidaai/react';
import { useCurrentCredential } from '@/hooks/use-credential';
import { DeleteProviderKey } from '@rapidaai/react';
import { GetCredentialResponse, VaultCredential } from '@rapidaai/react';
import { useRapidaStore } from '@/hooks';
import toast from 'react-hot-toast/headless';
import { GenericModal, ModalProps } from '@/app/components/base/modal';
import { useAllProviderCredentials } from '@/hooks/use-model';

import {
  BlueBorderButton,
  IBlueBGButton,
  IRedBGButton,
} from '@/app/components/Form/Button';
import { useProviderContext } from '@/context/provider-context';
import { toHumanReadableRelativeTime } from '@/styles/media';
import { DeleteIcon } from '@/app/components/Icon/delete';
import { ServiceError } from '@rapidaai/react';
import { ModalFooter } from '@/app/components/base/modal/modal-footer';
import { ModalBody } from '@/app/components/base/modal/modal-body';
import { ModalFitHeightBlock } from '@/app/components/blocks/modal-fit-height-block';
import { ModalHeader } from '@/app/components/base/modal/modal-header';
import { ModalTitleBlock } from '@/app/components/blocks/modal-title-block';
import { connectionConfig } from '@/configs';
import { RapidaProvider } from '@/app/components/providers';

/**
 * creation provider key dialog props that gives ability for opening and closing modal props
 */
interface ViewProviderCredentialDialogProps extends ModalProps {
  /**
   * exiting provider if there
   */
  currentProvider: RapidaProvider;

  /**
   *
   * @param p
   * @returns
   */
  onSetupCredential: () => void;
}
/**
 *
 * to create a provider key for given model
 * @param props
 * @returns
 */
export const ViewProviderCredentialDialog: FC<
  ViewProviderCredentialDialogProps
> = props => {
  /**
   *current provider
   */
  const { authId, projectId, token } = useCurrentCredential();

  /**
   *
   */
  const { showLoader, hideLoader } = useRapidaStore();
  const { providerCredentials } = useAllProviderCredentials();
  const providerCtx = useProviderContext();
  const [currentProviderCredentials, setCurrentProviderCredentials] = useState<
    Array<VaultCredential>
  >([]);

  useEffect(() => {
    setCurrentProviderCredentials(
      providerCredentials.filter(
        y => y.getVaulttypeid() === props.currentProvider.id,
      ),
    );
  }, [providerCredentials, props.currentProvider]);
  /**
   * When a credetials mark as delete
   * @param err
   * @param gapcr
   * @returns
   */
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

  const onDelete = credId => {
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
  /**
   *
   */
  return (
    <GenericModal modalOpen={props.modalOpen} setModalOpen={props.setModalOpen}>
      <ModalFitHeightBlock>
        <ModalHeader
          onClose={() => {
            props.setModalOpen(false);
          }}
        >
          <ModalTitleBlock>View provider credential</ModalTitleBlock>
        </ModalHeader>

        <ModalBody className="space-y-2">
          {currentProviderCredentials.length > 0 ? (
            <>
              {currentProviderCredentials.map((x, idx) => {
                return (
                  <div
                    className="group mb-2 border-[0.5px] bg-gray-50 dark:bg-slate-900 rounded-[2px] dark:border-slate-700"
                    key={idx}
                  >
                    <div className="flex items-center px-3 py-[9px]">
                      <div className="border dark:border-slate-700 rounded-[2px] flex items-center justify-center shrink-0 h-10 w-10 p-1 mr-3">
                        <img
                          src={props.currentProvider.image}
                          alt={props.currentProvider.name}
                          className="rounded-[2px]"
                        />
                      </div>
                      <div className="grow">
                        <div className="flex items-center h-5">
                          <div className="text-sm font-semibold capitalize">
                            {x.getName()}
                          </div>
                        </div>

                        <div className="flex space-x-1.5">
                          <div className="flex whitespace-nowrap text-xs space-x-1 items-center">
                            <span className="font-semibold">Updated</span>
                            <span>
                              {x.getCreateddate() &&
                                toHumanReadableRelativeTime(
                                  x.getCreateddate()!,
                                )}
                            </span>
                          </div>
                          <p>•</p>
                          <div className="flex whitespace-nowrap text-xs space-x-1 items-center">
                            <span className="font-semibold">Last activity</span>
                            <span>
                              {x.getLastuseddate()
                                ? toHumanReadableRelativeTime(
                                    x.getLastuseddate()!,
                                  )
                                : 'No activity'}
                            </span>
                          </div>
                        </div>
                      </div>
                      <IRedBGButton
                        className="h-7 text-sm invisible group-hover:visible"
                        onClick={() => {
                          onDelete(x.getId());
                        }}
                      >
                        Delete <DeleteIcon className="w-3.5 h-3.5 ml-2" />
                      </IRedBGButton>
                    </div>
                  </div>
                );
              })}
            </>
          ) : (
            <div className="px-4 py-6 flex flex-col justify-center items-center">
              <div className="font-semibold">No Credential</div>
              <div>No provider credential to display</div>
              <BlueBorderButton
                onClick={() => props.onSetupCredential()}
                className="mt-3 h-8 text-sm font-medium border"
              >
                Setup Credential
              </BlueBorderButton>
            </div>
          )}
        </ModalBody>

        <ModalFooter>
          <IBlueBGButton
            className="px-4 rounded-[2px]"
            type="button"
            onClick={() => {
              props.setModalOpen(false);
            }}
          >
            Got it
          </IBlueBGButton>
        </ModalFooter>
      </ModalFitHeightBlock>
    </GenericModal>
  );
};
