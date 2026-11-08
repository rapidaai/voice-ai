import React, { useState, useEffect, useCallback } from 'react';
import { Helmet } from '@/app/components/app-shell/helmet';
import { DescriptiveHeading } from '@/app/components/layout/heading/descriptive-heading';
import { Input } from '@/app/components/ui/primitives';
import { Label } from '@/app/components/ui/primitives';
import { ArrowButton } from '@/app/components/ui/primitives/buttons/arrow-button';
import { DangerTertiaryButton } from '@/app/components/ui/primitives';
import {
  GetOrganizationResponse,
  Organization,
  ServiceError,
  UpdateOrganizationResponse,
} from '@rapidaai/react';
import toast from 'react-hot-toast/headless';
import { useRapidaStore } from '@/stores/app';
import { useCredential } from '@/hooks/use-credential';
import { useForm } from 'react-hook-form';
import {
  getWorkspaceOrganization,
  updateWorkspaceOrganization,
} from '@/clients';

export function OverviewPage() {
  const [organization, setOrganization] = useState<
    Partial<Organization.AsObject>
  >({});
  const { showLoader, hideLoader } = useRapidaStore();
  const [userId, token] = useCredential();
  const { register, handleSubmit } = useForm();

  const afterUpdateOrganization = useCallback(
    (err: ServiceError | null, uor: UpdateOrganizationResponse | null) => {
      hideLoader();
      if (err) {
        toast.error('Unable to process your request. please try again later.');
        return;
      }
      if (uor?.getSuccess()) {
        toast.success('Your organization details are successfully updated.');
      } else {
        let errorMessage = uor?.getError();
        if (errorMessage) toast.error(errorMessage.getHumanmessage());
        else
          toast.error(
            'Unable to process your request. please try again later.',
          );
        return;
      }
    },
    [hideLoader],
  );

  const onUpdateOrganization = data => {
    const orgId = organization?.id;
    if (!orgId) return;
    showLoader();
    updateWorkspaceOrganization({
      organizationId: orgId,
      name: data.organizationName,
      industry: data.organizationIndustry,
      contact: data.organizationContact,
      auth: { token, userId },
      callback: afterUpdateOrganization,
    });
  };

  const afterGetOrganization = useCallback(
    (err: ServiceError | null, gor: GetOrganizationResponse | null) => {
      if (err) {
        hideLoader();
        toast.error('Unable to process your request. please try again later.');
        return;
      }
      if (gor?.getSuccess()) {
        hideLoader();
        const org = gor.getData()?.toObject();
        if (org) setOrganization(org);
      } else {
        const errorMessage = gor?.getError();
        if (errorMessage) toast.error(errorMessage.getHumanmessage());
        else
          toast.error(
            'Unable to process your request. please try again later.',
          );
        return;
      }
    },
    [hideLoader],
  );

  useEffect(() => {
    getWorkspaceOrganization({
      auth: { token, userId },
      callback: afterGetOrganization,
    });
  }, [afterGetOrganization, token, userId]);

  return (
    <>
      <Helmet title="Organization Overview" />
      <div className="flex items-center justify-between py-2 px-4">
        <DescriptiveHeading heading="Organization Profile" />
      </div>
      <form
        className="space-y-8 py-4 px-4 border-t border-border-subtle"
        onSubmit={handleSubmit(onUpdateOrganization)}
      >
        <fieldset>
          <input
            {...register('organizationId')}
            type="hidden"
            value={organization?.id}
          />
          <div className="items-center md:mx-0 mx-5 grid grid-cols-6 md:grid-cols-9 gap-4 w-full">
            <div className="space-y-2 col-span-3">
              <Label for="Organization Name" text="Organization Name"></Label>
              <Input
                type="text"
                placeholder="eg: Acme Voice Studio"
                {...register('organizationName')}
                value={organization?.name}
                onChange={e => {
                  setOrganization({
                    ...organization,
                    name: e.target.value,
                  });
                }}
              ></Input>
            </div>
            <div className="space-y-2 col-span-3">
              <Label
                for="Organization Industry"
                text="Organization Industry"
              ></Label>
              <Input
                type="text"
                {...register('organizationIndustry')}
                value={organization?.industry}
                onChange={e => {
                  setOrganization({
                    ...organization,
                    industry: e.target.value,
                  });
                }}
                placeholder="eg: Agency services, healthcare, fintech"
              ></Input>
            </div>
          </div>
        </fieldset>
        <fieldset>
          <h3 className="font-semibold text-lg leading-normal">
            Organization Contact
          </h3>
          <p className="text-sm mt-1">
            Organization-level alerts, billing updates, and operational
            notifications will be sent to this email.
          </p>
          <div className="md:mx-0 m-5 flex space-y-2 w-full flex-col">
            <Label for="Email" text="Email"></Label>
            <div className="flex space-x-3">
              <div className="w-60">
                <Input
                  type="email"
                  placeholder="eg: ops@acmevoice.com"
                  {...register('organizationContact')}
                  value={organization?.contact}
                  onChange={e => {
                    setOrganization({
                      ...organization,
                      contact: e.target.value,
                    });
                  }}
                ></Input>
              </div>
            </div>
          </div>
        </fieldset>
        <fieldset className="hidden">
          <DescriptiveHeading
            heading="Archieve Organization"
            subheading="No longer want to use our service? You can archieve your
              organization here. This action is not reversible. All information
              related to this organization will be deleted permanently."
          ></DescriptiveHeading>

          <div className="md:mx-0 m-5 flex space-y-2 w-full flex-col">
            <div className="w-60">
              <DangerTertiaryButton size="sm" type="button" onClick={() => {}}>
                Archieve Organization
              </DangerTertiaryButton>
            </div>
          </div>
        </fieldset>
        <footer className="border-t pt-8 border-border-subtle">
          <ul className="flex justify-end">
            <li className="ml-3">
              <ArrowButton type="submit" label="Update details" />
            </li>
          </ul>
        </footer>
      </form>
    </>
  );
}
