import { useCallback, useContext, useState } from 'react';
import type {
  Project,
  ServiceError,
  UpdateProjectResponse,
} from '@rapidaai/react';
import toast from 'react-hot-toast/headless';
import { useForm } from 'react-hook-form';
import { useCredential } from '@/hooks/use-credential';
import { useRapidaStore } from '@/stores/app';
import { ErrorMessage } from '@/app/components/ui/feedback';
import { AuthContext } from '@/context/auth-context';
import {
  Form,
  Modal,
  ModalBody,
  ModalFooter,
  ModalHeader,
  type ModalProps,
  PrimaryButton,
  SecondaryButton,
  Stack,
  TextArea,
  TextInput,
} from '@/app/components/ui/primitives';
import { updateWorkspaceProject } from '@/clients';

const PROJECT_ACTION_ERROR =
  'Unable to process your request. please try again later.';

interface UpdateProjectDialogProps extends ModalProps {
  afterUpdateProject: () => void;
  existingProject: Project.AsObject;
}

type ProjectFormValues = {
  projectId: string;
  projectName: string;
  projectDescription: string;
};

export const UpdateProjectDialog = ({
  afterUpdateProject,
  existingProject,
  modalOpen,
  setModalOpen,
}: UpdateProjectDialogProps) => {
  const { register, handleSubmit } = useForm<ProjectFormValues>({
    defaultValues: {
      projectId: existingProject.id,
      projectName: existingProject.name,
      projectDescription: existingProject.description,
    },
  });
  const [error, setError] = useState<string>();
  const [userId, token, , organizationId] = useCredential();
  const { loading, showLoader, hideLoader } = useRapidaStore();
  const { authorize } = useContext(AuthContext);

  const closeDialog = useCallback(() => {
    setModalOpen(false);
  }, [setModalOpen]);

  const handleUpdateProject = useCallback(
    (err: ServiceError | null, upr: UpdateProjectResponse | null) => {
      if (err) {
        hideLoader();
        toast.error(PROJECT_ACTION_ERROR);
        setError(PROJECT_ACTION_ERROR);
        return;
      }

      if (upr?.getSuccess()) {
        if (authorize) {
          authorize(
            () => {
              hideLoader();
              toast.success('Your project has been updated successfully.');
              closeDialog();
              afterUpdateProject();
            },
            () => {
              hideLoader();
              toast.error(PROJECT_ACTION_ERROR);
              setError(PROJECT_ACTION_ERROR);
            },
          );
        } else {
          hideLoader();
        }
        return;
      } else {
        hideLoader();
        const errorMessage = upr?.getError();
        if (errorMessage) {
          toast.error(errorMessage.getHumanmessage());
          setError(errorMessage.getHumanmessage());
        } else {
          setError(PROJECT_ACTION_ERROR);
          toast.error(PROJECT_ACTION_ERROR);
        }
        return;
      }
    },
    [afterUpdateProject, authorize, closeDialog, hideLoader],
  );

  const onUpdateProject = (data: ProjectFormValues) => {
    setError(undefined);
    showLoader();
    updateWorkspaceProject({
      projectId: existingProject.id,
      name: data.projectName,
      description: data.projectDescription,
      auth: { organizationId, token, userId },
      callback: handleUpdateProject,
    });
  };

  return (
    <Modal open={modalOpen} onClose={closeDialog} size="sm">
      <ModalHeader
        label="Project"
        title="Update the project"
        onClose={closeDialog}
      />
      <Form onSubmit={handleSubmit(onUpdateProject)}>
        <input {...register('projectId')} type="hidden" />
        <ModalBody hasForm>
          <Stack gap={6}>
            <TextInput
              id="projectName"
              labelText="Project Name"
              placeholder="eg: your favorite project"
              required
              {...register('projectName')}
            />
            <TextArea
              id="projectDescription"
              labelText="Project Description"
              placeholder="A description of what this project is about..."
              rows={3}
              required
              {...register('projectDescription')}
            />
            <ErrorMessage message={error} />
          </Stack>
        </ModalBody>
        <ModalFooter>
          <SecondaryButton size="lg" onClick={closeDialog}>
            Cancel
          </SecondaryButton>
          <PrimaryButton size="lg" type="submit" isLoading={loading}>
            Update
          </PrimaryButton>
        </ModalFooter>
      </Form>
    </Modal>
  );
};
