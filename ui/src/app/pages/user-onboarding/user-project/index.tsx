import { useCallback, useContext, useState } from 'react';
import type { CreateProjectResponse, ServiceError } from '@rapidaai/react';
import { ArrowRight } from '@carbon/icons-react';
import { useForm } from 'react-hook-form';
import { useNavigate } from 'react-router-dom';
import { Helmet } from '@/app/components/app-shell/helmet';
import { Notification } from '@/app/components/ui/feedback';
import {
  PrimaryButton,
  Stack,
  TextArea,
  TextInput,
} from '@/app/components/ui/primitives';
import { createWorkspaceProject } from '@/clients';
import { AuthContext } from '@/context/auth-context';
import { useCurrentCredential } from '@/hooks/use-credential';
import { useRapidaStore } from '@/stores/app';

type ProjectFormValues = {
  projectName: string;
  projectDescription: string;
};

const PROJECT_ERROR = 'Unable to create your project. Please try again.';

export function CreateProjectPage() {
  const navigate = useNavigate();
  const { loading, showLoader, hideLoader } = useRapidaStore();
  const { authorize } = useContext(AuthContext);
  const { authId, token, user } = useCurrentCredential();
  const profileName = user?.name?.trim();
  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<ProjectFormValues>({
    mode: 'onBlur',
    reValidateMode: 'onChange',
    defaultValues: {
      projectName: profileName ? `${profileName}'s workspace` : '',
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
        setError('Unable to refresh your workspace. Please sign in again.');
        return;
      }

      authorize(
        () => {
          hideLoader();
          navigate('/dashboard');
        },
        () => {
          hideLoader();
          setError('Unable to refresh your workspace. Please sign in again.');
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
      description: data.projectDescription,
      auth: { token, userId: authId },
      callback: afterCreateProject,
    });
  };

  return (
    <>
      <Helmet title="Onboarding: Create a project" />
      <div className="mb-8">
        <h1 className="text-2xl font-light tracking-tight">
          Create your first project
        </h1>
        <p className="mt-3 text-sm leading-5 text-muted">
          Projects separate client accounts, brands, regions, or internal
          product teams.
        </p>
      </div>

      <form
        noValidate
        onChange={() => {
          if (error) setError('');
        }}
        onSubmit={handleSubmit(onCreateProject)}
      >
        <Stack gap={6}>
          <TextInput
            id="project-name"
            labelText="Project name"
            type="text"
            required
            placeholder="e.g., Acme Support Operations"
            helperText="Use one project per client, brand, region, or team."
            invalid={Boolean(errors.projectName)}
            invalidText={errors.projectName?.message}
            {...register('projectName', {
              required: 'Enter a project name.',
              setValueAs: value => value.trim(),
            })}
          />
          <TextArea
            id="project-description"
            labelText="Project description (optional)"
            rows={3}
            placeholder="e.g., White-label inbound voice agents for healthcare support across US and UK"
            helperText="Capture the ownership or scope of this project."
            {...register('projectDescription', {
              setValueAs: value => value.trim(),
            })}
          />
          {error && (
            <Notification
              kind="error"
              title="Project not created"
              subtitle={error}
            />
          )}
          <PrimaryButton
            size="lg"
            renderIcon={ArrowRight}
            type="submit"
            isLoading={loading}
            className="!w-full !max-w-none !justify-between"
          >
            Create project
          </PrimaryButton>
        </Stack>
      </form>
    </>
  );
}
