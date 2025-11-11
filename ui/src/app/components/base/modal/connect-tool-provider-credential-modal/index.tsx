import React, { FC, useCallback, useState } from 'react';
import { useForm } from 'react-hook-form';
import { useCredential } from '@/hooks/use-credential';
import { CreateToolCredential } from '@rapidaai/react';
import { GetCredentialResponse } from '@rapidaai/react';
import { Label } from '@/app/components/Form/Label';
import { Input } from '@/app/components/Form/Input';
import { cn } from '@/styles/media';
import { ErrorMessage } from '@/app/components/Form/error-message';
import { useRapidaStore } from '@/hooks';
import toast from 'react-hot-toast/headless';
import { ModalProps } from '@/app/components/base/modal';
import { FieldSet } from '@/app/components/Form/Fieldset';
import { useProviderContext } from '@/context/provider-context';
import { OutlineButton } from '../../../Form/Button';
import { ServiceError } from '@rapidaai/react';
import { ToolProvider } from '@rapidaai/react';
import { CenterModal } from '@/app/components/base/modal/content-modal';
import { connectionConfig } from '@/configs';

/**
 * creation provider key dialog props that gives ability for opening and closing modal props
 */
interface CreateToolProviderCredentialDialogProps extends ModalProps {
  /**
   * exiting tool that will need to get connected with
   */
  toolProvider: ToolProvider;
}
/**
 *
 * to create a provider key for given model
 * @param props
 * @returns
 */
export const CreateToolProviderCredentialDialog: FC<
  CreateToolProviderCredentialDialogProps
> = ({ toolProvider, setModalOpen, modalOpen }) => {
  /**
   *current provider
   */
  const [userId, token] = useCredential();

  /**
   *
   */
  const { loading, showLoader, hideLoader } = useRapidaStore();
  /**
   * form controlling
   */
  const { register, handleSubmit, reset } = useForm();
  const [error, setError] = useState('');

  const providerCtx = useProviderContext();

  /**
   * after creating provider key
   */
  const afterCreateToolCredential = useCallback(
    (err: ServiceError | null, cpkr: GetCredentialResponse | null) => {
      hideLoader();
      if (cpkr?.getSuccess()) {
        toast.success(
          'Tool credential have been successfully added to the vault.',
        );
        providerCtx.reloadProviderCredentials();
        setError('');
        setModalOpen(false);
        reset();
      } else {
        let errorMessage = cpkr?.getError();
        if (errorMessage) {
          setError(errorMessage.getHumanmessage());
          //   toast.error(errorMessage.getHumanmessage());
          return;
        } else
          setError('Unable to process your request. please try again later.');
        // toast.error('Unable to process your request. please try again later.');
        return;
      }
    },
    [],
  );
  /**
   *
   * @param provider
   * @param data
   * @returns
   */
  const onCreateToolCredential = data => {
    if (!toolProvider) {
      setError('Please select the provider which you want to create the key.');
      return;
    }
    showLoader();
    CreateToolCredential(
      connectionConfig,
      toolProvider.getId(),
      toolProvider.getName(),
      data.config,
      data.keyName,
      afterCreateToolCredential,
      {
        authorization: token,
        'x-auth-id': userId,
      },
    );
  };
  /**
   *
   */
  return (
    <CenterModal
      modalOpen={modalOpen}
      setModalOpen={setModalOpen}
      title="Connect tool"
      action="Connect"
      onSubmit={handleSubmit(onCreateToolCredential)}
      actionWrapper={(action: any) => {
        return (
          <OutlineButton
            type="submit"
            className="text-sm h-8! font-medium"
            isLoading={loading}
          >
            {action}
          </OutlineButton>
        );
      }}
    >
      <div className={cn('space-y-6')}>
        <FieldSet>
          <Label for="keyName" text="Key Name"></Label>
          <Input
            type="text"
            {...register('keyName')}
            required
            placeholder="Assign a unique name to this provider key for easy identification."
          ></Input>
        </FieldSet>

        {(() => {
          const formInput = toolProvider
            .getConnectconfigurationMap()
            .get('form_input');
          if (!formInput) return null;
          let parsedFormInput;
          try {
            parsedFormInput = JSON.parse(formInput);
          } catch (error) {
            console.error('Failed to parse form_input:', error);
            return null;
          }

          return parsedFormInput.map((x, idx) => (
            <FieldSet key={idx}>
              <Label htmlFor={`config.${x.name}`} text={x.label} />
              <Input
                type="text"
                required
                placeholder={x.label}
                {...register(`config.${x.name}`)}
              />
            </FieldSet>
          ));
        })()}

        <ErrorMessage message={error} />
      </div>
    </CenterModal>
  );
};
