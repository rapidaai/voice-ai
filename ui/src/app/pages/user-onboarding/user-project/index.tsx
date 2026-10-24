import { useCallback, useContext, useState } from 'react';
import type { CreateProjectResponse, ServiceError } from '@rapidaai/react';
import { ArrowRight } from '@carbon/icons-react';
import { Controller, useForm } from 'react-hook-form';
import { useNavigate } from 'react-router-dom';
import { Helmet } from '@/app/components/app-shell/helmet';
import { Notification } from '@/app/components/ui/feedback';
import {
  FluidForm,
  FluidTextArea,
  FluidTextInput,
  PrimaryButton,
  Text,
} from '@/app/components/ui/primitives';
import { createWorkspaceProject } from '@/clients';
import { AuthContext } from '@/context/auth-context';
import { useCurrentCredential } from '@/hooks/use-credential';
import { useRapidaStore } from '@/stores/app';

type ProjectFormValues = {
  projectName: string;
  projectDescription: string;
};

const PROJECT_ERROR = "We couldn't create your project. Try again.";

export function CreateProjectPage() {
  const navigate = useNavigate();
  const { loading, showLoader, hideLoader } = useRapidaStore();
  const { authorize } = useContext(AuthContext);
  const { authId, token, user } = useCurrentCredential();
  const profileName = user?.name?.trim();
  const {
    register,
    control,
    handleSubmit,
    formState: { errors },
  } = useForm<ProjectFormValues>({
    mode: 'onBlur',
    reValidateMode: 'onChange',
    defaultValues: {
      projectName: profileName ? `${profileName}'s project` : '',
      projectDescription: '',
    },
  });
  const [error, setError] = useState('');

  const afterCreateProject = useCallback(
    (err: ServiceError | null, project: CreateProjectResponse | null) => {
      if (err) {
        hideLoader();
        setError(PROJECT_ERROR);
        return;
      }

      if (!project?.getSuccess()) {
        hideLoader();
        setError(project?.getError()?.getHumanmessage() || PROJECT_ERROR);
        return;
      }

      if (!authorize) {
        hideLoader();
        setError("We couldn't refresh your workspace. Log in again.");
        return;
      }

      authorize(
        () => {
          hideLoader();
          navigate('/dashboard');
        },
        () => {
          hideLoader();
          setError("We couldn't refresh your workspace. Log in again.");
        },
      );
    },
    [authorize, hideLoader, navigate],
  );

  const onCreateProject = (data: ProjectFormValues) => {
    setError('');
    showLoader('overlay');
    createWorkspaceProject({
      name: data.projectName,
      description: data.projectDescription.trim(),
      auth: { token, userId: authId },
      callback: afterCreateProject,
    });
  };

  return (
    <>
      <Helmet title="Onboarding: Create a project" />
      <div className="mb-10">
        <Text as="h1" variant="heading-05" className="m-0">
          Create your first project
        </Text>
        <Text as="p" variant="body-02" className="mt-2 text-muted">
          Use projects to organize work by client, brand, region, or team.
        </Text>
      </div>

      <FluidForm
        noValidate
        data-testid="project-form"
        onChange={() => {
          if (error) setError('');
        }}
        onSubmit={handleSubmit(onCreateProject)}
      >
        <div className="flex flex-col" data-testid="project-fields">
          <FluidTextInput
            id="project-name"
            labelText="Project name"
            type="text"
            required
            invalid={Boolean(errors.projectName)}
            invalidText={errors.projectName?.message}
            {...register('projectName', {
              required: 'Enter a project name.',
              setValueAs: value => value.trim(),
            })}
          />
          <Controller
            name="projectDescription"
            control={control}
            render={({ field }) => (
              <FluidTextArea
                id="project-description"
                labelText="Project description (optional)"
                rows={3}
                name={field.name}
                value={field.value}
                onBlur={field.onBlur}
                onChange={field.onChange}
              />
            )}
          />
        </div>
        {error && (
          <div className="mt-6">
            <Notification
              kind="error"
              title="Project not created"
              subtitle={error}
            />
          </div>
        )}
        <div className="mt-12" data-testid="project-actions">
          <PrimaryButton
            size="lg"
            renderIcon={ArrowRight}
            type="submit"
            isLoading={loading}
            className="!w-full !max-w-none !justify-between"
          >
            Create project
          </PrimaryButton>
        </div>
      </FluidForm>
    </>
  );
}
