import React, { useCallback, useContext, useState } from 'react';
import { Input } from '@/app/components/Form/Input';
import { Textarea } from '@/app/components/Form/Textarea';
import { UpdateProject } from '@rapidaai/react';

import { Project, UpdateProjectResponse } from '@rapidaai/react';
import toast from 'react-hot-toast/headless';
import { useForm } from 'react-hook-form';
import { useCredential } from '@/hooks/use-credential';
import { useRapidaStore } from '@/hooks';
import { ErrorMessage } from '@/app/components/Form/error-message';
import { GenericModal, ModalProps } from '@/app/components/base/modal';
import { ServiceError } from '@rapidaai/react';
import { AuthContext } from '@/context/auth-context';
import {
  IBlueBGArrowButton,
  ICancelButton,
} from '@/app/components/Form/Button';
import { ModalBody } from '@/app/components/base/modal/modal-body';
import { ModalFooter } from '@/app/components/base/modal/modal-footer';
import { FieldSet } from '@/app/components/Form/Fieldset';
import { FormLabel } from '@/app/components/form-label';
import { ModalTitleBlock } from '@/app/components/blocks/modal-title-block';
import { ModalFormBlock } from '@/app/components/blocks/modal-form-block';
import { ModalHeader } from '@/app/components/base/modal/modal-header';
import { connectionConfig } from '@/configs';

/**
 * for project creation dialog
 */
interface UpdateProjectDialogProps extends ModalProps {
  /**
   *
   * @param boolean
   * @returns
   */
  afterUpdateProject: () => void;

  /**
   * for updation of existing project
   */
  existingProject: Project.AsObject;
}

/**
 *
 * @param props
 * @returns
 */
export const UpdateProjectDialog = (props: UpdateProjectDialogProps) => {
  /**
   *form hook
   */
  const { register, handleSubmit } = useForm();

  /**
   * existing project
   */
  const [project, setProject] = useState<Partial<Project.AsObject>>(
    props.existingProject,
  );

  /**
   * error
   */
  const [error, setError] = useState<string>();

  /**
   * use credentials
   */
  const [userId, token] = useCredential();

  /**
   * set loading context
   */
  const { loading, showLoader, hideLoader } = useRapidaStore();

  /**
   * to re initiate the cookies
   */
  const { authorize } = useContext(AuthContext);

  /**
   * After update project
   */

  const afterUpdateProject = useCallback(
    (err: ServiceError | null, upr: UpdateProjectResponse | null) => {
      if (err) {
        hideLoader();
        toast.error('Unable to process your request. please try again later.');
        setError('Unable to process your request. please try again later.');
        return;
      }
      if (upr?.getSuccess()) {
        if (authorize)
          authorize(
            () => {
              toast.success('Your project has been updated successfully.');
              props.setModalOpen(false);
              props.afterUpdateProject();
            },
            err => {
              props.setModalOpen(false);
            },
          );
      } else {
        hideLoader();
        let errorMessage = upr?.getError();
        if (errorMessage) {
          toast.error(errorMessage.getHumanmessage());
          setError(errorMessage.getHumanmessage());
        } else {
          setError('Unable to process your request. please try again later.');
          toast.error(
            'Unable to process your request. please try again later.',
          );
        }

        return;
      }
    },
    [],
  );

  /**
   * updating the project
   * @param data
   */
  const onUpdateProject = data => {
    showLoader();
    UpdateProject(
      connectionConfig,
      props.existingProject.id,
      afterUpdateProject,
      {
        authorization: token,
        'x-auth-id': userId,
      },
      data.projectName,
      data.projectDescription,
    );
  };

  /**
   *
   */
  return (
    <GenericModal modalOpen={props.modalOpen} setModalOpen={props.setModalOpen}>
      <ModalFormBlock onSubmit={handleSubmit(onUpdateProject)}>
        <ModalHeader
          onClose={() => {
            props.setModalOpen(false);
          }}
        >
          <ModalTitleBlock>Update the project</ModalTitleBlock>
        </ModalHeader>
        <ModalBody>
          <FieldSet>
            <input
              {...register('projectId', { value: project?.id })}
              type="hidden"
            />

            <FormLabel>Project Name</FormLabel>
            <Input
              required
              type="text"
              {...register('projectName')}
              value={project?.name}
              onChange={e => {
                setProject({ ...project, name: e.target.value });
              }}
              placeholder="eg: your favorite project"
            ></Input>
          </FieldSet>
          <FieldSet>
            <FormLabel>Project Description</FormLabel>
            <Textarea
              id="description"
              row={3}
              {...register('projectDescription')}
              required
              value={project?.description}
              onChange={e => {
                setProject({ ...project, description: e.target.value });
              }}
              placeholder="An description of what this project about..."
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
            Update
          </IBlueBGArrowButton>
        </ModalFooter>
      </ModalFormBlock>
    </GenericModal>
  );
};
