import { IBlueBGButton, ICancelButton } from '@/app/components/Form/Button';
import { ErrorMessage } from '@/app/components/Form/error-message';
import { GenericModal, ModalProps } from '@/app/components/base/modal';
import { useRapidaStore } from '@/hooks';
import React, { useEffect, useState } from 'react';
import { FieldSet } from '@/app/components/Form/Fieldset';
import { Tooltip } from '@/app/components/Tooltip';
import { Input } from '@/app/components/Form/Input';
import { cn } from '@/styles/media';
import { InfoIcon } from '@/app/components/Icon/Info';
import { FormLabel } from '@/app/components/form-label';
import { ModalBody } from '@/app/components/base/modal/modal-body';
import { ModalFooter } from '@/app/components/base/modal/modal-footer';
import { MoveRight } from 'lucide-react';
import { ModalHeader } from '@/app/components/base/modal/modal-header';
import { ModalFitHeightBlock } from '@/app/components/blocks/modal-fit-height-block';
import { ModalTitleBlock } from '@/app/components/blocks/modal-title-block';
import { Textarea } from '@/app/components/Form/Textarea';

interface UpdateDescriptionDialogProps extends ModalProps {
  title?: string;
  name?: string;
  description?: string;
  onUpdateDescription: (
    name: string,
    description: string,
    onError: (err: string) => void,
    onSuccess: () => void,
  ) => void;
}

export function UpdateDescriptionDialog(props: UpdateDescriptionDialogProps) {
  const [error, setError] = useState('');
  const [name, setName] = useState<string>('');
  const [description, setDescription] = useState<string>('');
  const rapidaStore = useRapidaStore();

  useEffect(() => {
    if (props.name) setName(props.name);
    if (props.description) setDescription(props.description);
  }, [props.name, props.description]);

  const onUpdateDescription = () => {
    rapidaStore.showLoader('overlay');
    props.onUpdateDescription(
      name,
      description,
      err => {
        rapidaStore.hideLoader();
        setError(err);
      },
      () => {
        rapidaStore.hideLoader();
        props.setModalOpen(false);
      },
    );
  };

  return (
    <GenericModal modalOpen={props.modalOpen} setModalOpen={props.setModalOpen}>
      <ModalFitHeightBlock>
        <ModalHeader
          onClose={() => {
            props.setModalOpen(false);
          }}
        >
          <ModalTitleBlock>{props.title}</ModalTitleBlock>
        </ModalHeader>
        <ModalBody>
          <FieldSet>
            <FormLabel>
              Name
              <Tooltip icon={<InfoIcon className="w-4 h-4 ml-1" />}>
                <p className={cn('font-normal text-sm p-1 w-64')}>
                  Give a name that you can use to identify later.
                </p>
              </Tooltip>
            </FormLabel>
            <Input
              name="usecase"
              onChange={e => {
                setName(e.target.value);
              }}
              value={name}
              className="form-input"
              placeholder="eg: your emotion detector"
            ></Input>
          </FieldSet>

          <FieldSet>
            <FormLabel>
              Description
              <Tooltip
                icon={
                  <InfoIcon className="w-4 h-4 mt-[2px] ml-0.5 dark:text-gray-400" />
                }
              >
                <p className={cn('font-normal text-sm p-1 w-64')}>
                  Add a readable description and how to use it.
                </p>
              </Tooltip>
            </FormLabel>
            <Textarea
              rows={4}
              value={description}
              placeholder={`Provider a readable description and how to use it.`}
              onChange={v => {
                setDescription(v.target.value);
              }}
            />
          </FieldSet>
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
            type="button"
            onClick={onUpdateDescription}
          >
            Update details
            <MoveRight className="ml-2" strokeWidth={1.5} />
          </IBlueBGButton>
        </ModalFooter>
      </ModalFitHeightBlock>
    </GenericModal>
  );
}
