import React, { useCallback, useContext, useState } from 'react';
import { Input } from '@/app/components/Form/Input';
import { Textarea } from '@/app/components/Form/Textarea';
import { CreateProject } from '@rapidaai/react';
import { CreateProjectResponse } from '@rapidaai/react';
import { useForm } from 'react-hook-form';
import { useCurrentCredential } from '@/hooks/use-credential';
import { useRapidaStore } from '@/hooks';
import { ErrorMessage } from '@/app/components/Form/error-message';
import toast from 'react-hot-toast/headless';
import { GenericModal, ModalProps } from '@/app/components/base/modal';
import { ServiceError } from '@rapidaai/react';
import { AuthContext } from '@/context/auth-context';
import { FieldSet } from '@/app/components/Form/Fieldset';
import {
  IBlueBGArrowButton,
  ICancelButton,
} from '@/app/components/Form/Button';
import { ModalHeader } from '@/app/components/base/modal/modal-header';
import { ModalBody } from '@/app/components/base/modal/modal-body';
import { ModalFooter } from '@/app/components/base/modal/modal-footer';
import { FormLabel } from '@/app/components/form-label';
import { ModalTitleBlock } from '@/app/components/blocks/modal-title-block';
import { ModalFormBlock } from '@/app/components/blocks/modal-form-block';
import { connectionConfig } from '@/configs';

/**
 * for project creation dialog
 */
interface CreateProjectDialogProps extends ModalProps {
  /**
   *
   * @param boolean
   * @returns
   */
  afterCreateProject: () => void;
}

/**
 *
 * @param props
 * @returns
 */
export const CreateProjectDialog = (props: CreateProjectDialogProps) => {
  /**
   * form submit
   */
  const { register, handleSubmit } = useForm();

  /**
   * loading context
   */
  const { loading, showLoader, hideLoader } = useRapidaStore();

  /**
   * revive authorization
   */
  const { authorize } = useContext(AuthContext);
  /**
   * Credentials
   */
  const { authId, token } = useCurrentCredential();

  /**
   * error
   */
  const [error, setError] = useState<string>();

  /**
   * after creation of the project
   */
  const afterCreateProject = useCallback(
    async (err: ServiceError | null, cpr: CreateProjectResponse | null) => {
      if (err) {
        hideLoader();
        toast.error('Unable to process your request. please try again later.');
        setError('Unable to process your request. please try again later.');
        return;
      }
      if (cpr?.getSuccess()) {
        if (authorize)
          authorize(
            () => {
              console.log('success');
              hideLoader();
              toast.success('The project has been created successfully.');
              props.setModalOpen(false);
              props.afterCreateProject();
            },
            err => {
              console.log('failure');
              hideLoader();
            },
          );
      } else {
        hideLoader();
        let errorMessage = cpr?.getError();
        if (errorMessage) {
          toast.error(errorMessage.getHumanmessage());
          setError(errorMessage.getHumanmessage());
        } else {
          toast.error(
            'Unable to process your request. please try again later.',
          );
          setError('Unable to process your request. please try again later.');
        }
        return;
      }
    },
    [],
  );

  /**
   *
   * @param data
   */
  const onCreateProject = data => {
    showLoader();
    CreateProject(
      connectionConfig,
      data.projectName,
      data.projectDescription,
      {
        authorization: token,
        'x-auth-id': authId,
      },
      afterCreateProject,
    );
  };

  return (
    <GenericModal modalOpen={props.modalOpen} setModalOpen={props.setModalOpen}>
      <ModalFormBlock onSubmit={handleSubmit(onCreateProject)}>
        <ModalHeader
          onClose={() => {
            props.setModalOpen(false);
          }}
        >
          <ModalTitleBlock>Create a project</ModalTitleBlock>
        </ModalHeader>
        <ModalBody>
          <FieldSet>
            <FormLabel>Project Name</FormLabel>
            <Input
              required
              type="text"
              placeholder="eg: your favorite project"
              {...register('projectName')}
            ></Input>
          </FieldSet>
          <FieldSet>
            <FormLabel>Project Description</FormLabel>
            <Textarea
              required
              {...register('projectDescription')}
              row={3}
              placeholder="An optional description of what this project about..."
            ></Textarea>
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
          <IBlueBGArrowButton
            className="px-4 rounded-[2px]"
            type="submit"
            isLoading={loading}
          >
            Create Project
          </IBlueBGArrowButton>
        </ModalFooter>
      </ModalFormBlock>
    </GenericModal>
  );
};
