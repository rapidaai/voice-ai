import React, { useState, useEffect, useCallback } from 'react';
import { Helmet } from '@/app/components/Helmet';
import { DescriptiveHeading } from '@/app/components/Heading/DescriptiveHeading';
import { Input } from '@/app/components/Form/Input';
import { Label } from '@/app/components/Form/Label';
import { ArrowButton } from '@/app/components/Form/Button/ArrowButton';
import { BorderedButton } from '@/app/components/Form/Button/Bordered';
import { ServiceError } from '@rapidaai/react';
import { UpdateOrganization, GetOrganization } from '@rapidaai/react';

import {
  GetOrganizationResponse,
  UpdateOrganizationResponse,
} from '@rapidaai/react';
import { Organization } from '@rapidaai/react';
import toast from 'react-hot-toast/headless';
import { useRapidaStore } from '@/hooks';
import { useCredential } from '@/hooks/use-credential';
import { useForm } from 'react-hook-form';
import { connectionConfig } from '@/configs';

/**
 *
 * @returns
 */
export function OverviewPage() {
  /**
   * getting the updated information
   */
  const [organization, setOrganization] = useState<
    Partial<Organization.AsObject>
  >({});
  /**
   * setLoading context
   */
  const { showLoader, hideLoader } = useRapidaStore();
  /**
   * credentials
   */
  const [userId, token] = useCredential();

  /**
   * handle the form
   */
  const { register, handleSubmit } = useForm();

  /**
   * callback after update organization
   */
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
    [],
  );
  /**
   *
   * @param e
   */
  const onUpdateOrganization = data => {
    let orgId = organization?.id;
    if (!orgId) return;
    showLoader();
    UpdateOrganization(
      connectionConfig,
      orgId,
      {
        authorization: token,
        'x-auth-id': userId,
      },
      afterUpdateOrganization,
      data.organizationName,
      data.organizationIndustry,
      data.organizationContact,
    );
  };

  //   set the state of show the organization details
  const afterGetOrganization = useCallback(
    (err: ServiceError | null, gor: GetOrganizationResponse | null) => {
      if (err) {
        hideLoader();
        toast.error('Unable to process your request. please try again later.');
        return;
      }
      if (gor?.getSuccess()) {
        hideLoader();
        let org = gor.getData()?.toObject();
        if (org) setOrganization(org);
      } else {
        let errorMessage = gor?.getError();
        if (errorMessage) toast.error(errorMessage.getHumanmessage());
        else
          toast.error(
            'Unable to process your request. please try again later.',
          );
        return;
      }
    },
    [],
  );

  /**
   * when you come to the page then call the organization
   */
  useEffect(() => {
    GetOrganization(
      connectionConfig,
      {
        authorization: token,
        'x-auth-id': userId,
      },
      afterGetOrganization,
    );
  }, []);

  //
  return (
    <>
      <Helmet title="Organization Overview" />
      <div className="flex items-center justify-between py-2 px-4">
        <DescriptiveHeading heading="Organization Profile" />
      </div>
      <form
        className="space-y-8 py-4 px-4 border-t dark:border-gray-800"
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
                placeholder="eg: Lexatic inc"
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
                placeholder="eg: Software, Engineering"
              ></Input>
            </div>
          </div>
        </fieldset>
        <fieldset>
          <h3 className="font-semibold text-lg leading-normal">
            Organization Contact
          </h3>
          <p className="text-sm mt-1">
            All the organization conversation will be send to the given contact
            email.
          </p>
          <div className="md:mx-0 m-5 flex space-y-2 w-full flex-col">
            <Label for="Email" text="Email"></Label>
            <div className="flex space-x-3">
              <div className="w-60">
                <Input
                  type="email"
                  placeholder="eg: john@rapida.ai"
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
              <BorderedButton
                type="button"
                onClick={() => {}}
                size="sm"
                width="w-fit"
                color="red"
              >
                Archieve Organization
              </BorderedButton>
            </div>
          </div>
        </fieldset>
        <footer className="border-t pt-8 dark:border-gray-700">
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
