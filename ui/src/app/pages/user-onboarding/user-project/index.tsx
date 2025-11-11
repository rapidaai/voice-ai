import React, { useCallback, useContext, useState } from 'react';
import { DescriptiveHeading } from '@/app/components/Heading/DescriptiveHeading';
import { Helmet } from '@/app/components/Helmet';
import { Textarea } from '@/app/components/Form/Textarea';
import { Input } from '@/app/components/Form/Input';
import { useNavigate } from 'react-router-dom';
import { useForm } from 'react-hook-form';
import { CreateProject } from '@rapidaai/react';
import { CreateProjectResponse } from '@rapidaai/react';
import { useCurrentCredential } from '@/hooks/use-credential';
import { useRapidaStore } from '@/hooks';
import { IBlueBGArrowButton } from '@/app/components/Form/Button';
import { ErrorMessage } from '@/app/components/Form/error-message';
import { ServiceError } from '@rapidaai/react';
import { AuthContext } from '@/context/auth-context';
import { FieldSet } from '@/app/components/Form/Fieldset';
import { FormLabel } from '@/app/components/form-label';
import { connectionConfig } from '@/configs';
export function CreateProjectPage() {
  /**
   * To naviagate to dashboard
   */
  let navigate = useNavigate();

  /**
   * setLoading context
   */
  const { loading, showLoader, hideLoader } = useRapidaStore();

  /**
   * Authenticaiton Context
   */
  const { authorize } = useContext(AuthContext);

  /**
   * credentials
   */
  const { authId, token, user } = useCurrentCredential();

  /**
   * handle the form
   */
  const { register, handleSubmit } = useForm();
  const [error, setError] = useState('');

  /**
   *
   * @param project creation
   */
  const afterCreateProject = useCallback(
    async (err: ServiceError | null, cpr: CreateProjectResponse | null) => {
      hideLoader();
      if (err) {
        setError('Unable to process your request. please try again later.');
        return;
      }
      if (cpr?.getSuccess()) {
        authorize &&
          authorize(
            () => {
              navigate('/dashboard');
            },
            () => {
              setError('Unable to create project please check the details');
            },
          );
      } else {
        setError('Unable to create project please check the details');
      }
    },
    [],
  );

  const onCreateProject = data => {
    showLoader('overlay');
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
    <>
      <Helmet title="Onboarding: Create a Project"></Helmet>
      <DescriptiveHeading
        heading="Create your Project"
        subheading="This is where your team can work and collabrate on the projects."
      ></DescriptiveHeading>
      <form className="space-y-6 mt-6" onSubmit={handleSubmit(onCreateProject)}>
        <FieldSet>
          <FormLabel>Project Name</FormLabel>
          <Input
            required
            type="text"
            defaultValue={`${user?.name}'s Workspace`}
            placeholder="eg: your favorite project"
            {...register('projectName')}
          ></Input>
        </FieldSet>
        <FieldSet>
          <FormLabel>Project Description</FormLabel>
          <Textarea
            {...register('projectDescription')}
            row={3}
            placeholder="A description of what this project about..."
          ></Textarea>
        </FieldSet>
        <ErrorMessage message={error} />
        <IBlueBGArrowButton
          type="submit"
          className="w-full justify-between"
          isLoading={loading}
        >
          Continue
        </IBlueBGArrowButton>
      </form>
    </>
  );
}
