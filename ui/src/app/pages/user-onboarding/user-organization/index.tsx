import { useCallback, useContext, useState } from 'react';
import type { CreateOrganizationResponse, ServiceError } from '@rapidaai/react';
import { Select, SelectItem } from '@carbon/react';
import { ArrowRight } from '@carbon/icons-react';
import { useForm } from 'react-hook-form';
import { useNavigate } from 'react-router-dom';
import { Helmet } from '@/app/components/app-shell/helmet';
import { Notification } from '@/app/components/ui/feedback';
import {
  PrimaryButton,
  Stack,
  TextInput,
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

const ORGANIZATION_ERROR =
  'Unable to create your organization. Please try again.';

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
        setError('Unable to refresh your workspace. Please sign in again.');
        return;
      }

      authorize(
        () => {
          hideLoader();
          navigate('/onboarding/project');
        },
        () => {
          hideLoader();
          setError('Unable to refresh your workspace. Please sign in again.');
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
      <div className="mb-8">
        <h1 className="text-2xl font-light tracking-tight">
          Set up your organization
        </h1>
        <p className="mt-3 text-sm leading-5 text-muted">
          Create the workspace that owns your assistants, credentials, and
          governance settings.
        </p>
      </div>

      <form
        noValidate
        onChange={() => {
          if (error) setError('');
        }}
        onSubmit={handleSubmit(onCreateOrganization)}
      >
        <Stack gap={6}>
          <TextInput
            id="org-name"
            labelText="Organization name"
            type="text"
            required
            placeholder="e.g., Acme Voice Studio"
            helperText="Use your agency, brand, or operating company name."
            invalid={Boolean(errors.organizationName)}
            invalidText={errors.organizationName?.message}
            {...register('organizationName', {
              required: 'Enter an organization name.',
              setValueAs: value => value.trim(),
            })}
          />
          <Select
            id="org-size"
            labelText="Operating model"
            helperText="This helps tailor setup guidance for your team."
            required
            invalid={Boolean(errors.organizationSize)}
            invalidText={errors.organizationSize?.message}
            {...register('organizationSize', {
              required: 'Select an operating model.',
            })}
          >
            <SelectItem value="" text="Select an operating model" />
            <SelectItem value="agency" text="Agency or system integrator" />
            <SelectItem value="in-house" text="In-house product team" />
            <SelectItem
              value="enterprise"
              text="Enterprise or multi-team organization"
            />
          </Select>
          <TextInput
            id="org-industry"
            labelText="Industry"
            type="text"
            required
            placeholder="e.g., Agency services, healthcare, finance"
            helperText="Used to suggest relevant integrations and templates."
            invalid={Boolean(errors.organizationIndustry)}
            invalidText={errors.organizationIndustry?.message}
            {...register('organizationIndustry', {
              required: 'Enter your organization industry.',
              setValueAs: value => value.trim(),
            })}
          />
          {error && (
            <Notification
              kind="error"
              title="Organization not created"
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
            Create organization
          </PrimaryButton>
        </Stack>
      </form>
    </>
  );
}
