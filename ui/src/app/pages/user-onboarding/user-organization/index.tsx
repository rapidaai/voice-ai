import { useCallback, useContext, useState } from 'react';
import type { CreateOrganizationResponse, ServiceError } from '@rapidaai/react';
import { SelectItem } from '@carbon/react';
import { ArrowRight } from '@carbon/icons-react';
import { useForm } from 'react-hook-form';
import { useNavigate } from 'react-router-dom';
import { Helmet } from '@/app/components/app-shell/helmet';
import { Notification } from '@/app/components/ui/feedback';
import {
  FluidForm,
  FluidSelect,
  FluidTextInput,
  PrimaryButton,
  Text,
} from '@/app/components/ui/primitives';
import { createWorkspaceOrganization } from '@/clients';
import { AuthContext } from '@/context/auth-context';
import { useCurrentCredential } from '@/hooks/use-credential';
import { useRapidaStore } from '@/stores/app';

type OrganizationFormValues = {
  organizationName: string;
  organizationSize: string;
  organizationIndustry: string;
};

const ORGANIZATION_ERROR = "We couldn't create your organization. Try again.";

export function CreateOrganizationPage() {
  const navigate = useNavigate();
  const { loading, showLoader, hideLoader } = useRapidaStore();
  const { authorize } = useContext(AuthContext);
  const { user, authId, token } = useCurrentCredential();
  const profileName = user?.name?.trim();
  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<OrganizationFormValues>({
    mode: 'onBlur',
    reValidateMode: 'onChange',
    defaultValues: {
      organizationName: profileName ? `${profileName}'s organization` : '',
      organizationSize: '',
      organizationIndustry: '',
    },
  });
  const [error, setError] = useState('');

  const afterCreateOrganization = useCallback(
    (err: ServiceError | null, org: CreateOrganizationResponse | null) => {
      if (err) {
        hideLoader();
        setError(ORGANIZATION_ERROR);
        return;
      }

      if (!org?.getSuccess()) {
        hideLoader();
        setError(org?.getError()?.getHumanmessage() || ORGANIZATION_ERROR);
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
          navigate('/onboarding/project');
        },
        () => {
          hideLoader();
          setError("We couldn't refresh your workspace. Log in again.");
        },
      );
    },
    [authorize, hideLoader, navigate],
  );

  const onCreateOrganization = (data: OrganizationFormValues) => {
    setError('');
    showLoader('overlay');
    createWorkspaceOrganization({
      name: data.organizationName,
      size: data.organizationSize,
      industry: data.organizationIndustry,
      auth: { token, userId: authId },
      callback: afterCreateOrganization,
    });
  };

  return (
    <>
      <Helmet title="Onboarding: Create an organization" />
      <div className="mb-10">
        <Text as="h1" variant="heading-05" className="m-0">
          Create your organization
        </Text>
        <Text as="p" variant="body-02" className="mt-2 text-muted">
          Tell us about your organization to set up your workspace.
        </Text>
      </div>

      <FluidForm
        noValidate
        data-testid="organization-form"
        onChange={() => {
          if (error) setError('');
        }}
        onSubmit={handleSubmit(onCreateOrganization)}
      >
        <div className="flex flex-col" data-testid="organization-fields">
          <FluidTextInput
            id="org-name"
            labelText="Organization name"
            type="text"
            required
            invalid={Boolean(errors.organizationName)}
            invalidText={errors.organizationName?.message}
            {...register('organizationName', {
              required: 'Enter an organization name.',
              setValueAs: value => value.trim(),
            })}
          />
          <FluidSelect
            id="org-size"
            labelText="Organization type"
            required
            invalid={Boolean(errors.organizationSize)}
            invalidText={errors.organizationSize?.message}
            {...register('organizationSize', {
              required: 'Select an organization type.',
            })}
          >
            <SelectItem value="" text="Select an organization type" />
            <SelectItem value="agency" text="Agency or system integrator" />
            <SelectItem value="in-house" text="In-house product team" />
            <SelectItem
              value="enterprise"
              text="Enterprise with multiple teams"
            />
          </FluidSelect>
          <FluidTextInput
            id="org-industry"
            labelText="Industry"
            type="text"
            required
            invalid={Boolean(errors.organizationIndustry)}
            invalidText={errors.organizationIndustry?.message}
            {...register('organizationIndustry', {
              required: 'Enter an industry.',
              setValueAs: value => value.trim(),
            })}
          />
        </div>
        {error && (
          <div className="mt-6">
            <Notification
              kind="error"
              title="Organization not created"
              subtitle={error}
            />
          </div>
        )}
        <div className="mt-12" data-testid="organization-actions">
          <PrimaryButton
            size="lg"
            renderIcon={ArrowRight}
            type="submit"
            isLoading={loading}
            className="!w-full !max-w-none !justify-between"
          >
            Create organization
          </PrimaryButton>
        </div>
      </FluidForm>
    </>
  );
}
