import React, { useCallback, useContext, useState } from 'react';
import { DescriptiveHeading } from '@/app/components/Heading/DescriptiveHeading';
import { Helmet } from '@/app/components/Helmet';
import { IBlueBGArrowButton } from '@/app/components/Form/Button';
import { Input } from '@/app/components/Form/Input';
import { Select } from '@/app/components/Form/Select';
import { useNavigate } from 'react-router-dom';
import { useForm } from 'react-hook-form';
import { CreateOrganization } from '@rapidaai/react';
import { CreateOrganizationResponse } from '@rapidaai/react';
import { useCurrentCredential } from '@/hooks/use-credential';
import { useRapidaStore } from '@/hooks';
import { ErrorMessage } from '@/app/components/Form/error-message';
import { ServiceError } from '@rapidaai/react';
import { AuthContext } from '@/context/auth-context';
import { FieldSet } from '@/app/components/Form/Fieldset';
import { FormLabel } from '@/app/components/form-label';
import { connectionConfig } from '@/configs';
export function CreateOrganizationPage() {
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
  const { user, authId, token } = useCurrentCredential();

  /**
   * handle the form
   */
  const { register, handleSubmit } = useForm();
  const [error, setError] = useState('');

  /**
   * orgnization options
   */
  const OrgOptions = [
    { name: 'Startup', value: 'startup' },
    { name: 'Late stage', value: 'late-stage' },
    { name: 'Enterprise', value: 'enterprise' },
  ];

  /**
   *
   * @param err
   * @param org
   * @returns
   */
  const afterCreateOrganization = useCallback(
    (err: ServiceError | null, org: CreateOrganizationResponse | null) => {
      if (err) {
        hideLoader();
        setError('Unable to process your request. please try again later.');
        return;
      }
      if (org?.getSuccess()) {
        authorize &&
          authorize(
            () => {
              hideLoader();
              return navigate('/onboarding/project');
            },
            () => {
              hideLoader();
              setError(
                'Please provide valid credentials to signin into account.',
              );
            },
          );
      } else {
        hideLoader();
        setError('Please provide valid credentials to signin into account.');
        return;
      }
    },
    [],
  );

  /**
   *
   * @param data
   */
  const onCreateOrganization = data => {
    showLoader('overlay');
    CreateOrganization(
      connectionConfig,
      data.organizationName,
      data.organizationSize,
      data.organizationIndustry,
      {
        authorization: token,
        'x-auth-id': authId,
      },
      afterCreateOrganization,
    );
  };
  return (
    <>
      <Helmet title="Onboarding: Create an organization"></Helmet>
      <DescriptiveHeading
        heading="Create your organization"
        subheading="This is where your team can work and collabrate on the projects."
      ></DescriptiveHeading>

      <form
        className="space-y-6 mt-6"
        onSubmit={handleSubmit(onCreateOrganization)}
      >
        <FieldSet>
          <FormLabel>Organization Name</FormLabel>
          <Input
            type="text"
            defaultValue={`${user?.name}'s Organization`}
            placeholder="eg: Lexatic Inc"
            {...register('organizationName', {
              required: 'Please enter organization name',
            })}
          ></Input>
        </FieldSet>
        <FieldSet>
          <FormLabel>How large is your company?</FormLabel>
          <Select
            placeholder="Select your orgnization size"
            {...register('organizationSize')}
            options={OrgOptions}
          ></Select>
        </FieldSet>
        <FieldSet>
          <FormLabel>Organization Industry</FormLabel>
          <Input
            type="text"
            {...register('organizationIndustry', {
              required: 'Please enter industry',
            })}
            placeholder="eg: Software, Engineering"
          ></Input>
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
