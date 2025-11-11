import React, { useCallback, useState } from 'react';
import { Input } from '@/app/components/Form/Input';
import { MultipleProjectDropdown } from '@/app/components/Dropdown/ProjectDropdown/MultipleProjectDropdown';
import { ProjectRoleDropdown } from '@/app/components/Dropdown/ProjectRoleDropdown';
import { AddUsersToProject } from '@rapidaai/react';
import { AddUsersToProjectResponse } from '@rapidaai/react';
import { useCurrentCredential } from '@/hooks/use-credential';
import { useRapidaStore } from '@/hooks';
import { ErrorMessage } from '@/app/components/Form/error-message';
import toast from 'react-hot-toast/headless';
import { GenericModal, ModalProps } from '@/app/components/base/modal';
import {
  IBlueBGArrowButton,
  ICancelButton,
} from '@/app/components/Form/Button';
import { ServiceError } from '@rapidaai/react';
import { ModalBody } from '@/app/components/base/modal/modal-body';
import { ModalFooter } from '@/app/components/base/modal/modal-footer';
import { FieldSet } from '@/app/components/Form/Fieldset';
import { FormLabel } from '@/app/components/form-label';
import { ModalHeader } from '@/app/components/base/modal/modal-header';
import { ModalTitleBlock } from '@/app/components/blocks/modal-title-block';
import { ModalFitHeightBlock } from '@/app/components/blocks/modal-fit-height-block';
import { connectionConfig } from '@/configs';

interface InviteUserDialogProps extends ModalProps {
  /**
   * if inviting for project
   */
  projectId?: string;

  onSuccess?: () => void;
}

/**
 *
 * @param props
 * @returns
 */
export function InviteUserDialog(props: InviteUserDialogProps) {
  /**
   * projects
   */
  const [projects, setProjects] = useState<string[]>([]);

  /**
   * what project roles
   */
  const [projectRole, setProjectRole] = useState<string>('');

  /**
   * email
   */
  const [email, setEmail] = useState<string>('');

  /**
   * error
   */
  const [error, setError] = useState<string>('');

  /**
   * Credentials
   */
  const { authId, token } = useCurrentCredential();

  /**
   * loading context
   */
  const { loading, showLoader, hideLoader } = useRapidaStore();

  const afterAddToProject = useCallback(
    (err: ServiceError | null, aur: AddUsersToProjectResponse | null) => {
      hideLoader();
      if (err) {
        toast.error('Unable to process your request. please try again later.');
        setError('Unable to process your request. please try again later.');
        return;
      }
      if (aur?.getSuccess()) {
        setEmail('');
        setProjectRole('');
        props.setModalOpen(false);
        toast.success(
          'The invitation of joining the projects are successfully sent to the user.',
        );
        if (props.onSuccess) props.onSuccess();
      } else {
        toast.error('Unable to process your request. please try again later.');
        setError('Unable to process your request. please try again later.');
        return;
      }
    },
    [],
  );

  const addUserToProject = () => {
    showLoader('overlay');
    if (projectRole === '') {
      setError('Please provide select a role for the user.');
      return;
    }
    if (email === '') {
      setError('Please provide a valid email to invite user.');
      return;
    }
    if (!props.projectId && projects.length === 0) {
      setError('Please select one or more project for the user to invite.');
      return;
    }

    AddUsersToProject(
      connectionConfig,
      email,
      projectRole,
      props.projectId ? [props?.projectId] : projects,
      afterAddToProject,
      {
        authorization: token,
        'x-auth-id': authId,
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
          <ModalTitleBlock>Invite new user</ModalTitleBlock>
        </ModalHeader>
        <ModalBody>
          <FieldSet>
            <FormLabel>Project Role</FormLabel>
            <ProjectRoleDropdown
              projectRole={projectRole}
              setProjectRoleId={setProjectRole}
            />
          </FieldSet>
          <FieldSet>
            <FormLabel>Email Address</FormLabel>
            <Input
              value={email}
              name="email"
              type="email"
              placeholder="eg: john@deo.io"
              onChange={e => {
                setError('');
                setEmail(e.target.value);
              }}
            />
          </FieldSet>
          {!props.projectId && (
            <fieldset className="space-y-2 col-span-1">
              <FormLabel>Projects</FormLabel>
              <MultipleProjectDropdown
                projectIds={projects}
                setProjectIds={setProjects}
              ></MultipleProjectDropdown>
            </fieldset>
          )}
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
            type="button"
            onClick={addUserToProject}
            isLoading={loading}
          >
            Invite User
          </IBlueBGArrowButton>
        </ModalFooter>
      </ModalFitHeightBlock>
    </GenericModal>
  );
}
