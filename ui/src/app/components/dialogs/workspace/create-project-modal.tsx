import { useCallback, useContext, useState } from 'react';
import type { CreateProjectResponse, ServiceError } from '@rapidaai/react';
import { useForm } from 'react-hook-form';
import { useCurrentCredential } from '@/hooks/use-credential';
import { useRapidaStore } from '@/stores/app';
import { ErrorMessage } from '@/app/components/ui/feedback';
import toast from 'react-hot-toast/headless';
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
import { createWorkspaceProject } from '@/clients';

const PROJECT_ACTION_ERROR =
  'Unable to process your request. please try again later.';

interface CreateProjectDialogProps extends ModalProps {
  afterCreateProject: () => void;
}

type ProjectFormValues = {
  projectName: string;
  projectDescription: string;
};

export const CreateProjectDialog = ({
  afterCreateProject,
  modalOpen,
  setModalOpen,
}: CreateProjectDialogProps) => {
  const { register, handleSubmit } = useForm<ProjectFormValues>();
  const { loading, showLoader, hideLoader } = useRapidaStore();
  const { authorize } = useContext(AuthContext);
  const { authId, organizationId, token } = useCurrentCredential();
  const [error, setError] = useState<string>();

  const closeDialog = useCallback(() => {
    setModalOpen(false);
  }, [setModalOpen]);

  const handleCreateProject = useCallback(
    async (err: ServiceError | null, cpr: CreateProjectResponse | null) => {
      if (err) {
        hideLoader();
        toast.error(PROJECT_ACTION_ERROR);
        setError(PROJECT_ACTION_ERROR);
        return;
      }

      if (cpr?.getSuccess()) {
        if (authorize) {
          authorize(
            () => {
              hideLoader();
              toast.success('The project has been created successfully.');
              closeDialog();
              afterCreateProject();
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
        const errorMessage = cpr?.getError();
        if (errorMessage) {
          toast.error(errorMessage.getHumanmessage());
          setError(errorMessage.getHumanmessage());
        } else {
          toast.error(PROJECT_ACTION_ERROR);
          setError(PROJECT_ACTION_ERROR);
        }
        return;
      }
    },
    [afterCreateProject, authorize, closeDialog, hideLoader],
  );

  const onCreateProject = (data: ProjectFormValues) => {
    setError(undefined);
    showLoader();
    createWorkspaceProject({
      name: data.projectName,
      description: data.projectDescription,
      auth: { organizationId, token, userId: authId },
      callback: handleCreateProject,
    });
  };

  return (
    <Modal open={modalOpen} onClose={closeDialog} size="sm">
      <ModalHeader
        label="Project"
        title="Create a project"
        onClose={closeDialog}
      />
      <Form onSubmit={handleSubmit(onCreateProject)}>
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
              placeholder="An optional description of what this project about..."
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
            Create Project
          </PrimaryButton>
        </ModalFooter>
      </Form>
    </Modal>
  );
};
