import { useCallback, useEffect, useState } from 'react';
import {
  ConnectionConfig,
  CreateProviderCredentialRequest,
} from '@rapidaai/react';
import { useForm } from 'react-hook-form';
import { useCurrentCredential } from '@/hooks/use-credential';
import { CreateProviderKey } from '@rapidaai/react';
import { GetCredentialResponse } from '@rapidaai/react';
import { Input } from '@/app/components/Form/Input';
import { ProviderDropdown } from '@/app/components/Dropdown/ProviderDropdown';
import { ErrorMessage } from '@/app/components/Form/error-message';
import { useRapidaStore } from '@/hooks';
import toast from 'react-hot-toast/headless';
import { GenericModal, ModalProps } from '@/app/components/base/modal';
import { FieldSet } from '@/app/components/Form/Fieldset';
import { IBlueBGButton, ICancelButton } from '@/app/components/Form/Button';
import { ServiceError } from '@rapidaai/react';
import { ModalFooter } from '@/app/components/base/modal/modal-footer';
import { ModalBody } from '@/app/components/base/modal/modal-body';
import { FormLabel } from '@/app/components/form-label';
import { ModalFormBlock } from '@/app/components/blocks/modal-form-block';
import { ModalHeader } from '@/app/components/base/modal/modal-header';
import { ModalTitleBlock } from '@/app/components/blocks/modal-title-block';
import { connectionConfig } from '@/configs';
import { COMPLETE_PROVIDER, RapidaProvider } from '@/app/components/providers';
import { useProviderContext } from '@/context/provider-context';
import { Select } from '@/app/components/Form/Select';
import { Textarea } from '@/app/components/Form/Textarea';
import { Struct } from 'google-protobuf/google/protobuf/struct_pb';

/**
 * creation provider key dialog props that gives ability for opening and closing modal props
 */
interface CreateProviderCredentialDialogProps extends ModalProps {
  /**
   * exiting provider if there
   */
  currentProviderId?: string | null;
}
/**
 *
 * to create a provider key for given model
 * @param props
 * @returns
 */
export function CreateProviderCredentialDialog(
  props: CreateProviderCredentialDialogProps,
) {
  /**
   *current provider
   */
  const { authId, projectId, token } = useCurrentCredential();

  const [provider, setProvider] = useState<RapidaProvider | null>();

  const providerCtx = useProviderContext();
  /**
   *
   */
  const { loading, showLoader, hideLoader } = useRapidaStore();
  /**
   * form controlling
   */
  const { register, handleSubmit, reset } = useForm();
  const [error, setError] = useState('');

  useEffect(() => {
    setProvider(
      COMPLETE_PROVIDER.slice()
        .reverse()
        .find(x => x.id === props.currentProviderId),
    );
  }, [props.currentProviderId]);

  /**
   *
   * @param provider
   * @param data
   * @returns
   */
  const onCreateProviderKey = data => {
    if (!props.currentProviderId) {
      setError('Please select the provider which you want to create the key.');
      return;
    }
    if (!provider) {
      setError('Please select the provider which you want to create the key.');
      return;
    }

    if (!data.keyName) {
      setError('Please provide a valid key name for the credential.');
      return;
    }

    showLoader();
    const requestObject = new CreateProviderCredentialRequest();
    requestObject.setProviderid(provider.id);
    requestObject.setProvidername(provider.name);
    requestObject.setCredential(Struct.fromJavaScript(data.config));
    requestObject.setName(data.keyName);
    CreateProviderKey(
      connectionConfig,
      requestObject,
      ConnectionConfig.WithDebugger({
        authorization: token,
        userId: authId,
        projectId: projectId,
      }),
    )
      .then(cpkr => {
        hideLoader();
        if (cpkr?.getSuccess()) {
          toast.success(
            'Provider credential have been successfully added to the vault.',
          );
          providerCtx.reloadProviderCredentials();
          props.setModalOpen(false);
          setError('');
          reset();
        } else {
          let errorMessage = cpkr?.getError();
          if (errorMessage) {
            setError(errorMessage.getHumanmessage());
            return;
          } else
            setError('Unable to process your request. please try again later.');
          return;
        }
      })
      .catch(err => {
        hideLoader();
        toast.error(
          'Unable to create provider credential, please try again later.',
        );
      });
  };
  /**
   *
   */
  return (
    <GenericModal modalOpen={props.modalOpen} setModalOpen={props.setModalOpen}>
      <ModalFormBlock
        onSubmit={e => {
          e.preventDefault();
        }}
      >
        <ModalHeader
          onClose={() => {
            props.setModalOpen(false);
          }}
        >
          <ModalTitleBlock>Create provider credential</ModalTitleBlock>
        </ModalHeader>
        <ModalBody>
          <FieldSet>
            <FormLabel>Select your provider</FormLabel>
            <ProviderDropdown
              currentProvider={provider ? provider : undefined}
              setCurrentProvider={p => {
                setError('');
                reset();
                setProvider(p);
              }}
            ></ProviderDropdown>
          </FieldSet>

          <FieldSet>
            <FormLabel>Key Name</FormLabel>
            <Input
              type="text"
              {...register('keyName')}
              required
              placeholder="Assign a unique name to this provider key for easy identification."
            ></Input>
          </FieldSet>

          {provider &&
            provider.configurations.map((x, idx) => {
              return (
                <FieldSet key={idx}>
                  <FormLabel htmlFor={`config.${x.name}`}>{x.label}</FormLabel>
                  {x.type === 'select' ? (
                    <Select
                      required
                      {...register(`config.${x.name}`)}
                      options={
                        x.options?.map(option => ({
                          name: option, // Use the string as the name
                          value: option, // Use the string as the value
                        })) || []
                      }
                    ></Select>
                  ) : x.type === 'text' ? (
                    <Textarea
                      required
                      placeholder={x.label}
                      {...register(`config.${x.name}`)}
                    />
                  ) : (
                    <Input
                      type={'text'}
                      required
                      placeholder={x.label}
                      {...register(`config.${x.name}`)}
                    />
                  )}
                </FieldSet>
              );
            })}

          <ErrorMessage message={error} />
        </ModalBody>
        <ModalFooter>
          <ICancelButton
            className="px-4 rounded-[2px]"
            onClick={() => {
              props.setModalOpen(false);
            }}
          >
            Cancel
          </ICancelButton>
          <IBlueBGButton
            className="px-4 rounded-[2px]"
            onClick={handleSubmit(onCreateProviderKey)}
            isLoading={loading}
          >
            Configure
          </IBlueBGButton>
        </ModalFooter>
      </ModalFormBlock>
    </GenericModal>
  );
}
